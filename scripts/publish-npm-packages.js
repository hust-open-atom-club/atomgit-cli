#!/usr/bin/env node

const { createHash } = require("node:crypto");
const { existsSync } = require("node:fs");
const { mkdtemp, readdir, readFile, rm } = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const tar = require("tar");
const { PLATFORM_PACKAGES, PLATFORM_PACKAGE_NAMES } = require("./check-npm-version");

const MAIN_PACKAGE_NAME = "@hust-open-atom-club/atomgit-cli";
const DEFAULT_REGISTRY = "https://registry.npmjs.org";
const DEFAULT_VERIFY_ATTEMPTS = 12;
const DEFAULT_VERIFY_DELAY_MS = 5000;
const PUBLISH_MODE_STAGE = "stage";
const PUBLISH_MODE_DIRECT = "publish";
const PLATFORM_PACKAGES_BY_NAME = new Map(
  PLATFORM_PACKAGES.map((platformPackage) => [platformPackage.name, platformPackage]),
);

function normalizeVersion(value) {
  const match = /^v?((?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*))$/.exec(
    value || "",
  );
  if (!match) {
    throw new Error(`invalid version ${JSON.stringify(value)}; expected vX.Y.Z or X.Y.Z`);
  }
  return match[1];
}

function parseArguments(args, env = process.env) {
  const options = {
    dryRun: env.npm_config_dry_run === "true",
    mode: PUBLISH_MODE_STAGE,
    registry: DEFAULT_REGISTRY,
  };
  const positional = [];
  let selectedMode;

  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if (arg === "--dry-run") {
      options.dryRun = true;
    } else if (arg === "--stage" || arg === "--publish") {
      const mode = arg === "--stage" ? PUBLISH_MODE_STAGE : PUBLISH_MODE_DIRECT;
      if (selectedMode && selectedMode !== mode) {
        throw new Error("--stage and --publish are mutually exclusive");
      }
      selectedMode = mode;
      options.mode = mode;
    } else if (arg === "--help" || arg === "-h") {
      options.help = true;
    } else if (arg === "--version" || arg === "--dir" || arg === "--registry") {
      const value = args[index + 1];
      if (!value) {
        throw new Error(`${arg} requires a value`);
      }
      index += 1;
      if (arg === "--version") options.version = value;
      if (arg === "--dir") options.npmDir = value;
      if (arg === "--registry") options.registry = value;
    } else if (arg.startsWith("--version=")) {
      options.version = arg.slice("--version=".length);
    } else if (arg.startsWith("--dir=")) {
      options.npmDir = arg.slice("--dir=".length);
    } else if (arg.startsWith("--registry=")) {
      options.registry = arg.slice("--registry=".length);
    } else if (arg.startsWith("-")) {
      throw new Error(`unknown option: ${arg}`);
    } else {
      positional.push(arg);
    }
  }

  if (!options.version && positional.length > 0) options.version = positional.shift();
  if (!options.npmDir && positional.length > 0) options.npmDir = positional.shift();
  if (positional.length > 0) {
    throw new Error(`unexpected arguments: ${positional.join(" ")}`);
  }
  if (!options.help && (!options.version || !options.npmDir)) {
    throw new Error("both --version and --dir are required");
  }

  if (options.version) options.version = normalizeVersion(options.version);
  if (![PUBLISH_MODE_STAGE, PUBLISH_MODE_DIRECT].includes(options.mode)) {
    throw new Error(`unsupported publish mode ${JSON.stringify(options.mode)}`);
  }
  options.registry = options.registry.replace(/\/+$/, "");
  if (!options.registry) {
    throw new Error("registry must not be empty");
  }
  if (options.npmDir) options.npmDir = path.resolve(options.npmDir);
  return options;
}

function parseChecksums(contents) {
  const checksums = new Map();
  for (const [index, rawLine] of contents.split(/\r?\n/).entries()) {
    const line = rawLine.trim();
    if (!line) continue;
    const match = /^([0-9a-fA-F]{64})\s+\*?(.+)$/.exec(line);
    if (!match) {
      throw new Error(`invalid checksums.txt line ${index + 1}: ${JSON.stringify(rawLine)}`);
    }
    let fileName = match[2].trim().replaceAll("\\", "/");
    if (fileName.startsWith("./")) fileName = fileName.slice(2);
    if (!fileName || fileName.includes("/")) {
      throw new Error(`checksums.txt line ${index + 1} must name a file in the npm directory`);
    }
    if (checksums.has(fileName)) {
      throw new Error(`checksums.txt contains duplicate entry for ${fileName}`);
    }
    checksums.set(fileName, match[1].toLowerCase());
  }
  if (checksums.size === 0) {
    throw new Error("checksums.txt does not contain any package entries");
  }
  return checksums;
}

