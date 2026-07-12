'use strict';

const assert = require('node:assert/strict');
const path = require('path');
const test = require('node:test');

const pkg = require('../package.json');
const {
  buildGlobalInstallArgs,
  buildGlobalRootArgs,
  buildNpmInvocation,
  npmMajorVersion
} = require('./multiai');

test('legacy npx install command becomes a real global npm install', () => {
  const env = { npm_config_user_agent: 'npm/11.16.0 node/v24.18.0 win32 x64' };
  assert.deepEqual(buildGlobalInstallArgs(['install'], env), [
    'install',
    '--global',
    '--foreground-scripts',
    '--allow-scripts=multiai',
    `multiai@${pkg.version}`
  ]);
});

test('legacy custom install directory maps to npm prefix', () => {
  const env = { npm_config_user_agent: 'npm/10.9.0 node/v22.0.0 win32 x64' };
  const args = buildGlobalInstallArgs(['install', '--yes', 'portable'], env);
  assert.deepEqual(args, [
    'install',
    '--global',
    '--foreground-scripts',
    '--prefix',
    path.resolve('portable'),
    `multiai@${pkg.version}`
  ]);
});

test('normal CLI arguments are not treated as installation', () => {
  assert.equal(buildGlobalInstallArgs(['--version']), null);
  assert.equal(buildGlobalInstallArgs([]), null);
});

test('npm 11 install-script approval is feature-gated by npm major', () => {
  assert.equal(npmMajorVersion({ npm_config_user_agent: 'npm/11.16.0 node/v24.18.0' }), 11);
  assert.equal(npmMajorVersion({ npm_config_user_agent: 'npm/10.9.0 node/v22.0.0' }), 10);
  assert.equal(npmMajorVersion({}), 0);
});

test('global root lookup preserves a custom npm prefix', () => {
  assert.deepEqual(buildGlobalRootArgs([
    'install', '--global', '--prefix', 'D:/tools/multiai', `multiai@${pkg.version}`
  ]), ['root', '--global', '--prefix', 'D:/tools/multiai']);
  assert.deepEqual(buildGlobalRootArgs(['install', '--global', `multiai@${pkg.version}`]), [
    'root', '--global'
  ]);
});

test('npm invocation reuses the npm CLI that launched npx', () => {
  const invocation = buildNpmInvocation(['install'], { npm_execpath: 'C:/npm/npm-cli.js' }, 'win32');
  assert.equal(invocation.command, process.execPath);
  assert.deepEqual(invocation.args, ['C:/npm/npm-cli.js', 'install']);
});

test('npm invocation has a platform fallback outside npm', () => {
  assert.deepEqual(buildNpmInvocation(['install'], {}, 'win32'), {
    command: 'npm.cmd',
    args: ['install']
  });
  assert.deepEqual(buildNpmInvocation(['install'], {}, 'linux'), {
    command: 'npm',
    args: ['install']
  });
});
