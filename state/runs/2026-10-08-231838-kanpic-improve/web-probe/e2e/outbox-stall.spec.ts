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

// 서버가 계속 500 을 내면 큐는 3초마다 영원히 다시 붙었고, 화면은 "저장 중" 이라고만
// 했다. 멈춘 것을 말하지 않으면 사람은 저장된 줄 알고 창을 닫는다.
test('a save the server keeps refusing stops retrying and asks what to do', async ({ page, request }) => {
  // 포기까지 다섯 번, 다시 시도 뒤 또 다섯 번을 3초 간격으로 기다린다.
  test.setTimeout(150_000)
  const workbook=await seed(request,`큐 포기 ${Date.now()}`)
  let sent=0
  await page.route('**/cells:batch',route=>{sent+=1;return route.fulfill({status:500,contentType:'application/json',body:'{"error":{"message":"서버 오류"}}'})})
  await openEditor(page,workbook.id)
  await typeIntoA1(page,'막힌 값')

  await expect(page.locator('.stalled-save')).toContainText('변경 1건을 저장하지 못했습니다',{timeout:30_000})
  // 포기한 뒤로는 더 보내지 않는다.
  const afterGivingUp=sent
  await page.waitForTimeout(7000)
  expect(sent).toBe(afterGivingUp)

  // 다시 시도하면 다시 보낸다.
  await page.locator('.stalled-save button',{hasText:'다시 시도'}).click()
  await expect.poll(()=>sent,{timeout:30_000}).toBeGreaterThan(afterGivingUp)
  await expect(page.locator('.stalled-save')).toContainText('변경 1건을 저장하지 못했습니다',{timeout:30_000})

  // 버리면 큐에서 사라지고 서버의 값이 남는다.
  page.on('dialog',dialog=>void dialog.accept())
  await page.locator('.stalled-save button',{hasText:'버리기'}).click()
  await expect(page.locator('.stalled-save')).toHaveCount(0)
  expect(await valueAt(request,workbook.sheets[0].id,'A1')).toBeUndefined()
})

// 남의 워크북에서 막힌 작업 하나가 여기의 저장까지 세우면 안 된다.
test('a workbook whose queue gave up does not stop saving in another workbook', async ({ page, request }) => {
  const stamp=Date.now()
  const stuck=await seed(request,`막힌 A ${stamp}`),healthy=await seed(request,`멀쩡한 B ${stamp}`)

  await page.route('**/cells:batch',route=>route.fulfill({status:500,contentType:'application/json',body:'{"error":{"message":"서버 오류"}}'}))
  await openEditor(page,stuck.id)
  await typeIntoA1(page,'막힌 값')
  await expect(page.locator('.stalled-save')).toContainText('저장하지 못했습니다',{timeout:30_000})

  await page.unroute('**/cells:batch')
  await openEditor(page,healthy.id)
  await typeIntoA1(page,'멀쩡한 값')
  await expect.poll(async()=>valueAt(request,healthy.sheets[0].id,'A1'),{timeout:20_000}).toBe('멀쩡한 값')
  // B 는 자기 큐가 비어 있으므로 알림도 뜨지 않는다.
  await expect(page.locator('.stalled-save')).toHaveCount(0)
})

// 오프라인에서 몇 건이 밀려 있는지 화면이 말하지 않으면, 사람은 저장된 줄 알고 닫는다.
test('the badge counts what is still waiting to reach the server', async ({ page, context, request }) => {
  const workbook=await seed(request,`대기 수 ${Date.now()}`)
  await openEditor(page,workbook.id)
  await context.setOffline(true)
  await typeIntoA1(page,'첫 값')
  const box=await page.locator('.grid-canvas').boundingBox()
  if(!box)throw new Error('grid canvas is not visible')
  await page.mouse.click(box.x+80,box.y+62)
  await page.keyboard.type('둘째 값')
  await page.keyboard.press('Enter')

  await expect(page.locator('.sheet-status')).toContainText('2건 대기',{timeout:15_000})
  await context.setOffline(false)
  await expect(page.locator('.sheet-status')).toContainText('모든 변경사항 저장됨',{timeout:20_000})
  await expect(page.locator('.sheet-status')).not.toContainText('대기')
  expect(await valueAt(request,workbook.sheets[0].id,'A1')).toBe('첫 값')
})
