import { expect, test } from '@playwright/test'

// 1월·2월 시트가 부서 × 항목으로 되어 있으면 부서를 맞춰 더해야 한다. 시트마다
// 부서 차례가 다르므로 자리로 맞추면 1월 영업1팀에 2월 영업2팀이 더해진다.
test('two sheets are combined by label, not by position', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  let workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const january = workbook.sheets[0].id as string
  await page.request.patch(`/api/v1/sheets/${january}`, { data: { name: '1월' } })
  const february = await page.request.post(`/api/v1/workbooks/${workbookId}/sheets`, { data: { name: '2월' } }).then(r => r.json())

  const grid = (rows:Array<Array<string|number>>) => rows.flatMap((row,rowIndex)=>
    row.map((value,columnIndex)=>({row:rowIndex+1,column:columnIndex+1,value})))
  workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const a = await page.request.patch(`/api/v1/sheets/${january}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'cs-jan',
    cells: grid([['부서','매출'],['영업1팀',100],['영업2팀',200]]) }})
  expect(a.status(), await a.text()).toBeLessThan(300)
  workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  // 일부러 차례를 바꾼다.
  const b = await page.request.patch(`/api/v1/sheets/${february.id}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'cs-feb',
    cells: grid([['부서','매출'],['영업2팀',20],['영업1팀',10]]) }})
  expect(b.status(), await b.text()).toBeLessThan(300)

  await page.reload()
  await expect(page.locator('canvas.grid-canvas')).toBeVisible()
  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '여러 시트 합치기…' }).click()
  const dialog = page.getByRole('dialog', { name: '여러 시트 합치기' })
  const table = dialog.locator('.consolidate-preview')
  await expect(table).toBeVisible({ timeout: 15_000 })
  // 영업1팀 100+10=110, 영업2팀 200+20=220. 자리로 맞췄다면 120과 210이 된다.
  await expect(table.locator('tbody tr').first()).toContainText('110')
  await expect(table.locator('tbody tr').nth(1)).toContainText('220')

  await dialog.getByRole('button', { name: '결과를 새 시트로' }).click()
  await expect.poll(async()=>{
    const latest = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
    const made = (latest.sheets as Array<{id:string;name:string}>).find(sheet=>sheet.name.startsWith('통합'))
    if(!made)return ''
    const cells = await page.request.get(`/api/v1/sheets/${made.id}/ranges/A1:B3`).then(r => r.json())
    return (cells.items as Array<{row:number;column:number;value?:unknown}>)
      .filter(item=>item.column===2&&item.row>1).map(item=>item.value).join(',')
  },{timeout:15_000}).toBe('110,220')
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
