import { expect, test } from '@playwright/test'

// 키 하나가 새어 나가면 그 키만 끊을 수 있어야 한다. 지금까지는 그 사람
// 계정을 통째로 정지하는 수밖에 없었고, 그러면 키와 상관없는 일까지 멈춘다.
test('an administrator revokes one leaked key without suspending the person', async ({ page, request }) => {
  const stamp = Date.now()
  const owner = `keyholder.${stamp}`
  const created = await request.post('/api/v1/me/api-keys', {
    headers: { 'X-Kanpic-Actor': owner },
    data: { name: `유출된 키 ${stamp}`, scopes: ['spreadsheet.read'] },
  }).then(r => r.json())
  expect(created.key ?? created.token ?? created.id, JSON.stringify(created)).toBeTruthy()

  await page.goto('/admin?tab=keys')
  await expect(page.getByRole('heading', { name: 'API 키 현황' })).toBeVisible()
  const row = page.locator('.settings-row', { hasText: `유출된 키 ${stamp}` })
  await expect(row).toBeVisible({ timeout: 15_000 })
  await expect(row).toContainText('활성')

  page.once('dialog', dialog => dialog.accept())
  await row.getByRole('button', { name: '폐기' }).click()
  await expect(page.getByRole('status')).toContainText('폐기했습니다', { timeout: 15_000 })
  await expect(page.locator('.settings-row', { hasText: `유출된 키 ${stamp}` })).toContainText('폐기됨')

  // 그 사람 계정은 멀쩡해야 한다. 키만 끊는 것이 요점이다.
  const person = await request.get(`/api/v1/admin/users/${owner}`)
  if (person.status() === 200) expect((await person.json()).status).toBe('active')

  // 관리자 행위로 기록에 남는다.
  const csv = await request.get('/api/v1/admin/logs.csv?q=admin.action apikey.revoke')
  expect(csv.status()).toBe(200)
  expect(await csv.text()).toContain('apikey.revoke')
})
