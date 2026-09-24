const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { createHash } = require('node:crypto');
const root = path.resolve(__dirname, '..');
const unix = { skip: process.platform === 'win32' };
const installer = path.join(root, 'scripts/install-nix.sh');
const publisher = path.join(root, 'scripts/publish-nix-update.sh');
const TOKEN = 'SPEC_TOKEN_DO_NOT_LEAK_0123456789abcdef';

function fixture(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ag-nix-update-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  fs.mkdirSync(path.join(dir, 'bin'));
  return dir;
}
function script(file, body) { fs.writeFileSync(file, '#!/bin/sh\n' + body, { mode: 0o755 }); }
function run(file, args, env, cwd) {
  // An explicitly undefined value unsets the variable even when the host
  // environment provides it (CI runners export ATOMGIT_REPOSITORY and
  // ATOMGIT_REF_NAME themselves).
  const merged = { ...process.env, ...env };
  for (const key of Object.keys(env)) if (env[key] === undefined) delete merged[key];
  return spawnSync('sh', [file, ...args], { encoding: 'utf8', cwd, env: merged });
}

for (const corrupt of [false, true]) {
  test(`pinned Nix installer verifies before execution corrupt=${corrupt}`, unix, t => {
    const dir = fixture(t);
    fs.mkdirSync(path.join(dir, 'scripts'));
    const source = fs.readFileSync(installer, 'utf8');
    const version = source.match(/^version=(\S+)$/m);
    assert.ok(version, 'installer must keep a machine-readable version line');
    const archiveName = `nix-${version[1]}-x86_64-linux`;

    const payload = path.join(dir, 'payload');
    fs.mkdirSync(path.join(payload, archiveName), { recursive: true });
    const log = path.join(dir, 'executed');
    script(path.join(payload, archiveName, 'install'), `printf '%s\\n' "$@" >> "$EXEC_LOG"\n`);
    const archive = path.join(dir, 'archive.tar.xz');
    assert.equal(spawnSync('tar', ['-cJf', archive, '-C', payload, archiveName]).status, 0);

    // The reviewed digest is computed from the intact archive; corruption is
    // injected afterwards so only the runtime check can reject it.
    const hash = createHash('sha256').update(fs.readFileSync(archive)).digest('hex');
    const anchor = /^expected_sha256=[0-9a-f]{64}$/m;
    assert.match(source, anchor);
    fs.writeFileSync(
      path.join(dir, 'scripts/install-nix.sh'),
      source.replace(anchor, `expected_sha256=${hash}`),
    );
    if (corrupt) fs.appendFileSync(archive, 'corrupt');

    script(path.join(dir, 'bin/uname'), '[ "$1" = -s ] && echo Linux || echo x86_64\n');
    // Hash the downloaded bytes with Node so the Linux installer fixture
    // does not depend on GNU sha256sum being installed on the test host.
    const hashHelper = path.join(dir, 'sha256sum.js');
    fs.writeFileSync(hashHelper,
      "const fs = require('node:fs');\n" +
      "const { createHash } = require('node:crypto');\n" +
      "console.log(createHash('sha256').update(fs.readFileSync(process.argv[2])).digest('hex'));\n");
    script(path.join(dir, 'bin/sha256sum'), 'exec "$NODE_BINARY" "$HASH_HELPER" "$@"\n');
    script(path.join(dir, 'bin/curl'),
      'while [ "$#" -gt 0 ]; do\n' +
      '  if [ "$1" = -o ]; then cp "$FAKE_ARCHIVE" "$2"; exit 0; fi\n' +
      '  shift\n' +
      'done\n' +
      'exit 1\n');
    const out = run(path.join(dir, 'scripts/install-nix.sh'), [], {
      PATH: path.join(dir, 'bin') + path.delimiter + process.env.PATH,
      FAKE_ARCHIVE: archive,
      EXEC_LOG: log,
      NODE_BINARY: process.execPath,
      HASH_HELPER: hashHelper,
    });

    assert.equal(out.status, corrupt ? 1 : 0, out.stderr);
    assert.equal(fs.existsSync(log), !corrupt);
    if (corrupt) {
      assert.match(out.stderr, /Checksum mismatch/);
    } else {
      const args = fs.readFileSync(log, 'utf8').trim().split('\n');
      assert.deepEqual(args, ['--no-daemon', '--no-channel-add']);
    }
  });
}

