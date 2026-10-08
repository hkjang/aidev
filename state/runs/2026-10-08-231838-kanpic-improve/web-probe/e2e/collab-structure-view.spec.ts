import { expect, test } from '@playwright/test'

// 다른 사람이 행을 지우면 내 화면도 다시 읽어야 한다. 그렇다고 내 선택
// 위치와 치고 있던 값까지 버릴 이유는 없다. 만 행짜리 표에서 남의 편집
// 한 번에 맨 위로 튀면 일을 이어갈 수가 없다.
test("someone else's row delete moves my selection instead of throwing it away", async ({ browser }) => {
  const stamp=Date.now()
  const first=await browser.newContext(),second=await browser.newContext()
  const author=await first.newPage(),observer=await second.newPage()
  author.on('dialog',dialog=>void dialog.accept())
  const workbook=await author.request.post('/api/v1/workbooks',{data:{title:`협업 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await author.request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`cs-${stamp}`,
    cells:[1,2,3,4,5,6].map(row=>({row,column:1,value:`행${row}`}))}})
  for(const page of [author,observer]){
    await page.goto(`/workbooks/${workbook.id}`)
    await expect(page.locator('.grid-canvas')).toBeVisible()
  }

  // 관찰자는 행5(A5)를 보고 있다.
  const grid=(await observer.locator('.grid-canvas').boundingBox())!
  await observer.mouse.click(grid.x+48+53,grid.y+30+12+4*24)
  await expect(observer.getByLabel('이름 상자')).toHaveValue('A5')

  const authorGrid=(await author.locator('.grid-canvas').boundingBox())!
  await author.mouse.click(authorGrid.x+20,authorGrid.y+30+2*24+12,{button:'right'})
  await author.getByLabel('행 메뉴').getByRole('menuitem',{name:'행 3 삭제'}).click()

  // 행5는 4행으로 올라갔으니 선택도 따라가야 한다.
  await expect(observer.getByLabel('이름 상자')).toHaveValue('A4',{timeout:15000})
  await first.close();await second.close()
})

// 남이 행을 지웠다고 내가 치고 있던 값이 사라지면, 다시 치는 수밖에 없다.
test('typing survives someone else\'s row delete elsewhere', async ({ browser }) => {
  const stamp=Date.now()
  const first=await browser.newContext(),second=await browser.newContext()
  const author=await first.newPage(),observer=await second.newPage()
  author.on('dialog',dialog=>void dialog.accept())
  const workbook=await author.request.post('/api/v1/workbooks',{data:{title:`입력 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await author.request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`ct-${stamp}`,
    cells:[1,2,3,4,5,6].map(row=>({row,column:1,value:`행${row}`}))}})
  for(const page of [author,observer]){
    await page.goto(`/workbooks/${workbook.id}`)
    await expect(page.locator('.grid-canvas')).toBeVisible()
  }

  // 관찰자가 B6에 값을 치는 중이다. 아직 저장하지 않았다.
  const grid=(await observer.locator('.grid-canvas').boundingBox())!
  await observer.mouse.click(grid.x+48+53+94,grid.y+30+12+5*24)
  await expect(observer.getByLabel('이름 상자')).toHaveValue('B6')
  await observer.keyboard.type('작성 중',{delay:30})
  await expect(observer.getByLabel('수식 입력창')).toHaveValue('작성 중')

  const authorGrid=(await author.locator('.grid-canvas').boundingBox())!
  await author.mouse.click(authorGrid.x+20,authorGrid.y+30+2*24+12,{button:'right'})
  await author.getByLabel('행 메뉴').getByRole('menuitem',{name:'행 3 삭제'}).click()

  // 치고 있던 값은 그대로, 자리는 한 줄 위로.
  await expect(observer.getByLabel('이름 상자')).toHaveValue('B5',{timeout:15000})
  await expect(observer.getByLabel('수식 입력창')).toHaveValue('작성 중')
  await observer.keyboard.press('Enter')
  await expect.poll(async()=>{
    const items=(await observer.request.get(`/api/v1/sheets/${sheet}/ranges/B5:B5`).then(response=>response.json())).items as Array<{value?:unknown}>
    return items[0]?.value
  }).toBe('작성 중')
  await first.close();await second.close()
})
