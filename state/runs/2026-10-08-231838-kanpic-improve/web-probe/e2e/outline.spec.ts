import { expect, test, type APIRequestContext } from '@playwright/test'

const layout=(request:APIRequestContext,sheet:string,body:Record<string,unknown>)=>
  request.patch(`/api/v1/sheets/${sheet}/layout:apply`,{data:body})

// Grouping folds a section of a long sheet away behind one control, and the
// control has to work with a click on the outline gutter.
test('rows group, fold away and come back from the outline control', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`그룹 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}`,cells:
    ['머리글','상세1','상세2','상세3','요약'].map((value,index)=>({row:index+1,column:1,value}))}})

  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  // Select rows 2 to 4 by their headers, which is what a menu about whole rows
  // acts on. The canvas is sticky inside a very tall spacer, so every click is
  // aimed with page coordinates rather than an offset Playwright would scroll to.
  const canvas=(await page.locator('.grid-canvas').boundingBox())!
  const rowHeader=(row:number)=>({x:canvas.x+20,y:canvas.y+27+(row-1)*27+13})
  await page.mouse.click(rowHeader(2).x,rowHeader(2).y)
  await page.keyboard.down('Shift')
  await page.mouse.click(rowHeader(4).x,rowHeader(4).y)
  await page.keyboard.up('Shift')
  await page.mouse.click(rowHeader(4).x,rowHeader(4).y,{button:'right'})
  await page.getByRole('menuitem',{name:'행 2–4 그룹화'}).click()
  await expect.poll(async()=>{
    const sheets=(await (await request.get(`/api/v1/workbooks/${workbook.id}`)).json()).sheets
    return sheets[0].layout.row_groups?.length??0
  }).toBe(1)

  // Folding hides exactly the grouped rows; row 5 stays.
  const revision=(await (await request.get(`/api/v1/workbooks/${workbook.id}`)).json()).sheets[0].layout.revision
  await layout(request,sheet,{idempotency_key:`fold-${Date.now()}`,expected_revision:revision,action:'collapse',axis:'row',start:2,count:3})
  await page.reload()
  await page.waitForSelector('.grid-canvas')
  await page.locator('.name-box').fill('A5')
  await page.keyboard.press('Enter')
  await expect(page.getByLabel('A5 셀 입력')).toBeVisible()
  await expect(page.locator('.grid-viewport .sr-only')).toContainText('요약')

  // The control in the gutter brings them back. It sits beside the row after
  // the group, which is the first visible row once the group is folded.
  // With rows 2 to 4 folded away, row 5 is drawn directly under row 1, and the
  // control box sits in the gutter beside it.
  const gutter=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(gutter.x+6,gutter.y+27+27+13)
  await expect.poll(async()=>{
    const sheets=(await (await request.get(`/api/v1/workbooks/${workbook.id}`)).json()).sheets
    return sheets[0].layout.row_groups?.[0]?.collapsed
  }).toBe(false)
})

// The outline is part of the sheet, so it moves with the rows it wraps.
test('an outline group moves when rows are inserted above it', async ({ request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`그룹 이동 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await layout(request,sheet,{idempotency_key:`group-${Date.now()}`,expected_revision:1,action:'group',axis:'row',start:5,count:4})
  const version=(await (await request.get(`/api/v1/workbooks/${workbook.id}`)).json()).version
  await request.patch(`/api/v1/sheets/${sheet}/structure:apply`,{data:{axis:'row',action:'insert',index:2,count:3,idempotency_key:`ins-${Date.now()}`,base_version:version}})
  const groups=(await (await request.get(`/api/v1/workbooks/${workbook.id}`)).json()).sheets[0].layout.row_groups
  expect(groups).toEqual([{start:8,end:11,collapsed:false,depth:0}])
})
