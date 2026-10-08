import { expect, test } from '@playwright/test'

// 부서 관리자는 자기 부서와 그 아래 부서의 구성원 계정만 다룬다. 열어 준
// 것과 열지 않은 것이 실제로 그런지 통째로 확인한다.
test('a department manager sees only their members and cannot move data', async ({ page, request }) => {
  const stamp = Date.now()
  const lead = `lead.${stamp}`, member = `team.${stamp}`, outsider = `other.${stamp}`
  for (const id of [lead, member, outsider]) {
    await request.post('/api/v1/admin/users', { data: { user_id: id, display_name: id } })
  }
  const parent = await request.post('/api/v1/departments', { data: { name: `영업본부 ${stamp}` } }).then(r => r.json())
  const child = await request.post('/api/v1/departments', { data: { name: `영업1팀 ${stamp}`, parent_id: parent.id } }).then(r => r.json())
  const other = await request.post('/api/v1/departments', { data: { name: `관리본부 ${stamp}` } }).then(r => r.json())
  await request.post(`/api/v1/departments/${child.id}/members`, { data: { user_ids: [member] } })
  await request.post(`/api/v1/departments/${other.id}/members`, { data: { user_ids: [outsider] } })

  // 전역 관리자가 화면에서 부서 관리자를 지정한다.
  await page.goto('/admin?tab=departments')
  await expect(page.getByRole('heading', { name: /부서/ }).first()).toBeVisible()
  await page.locator('.department-row', { hasText: `영업본부 ${stamp}` }).first().click()
  await page.getByLabel('지정할 부서 관리자').fill(lead)
  await page.getByRole('button', { name: '지정' }).click()
  await expect(page.getByRole('status')).toContainText('부서 관리자를 지정했습니다', { timeout: 15_000 })

  const asLead = { 'X-Kanpic-Actor': lead }
  // 맡은 구성원만 보인다. 위 부서를 맡았어도 아래 부서 구성원이 들어온다.
  const listed = await request.get('/api/v1/admin/users', { headers: asLead }).then(r => r.json())
  expect((listed.items as Array<{user_id:string}>).map(item => item.user_id)).toEqual([member])
  expect(listed.scoped).toBe(true)

  // 구성원 계정은 다룰 수 있다.
  const suspended = await request.patch(`/api/v1/admin/users/${member}`, { headers: asLead, data: { status: 'suspended' } })
  expect(suspended.status()).toBe(200)

  // 다른 부서 사람, 자료를 옮기는 일, 조직 전체 보기는 열지 않는다.
  for (const call of [
    request.get(`/api/v1/admin/users/${outsider}`, { headers: asLead }),
    request.get(`/api/v1/admin/users/${member}/workbooks`, { headers: asLead }),
    request.post(`/api/v1/admin/users/${member}/workbooks:transfer`, { headers: asLead, data: { new_owner_id: lead } }),
    request.get('/api/v1/admin/overview', { headers: asLead }),
    request.get('/api/v1/admin/logs', { headers: asLead }),
    request.post('/api/v1/admin/users', { headers: asLead, data: { user_id: `sneaky.${stamp}` } }),
  ]) {
    expect((await call).status()).toBe(403)
  }
})
