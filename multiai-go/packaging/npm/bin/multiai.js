#!/usr/bin/env node
// Cross-platform shim: exec the native multiai binary.
// If the binary is missing (npm postinstall skipped or failed),
// downloads it automatically on first run (lazy init).

'use strict';

const { spawnSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const exe = process.platform === 'win32' ? 'multiai.exe' : 'multiai';
const native = path.join(__dirname, 'native', exe);

// Lazy init: download the native binary if missing.
if (!fs.existsSync(native)) {
  const installer = path.join(__dirname, '..', 'install.js');
  if (fs.existsSync(installer)) {
    console.error('multiai: downloading native binary (first run)...');
    try {
      require('child_process').execFileSync(process.execPath, [installer], {
        stdio: 'inherit',
        env: { ...process.env }
      });
    } catch (_) {
      console.error('multiai: download failed. Install manually:');
      console.error('  npm rebuild multiai');
      console.error('  or: https://github.com/lrochetta/multiai/releases');
      process.exit(1);
    }
  } else {
    console.error('multiai: native binary missing and installer not found.');
    console.error('Reinstall with: npm install multiai');
    process.exit(1);
  }
}

// Second check after attempted download.
if (!fs.existsSync(native)) {
  console.error('multiai: native binary still missing after download attempt.');
  console.error('Try: npm rebuild multiai');
  process.exit(1);
}

const result = spawnSync(native, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  console.error('multiai: failed to start binary:', result.error.message);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
