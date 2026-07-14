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
const http = require('http');
const https = require('https');
const os = require('os');
const path = require('path');
const tls = require('tls');
const { runVersionProbe } = require('./lib/version-probe');

const pkg = require('./package.json');
const VERSION = pkg.version;
const REPO = 'lrochetta/multiai';
const BINARY_NAME = process.platform === 'win32' ? 'multiai.exe' : 'multiai';
const NATIVE_DIR = path.join(__dirname, 'bin', 'native');
const TIMEOUT_MS = 60000; // 60s total for download
const REQUEST_TIMEOUT_MS = 30000;
const EXTRACT_TIMEOUT_MS = 30000;
const BINARY_SMOKE_TIMEOUT_MS = 20000;
const MAX_REDIRECTS = 5;
const MAX_CHECKSUMS_BYTES = 1024 * 1024;
const MAX_ARCHIVE_BYTES = 100 * 1024 * 1024;
const ALLOWED_DOWNLOAD_HOSTS = new Set([
  'github.com',
  'release-assets.githubusercontent.com'
]);

// npm and the operating system may trust a local/company CA that Node's
// bundled Mozilla store does not know about. On recent Node versions, merge
// the OS trust store into the existing defaults before contacting GitHub.
// The package requires Node 24.14+, but guards keep this code defensive on
// non-standard runtimes.
function configureNetworkTrust(tlsModule = tls, httpModule = http) {
  let systemCAEnabled = false;
  if (
    typeof tlsModule.getCACertificates === 'function' &&
    typeof tlsModule.setDefaultCACertificates === 'function'
  ) {
    const defaults = tlsModule.getCACertificates('default');
    const system = tlsModule.getCACertificates('system');
    if (system.length > 0) {
      tlsModule.setDefaultCACertificates([...new Set([...defaults, ...system])]);
      systemCAEnabled = true;
    }
  }

  // Node 24.14+ can honour HTTPS_PROXY/NO_PROXY dynamically.
  if (typeof httpModule.setGlobalProxyFromEnv === 'function') {
    httpModule.setGlobalProxyFromEnv();
  }

  return systemCAEnabled;
}

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

function validateDownloadUrl(value) {
  const parsed = value instanceof URL ? value : new URL(value);
  if (parsed.protocol !== 'https:' || parsed.username || parsed.password ||
      !ALLOWED_DOWNLOAD_HOSTS.has(parsed.hostname.toLowerCase())) {
    throw new Error(`Refusing untrusted download URL: ${parsed.origin}`);
  }
  return parsed;
}

function resolveRedirectUrl(location, currentUrl) {
  return validateDownloadUrl(new URL(location, validateDownloadUrl(currentUrl)));
}

function readResponseBody(res, maxBytes, url) {
  return new Promise((resolve, reject) => {
    const declared = Number(res.headers['content-length']);
    if (Number.isFinite(declared) && declared > maxBytes) {
      res.resume();
      reject(new Error(`Download exceeds ${maxBytes} bytes for ${url}`));
      return;
    }
    const chunks = [];
    let received = 0;
    res.on('data', chunk => {
      received += chunk.length;
      if (received > maxBytes) {
        res.destroy(new Error(`Download exceeds ${maxBytes} bytes for ${url}`));
        return;
      }
      chunks.push(chunk);
    });
    res.on('end', () => resolve(Buffer.concat(chunks)));
    res.on('error', reject);
  });
}

