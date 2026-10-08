import { expect, test } from '@playwright/test'

// 은행에서 받은 내역과 장부를 견주는 일이 대사다. 금액이 한쪽은 "1,000"
// 글자로, 다른 쪽은 1000 숫자로 오는 것이 보통이므로 그것까지 맞춰야 한다.
test('comparing two sheets finds what is missing on each side', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  let workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const first = workbook.sheets[0].id as string

  const ledger = await page.request.post(`/api/v1/workbooks/${workbookId}/sheets`, { data: { name: '장부' } }).then(r => r.json())
  workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())

  // 왼쪽(은행): 글자로 담긴 금액. 3,000 은 장부에 없다.
  const bank = await page.request.patch(`/api/v1/sheets/${first}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'cl-bank', cells: [
      {row:1,column:1,value:'금액'},
      {row:2,column:1,value:'1,000'},{row:3,column:1,value:'2,000'},{row:4,column:1,value:'3,000'},
    ] }})
  expect(bank.status(), await bank.text()).toBeLessThan(300)
  workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  // 오른쪽(장부): 숫자. 4000 은 은행에 없다.
  const book = await page.request.patch(`/api/v1/sheets/${ledger.id}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'cl-book', cells: [
      {row:1,column:1,value:'금액'},
      {row:2,column:1,value:1000},{row:3,column:1,value:2000},{row:4,column:1,value:4000},
    ] }})
  expect(book.status(), await book.text()).toBeLessThan(300)

  await page.reload()
  const canvas = page.locator('canvas.grid-canvas')
  await canvas.click({ position: { x: 70, y: 42 } })
  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '두 목록 비교…' }).click()
  const ask = page.getByRole('dialog', { name: '두 목록 비교' })
  await ask.getByLabel('비교할 범위').fill('장부!A1:A4')
  await ask.getByRole('button', { name: '비교' }).click()

  const dialog = page.getByRole('dialog', { name: '두 목록 비교' })
  // 글자 "1,000" 과 숫자 1000 이 같은 항목으로 맞아야 한다. 아니면 셋 다 어긋난다.
  await expect(dialog.locator('.compare-counts')).toContainText('양쪽에 2', { timeout: 15_000 })
  await expect(dialog.locator('.compare-counts')).toContainText('왼쪽에만 1')
  await expect(dialog.locator('.compare-counts')).toContainText('오른쪽에만 1')

  await dialog.getByRole('button', { name: '결과를 새 시트로' }).click()
  // 결과는 새 시트에만 적히고 원래 자료는 그대로다.
  await expect.poll(async()=>{
    const sheets = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
    return (sheets.sheets as Array<{name:string}>).some(sheet=>sheet.name.startsWith('대사'))
  },{timeout:15_000}).toBe(true)
  const untouched = await page.request.get(`/api/v1/sheets/${first}/ranges/A2:A4`).then(r => r.json())
  expect((untouched.items as Array<{value?:unknown}>).map(item=>item.value)).toEqual(['1,000','2,000','3,000'])
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
