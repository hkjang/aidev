import { expect, test } from '@playwright/test'

// 무엇이 잠들어 있는지 알아야 정리할지 남길지 정할 수 있다. 개요에서 세어
// 보여 주고, 거기서 목록으로 넘어가는 데까지 확인한다.
test('the admin console counts what has gone quiet and lists it', async ({ page, request }) => {
  const stamp = Date.now()
  const sleeper = `never.${stamp}`
  await request.post('/api/v1/admin/users', { data: { user_id: sleeper, display_name: '한번도 안 옴' } })

  await page.goto('/admin?tab=overview')
  await expect(page.getByRole('heading', { name: '개요' })).toBeVisible()
  const row = page.locator('.attention-row', { hasText: '1년 이상 손대지 않은 워크북' })
  await expect(row).toBeVisible({ timeout: 15_000 })
  await row.getByRole('button', { name: '목록 보기' }).click()
  await expect(page.getByRole('heading', { name: '워크북 거버넌스' })).toBeVisible()

  // 한 번도 들어온 적 없는 계정을 잠든 것으로 센다. 미리 등록해 두고
  // 아무도 쓰지 않은 계정이 그대로 남는 일이 흔하다.
  await page.goto('/admin?tab=users')
  await expect(page.getByRole('heading', { name: '사용자 및 역할' })).toBeVisible()
  await expect(page.locator('.user-row', { hasText: sleeper })).toBeVisible({ timeout: 15_000 })
  await page.getByLabel('잠든 계정만').check()
  await expect(page.locator('.user-row', { hasText: sleeper })).toBeVisible()

  const overview = await request.get('/api/v1/admin/overview').then(r => r.json())
  expect(overview.overview.dormant_users).toBeGreaterThan(0)
})
