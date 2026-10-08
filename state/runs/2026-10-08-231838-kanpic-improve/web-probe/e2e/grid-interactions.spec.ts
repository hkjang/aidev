import { MAX_GRID_ROWS } from '../src/lib/clipboard'
import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

const HEADER_WIDTH=46,HEADER_HEIGHT=27,COLUMN_WIDTH=108,ROW_HEIGHT=27

async function seedWorkbook(request:APIRequestContext,title:string){
  const workbook=await request.post('/api/v1/workbooks',{data:{title}}).then(response=>response.json())
  const sheet=workbook.sheets[0]
  const cells:Array<Record<string,unknown>>=[{row:1,column:1,value:'도시'},{row:1,column:2,value:'매출'}]
  ;['서울','부산','대구','광주'].forEach((name,index)=>{
    cells.push({row:index+2,column:1,value:`${name} 지사`},{row:index+2,column:2,value:(index+1)*1000})
  })
  cells.push({row:6,column:2,formula:'=SUM(B2:B5)'})
  const response=await request.patch(`/api/v1/sheets/${sheet.id}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`e2e-seed-${workbook.id}`,cells}})
  expect(response.ok()).toBeTruthy()
  return{workbookId:workbook.id as string,sheetId:sheet.id as string}
}

async function openEditor(page:Page,workbookId:string){
  await page.goto(`/workbooks/${workbookId}`)
  await page.waitForSelector('.grid-canvas')
  await page.waitForTimeout(1000)
  const box=await page.locator('.grid-canvas').boundingBox()
  if(!box)throw new Error('grid canvas is not visible')
  return{
    box,
    columnCenter:(index:number)=>box.x+HEADER_WIDTH+COLUMN_WIDTH*(index-1)+COLUMN_WIDTH/2,
    columnEdge:(index:number)=>box.x+HEADER_WIDTH+COLUMN_WIDTH*index,
    rowCenter:(index:number)=>box.y+HEADER_HEIGHT+ROW_HEIGHT*(index-1)+ROW_HEIGHT/2,
    columnHeader:box.y+HEADER_HEIGHT/2,
  }
}

test('column header selects the column and its context menu edits structure', async ({ page, request }) => {
  const {workbookId,sheetId}=await seedWorkbook(request,'E2E 열 머리글')
  const grid=await openEditor(page,workbookId)

  await page.mouse.click(grid.columnCenter(2),grid.columnHeader)
  // 열 전체 선택은 편집기의 행 한도까지를 가리킨다. 그 한도는 서버가 담는
  // 시트보다 작으면 안 되므로 상수에서 읽는다.
  await expect(page.locator('.name-box')).toHaveValue(`B1:B${MAX_GRID_ROWS}`)

  await page.mouse.click(grid.columnCenter(2),grid.columnHeader,{button:'right'})
  const menu=page.locator('.context-menu')
  await expect(menu).toBeVisible()
  for(const entry of ['왼쪽에 열 1개 삽입','열 B 숨기기','이 열 기준 오름차순 정렬','열 너비 자동 맞춤'])
    await expect(menu).toContainText(entry)
  await page.getByRole('menuitem',{name:'왼쪽에 열 1개 삽입'}).click()
  await page.waitForTimeout(1200)

  const moved=await request.get(`/api/v1/sheets/${sheetId}/ranges/A1:D2`).then(response=>response.json())
  expect(moved.items.find((cell:{row:number;column:number})=>cell.row===1&&cell.column===3)?.value).toBe('매출')
})

test('dragging a header boundary stores the new column width', async ({ page, request }) => {
  const {workbookId}=await seedWorkbook(request,'E2E 열 너비')
  const grid=await openEditor(page,workbookId)

  await page.mouse.move(grid.columnEdge(1)-1,grid.columnHeader)
  await page.mouse.down()
  await page.mouse.move(grid.columnEdge(1)+79,grid.columnHeader,{steps:8})
  await page.mouse.up()
  await page.waitForTimeout(1200)

  const latest=await request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const width=latest.sheets[0].layout?.column_widths?.find((entry:{index:number})=>entry.index===1)?.size
  expect(width).toBeGreaterThan(180)
  expect(width).toBeLessThan(196)
})

test('the workbook menu bar toggles the formula view', async ({ page, request }) => {
  const {workbookId}=await seedWorkbook(request,'E2E 메뉴바')
  await openEditor(page,workbookId)

  await page.getByRole('menuitem',{name:'보기'}).click()
  await page.getByRole('menuitemcheckbox',{name:/수식 표시/}).click()
  await page.getByRole('menuitem',{name:'보기'}).click()
  await expect(page.getByRole('menuitemcheckbox',{name:/수식 표시/})).toHaveAttribute('aria-checked','true')
})

test('find and replace rewrites every matching cell', async ({ page, request }) => {
  const {workbookId}=await seedWorkbook(request,'E2E 찾기 바꾸기')
  const grid=await openEditor(page,workbookId)
  await page.mouse.click(grid.columnCenter(1),grid.rowCenter(9))

  await page.keyboard.press('Control+h')
  await page.getByRole('textbox',{name:'검색어'}).fill('지사')
  await page.getByRole('textbox',{name:'바꿀 내용'}).fill('본부')
  page.once('dialog',dialog=>dialog.accept())
  await page.getByRole('button',{name:'모두 바꾸기'}).click()
  await expect(page.locator('.workbook-search-status')).toContainText('4개 셀')

  await page.getByRole('button',{name:'정규식 사용'}).click()
  await page.getByRole('textbox',{name:'검색어'}).fill('^(서울|부산) 본부$')
  await expect(page.locator('.workbook-search-results [role="option"]')).toHaveCount(2)
})

test('keyboard shortcuts insert the current date and an auto sum', async ({ page, request }) => {
  const {workbookId}=await seedWorkbook(request,'E2E 단축키')
  const grid=await openEditor(page,workbookId)

  await page.mouse.click(grid.columnCenter(4),grid.rowCenter(2))
  await page.keyboard.press('Control+;')
  await page.waitForTimeout(1200)
  // The editor inserts the local date, so the expectation is built the same way.
  const now=new Date()
  const pad=(value:number)=>String(value).padStart(2,'0')
  const today=`${now.getFullYear()}-${pad(now.getMonth()+1)}-${pad(now.getDate())}`
  const found=await request.get(`/api/v1/workbooks/${workbookId}/search?q=${today}&whole_cell=true`).then(response=>response.json())
  expect(found.items).toHaveLength(1)

  await page.mouse.click(grid.columnCenter(2),grid.rowCenter(7))
  await page.keyboard.press('Alt+Equal')
  await expect(page.locator('.cell-editor')).toHaveValue(/^=SUM\(B\d+:B\d+\)$/)
})
