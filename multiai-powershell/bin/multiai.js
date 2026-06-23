#!/usr/bin/env node
'use strict';

const { spawnSync } = require('child_process');
const path = require('path');
const fs   = require('fs');
const os   = require('os');

const PKG_DIR   = path.join(__dirname, '..');
const isWindows = os.platform() === 'win32';

// ── Helpers ───────────────────────────────────────────────────────────────────

function run(cmd, cmdArgs) {
  const result = spawnSync(cmd, cmdArgs, { stdio: 'inherit', shell: false });
  return result.status != null ? result.status : 1;
}

function hasBin(name) {
  const check = isWindows ? 'where' : 'which';
  const r = spawnSync(check, [name], { stdio: 'pipe' });
  return r.status === 0;
}

function showHelp() {
  console.log('');
  console.log('  AI Code CLI Router — multiai');
  console.log('  ---------------------------------------------------------');
  console.log('  npx multiai install              Install (defaults)');
  console.log('  npx multiai install --yes        Install silently');
  console.log('  npx multiai install <dir>        Install to custom dir');
  console.log('');
  console.log('  Supports : Claude Code · Codex CLI · OpenCode');
  console.log('  Platforms: Windows · macOS · Linux (Ubuntu)');
  console.log('');
  console.log('  Requires: PowerShell 5.1+ (Windows) or pwsh/PowerShell Core (macOS/Linux)');
  console.log('  macOS   : brew install powershell/tap/powershell');
  console.log('  Ubuntu  : sudo apt-get install -y powershell');
  console.log('');
  console.log('  Author  : Laurent Rochetta — https://follow.ovh/bio/laurent');
  console.log('  Blog    : https://rochetta.fr');
  console.log('');
}

// ── Main ──────────────────────────────────────────────────────────────────────

const args    = process.argv.slice(2);
const command = args[0] || '';
const rest    = args.slice(1);

if (!command || command === 'help' || command === '--help' || command === '-h') {
  showHelp();
  process.exit(0);
}

if (command !== 'install') {
  console.error(`Commande inconnue : ${command}`);
  showHelp();
  process.exit(1);
}

// Parse flags
const customDir = rest.find(a => !a.startsWith('-')) || null;

if (isWindows) {
  const ps1    = path.join(PKG_DIR, 'install.ps1');
  const psArgs = ['-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', ps1];
  if (customDir) psArgs.push('-InstallDir', customDir);
  process.exit(run('powershell', psArgs));
} else {
  // macOS / Linux — prefer pwsh, fallback to bash install.sh
  if (hasBin('pwsh')) {
    const ps1    = path.join(PKG_DIR, 'install.ps1');
    const psArgs = ['-NoProfile', '-File', ps1];
    if (customDir) psArgs.push('-InstallDir', customDir);
    process.exit(run('pwsh', psArgs));
  } else {
    const sh = path.join(PKG_DIR, 'install.sh');
    try { fs.chmodSync(sh, 0o755); } catch {}

    // Normalise CRLF->LF au cas ou le fichier vient d'etre cree sous Windows
    try {
      const raw = fs.readFileSync(sh, 'utf8');
      if (raw.includes('\r')) {
        fs.writeFileSync(sh, raw.replace(/\r\n/g, '\n').replace(/\r/g, '\n'), 'utf8');
      }
    } catch {}

    const shArgs = customDir ? [sh, customDir] : [sh];
    process.exit(run('bash', shArgs));
  }
}
