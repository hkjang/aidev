import { expect, test } from '@playwright/test'

// "링크 공개 47개" 를 보고 마흔일곱 번 누르는 사람은 없다. 몇 개는 남고,
// 남은 것은 아무도 세지 않는다. 한 번에 닫고 몇 개가 남았는지 말하는 데까지
// 확인한다.
test('an administrator closes every open link at once', async ({ page, request }) => {
  const stamp = Date.now()
  const opened:string[] = []
  for (const suffix of ['가', '나']) {
    const book = await request.post('/api/v1/workbooks', { data: { title: `공개 ${suffix} ${stamp}` } }).then(r => r.json())
    await request.patch(`/api/v1/workbooks/${book.id}/sharing`, { data: { link_access: 'anyone' } })
    opened.push(book.id)
  }
  // 조직 공개는 다른 거르개이므로 건드리지 않아야 한다.
  const org = await request.post('/api/v1/workbooks', { data: { title: `조직 ${stamp}` } }).then(r => r.json())
  await request.patch(`/api/v1/workbooks/${org.id}/sharing`, { data: { link_access: 'organization' } })

  await page.goto('/admin?tab=workbooks')
  await expect(page.getByRole('heading', { name: '워크북 거버넌스' })).toBeVisible()
  await page.getByRole('button', { name: '링크 공개', exact: true }).click()

  page.once('dialog', dialog => dialog.accept())
  await page.getByRole('button', { name: '이 목록 전체 공개 해제' }).click()
  await expect(page.getByRole('status')).toContainText('제한했습니다', { timeout: 20_000 })

  for (const id of opened) {
    const sharing = await request.get(`/api/v1/workbooks/${id}/sharing`).then(r => r.json())
    expect(sharing.sharing.link_access).toBe('restricted')
  }
  // 고른 거르개만 닫는다.
  const untouched = await request.get(`/api/v1/workbooks/${org.id}/sharing`).then(r => r.json())
  expect(untouched.sharing.link_access).toBe('organization')

  // 관리자 행위로 남는다.
  const csv = await request.get('/api/v1/admin/logs.csv?q=admin.action workbooks.restrict_links')
  expect(await csv.text()).toContain('workbooks.restrict_links')
})
