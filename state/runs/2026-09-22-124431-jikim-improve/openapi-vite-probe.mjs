import assert from 'node:assert/strict';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
const [root, target] = process.argv.slice(2);
const { createServer, loadConfigFromFile } = await import(pathToFileURL(join(root, 'node_modules/vite/dist/node/index.js')));
const loaded = await loadConfigFromFile({ command: 'serve', mode: 'development' }, join(root, 'vite.config.ts'), root, undefined, undefined, 'runner');
const cacheDir = await mkdtemp(join(tmpdir(), 'jikim-go-vite-'));
let vite;
try {
  const proxy = Object.fromEntries(Object.entries(loaded.config.server.proxy).map(([path, options]) => [path, typeof options === 'string' ? target : { ...options, target }]));
  vite = await createServer({ ...loaded.config, configFile: false, root, cacheDir, optimizeDeps: { noDiscovery: true, include: [] }, server: { ...loaded.config.server, host: '127.0.0.1', port: 0, proxy, open: false } });
  await vite.listen();
  const origin = `http://127.0.0.1:${vite.httpServer.address().port}`;
  for (const accept of ['text/html', 'application/json']) {
    const direct = await fetch(target + '/api/openapi.json', { headers: { Accept: accept } });
    const expected = await direct.text();
    const response = await fetch(origin + '/api/openapi.json', { headers: { Accept: accept } });
    const body = await response.text();
    assert.equal(response.status, 200);
    assert.match(response.headers.get('content-type'), /application\/json/);
    assert.equal(body, expected);
    assert.equal(JSON.parse(body).openapi, '3.1.0');
    console.log(JSON.stringify({ accept, status: response.status, contentType: response.headers.get('content-type'), openapi: JSON.parse(body).openapi, identicalToGo: body === expected }));
  }
} finally {
  await vite?.close();
  await rm(cacheDir, { recursive: true, force: true });
}
