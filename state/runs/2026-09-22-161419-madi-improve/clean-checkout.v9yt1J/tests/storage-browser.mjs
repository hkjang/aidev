import { chromium } from 'playwright';
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import path from 'node:path';

// Uses a dedicated disposable workspace. Does not change global storage or backup policy.
const base = process.env.MADI_BASE_URL || 'http://127.0.0.1:8080';
const output = path.resolve(process.env.MADI_SCREENSHOT_DIR || 'test-results/storage');
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'ko-KR', timezoneId: 'Asia/Seoul' });
const page = await context.newPage();
const errors = [];
page.on('pageerror', e => errors.push(e.message));
page.on('response', r => { if (r.status() >= 500) errors.push(`${r.status()} ${r.url()}`); });
async function api(endpoint, method = 'GET', data) {
  const response = await context.request.fetch(base + '/api/v1' + endpoint, { method, data, headers: { 'X-Madi-Request': '1' } });
  assert.ok(response.ok(), `${method} ${endpoint}: ${response.status()} ${await response.text()}`);
  return response.json();
}
async function shot(name) {
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({ path: path.join(output, name + '.png'), fullPage: true });
}
try {
  await page.goto(base + '/login');
  await api('/auth/login', 'POST', { email: process.env.MADI_TEST_EMAIL || 'admin@example.test', password: process.env.MADI_TEST_PASSWORD || 'Browser-Test-Password-2026!' });
  const stamp = Date.now();
  const workspace = await api('/workspaces', 'POST', { name: '스토리지 UI 검증 ' + stamp });
  await page.evaluate(id => localStorage.setItem('madi.workspace', id), workspace.id);
  await page.goto(base + '/admin/storage');
  await page.getByRole('heading', { name: '서비스 저장소', exact: true }).waitFor();
  await page.getByRole('button', { name: '저장소 연결', exact: true }).click();
  await page.getByLabel('연결 이름', { exact: true }).fill('격리 로컬 검증 ' + stamp);
  await page.getByLabel('저장소 유형', { exact: true }).selectOption('local');
  await page.getByLabel('사용 범위', { exact: true }).selectOption(workspace.id);
  await page.getByLabel('로컬 절대 경로', { exact: true }).fill('/tmp/madi-storage-browser-' + stamp);
  await shot('storage-local-editor');
  await page.getByRole('button', { name: '연결 저장', exact: true }).click();
  await page.getByRole('dialog').waitFor({ state: 'hidden' });
  const localCard = page.locator('article').filter({ has: page.getByRole('heading', { name: '격리 로컬 검증 ' + stamp, exact: true }) });
  await localCard.getByRole('button', { name: '연결 진단', exact: true }).click();
  await page.getByText('쓰기 · 읽기 · 체크섬 · 삭제 진단을 모두 통과했습니다.', { exact: true }).waitFor();
  await shot('admin-storage');
  const providers = await api('/storage/providers?workspace_id=' + workspace.id);
  const local = providers.find(p => p.name === '격리 로컬 검증 ' + stamp);
  assert.ok(local);
  await page.goto(base + '/app/storage');
  await page.getByRole('heading', { name: '워크스페이스 저장소', exact: true }).waitFor();
  await page.getByLabel('이 워크스페이스의 저장소', { exact: true }).selectOption(local.id);
  await page.getByRole('button', { name: '기본 저장소 저장', exact: true }).click();
  await page.getByText('새 파일의 기본 저장소를 변경했습니다. 기존 첨부파일은 원래 저장소에서 읽습니다.', { exact: true }).waitFor();
  await page.reload();
  await page.getByLabel('이 워크스페이스의 저장소', { exact: true }).waitFor();
  assert.equal(await page.getByLabel('이 워크스페이스의 저장소', { exact: true }).inputValue(), local.id);
  await shot('workspace-storage');
  await page.getByRole('button', { name: '저장소 연결', exact: true }).click();
  await page.getByLabel('연결 이름', { exact: true }).fill('미사용 MinIO 설정 ' + stamp);
  await page.getByLabel('S3 엔드포인트', { exact: true }).fill('http://127.0.0.1:9');
  await page.getByLabel('버킷', { exact: true }).fill('madi-ui-test');
  await page.getByLabel('액세스 키', { exact: true }).fill('ui-only-dummy-access');
  await page.getByLabel('비밀 키', { exact: true }).fill('ui-only-dummy-secret');
  await page.getByRole('checkbox', { name: /사내 HTTP 사용 허용/ }).check();
  await page.getByRole('checkbox', { name: /새 파일 저장 허용/ }).uncheck();
  await shot('storage-s3-editor');
  await page.getByRole('button', { name: '연결 저장', exact: true }).click();
  await page.getByRole('dialog').waitFor({ state: 'hidden' });
  const s3Card = page.locator('article').filter({ has: page.getByRole('heading', { name: '미사용 MinIO 설정 ' + stamp, exact: true }) });
  await s3Card.getByRole('button', { name: '설정 · 자격 증명 회전', exact: true }).click();
  assert.equal(await page.getByLabel('액세스 키 교체', { exact: true }).inputValue(), '');
  assert.equal(await page.getByLabel('비밀 키 교체', { exact: true }).inputValue(), '');
  await page.getByLabel('연결 이름', { exact: true }).fill('키 유지 검증 ' + stamp);
  await page.getByRole('button', { name: '연결 저장', exact: true }).click();
  await page.getByRole('dialog').waitFor({ state: 'hidden' });
  const stored = (await api('/storage/providers?workspace_id=' + workspace.id)).find(p => p.name === '키 유지 검증 ' + stamp);
  assert.equal(stored.config.access_key, '');
  assert.equal(stored.config.secret_key, '');
  assert.equal(stored.config.access_key_configured, true);
  assert.equal(stored.config.secret_key_configured, true);
  await page.goto(base + '/admin/backup-schedule');
  await page.getByRole('heading', { name: '예약 백업', exact: true }).waitFor();
  await page.getByLabel('백업 저장소', { exact: true }).waitFor();
  await shot('backup-schedule');
  await page.reload();
  await page.getByRole('heading', { name: '예약 백업', exact: true }).waitFor();
  await page.setViewportSize({ width: 390, height: 844 });
  for (const [route, heading, name] of [['/admin/storage', '서비스 저장소', 'admin-storage-mobile'], ['/app/storage', '워크스페이스 저장소', 'workspace-storage-mobile'], ['/admin/backup-schedule', '예약 백업', 'backup-schedule-mobile']]) {
    await page.goto(base + route);
    await page.getByRole('heading', { name: heading, exact: true }).waitFor();
    await shot(name);
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), `mobile overflow ${route}`);
  }
  assert.deepEqual(errors, []);
  console.log(JSON.stringify({ ok: true, workspace_id: workspace.id, screenshots: output }));
} finally {
  await browser.close();
}
