'use strict';

const { spawnSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const PATH_SCRIPT = path.join(__dirname, '..', 'scripts', 'ensure-user-path.ps1');
const PATH_UPDATE_TIMEOUT_MS = 20000;

function customPrefixFromInstallArgs(installArgs) {
  const prefixIndex = installArgs.indexOf('--prefix');
  if (prefixIndex >= 0 && installArgs[prefixIndex + 1]) {
    return installArgs[prefixIndex + 1];
  }
  return null;
}

function buildGlobalPrefixArgs(installArgs) {
  const args = ['prefix', '--global'];
  const customPrefix = customPrefixFromInstallArgs(installArgs);
  if (customPrefix) args.push('--prefix', customPrefix);
  return args;
}

function powershellExecutable(env = process.env) {
  const systemRoot = env.SystemRoot || env.SYSTEMROOT || env.WINDIR;
  if (!systemRoot) {
    throw new Error('SystemRoot is unavailable; cannot safely locate Windows PowerShell');
  }
  return path.win32.join(
    systemRoot,
    'System32',
    'WindowsPowerShell',
    'v1.0',
    'powershell.exe'
  );
}

function windowsCommandProcessor(env = process.env) {
  const systemRoot = env.SystemRoot || env.SYSTEMROOT || env.WINDIR;
  if (!systemRoot) {
    throw new Error('SystemRoot is unavailable; cannot safely locate cmd.exe');
  }
  return path.win32.join(systemRoot, 'System32', 'cmd.exe');
}

function ensureWindowsUserPath(prefix, options = {}) {
  const platform = options.platform || process.platform;
  const env = options.env || process.env;
  const fsModule = options.fsModule || fs;
  const spawn = options.spawnSync || spawnSync;

  if (platform !== 'win32') return { status: 'not-windows', prefix };
  if (!/^[A-Za-z]:[\\/]/.test(prefix) || /[;\0\r\n]/.test(prefix)) {
    throw new Error(`npm returned an unsafe Windows prefix: ${prefix}`);
  }

  const commandShim = path.win32.join(prefix, 'multiai.cmd');
  if (!fsModule.existsSync(commandShim)) {
    throw new Error(`npm did not create the command shim: ${commandShim}`);
  }
  if (env.MULTIAI_SKIP_PATH_UPDATE === '1') return { status: 'skipped', prefix };
  if (!fsModule.existsSync(PATH_SCRIPT)) {
    throw new Error(`PATH helper is missing from the npm package: ${PATH_SCRIPT}`);
  }

  const powershell = powershellExecutable(env);
  if (!fsModule.existsSync(powershell)) {
    throw new Error(`Windows PowerShell was not found at ${powershell}`);
  }

  const result = spawn(powershell, [
    '-NoProfile',
    '-NonInteractive',
    '-File',
    PATH_SCRIPT
  ], {
    encoding: 'utf8',
    env: { ...env, MULTIAI_PATH_ENTRY: prefix },
    maxBuffer: 64 * 1024,
    timeout: PATH_UPDATE_TIMEOUT_MS,
    windowsHide: true
  });

  if (result.error) throw result.error;
  if (result.status !== 0) {
    const detail = String(result.stderr || result.stdout || '').trim();
    throw new Error(detail || `PATH helper exited with status ${result.status}`);
  }

  const output = String(result.stdout || '').trim().split(/\r?\n/).pop().replace(/^\uFEFF/, '');
  let payload;
  try {
    payload = JSON.parse(output);
  } catch {
    throw new Error(`PATH helper returned an unexpected result: ${output || '(empty)'}`);
  }
  if (
    !['added', 'present:user', 'present:machine'].includes(payload.status) ||
    typeof payload.effectivePath !== 'string' ||
    typeof payload.resolvedShim !== 'string'
  ) {
    throw new Error(`PATH helper returned an invalid result: ${output}`);
  }
  return {
    status: payload.status,
    prefix,
    effectivePath: payload.effectivePath,
    resolvedShim: payload.resolvedShim
  };
}

function processPathKey(env) {
  return Object.keys(env).find(key => key.toLowerCase() === 'path') || 'PATH';
}

function replaceProcessPath(env, value) {
  const result = { ...env };
  result[processPathKey(result)] = value;
  return result;
}

function prependProcessPath(env, prefix, platform = process.platform) {
  const result = { ...env };
  const pathKey = processPathKey(result);
  const delimiter = platform === 'win32' ? ';' : path.delimiter;
  result[pathKey] = result[pathKey] ? `${prefix}${delimiter}${result[pathKey]}` : prefix;
  return result;
}

module.exports = {
  buildGlobalPrefixArgs,
  ensureWindowsUserPath,
  powershellExecutable,
  prependProcessPath,
  replaceProcessPath,
  windowsCommandProcessor
};
