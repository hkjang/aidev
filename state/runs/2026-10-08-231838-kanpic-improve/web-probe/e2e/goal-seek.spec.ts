import { expect, test } from '@playwright/test'

// 스프레드시트는 대개 앞으로 계산하려고 만들지만, 정작 묻고 싶은 것은 거꾸로인
// 경우가 많다. "이 상환액이 되려면 이자율이 얼마여야 하나".
test('goal seek finds the input that reaches a value, and only writes when asked', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`목표값 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`loan-${stamp}`,cells:[
    {row:1,column:1,value:'원금'},{row:1,column:2,value:100000000},
    {row:2,column:1,value:'연이자율'},{row:2,column:2,value:0.05},
    {row:3,column:1,value:'기간'},{row:3,column:2,value:120},
    {row:4,column:1,value:'월 상환액'},{row:4,column:2,formula:'=PMT(B2/12,B3,-B1)'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'데이터',exact:true}).click()
  await page.getByRole('menu',{name:'데이터 메뉴'}).getByRole('menuitem',{name:'가정 분석'}).click()
  await page.getByRole('menuitem',{name:'목표값 찾기…'}).click()
  const dialog=page.getByRole('dialog',{name:'목표값 찾기'})
  await expect(dialog).toBeVisible()
  await dialog.getByLabel('수식 셀').fill('B4')
  await dialog.getByLabel('찾는 값').fill('900000')
  await dialog.getByLabel('바꿀 셀').fill('B2')
  await dialog.getByRole('button',{name:'찾기',exact:true}).click()
  await expect(dialog.getByRole('status')).toContainText('B2 →',{timeout:15000})

  // 찾기만 해서는 아무것도 쓰지 않는다. 답을 보고 나서 사람이 정한다.
  const untouched=await request.get(`/api/v1/sheets/${sheet}/ranges/B2:B2`).then(r=>r.json())
  expect(untouched.items[0].value).toBe(0.05)

  await dialog.getByRole('button',{name:'값 넣기'}).click()
  await expect(dialog.getByText('셀에 값을 넣었습니다.')).toBeVisible()
  await expect.poll(async()=>{
    const after=await request.get(`/api/v1/sheets/${sheet}/ranges/B2:B4`).then(r=>r.json())
    return after.items.find((cell:{row:number})=>cell.row===4)?.value
  },{timeout:10000}).toBeCloseTo(900000,0)

  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 답이 없는 물음에는 마지막으로 시도한 숫자를 답인 척 내밀면 안 된다.
test('goal seek says so when the value cannot be reached', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`불가능 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`sq-${stamp}`,cells:[
    {row:1,column:1,value:2},{row:2,column:1,formula:'=A1*A1'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'데이터',exact:true}).click()
  await page.getByRole('menu',{name:'데이터 메뉴'}).getByRole('menuitem',{name:'가정 분석'}).click()
  await page.getByRole('menuitem',{name:'목표값 찾기…'}).click()
  const dialog=page.getByRole('dialog',{name:'목표값 찾기'})
  await dialog.getByLabel('수식 셀').fill('A2')
  await dialog.getByLabel('찾는 값').fill('-100')
  await dialog.getByLabel('바꿀 셀').fill('A1')
  await dialog.getByRole('button',{name:'찾기',exact:true}).click()
  await expect(dialog.getByRole('status')).toContainText('목표에 이르지 못했습니다',{timeout:15000})
  // 실패했으면 넣을 것도 없다.
  await expect(dialog.getByRole('button',{name:'값 넣기'})).toHaveCount(0)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 메뉴가 길어지면 아래쪽 항목은 스크롤 뒤로 숨는다. 항목을 하나 더할 때마다
// 조금씩 다가가다가 어느 날 넘어가고, 그때 무엇이 밀려났는지는 아무도 모른다.
//
// 픽셀로 재지 않는 이유: 글꼴이 환경마다 달라 같은 항목 수라도 높이가 다르다.
// 이 시험이 처음에 그렇게 쓰여 CI에서만 걸렸다. 줄 수는 어디서나 같고, 막으려는
// 것도 결국 줄이 늘어나는 것이다. 스무 줄은 1280×800 창에 들어가는 한도로
// 재어 본 값이다.
const DATA_MENU_ROW_LIMIT=20

test('the data menu does not grow past what a window holds', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`메뉴 ${Date.now()}`}}).then(r=>r.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'데이터',exact:true}).click()
  const list=page.locator('.context-menu-list').first()
  await expect(list).toBeVisible()
  const rows=await list.evaluate(element=>element.querySelectorAll('[role=menuitem]').length)
  expect(rows).toBeGreaterThan(0)
  expect(rows).toBeLessThanOrEqual(DATA_MENU_ROW_LIMIT)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
