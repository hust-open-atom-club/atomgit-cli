const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { createHash } = require('node:crypto');
const root = path.resolve(__dirname, '..');
const version = fs.readFileSync(path.join(root, '.goreleaser-version'), 'utf8').trim();
const unix = { skip: process.platform === 'win32' };
function fixture(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ag-goreleaser-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  fs.mkdirSync(path.join(dir, 'bin'));
  return dir;
}
function script(file, body) { fs.writeFileSync(file, '#!/bin/sh\n' + body, { mode: 0o755 }); }
function run(file, args, env) {
  return spawnSync('sh', [file, ...args], { encoding: 'utf8', env: { ...process.env, ...env } });
}
for (const snapshot of ['0', '1']) {
  for (const scenario of ['expected', 'wrong', 'prerelease', 'malformed', 'failed', 'missing']) {
    test(`packaging preflight snapshot=${snapshot} ${scenario}`, unix, t => {
      const dir = fixture(t);
      const tool = path.join(dir, 'tool with spaces');
      const log = path.join(dir, 'calls');
      if (scenario !== 'missing') {
        const output = { expected: `GitVersion:    ${version}`, wrong: 'GitVersion: 0.1.0', prerelease: `GitVersion: ${version}-next`, malformed: `something ${version}`, failed: '' }[scenario];
        script(tool, `echo "$*" >> "$CALL_LOG"\nprintf '%s\\n' '${output}'\nexit ${scenario === 'failed' ? 1 : 0}\n`);
      }
      script(path.join(dir, 'bin', 'go'), 'echo TOOLCHAIN_REACHED >&2\nexit 91\n');
      const out = run(path.join(root, 'scripts/build-release.sh'), [], {
        GORELEASER: tool, AG_RELEASE_SNAPSHOT: snapshot, AG_VERIFY_ONLY: '',
        TAG: 'abcdef', CALL_LOG: log, PATH: path.join(dir, 'bin') + path.delimiter + process.env.PATH,
      });
      assert.notEqual(out.status, 0);
      assert.equal(out.stderr.includes('TOOLCHAIN_REACHED'), scenario === 'expected', out.stderr);
      if (scenario !== 'missing') assert.equal(fs.readFileSync(log, 'utf8'), '--version\n');
      if (scenario === 'missing') assert.match(out.stderr, /executable not found/);
      else if (scenario !== 'expected') assert.match(out.stderr, /version mismatch|Cannot query/);
    });
  }
}
test('default PATH tool uses the same version check', unix, t => {
  const dir = fixture(t);
  script(path.join(dir, 'bin', 'goreleaser'), `echo 'GitVersion: ${version}'`);
  const out = run(path.join(root, 'scripts/check-goreleaser.sh'), [], { GORELEASER: '', PATH: path.join(dir, 'bin') + path.delimiter + process.env.PATH });
  assert.equal(out.status, 0, out.stderr);
});
for (const corrupt of [false, true]) {
  test(`installer verifies before extraction/execution corrupt=${corrupt}`, unix, t => {
    const dir = fixture(t);
    const scripts = path.join(dir, 'scripts');
    fs.mkdirSync(scripts);
    for (const name of ['check-goreleaser.sh', 'install-goreleaser.sh']) fs.copyFileSync(path.join(root, 'scripts', name), path.join(scripts, name));
    fs.writeFileSync(path.join(dir, '.goreleaser-version'), version + '\n');
    const payload = path.join(dir, 'payload');
    fs.mkdirSync(payload);
    script(path.join(payload, 'goreleaser'), `echo executed >> "$EXEC_LOG"\necho 'GitVersion: ${version}'\n`);
    const archive = path.join(dir, 'archive.tar.gz');
    const packed = spawnSync('tar', ['-czf', archive, '-C', payload, 'goreleaser']);
    assert.equal(packed.status, 0);
    const hash = createHash('sha256').update(fs.readFileSync(archive)).digest('hex');
    fs.writeFileSync(path.join(scripts, 'goreleaser-checksums.txt'), `${hash}  goreleaser_Linux_x86_64.tar.gz\n`);
    script(path.join(dir, 'bin', 'uname'), '[ "$1" = -s ] && echo Linux || echo x86_64\n');
    script(path.join(dir, 'bin', 'curl'), 'while [ "$#" -gt 0 ]; do\n if [ "$1" = -o ]; then cp "$FAKE_ARCHIVE" "$2"; exit; fi\n shift\ndone\nexit 1\n');
    if (corrupt) fs.appendFileSync(archive, 'corrupt');
    const installed = path.join(dir, 'installed');
    const log = path.join(dir, 'executed');
    const out = run(path.join(scripts, 'install-goreleaser.sh'), [installed], { PATH: path.join(dir, 'bin') + path.delimiter + process.env.PATH, FAKE_ARCHIVE: archive, EXEC_LOG: log });
    assert.equal(out.status, corrupt ? 1 : 0, out.stderr);
    assert.equal(fs.existsSync(path.join(installed, 'goreleaser')), !corrupt);
    assert.equal(fs.existsSync(log), !corrupt);
    if (corrupt) assert.match(out.stderr, /Checksum mismatch/);
  });
}
