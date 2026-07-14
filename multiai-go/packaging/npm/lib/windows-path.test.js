'use strict';

const assert = require('node:assert/strict');
const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawnSync } = require('child_process');
const test = require('node:test');

const {
  ensureWindowsUserPath,
  powershellExecutable,
  prependProcessPath,
  replaceProcessPath,
  windowsCommandProcessor
} = require('./windows-path');

test('powershellExecutable uses the trusted SystemRoot binary', () => {
  assert.equal(
    powershellExecutable({ SystemRoot: 'C:\\Windows' }),
    'C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe'
  );
  assert.throws(() => powershellExecutable({}), /SystemRoot/);
});

test('windowsCommandProcessor uses the trusted SystemRoot binary', () => {
  assert.equal(
    windowsCommandProcessor({ SystemRoot: 'C:\\Windows' }),
    'C:\\Windows\\System32\\cmd.exe'
  );
  assert.throws(() => windowsCommandProcessor({}), /SystemRoot/);
});

test('process PATH helpers preserve the existing Windows PATH key', () => {
  assert.deepEqual(
    prependProcessPath({ Path: 'C:\\Windows', OTHER: 'value' }, 'D:\\npm', 'win32'),
    { Path: 'D:\\npm;C:\\Windows', OTHER: 'value' }
  );
  assert.deepEqual(prependProcessPath({}, 'D:\\npm', 'win32'), { PATH: 'D:\\npm' });
  assert.deepEqual(
    replaceProcessPath({ Path: 'old', OTHER: 'value' }, 'C:\\Windows;D:\\npm'),
    { Path: 'C:\\Windows;D:\\npm', OTHER: 'value' }
  );
});

test('ensureWindowsUserPath honours the enterprise escape hatch', () => {
  assert.deepEqual(ensureWindowsUserPath('D:\\npm', {
    platform: 'win32',
    env: { MULTIAI_SKIP_PATH_UPDATE: '1' },
    fsModule: { existsSync: () => true }
  }), { status: 'skipped', prefix: 'D:\\npm' });
});

test('ensureWindowsUserPath rejects unsafe prefixes before spawning PowerShell', () => {
  assert.throws(() => ensureWindowsUserPath('relative', {
    platform: 'win32',
    env: {}
  }), /unsafe Windows prefix/);
  assert.throws(() => ensureWindowsUserPath('D:\\safe;D:\\injected', {
    platform: 'win32',
    env: {}
  }), /unsafe Windows prefix/);
  for (const prefix of [
    '\\\\server\\share',
    '\\\\?\\C:\\device',
    '\\\\.\\C:\\device'
  ]) {
    assert.throws(() => ensureWindowsUserPath(prefix, {
      platform: 'win32',
      env: {}
    }), /unsafe Windows prefix/);
  }
});

test('ensureWindowsUserPath passes the prefix through the environment only', () => {
  const calls = [];
  const fakeSpawn = (command, args, options) => {
    calls.push({ command, args, options });
    return {
      status: 0,
      stdout: '{"status":"added","effectivePath":"C:\\\\Windows;D:\\\\Tools\\\\multiai","resolvedShim":"D:\\\\Tools\\\\multiai\\\\multiai.cmd"}\r\n',
      stderr: ''
    };
  };
  const result = ensureWindowsUserPath('D:\\Tools\\multiai', {
    platform: 'win32',
    env: { SystemRoot: 'C:\\Windows', Path: 'C:\\Windows' },
    fsModule: { existsSync: () => true },
    spawnSync: fakeSpawn
  });

  assert.deepEqual(result, {
    status: 'added',
    prefix: 'D:\\Tools\\multiai',
    effectivePath: 'C:\\Windows;D:\\Tools\\multiai',
    resolvedShim: 'D:\\Tools\\multiai\\multiai.cmd'
  });
  assert.equal(calls.length, 1);
  assert.equal(calls[0].command, 'C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe');
  assert.deepEqual(calls[0].args.slice(0, 2), ['-NoProfile', '-NonInteractive']);
  assert.equal(calls[0].options.env.MULTIAI_PATH_ENTRY, 'D:\\Tools\\multiai');
  assert.equal(calls[0].args.join(' ').includes('D:\\Tools\\multiai'), false);
  assert.equal(calls[0].options.windowsHide, true);
});

