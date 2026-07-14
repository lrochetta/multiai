'use strict';

const { execFileSync } = require('child_process');
const path = require('path');

function trustedPowerShell(env = process.env) {
  const root = env.SystemRoot || env.WINDIR;
  if (!root || !/^[A-Za-z]:[\\/]/.test(root)) {
    throw new Error('SystemRoot is unavailable');
  }
  return path.win32.join(root, 'System32', 'WindowsPowerShell', 'v1.0', 'powershell.exe');
}

function runVersionProbe(binaryPath, timeoutMs, options = {}) {
  const platform = options.platform || process.platform;
  const exec = options.exec || execFileSync;
  const env = options.env || process.env;
  if (platform !== 'win32') {
    return exec(binaryPath, ['--version'], {
      encoding: 'utf8', env: { ...env, MULTIAI_SKIP_UPDATE: '1' },
      stdio: ['ignore', 'pipe', 'pipe'], timeout: timeoutMs
    });
  }

  const controller = path.join(__dirname, '..', 'scripts', 'version-probe-controller.ps1');
  return exec(trustedPowerShell(env), [
    '-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-File', controller,
    '-BinaryPath', path.resolve(binaryPath), '-TimeoutMilliseconds', String(timeoutMs)
  ], {
    encoding: 'utf8', env: { ...env, MULTIAI_SKIP_UPDATE: '1' },
    stdio: ['ignore', 'pipe', 'pipe'], timeout: timeoutMs + 10000,
    windowsHide: true
  });
}

module.exports = { runVersionProbe, trustedPowerShell };
