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

// 저장 큐는 시트를 가리지 않고 밀린 작업을 모두 보낸다. 그래서 A 워크북의 편집이 큐에
// 남은 채 B 를 열면, A 의 응답이 B 의 화면에 얹혀 B 의 되돌리기 더미에 A 의 작업 번호가
// 쌓였다. B 에서 Ctrl+Z 를 누르면 보이지도 않는 A 의 셀이 되돌아갔다.
test('an edit queued by another workbook never lands on the open one', async ({ page, request }) => {
  const stamp=Date.now()
  const first=await seed(request,`큐 격리 A ${stamp}`),second=await seed(request,`큐 격리 B ${stamp}`)
  const firstSheet=first.sheets[0].id

  // 셀 저장만 끊어 두면 편집은 큐에 남고 워크북은 계속 열 수 있다.
  await page.route('**/cells:batch',route=>route.abort())
  await openEditor(page,first.id)
  await typeIntoA1(page,'밀린 값')
  await page.waitForTimeout(1000)
  expect(await valueAt(request,firstSheet,'A1')).toBeUndefined()

  await page.unroute('**/cells:batch')
  await openEditor(page,second.id)
  // B 의 3초 주기 비우기가 A 의 작업을 보낸다.
  await expect.poll(async()=>valueAt(request,firstSheet,'A1'),{timeout:20_000}).toBe('밀린 값')

  // B 에서 되돌리기를 눌러도 A 의 셀은 그대로여야 한다.
  await page.locator('.grid-viewport').click()
  await page.keyboard.press('Control+z')
  await page.waitForTimeout(2000)
  expect(await valueAt(request,firstSheet,'A1')).toBe('밀린 값')
})
