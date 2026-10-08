import { expect, test, type APIRequestContext } from '@playwright/test'

/**
 * The grid only loads the rows on screen. Anything that summarises or lists a
 * whole column therefore has to read from the server — these tests exist
 * because computing from the loaded cells produced confidently wrong totals.
 */
const ROWS=400

async function tallSheet(request:APIRequestContext,title:string){
  const workbook=await request.post('/api/v1/workbooks',{data:{title}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const cells=[{row:1,column:1,value:'지역'},{row:1,column:2,value:'매출'}]
  for(let row=2;row<=ROWS+1;row+=1){
    // The last rows hold a region that appears nowhere near the top of the sheet.
    cells.push({row,column:1,value:row>ROWS-1?'제주':'서울'})
    cells.push({row,column:2,value:100})
  }
  await request.patch(`/api/v1/sheets/${sheet}/cells:paste`,{data:{idempotency_key:`seed-${Date.now()}`,cells}})
  return {workbook,sheet}
}

test('the selection summary totals the whole selection, not just the visible rows', async ({ page, request }) => {
  const {workbook}=await tallSheet(request,`큰 시트 요약 ${Date.now()}`)
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  await page.locator('.name-box').fill(`B2:B${ROWS+1}`)
  await page.keyboard.press('Enter')

  // 400 rows of 100 each, far more than a screen holds.
  await expect(page.locator('.selection-summary')).toContainText(`합계 ${(ROWS*100).toLocaleString('ko-KR')}`)
  await expect(page.locator('.selection-summary')).toContainText(`개수 ${ROWS.toLocaleString('ko-KR')}`)
})

test('the column filter lists values that only appear far down the sheet', async ({ page, request }) => {
  const {workbook,sheet}=await tallSheet(request,`큰 시트 필터 ${Date.now()}`)
  await request.post(`/api/v1/sheets/${sheet}/filter-views`,{data:{
    name:'기본 필터',range:`A1:B${ROWS+1}`,header_rows:1,active:true,criteria:[],idempotency_key:`filter-${Date.now()}`}})
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  const canvas=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(canvas.x+46+108-14,canvas.y+13)

  const menu=page.getByRole('dialog',{name:'A열 필터'})
  // 제주 only exists in the last rows, which are nowhere near the viewport.
  await expect(menu.getByRole('checkbox',{name:/제주/})).toBeVisible()
  await expect(menu.getByRole('checkbox',{name:/서울/})).toBeVisible()

  await menu.getByRole('button',{name:'선택 해제'}).click()
  await menu.getByRole('checkbox',{name:/제주/}).click()
  await menu.getByRole('button',{name:'적용'}).click()
  await expect.poll(async()=>{
    const views=(await (await request.get(`/api/v1/sheets/${sheet}/filter-views`)).json()).items
    return views[0]?.criteria
  }).toEqual([{column:1,operator:'values',values:['제주']}])
})

// Sorting has to cover the whole table. Working the block out from the rows in
// memory sorted only the visible part, which scrambles a table against itself.
test('quick sort covers the whole table, not the rows on screen', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`큰 시트 정렬 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const cells:Array<Record<string,unknown>>=[{row:1,column:1,value:'번호'}]
  for(let row=2;row<=ROWS+1;row+=1)cells.push({row,column:1,value:ROWS+2-row})
  await request.patch(`/api/v1/sheets/${sheet}/cells:paste`,{data:{idempotency_key:`seed-${Date.now()}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  page.on('dialog',dialog=>void dialog.accept())
  await page.locator('.name-box').fill('A2')
  await page.keyboard.press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'정렬',exact:true}).click()
  await page.getByRole('menuitem',{name:'선택 열 기준 정렬 A → Z'}).click()
  await page.getByRole('dialog',{name:'정렬 범위 확인'}).getByRole('button',{name:'정렬'}).click()

  await expect.poll(async()=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/A2:A${ROWS+1}`)).json()).items as Array<{row:number;value:number}>
    const values=items.sort((first,second)=>first.row-second.row).map(item=>item.value)
    return values.length===ROWS&&values.every((value,index)=>index===0||values[index-1]<=value)
  },{timeout:15_000}).toBe(true)
})

// Printing from memory produced a page of whatever happened to be scrolled to.
test('printing covers the whole sheet, not the rows on screen', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`큰 시트 인쇄 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const cells:Array<Record<string,unknown>>=[{row:1,column:1,value:'항목'}]
  for(let row=2;row<=ROWS+1;row+=1)cells.push({row,column:1,value:`항목 ${row-1}`})
  await request.patch(`/api/v1/sheets/${sheet}/cells:paste`,{data:{idempotency_key:`seed-${Date.now()}`,cells}})

  // The print dialog would block the run, so only the document is inspected.
  await page.addInitScript(()=>{window.print=()=>{}})
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  await page.getByRole('menuitem',{name:'파일'}).click()
  await page.getByRole('menuitem',{name:/^인쇄(?!\s*영역)/}).click()

  await expect.poll(async()=>page.evaluate(()=>{
    for(const frame of [...document.querySelectorAll('iframe')]){
      const doc=frame.contentDocument
      if(doc&&doc.querySelectorAll('tr').length>0)return doc.body.innerText.includes(`항목 ${400}`)
    }
    return false
  }),{timeout:10_000}).toBe(true)
})

// Auto sum walked up through the rows the grid had loaded, so a long column
// summed its visible tail and looked perfectly ordinary doing it.
test('auto sum covers the whole column, not the rows on screen', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`자동 합계 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const cells:Array<Record<string,unknown>>=[]
  for(let row=1;row<=ROWS;row+=1)cells.push({row,column:1,value:1})
  await request.patch(`/api/v1/sheets/${sheet}/cells:paste`,{data:{idempotency_key:`sum-seed-${Date.now()}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  await page.locator('.name-box').fill(`A${ROWS+1}`)
  await page.keyboard.press('Enter')
  // The grid takes the selection before it takes keystrokes; pressing the
  // shortcut too early lands it on whatever was selected before.
  await expect(page.locator('.name-box')).toHaveValue(`A${ROWS+1}`)
  // Under a loaded suite the grid occasionally takes focus a beat after the
  // name box hands it back, and the shortcut goes nowhere. Retrying the key
  // press tolerates that without weakening what is being checked.
  await expect.poll(async()=>{
    if(await page.locator('.cell-editor').inputValue()==='')await page.keyboard.press('Alt+Equal')
    return page.locator('.cell-editor').inputValue()
  },{timeout:15_000}).toBe(`=SUM(A1:A${ROWS})`)
  await page.keyboard.press('Enter')

  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A${ROWS+1}:A${ROWS+1}`).then(response=>response.json())
    return range.items[0]?.value
  },{timeout:15_000}).toBe(ROWS)
})
