import { expect, test } from '@playwright/test'

// 팀 하나를 들이려고 스무 번을 누르는 일을 없앤다. 사람을 여럿 만드는 것은
// 되돌리기 번거로우므로, 먼저 무엇이 바뀌는지 보여 주는 데까지 확인한다.
test('an administrator previews a CSV before it creates people', async ({ page, request }) => {
  const stamp = Date.now()
  const fresh = `new.${stamp}`
  const existing = `old.${stamp}`
  await request.post('/api/v1/admin/users', { data: { user_id: existing, display_name: '옛사람' } })

  await page.goto('/admin?tab=users')
  await expect(page.getByRole('heading', { name: '사용자 및 역할' })).toBeVisible()
  await page.getByRole('button', { name: '일괄 등록…' }).click()
  const dialog = page.getByRole('dialog', { name: '사용자 일괄 등록' })

  // 한글 머리글로 적어 오는 것이 보통이다.
  await dialog.getByLabel('CSV 내용').fill(
    `사용자 ID,이름,이메일\n${fresh},새사람,${fresh}@corp.example\n${existing},옛사람(갱신),${existing}@corp.example\n,이름만`)
  await dialog.getByRole('button', { name: '미리 보기' }).click()

  await expect(dialog.locator('.user-import-counts')).toContainText('새로 만듦 1', { timeout: 15_000 })
  await expect(dialog.locator('.user-import-counts')).toContainText('갱신 1')
  await expect(dialog.locator('.user-import-counts')).toContainText('건너뜀 1')

  // 미리보기는 아무것도 만들지 않는다.
  const before = await request.get(`/api/v1/admin/users/${fresh}`)
  expect(before.status()).toBe(404)

  await dialog.getByRole('button', { name: '2명 등록' }).click()
  await expect(page.getByRole('status')).toContainText('등록하고', { timeout: 15_000 })

  const made = await request.get(`/api/v1/admin/users/${fresh}`).then(r => r.json())
  expect(made.display_name).toBe('새사람')
  const updated = await request.get(`/api/v1/admin/users/${existing}`).then(r => r.json())
  expect(updated.display_name).toBe('옛사람(갱신)')
})