function digest(buffer, algorithm, encoding) {
  return createHash(algorithm).update(buffer).digest(encoding);
}

async function inspectTarball(filePath) {
  const entries = new Map();
  const manifestReads = [];
  let manifestText;

  await tar.t({
    file: filePath,
    strict: true,
    onentry(entry) {
      const entryPath = entry.path.replaceAll("\\", "/");
      if (entry.type === "File") {
        entries.set(entryPath, entry.mode);
      }
      if (entryPath === "package/package.json") {
        manifestReads.push(
          entry.concat().then((buffer) => {
            manifestText = buffer.toString("utf8");
          }),
        );
      } else {
        entry.resume();
      }
    },
  });
  await Promise.all(manifestReads);

  if (!manifestText) {
    throw new Error(`${path.basename(filePath)} does not contain package/package.json`);
  }

  let manifest;
  try {
    manifest = JSON.parse(manifestText);
  } catch (error) {
    throw new Error(`${path.basename(filePath)} contains invalid package.json: ${error.message}`);
  }
  return { entries, manifest };
}

function assertExactFiles(packageName, entries, expectedFiles) {
  const expected = new Set(expectedFiles);
  const present = new Set(entries.keys());
  const missing = [...expected].filter((file) => !present.has(file));
  const extra = [...present].filter((file) => !expected.has(file));
  if (missing.length > 0 || extra.length > 0) {
    const details = [];
    if (missing.length > 0) details.push(`missing ${missing.join(", ")}`);
    if (extra.length > 0) details.push(`unexpected ${extra.join(", ")}`);
    throw new Error(`${packageName} tarball contents are invalid: ${details.join("; ")}`);
  }
}

function assertExecutableBit(packageName, entries, entryPath) {
  const mode = entries.get(entryPath);
  if (typeof mode !== "number" || (mode & 0o111) === 0) {
    const modeText = mode === undefined ? "missing" : `0${mode.toString(8)}`;
    throw new Error(
      `${packageName} entry ${entryPath} must be executable; received mode ${modeText}`,
    );
  }
}

function validateMainPackage(packageInfo, version) {
  const { entries, manifest } = packageInfo;
  if (manifest.name !== MAIN_PACKAGE_NAME) {
    throw new Error(`main package name must be ${MAIN_PACKAGE_NAME}, got ${manifest.name}`);
  }
  if (manifest.version !== version) {
    throw new Error(`${manifest.name} version ${manifest.version} does not match ${version}`);
  }
  if (manifest.bin?.ag !== "bin/ag.js") {
    throw new Error(`${manifest.name} must expose bin/ag.js as the ag executable`);
  }
  assertExactFiles(manifest.name, entries, [
    "package/LICENSE",
    "package/README.md",
    "package/bin/ag.js",
    "package/package.json",
  ]);
}

function validatePlatformPackage(packageInfo, expectedPlatform, version) {
  const { entries, manifest } = packageInfo;
  const { name: expectedName, os: expectedOS, cpu: expectedCPU } = expectedPlatform;
  if (manifest.name !== expectedName) {
    throw new Error(`expected platform package ${expectedName}, got ${manifest.name}`);
  }
  if (manifest.version !== version) {
    throw new Error(`${manifest.name} version ${manifest.version} does not match ${version}`);
  }
  if (!Array.isArray(manifest.os) || manifest.os.length !== 1) {
    throw new Error(`${manifest.name} must declare exactly one supported OS`);
  }
  if (!Array.isArray(manifest.cpu) || manifest.cpu.length !== 1) {
    throw new Error(`${manifest.name} must declare exactly one supported CPU architecture`);
  }
  if (manifest.os[0] !== expectedOS || manifest.cpu[0] !== expectedCPU) {
    throw new Error(
      `${manifest.name} must declare os=${expectedOS} and cpu=${expectedCPU}, got ` +
        `os=${manifest.os[0]} and cpu=${manifest.cpu[0]}`,
    );
  }
  const executable = expectedOS === "win32" ? "ag.exe" : "ag";
  const executablePath = `package/bin/${executable}`;
  assertExactFiles(manifest.name, entries, [
    "package/LICENSE",
    executablePath,
    "package/package.json",
  ]);
  if (expectedOS !== "win32") {
    assertExecutableBit(manifest.name, entries, executablePath);
  }
}

