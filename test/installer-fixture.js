// Shared fixture for tests that run install.sh without network access.
//
// The installer now refuses to continue unless the downloaded archive matches
// release checksums.txt, so every Unix installer test must serve both files
// through the fake curl below.

const crypto = require("node:crypto");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const root = path.resolve(__dirname, "..");
const asset = "ag_linux_amd64.tar.gz";

function sha256(buffer) {
  return crypto.createHash("sha256").update(buffer).digest("hex");
}

// createUnixInstallFixture prepares a fake release containing ag_linux_amd64
// and a checksums.txt entry. Pass checksumOverride to simulate a tampered
// archive or a malformed checksum file.
function createUnixInstallFixture(t, { checksumOverride, checksumsContent } = {}) {
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

  fs.writeFileSync(
    path.join(commands, "uname"),
    '#!/bin/sh\nif [ "$1" = -s ]; then echo Linux; else echo x86_64; fi\n',
    { mode: 0o755 },
  );
  // Real curl receives -fsSL <url> -o <destination>. Serve checksums.txt and
  // the archive by URL and record each request for assertions.
  fs.writeFileSync(
    path.join(commands, "curl"),
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
case "$url" in
*/checksums.txt) cp "$FAKE_CHECKSUMS" "$dest" ;;
*) cp "$FAKE_ARCHIVE" "$dest" ;;
esac
`,
    { mode: 0o755 },
  );

  const urlLog = path.join(dir, "urls");
  const env = {
    ...process.env,
    PATH: commands + path.delimiter + process.env.PATH,
    AG_VERSION: "v1.2.3",
    AG_FROM_SOURCE: "0",
    AG_INSTALL_DIR: destination,
    AG_REPO_OWNER: "fixture",
    AG_REPO_NAME: "repo",
    FAKE_ARCHIVE: archive,
    FAKE_CHECKSUMS: checksumsPath,
    URL_LOG: urlLog,
  };

  return {
    archive,
    destination,
    dir,
    env,
    urlLog,
    run() {
      return spawnSync("sh", [path.join(root, "install.sh")], { encoding: "utf8", env });
    },
  };
}

module.exports = { asset, createUnixInstallFixture, sha256 };
