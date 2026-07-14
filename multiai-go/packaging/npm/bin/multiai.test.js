'use strict';

const assert = require('node:assert/strict');
const path = require('path');
const test = require('node:test');

const pkg = require('../package.json');
const {
  buildGlobalInstallArgs,
  buildGlobalPrefixArgs,
  buildGlobalRootArgs,
  buildNpmInvocation,
  npmMajorVersion,
  runNative
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

test('global prefix lookup preserves a custom npm prefix', () => {
  assert.deepEqual(buildGlobalPrefixArgs([
    'install', '--global', '--prefix', 'D:/tools/multiai', `multiai@${pkg.version}`
  ]), ['prefix', '--global', '--prefix', 'D:/tools/multiai']);
  assert.deepEqual(buildGlobalPrefixArgs([
    'install', '--global', `multiai@${pkg.version}`
  ]), ['prefix', '--global']);
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

test('native shim bounds execution and preserves the native exit code', () => {
  const calls = [];
  const fakeSpawn = (file, args, options) => {
    calls.push({ file, args, options });
    return { status: 7 };
  };

  assert.equal(runNative(['--version'], fakeSpawn, 'multiai.exe'), 7);
  assert.equal(calls.length, 1);
  assert.deepEqual(calls[0].args, ['--version']);
  assert.equal(calls[0].options.timeout, 30000);
  assert.equal(calls[0].options.windowsHide, true);
});

test('native shim converts a timeout into a controlled failure', () => {
  const timeout = Object.assign(new Error('timed out'), { code: 'ETIMEDOUT' });
  const originalError = console.error;
  const messages = [];
  console.error = (...args) => messages.push(args.join(' '));
  try {
    assert.equal(runNative([], () => ({ error: timeout }), 'multiai.exe'), 1);
  } finally {
    console.error = originalError;
  }
  assert.match(messages.join('\n'), /timed out after 30s/);
});

test('native shim does not cap interactive or launched commands', () => {
  let options;
  assert.equal(runNative(['launch', '-p', 'ds'], (_file, _args, value) => {
    options = value;
    return { status: 0 };
  }, 'multiai.exe'), 0);
  assert.equal(Object.hasOwn(options, 'timeout'), false);
});