async function inspectArtifacts({ npmDir, version }) {
  const normalizedVersion = normalizeVersion(version);
  const checksumPath = path.join(npmDir, "checksums.txt");
  const checksums = parseChecksums(await readFile(checksumPath, "utf8"));
  const tarballNames = (await readdir(npmDir))
    .filter((fileName) => fileName.endsWith(".tgz"))
    .sort();

  const checksumNames = [...checksums.keys()].sort();
  if (JSON.stringify(tarballNames) !== JSON.stringify(checksumNames)) {
    const tarballSet = new Set(tarballNames);
    const checksumSet = new Set(checksumNames);
    const missingChecksums = tarballNames.filter((name) => !checksumSet.has(name));
    const missingTarballs = checksumNames.filter((name) => !tarballSet.has(name));
    const details = [];
    if (missingChecksums.length > 0) {
      details.push(`tarballs missing from checksums.txt: ${missingChecksums.join(", ")}`);
    }
    if (missingTarballs.length > 0) {
      details.push(`checksums without tarballs: ${missingTarballs.join(", ")}`);
    }
    throw new Error(`npm artifact set is incomplete: ${details.join("; ")}`);
  }

  const packages = [];
  for (const fileName of tarballNames) {
    const filePath = path.join(npmDir, fileName);
    const archive = await readFile(filePath);
    const sha256 = digest(archive, "sha256", "hex");
    if (sha256 !== checksums.get(fileName)) {
      throw new Error(`checksum mismatch for ${fileName}`);
    }
    const inspected = await inspectTarball(filePath);
    packages.push({
      ...inspected,
      fileName,
      filePath,
      integrity: `sha512-${digest(archive, "sha512", "base64")}`,
      shasum: digest(archive, "sha1", "hex"),
    });
  }

  const mainPackages = packages.filter(({ manifest }) => manifest.name === MAIN_PACKAGE_NAME);
  if (mainPackages.length !== 1) {
    throw new Error(`expected exactly one ${MAIN_PACKAGE_NAME} tarball, found ${mainPackages.length}`);
  }
  const mainPackage = mainPackages[0];
  validateMainPackage(mainPackage, normalizedVersion);

  const declaredPlatformNames = Object.entries(mainPackage.manifest.optionalDependencies || {})
    .filter(([name]) => name.startsWith(`${MAIN_PACKAGE_NAME}-`))
    .map(([name, dependencyVersion]) => {
      if (dependencyVersion !== normalizedVersion) {
        throw new Error(
          `${MAIN_PACKAGE_NAME} optional dependency ${name}=${dependencyVersion} does not match ${normalizedVersion}`,
        );
      }
      return name;
    })
    .sort();
  const platformNames = [...PLATFORM_PACKAGE_NAMES].sort();
  if (JSON.stringify(declaredPlatformNames) !== JSON.stringify(platformNames)) {
    const declared = new Set(declaredPlatformNames);
    const supported = new Set(platformNames);
    const missing = platformNames.filter((name) => !declared.has(name));
    const unexpected = declaredPlatformNames.filter((name) => !supported.has(name));
    const details = [];
    if (missing.length > 0) details.push(`missing ${missing.join(", ")}`);
    if (unexpected.length > 0) details.push(`unexpected ${unexpected.join(", ")}`);
    throw new Error(
      `${MAIN_PACKAGE_NAME} platform dependencies must exactly match the supported set: ` +
        details.join("; "),
    );
  }

  const packagesByName = new Map();
  for (const packageInfo of packages) {
    const packageName = packageInfo.manifest.name;
    if (!packageName) {
      throw new Error(`${packageInfo.fileName} package.json does not contain a name`);
    }
    if (packagesByName.has(packageName)) {
      throw new Error(`multiple tarballs declare package ${packageName}`);
    }
    packagesByName.set(packageName, packageInfo);
  }

  const expectedNames = new Set([MAIN_PACKAGE_NAME, ...platformNames]);
  const unexpectedNames = [...packagesByName.keys()].filter((name) => !expectedNames.has(name));
  const missingNames = [...expectedNames].filter((name) => !packagesByName.has(name));
  if (missingNames.length > 0 || unexpectedNames.length > 0) {
    const details = [];
    if (missingNames.length > 0) details.push(`missing packages: ${missingNames.join(", ")}`);
    if (unexpectedNames.length > 0) {
      details.push(`unexpected packages: ${unexpectedNames.join(", ")}`);
    }
    throw new Error(`npm package set does not match the main package metadata: ${details.join("; ")}`);
  }

  const platformPackages = platformNames.map((name) => {
    const packageInfo = packagesByName.get(name);
    validatePlatformPackage(packageInfo, PLATFORM_PACKAGES_BY_NAME.get(name), normalizedVersion);
    return packageInfo;
  });
  return {
    version: normalizedVersion,
    npmDir,
    platformPackages,
    mainPackage,
    packages: [...platformPackages, mainPackage],
  };
}

