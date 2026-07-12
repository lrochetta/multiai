'use strict';

const { execFileSync } = require('child_process');
const path = require('path');

const pkg = require('./package.json');
const repoRoot = path.resolve(__dirname, '..', '..', '..');

function validateReleaseState(version, statusOutput, tagsOutput) {
  if (statusOutput.trim()) {
    throw new Error('refusing npm publish from a dirty Git worktree');
  }

  const expectedTag = `v${version}`;
  const tags = tagsOutput.split(/\r?\n/).map(tag => tag.trim()).filter(Boolean);
  if (!tags.includes(expectedTag)) {
    throw new Error(`HEAD must be tagged ${expectedTag} before npm publish`);
  }
}

function git(args) {
  return execFileSync('git', args, {
    cwd: repoRoot,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe']
  });
}

function run() {
  const status = git(['status', '--porcelain']);
  const tags = git(['tag', '--points-at', 'HEAD']);
  validateReleaseState(pkg.version, status, tags);
  console.log(`OK: clean worktree and tag v${pkg.version}`);
}

if (require.main === module) {
  try {
    run();
  } catch (err) {
    console.error('Release preflight failed: ' + err.message);
    process.exit(1);
  }
}

module.exports = { validateReleaseState };
