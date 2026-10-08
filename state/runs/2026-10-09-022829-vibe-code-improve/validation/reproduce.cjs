const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const esbuild = require('/mnt/c/Users/USER/projects/vibe-code/node_modules/esbuild');
const run = __dirname;
const root = '/home/hkjang/.cache/auto-improve-wt/vibe-code';
esbuild.buildSync({ entryPoints: [path.join(root, 'src/features/update-check.ts')], bundle: true, platform: 'node', format: 'cjs', outfile: path.join(run, 'update-check.cjs'), alias: { vscode: path.join(root, 'tests/unit/vscode-stub.ts') } });
const {latestVsixVersion} = require('./update-check.cjs');
const dir = fs.mkdtempSync(path.join(run, 'vsix-fixture-'));
try {
 fs.writeFileSync(path.join(dir, 'vibe-code-1.10.1.vsix'), 'fixture');
 fs.mkdirSync(path.join(dir, 'vibe-code-99.0.0.vsix'));
 const actual = latestVsixVersion(dir);
 console.log(JSON.stringify({ node: process.version, mixed: { expected: '1.10.1', actual }, selectedIsDirectory: fs.statSync(path.join(dir, `vibe-code-${actual}.vsix`)).isDirectory() }));
 fs.unlinkSync(path.join(dir, 'vibe-code-1.10.1.vsix'));
 console.log(JSON.stringify({ directoryOnly: { expected: 'undefined', actual: latestVsixVersion(dir) } }));
 assert.equal(actual, '1.10.1', 'a VSIX-named directory must not hide the real package');
} finally { fs.rmSync(dir, { recursive: true, force: true }); }
