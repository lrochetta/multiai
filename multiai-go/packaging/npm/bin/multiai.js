#!/usr/bin/env node
// Cross-platform shim: exec the native multiai binary.

'use strict';

const { spawnSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const pkg = require('../package.json');

const exe = process.platform === 'win32' ? 'multiai.exe' : 'multiai';
const native = path.join(__dirname, 'native', exe);

function npmMajorVersion(env = process.env) {
  const match = /(?:^|\s)npm\/(\d+)/.exec(env.npm_config_user_agent || '');
  return match ? Number(match[1]) : 0;
}

function buildGlobalInstallArgs(argv, env = process.env) {
  if (argv[0] !== 'install') return null;

  const customDir = argv.slice(1).find(arg => !arg.startsWith('-'));
  const args = ['install', '--global', '--foreground-scripts'];
  if (npmMajorVersion(env) >= 11) args.push('--allow-scripts=multiai');
  if (customDir) args.push('--prefix', path.resolve(customDir));
  args.push(`multiai@${pkg.version}`);
  return args;
}

function buildGlobalRootArgs(installArgs) {
  const args = ['root', '--global'];
  const prefixIndex = installArgs.indexOf('--prefix');
  if (prefixIndex >= 0 && installArgs[prefixIndex + 1]) {
    args.push('--prefix', installArgs[prefixIndex + 1]);
  }
  return args;
}

function buildNpmInvocation(args, env = process.env, platform = process.platform) {
  if (env.npm_execpath) {
    return { command: process.execPath, args: [env.npm_execpath, ...args] };
  }
  return { command: platform === 'win32' ? 'npm.cmd' : 'npm', args };
}

function installGlobally(args) {
  console.log(`Installing multiai ${pkg.version} globally...`);
  const invocation = buildNpmInvocation(args);
  const result = spawnSync(invocation.command, invocation.args, { stdio: 'inherit' });

  if (result.error) {
    console.error('multiai: global install failed:', result.error.message);
    return 1;
  }
  if (result.status !== 0) return result.status === null ? 1 : result.status;

  const rootArgs = buildGlobalRootArgs(args);
  const rootInvocation = buildNpmInvocation(rootArgs);
  const rootResult = spawnSync(rootInvocation.command, rootInvocation.args, {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'inherit']
  });
  if (rootResult.error || rootResult.status !== 0) {
    console.error('multiai: unable to locate the global npm installation.');
    return 1;
  }

  const globalRoot = rootResult.stdout.trim().split(/\r?\n/).pop();
  const globalShim = path.join(globalRoot, 'multiai', 'bin', 'multiai.js');
  if (!fs.existsSync(globalShim)) {
    console.error('multiai: global package installed without its command shim.');
    return 1;
  }

  const smoke = spawnSync(process.execPath, [globalShim, '--version'], {
    env: { ...process.env, MULTIAI_SKIP_UPDATE: '1' },
    stdio: 'inherit',
    timeout: 15000
  });
  if (smoke.error || smoke.status !== 0) {
    console.error('multiai: global install smoke test failed.');
    return 1;
  }

  console.log('multiai installed globally. Open a new terminal, then run: multiai');
  return 0;
}

function main(argv = process.argv.slice(2)) {
  if (!fs.existsSync(native)) {
    console.error('');
    console.error('  multiai: native binary download did not complete.');
    console.error('');
    console.error('  Re-run npx with install scripts enabled:');
    console.error('    npx --yes --allow-scripts=multiai multiai@latest install');
    console.error('');
    console.error('  Or download manually:');
    console.error('    https://github.com/lrochetta/multiai/releases/latest');
    console.error('');
    return 1;
  }

  const globalInstallArgs = buildGlobalInstallArgs(argv);
  if (globalInstallArgs) return installGlobally(globalInstallArgs);

  const result = spawnSync(native, argv, { stdio: 'inherit' });

  if (result.error) {
    console.error('multiai: failed to start binary:', result.error.message);
    return 1;
  }
  return result.status === null ? 1 : result.status;
}

if (require.main === module) {
  process.exitCode = main();
}

module.exports = {
  buildGlobalInstallArgs,
  buildGlobalRootArgs,
  buildNpmInvocation,
  npmMajorVersion
};
