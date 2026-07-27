#!/usr/bin/env node

const { spawnSync } = require("node:child_process");
const { chmod, copyFile, mkdir, readFile, rm, writeFile } = require("node:fs/promises");
const path = require("node:path");
const AdmZip = require("adm-zip");
const tar = require("tar");

const PACKAGE_SCOPE = "@hust-open-atom-club";
const TARGETS = [
  {
    platform: "linux",
    arch: "x64",
    executable: "ag",
    archive: "ag_linux_amd64_npm.tar.gz",
  },
  {
    platform: "linux",
    arch: "arm64",
    executable: "ag",
    archive: "ag_linux_arm64_npm.tar.gz",
  },
  {
    platform: "linux",
    arch: "loong64",
    executable: "ag",
    archive: "ag_linux_loong64_npm.tar.gz",
  },
  {
    platform: "darwin",
    arch: "x64",
    executable: "ag",
    archive: "ag_darwin_amd64_npm.tar.gz",
  },
  {
    platform: "darwin",
    arch: "arm64",
    executable: "ag",
    archive: "ag_darwin_arm64_npm.tar.gz",
  },
  {
    platform: "win32",
    arch: "x64",
    executable: "ag.exe",
    archive: "ag_windows_amd64_npm.zip",
  },
  {
    platform: "win32",
    arch: "arm64",
    executable: "ag.exe",
    archive: "ag_windows_arm64_npm.zip",
  },
].map((target) => ({
  ...target,
  packageName: `${PACKAGE_SCOPE}/atomgit-cli-${target.platform}-${target.arch}`,
  directoryName: `${target.platform}-${target.arch}`,
}));

function platformPackageManifest(target, version) {
  return {
    name: target.packageName,
    version,
    description: `AtomGit CLI binary for ${target.platform}/${target.arch}`,
    license: "MulanPSL-2.0",
    repository: {
      type: "git",
      url: "git+https://atomgit.com/hust-open-atom-club/atomgit-cli.git",
    },
    homepage: "https://atomgit.com/hust-open-atom-club/atomgit-cli",
    os: [target.platform],
    cpu: [target.arch],
    files: [`bin/${target.executable}`],
    publishConfig: { access: "public" },
  };
}

async function extractBinary(archivePath, target, destination) {
  if (archivePath.endsWith(".zip")) {
    const entry = new AdmZip(archivePath).getEntry(target.executable);
    if (!entry) {
      throw new Error(`${target.archive} does not contain ${target.executable}`);
    }
    await writeFile(destination, entry.getData());
  } else {
    await tar.x({
      cwd: path.dirname(destination),
      file: archivePath,
      strict: true,
    }, [target.executable]);
  }
  await chmod(destination, 0o755);
}

function packPackage(source, destination, cache) {
  const npm = process.platform === "win32" ? "npm.cmd" : "npm";
  const result = spawnSync(
    npm,
    ["pack", source, "--pack-destination", destination, "--ignore-scripts", "--cache", cache],
    { stdio: "inherit" },
  );
  if (result.error) {
    throw result.error;
  }
  if (result.signal) {
    throw new Error(`npm pack terminated by signal ${result.signal}`);
  }
  if (result.status !== 0) {
    throw new Error(`npm pack exited with status ${result.status}`);
  }
}

async function buildNpmPackages({ root, releaseDir, version, pack = true }) {
  const packageJson = JSON.parse(await readFile(path.join(root, "package.json"), "utf8"));
  if (packageJson.version !== version) {
    throw new Error(
      `package.json version ${JSON.stringify(packageJson.version)} does not match ${version}`,
    );
  }

  for (const target of TARGETS) {
    if (packageJson.optionalDependencies?.[target.packageName] !== version) {
      throw new Error(`optional dependency ${target.packageName} must be pinned to ${version}`);
    }
  }

  const npmDir = path.join(releaseDir, "npm");
  const packagesDir = path.join(npmDir, "packages");
  const cacheDir = path.join(npmDir, ".cache");
  await rm(npmDir, { recursive: true, force: true });
  await mkdir(packagesDir, { recursive: true });

  for (const target of TARGETS) {
    const packageDir = path.join(packagesDir, target.directoryName);
    const binDir = path.join(packageDir, "bin");
    await mkdir(binDir, { recursive: true });
    await writeFile(
      path.join(packageDir, "package.json"),
      `${JSON.stringify(platformPackageManifest(target, version), null, 2)}\n`,
    );
    await copyFile(path.join(root, "LICENSE"), path.join(packageDir, "LICENSE"));
    await extractBinary(
      path.join(releaseDir, "package-managers", target.archive),
      target,
      path.join(binDir, target.executable),
    );
  }

  if (pack) {
    for (const target of TARGETS) {
      packPackage(path.join(packagesDir, target.directoryName), npmDir, cacheDir);
    }
    packPackage(root, npmDir, cacheDir);
    await rm(packagesDir, { recursive: true, force: true });
    await rm(cacheDir, { recursive: true, force: true });
  }

  return npmDir;
}

async function main(args = process.argv.slice(2)) {
  if (args.length !== 2) {
    throw new Error("usage: node scripts/build-npm-packages.js <release-dir> <version>");
  }
  const root = path.join(__dirname, "..");
  const npmDir = await buildNpmPackages({
    root,
    releaseDir: path.resolve(args[0]),
    version: args[1],
  });
  console.log(`Generated npm packages in ${npmDir}`);
}

if (require.main === module) {
  main().catch((error) => {
    console.error(`Error: ${error.message}`);
    process.exitCode = 1;
  });
}

module.exports = { TARGETS, buildNpmPackages, extractBinary, platformPackageManifest };
