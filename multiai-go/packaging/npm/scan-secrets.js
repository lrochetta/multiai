#!/usr/bin/env node
// prepublishOnly guard: refuse to publish if a real API key is found in any
// .env profile template (npm package dir, embedded Go templates, local
// configs). Templates must only contain placeholders (PASTE_YOUR_KEY_HERE).
//
// Mirrors the Go test TestEmbeddedProfilesContainNoRealSecrets: the
// credential-store sentinel (__MULTIAI_CREDSTORE__) is also rejected, because
// a template containing the sentinel would break first-run resolution.

'use strict';

const fs = require('fs');
const path = require('path');

// Publish preflight: refuse to publish a -dev version. Only enforced when
// npm runs this as the prepublishOnly hook, so the scanner stays usable
// standalone during development.
if (process.env.npm_lifecycle_event === 'prepublishOnly') {
  const version = require('./package.json').version;
  if (/-dev(\.|$)/.test(version)) {
    console.error(
      `SECURITY: refusing to publish dev version ${version}. ` +
        'Set a release version (and tag it) first.'
    );
    process.exit(1);
  }
}

const SENTINEL = '__MULTIAI_CREDSTORE__';

// Directories scanned for .env files (missing dirs are skipped).
const SCAN_DIRS = [
  __dirname, // the npm package itself
  path.join(__dirname, '..', '..', 'internal', 'assets', 'profiles'),
  path.join(__dirname, '..', '..', 'configs', 'profiles'),
];

// A key with one of these name fragments must hold a placeholder, never a
// real value.
const SECRET_KEY_RE =
  /(API_KEY|AUTH_TOKEN|_TOKEN$|_SECRET$|PASSWORD|ACCESS_KEY|SESSION_TOKEN|CLIENT_SECRET)/;

// Values that look like live credentials, whatever the key name.
const LIVE_VALUE_RE =
  /^(sk-ant-|sk-proj-|sk-or-v1-|sk-[0-9a-f]{32}|ghp_[A-Za-z0-9]{20,}|github_pat_|xox[baprs]-|AKIA[0-9A-Z]{16}|ya29\.|gsk_[A-Za-z0-9]{20,}|glpat-)/;

function isPlaceholder(value) {
  return (
    value === '' ||
    value === 'TODO' ||
    /^(PASTE_|YOUR_|REPLACE|CHANGE_ME|sk-xxxx|xxxx|<|%|\$\{)/i.test(value)
  );
}

function scanFile(filePath) {
  const findings = [];
  const lines = fs.readFileSync(filePath, 'utf8').split(/\r?\n/);
  lines.forEach((line, i) => {
    const trimmed = line.trim();
    if (trimmed === '' || trimmed.startsWith('#')) return;
    const idx = trimmed.indexOf('=');
    if (idx < 1) return;
    const key = trimmed.slice(0, idx).trim();
    let value = trimmed.slice(idx + 1).trim();
    value = value.replace(/^["']|["']$/g, '');

    if (value === SENTINEL) {
      findings.push(`${filePath}:${i + 1}: ${key} contains the credential-store sentinel`);
      return;
    }
    if (isPlaceholder(value)) return;
    if (LIVE_VALUE_RE.test(value)) {
      findings.push(`${filePath}:${i + 1}: ${key} looks like a LIVE credential`);
      return;
    }
    if (SECRET_KEY_RE.test(key) && value.length >= 20) {
      findings.push(`${filePath}:${i + 1}: ${key} holds a non-placeholder value (${value.length} chars)`);
    }
  });
  return findings;
}

function collectEnvFiles(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs
    .readdirSync(dir)
    .filter((f) => f.endsWith('.env'))
    .map((f) => path.join(dir, f));
}

let files = [];
for (const dir of SCAN_DIRS) {
  files = files.concat(collectEnvFiles(dir));
}

let findings = [];
for (const file of files) {
  findings = findings.concat(scanFile(file));
}

if (findings.length > 0) {
  console.error('SECURITY: potential real secrets detected, publish aborted:');
  for (const f of findings) console.error('  ' + f);
  process.exit(1);
}

console.log(`OK: ${files.length} .env file(s) scanned, no real key detected.`);
