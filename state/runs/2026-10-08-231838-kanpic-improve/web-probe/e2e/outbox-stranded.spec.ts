import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

async function seed(request:APIRequestContext,title:string){
  return request.post('/api/v1/workbooks',{data:{title}}).then(response=>response.json())
}

async function openEditor(page:Page,workbookId:string){
  await page.goto(`/workbooks/${workbookId}`)
  await page.waitForSelector('.grid-canvas')
  await page.waitForTimeout(800)
}

async function typeIntoA1(page:Page,text:string){
  const box=await page.locator('.grid-canvas').boundingBox()
  if(!box)throw new Error('grid canvas is not visible')
  await page.mouse.click(box.x+80,box.y+42)
  await page.keyboard.type(text)
  await page.keyboard.press('Enter')
}

const valueAt=async(request:APIRequestContext,sheetId:string,address:string)=>{
  const range=await request.get(`/api/v1/sheets/${sheetId}/ranges/${address}:${address}`).then(response=>response.json())
  return (range.items??[])[0]?.value
}

// 큐를 비우는 일이 편집기에서만 돌면, 다시 열지 않는 워크북에 갇힌 편집은 나가지 못한다.
test('the workbook list sends an edit stranded by a workbook nobody reopened', async ({ page, request }) => {
  const workbook=await seed(request,`갇힌 편집 ${Date.now()}`)
  await page.route('**/cells:batch',route=>route.abort())
  await openEditor(page,workbook.id)
  await typeIntoA1(page,'갇힌 값')
  await page.waitForTimeout(1000)
  expect(await valueAt(request,workbook.sheets[0].id,'A1')).toBeUndefined()

  await page.unroute('**/cells:batch')
  await page.goto('/')
  await page.waitForSelector('.home-content')
  await expect.poll(async()=>valueAt(request,workbook.sheets[0].id,'A1'),{timeout:20_000}).toBe('갇힌 값')
  // 나간 편집은 알림으로 남지 않는다.
  await expect(page.locator('.stranded-edits')).toHaveCount(0)
})

// 목록에서도 나가지 못한 편집은 사람에게 보여야 한다.
test('the workbook list names a workbook whose edits gave up and can drop them', async ({ page, request }) => {
  test.setTimeout(120_000)
  const workbook=await seed(request,`포기한 편집 ${Date.now()}`)
  await page.route('**/cells:batch',route=>route.fulfill({status:500,contentType:'application/json',body:'{"error":{"message":"서버 오류"}}'}))
  await openEditor(page,workbook.id)
  await typeIntoA1(page,'포기한 값')
  await expect(page.locator('.stalled-save')).toBeVisible({timeout:40_000})

  await page.goto('/')
  await page.waitForSelector('.home-content')
  await expect(page.locator('.stranded-edits')).toContainText('편집이 1건 남아 있습니다',{timeout:20_000})
  await expect(page.locator('.stranded-actions')).toContainText(`${workbook.title} 1건 열기`)

  page.on('dialog',dialog=>void dialog.accept())
  await page.locator('.stranded-actions button',{hasText:'버리기'}).click()
  await expect(page.locator('.stranded-edits')).toHaveCount(0)
})
