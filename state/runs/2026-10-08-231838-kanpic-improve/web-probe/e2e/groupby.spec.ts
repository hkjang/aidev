import { expect, test } from '@playwright/test'

// GROUPBY 는 피벗 테이블을 만들지 않고 수식 하나로 집계한다. 결과가 여러
// 칸이므로 실제로 칸에 펼쳐지는지, 원본이 바뀌면 따라 바뀌는지 확인한다.
test('GROUPBY spills a grouped table and follows the source', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'gb-seed', cells: [
      {row:1,column:1,value:'영업1팀'},{row:1,column:2,value:100},
      {row:2,column:1,value:'영업2팀'},{row:2,column:2,value:200},
      {row:3,column:1,value:'영업1팀'},{row:3,column:2,value:50},
      // 이름만 적어 넘기는 꼴. LAMBDA 로 감싸지 않는다.
      {row:1,column:4,formula:'=GROUPBY(A1:A3,B1:B3,SUM)'},
    ],
  }})
  expect(write.status(), await write.text()).toBeLessThan(300)

  const read = async () => {
    const result = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/D1:E3`).then(r => r.json())
    const cells = result.items as Array<{row:number;column:number;value?:unknown}>
    return [1,2,3].map(row => [1,2].map(offset =>
      cells.find(item=>item.row===row&&item.column===3+offset)?.value ?? null))
  }
  await expect.poll(read,{timeout:15_000}).toEqual([['영업1팀',150],['영업2팀',200],['총합계',350]])

  // 원본을 고치면 집계도 따라 바뀐다. 피벗 테이블처럼 새로 고칠 필요가 없다.
  const latest = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const update = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: latest.version, idempotency_key: 'gb-update', cells: [{row:3,column:2,value:150}] }})
  expect(update.status(), await update.text()).toBeLessThan(300)
  await expect.poll(read,{timeout:15_000}).toEqual([['영업1팀',250],['영업2팀',200],['총합계',450]])
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
