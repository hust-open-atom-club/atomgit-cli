// Verify every Unix archive contains exactly the generated command-tree pages.
const { readdir } = require("node:fs/promises");
const path = require("node:path");
const { validateArchive, REQUIRED_ARCHIVES } = require("./publish-atomgit-release");

async function checkReleaseManpages(releaseDir, manDir) {
  const entries = await readdir(manDir, { withFileTypes: true });
  const names = entries.filter((entry) => entry.isFile() && entry.name.endsWith(".1"))
    .map((entry) => entry.name).sort();
  if (!names.includes("ag-cli.1")) throw new Error("generated root manual ag-cli.1 is missing");
  for (const archive of REQUIRED_ARCHIVES) {
    await validateArchive(path.join(releaseDir, archive), names);
  }
  return names.length;
}

if (require.main === module) {
  const [releaseDir, manDir] = process.argv.slice(2);
  if (!releaseDir || !manDir || process.argv.length !== 4) {
    console.error("usage: node scripts/check-release-manpages.js <release-dir> <generated-man-dir>");
    process.exitCode = 1;
  } else {
    checkReleaseManpages(releaseDir, manDir).then((count) => {
      console.log(`Verified ${count} manpages in every Unix release archive.`);
    }).catch((error) => {
      console.error(error.message);
      process.exitCode = 1;
    });
  }
}

module.exports = { checkReleaseManpages };
