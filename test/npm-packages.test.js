const test = require("node:test");
const assert = require("node:assert/strict");
const { mkdir, mkdtemp, readFile, rm, writeFile } = require("node:fs/promises");
const path = require("node:path");
const AdmZip = require("adm-zip");
const tar = require("tar");

const { resolveBinary, run } = require("../bin/ag.js");
const {
  TARGETS,
  buildNpmPackages,
  extractBinary,
  platformPackageManifest,
} = require("../scripts/build-npm-packages");
const { assertPackageVersions } = require("../scripts/check-npm-version");
const { setNpmVersion } = require("../scripts/set-npm-version");

function versionMetadata(version) {
  const optionalDependencies = Object.fromEntries(
    TARGETS.map(({ packageName }) => [packageName, version]),
  );
  const packages = {
    "": { version, optionalDependencies: { ...optionalDependencies } },
  };
  for (const target of TARGETS) {
    packages[`node_modules/${target.packageName}`] = {
      version,
      cpu: [target.arch],
      optional: true,
      os: [target.platform],
    };
  }
  return {
    packageJson: { version, optionalDependencies: { ...optionalDependencies } },
    packageLock: { version, packages },
  };
}

test("resolves every supported platform package", () => {
  for (const target of TARGETS) {
    const calls = [];
    const binary = resolveBinary(target.platform, target.arch, (specifier) => {
      calls.push(specifier);
      return `/node_modules/${specifier}`;
    });

    assert.deepEqual(calls, [`${target.packageName}/bin/${target.executable}`]);
    assert.equal(binary, `/node_modules/${target.packageName}/bin/${target.executable}`);
  }
});

test("rejects unsupported platforms and architectures", () => {
  assert.throws(() => resolveBinary("freebsd", "x64"), /unsupported platform: freebsd\/x64/);
  assert.throws(() => resolveBinary("linux", "ia32"), /unsupported platform: linux\/ia32/);
});

test("reports a missing optional platform package", () => {
  assert.throws(
    () =>
      resolveBinary("linux", "x64", () => {
        throw new Error("not found");
      }),
    /atomgit-cli-linux-x64 is missing; reinstall without omitting optional dependencies/,
  );
});

test("forwards arguments and the child exit code to the platform binary", () => {
  const calls = [];
  const result = run(["version", "--json"], {
    platform: "linux",
    arch: "x64",
    resolve(specifier) {
      assert.equal(specifier, "@hust-open-atom-club/atomgit-cli-linux-x64/bin/ag");
      return "/platform-package/bin/ag";
    },
    spawn(command, args, options) {
      calls.push({ command, args, options });
      return { status: 7, signal: null };
    },
  });

  assert.equal(result, 7);
  assert.deepEqual(calls, [
    {
      command: "/platform-package/bin/ag",
      args: ["version", "--json"],
      options: { stdio: "inherit" },
    },
  ]);
});

test("creates platform-specific npm metadata", () => {
  const target = TARGETS.find(({ platform, arch }) => platform === "linux" && arch === "x64");
  assert.deepEqual(platformPackageManifest(target, "1.2.3"), {
    name: "@hust-open-atom-club/atomgit-cli-linux-x64",
    version: "1.2.3",
    description: "AtomGit CLI binary for linux/x64",
    license: "MulanPSL-2.0",
    repository: {
      type: "git",
      url: "git+https://atomgit.com/hust-open-atom-club/atomgit-cli.git",
    },
    homepage: "https://atomgit.com/hust-open-atom-club/atomgit-cli",
    os: ["linux"],
    cpu: ["x64"],
    files: ["bin/ag"],
    publishConfig: { access: "public" },
  });
});

