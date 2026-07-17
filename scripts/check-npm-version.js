const { readFileSync } = require("node:fs");
const path = require("node:path");

function assertPackageVersions(expectedVersion, packageJson, packageLock) {
  const versions = [
    ["package.json", packageJson?.version],
    ["package-lock.json", packageLock?.version],
    ['package-lock.json packages[""]', packageLock?.packages?.[""]?.version],
  ];
  const mismatches = versions.filter(([, version]) => version !== expectedVersion);

  if (mismatches.length > 0) {
    const details = mismatches
      .map(([source, version]) => `${source}=${JSON.stringify(version)}`)
      .join(", ");
    throw new Error(
      `npm version mismatch for release v${expectedVersion}: ${details}; ` +
        `run "npm version ${expectedVersion} --no-git-tag-version" before creating the release tag`,
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

module.exports = { assertPackageVersions };
