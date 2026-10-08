import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

async function seed(request:APIRequestContext,title:string,cells:Array<{row:number;column:number;value:unknown}>=[]){
  const workbook=await request.post('/api/v1/workbooks',{data:{title}}).then(response=>response.json())
  if(cells.length>0)await request.patch(`/api/v1/sheets/${workbook.sheets[0].id}/cells:batch`,{data:{
    base_version:workbook.version,idempotency_key:`input-seed-${workbook.id}`,cells,
  }})
  return workbook
}

async function openEditor(page:Page,workbookId:string){
  await page.goto(`/workbooks/${workbookId}`)
  await page.waitForSelector('.grid-canvas')
  await page.waitForTimeout(800)
}

const valuesOf=async(request:APIRequestContext,sheetId:string)=>{
  const range=await request.get(`/api/v1/sheets/${sheetId}/ranges/A1:E10`).then(response=>response.json())
  return Object.fromEntries((range.items??[]).map((cell:{row:number;column:number;value:unknown})=>[`${cell.row},${cell.column}`,cell.value]))
}

test('an IME composition starting on the grid keeps every syllable', async ({ page, request }) => {
  const workbook=await seed(request,`한글 입력 ${Date.now()}`)
  const sheet=workbook.sheets[0].id
  await openEditor(page,workbook.id)
  const box=await page.locator('.grid-canvas').boundingBox()
  if(!box)throw new Error('grid canvas is not visible')
  await page.mouse.click(box.x+80,box.y+42)

  // A Korean IME reports composition events rather than plain key presses. The
  // first syllable used to be swallowed as the raw latin key because no editor
  // existed yet, which turned 안녕 into dㅏ녕.
  await page.evaluate(()=>{
    const input=document.querySelector('.cell-editor') as HTMLTextAreaElement
    const setValue=Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype,'value')!.set!
    input.dispatchEvent(new CompositionEvent('compositionstart',{bubbles:true}))
    for(const [text,done] of [['ㅇ',false],['아',false],['안',false],['안ㄴ',false],['안녀',false],['안녕',true]] as Array<[string,boolean]>){
      setValue.call(input,text)
      input.dispatchEvent(new InputEvent('input',{bubbles:true,data:text,isComposing:!done}))
      if(done)input.dispatchEvent(new CompositionEvent('compositionend',{bubbles:true,data:text}))
    }
  })
  await expect(page.locator('.cell-editor')).toHaveValue('안녕')
  await page.keyboard.press('Enter')
  await expect.poll(async()=>(await valuesOf(request,sheet))['1,1'],{timeout:10_000}).toBe('안녕')
})

test('typing continues to work in the cell a commit moved to', async ({ page, request }) => {
  const workbook=await seed(request,`이동 후 입력 ${Date.now()}`)
  const sheet=workbook.sheets[0].id
  await openEditor(page,workbook.id)
  const box=await page.locator('.grid-canvas').boundingBox()
  if(!box)throw new Error('grid canvas is not visible')
  await page.mouse.click(box.x+80,box.y+42)

  await page.keyboard.type('첫값')
  await page.keyboard.press('Tab')
  await page.keyboard.type('둘째')
  await page.keyboard.press('Enter')
  await page.keyboard.type('셋째')
  await page.keyboard.press('Enter')
  await expect.poll(async()=>{
    const values=await valuesOf(request,sheet)
    return [values['1,1'],values['1,2'],values['2,2']].join('|')
  },{timeout:10_000}).toBe('첫값|둘째|셋째')
})

test('the keyboard stays on the grid after a name box jump and shortcuts still fire', async ({ page, request }) => {
  const workbook=await seed(request,`이동 후 단축키 ${Date.now()}`)
  const sheet=workbook.sheets[0].id
  await openEditor(page,workbook.id)

  await page.locator('.name-box').fill('C3')
  await page.keyboard.press('Enter')
  await page.keyboard.type('점프')
  await page.keyboard.press('Enter')
  await expect.poll(async()=>(await valuesOf(request,sheet))['3,3'],{timeout:10_000}).toBe('점프')

  // The hidden grid input must not swallow the workbook shortcuts.
  await page.keyboard.press('Control+k')
  await expect(page.getByRole('dialog',{name:'빠른 이동'})).toBeVisible()
  await page.keyboard.press('Escape')
  await page.keyboard.press('Control+f')
  await expect(page.getByRole('dialog',{name:/찾기/})).toBeVisible()
  await page.keyboard.press('Escape')
})

test('an administrator can list workbooks and reach the console from the profile menu', async ({ page, request }) => {
  const title=`관리자 목록 ${Date.now()}`
  await seed(request,title)
  // Listing runs a different query for administrators, which used to bind
  // parameters the statement no longer referenced and fail with a 500.
  expect((await request.get('/api/v1/workbooks')).status()).toBe(200)
  expect((await request.get('/api/v1/workbooks/trash')).status()).toBe(200)

  await page.goto('/')
  await expect(page.getByText(title,{exact:true})).toBeVisible()
  await page.locator('.profile-trigger').click()
  await expect(page.getByRole('link',{name:/관리자 콘솔/})).toBeVisible()
})

test('an edit is saved when the selection moves by click or by the formula bar', async ({ page, request }) => {
  const workbook=await seed(request,`클릭 저장 ${Date.now()}`,[{row:1,column:1,value:'원본'}])
  const sheet=workbook.sheets[0].id
  await openEditor(page,workbook.id)
  const box=await page.locator('.grid-canvas').boundingBox()
  if(!box)throw new Error('grid canvas is not visible')

  // Double click keeps the existing text, and clicking another cell saves it
  // instead of throwing the edit away.
  await page.mouse.dblclick(box.x+80,box.y+42)
  await expect(page.locator('.cell-editor')).toHaveValue('원본')
  await page.keyboard.type('추가')
  await page.mouse.click(box.x+220,box.y+42)
  await expect.poll(async()=>(await valuesOf(request,sheet))['1,1'],{timeout:10_000}).toBe('원본추가')

  // The formula bar edits the active cell and Enter moves down like the grid.
  await page.locator('.name-box').fill('C1')
  await page.keyboard.press('Enter')
  await page.getByRole('textbox',{name:'수식 입력창'}).click()
  await page.keyboard.type('=UPPER("kanpic")')
  await page.keyboard.press('Enter')
  await expect.poll(async()=>(await valuesOf(request,sheet))['1,3'],{timeout:10_000}).toBe('KANPIC')
  await page.keyboard.type('아래칸')
  await page.keyboard.press('Enter')
  await expect.poll(async()=>(await valuesOf(request,sheet))['2,3'],{timeout:10_000}).toBe('아래칸')
})
