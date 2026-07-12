'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');

const { validateReleaseState } = require('./release-preflight');

test('release preflight accepts a clean worktree tagged with the package version', () => {
  assert.doesNotThrow(() => validateReleaseState('1.2.3', '', 'v1.2.3\n'));
});

test('release preflight rejects a dirty worktree', () => {
  assert.throws(
    () => validateReleaseState('1.2.3', ' M package.json\n', 'v1.2.3\n'),
    /dirty Git worktree/
  );
});

test('release preflight rejects a mismatched tag', () => {
  assert.throws(
    () => validateReleaseState('1.2.3', '', 'v1.2.2\n'),
    /tagged v1\.2\.3/
  );
});
