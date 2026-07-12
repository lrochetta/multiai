#!/usr/bin/env node
// multiai npm installer.
//
// Runs on `npm install` (postinstall). Downloads the native Go binary from
// the GitHub release matching this package's version, verifies its SHA256
// against the release checksums.txt, extracts it into bin/native/ so the
// bin/multiai.js shim can exec it.
//
// Escape hatches:
//   MULTIAI_SKIP_DOWNLOAD=1   skip the download entirely (CI, offline dev)
//   MULTIAI_INSTALL_DIR=path  also copy the verified binary there

'use strict';

const { execFileSync } = require('child_process');
const crypto = require('crypto');
const fs = require('fs');
const https = require('https');
const os = require('os');
const path = require('path');

const pkg = require('./package.json');
const VERSION = pkg.version;
const REPO = 'lrochetta/multiai';
const BINARY_NAME = process.platform === 'win32' ? 'multiai.exe' : 'multiai';
const NATIVE_DIR = path.join(__dirname, 'bin', 'native');
const TIMEOUT_MS = 60000; // 60s total for download

function getTarget() {
  const map = {
    'win32-x64':   'windows_amd64',
    'darwin-x64':  'darwin_amd64',
    'darwin-arm64': 'darwin_arm64',
    'linux-x64':   'linux_amd64',
    'linux-arm64':  'linux_arm64',
  };
  return map[`${process.platform}-${os.arch()}`] || null;
}

// Simple HTTPS fetch with timeout.
function fetch(url) {
  return new Promise((resolve, reject) => {
    const req = https.get(
      url,
      {
        headers: { 'User-Agent': `multiai-npm/${VERSION}` },
        timeout: 30000
      },
      (res) => {
        if (res.statusCode >= 301 && res.statusCode <= 308 && res.headers.location) {
          res.resume();
          resolve(fetch(res.headers.location));
          return;
        }
        if (res.statusCode !== 200) {
          res.resume();
          reject(new Error(`HTTP ${res.statusCode} for ${url}`));
          return;
        }
        const chunks = [];
        res.on('data', c => chunks.push(c));
        res.on('end', () => resolve(Buffer.concat(chunks)));
        res.on('error', reject);
      }
    );
    req.on('timeout', () => { req.destroy(); reject(new Error('Request timed out')); });
    req.on('error', reject);
  });
}

function sha256(buffer) {
  return crypto.createHash('sha256').update(buffer).digest('hex');
}

function expectedChecksum(checksumsText, fileName) {
  for (const line of checksumsText.split(/\r?\n/)) {
    const m = line.trim().match(/^([0-9a-f]{64})\s+\*?(.+)$/i);
    if (m && m[2].trim() === fileName) return m[1].toLowerCase();
  }
  return null;
}

function extract(archivePath, destDir) {
  if (process.platform === 'win32') {
    execFileSync('powershell', [
      '-NoProfile', '-NonInteractive', '-Command',
      `Expand-Archive -LiteralPath '${archivePath}' -DestinationPath '${destDir}' -Force`
    ], { stdio: 'ignore' });
  } else {
    execFileSync('tar', ['xzf', archivePath, '-C', destDir], { stdio: 'ignore' });
  }
}

async function main() {
  if (process.env.MULTIAI_SKIP_DOWNLOAD) {
    console.log('multiai: MULTIAI_SKIP_DOWNLOAD set, skipping binary download.');
    return;
  }
  if (/-dev/.test(VERSION)) {
    console.log('multiai ' + VERSION + ': dev version, no GitHub release.');
    return;
  }

  const target = getTarget();
  if (!target) {
    console.error('multiai: unsupported platform ' + process.platform + '-' + os.arch());
    console.error('Download manually: https://github.com/' + REPO + '/releases');
    process.exit(1);
  }

  const ext = process.platform === 'win32' ? '.zip' : '.tar.gz';
  const archiveName = 'multiai_' + VERSION + '_' + target + ext;
  const base = 'https://github.com/' + REPO + '/releases/download/v' + VERSION;

  console.log('multiai ' + VERSION + ' — downloading ' + archiveName + '...');

  // 1. Checksums
  const checksumsText = (await fetch(base + '/checksums.txt')).toString('utf8');
  const expected = expectedChecksum(checksumsText, archiveName);
  if (!expected) throw new Error(archiveName + ' not found in checksums.txt');

  // 2. Archive
  const archive = await fetch(base + '/' + archiveName);
  const actual = sha256(archive);
  if (actual !== expected) {
    throw new Error('SHA256 mismatch for ' + archiveName + '\n  expected: ' + expected + '\n  actual: ' + actual);
  }
  console.log('multiai: SHA256 verified (' + expected.slice(0, 16) + '...)');

  // 3. Extract
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'multiai-'));
  try {
    const archivePath = path.join(tmpDir, archiveName);
    fs.writeFileSync(archivePath, archive);
    extract(archivePath, tmpDir);

    const extracted = path.join(tmpDir, BINARY_NAME);
    if (!fs.existsSync(extracted)) throw new Error(BINARY_NAME + ' not found in archive');

    fs.mkdirSync(NATIVE_DIR, { recursive: true });
    const dest = path.join(NATIVE_DIR, BINARY_NAME);
    fs.copyFileSync(extracted, dest);
    if (process.platform !== 'win32') fs.chmodSync(dest, 0o755);

    if (process.env.MULTIAI_INSTALL_DIR) {
      const extraDir = process.env.MULTIAI_INSTALL_DIR;
      fs.mkdirSync(extraDir, { recursive: true });
      const extra = path.join(extraDir, BINARY_NAME);
      fs.copyFileSync(dest, extra);
      if (process.platform !== 'win32') fs.chmodSync(extra, 0o755);
      console.log('multiai: extra copy -> ' + extra);
    }
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }

  console.log('multiai installed. Try: multiai help');
}

// Overall timeout — don't hang CI or user terminals.
const timer = setTimeout(() => { console.error('multiai: download timed out'); process.exit(1); }, TIMEOUT_MS);
main().then(() => clearTimeout(timer)).catch(err => {
  clearTimeout(timer);
  console.error('multiai install failed: ' + err.message);
  console.error('Manual download: https://github.com/' + REPO + '/releases/tag/v' + VERSION);
  process.exit(1);
});
