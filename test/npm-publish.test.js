const test = require("node:test");
const assert = require("node:assert/strict");
const { createHash } = require("node:crypto");
const {
  copyFile,
  mkdir,
  mkdtemp,
  readFile,
  rm,
  writeFile,
} = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const tar = require("tar");
const { PLATFORM_PACKAGES } = require("../scripts/check-npm-version");

const {
  MAIN_PACKAGE_NAME,
  PUBLISH_MODE_DIRECT,
  PUBLISH_MODE_STAGE,
  inspectArtifacts,
  normalizeRegistryMetadata,
  normalizeStagedPackages,
  normalizeVersion,
  parseArguments,
  publishArtifacts,
  redactUrlUserInfo,
} = require("../scripts/publish-npm-packages");

const PLATFORM_FIXTURES = PLATFORM_PACKAGES.map((target) => ({
  ...target,
  executable: target.os === "win32" ? "ag.exe" : "ag",
}));

function hash(buffer, algorithm, encoding) {
  return createHash(algorithm).update(buffer).digest(encoding);
}

async function createTarball(npmDir, index, manifest, files, executable = []) {
  const source = path.join(npmDir, `.fixture-${index}`);
  const packageDir = path.join(source, "package");
  await mkdir(packageDir, { recursive: true });
  await writeFile(path.join(packageDir, "package.json"), `${JSON.stringify(manifest, null, 2)}\n`);
  for (const [fileName, contents] of Object.entries(files)) {
    const destination = path.join(packageDir, fileName);
    await mkdir(path.dirname(destination), { recursive: true });
    await writeFile(destination, contents, executable.includes(fileName) ? { mode: 0o755 } : {});
  }
  const fileName = `artifact-${index}.tgz`;
  const executableEntries = new Set(executable.map((fileName) => `package/${fileName}`));
  await tar.c(
    {
      cwd: source,
      file: path.join(npmDir, fileName),
      gzip: true,
      filter(entryPath, entry) {
        if (executableEntries.has(entryPath.replaceAll("\\", "/"))) {
          entry.mode = (entry.mode & ~0o777) | 0o755;
        }
        return true;
      },
    },
    ["package"],
  );
  await rm(source, { recursive: true, force: true });
  return fileName;
}

async function writeChecksums(npmDir, fileNames) {
  const lines = [];
  for (const fileName of [...fileNames].reverse()) {
    const contents = await readFile(path.join(npmDir, fileName));
    lines.push(`${hash(contents, "sha256", "hex")}  ./${fileName}`);
  }
  await writeFile(path.join(npmDir, "checksums.txt"), `${lines.join("\n")}\n`);
}

