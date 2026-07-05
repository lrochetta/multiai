#!/usr/bin/env node
// Cross-platform shim: exec the native multiai binary downloaded by
// install.js (postinstall), forwarding args, stdio and exit code.

'use strict';

const { spawnSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const exe = process.platform === 'win32' ? 'multiai.exe' : 'multiai';
const native = path.join(__dirname, 'native', exe);

if (!fs.existsSync(native)) {
  console.error('multiai: native binary missing.');
  console.error('Reinstall it with: npm rebuild multiai  (runs the postinstall download)');
  console.error('Or download manually: https://github.com/lrochetta/multiai/releases');
  process.exit(1);
}

const result = spawnSync(native, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  console.error(`multiai: failed to start binary: ${result.error.message}`);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
