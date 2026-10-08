import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

const seed=async(request:APIRequestContext,title:string,rows:number)=>{
  const workbook=await request.post('/api/v1/workbooks',{data:{title}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${workbook.id}`,
    cells:[...Array(rows)].map((_,index)=>({row:index+1,column:1,value:`행${index+1}`}))}})
  return {workbook,sheet}
}
const typeInto=async(page:Page,row:number,column:number,text:string)=>{
  const grid=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(grid.x+48+53+(column-1)*94,grid.y+30+12+(row-1)*27)
  await page.keyboard.type(text)
  await page.keyboard.press('Enter')
}
const columnB=async(request:APIRequestContext,sheet:string)=>{
  const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/B1:B8`).then(response=>response.json())).items as Array<{row:number;value?:unknown}>
  return items.map(item=>[item.row,item.value])
}

// 끊긴 채로 친 값은 대기열에 남는다. 탭을 닫아도 살아 있어야 한다.
test('edits made offline survive closing the tab', async ({ browser }) => {
  const context=await browser.newContext()
  const page=await context.newPage()
  const {workbook,sheet}=await seed(page.request,`오프라인 재시작 ${Date.now()}`,3)
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  await context.setOffline(true)
  await typeInto(page,1,2,'끊긴 채로 쓴 값')
  await expect(page.getByText('오프라인 · 로컬 저장',{exact:true})).toBeVisible()
  await page.close()

  await context.setOffline(false)
  const reopened=await context.newPage()
  await reopened.goto(`/workbooks/${workbook.id}`)
  await expect(reopened.locator('.grid-canvas')).toBeVisible()
  await expect.poll(async()=>(await columnB(reopened.request,sheet))[0]?.[1],{timeout:20_000}).toBe('끊긴 채로 쓴 값')
  await context.close()
})

// 끊겨 있는 동안 다른 사람이 위쪽 행을 지우면, 돌아왔을 때 내 편집은 원래
// 겨냥한 줄을 따라가야 한다. 행 번호 그대로 적용하면 남의 값을 덮어쓴다.
test('an offline edit follows its row when someone deletes above it', async ({ browser }) => {
  const mine=await browser.newContext(),theirs=await browser.newContext()
  const page=await mine.newPage(),author=await theirs.newPage()
  author.on('dialog',dialog=>void dialog.accept())
  const {workbook,sheet}=await seed(page.request,`오프라인 재배치 ${Date.now()}`,6)
  for(const each of [page,author]){
    await each.goto(`/workbooks/${workbook.id}`)
    await expect(each.locator('.grid-canvas')).toBeVisible()
  }
  await mine.setOffline(true)
  await typeInto(page,5,2,'오프라인 B5')

  const grid=(await author.locator('.grid-canvas').boundingBox())!
  await author.mouse.click(grid.x+20,grid.y+30+2*27+12,{button:'right'})
  await author.getByLabel('행 메뉴').getByRole('menuitem',{name:'행 3 삭제'}).click()
  await expect.poll(async()=>{
    const items=(await author.request.get(`/api/v1/sheets/${sheet}/ranges/A1:A5`).then(response=>response.json())).items as Array<{value?:unknown}>
    return items.map(item=>item.value)
  }).toEqual(['행1','행2','행4','행5','행6'])

  await mine.setOffline(false)
  // 행5는 4행으로 올라갔으니 그 옆의 편집도 4행에 있어야 한다.
  await expect.poll(async()=>await columnB(author.request,sheet),{timeout:20_000}).toEqual([[4,'오프라인 B5']])
  await mine.close();await theirs.close()
})

// 겨냥한 행 자체가 사라졌다면 갈 곳이 없다. 그 자리를 물려받은 값을 덮어쓰지
// 않고, 반영되지 않았다고 알려 준다.
test('an offline edit into a deleted row is refused and reported', async ({ browser }) => {
  const mine=await browser.newContext(),theirs=await browser.newContext()
  const page=await mine.newPage(),author=await theirs.newPage()
  author.on('dialog',dialog=>void dialog.accept())
  const {workbook,sheet}=await seed(page.request,`오프라인 삭제 ${Date.now()}`,6)
  for(const each of [page,author]){
    await each.goto(`/workbooks/${workbook.id}`)
    await expect(each.locator('.grid-canvas')).toBeVisible()
  }
  await mine.setOffline(true)
  await typeInto(page,3,2,'사라질 행에 쓴 값')

  const grid=(await author.locator('.grid-canvas').boundingBox())!
  await author.mouse.click(grid.x+20,grid.y+30+2*27+12,{button:'right'})
  await author.getByLabel('행 메뉴').getByRole('menuitem',{name:'행 3 삭제'}).click()
  await expect.poll(async()=>{
    const items=(await author.request.get(`/api/v1/sheets/${sheet}/ranges/A1:A5`).then(response=>response.json())).items as Array<{value?:unknown}>
    return items.map(item=>item.value)
  }).toEqual(['행1','행2','행4','행5','행6'])

  await mine.setOffline(false)
  await expect(page.locator('.formula-issue')).toContainText('편집 1곳이 반영되지 않았습니다',{timeout:20_000})
  await expect(page.locator('.formula-issue')).toContainText('B3')
  // 그 자리를 물려받은 행은 그대로다.
  expect(await columnB(author.request,sheet)).toEqual([])
  await mine.close();await theirs.close()
})
