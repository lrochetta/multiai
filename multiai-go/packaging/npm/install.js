#!/usr/bin/env node
// multiai npm installer -- downloads the native binary for the current platform

const { execSync } = require('child_process');
const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');

const VERSION = '0.5.0';
const BINARY_NAME = process.platform === 'win32' ? 'multiai.exe' : 'multiai';

function getPlatform() {
    const platform = os.platform();
    const arch = os.arch();
    const map = {
        'win32-x64':   'windows-amd64',
        'linux-x64':   'linux-amd64',
        'linux-arm64': 'linux-arm64',
        'darwin-x64':  'darwin-amd64',
        'darwin-arm64':'darwin-arm64',
    };
    return map[`${platform}-${arch}`] || null;
}

function download(url, dest) {
    return new Promise((resolve, reject) => {
        const file = fs.createWriteStream(dest);
        https.get(url, (response) => {
            if (response.statusCode === 302) {
                https.get(response.headers.location, (res) => {
                    res.pipe(file);
                    file.on('finish', () => { file.close(); resolve(); });
                }).on('error', reject);
            } else {
                response.pipe(file);
                file.on('finish', () => { file.close(); resolve(); });
            }
        }).on('error', reject);
    });
}

async function main() {
    const platform = getPlatform();
    if (!platform) {
        console.error(`multiai: unsupported platform ${os.platform()}-${os.arch()}`);
        console.error('Install Go and run: go install github.com/lrochetta/multiai@latest');
        process.exit(1);
    }

    const installDir = process.env.MULTIAI_INSTALL_DIR ||
        path.join(os.homedir(), '.local', 'bin');

    if (!fs.existsSync(installDir)) {
        fs.mkdirSync(installDir, { recursive: true });
    }

    const dest = path.join(installDir, BINARY_NAME);
    const archiveExt = process.platform === 'win32' ? '.zip' : '.tar.gz';
    const archiveName = `multiai_${VERSION}_${platform}${archiveExt}`;
    const url = `https://github.com/lrochetta/multiai/releases/download/v${VERSION}/${archiveName}`;

    console.log(`multiai ${VERSION} -- downloading for ${platform}...`);
    console.log(`  -> ${dest}`);

    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'multiai-'));
    const archivePath = path.join(tmpDir, archiveName);

    try {
        await download(url, archivePath);

        // Extract
        if (process.platform === 'win32') {
            execSync(`powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${tmpDir}' -Force"`, { stdio: 'ignore' });
        } else {
            execSync(`tar xzf "${archivePath}" -C "${tmpDir}"`, { stdio: 'ignore' });
        }

        // Find and copy binary
        const files = fs.readdirSync(tmpDir);
        const binary = files.find(f => f === BINARY_NAME);
        if (binary) {
            fs.copyFileSync(path.join(tmpDir, binary), dest);
            if (process.platform !== 'win32') {
                fs.chmodSync(dest, 0o755);
            }
        }
    } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
    }

    console.log('multiai installed successfully!');
    console.log(`Run: ${dest} help`);
}

main().catch(err => {
    console.error('multiai install failed:', err.message);
    process.exit(1);
});
