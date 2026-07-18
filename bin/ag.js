#!/usr/bin/env node

const { spawnSync } = require("node:child_process");

const PLATFORM_PACKAGES = {
  "darwin-arm64": "@hust-open-atom-club/atomgit-cli-darwin-arm64",
  "darwin-x64": "@hust-open-atom-club/atomgit-cli-darwin-x64",
  "linux-arm64": "@hust-open-atom-club/atomgit-cli-linux-arm64",
  "linux-x64": "@hust-open-atom-club/atomgit-cli-linux-x64",
  "win32-arm64": "@hust-open-atom-club/atomgit-cli-win32-arm64",
  "win32-x64": "@hust-open-atom-club/atomgit-cli-win32-x64",
};

function resolveBinary(platform, arch, resolve = require.resolve) {
  const packageName = PLATFORM_PACKAGES[`${platform}-${arch}`];
  if (!packageName) {
    throw new Error(`unsupported platform: ${platform}/${arch}`);
  }

  const executable = platform === "win32" ? "ag.exe" : "ag";
  try {
    return resolve(`${packageName}/bin/${executable}`);
  } catch (error) {
    throw new Error(
      `platform package ${packageName} is missing; reinstall without omitting optional dependencies`,
      { cause: error },
    );
  }
}

function run(args, options = {}) {
  const platform = options.platform || process.platform;
  const arch = options.arch || process.arch;
  const resolve = options.resolve || require.resolve;
  const spawn = options.spawn || spawnSync;
  const executable = resolveBinary(platform, arch, resolve);
  const result = spawn(executable, args, { stdio: "inherit" });

  if (result.error) {
    throw result.error;
  }
  if (result.signal) {
    throw new Error(`ag terminated by signal ${result.signal}`);
  }
  return result.status === null ? 1 : result.status;
}

if (require.main === module) {
  try {
    process.exitCode = run(process.argv.slice(2));
  } catch (error) {
    console.error(`ag: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = { PLATFORM_PACKAGES, resolveBinary, run };
