const test = require("node:test");
const assert = require("node:assert/strict");
const { createHash } = require("node:crypto");
const { mkdir, mkdtemp, readFile, rm, writeFile } = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const AdmZip = require("adm-zip");
const tar = require("tar");

const {
  inspectArtifacts,
  parseArguments,
  publishRelease,
} = require("../scripts/publish-atomgit-release");

const ARCHIVES = [
  "ag_darwin_amd64.tar.gz",
  "ag_darwin_arm64.tar.gz",
  "ag_linux_amd64.tar.gz",
  "ag_linux_arm64.tar.gz",
  "ag_linux_loong64.tar.gz",
  "ag_windows_amd64.zip",
  "ag_windows_arm64.zip",
];
function digest(buffer) {
  return createHash("sha256").update(buffer).digest("hex");
}

async function createFixture(t, tag = "v1.2.3") {
  const root = await mkdtemp(path.join(os.tmpdir(), "ag-release-publish-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const releaseDir = path.join(root, tag);
  await mkdir(releaseDir);
  async function createArchive(directory, name, index) {
    if (name.endsWith(".zip")) {
      const zip = new AdmZip();
      zip.addFile("LICENSE", Buffer.from("license"));
      zip.addFile("ag.exe", Buffer.from(`binary-${index}`));
      zip.writeZip(path.join(directory, name));
    } else {
      const source = path.join(root, `source-${index}`);
      await mkdir(source);
      await writeFile(path.join(source, "LICENSE"), "license");
      await writeFile(path.join(source, "ag"), `binary-${index}`);
      await tar.c({ cwd: source, file: path.join(directory, name), gzip: true }, ["LICENSE", "ag"]);
    }
  }
  for (const [index, name] of ARCHIVES.entries()) await createArchive(releaseDir, name, index);
  await writeFile(path.join(releaseDir, "install.sh"), `_BUNDLED_TAG="${tag}"\n`);
  await writeFile(path.join(releaseDir, "install.ps1"), `$BundledTag = '${tag}'\n`);
  const checksumNames = [...ARCHIVES, "install.sh", "install.ps1"];
  const lines = [];
  for (const name of checksumNames) lines.push(`${digest(await readFile(path.join(releaseDir, name)))}  ${name}`);
  await writeFile(path.join(releaseDir, "checksums.txt"), `${lines.join("\n")}\n`);
  await mkdir(path.join(releaseDir, "npm"));
  const notesFile = path.join(root, "notes.md");
  await writeFile(notesFile, "Release notes\n");
  return { notesFile, releaseDir, tag };
}

function optionsFor(fixture, overrides = {}) {
  return {
    repo: "alice/demo",
    version: fixture.tag,
    dir: fixture.releaseDir,
    notesFile: fixture.notesFile,
    notes: "Release notes\n",
    target: "a".repeat(40),
    name: fixture.tag,
    prerelease: false,
    dryRun: false,
    ...overrides,
  };
}

function fakeAtomGit(plan, options, behavior = {}) {
  const calls = [];
  const artifactsByPath = new Map(plan.artifacts.map((artifact) => [artifact.filePath, artifact]));
  const contents = new Map();
  let release = behavior.release || null;

  function response(value, status = 0, raw = false) {
    return { status, stdout: raw ? value : typeof value === "string" ? value : JSON.stringify(value), stderr: "" };
  }
  function releaseJSON() {
    const sourceAssets = [`${options.version}.zip`, `${options.version}.tar.gz`, `${options.version}.tar.bz2`, `${options.version}.tar`]
      .map((name) => ({ id: null, name, type: "source", browser_download_url: `https://sources.example/${name}` }));
    return {
      ...release,
      assets: [...sourceAssets, ...[...contents.keys()].map((name, index) => ({
        id: index + 1,
        name,
        type: "attach",
        browser_download_url: `https://downloads.example/${name}`,
      })), ...(behavior.extraAssets || [])],
    };
  }
  async function runAg(args, runOptions = {}) {
    calls.push(args);
    if (args[0] === "api" && args[1] === "/user") {
      if (behavior.authError) return { status: 1, stdout: "", stderr: behavior.authError };
      return response({ login: "release-bot" });
    }
    if (args[0] === "api" && args[1].includes("/attach_files/")) {
      const name = decodeURIComponent(args[1].split("/attach_files/")[1].split("/download")[0]);
      if (!contents.has(name)) return { status: 1, stdout: Buffer.alloc(0), stderr: "404 Not Found" };
      return response(contents.get(name), 0, true);
    }
    if (args[0] === "api" && args[1].includes("/releases/tags/")) {
      return release ? response(releaseJSON()) : { status: 1, stdout: "", stderr: "404 Not Found" };
    }
    if (args[0] === "release" && args[1] === "create") {
      release = {
        tag_name: options.version,
        target_commitish: options.target,
        name: options.name,
        body: options.notes,
        release_status: behavior.createdReleaseStatus ?? (options.prerelease ? "pre" : "latest"),
      };
      return response("created");
    }
    if (args[0] === "release" && args[1] === "edit") {
      release = { ...release, name: options.name, body: options.notes, release_status: options.prerelease ? "pre" : "latest" };
      return response("edited");
    }
    if (args[0] === "release" && args[1] === "upload") {
      const artifact = artifactsByPath.get(args[4]);
      if (behavior.failUpload === artifact.name) return { status: 1, stdout: "", stderr: "upload failed" };
      contents.set(artifact.name, await readFile(artifact.filePath));
      return response("uploaded");
    }
    throw new Error(`unexpected ag command: ${args.join(" ")}; raw=${runOptions.raw}`);
  }
  async function seed(names) {
    release = {
      tag_name: options.version,
      target_commitish: options.target,
      name: options.name,
      body: options.notes,
      release_status: options.prerelease ? "pre" : "latest",
    };
    for (const name of names) {
      const artifact = plan.artifacts.find((item) => item.name === name);
      contents.set(name, await readFile(artifact.filePath));
    }
  }
  return { calls, contents, runAg, seed };
}

test("validates explicit release inputs", () => {
  const parsed = parseArguments([
    "--repo", "alice/demo", "--version", "v1.2.3", "--dir", "dist/v1.2.3",
    "--notes-file", "notes.md", "--target", "a".repeat(40), "--dry-run",
  ]);
  assert.equal(parsed.dryRun, true);
  assert.throws(() => parseArguments(["--repo", "alice/demo"]), /--version is required/);
  assert.throws(() => parseArguments([
    "--repo", "alice/demo", "--version", "1.2.3", "--dir", "dist", "--notes-file", "n", "--target", "a".repeat(40),
  ]), /expected vX.Y.Z/);
});

test("validates the release artifact set, checksums, archives, and installer tags", async (t) => {
  const fixture = await createFixture(t);
  const plan = await inspectArtifacts(fixture.releaseDir, fixture.tag);
  assert.equal(plan.artifacts.length, 10);

  await writeFile(path.join(fixture.releaseDir, "extra.txt"), "extra");
  await assert.rejects(inspectArtifacts(fixture.releaseDir, fixture.tag), /extra: extra.txt/);
});

test("rejects legacy package-manager artifact directories", async (t) => {
  const fixture = await createFixture(t);
  await mkdir(path.join(fixture.releaseDir, "package-managers"));
  await assert.rejects(inspectArtifacts(fixture.releaseDir, fixture.tag), /extra: package-managers/);
});

test("dry run performs no AtomGit requests", async (t) => {
  const fixture = await createFixture(t);
  const plan = await inspectArtifacts(fixture.releaseDir, fixture.tag);
  const calls = [];
  await publishRelease(plan, { ...optionsFor(fixture), dryRun: true, logger: () => {}, runAg: async (args) => calls.push(args) });
  assert.deepEqual(calls, []);
});

test("creates a release, uploads every attachment, and verifies downloads", async (t) => {
  const fixture = await createFixture(t);
  const plan = await inspectArtifacts(fixture.releaseDir, fixture.tag);
  const options = optionsFor(fixture);
  const atomgit = fakeAtomGit(plan, options);
  await publishRelease(plan, { ...options, logger: () => {}, runAg: atomgit.runAg });
  assert.equal(atomgit.contents.size, 10);
  assert.equal(atomgit.calls.filter((args) => args[0] === "release" && args[1] === "create").length, 1);
});

test("normalizes a newly created release from none to latest", async (t) => {
  const fixture = await createFixture(t);
  const plan = await inspectArtifacts(fixture.releaseDir, fixture.tag);
  const options = optionsFor(fixture);
  const atomgit = fakeAtomGit(plan, options, { createdReleaseStatus: "none" });
  await publishRelease(plan, { ...options, logger: () => {}, runAg: atomgit.runAg });
  assert.equal(atomgit.contents.size, 10);
  assert.equal(atomgit.calls.filter((args) => args[0] === "release" && args[1] === "create").length, 1);
  const editIndex = atomgit.calls.findIndex((args) => args[0] === "release" && args[1] === "edit");
  const uploadIndex = atomgit.calls.findIndex((args) => args[0] === "release" && args[1] === "upload");
  assert.ok(editIndex >= 0 && editIndex < uploadIndex);
  assert.ok(atomgit.calls[editIndex].includes("--latest"));
});

test("resumes an existing partial release without re-uploading verified attachments", async (t) => {
  const fixture = await createFixture(t);
  const plan = await inspectArtifacts(fixture.releaseDir, fixture.tag);
  const options = optionsFor(fixture);
  const atomgit = fakeAtomGit(plan, options);
  await atomgit.seed(plan.artifacts.slice(0, 3).map(({ name }) => name));
  await publishRelease(plan, { ...options, logger: () => {}, runAg: atomgit.runAg });
  assert.equal(atomgit.contents.size, 10);
  assert.equal(atomgit.calls.filter((args) => args[0] === "release" && args[1] === "upload").length, 7);
});

test("rejects unexpected AtomGit source archives", async (t) => {
  const fixture = await createFixture(t);
  const plan = await inspectArtifacts(fixture.releaseDir, fixture.tag);
  const options = optionsFor(fixture);
  const atomgit = fakeAtomGit(plan, options, { extraAssets: [{ id: null, name: "unexpected.tar.gz", type: "source" }] });
  await assert.rejects(
    publishRelease(plan, { ...options, logger: () => {}, runAg: atomgit.runAg }),
    /unexpected source archive/,
  );
});

test("upload failures report the remaining attachments and never claim success", async (t) => {
  const fixture = await createFixture(t);
  const plan = await inspectArtifacts(fixture.releaseDir, fixture.tag);
  const options = optionsFor(fixture);
  const failedName = plan.artifacts[2].name;
  const atomgit = fakeAtomGit(plan, options, { failUpload: failedName });
  await assert.rejects(
    publishRelease(plan, { ...options, logger: () => {}, runAg: atomgit.runAg }),
    new RegExp(`failed to upload ${failedName}.*remaining attachments`),
  );
  assert.equal(atomgit.contents.has(failedName), false);
});

test("authentication errors redact tokens", async (t) => {
  const fixture = await createFixture(t);
  const plan = await inspectArtifacts(fixture.releaseDir, fixture.tag);
  const options = optionsFor(fixture);
  const secret = "atomgit-secret";
  const atomgit = fakeAtomGit(plan, options, { authError: `Bearer ${secret}` });
  await assert.rejects(
    publishRelease(plan, { ...options, env: { ATOMGIT_TOKEN: secret }, logger: () => {}, runAg: atomgit.runAg }),
    (error) => !error.message.includes(secret) && error.message.includes("[REDACTED]"),
  );
});
