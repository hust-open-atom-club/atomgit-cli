const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { createUnixInstallFixture } = require("./installer-fixture");

test("Unix installer installs ag-cli without replacing an existing ag", {
  skip: process.platform === "win32",
}, (t) => {
  const fixture = createUnixInstallFixture(t);
  const result = fixture.run();
  assert.equal(result.status, 0, result.stderr);
  assert.equal(fs.readFileSync(path.join(fixture.destination, "ag"), "utf8"), "unrelated executable");
  assert.equal(fs.readFileSync(path.join(fixture.destination, "ag-cli"), "utf8"), "#!/bin/sh\nexit 0\n");
  assert.equal(fs.statSync(path.join(fixture.destination, "ag-cli")).mode & 0o777, 0o755);
  assert.match(fs.readFileSync(fixture.urlLog, "utf8"), /\/v1\.2\.3\/ag_linux_amd64\.tar\.gz/);
});