function publisherFixture(t, { changed = true } = {}) {
  const dir = fixture(t);
  fs.mkdirSync(path.join(dir, 'nix'));
  fs.writeFileSync(path.join(dir, 'nix/stable.nix'), 'old stable\n');
  fs.writeFileSync(path.join(dir, 'nix/latest.nix'), 'old latest\n');
  const env = {
    PATH: path.join(dir, 'bin') + path.delimiter + process.env.PATH,
    HOME: dir,
    GIT_CONFIG_GLOBAL: '/dev/null',
    GIT_CONFIG_NOSYSTEM: '1',
    ATOMGIT_REPOSITORY: 'fixture/repo',
    ATOMGIT_REF_NAME: 'main',
    CALL_LOG: path.join(dir, 'calls'),
    DATA_LOG: path.join(dir, 'data'),
    HEADER_LOG: path.join(dir, 'headers'),
  };
  for (const args of [
    ['init', '-q'],
    ['add', 'nix'],
    ['-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', 'commit', '-qm', 'fixture'],
  ]) {
    const git = spawnSync('git', args, { cwd: dir, env });
    assert.equal(git.status, 0, git.stderr);
  }
  if (changed) {
    fs.writeFileSync(path.join(dir, 'nix/stable.nix'), 'new stable\n');
    fs.writeFileSync(path.join(dir, 'nix/latest.nix'), 'new latest\n');
  }
  script(path.join(dir, 'bin/curl'),
    'printf \'%s\\n\' "$@" >> "$CALL_LOG"\n' +
    'printf \'%s\\n\' --CALL-- >> "$CALL_LOG"\n' +
    'while [ "$#" -gt 0 ]; do\n' +
    '  case "$1" in\n' +
    '    --output) shift; printf \'%s\' "$FAKE_BODY" > "$1" ;;\n' +
    '    --data) shift; printf \'%s\\n\' "$1" >> "$DATA_LOG" ;;\n' +
    '    @*) cat "${1#@}" >> "$HEADER_LOG"; printf \'\\n\' >> "$HEADER_LOG" ;;\n' +
    '  esac\n' +
    '  shift\n' +
    'done\n' +
    'if [ "${FAKE_CURL_EXIT:-0}" -ne 0 ]; then exit "$FAKE_CURL_EXIT"; fi\n' +
    'printf \'%s\' "${FAKE_STATUS:-200}"\n');
  return { dir, env };
}
const callCount = env => fs.readFileSync(env.CALL_LOG, 'utf8').split('--CALL--').length - 1;

test('publish is a no-op when nothing changed, even without a token', unix, t => {
  const { dir, env } = publisherFixture(t, { changed: false });
  const out = run(publisher, [], { ...env, NIX_UPDATE_TOKEN: '' }, dir);
  assert.equal(out.status, 0, out.stderr);
  assert.match(out.stdout, /already current/);
  assert.equal(fs.existsSync(env.CALL_LOG), false);
});

test('publish fails closed when the update token is missing', unix, t => {
  const { dir, env } = publisherFixture(t);
  const out = run(publisher, [], { ...env, NIX_UPDATE_TOKEN: '' }, dir);
  assert.equal(out.status, 1);
  assert.match(out.stderr, /NIX_UPDATE_TOKEN/);
  assert.equal(fs.existsSync(env.CALL_LOG), false);
});

