const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const root = path.resolve(__dirname, '..');
const unix = { skip: process.platform === 'win32' };

function writeScript(file, body) {
  fs.writeFileSync(file, `#!/bin/sh\n${body}`, { mode: 0o755 });
}

function fixture(t, scenario, options = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ag-vulncheck-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));

  const callLog = path.join(dir, 'calls.log');
  const binaryDir = path.join(dir, 'binaries');
  if (options.prebuilt) {
    fs.mkdirSync(binaryDir);
    fs.writeFileSync(path.join(binaryDir, 'ag-linux-amd64'), 'prebuilt');
    fs.writeFileSync(path.join(binaryDir, 'ag-windows-amd64'), 'prebuilt');
  }
  const fakeGo = path.join(dir, 'go');
  writeScript(fakeGo, `
echo "go $GOOS/$GOARCH $*" >> "$CALL_LOG"
case "$1" in
  build)
    while [ "$#" -gt 0 ]; do
      if [ "$1" = -o ]; then
        : > "$2"
        exit 0
      fi
      shift
    done
    exit 2
    ;;
  version)
    printf '%s: %s\\n' "$2" "$BINARY_TOOLCHAIN"
    ;;
  *)
    echo "unexpected go command: $*" >&2
    exit 90
    ;;
esac
`);

  const fakeScanner = path.join(dir, 'govulncheck');
  writeScript(fakeScanner, `
if [ "$1" = -version ]; then
  echo 'Go: go1.26.8'
  echo 'Scanner: govulncheck@v1.7.0'
  echo 'DB: https://vuln.go.dev'
  exit 0
fi
echo "scan $GOOS/$GOARCH $*" >> "$CALL_LOG"
case "$SCENARIO:$*" in
  windows-reachable:*'-mode=binary '*ag-windows-amd64)
    echo 'mock reachable Windows vulnerability'
    exit 3
    ;;
  informational:*'-scan=module '*|informational:*'-scan=package '*)
    echo 'mock informational finding'
    exit 3
    ;;
  tool-failure:*'-scan=module '*)
    echo 'mock database failure' >&2
    exit 1
    ;;
esac
echo 'No vulnerabilities found.'
`);

  const result = spawnSync('sh', [path.join(root, 'scripts/vulncheck.sh')], {
    cwd: root,
    encoding: 'utf8',
    env: {
      ...process.env,
      GO: fakeGo,
      GOVULNCHECK: fakeScanner,
      GOVULNCHECK_VERSION: 'v1.7.0',
      RELEASE_GO_TOOLCHAIN: 'go1.26.8',
      VULNCHECK_TARGETS: 'linux/amd64 windows/amd64',
      VULNCHECK_PACKAGE_TARGETS: 'linux/amd64 windows/amd64',
      VULNCHECK_BUILD: options.prebuilt ? '0' : '1',
      VULNCHECK_BINARY_DIR: binaryDir,
      CALL_LOG: callLog,
      BINARY_TOOLCHAIN: options.binaryToolchain || 'go1.26.8',
      SCENARIO: scenario,
    },
  });
  return { result, calls: fs.readFileSync(callLog, 'utf8') };
}

test('a Windows-only reachable finding fails the multi-platform gate', unix, t => {
  const { result, calls } = fixture(t, 'windows-reachable');

  assert.equal(result.status, 3, result.stderr);
  assert.match(result.stderr, /Reachable vulnerabilities detected for windows\/amd64/);
  assert.match(calls, /scan linux\/amd64 .*?-mode=binary .*ag-linux-amd64/);
  assert.match(calls, /scan windows\/amd64 .*?-mode=binary .*ag-windows-amd64/);
});

test('package and module findings stay visible without failing the reachable gate', unix, t => {
  const { result } = fixture(t, 'informational');

  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /module-level findings detected for linux\/amd64 \(triage only\)/);
  assert.match(result.stdout, /package-level findings detected for windows\/amd64 \(triage only\)/);
  assert.match(result.stdout, /All release targets passed the reachable vulnerability gate/);
});

test('scanner and database failures cannot report a clean scan', unix, t => {
  const { result } = fixture(t, 'tool-failure');

  assert.equal(result.status, 1);
  assert.match(result.stderr, /module-level vulnerability scan failed .* exit code 1/);
  assert.doesNotMatch(result.stdout, /passed the reachable vulnerability gate/);
});

test('prebuilt artifacts cannot bypass the release toolchain check', unix, t => {
  const { result, calls } = fixture(t, 'clean', {
    prebuilt: true,
    binaryToolchain: 'go1.26.7',
  });

  assert.equal(result.status, 1);
  assert.match(result.stderr, /Expected linux\/amd64 binary built with go1\.26\.8/);
  assert.doesNotMatch(calls, /scan /);
});
