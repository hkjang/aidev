import { chromium } from 'playwright';
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import path from 'node:path';

// This creates one real full-service backup in a dedicated path, never restores
// or deletes data. Global policy is restored in finally; use only a test service.
if (process.env.MADI_BACKUP_LAYOUT_ACK !== 'disposable-service')
  throw Error('MADI_BACKUP_LAYOUT_ACK=disposable-service is required');
const base = process.env.MADI_BASE_URL || 'http://127.0.0.1:8080';
assert.ok(['127.0.0.1', 'localhost'].includes(new URL(base).hostname), 'A local disposable service is required');
const output = path.resolve(process.env.MADI_SCREENSHOT_DIR || 'test-results/backup-layout');
await mkdir(output, { recursive: true });
const browser = await chromium.launch();
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'ko-KR' });
const page = await context.newPage();
const errors = [];
page.on('pageerror', error => errors.push(error.message));
let original;
async function api(endpoint, method = 'GET', data) {
  const response = await context.request.fetch(base + '/api/v1' + endpoint, {
    method, data, headers: { 'X-Madi-Request': '1' }, timeout: 120_000,
  });
  assert.ok(response.ok(), `${method} ${endpoint}: ${response.status()} ${await response.text()}`);
  return response.json();
}
try {
  await api('/auth/login', 'POST', {
    email: process.env.MADI_TEST_EMAIL || 'admin@example.test',
    password: process.env.MADI_TEST_PASSWORD || 'Browser-Test-Password-2026!',
  });
  original = await api('/admin/backups/policy');
  const jobs = await api('/admin/jobs/settings');
  assert.equal(jobs.paused, false, 'The test queue must already be unpaused; this test never activates unrelated queued work');
  const localPath = '/tmp/madi-backup-layout-' + Date.now();
  await api('/admin/backups/policy', 'PUT', {
    enabled: false, provider_id: '', local_path: localPath,
    interval_minutes: 1440, retention_count: 1000,
  });
  const queued = await api('/admin/backups/run', 'POST', {});
  let job;
  const deadline = Date.now() + 180_000;
  while (Date.now() < deadline) {
    job = (await api('/jobs/' + queued.job_id)).job;
    if (['succeeded', 'failed', 'cancelled'].includes(job.status)) break;
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  assert.equal(job?.status, 'succeeded', 'A real durable backup job must complete before the layout check');
  const artifact = (await api('/admin/backups')).find(row => row.job_id === queued.job_id);
  assert.ok(artifact, 'The real generated backup must be listed');
  const download = await context.request.get(base + '/api/v1/admin/backups/' + artifact.id + '/download', { timeout: 120_000 });
  assert.ok(download.ok(), 'The listed backup is downloadable');
  const archive = await download.body();
  assert.equal(archive.length, artifact.size);
  assert.equal(createHash('sha256').update(archive).digest('hex'), artifact.checksum_sha256);
  await page.goto(base + '/admin/backup-schedule');
  const table = page.locator('.backup-artifact-table');
  const link = table.getByRole('link', { name: '다운로드', exact: true }).first();
  await link.waitFor();
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({ path: path.join(output, 'backup-artifacts.png'), fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await table.scrollIntoViewIfNeeded();
  assert.equal(await link.evaluate(element => getComputedStyle(element).whiteSpace), 'nowrap');
  const box = await link.boundingBox();
  assert.ok(box.height >= 44 && box.width >= 90, 'The real download action must remain a readable touch target');
  assert.ok(await table.evaluate(element => element.scrollWidth > element.clientWidth), 'The table must scroll internally');
  await table.evaluate(element => { element.scrollLeft = element.scrollWidth; });
  assert.ok(await table.evaluate(element => element.scrollLeft > 0));
  assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), 'The viewport must not overflow');
  await page.screenshot({ path: path.join(output, 'backup-artifacts-mobile.png'), fullPage: false });
  assert.deepEqual(errors, []);
  console.log(JSON.stringify({ ok: true, job_id: queued.job_id, artifact_id: artifact.id, archive_bytes: archive.length, screenshots: output }));
} finally {
  try {
    if (original) await api('/admin/backups/policy', 'PUT', {
      enabled: original.enabled, provider_id: original.provider_id || '', local_path: original.local_path,
      interval_minutes: original.interval_minutes, retention_count: original.retention_count,
    });
  } finally {
    await browser.close();
  }
}