test('publish requires repository context', unix, t => {
  const { dir, env } = publisherFixture(t);
  const out = run(publisher, [], { ...env, NIX_UPDATE_TOKEN: TOKEN, ATOMGIT_REPOSITORY: undefined }, dir);
  assert.equal(out.status, 1);
  assert.match(out.stderr, /ATOMGIT_REPOSITORY/);
  assert.equal(fs.existsSync(env.CALL_LOG), false);
});

test('publish rejects unsafe branch names before any request', unix, t => {
  const { dir, env } = publisherFixture(t);
  const out = run(publisher, [], { ...env, NIX_UPDATE_TOKEN: TOKEN, ATOMGIT_REF_NAME: 'we"ird' }, dir);
  assert.equal(out.status, 1);
  assert.match(out.stderr, /[Bb]ranch/);
  assert.equal(fs.existsSync(env.CALL_LOG), false);
});

test('publish reports bounded redacted diagnostics on API failure', unix, t => {
  const { dir, env } = publisherFixture(t);
  const body = `{"message":"denied ${TOKEN} ${'y'.repeat(500)}"}`;
  const out = run(publisher, [], { ...env, NIX_UPDATE_TOKEN: TOKEN, FAKE_STATUS: '403', FAKE_BODY: body }, dir);
  assert.equal(out.status, 1);
  assert.match(out.stdout, /HTTP status 403/);
  assert.match(out.stdout, /denied <redacted>/);
  assert.ok(!out.stdout.includes(TOKEN) && !out.stderr.includes(TOKEN), 'token leaked to output');
  for (const line of out.stdout.split('\n')) {
    if (line.includes('y'.repeat(20))) assert.ok(line.length <= 302, `summary not bounded: ${line.length}`);
  }
  assert.ok(!fs.existsSync(path.join(dir, 'contents-api-response.json')), 'response leaked into workspace');
  assert.equal(callCount(env), 1);
  const calls = fs.readFileSync(env.CALL_LOG, 'utf8');
  assert.ok(calls.includes('api.atomgit.com/api/v5/repos/fixture/repo/contents/nix/stable.nix'));
  assert.ok(!calls.includes(TOKEN), 'token leaked to curl arguments');
  assert.ok(!calls.includes(`Bearer ${TOKEN}`), 'authorization inlined instead of header file');
  assert.ok(fs.readFileSync(env.HEADER_LOG, 'utf8').includes(`Authorization: Bearer ${TOKEN}`));
});

for (const [name, body] of [
  ['repeated tokens', JSON.stringify({ message: (TOKEN + ' ').repeat(20) })],
  ['control characters before a token', '\n'.repeat(390) + TOKEN + '\x7f'],
  ['token at the output boundary', 'x'.repeat(295) + TOKEN],
]) {
  test(`publish redacts before truncating: ${name}`, unix, t => {
    const { dir, env } = publisherFixture(t);
    const out = run(publisher, [], {
      ...env, NIX_UPDATE_TOKEN: TOKEN, FAKE_STATUS: '403', FAKE_BODY: body,
    }, dir);
    assert.equal(out.status, 1, out.stderr);
    const sanitized = body.replace(/[\x00-\x1f\x7f]/g, '').split(TOKEN).join('<redacted>').slice(0, 300);
    const summary = out.stdout.trimEnd().split('\n').at(-1);
    assert.equal(summary, sanitized);
    assert.ok(!out.stdout.includes('SPEC_TOKEN') && !out.stderr.includes('SPEC_TOKEN'));
    assert.equal(callCount(env), 1);
  });
}

test('publish fails on transport errors', unix, t => {
  const { dir, env } = publisherFixture(t);
  const out = run(publisher, [], { ...env, NIX_UPDATE_TOKEN: TOKEN, FAKE_CURL_EXIT: '7' }, dir);
  assert.equal(out.status, 1);
  assert.match(out.stdout, /curl exit code 7/);
  assert.equal(callCount(env), 1);
});