async function createArtifactSet(t, options = {}) {
  const version = options.version || "1.2.3";
  const root = await mkdtemp(path.join(os.tmpdir(), "ag-npm-publish-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const npmDir = path.join(root, "npm");
  await mkdir(npmDir);

  const platformFixtures = options.platformFixtures || PLATFORM_FIXTURES;
  const fileNames = [];
  for (const [index, platform] of platformFixtures.entries()) {
    const binName = `bin/${platform.executable}`;
    const files = {
      LICENSE: "license",
      [binName]: `${platform.os}/${platform.cpu}`,
    };
    if (options.omitBinaryFor === platform.name) delete files[binName];
    const executable =
      platform.os !== "win32" && options.nonExecutableBinaryFor !== platform.name ? [binName] : [];
    fileNames.push(
      await createTarball(
        npmDir,
        index,
        {
          name: platform.name,
          version,
          os: [platform.os],
          cpu: [platform.cpu],
          files: [`bin/${platform.executable}`],
        },
        files,
        executable,
      ),
    );
  }

  const optionalDependencies = Object.fromEntries(
    platformFixtures.map(({ name }) => [name, version]),
  );
  fileNames.push(
    await createTarball(
      npmDir,
      platformFixtures.length,
      {
        name: MAIN_PACKAGE_NAME,
        version,
        bin: { ag: "bin/ag.js" },
        files: ["bin/ag.js"],
        optionalDependencies,
      },
      {
        LICENSE: "license",
        "README.md": "readme",
        "bin/ag.js": "launcher",
      },
    ),
  );
  await writeChecksums(npmDir, fileNames);
  return { fileNames, npmDir, version };
}

function packageInfo(name, version = "1.2.3") {
  const key = name.replaceAll("/", "-").replaceAll("@", "");
  return {
    manifest: { name, version },
    filePath: `C:\\artifacts\\${key}.tgz`,
    integrity: `sha512-${key}`,
    shasum: `sha1-${key}`,
  };
}

function publicationPlan() {
  const platformPackages = [
    packageInfo(`${MAIN_PACKAGE_NAME}-linux-arm64`),
    packageInfo(`${MAIN_PACKAGE_NAME}-linux-x64`),
  ];
  const mainPackage = packageInfo(MAIN_PACKAGE_NAME);
  return {
    version: "1.2.3",
    platformPackages,
    mainPackage,
    packages: [...platformPackages, mainPackage],
  };
}

function remoteMetadata(packageValue, overrides = {}) {
  return {
    name: packageValue.manifest.name,
    version: packageValue.manifest.version,
    dist: {
      integrity: packageValue.integrity,
      shasum: packageValue.shasum,
    },
    ...overrides,
  };
}

function registryRunner(plan, options = {}) {
  const calls = [];
  const packagesBySpec = new Map(
    plan.packages.map((item) => [`${item.manifest.name}@${item.manifest.version}`, item]),
  );
  const packagesByPath = new Map(plan.packages.map((item) => [item.filePath, item]));
  const registry = new Map();
  const staged = new Map();
  const stageIds = new Map(
    plan.packages.map((item, index) => [
      item.manifest.name,
      `00000000-0000-4000-8000-${String(index + 1).padStart(12, "0")}`,
    ]),
  );
  for (const packageName of options.existing || []) {
    const packageValue = plan.packages.find((item) => item.manifest.name === packageName);
    registry.set(packageName, remoteMetadata(packageValue));
  }
  if (options.conflict) {
    const packageValue = plan.packages.find((item) => item.manifest.name === options.conflict);
    registry.set(options.conflict, remoteMetadata(packageValue, { dist: { integrity: "wrong" } }));
  }
  for (const packageName of options.staged || []) {
    const packageValue = plan.packages.find((item) => item.manifest.name === packageName);
    staged.set(packageName, {
      id: stageIds.get(packageName),
      packageName,
      version: packageValue.manifest.version,
      tag: "latest",
      shasum: packageValue.shasum,
    });
  }
  if (options.stagedConflict) {
    const packageValue = plan.packages.find(
      (item) => item.manifest.name === options.stagedConflict,
    );
    staged.set(options.stagedConflict, {
      id: stageIds.get(options.stagedConflict),
      packageName: options.stagedConflict,
      version: packageValue.manifest.version,
      tag: "latest",
      shasum: "wrong",
    });
  }

  async function run(args) {
    calls.push(args);
    if (args[0] === "whoami") {
      if (options.authFailure) {
        return { status: 1, stdout: "", stderr: options.authFailure };
      }
      return { status: 0, stdout: "release-bot\n", stderr: "" };
    }
    if (args[0] === "view") {
      const packageValue = packagesBySpec.get(args[1]);
      const metadata = registry.get(packageValue.manifest.name);
      if (!metadata) return { status: 1, stdout: "", stderr: "npm error code E404" };
      const body = options.viewAsArray ? JSON.stringify([metadata]) : JSON.stringify(metadata);
      return { status: 0, stdout: body, stderr: "" };
    }
    if (args[0] === "publish") {
      const packageValue = packagesByPath.get(args[1]);
      if (options.failPublish === packageValue.manifest.name) {
        return { status: 1, stdout: "", stderr: options.publishFailure || "publish failed" };
      }
      if (options.hideAfterPublish !== packageValue.manifest.name) {
        registry.set(packageValue.manifest.name, remoteMetadata(packageValue));
      }
      return { status: 0, stdout: "+ published", stderr: "" };
    }
    if (args[0] === "stage" && args[1] === "list") {
      if (options.authFailure) {
        return { status: 1, stdout: "", stderr: options.authFailure };
      }
      const item = staged.get(args[2]);
      return { status: 0, stdout: JSON.stringify(item ? [item] : []), stderr: "" };
    }
    if (args[0] === "stage" && args[1] === "publish") {
      const packageValue = packagesByPath.get(args[2]);
      if (options.failStage === packageValue.manifest.name) {
        return { status: 1, stdout: "", stderr: options.stageFailure || "stage failed" };
      }
      const item = {
        id: stageIds.get(packageValue.manifest.name),
        packageName: packageValue.manifest.name,
        version: packageValue.manifest.version,
        tag: "latest",
        shasum: packageValue.shasum,
      };
      staged.set(packageValue.manifest.name, item);
      return { status: 0, stdout: JSON.stringify({ [packageValue.manifest.name]: item }), stderr: "" };
    }
    if (args[0] === "install") {
      if (options.installFailure) {
        return { status: 1, stdout: "", stderr: options.installFailure };
      }
      return { status: 0, stdout: "added packages", stderr: "" };
    }
    if (args[0] === "exec") {
      if (options.commandFailure) {
        return { status: 1, stdout: "", stderr: options.commandFailure };
      }
      return {
        status: 0,
        stdout: JSON.stringify({ version: options.reportedVersion || `v${plan.version}` }),
        stderr: "",
      };
    }
    throw new Error(`unexpected npm command: ${args.join(" ")}`);
  }
  return { calls, registry, staged, run };
}

test("normalizes release versions and requires explicit publication inputs", () => {
  assert.equal(normalizeVersion("v1.2.3"), "1.2.3");
  assert.equal(normalizeVersion("1.2.3"), "1.2.3");
  assert.throws(() => normalizeVersion("1.2"), /expected vX.Y.Z or X.Y.Z/);
  assert.deepEqual(parseArguments(["v1.2.3", "dist/v1.2.3/npm", "--dry-run"]), {
    dryRun: true,
    mode: PUBLISH_MODE_STAGE,
    registry: "https://registry.npmjs.org",
    version: "1.2.3",
    npmDir: path.resolve("dist/v1.2.3/npm"),
  });
  assert.equal(
    parseArguments(["v1.2.3", "dist/v1.2.3/npm", "--publish"]).mode,
    PUBLISH_MODE_DIRECT,
  );
  assert.throws(
    () => parseArguments(["v1.2.3", "dist/v1.2.3/npm", "--stage", "--publish"]),
    /mutually exclusive/,
  );
  assert.equal(
    parseArguments(["v1.2.3", "dist/v1.2.3/npm"], { npm_config_dry_run: "true" }).dryRun,
    true,
  );
});

test("discovers the complete supported package set and orders the main package last", async (t) => {
  const fixture = await createArtifactSet(t);
  const plan = await inspectArtifacts(fixture);

  assert.equal(plan.packages.length, PLATFORM_FIXTURES.length + 1);
  assert.equal(plan.platformPackages.length, PLATFORM_FIXTURES.length);
  assert.deepEqual(
    plan.platformPackages.map(({ manifest }) => manifest.name),
    PLATFORM_FIXTURES.map(({ name }) => name).sort(),
  );
  assert.equal(plan.packages.at(-1).manifest.name, MAIN_PACKAGE_NAME);
});

test("rejects a self-consistent but incomplete supported platform set", async (t) => {
  const fixture = await createArtifactSet(t, {
    platformFixtures: PLATFORM_FIXTURES.slice(0, -1),
  });
  await assert.rejects(
    inspectArtifacts(fixture),
    /platform dependencies must exactly match the supported set: missing/,
  );
});

test("rejects a non-Windows platform binary without an executable bit", async (t) => {
  const nonWindows = PLATFORM_FIXTURES.find((platform) => platform.os !== "win32");
  const fixture = await createArtifactSet(t, {
    nonExecutableBinaryFor: nonWindows.name,
  });
  await assert.rejects(
    inspectArtifacts(fixture),
    new RegExp(`${nonWindows.name} entry package/bin/ag must be executable`),
  );
});

test("rejects version mismatches and invalid platform contents", async (t) => {
  const versionFixture = await createArtifactSet(t);
  await assert.rejects(
    inspectArtifacts({ npmDir: versionFixture.npmDir, version: "1.2.4" }),
    /does not match 1.2.4/,
  );

  const contentFixture = await createArtifactSet(t, {
    omitBinaryFor: PLATFORM_FIXTURES[0].name,
  });
  await assert.rejects(inspectArtifacts(contentFixture), /tarball contents are invalid: missing/);
});

test("rejects missing, extra, and checksum-mismatched tarballs", async (t) => {
  const missingFixture = await createArtifactSet(t);
  await rm(path.join(missingFixture.npmDir, missingFixture.fileNames[0]));
  await assert.rejects(inspectArtifacts(missingFixture), /checksums without tarballs/);

  const extraFixture = await createArtifactSet(t);
  await copyFile(
    path.join(extraFixture.npmDir, extraFixture.fileNames[0]),
    path.join(extraFixture.npmDir, "extra.tgz"),
  );
  await assert.rejects(inspectArtifacts(extraFixture), /tarballs missing from checksums.txt/);

  const checksumFixture = await createArtifactSet(t);
  const checksumPath = path.join(checksumFixture.npmDir, "checksums.txt");
  const checksums = await readFile(checksumPath, "utf8");
  const replacement = checksums[0] === "0" ? "1" : "0";
  await writeFile(checksumPath, `${replacement}${checksums.slice(1)}`);
  await assert.rejects(inspectArtifacts(checksumFixture), /checksum mismatch/);
});

test("dry run validates the plan without touching the registry", async () => {
  const plan = publicationPlan();
  const calls = [];
  const logs = [];
  const result = await publishArtifacts(plan, {
    dryRun: true,
    logger: (message) => logs.push(message),
    runNpm: async (args) => {
      calls.push(args);
      throw new Error("registry must not be called");
    },
  });

  assert.deepEqual(result, { published: [], staged: [], skipped: [], pending: [] });
  assert.deepEqual(calls, []);
  assert.match(logs[1], /Would stage platform package/);
  assert.match(logs.at(-1), /no registry requests/);
});

test("stages only platform packages until every platform version is public", async () => {
  const plan = publicationPlan();
  const publishedPlatform = plan.platformPackages[0].manifest.name;
  const stagedPlatform = plan.platformPackages[1].manifest.name;
  const registry = registryRunner(plan, {
    existing: [publishedPlatform],
    staged: [stagedPlatform],
  });
  const logs = [];

  const result = await publishArtifacts(plan, {
    logger: (message) => logs.push(message),
    runNpm: registry.run,
    verifyAttempts: 1,
  });

  assert.deepEqual(result.skipped, [publishedPlatform]);
  assert.deepEqual(result.staged, []);
  assert.deepEqual(
    result.pending.map(({ packageName }) => packageName),
    [stagedPlatform],
  );
  const stagePaths = registry.calls
    .filter(([command, subcommand]) => command === "stage" && subcommand === "publish")
    .map(([, , filePath]) => filePath);
  assert.deepEqual(stagePaths, []);
  assert.equal(registry.calls.some(([command]) => command === "publish"), false);
  assert.equal(registry.calls.some(([command]) => command === "install"), false);
  assert.match(logs.join("\n"), /not public until a maintainer approves them with 2FA/);
  assert.match(logs.join("\n"), /main package will not be staged until every platform version is public/);
});

test("stages the main package only after every platform version is public", async () => {
  const plan = publicationPlan();
  const registry = registryRunner(plan, {
    existing: plan.platformPackages.map(({ manifest }) => manifest.name),
  });

  const result = await publishArtifacts(plan, {
    logger: () => {},
    runNpm: registry.run,
    verifyAttempts: 1,
  });

  assert.deepEqual(result.staged, [MAIN_PACKAGE_NAME]);
  assert.deepEqual(result.pending.map(({ packageName }) => packageName), [MAIN_PACKAGE_NAME]);
  const stagePaths = registry.calls
    .filter(([command, subcommand]) => command === "stage" && subcommand === "publish")
    .map(([, , filePath]) => filePath);
  assert.deepEqual(stagePaths, [plan.mainPackage.filePath]);
});

test("rejects a premature main-package stage while a platform version is not public", async () => {
  const plan = publicationPlan();
  const registry = registryRunner(plan, {
    existing: [plan.platformPackages[0].manifest.name],
    staged: [plan.platformPackages[1].manifest.name, MAIN_PACKAGE_NAME],
  });

  await assert.rejects(
    publishArtifacts(plan, {
      logger: () => {},
      runNpm: registry.run,
      verifyAttempts: 1,
    }),
    /main-package stage before continuing/,
  );
  assert.equal(
    registry.calls.some(
      ([command, subcommand]) => command === "stage" && subcommand === "publish",
    ),
    false,
  );
});

test("rejects an existing staged package with a conflicting shasum", async () => {
  const plan = publicationPlan();
  const conflict = plan.platformPackages[0].manifest.name;
  const registry = registryRunner(plan, { stagedConflict: conflict });

  await assert.rejects(
    publishArtifacts(plan, { logger: () => {}, runNpm: registry.run }),
    /staged .* has conflicting shasum/,
  );
  assert.equal(
    registry.calls.some(
      ([command, subcommand]) => command === "stage" && subcommand === "publish",
    ),
    false,
  );
});

test("staged publishing reports its npm and Node.js version requirements", async () => {
  const plan = publicationPlan();
  const registry = registryRunner(plan, {
    failStage: plan.platformPackages[0].manifest.name,
    stageFailure: 'Unknown command: "stage"',
  });

  await assert.rejects(
    publishArtifacts(plan, {
      logger: () => {},
      runNpm: registry.run,
      verifyAttempts: 1,
    }),
    /requires npm 11\.15\.0 or later and Node\.js 22\.14\.0 or later/,
  );
});

test("rerunning staged mode after approval performs the final smoke test", async () => {
  const plan = publicationPlan();
  const registry = registryRunner(plan, {
    existing: plan.packages.map(({ manifest }) => manifest.name),
  });

  const result = await publishArtifacts(plan, {
    logger: () => {},
    runNpm: registry.run,
    verifyAttempts: 1,
  });

  assert.deepEqual(result.pending, []);
  assert.equal(registry.calls.some(([command]) => command === "stage"), false);
  assert.equal(registry.calls.some(([command]) => command === "install"), true);
  const execCall = registry.calls.find(([command]) => command === "exec");
  assert.deepEqual(execCall.slice(0, 3), ["exec", "--yes=false", "--prefix"]);
  assert.deepEqual(execCall.slice(-3), ["ag", "version", "--json"]);
});

test("fails when the installed npm command entry cannot execute", async () => {
  const plan = publicationPlan();
  const registry = registryRunner(plan, {
    existing: plan.packages.map(({ manifest }) => manifest.name),
    commandFailure: "installed ag command is not executable",
  });

  await assert.rejects(
    publishArtifacts(plan, {
      logger: () => {},
      runNpm: registry.run,
      verifyAttempts: 1,
    }),
    /failed to execute ag version/,
  );
  assert.equal(registry.calls.some(([command]) => command === "exec"), true);
});

test("resumes partial publication and always publishes the main package last", async () => {
  const plan = publicationPlan();
  const existing = plan.platformPackages[0].manifest.name;
  const registry = registryRunner(plan, { existing: [existing] });
  const result = await publishArtifacts(plan, {
    mode: PUBLISH_MODE_DIRECT,
    logger: () => {},
    runNpm: registry.run,
    verifyAttempts: 1,
  });

  assert.deepEqual(result.skipped, [existing]);
  assert.deepEqual(result.published, [
    plan.platformPackages[1].manifest.name,
    MAIN_PACKAGE_NAME,
  ]);
  const publishedPaths = registry.calls
    .filter(([command]) => command === "publish")
    .map(([, filePath]) => filePath);
  assert.deepEqual(publishedPaths, [plan.platformPackages[1].filePath, plan.mainPackage.filePath]);
  assert.equal(registry.calls.some(([command]) => command === "whoami"), false);
  assert.equal(registry.calls.some(([command]) => command === "install"), true);
  assert.deepEqual(
    registry.calls.find(([command]) => command === "exec").slice(-3),
    ["ag", "version", "--json"],
  );
});

test("normalizeRegistryMetadata unwraps npm 12 single-element array responses", () => {
  const spec = "pkg@1.0.0";
  const manifest = { name: "pkg", version: "1.0.0", dist: { integrity: "sha512-abc" } };
  // modern npm: a plain object passes through unchanged
  assert.equal(normalizeRegistryMetadata(manifest, spec), manifest);
  // npm 12: an exact-version query wraps the manifest in a single-element array
  assert.equal(normalizeRegistryMetadata([manifest], spec), manifest);
  // ambiguous / unsupported shapes are rejected instead of being misread as unknown@unknown
  assert.throws(() => normalizeRegistryMetadata([], spec), /ambiguous array/);
  assert.throws(() => normalizeRegistryMetadata([manifest, manifest], spec), /ambiguous array/);
  assert.throws(() => normalizeRegistryMetadata("pkg", spec), /unexpected metadata type/);
  assert.throws(() => normalizeRegistryMetadata(null, spec), /unexpected metadata type/);
});

test("normalizes staged package records and rejects ambiguous or conflicting entries", () => {
  const packageValue = packageInfo(`${MAIN_PACKAGE_NAME}-linux-x64`);
  const staged = {
    id: "00000000-0000-4000-8000-000000000001",
    packageName: packageValue.manifest.name,
    version: packageValue.manifest.version,
    shasum: packageValue.shasum,
  };
  assert.deepEqual(normalizeStagedPackages([staged], packageValue), staged);
  assert.equal(normalizeStagedPackages([], packageValue), null);
  assert.throws(
    () => normalizeStagedPackages([{ ...staged, shasum: "wrong" }], packageValue),
    /conflicting shasum/,
  );
  assert.throws(
    () => normalizeStagedPackages([staged, staged], packageValue),
    /multiple staged entries/,
  );
});

test("resumes partial publication under npm 12 array registry responses", async () => {
  const plan = publicationPlan();
  const existing = plan.platformPackages[0].manifest.name;
  const registry = registryRunner(plan, { existing: [existing], viewAsArray: true });
  const result = await publishArtifacts(plan, {
    mode: PUBLISH_MODE_DIRECT,
    logger: () => {},
    runNpm: registry.run,
    verifyAttempts: 1,
  });

  // array responses are transparent: the outcome matches the object-response case
  assert.deepEqual(result.skipped, [existing]);
  assert.deepEqual(result.published, [
    plan.platformPackages[1].manifest.name,
    MAIN_PACKAGE_NAME,
  ]);
});

test("rejects a published package whose installed CLI reports the wrong version", async () => {
  const plan = publicationPlan();
  const registry = registryRunner(plan, {
    existing: plan.packages.map(({ manifest }) => manifest.name),
    reportedVersion: "v9.9.9",
  });

  await assert.rejects(
    publishArtifacts(plan, {
      mode: PUBLISH_MODE_DIRECT,
      logger: () => {},
      runNpm: registry.run,
      verifyAttempts: 1,
    }),
    /reported version "v9\.9\.9"; expected v1\.2\.3/,
  );
});

test("preflight conflicts stop before any package is published", async () => {
  const plan = publicationPlan();
  const registry = registryRunner(plan, { conflict: plan.platformPackages[1].manifest.name });

  await assert.rejects(
    publishArtifacts(plan, {
      mode: PUBLISH_MODE_DIRECT,
      logger: () => {},
      runNpm: registry.run,
    }),
    /conflicting integrity/,
  );
  assert.equal(registry.calls.some(([command]) => command === "publish"), false);
});

test("a platform publication failure prevents the main package from publishing", async () => {
  const plan = publicationPlan();
  const registry = registryRunner(plan, { failPublish: plan.platformPackages[1].manifest.name });

  await assert.rejects(
    publishArtifacts(plan, {
      mode: PUBLISH_MODE_DIRECT,
      logger: () => {},
      runNpm: registry.run,
      verifyAttempts: 1,
    }),
    /failed to publish/,
  );
  const publishedPaths = registry.calls
    .filter(([command]) => command === "publish")
    .map(([, filePath]) => filePath);
  assert.equal(publishedPaths.includes(plan.mainPackage.filePath), false);
});

test("2FA publication failures direct maintainers to staged or trusted publishing", async () => {
  const plan = publicationPlan();
  const failures = [
    "npm error code EOTP\nnpm error This operation requires a one-time password",
    "npm error code E403\nnpm error Two-factor authentication or a granular access token with bypass 2fa enabled is required",
  ];

  for (const publishFailure of failures) {
    const registry = registryRunner(plan, {
      failPublish: plan.platformPackages[0].manifest.name,
      publishFailure,
    });

    await assert.rejects(
      publishArtifacts(plan, {
        mode: PUBLISH_MODE_DIRECT,
        logger: () => {},
        runNpm: registry.run,
        verifyAttempts: 1,
      }),
      (error) => {
        assert.match(error.message, /cannot answer npm's two-factor authentication challenge/);
        assert.match(error.message, /staged publishing/);
        assert.match(error.message, /Trusted Publishing/);
        assert.doesNotMatch(error.message, /Bypass 2FA/);
        return true;
      },
    );
  }
});

test("an unpublished platform visibility check prevents the main package from publishing", async () => {
  const plan = publicationPlan();
  const registry = registryRunner(plan, {
    hideAfterPublish: plan.platformPackages[1].manifest.name,
  });

  await assert.rejects(
    publishArtifacts(plan, {
      mode: PUBLISH_MODE_DIRECT,
      logger: () => {},
      runNpm: registry.run,
      verifyAttempts: 1,
    }),
    /was not visible after publication/,
  );
  const publishedPaths = registry.calls
    .filter(([command]) => command === "publish")
    .map(([, filePath]) => filePath);
  assert.equal(publishedPaths.includes(plan.mainPackage.filePath), false);
});

test("authentication errors redact npm tokens", async () => {
  const plan = publicationPlan();
  const secret = "npm_secret_value";
  const registry = registryRunner(plan, { authFailure: `invalid token ${secret}` });

  await assert.rejects(
    publishArtifacts(plan, {
      env: { NPM_TOKEN: secret },
      logger: () => {},
      runNpm: registry.run,
    }),
    (error) => {
      assert.match(error.message, /\[REDACTED\]/);
      assert.doesNotMatch(error.message, new RegExp(secret));
      return true;
    },
  );
});

test("redactUrlUserInfo strips embedded credentials from registry URLs", () => {
  // user:password userinfo is removed; the host is preserved.
  assert.equal(
    redactUrlUserInfo("https://release-user:npm_secret_value@registry.example.test"),
    "https://registry.example.test",
  );
  // username-only userinfo is removed too.
  assert.equal(
    redactUrlUserInfo("https://release-user@registry.example.test"),
    "https://registry.example.test",
  );
  // port and trailing path are preserved when userinfo is present.
  assert.equal(
    redactUrlUserInfo("https://user:pass@registry.example.test:8443/v1/"),
    "https://registry.example.test:8443/v1/",
  );
  // URLs without userinfo are unchanged, including scoped-package paths with @.
  assert.equal(redactUrlUserInfo("https://registry.npmjs.org"), "https://registry.npmjs.org");
  assert.equal(
    redactUrlUserInfo("https://registry.npmjs.org/@scope/pkg"),
    "https://registry.npmjs.org/@scope/pkg",
  );
  // falsy input is safe.
  assert.equal(redactUrlUserInfo(undefined), "");
});

test("authentication errors redact credentials embedded in the registry URL", async () => {
  const plan = publicationPlan();
  const username = "release-user";
  const password = "npm_secret_value";
  const registryUrl = `https://${username}:${password}@registry.example.test`;
  const registry = registryRunner(plan, { authFailure: "authentication failed" });

  await assert.rejects(
    publishArtifacts(plan, {
      registry: registryUrl,
      logger: () => {},
      runNpm: registry.run,
    }),
    (error) => {
      assert.match(
        error.message,
        /failed to list staged versions/,
      );
      assert.doesNotMatch(error.message, new RegExp(password));
      assert.doesNotMatch(error.message, new RegExp(username));
      return true;
    },
  );

  // The raw URL (with credentials) must still be handed to npm itself for auth.
  assert.deepEqual(registry.calls.find((args) => args[0] === "stage"), [
    "stage",
    "list",
    plan.platformPackages[0].manifest.name,
    "--json",
    "--registry",
    registryUrl,
  ]);
});
