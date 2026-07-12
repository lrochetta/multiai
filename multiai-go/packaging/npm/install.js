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
//
// Dev versions (x.y.z-dev) have no GitHub release: the download is skipped
// with a notice instead of failing the install.

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
const MAX_REDIRECTS = 5;

function getTarget() {
  // Must match the goreleaser archive name_template:
  //   multiai_<version>_<os>_<arch>.tar.gz|.zip
  const map = {
    'win32-x64': 'windows_amd64',
    'darwin-x64': 'darwin_amd64',
    'darwin-arm64': 'darwin_arm64',
    'linux-x64': 'linux_amd64',
    'linux-arm64': 'linux_arm64',
  };
  return map[`${process.platform}-${os.arch()}`] || null;
}

function fetch(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > MAX_REDIRECTS) {
      reject(new Error(`too many redirects for ${url}`));
      return;
    }
    const req = https.get(
      url,
      { headers: { 'User-Agent': `multiai-npm/${VERSION}` } },
      (res) => {
        if (
          res.statusCode >= 301 &&
          res.statusCode <= 308 &&
          res.headers.location
        ) {
          res.resume();
          resolve(fetch(res.headers.location, redirects + 1));
          return;
        }
        if (res.statusCode !== 200) {
          res.resume();
          reject(new Error(`HTTP ${res.statusCode} for ${url}`));
          return;
        }
        const chunks = [];
        res.on('data', (c) => chunks.push(c));
        res.on('end', () => resolve(Buffer.concat(chunks)));
        res.on('error', reject);
      }
    );
    req.on('error', reject);
  });
}

function sha256(buffer) {
  return crypto.createHash('sha256').update(buffer).digest('hex');
}

// checksums.txt format: "<sha256-hex>  <filename>" per line.
function expectedChecksum(checksumsText, fileName) {
  for (const line of checksumsText.split(/\r?\n/)) {
    const m = line.trim().match(/^([0-9a-f]{64})\s+\*?(.+)$/i);
    if (m && m[2].trim() === fileName) {
      return m[1].toLowerCase();
    }
  }
  return null;
}

function extract(archivePath, destDir) {
  if (process.platform === 'win32') {
    execFileSync(
      'powershell',
      [
        '-NoProfile',
        '-NonInteractive',
        '-Command',
        `Expand-Archive -LiteralPath '${archivePath}' -DestinationPath '${destDir}' -Force`,
      ],
      { stdio: 'ignore' }
    );
  } else {
    execFileSync('tar', ['xzf', archivePath, '-C', destDir], {
      stdio: 'ignore',
    });
  }
}

async function main() {
  if (process.env.MULTIAI_SKIP_DOWNLOAD) {
    console.log('multiai: MULTIAI_SKIP_DOWNLOAD set, skipping binary download.');
    return;
  }
  if (/-dev(\.|$)/.test(VERSION)) {
    console.log(
      `multiai ${VERSION}: dev version, no GitHub release to download.`
    );
    console.log(
      'Build locally instead: cd multiai-go && go build ./cmd/multiai/'
    );
    return;
  }

  const target = getTarget();
  if (!target) {
    console.error(
      `multiai: unsupported platform ${process.platform}-${os.arch()}.`
    );
    console.error(
      `Download a binary manually: https://github.com/${REPO}/releases`
    );
    process.exit(1);
  }

  const ext = process.platform === 'win32' ? '.zip' : '.tar.gz';

  // Resolve the actual release version to download.  The npm package may be
  // newer than the Go binary (JS-only fixes).  Try the exact version first;
  // if the release is missing (404), fall back to the latest GitHub release.
  let releaseVersion = VERSION;
  let archiveName = `multiai_${releaseVersion}_${target}${ext}`;
  let base = `https://github.com/${REPO}/releases/download/v${releaseVersion}`;

  console.log(`multiai ${VERSION} — downloading ${archiveName}...`);

  // 1. Checksums first: no checksums, no install.
  let checksumsText;
  try {
    checksumsText = (await fetch(`${base}/checksums.txt`)).toString('utf8');
  } catch (err) {
    // Release not found for this exact npm version — fall back to latest.
    if (err.message && err.message.includes('HTTP')) {
      console.log('Release not found for v' + releaseVersion + ', fetching latest...');
      try {
        const api = `https://api.github.com/repos/${REPO}/releases/latest`;
        const latest = JSON.parse((await fetch(api)).toString('utf8'));
        releaseVersion = latest.tag_name.replace(/^v/, '');
        archiveName = `multiai_${releaseVersion}_${target}${ext}`;
        base = `https://github.com/${REPO}/releases/download/v${releaseVersion}`;
        console.log('Using v' + releaseVersion);
        checksumsText = (await fetch(`${base}/checksums.txt`)).toString('utf8');
      } catch (_) {
        throw err; // original error if fallback also fails
      }
    } else {
      throw err;
    }
  }
  const expected = expectedChecksum(checksumsText, archiveName);
  if (!expected) {
    throw new Error(`${archiveName} not found in checksums.txt`);
  }

  // 2. Archive, then verify BEFORE extracting anything.
  const archive = await fetch(`${base}/${archiveName}`);
  const actual = sha256(archive);
  if (actual !== expected) {
    throw new Error(
      `SHA256 mismatch for ${archiveName}\n  expected: ${expected}\n  actual:   ${actual}\nRefusing to install.`
    );
  }
  console.log(`multiai: SHA256 verified (${expected.slice(0, 16)}...)`);

  // 3. Extract in a temp dir, then move the binary into the package.
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'multiai-'));
  try {
    const archivePath = path.join(tmpDir, archiveName);
    fs.writeFileSync(archivePath, archive);
    extract(archivePath, tmpDir);

    const extracted = path.join(tmpDir, BINARY_NAME);
    if (!fs.existsSync(extracted)) {
      throw new Error(`${BINARY_NAME} not found in ${archiveName}`);
    }

    fs.mkdirSync(NATIVE_DIR, { recursive: true });
    const dest = path.join(NATIVE_DIR, BINARY_NAME);
    fs.copyFileSync(extracted, dest);
    if (process.platform !== 'win32') {
      fs.chmodSync(dest, 0o755);
    }

    // Optional extra copy for users who want the raw binary on their PATH.
    if (process.env.MULTIAI_INSTALL_DIR) {
      const extraDir = process.env.MULTIAI_INSTALL_DIR;
      fs.mkdirSync(extraDir, { recursive: true });
      const extra = path.join(extraDir, BINARY_NAME);
      fs.copyFileSync(dest, extra);
      if (process.platform !== 'win32') {
        fs.chmodSync(extra, 0o755);
      }
      console.log(`multiai: extra copy -> ${extra}`);
    }
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }

  console.log('multiai installed. Try: multiai help');
}

main().catch((err) => {
  console.error(`multiai install failed: ${err.message}`);
  console.error(
    `Manual download (with checksums + cosign signature): https://github.com/${REPO}/releases/tag/v${VERSION}`
  );
  process.exit(1);
});
