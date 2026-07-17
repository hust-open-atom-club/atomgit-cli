const PLATFORM_NAMES = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const { createHash } = require("node:crypto");
const { chmod, mkdir, mkdtemp, rename, rm, writeFile } = require("node:fs/promises");
const path = require("node:path");

const DEFAULT_DOWNLOAD_TIMEOUT_MS = 30_000;

const ARCH_NAMES = {
  arm64: "arm64",
  x64: "amd64",
};

function resolveTarget(platform, arch) {
  const releasePlatform = PLATFORM_NAMES[platform];
  const releaseArch = ARCH_NAMES[arch];

  if (!releasePlatform || !releaseArch) {
    throw new Error(`unsupported platform: ${platform}/${arch}`);
  }

  const windows = platform === "win32";
  return {
    asset: `ag_${releasePlatform}_${releaseArch}.${windows ? "zip" : "tar.gz"}`,
    executable: windows ? "ag.exe" : "ag",
  };
}

function buildReleaseInfo({ baseUrl, version, asset }) {
  const releaseUrl = `${baseUrl.replace(/\/$/, "")}/v${version}`;
  return {
    assetUrl: `${releaseUrl}/${asset}`,
    checksumsUrl: `${releaseUrl}/checksums.txt`,
  };
}

function parseChecksums(contents) {
  const checksums = new Map();
  for (const line of contents.split(/\r?\n/)) {
    const match = line.trim().match(/^([a-fA-F0-9]{64})\s+\*?(.+)$/);
    if (match) {
      checksums.set(match[2], match[1].toLowerCase());
    }
  }
  return checksums;
}

function verifyChecksum(contents, expected) {
  const actual = createHash("sha256").update(contents).digest("hex");
  if (actual !== expected.toLowerCase()) {
    throw new Error(`checksum mismatch: expected ${expected}, received ${actual}`);
  }
}

async function download(url, timeoutMs = DEFAULT_DOWNLOAD_TIMEOUT_MS) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, { signal: controller.signal });
    if (!response.ok) {
      throw new Error(`failed to download ${url}: HTTP ${response.status}`);
    }
    return Buffer.from(await response.arrayBuffer());
  } catch (error) {
    if (controller.signal.aborted) {
      throw new Error(`timed out downloading ${url} after ${timeoutMs}ms`, {
        cause: error,
      });
    }
    throw error;
  } finally {
    clearTimeout(timer);
  }
}

async function extractArchive(archivePath, target, destination) {
  if (archivePath.endsWith(".zip")) {
    const AdmZip = require("adm-zip");
    const entry = new AdmZip(archivePath).getEntry(target.executable);
    if (!entry) {
      throw new Error(`archive does not contain ${target.executable}`);
    }
    await writeFile(destination, entry.getData());
    return;
  }

  const tar = require("tar");
  const extractDir = path.dirname(destination);
  await tar.x({ cwd: extractDir, file: archivePath, strict: true }, [target.executable]);
  const extracted = path.join(extractDir, target.executable);
  if (extracted !== destination) {
    await rename(extracted, destination);
  }
}

async function installBinary({
  baseUrl,
  destination,
  platform,
  arch,
  version,
  extract = extractArchive,
  downloadTimeoutMs = DEFAULT_DOWNLOAD_TIMEOUT_MS,
}) {
  const target = resolveTarget(platform, arch);
  const urls = buildReleaseInfo({ baseUrl, version, asset: target.asset });
  const checksumContents = await download(urls.checksumsUrl, downloadTimeoutMs);
  const expected = parseChecksums(checksumContents.toString("utf8")).get(target.asset);
  if (!expected) {
    throw new Error(`checksums.txt does not contain ${target.asset}`);
  }

  const archive = await download(urls.assetUrl, downloadTimeoutMs);
  verifyChecksum(archive, expected);

  const destinationDir = path.dirname(destination);
  await mkdir(destinationDir, { recursive: true });
  const stagingDir = await mkdtemp(path.join(destinationDir, ".ag-npm-install-"));
  try {
    const archivePath = path.join(stagingDir, target.asset);
    const extractedPath = path.join(stagingDir, path.basename(destination));
    await writeFile(archivePath, archive);
    await extract(archivePath, target, extractedPath);
    await chmod(extractedPath, 0o755);
    await rm(destination, { force: true });
    await rename(extractedPath, destination);
  } finally {
    await rm(stagingDir, { recursive: true, force: true });
  }
}

module.exports = {
  buildReleaseInfo,
  extractArchive,
  installBinary,
  parseChecksums,
  resolveTarget,
  verifyChecksum,
};
