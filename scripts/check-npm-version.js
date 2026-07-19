const { readFileSync } = require("node:fs");
const path = require("node:path");

const PLATFORM_PACKAGES = [
  { name: "@hust-open-atom-club/atomgit-cli-darwin-arm64", os: "darwin", cpu: "arm64" },
  { name: "@hust-open-atom-club/atomgit-cli-darwin-x64", os: "darwin", cpu: "x64" },
  { name: "@hust-open-atom-club/atomgit-cli-linux-arm64", os: "linux", cpu: "arm64" },
  { name: "@hust-open-atom-club/atomgit-cli-linux-x64", os: "linux", cpu: "x64" },
  { name: "@hust-open-atom-club/atomgit-cli-win32-arm64", os: "win32", cpu: "arm64" },
  { name: "@hust-open-atom-club/atomgit-cli-win32-x64", os: "win32", cpu: "x64" },
];
const PLATFORM_PACKAGE_NAMES = PLATFORM_PACKAGES.map(({ name }) => name);

function assertPackageVersions(expectedVersion, packageJson, packageLock) {
  const versions = [
    ["package.json", packageJson?.version],
    ["package-lock.json", packageLock?.version],
    ['package-lock.json packages[""]', packageLock?.packages?.[""]?.version],
  ];
  for (const packageName of PLATFORM_PACKAGE_NAMES) {
    versions.push(
      [
        `package.json optionalDependencies[${JSON.stringify(packageName)}]`,
        packageJson?.optionalDependencies?.[packageName],
      ],
      [
        `package-lock.json packages[""] optionalDependencies[${JSON.stringify(packageName)}]`,
        packageLock?.packages?.[""]?.optionalDependencies?.[packageName],
      ],
      [
        `package-lock.json packages[${JSON.stringify(`node_modules/${packageName}`)}]`,
        packageLock?.packages?.[`node_modules/${packageName}`]?.version,
      ],
    );
  }
  const mismatches = versions.filter(([, version]) => version !== expectedVersion);

  if (mismatches.length > 0) {
    const details = mismatches
      .map(([source, version]) => `${source}=${JSON.stringify(version)}`)
      .join(", ");
    throw new Error(
      `npm version mismatch for release v${expectedVersion}: ${details}; ` +
        `run "npm run version:npm -- ${expectedVersion}" before creating the release tag`,
    );
  }

  return expectedVersion;
}

function readJson(filePath) {
  return JSON.parse(readFileSync(filePath, "utf8"));
}

function main(args = process.argv.slice(2)) {
  if (args.length !== 1 || !args[0]) {
    throw new Error("usage: node scripts/check-npm-version.js <version>");
  }

  const root = path.join(__dirname, "..");
  const version = assertPackageVersions(
    args[0],
    readJson(path.join(root, "package.json")),
    readJson(path.join(root, "package-lock.json")),
  );
  console.log(`npm package version matches release: ${version}`);
}

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(`Error: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = { PLATFORM_PACKAGES, PLATFORM_PACKAGE_NAMES, assertPackageVersions };