test('ensureWindowsUserPath fails closed when the helper fails', () => {
  assert.throws(() => ensureWindowsUserPath('D:\\Tools\\multiai', {
    platform: 'win32',
    env: { SystemRoot: 'C:\\Windows' },
    fsModule: { existsSync: () => true },
    spawnSync: () => ({ status: 1, stdout: '', stderr: 'registry denied' })
  }), /registry denied/);
});

test('PowerShell helper plans an idempotent user PATH update', {
  skip: process.platform !== 'win32'
}, () => {
  const prefix = fs.mkdtempSync(path.join(os.tmpdir(), 'multiai path é-'));
  const conflict = fs.mkdtempSync(path.join(os.tmpdir(), 'multiai conflict-'));
  const script = path.join(__dirname, '..', 'scripts', 'ensure-user-path.ps1');
  fs.writeFileSync(path.join(prefix, 'multiai.cmd'), '@echo off\r\n');
  fs.writeFileSync(path.join(conflict, 'multiai.cmd'), '@echo off\r\n');

  const runPlan = (userPath, machinePath, extraEnv = {}) => spawnSync(
    powershellExecutable(process.env),
    ['-NoProfile', '-NonInteractive', '-File', script, '-Mode', 'Plan'],
    {
      encoding: 'utf8',
      env: {
        ...process.env,
        ...extraEnv,
        MULTIAI_PATH_ENTRY: prefix,
        MULTIAI_TEST_USER_PATH: userPath,
        MULTIAI_TEST_MACHINE_PATH: machinePath
      },
      windowsHide: true
    }
  );

  try {
    const missing = runPlan('', 'C:\\Windows');
    assert.equal(missing.status, 0, missing.stderr);
    assert.equal(missing.stdout.trim(), `planned\t${prefix}`);

    const differentCase = runPlan(`C:\\Tools;"${prefix.toUpperCase()}\\"`, 'C:\\Windows');
    assert.equal(differentCase.status, 0, differentCase.stderr);
    assert.equal(JSON.parse(differentCase.stdout).status, 'present:user');

    const expanded = runPlan('%MULTIAI_TEST_PREFIX%\\', 'C:\\Windows', {
      MULTIAI_TEST_PREFIX: prefix
    });
    assert.equal(expanded.status, 0, expanded.stderr);
    assert.equal(JSON.parse(expanded.stdout).status, 'present:user');

    const machine = runPlan('C:\\Tools', `${prefix};C:\\Windows`);
    assert.equal(machine.status, 0, machine.stderr);
    assert.equal(JSON.parse(machine.stdout).status, 'present:machine');

    const shadowed = runPlan(`${conflict};${prefix}`, 'C:\\Windows');
    assert.equal(shadowed.status, 0, shadowed.stderr);
    assert.equal(
      path.win32.normalize(JSON.parse(shadowed.stdout).resolvedShim).toLowerCase(),
      path.win32.join(conflict, 'multiai.cmd').toLowerCase()
    );
  } finally {
    fs.rmSync(prefix, { recursive: true, force: true });
    fs.rmSync(conflict, { recursive: true, force: true });
  }
});

test('Windows command smoke resolves a generated shim through PATH', {
  skip: process.platform !== 'win32'
}, () => {
  const prefix = fs.mkdtempSync(path.join(os.tmpdir(), 'multiai fumée-'));
  fs.writeFileSync(path.join(prefix, 'multiai-smoke.cmd'), '@echo off\r\nexit /b 0\r\n');
  try {
    const effectivePath = `${prefix};${process.env.Path || process.env.PATH || ''}`;
    const smokeEnv = replaceProcessPath(process.env, effectivePath);
    const result = spawnSync(
      windowsCommandProcessor(process.env),
      ['/d', '/s', '/c', 'multiai-smoke.cmd'],
      {
        env: smokeEnv,
        windowsHide: true
      }
    );
    assert.equal(result.status, 0, result.error && result.error.message);
  } finally {
    fs.rmSync(prefix, { recursive: true, force: true });
  }
});
