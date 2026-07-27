#!/usr/bin/env node

const { createHash } = require("node:crypto");
const { readFile, readdir } = require("node:fs/promises");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const AdmZip = require("adm-zip");
const tar = require("tar");

const REQUIRED_ARCHIVES = [
  "ag_darwin_amd64.tar.gz",
  "ag_darwin_arm64.tar.gz",
  "ag_linux_amd64.tar.gz",
  "ag_linux_arm64.tar.gz",
  "ag_windows_amd64.zip",
  "ag_windows_arm64.zip",
];
const OPTIONAL_ARCHIVES = ["ag_linux_loong64.tar.gz"];
const INSTALLERS = ["install.sh", "install.ps1"];

function parseArguments(args) {
  const options = { dryRun: false, prerelease: false };
  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if (arg === "--dry-run") options.dryRun = true;
    else if (arg === "--prerelease") options.prerelease = true;
    else if (arg === "--help" || arg === "-h") options.help = true;
    else if (["--repo", "--version", "--dir", "--notes-file", "--target", "--name"].includes(arg)) {
      const value = args[index + 1];
      if (!value) throw new Error(`${arg} requires a value`);
      index += 1;
      options[arg.slice(2).replace("notes-file", "notesFile")] = value;
    } else throw new Error(`unknown argument: ${arg}`);
  }
  if (options.help) return options;
  for (const key of ["repo", "version", "dir", "notesFile", "target"]) {
    if (!options[key]) throw new Error(`--${key === "notesFile" ? "notes-file" : key} is required`);
  }
  if (!/^[^/\s]+\/[^/\s]+$/.test(options.repo)) throw new Error(`invalid repository: ${options.repo}`);
  if (!/^v(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$/.test(options.version)) {
    throw new Error(`invalid version ${JSON.stringify(options.version)}; expected vX.Y.Z`);
  }
  if (!/^[0-9a-f]{40}$/i.test(options.target)) throw new Error("--target must be a full 40-character commit SHA");
  options.dir = path.resolve(options.dir);
  options.notesFile = path.resolve(options.notesFile);
  options.name ||= options.version;
  return options;
}

function parseChecksums(contents) {
  const result = new Map();
  for (const [index, raw] of contents.split(/\r?\n/).entries()) {
    if (!raw.trim()) continue;
    const match = /^([0-9a-fA-F]{64})\s+\*?(?:\.\/)?([^/\\]+)$/.exec(raw.trim());
    if (!match) throw new Error(`invalid checksums.txt line ${index + 1}`);
    if (result.has(match[2])) throw new Error(`duplicate checksum entry for ${match[2]}`);
    result.set(match[2], match[1].toLowerCase());
  }
  return result;
}

function sha256(buffer) {
  return createHash("sha256").update(buffer).digest("hex");
}

async function validateArchive(filePath) {
  const fileName = path.basename(filePath);
  const expectedBinary = fileName.endsWith(".zip") ? "ag.exe" : "ag";
  let entries;
  if (fileName.endsWith(".zip")) {
    entries = new Set(new AdmZip(filePath).getEntries().filter((entry) => !entry.isDirectory).map((entry) => entry.entryName));
  } else {
    entries = new Set();
    await tar.t({
      file: filePath,
      strict: true,
      onentry(entry) {
        if (entry.type === "File") entries.add(entry.path.replaceAll("\\", "/"));
        entry.resume();
      },
    });
  }
  for (const required of [expectedBinary, "LICENSE"]) {
    if (!entries.has(required)) throw new Error(`${fileName} does not contain ${required}`);
  }
  if ([...entries].some((entry) => entry.includes("..") || path.posix.isAbsolute(entry))) {
    throw new Error(`${fileName} contains an unsafe archive path`);
  }
}

async function inspectArtifacts(directory, tag) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = entries.filter((entry) => entry.isFile()).map((entry) => entry.name).sort();
  const optional = OPTIONAL_ARCHIVES.filter((name) => files.includes(name));
  const checksumNames = [...REQUIRED_ARCHIVES, ...optional, ...INSTALLERS].sort();
  const expectedFiles = [...checksumNames, "checksums.txt"].sort();
  if (JSON.stringify(files) !== JSON.stringify(expectedFiles)) {
    const expected = new Set(expectedFiles);
    const actual = new Set(files);
    const missing = expectedFiles.filter((name) => !actual.has(name));
    const extra = files.filter((name) => !expected.has(name));
    throw new Error(`invalid release artifact set; missing: ${missing.join(", ") || "none"}; extra: ${extra.join(", ") || "none"}`);
  }

  const checksums = parseChecksums(await readFile(path.join(directory, "checksums.txt"), "utf8"));
  if (JSON.stringify([...checksums.keys()].sort()) !== JSON.stringify(checksumNames)) {
    throw new Error("checksums.txt entries do not match the release attachment set");
  }

  const artifacts = [];
  for (const name of checksumNames) {
    const filePath = path.join(directory, name);
    const contents = await readFile(filePath);
    const digest = sha256(contents);
    if (checksums.get(name) !== digest) throw new Error(`checksum mismatch for ${name}`);
    if (name.endsWith(".zip") || name.endsWith(".tar.gz")) await validateArchive(filePath);
    artifacts.push({ name, filePath, sha256: digest });
  }
  const shellInstaller = await readFile(path.join(directory, "install.sh"), "utf8");
  const powershellInstaller = await readFile(path.join(directory, "install.ps1"), "utf8");
  if (!shellInstaller.includes(`_BUNDLED_TAG="${tag}"`)) throw new Error(`install.sh is not bound to ${tag}`);
  if (!powershellInstaller.includes(`$BundledTag = '${tag}'`)) throw new Error(`install.ps1 is not bound to ${tag}`);

  const checksumContents = await readFile(path.join(directory, "checksums.txt"));
  artifacts.push({ name: "checksums.txt", filePath: path.join(directory, "checksums.txt"), sha256: sha256(checksumContents) });
  return { artifacts, directory };
}

