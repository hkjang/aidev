import { expect, test } from '@playwright/test'

// 표가 A1 에서 시작한다고 여기면, C3 에서 시작하는 시트는 첫 행이 비어 있어
// 항목 이름을 하나도 못 찾는다. 고른 목록에는 남아 있는 채로 아무것도 보태지
// 않으므로, 사람은 합계가 왜 작은지 알 길이 없다.
test('a table that does not start at A1 is still consolidated', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  let workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const first = workbook.sheets[0].id as string
  await page.request.patch(`/api/v1/sheets/${first}`, { data: { name: '1월' } })
  const second = await page.request.post(`/api/v1/workbooks/${workbookId}/sheets`, { data: { name: '2월' } }).then(r => r.json())

  // 두 시트 모두 C3 에서 시작한다.
  const offset = (amount:number) => ([
    {row:3,column:3,value:'부서'},{row:3,column:4,value:'매출'},
    {row:4,column:3,value:'영업1팀'},{row:4,column:4,value:amount},
  ])
  workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const a = await page.request.patch(`/api/v1/sheets/${first}/cells:batch`, {
    data: { base_version: workbook.version, idempotency_key: 'co-a', cells: offset(100) } })
  expect(a.status(), await a.text()).toBeLessThan(300)
  workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const b = await page.request.patch(`/api/v1/sheets/${second.id}/cells:batch`, {
    data: { base_version: workbook.version, idempotency_key: 'co-b', cells: offset(10) } })
  expect(b.status(), await b.text()).toBeLessThan(300)

  await page.reload()
  await expect(page.locator('canvas.grid-canvas')).toBeVisible()
  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '여러 시트 합치기…' }).click()
  const dialog = page.getByRole('dialog', { name: '여러 시트 합치기' })
  const table = dialog.locator('.consolidate-preview')
  await expect(table).toBeVisible({ timeout: 15_000 })
  // 예전 코드였다면 표가 아예 그려지지 않는다.
  await expect(table.locator('tbody tr').first()).toContainText('영업1팀')
  await expect(table.locator('tbody tr').first()).toContainText('110')
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
