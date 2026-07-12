#!/usr/bin/env node
// Cross-platform shim: exec the native multiai binary.

'use strict';

const { spawnSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const exe = process.platform === 'win32' ? 'multiai.exe' : 'multiai';
const native = path.join(__dirname, 'native', exe);

if (!fs.existsSync(native)) {
  console.error('');
  console.error('  multiai: native binary not installed yet.');
  console.error('');
  console.error('  This happens when npm blocks postinstall scripts.');
  console.error('  Run this ONCE to fix it:');
  console.error('');
  console.error('    npm approve-scripts multiai');
  console.error('    npm rebuild multiai');
  console.error('');
  console.error('  Or download manually:');
  console.error('    https://github.com/lrochetta/multiai/releases/latest');
  console.error('');
  process.exit(1);
}

const result = spawnSync(native, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  console.error('multiai: failed to start binary:', result.error.message);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
