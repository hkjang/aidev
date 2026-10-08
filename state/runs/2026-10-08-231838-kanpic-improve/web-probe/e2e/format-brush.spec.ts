import { expect, test, type APIRequestContext } from '@playwright/test'

const styleOf=async (request:APIRequestContext,sheet:string,range:string)=>
  ((await (await request.get(`/api/v1/sheets/${sheet}/ranges/${range}`)).json()).items[0]?.style)

// Copying a format is one of the most used toolbar buttons anywhere, so it has
// to make the target look like the source and leave the values alone.
test('the format brush copies a cell format onto another range', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`서식 복사 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}`,cells:[
    {row:1,column:1,value:'서식 원본',style:{bold:true,background:'#fee2e2',horizontal_align:'center'}},
    {row:1,column:2,value:'대상1',style:{italic:true}},
    {row:1,column:3,value:'대상2'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  const canvas=(await page.locator('.grid-canvas').boundingBox())!
  const cell=(column:number)=>({x:canvas.x+46+(column-1)*108+50,y:canvas.y+27+13})

  await page.mouse.click(cell(1).x,cell(1).y)
  await page.getByLabel('서식 복사').click()
  await page.mouse.click(cell(2).x,cell(2).y)

  // The target now matches the source, including losing the italic it had.
  await expect.poll(()=>styleOf(request,sheet,'B1:B1')).toMatchObject({bold:true,background:'#fee2e2',horizontal_align:'center'})
  expect(await styleOf(request,sheet,'B1:B1')).not.toHaveProperty('italic')
  // The value is untouched and the brush is put down after one use.
  const cellB=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/B1:B1`)).json()).items[0]
  expect(cellB.value).toBe('대상1')
  await page.mouse.click(cell(3).x,cell(3).y)
  await page.waitForTimeout(400)
  expect(await styleOf(request,sheet,'C1:C1')).toBeUndefined()
})

// Double clicking keeps the brush loaded, and Escape puts it down.
test('a double clicked brush paints repeatedly until Escape', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`연속 복사 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}`,cells:[
    {row:1,column:1,value:'원본',style:{bold:true}},{row:1,column:2,value:'가'},{row:1,column:3,value:'나'},{row:1,column:4,value:'다'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  const canvas=(await page.locator('.grid-canvas').boundingBox())!
  const cell=(column:number)=>({x:canvas.x+46+(column-1)*108+50,y:canvas.y+27+13})

  await page.mouse.click(cell(1).x,cell(1).y)
  await page.getByLabel('서식 복사').dblclick()
  await page.mouse.click(cell(2).x,cell(2).y)
  await expect.poll(()=>styleOf(request,sheet,'B1:B1')).toMatchObject({bold:true})
  await page.mouse.click(cell(3).x,cell(3).y)
  await expect.poll(()=>styleOf(request,sheet,'C1:C1')).toMatchObject({bold:true})

  await page.keyboard.press('Escape')
  await expect(page.getByLabel('서식 복사')).not.toHaveClass(/active/)
  await page.mouse.click(cell(4).x,cell(4).y)
  await page.waitForTimeout(400)
  expect(await styleOf(request,sheet,'D1:D1')).toBeUndefined()
})