function redact(value, env = process.env) {
  let output = String(value || "");
  for (const key of ["ATOMGIT_TOKEN", "AG_TOKEN", "ACCESS_TOKEN"]) {
    if (env[key]) output = output.split(env[key]).join("[REDACTED]");
  }
  return output.replace(/(authorization:\s*bearer\s+)\S+/gi, "$1[REDACTED]");
}

function defaultRunAg(args, options = {}) {
  const binary = process.env.AG_RELEASE_CLI || "ag";
  const result = spawnSync(binary, args, {
    encoding: options.raw ? null : "utf8",
    env: process.env,
    maxBuffer: 128 * 1024 * 1024,
    windowsHide: true,
  });
  if (result.error) return { status: 1, stdout: result.stdout || "", stderr: result.error.message };
  return { status: result.status ?? 1, stdout: result.stdout || "", stderr: result.stderr || "" };
}

async function callAg(runAg, args, options = {}) {
  return runAg(args, options);
}

function commandError(context, result, env) {
  const raw = result.stderr || result.stdout || `exit status ${result.status}`;
  return new Error(`${context}: ${redact(Buffer.isBuffer(raw) ? raw.toString("utf8") : raw, env).trim()}`);
}

function releaseEndpoint(repo, tag) {
  const [owner, name] = repo.split("/");
  return `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/releases/tags/${encodeURIComponent(tag)}`;
}

async function getRelease(runAg, repo, tag, env) {
  const result = await callAg(runAg, ["api", releaseEndpoint(repo, tag), "--raw-output"]);
  if (result.status !== 0) {
    if (/404|not found/i.test(`${result.stdout}\n${result.stderr}`)) return null;
    throw commandError(`failed to query release ${tag}`, result, env);
  }
  try {
    return JSON.parse(result.stdout);
  } catch (error) {
    throw new Error(`release ${tag} returned invalid JSON: ${error.message}`);
  }
}

function expectedStatus(options) {
  return options.prerelease ? "pre" : "latest";
}

function assertReleaseMetadata(release, options) {
  if (release.tag_name !== options.version) throw new Error(`release tag is ${release.tag_name}, expected ${options.version}`);
  if (release.target_commitish !== options.target) throw new Error(`release target is ${release.target_commitish}, expected ${options.target}`);
  if (release.name !== options.name || release.body !== options.notes || release.release_status !== expectedStatus(options)) {
    throw new Error(`release ${options.version} metadata does not match the requested name, notes, or status`);
  }
}

async function downloadAttachment(runAg, options, name, env) {
  const endpoint = `${releaseEndpoint(options.repo, options.version).replace("/tags/", "/")}/attach_files/${encodeURIComponent(name)}/download`;
  const result = await callAg(runAg, ["api", endpoint, "--accept", "application/octet-stream", "--raw-output"], { raw: true });
  if (result.status !== 0) throw commandError(`failed to download attachment ${name}`, result, env);
  return Buffer.isBuffer(result.stdout) ? result.stdout : Buffer.from(result.stdout);
}

function attachmentMap(release, expectedNames) {
  const expected = new Set(expectedNames);
  const result = new Map();
  for (const asset of release.assets || []) {
    if (asset.type !== "attach") continue;
    if (!expected.has(asset.name)) throw new Error(`release contains unexpected attachment ${asset.name}`);
    if (result.has(asset.name)) throw new Error(`release contains duplicate attachment ${asset.name}`);
    result.set(asset.name, asset);
  }
  return result;
}

async function verifyAttachment(runAg, options, artifact, asset, env) {
  if (!asset?.browser_download_url) throw new Error(`attachment ${artifact.name} has no download URL`);
  const contents = await downloadAttachment(runAg, options, artifact.name, env);
  if (sha256(contents) !== artifact.sha256) throw new Error(`remote attachment ${artifact.name} checksum does not match local artifact`);
}

