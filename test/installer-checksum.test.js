const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { asset, createUnixInstallFixture, sha256 } = require("./installer-fixture");

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
  assert.match(result.stderr, /Checksum mismatch/);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), false);
});

test("Unix installer refuses a checksums.txt without the archive entry", skipOnWindows, (t) => {
  const fixture = createUnixInstallFixture(t, {
    checksumsContent: `${"1".repeat(64)}  ag_linux_arm64.tar.gz\n`,
  });
  const result = fixture.run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /does not contain an entry for ag_linux_amd64\.tar\.gz/);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), false);
});

test("Unix installer refuses a malformed checksum entry", skipOnWindows, (t) => {
  const fixture = createUnixInstallFixture(t, {
    checksumsContent: `not-a-hash  ${asset}\n`,
  });
  const result = fixture.run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Invalid checksum entry for ag_linux_amd64\.tar\.gz/);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), false);
});

test("Unix installer refuses to install when checksums.txt cannot be downloaded", skipOnWindows, (t) => {
  const fixture = createUnixInstallFixture(t, { failMatch: "checksums.txt" });
  const result = fixture.run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Failed to download checksums\.txt/);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), false);
});

test("Unix installer refuses to install without a SHA-256 tool", skipOnWindows, (t) => {
  const fixture = createUnixInstallFixture(t, { hideHashTools: true });
  const result = fixture.run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Failed to compute SHA-256 for ag_linux_amd64\.tar\.gz/);
  assert.equal(fs.existsSync(path.join(fixture.destination, "ag-cli")), false);
});

test("Unix installer reports a hash tool failure as a tool error", skipOnWindows, (t) => {
  // A sha256sum that exists but fails must not surface as a checksum mismatch.
  const fixture = createUnixInstallFixture(t);
  fs.writeFileSync(path.join(fixture.dir, "commands", "sha256sum"),
    "#!/bin/sh\nexit 1\n", { mode: 0o755 });
  const result = fixture.run();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Failed to compute SHA-256 for ag_linux_amd64\.tar\.gz/);
  assert.doesNotMatch(result.stderr, /Checksum mismatch/);
});

// Windows PowerShell cannot run on every hosted Linux runner, so the static
// assertion always runs and the behavioural tests run where pwsh exists.
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

const powershell = hasPowerShell();

test("PowerShell checksum helpers accept star- and dot-prefixed entries", {
  skip: powershell ? false : "pwsh is not installed",
}, (t) => {
  const fixture = createPowerShellFixture(t);
  const hash = sha256(fs.readFileSync(fixture.archive));
  for (const line of [`${hash} *${fixture.asset}`, `${hash}  ./${fixture.asset}`]) {
    fixture.writeChecksums(`${"9".repeat(64)} *other.zip\n${line}\r\n`);
    const result = fixture.verify();
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, /verified/);
  }
});

test("PowerShell checksum helpers reject a tampered archive", {
  skip: powershell ? false : "pwsh is not installed",
}, (t) => {
  const fixture = createPowerShellFixture(t);
  fixture.writeChecksums(`${"0".repeat(64)}  ${fixture.asset}\n`);
  const result = fixture.verify();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Checksum mismatch/);
});

test("PowerShell checksum helpers reject a missing entry", {
  skip: powershell ? false : "pwsh is not installed",
}, (t) => {
  const fixture = createPowerShellFixture(t);
  fixture.writeChecksums(`${"0".repeat(64)}  other.zip\n`);
  const result = fixture.verify();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /does not contain an entry/);
});

test("PowerShell checksum helpers reject a malformed entry", {
  skip: powershell ? false : "pwsh is not installed",
}, (t) => {
  const fixture = createPowerShellFixture(t);
  fixture.writeChecksums(`not-a-hash *${fixture.asset}\n`);
  const result = fixture.verify();
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Invalid checksum entry/);
});

function hasPowerShell() {
  return spawnSync("pwsh", ["-NoProfile", "-Command", "exit 0"]).status === 0;
}

// createPowerShellFixture extracts only the two checksum helpers from
// install.ps1 through the PowerShell AST, so the full installer never runs and
// no network or architecture detection is involved.
function createPowerShellFixture(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "atomgit-install-ps-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));

  const assetName = "ag_windows_amd64.zip";
  const archive = path.join(dir, assetName);
  fs.writeFileSync(archive, "fake zip bytes");
  const checksums = path.join(dir, "checksums.txt");
  const driver = path.join(dir, "driver.ps1");
  fs.writeFileSync(driver, POWERSHELL_DRIVER);

  const env = {
    ...process.env,
    AG_INSTALLER: path.join(root, "install.ps1"),
    AG_CHECKSUMS: checksums,
    AG_ARCHIVE: archive,
    AG_ASSET: assetName,
  };

  return {
    archive,
    asset: assetName,
    writeChecksums(contents) {
      fs.writeFileSync(checksums, contents);
    },
    verify() {
      return spawnSync("pwsh", ["-NoProfile", "-File", driver], { encoding: "utf8", env });
    },
  };
}

// POWERSHELL_DRIVER keeps ASCII-only bytes and references paths through the
// environment so no quoting of caller-supplied values is needed.
const POWERSHELL_DRIVER = `
$ErrorActionPreference = "Stop"

function Die([string]$Msg) {
    $host.UI.WriteErrorLine($Msg)
    exit 1
}

$tokens = $null
$parseErrors = $null
$ast = [System.Management.Automation.Language.Parser]::ParseFile(
    $env:AG_INSTALLER, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -gt 0) {
    Die "install.ps1 has parse errors"
}

$wanted = @("Get-ExpectedHash", "Assert-ArchiveChecksum")
$found = @{}
foreach ($fn in $ast.FindAll({ param($node) $node -is [System.Management.Automation.Language.FunctionDefinitionAst] }, $true)) {
    if ($wanted -contains $fn.Name) { $found[$fn.Name] = $fn.Extent.Text }
}
foreach ($name in $wanted) {
    if (-not $found.ContainsKey($name)) { Die "install.ps1 is missing $name" }
    Invoke-Expression $found[$name]
}

Assert-ArchiveChecksum -ChecksumsPath $env:AG_CHECKSUMS -ArchivePath $env:AG_ARCHIVE -AssetName $env:AG_ASSET
Write-Output "verified"
`;