// HTTPS fetch with allowlisted redirects, streaming size bound and timeout.
function fetch(url, maxBytes, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > MAX_REDIRECTS) {
      reject(new Error(`Too many redirects for ${url}`));
      return;
    }
    let trustedUrl;
    try { trustedUrl = validateDownloadUrl(url); } catch (err) { reject(err); return; }
    const req = https.get(
      trustedUrl,
      {
        headers: { 'User-Agent': `multiai-npm/${VERSION}` },
        timeout: REQUEST_TIMEOUT_MS
      },
      (res) => {
        if (res.statusCode >= 301 && res.statusCode <= 308 && res.headers.location) {
          res.resume();
          let redirect;
          try { redirect = resolveRedirectUrl(res.headers.location, trustedUrl); }
          catch (err) { reject(err); return; }
          resolve(fetch(redirect, maxBytes, redirects + 1));
          return;
        }
        if (res.statusCode !== 200) {
          res.resume();
          reject(new Error(`HTTP ${res.statusCode} for ${url}`));
          return;
        }
        resolve(readResponseBody(res, maxBytes, trustedUrl));
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
      'Expand-Archive -LiteralPath $env:MULTIAI_ARCHIVE -DestinationPath $env:MULTIAI_DEST -Force'
    ], {
      env: {
        ...process.env,
        MULTIAI_ARCHIVE: archivePath,
        MULTIAI_DEST: destDir
      },
      stdio: 'ignore',
      timeout: EXTRACT_TIMEOUT_MS
    });
  } else {
    execFileSync('tar', ['xzf', archivePath, '-C', destDir], {
      stdio: 'ignore',
      timeout: EXTRACT_TIMEOUT_MS
    });
  }
}

function isCertificateError(err) {
  const code = err && err.code ? String(err.code) : '';
  const message = err && err.message ? String(err.message) : '';
  return [
    'UNABLE_TO_VERIFY_LEAF_SIGNATURE',
    'SELF_SIGNED_CERT_IN_CHAIN',
    'DEPTH_ZERO_SELF_SIGNED_CERT',
    'UNABLE_TO_GET_ISSUER_CERT_LOCALLY'
  ].includes(code) || /unable to verify|self[- ]signed certificate|issuer certificate/i.test(message);
}

function isSupportedNode(version = process.versions.node) {
  const [major, minor] = String(version).split('.').map(Number);
  return major > 24 || (major === 24 && minor >= 14);
}

function verifyBinary(binaryPath, version = VERSION, exec = runVersionProbe) {
  let output;
  try {
    output = exec(binaryPath, BINARY_SMOKE_TIMEOUT_MS);
  } catch (err) {
    if (err && (err.code === 'ETIMEDOUT' || err.killed || err.status === 124)) {
      throw new Error(`native binary smoke test timed out after ${BINARY_SMOKE_TIMEOUT_MS / 1000}s`);
    }
    throw new Error(`native binary smoke test failed: ${err.message}`);
  }
  const expected = `multiai ${version}`;
  if (String(output).trim() !== expected) {
    throw new Error(`native binary reported ${String(output).trim() || '(no version)'} instead of ${expected}`);
  }
}

async function main() {
  if (!isSupportedNode()) {
    throw new Error(`Node.js 24.14+ is required (current: ${process.versions.node})`);
  }
  configureNetworkTrust();

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
  const checksumsText = (await fetch(base + '/checksums.txt', MAX_CHECKSUMS_BYTES)).toString('utf8');
  const expected = expectedChecksum(checksumsText, archiveName);
  if (!expected) throw new Error(archiveName + ' not found in checksums.txt');

  // 2. Archive
  const archive = await fetch(base + '/' + archiveName, MAX_ARCHIVE_BYTES);
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

    // Fail the install before announcing success if Windows security software
    // blocks the generated executable at process startup.
    verifyBinary(dest);

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
function run() {
  const timer = setTimeout(() => {
    console.error('multiai: download timed out');
    process.exit(1);
  }, TIMEOUT_MS);

  main().then(() => clearTimeout(timer)).catch(err => {
    clearTimeout(timer);
    console.error('multiai install failed: ' + err.message);
    if (isCertificateError(err)) {
      console.error('Node.js could not trust GitHub\'s TLS certificate.');
      console.error('Try NODE_USE_SYSTEM_CA=1 or configure NODE_EXTRA_CA_CERTS.');
    }
    console.error('Manual download: https://github.com/' + REPO + '/releases/tag/v' + VERSION);
    process.exit(1);
  });
}

if (require.main === module) {
  run();
}

module.exports = {
  configureNetworkTrust,
  expectedChecksum,
  getTarget,
  isCertificateError,
  isSupportedNode,
  readResponseBody,
  resolveRedirectUrl,
  validateDownloadUrl,
  verifyBinary
};
