const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { join } = require('node:path');
const test = require('node:test');

const workflow = readFileSync(
  join(__dirname, '..', '.gitcode', 'workflows', 'ci.yml'),
  'utf8',
);

function job(name) {
  const startMarker = `  ${name}:\n`;
  const start = workflow.indexOf(startMarker);
  assert.notEqual(start, -1, `missing ${name} job`);

  const remainder = workflow.slice(start + startMarker.length);
  const nextJob = remainder.search(/^  [a-z][a-z0-9-]*:\n/m);
  return nextJob === -1 ? remainder : remainder.slice(0, nextJob);
}

test('fast quality gate avoids duplicate preferred-toolchain tests', () => {
  const gate = job('quality-and-test');

  assert.match(gate, /run: time make go-min-version test-min-go/);
  assert.match(gate, /run: time make lint/);
  assert.match(gate, /run: time make build/);
  assert.match(gate, /run: time make docs-reference-check/);
  assert.doesNotMatch(gate, /run: (?:time )?make test\s*$/m);
});

test('expensive Go jobs wait for the quality gate and then remain independent', () => {
  for (const name of [
    'race-test',
    'cross-build',
    'platform-test-compile',
    'vulnerability-scan',
  ]) {
    assert.match(job(name), /^    needs: quality-and-test$/m, name);
  }

  assert.doesNotMatch(job('npm-test'), /^    needs:/m);
});

test('each Go job explicitly restores module, toolchain, and build caches', () => {
  const goJobs = [
    'quality-and-test',
    'race-test',
    'cross-build',
    'platform-test-compile',
    'vulnerability-scan',
  ];

  for (const name of goJobs) {
    const body = job(name);
    assert.equal((body.match(/uses: setup-go/g) || []).length, 1, name);
    assert.equal((body.match(/uses: cache/g) || []).length, 2, name);
    assert.match(body, /^          path: ~\/go\/pkg\/mod$/m, name);
    assert.match(body, /^          path: ~\/\.cache\/go-build$/m, name);
    assert.match(
      body,
      /^          key: go-mod-\$\{\{ runner\.os \}\}-\$\{\{ hashFiles\('go\.mod', 'go\.sum'\) \}\}$/m,
      name,
    );
    assert.match(
      body,
      /^          key: go-build-\$\{\{ runner\.os \}\}-\$\{\{ hashFiles\('\*\*\/\*\.go', 'go\.mod', 'go\.sum'\) \}\}$/m,
      name,
    );
    assert.doesNotMatch(body, /^          cache: true$/m, name);
  }
});
