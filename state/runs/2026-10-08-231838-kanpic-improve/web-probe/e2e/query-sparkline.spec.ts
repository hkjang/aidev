import { expect, test, type APIRequestContext } from '@playwright/test'

const read=async (request:APIRequestContext,sheet:string,range:string)=>{
  const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/${range}`)).json()).items as Array<{row:number;column:number;value?:unknown}>
  return new Map(items.map(item=>[`${item.row}:${item.column}`,item.value]))
}

// QUERY is the one formula that replaces a filter, a sort and a pivot, so the
// result has to spill as a real table.
test('QUERY groups a table and spills the result with its header', async ({ request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`QUERY ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','채널','매출'],['서울','온라인',4200000],['부산','오프라인',1850000],
    ['서울','오프라인',3100000],['대구','온라인',970000],['서울','온라인',5600000]]
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}`,cells:[
    ...rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value}))),
    {row:1,column:5,formula:'=QUERY(A1:C6,"select A, sum(C) where C > 1000000 group by A order by sum(C) desc")'},
  ]}})
  const grid=await read(request,sheet,'E1:F5')
  expect(grid.get('1:5')).toBe('지역')
  expect(grid.get('1:6')).toBe('sum 매출')
  expect(grid.get('2:5')).toBe('서울')
  expect(grid.get('2:6')).toBe(12900000)
  expect(grid.get('3:5')).toBe('부산')
  // 대구 is filtered out by the where clause.
  expect(grid.get('4:5')).toBeUndefined()
})

// A sparkline is a chart drawn inside the cell, so nothing should be written
// there as text.
test('SPARKLINE draws inside the cell instead of writing a value', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`스파크라인 ${Date.now()}`,template_id:'formula-query'}}).then(response=>response.json())
  const sales=workbook.sheets.find((sheet:{name:string})=>sheet.name==='판매')
  const grid=await read(request,sales.id,'H4:H7')
  expect(grid.get('4:8')).toMatchObject({kanpic:'sparkline',chart:'line'})
  expect(grid.get('7:8')).toMatchObject({chart:'column',color:'#5268a6'})

  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  await page.locator('.name-box').fill('H4')
  await page.keyboard.press('Enter')
  // The cell holds a formula, and the value behind it never appears as text.
  await expect(page.getByLabel('수식 입력창')).toHaveValue('=SPARKLINE(B4:G4)')
  await expect(page.getByLabel('H4 셀 입력')).toHaveValue('')
  // A drawing is invisible to a screen reader, so it is described in words.
  await expect(page.locator('.grid-viewport .sr-only')).toContainText('선형 미니 차트, 값 6개, 90부터 210까지, 상승')
})
