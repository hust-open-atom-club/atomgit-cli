#!/usr/bin/env node

const path = require("node:path");
const packageJson = require("../package.json");
const { installBinary } = require("./install-lib");

const RELEASES_URL =
  "https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/download";

async function main() {
  const destination = path.join(
    __dirname,
    "..",
    "bin",
    process.platform === "win32" ? "ag.exe" : "ag",
  );
  await installBinary({
    baseUrl: process.env.AG_NPM_RELEASES_URL || RELEASES_URL,
    destination,
    platform: process.platform,
    arch: process.arch,
    version: packageJson.version,
  });
  console.log(`Installed AtomGit CLI ${packageJson.version} for ${process.platform}/${process.arch}`);
}

main().catch((error) => {
  console.error(`AtomGit CLI installation failed: ${error.message}`);
  process.exitCode = 1;
});