function redactUrlUserInfo(value) {
  // Strip RFC 3986 userinfo (the `user[:password]@` segment that precedes the
  // host) from URLs so credentials supplied via --registry never reach logs or
  // error messages. The host, port, path, query, and fragment are preserved.
  return String(value || "").replace(/([A-Za-z][A-Za-z0-9+.-]*:\/\/)[^/?#]+@/g, "$1");
}

function redactSecrets(value, env = process.env) {
  let redacted = String(value || "");
  for (const key of ["NODE_AUTH_TOKEN", "NPM_TOKEN", "NPM_CONFIG__AUTH", "NPM_CONFIG__AUTHTOKEN"]) {
    const secret = env[key];
    if (secret) redacted = redacted.split(secret).join("[REDACTED]");
  }
  return redacted
    .replace(/(_authToken\s*=\s*)\S+/gi, "$1[REDACTED]")
    .replace(/(authorization:\s*bearer\s+)\S+/gi, "$1[REDACTED]")
    .replace(/([A-Za-z][A-Za-z0-9+.-]*:\/\/)[^/?#]+@/g, "$1");
}

function defaultRunNpm(args) {
  const npmExecPath = process.env.npm_execpath;
  const bundledNpmCli = path.join(
    path.dirname(process.execPath),
    "node_modules",
    "npm",
    "bin",
    "npm-cli.js",
  );
  const npmCli = npmExecPath || (existsSync(bundledNpmCli) ? bundledNpmCli : null);
  const command = npmCli ? process.execPath : process.platform === "win32" ? "npm.cmd" : "npm";
  const commandArgs = npmCli ? [npmCli, ...args] : args;
  const result = spawnSync(command, commandArgs, {
    encoding: "utf8",
    env: process.env,
    windowsHide: true,
  });
  if (result.error) {
    return { status: 1, stdout: result.stdout || "", stderr: result.error.message };
  }
  return {
    status: result.status === null ? 1 : result.status,
    stdout: result.stdout || "",
    stderr: result.stderr || "",
  };
}

async function runNpmCommand(runNpm, args) {
  const result = await runNpm(args);
  return {
    status: result?.status ?? 1,
    stdout: result?.stdout || "",
    stderr: result?.stderr || "",
  };
}

function npmFailure(context, result, env) {
  const details = redactSecrets(result.stderr || result.stdout || `exit status ${result.status}`, env)
    .trim();
  return new Error(`${context}: ${details}`);
}

function isTwoFactorPublishFailure(result) {
  const output = `${result.stdout}\n${result.stderr}`;
  return /\bEOTP\b|one[- ]time password|two[- ]factor authentication|2fa required|bypass 2fa|auth-and-writes/i.test(
    output,
  );
}

function npmPublishFailure(context, result, env) {
  if (isTwoFactorPublishFailure(result)) {
    return new Error(
      `${context}; direct non-interactive publish cannot answer npm's two-factor authentication challenge. ` +
        "Use staged publishing and approve the staged package with 2FA, or run direct publish from a supported Trusted Publishing workflow",
    );
  }
  return npmFailure(context, result, env);
}

function npmStageFailure(context, result, env) {
  const output = `${result.stdout}\n${result.stderr}`;
  if (/unknown command(?::|\s).*['"]?stage/i.test(output)) {
    return npmFailure(
      `${context}; staged publishing requires npm 11.15.0 or later and Node.js 22.14.0 or later`,
      result,
      env,
    );
  }
  return npmFailure(context, result, env);
}

function isNotFoundResult(result) {
  return /(?:E404|404 Not Found|is not in this registry)/i.test(
    `${result.stdout}\n${result.stderr}`,
  );
}

// `npm view <name>@<version> --json` returns the manifest object on modern npm,
// but older clients (notably npm 12) wrap the result in a single-element array.
// Normalize so callers always receive the manifest object, and reject ambiguous
// (empty or multi-element) responses that would otherwise be misread as
// "unknown@unknown" and break the resume-from-breakpoint flow (issue #63).
function normalizeRegistryMetadata(parsed, spec) {
  if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
    return parsed;
  }
  if (Array.isArray(parsed)) {
    if (parsed.length === 1 && parsed[0] && typeof parsed[0] === "object") {
      return parsed[0];
    }
    throw new Error(
      `registry metadata for ${spec} returned an ambiguous array response with ${parsed.length} entr${parsed.length === 1 ? "y" : "ies"}`,
    );
  }
  throw new Error(
    `registry returned unexpected metadata type for ${spec}: ${parsed === null ? "null" : typeof parsed}`,
  );
}

async function lookupRemotePackage(packageInfo, { registry, runNpm, env }) {
  const spec = `${packageInfo.manifest.name}@${packageInfo.manifest.version}`;
  const result = await runNpmCommand(runNpm, ["view", spec, "--json", "--registry", registry]);
  if (result.status !== 0) {
    if (isNotFoundResult(result)) return null;
    throw npmFailure(`failed to query ${spec}`, result, env);
  }
  let parsed;
  try {
    parsed = JSON.parse(result.stdout);
  } catch (error) {
    throw new Error(`registry returned invalid JSON for ${spec}: ${error.message}`);
  }
  return normalizeRegistryMetadata(parsed, spec);
}

function normalizeStagedPackages(parsed, packageInfo) {
  if (!Array.isArray(parsed)) {
    throw new Error(`npm stage list for ${packageInfo.manifest.name} did not return an array`);
  }
  const matches = parsed.filter(
    (item) =>
      item?.packageName === packageInfo.manifest.name &&
      item?.version === packageInfo.manifest.version,
  );
  if (matches.length > 1) {
    throw new Error(
      `npm stage list returned multiple staged entries for ${packageInfo.manifest.name}@${packageInfo.manifest.version}`,
    );
  }
  if (matches.length === 0) return null;

  const staged = matches[0];
  if (!staged.id) {
    throw new Error(
      `staged ${packageInfo.manifest.name}@${packageInfo.manifest.version} does not include a stage id`,
    );
  }
  if (!staged.shasum) {
    throw new Error(
      `staged ${packageInfo.manifest.name}@${packageInfo.manifest.version} does not include a shasum`,
    );
  }
  if (staged.shasum !== packageInfo.shasum) {
    throw new Error(
      `staged ${packageInfo.manifest.name}@${packageInfo.manifest.version} has conflicting shasum`,
    );
  }
  return staged;
}

async function lookupStagedPackage(packageInfo, { registry, runNpm, env }) {
  const result = await runNpmCommand(runNpm, [
    "stage",
    "list",
    packageInfo.manifest.name,
    "--json",
    "--registry",
    registry,
  ]);
  if (result.status !== 0) {
    throw npmStageFailure(`failed to list staged versions of ${packageInfo.manifest.name}`, result, env);
  }
  let parsed;
  try {
    parsed = JSON.parse(result.stdout);
  } catch (error) {
    throw new Error(
      `npm stage list returned invalid JSON for ${packageInfo.manifest.name}: ${error.message}`,
    );
  }
  return normalizeStagedPackages(parsed, packageInfo);
}

function assertRemoteMatches(packageInfo, metadata) {
  const expectedName = packageInfo.manifest.name;
  const expectedVersion = packageInfo.manifest.version;
  if (metadata?.name !== expectedName || metadata?.version !== expectedVersion) {
    throw new Error(
      `registry metadata for ${expectedName}@${expectedVersion} returned ` +
        `${metadata?.name || "unknown"}@${metadata?.version || "unknown"}`,
    );
  }
  const remoteIntegrity = metadata?.dist?.integrity;
  const remoteShasum = metadata?.dist?.shasum;
  if (!remoteIntegrity && !remoteShasum) {
    throw new Error(
      `registry metadata for ${expectedName}@${expectedVersion} has no integrity or shasum`,
    );
  }
  if (remoteIntegrity && remoteIntegrity !== packageInfo.integrity) {
    throw new Error(`published ${expectedName}@${expectedVersion} has conflicting integrity`);
  }
  if (remoteShasum && remoteShasum !== packageInfo.shasum) {
    throw new Error(`published ${expectedName}@${expectedVersion} has conflicting shasum`);
  }
}

async function verifyPublishedPackage(packageInfo, options) {
  const attempts = options.verifyAttempts ?? DEFAULT_VERIFY_ATTEMPTS;
  const delayMs = options.verifyDelayMs ?? DEFAULT_VERIFY_DELAY_MS;
  const sleep = options.sleep || ((milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds)));

  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    const metadata = await lookupRemotePackage(packageInfo, options);
    if (metadata) {
      assertRemoteMatches(packageInfo, metadata);
      return;
    }
    if (attempt < attempts) await sleep(delayMs);
  }
  throw new Error(
    `${packageInfo.manifest.name}@${packageInfo.manifest.version} was not visible after publication`,
  );
}

async function publishPackage(packageInfo, options) {
  const { registry, runNpm, env } = options;
  const result = await runNpmCommand(runNpm, [
    "publish",
    packageInfo.filePath,
    "--access",
    "public",
    "--ignore-scripts",
    "--registry",
    registry,
  ]);
  if (result.status !== 0) {
    throw npmPublishFailure(
      `failed to publish ${packageInfo.manifest.name}@${packageInfo.manifest.version}`,
      result,
      env,
    );
  }
  await verifyPublishedPackage(packageInfo, options);
}

async function verifyStagedPackage(packageInfo, options) {
  const attempts = options.verifyAttempts ?? DEFAULT_VERIFY_ATTEMPTS;
  const delayMs = options.verifyDelayMs ?? DEFAULT_VERIFY_DELAY_MS;
  const sleep =
    options.sleep || ((milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds)));

  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    const staged = await lookupStagedPackage(packageInfo, options);
    if (staged) return staged;
    if (attempt < attempts) await sleep(delayMs);
  }
  throw new Error(
    `${packageInfo.manifest.name}@${packageInfo.manifest.version} was not visible in npm stage list after staging`,
  );
}

