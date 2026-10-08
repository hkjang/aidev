import { expect, test, type APIRequestContext } from '@playwright/test'

const cells=(request:APIRequestContext,sheet:string,body:unknown)=>
  request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:body})

const read=async (request:APIRequestContext,sheet:string,range:string)=>{
  const response=await request.get(`/api/v1/sheets/${sheet}/ranges/${range}`)
  const items=(await response.json()).items as Array<{row:number;column:number;value?:unknown;formula?:string}>
  return new Map(items.map(item=>[`${item.row}:${item.column}`,item]))
}

// Inserting rows and columns must leave every formula pointing at the same
// data, which is the behaviour people rely on from Google Sheets.
test('formula ranges follow rows and columns as the sheet changes', async ({ request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`범위 조정 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await cells(request,sheet,{idempotency_key:`seed-${Date.now()}`,cells:[
    {row:1,column:1,value:10},{row:2,column:1,value:20},{row:3,column:1,value:30},
    {row:1,column:3,formula:'=SUM(A1:A3)'},{row:2,column:3,formula:'=A1+A3'},
    {row:3,column:3,formula:'=SUM(A:A)'},{row:4,column:3,formula:'=COUNT(A:A)'},
  ]})
  let grid=await read(request,sheet,'C1:C4')
  expect(grid.get('1:3')?.value).toBe(60)
  expect(grid.get('3:3')?.value).toBe(60)

  // A row inserted inside the range widens it instead of dropping a value.
  const version=(await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())).version
  await request.patch(`/api/v1/sheets/${sheet}/structure:apply`,{data:{axis:'row',action:'insert',index:2,count:1,idempotency_key:`ins-${Date.now()}`,base_version:version}})
  // Every formula below the insertion moved down a row with the cells it sat in.
  grid=await read(request,sheet,'C1:C5')
  expect(grid.get('1:3')?.formula).toBe('=SUM(A1:A4)')
  expect(grid.get('1:3')?.value).toBe(60)
  expect(grid.get('3:3')?.formula).toBe('=A1+A4')
  // The whole-column reference does not move for a row change, and still totals.
  expect(grid.get('4:3')?.formula).toBe('=SUM(A:A)')
  expect(grid.get('4:3')?.value).toBe(60)

  // New data below the last row joins the whole-column total with no edit.
  await cells(request,sheet,{idempotency_key:`grow-${Date.now()}`,cells:[{row:40,column:1,value:5}]})
  grid=await read(request,sheet,'C4:C5')
  expect(grid.get('4:3')?.value).toBe(65)
  expect(grid.get('5:3')?.value).toBe(4)

  // A column inserted to the left shifts every reference one column across.
  const next=(await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())).version
  await request.patch(`/api/v1/sheets/${sheet}/structure:apply`,{data:{axis:'column',action:'insert',index:1,count:1,idempotency_key:`col-${Date.now()}`,base_version:next}})
  grid=await read(request,sheet,'D1:D5')
  expect(grid.get('1:4')?.formula).toBe('=SUM(B1:B4)')
  expect(grid.get('4:4')?.formula).toBe('=SUM(B:B)')
  expect(grid.get('4:4')?.value).toBe(65)
})

test('a workbook made from a formula template opens with every cell calculated', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`수식 템플릿 ${Date.now()}`,template_id:'formula-finance'}}).then(response=>response.json())
  const grid=await read(request,workbook.sheets[0].id,'A1:F16')
  const values=[...grid.values()].filter(item=>item.formula).map(item=>item.value)
  expect(values.length).toBeGreaterThan(10)
  expect(values.filter(value=>typeof value==='string'&&value.startsWith('#'))).toEqual([])
  // 월 상환액 is the number the sheet exists to produce.
  expect(Math.round(Number(grid.get('12:2')?.value))).toBe(-1520056)

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.getByRole('button',{name:'대출',exact:true})).toBeVisible()
})
