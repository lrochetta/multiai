'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');
const { runVersionProbe, trustedPowerShell } = require('./version-probe');

test('Windows probe delegates to the external bounded controller', () => {
  let call;
  const output = runVersionProbe('C:\\Program Files\\multiai.exe', 20000, {
    platform: 'win32', env: { SystemRoot: 'C:\\Windows' },
    exec(file, args, options) { call = { file, args, options }; return 'multiai 0.6.9\n'; }
  });
  assert.equal(output, 'multiai 0.6.9\n');
  assert.equal(call.file, 'C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe');
  assert.ok(call.args.includes('20000'));
  assert.equal(call.options.timeout, 30000);
  assert.equal(call.options.env.MULTIAI_SKIP_UPDATE, '1');
});

test('non-Windows probe directly bounds the binary', () => {
  let call;
  runVersionProbe('/tmp/multiai', 5000, {
    platform: 'linux', env: {}, exec(file, args, options) { call = { file, args, options }; return ''; }
  });
  assert.deepEqual(call.args, ['--version']);
  assert.equal(call.options.timeout, 5000);
});

test('trusted PowerShell rejects a missing system root', () => {
  assert.throws(() => trustedPowerShell({}), /SystemRoot is unavailable/);
});