test('publish sends both files with bearer headers and no credential traces', unix, t => {
  const { dir, env } = publisherFixture(t);
  const body = `{"content":"${TOKEN}","message":"ok ${'z'.repeat(2000)}"}`;
  const out = run(publisher, [], { ...env, NIX_UPDATE_TOKEN: TOKEN, FAKE_STATUS: '200', FAKE_BODY: body }, dir);
  assert.equal(out.status, 0, out.stderr);
  assert.ok(!out.stdout.includes(TOKEN) && !out.stderr.includes(TOKEN), 'token leaked to output');
  assert.ok(!out.stdout.includes('z'.repeat(50)), 'raw response body printed');

  const calls = fs.readFileSync(env.CALL_LOG, 'utf8');
  assert.equal(callCount(env), 2);
  assert.ok(calls.indexOf('contents/nix/stable.nix') < calls.indexOf('contents/nix/latest.nix'));
  assert.ok(!calls.includes(TOKEN), 'token leaked to curl arguments');

  const headers = fs.readFileSync(env.HEADER_LOG, 'utf8');
  assert.equal(headers.split(`Authorization: Bearer ${TOKEN}`).length - 1, 2);

  const payloads = fs.readFileSync(env.DATA_LOG, 'utf8').trim().split('\n').map(JSON.parse);
  assert.deepEqual(payloads.map(p => p.branch), ['main', 'main']);
  assert.deepEqual(payloads.map(p => p.message), ['chore: update Nix packages', 'chore: update Nix packages']);
  for (const [i, file] of ['nix/stable.nix', 'nix/latest.nix'].entries()) {
    assert.equal(payloads[i].content, fs.readFileSync(path.join(dir, file)).toString('base64'));
    const sha = spawnSync('git', ['rev-parse', `HEAD:${file}`], { cwd: dir, env, encoding: 'utf8' });
    assert.equal(payloads[i].sha, sha.stdout.trim());
  }
});

const guard = path.join(root, 'scripts/guard-nix-update-context.sh');
const GUARD_ENV = {
  GUARD_REPOSITORY: 'hust-open-atom-club/atomgit-cli',
  GUARD_REF: 'refs/heads/main',
  GUARD_EVENT: 'push',
};

test('context guard accepts the intended repository, ref, and event', unix, () => {
  for (const ref of ['refs/heads/main', 'refs/heads/test', 'refs/heads/nix-update']) {
    for (const event of ['push', 'workflow_dispatch']) {
      const out = run(guard, [], { ...GUARD_ENV, GUARD_REF: ref, GUARD_EVENT: event });
      assert.equal(out.status, 0, out.stderr);
    }
  }
});

test('context guard refuses other repositories, refs, and events', unix, () => {
  const cases = [
    [{ ...GUARD_ENV, GUARD_REPOSITORY: 'someone/atomgit-cli' }, /repository 'someone\/atomgit-cli'/],
    [{ ...GUARD_ENV, GUARD_REPOSITORY: undefined }, /repository ''/],
    [{ ...GUARD_ENV, GUARD_REF: 'refs/heads/feature' }, /ref 'refs\/heads\/feature'/],
    [{ ...GUARD_ENV, GUARD_REF: 'refs/tags/v1.0.0' }, /ref 'refs\/tags\/v1\.0\.0'/],
    [{ ...GUARD_ENV, GUARD_REF: undefined }, /ref ''/],
    [{ ...GUARD_ENV, GUARD_EVENT: 'schedule' }, /event 'schedule'/],
    [{ ...GUARD_ENV, GUARD_EVENT: undefined }, /event ''/],
  ];
  for (const [env, pattern] of cases) {
    const out = run(guard, [], env);
    assert.equal(out.status, 1, `expected refusal for ${JSON.stringify(env)}`);
    assert.match(out.stderr, pattern);
  }
});
