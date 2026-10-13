const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const root = path.resolve(__dirname, "..");

test("Unix installer installs ag-cli without replacing an existing ag", {
  skip: process.platform === "win32",
}, (t) => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "atomgit-rename-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const source = path.join(dir, "source");
  const commands = path.join(dir, "commands");
  const destination = path.join(dir, "installed");
  for (const directory of [source, commands, destination]) fs.mkdirSync(directory);
  fs.writeFileSync(path.join(source, "ag-cli"), "#!/bin/sh\nexit 0\n", { mode: 0o755 });
  fs.writeFileSync(path.join(destination, "ag"), "unrelated executable");
  const archive = path.join(dir, "ag_linux_amd64.tar.gz");
  const pack = spawnSync("tar", ["-czf", archive, "-C", source, "ag-cli"]);
  assert.equal(pack.status, 0, String(pack.stderr));
  fs.writeFileSync(path.join(commands, "uname"),
    '#!/bin/sh\nif [ "$1" = -s ]; then echo Linux; else echo x86_64; fi\n', { mode: 0o755 });
  fs.writeFileSync(path.join(commands, "curl"),
    '#!/bin/sh\nprintf "%s\\n" "$2" > "$URL_LOG"\ncp "$FAKE_ARCHIVE" "$4"\n', { mode: 0o755 });
  const result = spawnSync("sh", [path.join(root, "install.sh")], {
    encoding: "utf8",
    env: {
      ...process.env,
      PATH: commands + path.delimiter + process.env.PATH,
      AG_VERSION: "v1.2.3", AG_FROM_SOURCE: "0", AG_INSTALL_DIR: destination,
      AG_REPO_OWNER: "fixture", AG_REPO_NAME: "repo",
      FAKE_ARCHIVE: archive, URL_LOG: path.join(dir, "url"),
    },
  });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(fs.readFileSync(path.join(destination, "ag"), "utf8"), "unrelated executable");
  assert.equal(fs.readFileSync(path.join(destination, "ag-cli"), "utf8"), "#!/bin/sh\nexit 0\n");
  assert.equal(fs.statSync(path.join(destination, "ag-cli")).mode & 0o777, 0o755);
  assert.match(fs.readFileSync(path.join(dir, "url"), "utf8"), /\/v1\.2\.3\/ag_linux_amd64\.tar\.gz/);
});
