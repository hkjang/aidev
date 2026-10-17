// Run from the repository root with Node 22 --experimental-strip-types.
// Reads the unmodified production API client; fails on the current missing notification.
import { registerHooks } from 'node:module'
import { createServer } from 'node:http'
import { pathToFileURL } from 'node:url'
import { resolve } from 'node:path'
import assert from 'node:assert/strict'
const hook = registerHooks({ resolve(specifier, context, nextResolve) {
  if (specifier === './cache' && context.parentURL?.endsWith('/apps/web/src/api/client.ts')) {
    return nextResolve('./cache.ts', context)
  }
  return nextResolve(specifier, context)
} })
globalThis.window = { location: { pathname: '/admin/rewards' } }
globalThis.document = { cookie: '' }
const { api, onServiceDegraded, ApiError } = await import(pathToFileURL(resolve('apps/web/src/api/client.ts')).href)
const server = createServer((_req, res) => {
  res.writeHead(503, { 'content-type': 'application/json' })
  res.end(JSON.stringify({ error: { code: 'db_unavailable', message: '데이터베이스 점검 중' } }))
})
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
const endpoint = `http://127.0.0.1:${server.address().port}/api/v1/admin/rewards/export`
let count = 0
const unsubscribe = onServiceDegraded(() => count++)
const isExpected = e => e instanceof ApiError && e.code === 'db_unavailable' && e.status === 503
try {
  await assert.rejects(api.post(endpoint, {}), isExpected)
  console.log('ordinary post degraded notifications:', count)
  assert.equal(count, 1)
  count = 0
  await assert.rejects(api.download(endpoint, {}, 'payouts.csv'), isExpected)
  console.log('download degraded notifications:', count)
  assert.equal(count, 1, 'download must notify the existing degraded subscribers once')
} finally {
  unsubscribe()
  await new Promise(resolve => server.close(resolve))
  hook.deregister()
  delete globalThis.window
  delete globalThis.document
}
