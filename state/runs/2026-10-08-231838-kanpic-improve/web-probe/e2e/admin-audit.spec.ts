import { expect, test } from '@playwright/test'

// 관리자 콘솔에서 한 일은 기록으로 남아야 하고, 그 기록을 감사에서 찾아
// 내보낼 수 있어야 한다. 남기는 것과 찾는 것과 내보내는 것을 한 줄로 확인한다.
test('admin actions are recorded, findable and exportable', async ({ page, request }) => {
  const stamp = Date.now()
  const target = `audited.${stamp}`
  await request.post('/api/v1/admin/users', { data: { user_id: target, display_name: '기록 대상' } })
  await request.patch(`/api/v1/admin/users/${target}`, { data: { status: 'suspended' } })

  await page.goto('/admin?tab=logs')
  await expect(page.getByRole('heading', { name: '서버 로그' })).toBeVisible()
  await page.getByLabel('관리자 행위만').check()

  // 방금 한 정지가 목록에 보여야 한다.
  await expect(page.locator('.log-row', { hasText: 'admin.action user.status' }).first())
    .toBeVisible({ timeout: 20_000 })
  await expect(page.locator('.log-row', { hasText: target }).first()).toBeVisible()

  // 거른 조건 그대로 CSV 로 내려받는다.
  const link = page.getByRole('link', { name: 'CSV 내보내기' })
  await expect(link).toHaveAttribute('href', /q=admin\.action/)
  const csv = await request.get('/api/v1/admin/logs.csv?q=admin.action')
  expect(csv.status()).toBe(200)
  const body = await csv.text()
  expect(body).toContain('admin.action user.status')
  // 누가 했는지와 전역인지 위임인지가 함께 적혀야 감사에 쓸모가 있다.
  expect(body).toContain(target)
  expect(body).toContain('""scope"":""global""')
})