async function stagePackage(packageInfo, options) {
  const { registry, runNpm, env } = options;
  const result = await runNpmCommand(runNpm, [
    "stage",
    "publish",
    packageInfo.filePath,
    "--access",
    "public",
    "--ignore-scripts",
    "--json",
    "--registry",
    registry,
  ]);
  if (result.status !== 0) {
    throw npmStageFailure(
      `failed to stage ${packageInfo.manifest.name}@${packageInfo.manifest.version}`,
      result,
      env,
    );
  }
  return verifyStagedPackage(packageInfo, options);
}

async function smokeTestPublishedPackage(plan, options) {
  const installDir = await mkdtemp(path.join(os.tmpdir(), "ag-npm-smoke-"));
  const packageSpec = `${MAIN_PACKAGE_NAME}@${plan.version}`;
  try {
    const install = await runNpmCommand(options.runNpm, [
      "install",
      packageSpec,
      "--ignore-scripts",
      "--no-audit",
      "--no-fund",
      "--prefix",
      installDir,
      "--registry",
      options.registry,
    ]);
    if (install.status !== 0) {
      throw npmFailure(`failed to install ${packageSpec} for smoke verification`, install, options.env);
    }

    const versionResult = await runNpmCommand(options.runNpm, [
      "exec",
      "--yes=false",
      "--prefix",
      installDir,
      "--",
      "ag",
      "version",
      "--json",
    ]);
    if (versionResult.status !== 0) {
      throw npmFailure(`installed ${packageSpec} failed to execute ag version`, versionResult, options.env);
    }

    let versionInfo;
    try {
      versionInfo = JSON.parse(versionResult.stdout);
    } catch (error) {
      throw new Error(`installed ${packageSpec} returned invalid version JSON: ${error.message}`);
    }
    const expectedVersion = `v${plan.version}`;
    if (versionInfo?.version !== expectedVersion) {
      throw new Error(
        `installed ${packageSpec} reported version ${JSON.stringify(versionInfo?.version)}; ` +
          `expected ${expectedVersion}`,
      );
    }
  } finally {
    await rm(installDir, { recursive: true, force: true });
  }
}

