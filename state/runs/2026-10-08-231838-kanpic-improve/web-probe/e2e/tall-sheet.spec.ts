import { expect, test } from '@playwright/test'

// 서버는 5만 행짜리 시트를 담고 내보내기에도 넣어 주지만, 편집기는 만 행
// 너머를 아예 보여 주지 않았다. A20000으로 이동하면 A10000에 멈췄고, 2만
// 행짜리 표를 정렬하면 앞의 절반만 정렬됐다.
test('rows past ten thousand are reachable, editable and sortable', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`긴 시트 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=12_000
  for(let start=1;start<=rows;start+=3000){
    const end=Math.min(start+2999,rows),cells=[]
    for(let row=start;row<=end;row+=1)cells.push({row,column:1,value:`항목${(row*7919)%rows}`},{row,column:2,value:(row*7)%1000})
    await request.patch(`/api/v1/sheets/${sheet}/cells:paste`,{data:{idempotency_key:`tall-${stamp}-${start}`,cells}})
  }

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const nameBox=page.getByLabel('이름 상자')
  await nameBox.fill('C12000')
  await nameBox.press('Enter')
  // 예전에는 여기서 C10000에 멈췄다.
  await expect(nameBox).toHaveValue('C12000')

  await page.keyboard.type('맨 아래')
  await page.keyboard.press('Enter')
  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/C12000:C12000`).then(response=>response.json())).items as Array<{value?:unknown}>
    return items[0]?.value
  }).toBe('맨 아래')

  // 정렬도 표 전체를 대상으로 삼아야 한다.
  const grid=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(grid.x+48+53+94,grid.y+12,{button:'right'})
  await page.getByLabel('열 메뉴').getByRole('menuitem',{name:/오름차순/}).first().click()
  // 표 전체가 대상이어야 한다. 예전에는 9,999행만 정렬한다고 나왔다.
  await expect(page.getByRole('dialog').first()).toContainText('12,000행이 정렬됩니다')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 열 쪽도 같았다. 서버는 엑셀과 같은 XFD열까지 받는데 편집기는 500번째
// 열(SF)에서 끊겨, 그 너머에 저장된 값은 화면에서 볼 수 없었다.
test('columns past the five hundredth are reachable and editable', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`넓은 시트 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`wide-${stamp}`,cells:[
    {row:1,column:1,value:'첫 열'},{row:1,column:703,value:'AAA에 저장된 값'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const nameBox=page.getByLabel('이름 상자')
  await nameBox.fill('AAA1')
  await nameBox.press('Enter')
  // 예전에는 여기서 SF1에 멈췄다.
  await expect(nameBox).toHaveValue('AAA1')
  await expect(page.getByLabel('수식 입력창')).toHaveValue('AAA에 저장된 값')

  // 엑셀의 마지막 열까지 닿고, 그 너머는 잡아 둔다.
  await nameBox.fill('XFD1')
  await nameBox.press('Enter')
  await expect(nameBox).toHaveValue('XFD1')
  await nameBox.fill('XFE1')
  await nameBox.press('Enter')
  await expect(nameBox).toHaveValue('XFD1')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
