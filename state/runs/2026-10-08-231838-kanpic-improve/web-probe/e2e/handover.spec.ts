import { expect, test } from '@playwright/test'

// 퇴사자가 워크북을 마흔 개 가지고 있으면 지금까지는 마흔 번을 눌러야 했고,
// 몇 개를 빠뜨렸는지는 아무도 몰랐다. 빠뜨린 것은 정지된 계정에 묶인 채로
// 잊힌다. 한 번에 넘기고, 무엇이 넘어갔는지 세어 보여 주는 데까지 확인한다.
test('an administrator hands over everything a leaver owns, in one go', async ({ page, request }) => {
  const stamp = Date.now()
  const leaver = `gone.${stamp}`
  const receiver = `next.${stamp}`
  const titles = [`인수인계 A ${stamp}`, `인수인계 B ${stamp}`]

  for (const title of titles) {
    const created = await request.post('/api/v1/workbooks', {
      data: { title }, headers: { 'X-Kanpic-Actor': leaver } }).then(r => r.json())
    expect(created.id, JSON.stringify(created)).toBeTruthy()
  }
  await request.post('/api/v1/admin/users', { data: { user_id: receiver, display_name: '받는 사람' } })
  await request.post('/api/v1/admin/users', { data: { user_id: leaver, display_name: '떠나는 사람' } })

  await page.goto('/admin?tab=users')
  await expect(page.getByRole('heading', { name: '사용자 및 역할' })).toBeVisible()
  await page.locator('.user-row', { hasText: leaver }).first().click()
  await page.getByRole('button', { name: '가진 워크북 인수인계…' }).click()

  const dialog = page.getByRole('dialog', { name: '가진 워크북 인수인계' })
  await expect(dialog).toContainText('소유한 워크북', { timeout: 15_000 })
  // 무엇을 넘기는지 먼저 보여 준다.
  for (const title of titles) await expect(dialog).toContainText(title)

  await dialog.getByLabel('새 소유자').fill(receiver)
  await dialog.getByRole('button', { name: /개 넘기기/ }).click()

  await expect(page.getByRole('status')).toContainText('넘겼습니다', { timeout: 15_000 })

  // 서버에서도 주인이 바뀌어 있어야 한다. 화면만 바뀌는 것으로는 부족하다.
  const owned = await request.get(`/api/v1/admin/users/${leaver}/workbooks`).then(r => r.json())
  expect(owned.items).toHaveLength(0)
  const received = await request.get(`/api/v1/admin/users/${receiver}/workbooks`).then(r => r.json())
  expect((received.items as Array<{title:string}>).map(item => item.title).sort()).toEqual(titles.slice().sort())
})
