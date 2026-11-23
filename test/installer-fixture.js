// Shared fixture for tests that run install.sh without network access.
//
// The installer refuses to continue unless the downloaded archive matches
// release checksums.txt, so every Unix installer test must serve both files
// through the fake curl below.

const crypto = require("node:crypto");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const root = path.resolve(__dirname, "..");
const asset = "ag_linux_amd64.tar.gz";

// Tools the installer needs from PATH. Hash tools are intentionally absent so
// hideHashTools can prove the "no verification tool" branch fails closed.
const baseTools = ["awk", "cp", "install", "mkdir", "mktemp", "rm", "tar", "gzip"];

function sha256(buffer) {
  return crypto.createHash("sha256").update(buffer).digest("hex");
}

// createUnixInstallFixture prepares a fake release containing ag_linux_amd64
// and a checksums.txt entry. Pass checksumOverride to simulate a tampered
// archive, checksumsContent to replace the file, failMatch to make the fake
// curl fail for a URL substring, or hideHashTools to remove every SHA-256 tool
// from PATH.
function createUnixInstallFixture(t, {
  checksumOverride,
  checksumsContent,
  failMatch,
  hideHashTools,
} = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "atomgit-install-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));

  const source = path.join(dir, "source");
  const commands = path.join(dir, "commands");
  const destination = path.join(dir, "installed");
  for (const directory of [source, commands, destination]) fs.mkdirSync(directory);

  fs.writeFileSync(path.join(source, "ag-cli"), "#!/bin/sh\nexit 0\n", { mode: 0o755 });
  fs.writeFileSync(path.join(destination, "ag"), "unrelated executable");
  const archive = path.join(dir, asset);
  const pack = spawnSync("tar", ["-czf", archive, "-C", source, "ag-cli"]);
  if (pack.status !== 0) throw new Error(String(pack.stderr));

  const archiveHash = checksumOverride ?? sha256(fs.readFileSync(archive));
  const checksumsPath = path.join(dir, "checksums.txt");
  fs.writeFileSync(
    checksumsPath,
    checksumsContent ?? `${archiveHash}  ${asset}\n`,
  );

  writeFakeCommands(commands);
  const binPath = hideHashTools
    ? buildRestrictedPath(dir, commands)
    : commands + path.delimiter + process.env.PATH;

  const urlLog = path.join(dir, "urls");
  const env = {
    ...process.env,
    PATH: binPath,
    AG_VERSION: "v1.2.3",
    AG_FROM_SOURCE: "0",
    AG_INSTALL_DIR: destination,
    AG_REPO_OWNER: "fixture",
    AG_REPO_NAME: "repo",
    FAKE_ARCHIVE: archive,
    FAKE_CHECKSUMS: checksumsPath,
    URL_LOG: urlLog,
  };
  if (failMatch) env.FAIL_MATCH = failMatch;

  return {
    archive,
    destination,
    dir,
    env,
    urlLog,
    run() {
      // Resolve sh explicitly so a restricted PATH cannot hide the interpreter.
      return spawnSync("/bin/sh", [path.join(root, "install.sh")], { encoding: "utf8", env });
    },
  };
}

function writeFakeCommands(dir) {
  fs.writeFileSync(
    path.join(dir, "uname"),
    '#!/bin/sh\nif [ "$1" = -s ]; then echo Linux; else echo x86_64; fi\n',
    { mode: 0o755 },
  );
  // Real curl receives -fsSL <url> -o <destination>. Serve checksums.txt and
  // the archive by URL, record each request, and fail on FAIL_MATCH.
  fs.writeFileSync(
    path.join(dir, "curl"),
    `#!/bin/sh
url=""
dest=""
while [ $# -gt 0 ]; do
  case "$1" in
  -o) dest=$2; shift 2 ;;
  -*) shift ;;
  *) url=$1; shift ;;
  esac
done
printf "%s\\n" "$url" >> "$URL_LOG"
if [ -n "\${FAIL_MATCH:-}" ]; then
  case "$url" in
  *"$FAIL_MATCH"*) exit 7 ;;
  esac
fi
case "$url" in
*/checksums.txt) cp "$FAKE_CHECKSUMS" "$dest" ;;
*) cp "$FAKE_ARCHIVE" "$dest" ;;
esac
`,
    { mode: 0o755 },
  );
}

// buildRestrictedPath returns a PATH containing only the fake commands and
// symlinks to the base tools, so no SHA-256 implementation is reachable.
function buildRestrictedPath(dir, commands) {
  const bin = path.join(dir, "bin");
  fs.mkdirSync(bin);
  for (const name of fs.readdirSync(commands)) {
    fs.symlinkSync(path.join(commands, name), path.join(bin, name));
  }
  for (const name of baseTools) {
    const resolved = spawnSync("sh", ["-c", `command -v ${name}`], { encoding: "utf8" });
    const target = resolved.stdout.trim();
    if (resolved.status !== 0 || !target) continue;
    fs.symlinkSync(target, path.join(bin, name));
  }
  // A hash tool must not leak in through the caller's environment.
  for (const name of ["sha256sum", "shasum", "openssl"]) {
    if (fs.existsSync(path.join(bin, name))) {
      throw new Error(`restricted PATH still contains ${name}`);
    }
  }
  return bin;
}

module.exports = { asset, createUnixInstallFixture, sha256 };