test("extracts binaries from release tar.gz and zip archives", async (t) => {
  const root = await mkdtemp(path.join(process.cwd(), ".ag-npm-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const source = path.join(root, "source");
  const output = path.join(root, "output");
  await mkdir(source);
  await mkdir(output);
  await writeFile(path.join(source, "ag"), "unix binary");
  await writeFile(path.join(source, "ag.exe"), "windows binary");

  const unixTarget = TARGETS.find(({ platform, arch }) => platform === "linux" && arch === "x64");
  const tarPath = path.join(root, unixTarget.archive);
  await tar.c({ cwd: source, file: tarPath, gzip: true }, ["ag"]);
  const unixDestination = path.join(output, "ag");
  await extractBinary(tarPath, unixTarget, unixDestination);
  assert.equal(await readFile(unixDestination, "utf8"), "unix binary");

  const windowsTarget = TARGETS.find(
    ({ platform, arch }) => platform === "win32" && arch === "x64",
  );
  const zipPath = path.join(root, windowsTarget.archive);
  const zip = new AdmZip();
  zip.addFile("ag.exe", Buffer.from("windows binary"));
  zip.writeZip(zipPath);
  const windowsDestination = path.join(output, "ag.exe");
  await extractBinary(zipPath, windowsTarget, windowsDestination);
  assert.equal(await readFile(windowsDestination, "utf8"), "windows binary");
});

test("builds all seven platform package directories from release archives", async (t) => {
  const root = await mkdtemp(path.join(process.cwd(), ".ag-npm-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const releaseDir = path.join(root, "release");
  const managedDir = path.join(releaseDir, "package-managers");
  const sourceDir = path.join(root, "source");
  await mkdir(releaseDir);
  await mkdir(managedDir);
  await mkdir(sourceDir);

  for (const target of TARGETS) {
    assert.match(target.archive, /_npm\.(?:tar\.gz|zip)$/);
    const contents = `${target.platform}/${target.arch}`;
    await writeFile(path.join(sourceDir, target.executable), contents);
    const archivePath = path.join(managedDir, target.archive);
    if (target.archive.endsWith(".zip")) {
      const zip = new AdmZip();
      zip.addFile(target.executable, Buffer.from(contents));
      zip.writeZip(archivePath);
    } else {
      await tar.c(
        { cwd: sourceDir, file: archivePath, gzip: true },
        [target.executable],
      );
    }
  }

  await buildNpmPackages({
    root: path.join(__dirname, ".."),
    releaseDir,
    version: require("../package.json").version,
    pack: false,
  });

  for (const target of TARGETS) {
    const packageDir = path.join(releaseDir, "npm", "packages", target.directoryName);
    const manifest = JSON.parse(await readFile(path.join(packageDir, "package.json"), "utf8"));
    assert.equal(manifest.name, target.packageName);
    assert.equal(manifest.version, require("../package.json").version);
    assert.deepEqual(manifest.os, [target.platform]);
    assert.deepEqual(manifest.cpu, [target.arch]);
    assert.equal(
      await readFile(path.join(packageDir, "bin", target.executable), "utf8"),
      `${target.platform}/${target.arch}`,
    );
  }
});

test("all npm targets use managed archives instead of ordinary release archives", () => {
  assert.deepEqual(
    TARGETS.map(({ archive }) => archive),
    [
      "ag_linux_amd64_npm.tar.gz",
      "ag_linux_arm64_npm.tar.gz",
      "ag_linux_loong64_npm.tar.gz",
      "ag_darwin_amd64_npm.tar.gz",
      "ag_darwin_arm64_npm.tar.gz",
      "ag_windows_amd64_npm.zip",
      "ag_windows_arm64_npm.zip",
    ],
  );
  for (const target of TARGETS) {
    assert.notEqual(
      target.archive,
      target.archive.replace("_npm", ""),
      `${target.platform}/${target.arch} must use the npm profile`,
    );
  }
});

test("accepts release metadata when all npm versions match", () => {
  const version = "1.2.3";
  const { packageJson, packageLock } = versionMetadata(version);
  assert.equal(
    assertPackageVersions(version, packageJson, packageLock),
    version,
  );
});

test("rejects stale platform package versions", () => {
  const version = "1.2.3";
  const { packageJson, packageLock } = versionMetadata(version);
  packageJson.optionalDependencies[TARGETS[0].packageName] = "1.2.2";
  assert.throws(
    () => assertPackageVersions(version, packageJson, packageLock),
    /atomgit-cli-linux-x64|atomgit-cli-darwin-arm64/,
  );
});

test("rejects a missing platform package lock entry", () => {
  const version = "1.2.3";
  const { packageJson, packageLock } = versionMetadata(version);
  delete packageLock.packages[`node_modules/${TARGETS[0].packageName}`];

  assert.throws(
    () => assertPackageVersions(version, packageJson, packageLock),
    /package-lock\.json packages\["node_modules\/.*"\]=undefined/,
  );
});

test("updates the main and platform package versions together", () => {
  const { packageJson, packageLock } = versionMetadata("1.2.3");

  setNpmVersion("2.0.0-beta.1", packageJson, packageLock);

  assert.equal(packageJson.version, "2.0.0-beta.1");
  assert.equal(packageLock.version, "2.0.0-beta.1");
  assert.equal(packageLock.packages[""].version, "2.0.0-beta.1");
  for (const target of TARGETS) {
    assert.equal(packageJson.optionalDependencies[target.packageName], "2.0.0-beta.1");
    assert.equal(
      packageLock.packages[""].optionalDependencies[target.packageName],
      "2.0.0-beta.1",
    );
    assert.deepEqual(packageLock.packages[`node_modules/${target.packageName}`], {
      version: "2.0.0-beta.1",
      cpu: [target.arch],
      optional: true,
      os: [target.platform],
    });
  }
});