async function ensureRelease(runAg, options, env) {
  let release = await getRelease(runAg, options.repo, options.version, env);
  const createArgs = ["release", "create", options.repo, options.version, "--name", options.name, "--body-file", options.notesFile, "--target", options.target];
  if (options.prerelease) createArgs.push("--prerelease");
  if (!release) {
    const created = await callAg(runAg, createArgs);
    if (created.status !== 0) {
      release = await getRelease(runAg, options.repo, options.version, env);
      if (!release) throw commandError(`failed to create release ${options.version}`, created, env);
    } else release = await getRelease(runAg, options.repo, options.version, env);
  } else {
    if (release.target_commitish !== options.target) throw new Error(`existing release target ${release.target_commitish} does not match ${options.target}`);
    const desiredStatus = expectedStatus(options);
    if (release.name !== options.name || release.body !== options.notes || release.release_status !== desiredStatus) {
      const editArgs = ["release", "edit", options.repo, options.version, "--name", options.name, "--body-file", options.notesFile, options.prerelease ? "--prerelease" : "--latest"];
      const edited = await callAg(runAg, editArgs);
      release = await getRelease(runAg, options.repo, options.version, env);
      if (edited.status !== 0) {
        try {
          assertReleaseMetadata(release, options);
        } catch {
          throw commandError(`failed to update release ${options.version}`, edited, env);
        }
      }
    }
  }
  if (!release) throw new Error(`release ${options.version} was not visible after creation`);
  assertReleaseMetadata(release, options);
  return release;
}

async function publishRelease(plan, options = {}) {
  const runAg = options.runAg || defaultRunAg;
  const logger = options.logger || console.log;
  const env = options.env || process.env;
  if (options.dryRun) {
    logger(`Validated ${plan.artifacts.length} release attachments for ${options.version}.`);
    for (const artifact of plan.artifacts) logger(`Would upload ${artifact.name}`);
    logger("Dry run complete; no AtomGit API requests were made.");
    return;
  }
  const auth = await callAg(runAg, ["api", "/user", "--raw-output"]);
  if (auth.status !== 0) throw commandError("AtomGit authentication failed", auth, env);
  let identity;
  try { identity = JSON.parse(auth.stdout).login; } catch { identity = "authenticated user"; }
  logger(`Authenticated to AtomGit as ${identity || "authenticated user"}.`);

  let release = await ensureRelease(runAg, options, env);
  const expectedNames = plan.artifacts.map(({ name }) => name);
  let assets = attachmentMap(release, expectedNames);
  for (let index = 0; index < plan.artifacts.length; index += 1) {
    const artifact = plan.artifacts[index];
    if (assets.has(artifact.name)) {
      await verifyAttachment(runAg, options, artifact, assets.get(artifact.name), env);
      logger(`Skipping verified attachment ${artifact.name}.`);
      continue;
    }
    const uploaded = await callAg(runAg, ["release", "upload", options.repo, options.version, artifact.filePath, "--name", artifact.name]);
    if (uploaded.status !== 0) {
      release = await getRelease(runAg, options.repo, options.version, env);
      assets = release ? attachmentMap(release, expectedNames) : new Map();
      if (!assets.has(artifact.name)) {
        const remaining = plan.artifacts.slice(index).map(({ name }) => name).join(", ");
        throw new Error(`${commandError(`failed to upload ${artifact.name}`, uploaded, env).message}; remaining attachments: ${remaining}`);
      }
      await verifyAttachment(runAg, options, artifact, assets.get(artifact.name), env);
    }
    logger(`Uploaded and verified ${artifact.name}.`);
    release = await getRelease(runAg, options.repo, options.version, env);
    assets = attachmentMap(release, expectedNames);
  }

  release = await getRelease(runAg, options.repo, options.version, env);
  if (!release) throw new Error(`release ${options.version} disappeared during final verification`);
  assertReleaseMetadata(release, options);
  assets = attachmentMap(release, expectedNames);
  if (assets.size !== plan.artifacts.length) throw new Error(`release verification found ${assets.size} of ${plan.artifacts.length} expected attachments`);
  for (const artifact of plan.artifacts) await verifyAttachment(runAg, options, artifact, assets.get(artifact.name), env);
  logger(`Published and verified ${options.version}: https://atomgit.com/${options.repo}/releases/tag/${options.version}`);
}

function usage() {
  return "Usage: node scripts/publish-atomgit-release.js --repo owner/repo --version vX.Y.Z --dir dist/vX.Y.Z --notes-file notes.md --target <40-char-sha> [--name title] [--prerelease] [--dry-run]";
}

async function main(args = process.argv.slice(2)) {
  const options = parseArguments(args);
  if (options.help) return console.log(usage());
  options.notes = await readFile(options.notesFile, "utf8");
  if (!options.notes.trim()) throw new Error("release notes must not be empty");
  const plan = await inspectArtifacts(options.dir, options.version);
  await publishRelease(plan, options);
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`Error: ${redact(error.message)}`);
    process.exitCode = 1;
  });
}

module.exports = { inspectArtifacts, parseArguments, parseChecksums, publishRelease, redact, sha256 };