async function publishArtifacts(plan, options = {}) {
  const registry = (options.registry || DEFAULT_REGISTRY).replace(/\/+$/, "");
  const mode = options.mode || PUBLISH_MODE_STAGE;
  if (![PUBLISH_MODE_STAGE, PUBLISH_MODE_DIRECT].includes(mode)) {
    throw new Error(`unsupported publish mode ${JSON.stringify(mode)}`);
  }
  // The real registry URL (which may carry embedded credentials) is only handed
  // to npm itself; every human-facing log or error uses this redacted form.
  const safeRegistry = redactUrlUserInfo(registry);
  const runNpm = options.runNpm || defaultRunNpm;
  const logger = options.logger || console.log;
  const env = options.env || process.env;
  const runtime = { ...options, mode, registry, runNpm, env };

  if (options.dryRun) {
    const action = mode === PUBLISH_MODE_STAGE ? "stage" : "publish";
    logger(`Validated ${plan.packages.length} npm tarballs for ${plan.version}.`);
    for (const packageInfo of plan.platformPackages) {
      logger(`Would ${action} platform package ${packageInfo.manifest.name}@${plan.version}`);
    }
    if (mode === PUBLISH_MODE_STAGE) {
      logger(
        `Would stage main package ${plan.mainPackage.manifest.name}@${plan.version} ` +
          "on a later run after every platform version is public.",
      );
    } else {
      logger(`Would publish main package ${plan.mainPackage.manifest.name}@${plan.version} last.`);
    }
    logger("Dry run complete; no registry requests were made.");
    return { published: [], staged: [], skipped: [], pending: [] };
  }

  const remoteState = new Map();
  for (const packageInfo of plan.packages) {
    const metadata = await lookupRemotePackage(packageInfo, runtime);
    if (metadata) {
      assertRemoteMatches(packageInfo, metadata);
      remoteState.set(packageInfo.manifest.name, { kind: "published" });
      continue;
    }
    if (mode === PUBLISH_MODE_STAGE) {
      const staged = await lookupStagedPackage(packageInfo, runtime);
      if (staged) {
        remoteState.set(packageInfo.manifest.name, { kind: "staged", staged });
        continue;
      }
    }
    remoteState.set(packageInfo.manifest.name, { kind: "missing" });
  }

  if (mode === PUBLISH_MODE_STAGE) {
    const staged = [];
    const skipped = [];
    const pending = [];
    const unpublishedPlatforms = plan.platformPackages.filter(
      (packageInfo) => remoteState.get(packageInfo.manifest.name).kind !== "published",
    );
    const mainState = remoteState.get(plan.mainPackage.manifest.name);

    if (unpublishedPlatforms.length > 0 && mainState.kind !== "missing") {
      const mainSpec = `${plan.mainPackage.manifest.name}@${plan.version}`;
      if (mainState.kind === "staged") {
        throw new Error(
          `${mainSpec} is already staged as ${mainState.staged.id} before every platform ` +
            "package is public; reject that main-package stage before continuing",
        );
      }
      throw new Error(`${mainSpec} is already public while platform packages are missing`);
    }

    const stageCandidates =
      unpublishedPlatforms.length > 0 ? plan.platformPackages : [plan.mainPackage];
    for (const packageInfo of stageCandidates) {
      const packageSpec = `${packageInfo.manifest.name}@${plan.version}`;
      const state = remoteState.get(packageInfo.manifest.name);
      if (state.kind === "published") {
        logger(`Skipping ${packageSpec}; the exact tarball is already published.`);
        skipped.push(packageInfo.manifest.name);
        continue;
      }
      if (state.kind === "staged") {
        logger(`Skipping ${packageSpec}; the exact tarball is already staged as ${state.staged.id}.`);
        pending.push(state.staged);
        continue;
      }
      logger(`Staging ${packageSpec} for maintainer approval...`);
      const stagedPackage = await stagePackage(packageInfo, runtime);
      logger(`Staged ${packageSpec} as ${stagedPackage.id}.`);
      staged.push(packageInfo.manifest.name);
      pending.push(stagedPackage);
    }

    if (unpublishedPlatforms.length === 0 && mainState.kind === "published") {
      const mainSpec = `${plan.mainPackage.manifest.name}@${plan.version}`;
      logger(`Installing ${mainSpec} for an isolated version smoke test...`);
      await smokeTestPublishedPackage(plan, runtime);
      logger(`Installed ${mainSpec} and verified ag version v${plan.version}.`);
      logger(`npm publication complete for ${plan.version}.`);
      return { published: [], staged, skipped, pending };
    }

    logger("Staging complete; staged packages are not public until a maintainer approves them with 2FA.");
    for (const packageInfo of plan.platformPackages) {
      const item = pending.find(
        (candidate) => candidate.packageName === packageInfo.manifest.name,
      );
      if (!item) continue;
      logger(`Review ${item.packageName}@${item.version}: npm stage view ${item.id} --registry ${safeRegistry}`);
      logger(`Approve ${item.packageName}@${item.version}: npm stage approve ${item.id} --registry ${safeRegistry}`);
    }
    const mainStage = pending.find(
      (candidate) => candidate.packageName === plan.mainPackage.manifest.name,
    );
    if (mainStage) {
      logger(
        `Approve the main package: npm stage approve ${mainStage.id} --registry ${safeRegistry}`,
      );
    } else {
      logger(
        "Approve every staged platform package, then rerun this command; " +
          "the main package will not be staged until every platform version is public.",
      );
    }
    logger("Rerun this command after approval to verify registry integrity and the installed CLI version.");
    return { published: [], staged, skipped, pending };
  }

  const published = [];
  const skipped = [];
  for (const packageInfo of plan.platformPackages) {
    const packageSpec = `${packageInfo.manifest.name}@${plan.version}`;
    if (remoteState.get(packageInfo.manifest.name).kind === "published") {
      logger(`Skipping ${packageSpec}; the exact tarball is already published.`);
      skipped.push(packageInfo.manifest.name);
      continue;
    }
    logger(`Publishing platform package ${packageSpec}...`);
    await publishPackage(packageInfo, runtime);
    logger(`Published and verified ${packageSpec}.`);
    published.push(packageInfo.manifest.name);
  }

  for (const packageInfo of plan.platformPackages) {
    const metadata = await lookupRemotePackage(packageInfo, runtime);
    if (!metadata) {
      throw new Error(
        `${packageInfo.manifest.name}@${plan.version} is not visible; refusing to publish the main package`,
      );
    }
    assertRemoteMatches(packageInfo, metadata);
  }

  const mainSpec = `${plan.mainPackage.manifest.name}@${plan.version}`;
  if (remoteState.get(plan.mainPackage.manifest.name).kind === "published") {
    logger(`Skipping ${mainSpec}; the exact tarball is already published.`);
    skipped.push(plan.mainPackage.manifest.name);
  } else {
    logger(`Publishing main package ${mainSpec}...`);
    await publishPackage(plan.mainPackage, runtime);
    logger(`Published and verified ${mainSpec}.`);
    published.push(plan.mainPackage.manifest.name);
  }

  logger(`Installing ${mainSpec} for an isolated version smoke test...`);
  await smokeTestPublishedPackage(plan, runtime);
  logger(`Installed ${mainSpec} and verified ag version v${plan.version}.`);

  logger(`npm publication complete for ${plan.version}.`);
  return { published, staged: [], skipped, pending: [] };
}

