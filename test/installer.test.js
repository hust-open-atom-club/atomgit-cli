const test = require("node:test");
const assert = require("node:assert/strict");
const { createHash } = require("node:crypto");
const { mkdtemp, readFile, rm, writeFile } = require("node:fs/promises");
const { createServer } = require("node:http");
const { tmpdir } = require("node:os");
const path = require("node:path");

const {
  buildReleaseInfo,
  extractArchive,
  installBinary,
  parseChecksums,
  resolveTarget,
  verifyChecksum,
} = require("../scripts/install-lib");
const { run } = require("../bin/ag");

test("maps supported Node platforms and architectures to release assets", () => {
  assert.deepEqual(resolveTarget("linux", "x64"), {
    asset: "ag_linux_amd64.tar.gz",
    executable: "ag",
  });
  assert.deepEqual(resolveTarget("darwin", "arm64"), {
    asset: "ag_darwin_arm64.tar.gz",
    executable: "ag",
  });
  assert.deepEqual(resolveTarget("win32", "x64"), {
    asset: "ag_windows_amd64.zip",
    executable: "ag.exe",
  });
});

test("rejects unsupported platforms and architectures", () => {
  assert.throws(
    () => resolveTarget("freebsd", "x64"),
    /unsupported platform: freebsd\/x64/,
  );
  assert.throws(
    () => resolveTarget("linux", "ia32"),
    /unsupported platform: linux\/ia32/,
  );
});

test("builds release URLs from the npm package version", () => {
  assert.deepEqual(
    buildReleaseInfo({
      baseUrl: "https://atomgit.example/org/repo/releases/download",
      version: "1.2.3-beta.1",
      asset: "ag_linux_amd64.tar.gz",
    }),
    {
      assetUrl:
        "https://atomgit.example/org/repo/releases/download/v1.2.3-beta.1/ag_linux_amd64.tar.gz",
      checksumsUrl:
        "https://atomgit.example/org/repo/releases/download/v1.2.3-beta.1/checksums.txt",
    },
  );
});

test("parses checksums and verifies an archive", () => {
  const archive = Buffer.from("release archive");
  const checksum = createHash("sha256").update(archive).digest("hex");

  assert.equal(
    parseChecksums(`${checksum}  ag_linux_amd64.tar.gz\r\n`).get(
      "ag_linux_amd64.tar.gz",
    ),
    checksum,
  );
  assert.doesNotThrow(() => verifyChecksum(archive, checksum));
  assert.throws(
    () => verifyChecksum(Buffer.from("tampered"), checksum),
    /checksum mismatch/,
  );
});

test("downloads, verifies, and extracts a release without external network access", async (t) => {
  const archive = Buffer.from("fake archive");
  const checksum = createHash("sha256").update(archive).digest("hex");
  const server = createServer((request, response) => {
    if (request.url === "/v0.5.0/checksums.txt") {
      response.end(`${checksum}  ag_linux_amd64.tar.gz\n`);
      return;
    }
    if (request.url === "/v0.5.0/ag_linux_amd64.tar.gz") {
      response.end(archive);
      return;
    }
    response.writeHead(404).end();
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  t.after(() => server.close());

  const root = await mkdtemp(path.join(process.cwd(), ".ag-npm-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const destination = path.join(root, "ag");
  const { port } = server.address();

  await installBinary({
    baseUrl: `http://127.0.0.1:${port}`,
    destination,
    platform: "linux",
    arch: "x64",
    version: "0.5.0",
    extract: async (archivePath, _target, extractedPath) => {
      assert.deepEqual(await readFile(archivePath), archive);
      await writeFile(extractedPath, "installed binary");
    },
  });

  assert.equal(await readFile(destination, "utf8"), "installed binary");
});

test("reports failed release downloads", async () => {
  const server = createServer((_request, response) => {
    response.writeHead(404).end("missing");
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));

  try {
    const { port } = server.address();
    await assert.rejects(
      installBinary({
        baseUrl: `http://127.0.0.1:${port}`,
        destination: path.join(tmpdir(), "unused-ag"),
        platform: "linux",
        arch: "x64",
        version: "0.5.0",
      }),
      /failed to download .*checksums\.txt: HTTP 404/,
    );
  } finally {
    server.close();
  }
});

test("rejects a release asset when its checksum does not match", async () => {
  const server = createServer((request, response) => {
    if (request.url.endsWith("checksums.txt")) {
      response.end(`${"0".repeat(64)}  ag_linux_amd64.tar.gz\n`);
      return;
    }
    response.end("tampered archive");
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));

  try {
    const { port } = server.address();
    await assert.rejects(
      installBinary({
        baseUrl: `http://127.0.0.1:${port}`,
        destination: path.join(tmpdir(), "unused-ag"),
        platform: "linux",
        arch: "x64",
        version: "0.5.0",
      }),
      /checksum mismatch/,
    );
  } finally {
    server.close();
  }
});

test("forwards arguments and the child exit code to the installed binary", () => {
  const calls = [];
  const result = run(["version", "--json"], {
    platform: "linux",
    directory: "/package/bin",
    spawn(command, args, options) {
      calls.push({ command, args, options });
      return { status: 7, signal: null };
    },
  });

  assert.equal(result, 7);
  assert.deepEqual(calls, [
    {
      command: path.join("/package/bin", "ag"),
      args: ["version", "--json"],
      options: { stdio: "inherit" },
    },
  ]);
});

test("extracts the ag executable from tar.gz and zip release archives", async (t) => {
  const root = await mkdtemp(path.join(process.cwd(), ".ag-npm-test-"));
  t.after(() => rm(root, { recursive: true, force: true }));
  const source = path.join(root, "source");
  await require("node:fs/promises").mkdir(source);
  await writeFile(path.join(source, "ag"), "unix binary");
  await writeFile(path.join(source, "ag.exe"), "windows binary");

  const tarPath = path.join(root, "ag_linux_amd64.tar.gz");
  await require("tar").c({ cwd: source, file: tarPath, gzip: true }, ["ag"]);
  const unixDestination = path.join(root, "unix", "ag");
  await require("node:fs/promises").mkdir(path.dirname(unixDestination));
  await extractArchive(
    tarPath,
    { executable: "ag" },
    unixDestination,
  );
  assert.equal(await readFile(unixDestination, "utf8"), "unix binary");

  const zipPath = path.join(root, "ag_windows_amd64.zip");
  const zip = new (require("adm-zip"))();
  zip.addFile("ag.exe", Buffer.from("windows binary"));
  zip.writeZip(zipPath);
  const windowsDestination = path.join(root, "windows", "ag.exe");
  await require("node:fs/promises").mkdir(path.dirname(windowsDestination));
  await extractArchive(
    zipPath,
    { executable: "ag.exe" },
    windowsDestination,
  );
  assert.equal(await readFile(windowsDestination, "utf8"), "windows binary");
});
