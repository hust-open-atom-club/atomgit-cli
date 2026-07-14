#!/usr/bin/env node

const path = require("node:path");
const { spawnSync } = require("node:child_process");

function run(args, options = {}) {
  const platform = options.platform || process.platform;
  const directory = options.directory || __dirname;
  const spawn = options.spawn || spawnSync;
  const executable = path.join(directory, platform === "win32" ? "ag.exe" : "ag");
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

module.exports = { run };
