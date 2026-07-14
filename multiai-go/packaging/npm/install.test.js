'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');

const {
  configureNetworkTrust,
  expectedChecksum,
  isCertificateError,
  isSupportedNode,
  verifyBinary
} = require('./install');

test('configureNetworkTrust merges default and system CAs and enables proxy support', () => {
  const calls = [];
  let configured;
  let proxyEnabled = false;
  const fakeTls = {
    getCACertificates(type) {
      calls.push(type);
      return type === 'default' ? ['bundled', 'duplicate'] : ['system', 'duplicate'];
    },
    setDefaultCACertificates(certificates) {
      configured = certificates;
    }
  };
  const fakeHttp = {
    setGlobalProxyFromEnv() {
      proxyEnabled = true;
    }
  };

  assert.equal(configureNetworkTrust(fakeTls, fakeHttp), true);
  assert.deepEqual(calls, ['default', 'system']);
  assert.deepEqual(configured, ['bundled', 'duplicate', 'system']);
  assert.equal(proxyEnabled, true);
});

test('configureNetworkTrust is defensive when optional APIs are unavailable', () => {
  assert.equal(configureNetworkTrust({}, {}), false);
});

test('expectedChecksum accepts GoReleaser checksum lines', () => {
  const checksum = 'a'.repeat(64);
  const text = `${checksum}  multiai_0.6.6_windows_amd64.zip\n`;
  assert.equal(expectedChecksum(text, 'multiai_0.6.6_windows_amd64.zip'), checksum);
  assert.equal(expectedChecksum(text, 'other.zip'), null);
});

test('isCertificateError recognises Node TLS failures', () => {
  assert.equal(isCertificateError({ code: 'UNABLE_TO_VERIFY_LEAF_SIGNATURE' }), true);
  assert.equal(isCertificateError(new Error('unable to verify the first certificate')), true);
  assert.equal(isCertificateError(new Error('HTTP 404')), false);
});

test('Node 24.14 is the minimum supported bootstrap runtime', () => {
  assert.equal(isSupportedNode('24.14.0'), true);
  assert.equal(isSupportedNode('24.13.9'), false);
  assert.equal(isSupportedNode('22.21.0'), false);
  assert.equal(isSupportedNode('25.0.0'), true);
});

test('downloaded native binary must report the package version', () => {
  const calls = [];
  const fakeExec = (file, timeout) => {
    calls.push({ file, timeout });
    return 'multiai 0.6.8\n';
  };
  verifyBinary('multiai.exe', '0.6.8', fakeExec);
  assert.equal(calls.length, 1);
  assert.equal(calls[0].timeout, 20000);
});

test('native binary smoke timeout fails the install explicitly', () => {
  const timeout = Object.assign(new Error('timed out'), { code: 'ETIMEDOUT' });
  assert.throws(
    () => verifyBinary('multiai.exe', '0.6.8', () => { throw timeout; }),
    /smoke test timed out after 20s/
  );
});

test('Windows controller exit 124 is reported as a smoke timeout', () => {
  const timeout = Object.assign(new Error('controller timeout'), { status: 124 });
  assert.throws(
    () => verifyBinary('multiai.exe', '0.6.9', () => { throw timeout; }),
    /smoke test timed out after 20s/
  );
});

test('native binary smoke rejects a mismatched version', () => {
  assert.throws(
    () => verifyBinary('multiai.exe', '0.6.8', () => 'multiai 0.6.7\n'),
    /instead of multiai 0\.6\.8/
  );
});

test('native binary smoke wraps process startup failures', () => {
  assert.throws(
    () => verifyBinary('multiai.exe', '0.6.8', () => { throw new Error('access denied'); }),
    /native binary smoke test failed: access denied/
  );
});
