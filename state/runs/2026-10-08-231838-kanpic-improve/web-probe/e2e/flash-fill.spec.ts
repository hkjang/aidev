import { expect, test } from '@playwright/test'

// 손으로 한 줄 채워 놓으면 나머지를 같은 규칙으로 채운다. 스프레드시트에서
// 손으로 가장 많이 반복되는 일이다.
test('one filled cell teaches the rest of the column', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`빠른 채우기 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`ff-${stamp}`,cells:[
    {row:1,column:1,value:'이메일'},{row:1,column:2,value:'아이디'},
    {row:2,column:1,value:'hong@example.com'},{row:2,column:2,value:'hong'},
    {row:3,column:1,value:'kim@sample.co.kr'},
    {row:4,column:1,value:'lee.min@x.io'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('B2')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'데이터',exact:true}).click()
  await page.getByRole('menu',{name:'데이터 메뉴'}).getByRole('menuitem',{name:'데이터 정리'}).click()
  await page.getByRole('menuitem',{name:'빠른 채우기…'}).click()

  const dialog=page.getByRole('dialog',{name:'빠른 채우기'})
  await expect(dialog).toBeVisible()
  // 무엇을 쓸지 보고 나서 정한다.
  await expect(dialog.getByText('kim')).toBeVisible()
  await expect(dialog.getByText('lee.min')).toBeVisible()

  // 누르기 전에는 아무것도 쓰지 않는다.
  const before=await request.get(`/api/v1/sheets/${sheet}/ranges/B3:B4`).then(r=>r.json())
  expect(before.items).toHaveLength(0)

  await dialog.getByRole('button',{name:/칸 채우기/}).click()
  await expect(dialog).toHaveCount(0)
  await expect.poll(async()=>{
    const after=await request.get(`/api/v1/sheets/${sheet}/ranges/B3:B4`).then(r=>r.json())
    return after.items.map((cell:{row:number;value:unknown})=>`${cell.row}:${cell.value}`).sort()
  },{timeout:10000}).toEqual(['3:kim','4:lee.min'])

  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 규칙을 못 찾으면 아무것도 하지 않고 왜 못 했는지 말한다. 반쯤 맞는 규칙으로
// 수백 줄을 채우는 것이 이 기능이 할 수 있는 가장 나쁜 일이다.
test('it says so rather than guessing', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`규칙 없음 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`ff2-${stamp}`,cells:[
    {row:1,column:1,value:'값'},{row:1,column:2,value:'결과'},
    {row:2,column:1,value:'hong@example.com'},{row:2,column:2,value:'아무 상관 없는 말'},
    {row:3,column:1,value:'kim@sample.co.kr'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const messages:string[]=[]
  page.on('dialog',dialog=>{messages.push(dialog.message());void dialog.accept()})
  await page.getByRole('combobox',{name:'이름 상자'}).fill('B2')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'데이터',exact:true}).click()
  await page.getByRole('menu',{name:'데이터 메뉴'}).getByRole('menuitem',{name:'데이터 정리'}).click()
  await page.getByRole('menuitem',{name:'빠른 채우기…'}).click()

  await expect.poll(()=>messages.length,{timeout:10000}).toBeGreaterThan(0)
  expect(messages[0]).toContain('규칙을 찾지 못했습니다')
  await expect(page.getByRole('dialog',{name:'빠른 채우기'})).toHaveCount(0)
  // 아무것도 쓰지 않았다.
  const after=await request.get(`/api/v1/sheets/${sheet}/ranges/B3:B3`).then(r=>r.json())
  expect(after.items).toHaveLength(0)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 대상 열만 골라 놓는 것이 가장 자연스러운 몸짓이다. 그러면 고른 것이 그
// 열뿐이라 규칙을 만들 재료가 없어지므로, 옆 열들을 표에서 찾아 붙인다.
test('selecting just the column still finds the columns it reads from', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`열 선택 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`col-${stamp}`,cells:[
    {row:1,column:1,value:'이메일'},{row:1,column:2,value:'아이디'},
    {row:2,column:1,value:'hong@example.com'},{row:2,column:2,value:'hong'},
    {row:3,column:1,value:'kim@sample.co.kr'},
    {row:4,column:1,value:'lee@x.io'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('B2:B4')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'데이터',exact:true}).click()
  await page.getByRole('menu',{name:'데이터 메뉴'}).getByRole('menuitem',{name:'데이터 정리'}).click()
  await page.getByRole('menuitem',{name:'빠른 채우기…'}).click()

  const dialog=page.getByRole('dialog',{name:'빠른 채우기'})
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('kim')).toBeVisible()
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 엑셀을 쓰던 사람은 Ctrl+E 를 누른다.
test('Ctrl+E reaches for it the way Excel does', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`단축키 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`key-${stamp}`,cells:[
    {row:1,column:1,value:'hong@example.com'},{row:1,column:2,value:'hong'},
    {row:2,column:1,value:'kim@sample.co.kr'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('B1')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.locator('.grid-canvas').click({position:{x:200,y:60}})
  await page.getByRole('combobox',{name:'이름 상자'}).fill('B1')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.keyboard.press('Control+e')
  await expect(page.getByRole('dialog',{name:'빠른 채우기'})).toBeVisible({timeout:10000})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 사번과 상품코드는 늘 자릿수를 맞춰 쓴다.
test('it learns to pad a number to a fixed width', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`자릿수 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`pad-${stamp}`,cells:[
    {row:1,column:1,value:'번호'},{row:1,column:2,value:'사번'},
    {row:2,column:1,value:'7'},{row:2,column:2,value:'007'},
    {row:3,column:1,value:'12'},{row:3,column:2,value:'012'},
    {row:4,column:1,value:'5'},
    {row:5,column:1,value:'123'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('B4')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.keyboard.press('Control+e')
  const dialog=page.getByRole('dialog',{name:'빠른 채우기'})
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button',{name:/칸 채우기/}).click()
  await expect.poll(async()=>{
    const after=await request.get(`/api/v1/sheets/${sheet}/ranges/B4:B5`).then(r=>r.json())
    return after.items.map((cell:{row:number;value:unknown})=>`${cell.row}:${cell.value}`).sort()
  },{timeout:10000}).toEqual(['4:005','5:123'])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
