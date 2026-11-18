const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { asset, createUnixInstallFixture } = require("./installer-fixture");

const skipOnWindows = { skip: process.platform === "win32" };
const root = path.resolve(__dirname, "..");

test("Unix installer verifies the archive against checksums.txt", skipOnWindows, (t) => {
  const fixture = createUnixInstallFixture(t);
  const result = fixture.run();
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /Checksum verified: ag_linux_amd64\.tar\.gz/);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), true);
  const requested = fs.readFileSync(fixture.urlLog, "utf8");
  assert.match(requested, /\/v1\.2\.3\/checksums\.txt/);
  assert.match(requested, /\/v1\.2\.3\/ag_linux_amd64\.tar\.gz/);
});

test("Unix installer accepts asterisk-prefixed CRLF checksum entries", skipOnWindows, (t) => {
  const fixture = createUnixInstallFixture(t);
  const { sha256 } = require("./installer-fixture");
  fs.writeFileSync(
    path.join(fixture.dir, "checksums.txt"),
    `${sha256(fs.readFileSync(fixture.archive))} *${asset}\r\n`,
  );
  const result = fixture.run();
  assert.equal(result.status, 0, result.stderr);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), true);
});

test("Unix installer refuses a tampered archive", skipOnWindows, (t) => {
  const fixture = createUnixInstallFixture(t, {
    checksumOverride: "0".repeat(64),
  });
  const result = fixture.run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /校验和不匹配/);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), false);
});

test("Unix installer refuses a checksums.txt without the archive entry", skipOnWindows, (t) => {
  const fixture = createUnixInstallFixture(t, {
    checksumsContent: `${"1".repeat(64)}  ag_linux_arm64.tar.gz\n`,
  });
  const result = fixture.run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /没有 ag_linux_amd64\.tar\.gz 的记录/);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), false);
});

test("Unix installer refuses a malformed checksum entry", skipOnWindows, (t) => {
  const fixture = createUnixInstallFixture(t, {
    checksumsContent: `not-a-hash  ${asset}\n`,
  });
  const result = fixture.run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /SHA-256 格式无效/);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), false);
});

// Windows PowerShell cannot run on the hosted Linux runners, so this test
// checks the source order instead: verification must happen before extraction.
test("PowerShell installer verifies the archive before extracting it", () => {
  const installer = fs.readFileSync(path.join(root, "install.ps1"), "utf8");
  const checksumCall = installer.indexOf("Assert-ArchiveChecksum -ChecksumsPath");
  const expand = installer.indexOf("Expand-Archive -Path $archivePath");
  assert.notEqual(checksumCall, -1, "install.ps1 must verify the archive checksum");
  assert.notEqual(expand, -1, "install.ps1 must extract the downloaded archive");
  assert.ok(checksumCall < expand, "checksum verification must run before extraction");
  assert.match(installer, /Get-FileHash -LiteralPath \$ArchivePath -Algorithm SHA256/);
  assert.match(installer, /checksumsUrl/);
});