function usage() {
  return `Usage:
  npm run publish:npm -- vX.Y.Z dist/vX.Y.Z/npm [--stage] [--dry-run]
  npm run publish:npm -- vX.Y.Z dist/vX.Y.Z/npm --publish
  node scripts/publish-npm-packages.js --version vX.Y.Z --dir dist/vX.Y.Z/npm [--stage] [--dry-run]

Options:
  --version <version>   Release version in vX.Y.Z or X.Y.Z form
  --dir <directory>    Directory containing npm tarballs and checksums.txt
  --registry <url>     npm registry URL (default: ${DEFAULT_REGISTRY})
  --stage              Stage packages for separate maintainer 2FA approval (default)
  --publish            Publish directly; use only from a supported Trusted Publishing workflow
  --dry-run            Validate artifacts and print publication order without network access
  -h, --help           Show this help`;
}

async function main(args = process.argv.slice(2)) {
  const options = parseArguments(args);
  if (options.help) {
    console.log(usage());
    return;
  }
  const plan = await inspectArtifacts({ npmDir: options.npmDir, version: options.version });
  await publishArtifacts(plan, options);
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`Error: ${redactSecrets(error.message)}`);
    process.exitCode = 1;
  });
}

module.exports = {
  DEFAULT_REGISTRY,
  MAIN_PACKAGE_NAME,
  PUBLISH_MODE_DIRECT,
  PUBLISH_MODE_STAGE,
  assertRemoteMatches,
  inspectArtifacts,
  normalizeRegistryMetadata,
  normalizeStagedPackages,
  normalizeVersion,
  parseArguments,
  parseChecksums,
  publishArtifacts,
  redactSecrets,
  redactUrlUserInfo,
  smokeTestPublishedPackage,
};
