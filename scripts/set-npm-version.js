#!/usr/bin/env node

const { readFileSync, writeFileSync } = require("node:fs");
const path = require("node:path");
const { PLATFORM_PACKAGES } = require("./check-npm-version");

function readJson(filePath) {
  return JSON.parse(readFileSync(filePath, "utf8"));
}

function writeJson(filePath, value) {
  writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

function setNpmVersion(version, packageJson, packageLock) {
  if (!/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(version)) {
    throw new Error(`invalid npm version: ${version}`);
  }

  packageJson.version = version;
  packageLock.version = version;
  packageLock.packages[""].version = version;
  for (const platformPackage of PLATFORM_PACKAGES) {
    const { name, os, cpu } = platformPackage;
    packageJson.optionalDependencies[name] = version;
    packageLock.packages[""].optionalDependencies[name] = version;
    packageLock.packages[`node_modules/${name}`] = {
      version,
      cpu: [cpu],
      optional: true,
      os: [os],
    };
  }
}

function main(args = process.argv.slice(2)) {
  if (args.length !== 1) {
    throw new Error("usage: npm run version:npm -- <version>");
  }
  const root = path.join(__dirname, "..");
  const packagePath = path.join(root, "package.json");
  const lockPath = path.join(root, "package-lock.json");
  const packageJson = readJson(packagePath);
  const packageLock = readJson(lockPath);
  setNpmVersion(args[0], packageJson, packageLock);
  writeJson(packagePath, packageJson);
  writeJson(lockPath, packageLock);
  console.log(`Updated npm package versions to ${args[0]}`);
}

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(`Error: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = { setNpmVersion };
