import { expect, test } from '@playwright/test'

// 월 이름을 글자 그대로 정렬하면 10월과 12월이 1월보다 앞에 온다. 그렇게
// 정렬하고 싶은 사람은 없다.
test('sorting month names puts them in month order, and the dialog can turn that off', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`정렬 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`sort-${stamp}`,cells:[
    {row:1,column:1,value:'2월'},{row:2,column:1,value:'10월'},{row:3,column:1,value:'1월'},
    {row:4,column:1,value:'12월'},{row:5,column:1,value:'3월'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const months=async()=>(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A5`).then(r=>r.json())).items.map((cell:{value:unknown})=>cell.value)

  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1:A5')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('button',{name:'범위 정렬'}).click()
  const dialog=page.getByRole('dialog',{name:'범위 정렬'})
  await expect(dialog).toBeVisible()
  await dialog.getByRole('checkbox',{name:'첫 행을 헤더로 고정'}).uncheck()
  await dialog.getByRole('button',{name:'정렬 적용'}).click()
  await expect.poll(months).toEqual(['1월','2월','3월','10월','12월'])

  // 글자 그대로 견주고 싶은 사람은 끌 수 있어야 한다.
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1:A5')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('button',{name:'범위 정렬'}).click()
  await expect(dialog).toBeVisible()
  await dialog.getByRole('checkbox',{name:'첫 행을 헤더로 고정'}).uncheck()
  await dialog.getByRole('checkbox',{name:'숫자를 글자로 취급'}).check()
  await dialog.getByRole('button',{name:'정렬 적용'}).click()
  await expect.poll(months).toEqual(['10월','12월','1월','2월','3월'])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 정렬은 두 번 일어난다. 먼저 화면이 자기 계산으로 줄을 옮겨 보여주고,
// 곧이어 서버가 보낸 결과가 그 위에 덮인다. 두 계산이 다른 답을 내면 줄이
// 눈앞에서 한 번 튄다.
//
// 자바스크립트의 기본 문자열 비교는 UTF-16 조각을 견주므로 이모지를
// ￦(U+FFE6) 보다 앞에 놓지만, 서버는 UTF-8 바이트를 견주어 뒤에 놓는다.
// 아래는 서버 응답을 붙잡아 둔 채 화면이 먼저 그린 차례를 읽고, 응답을
// 풀어준 뒤 다시 읽어 두 차례가 같은지 본다.
test('the row the browser draws is the row the server confirms', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`이모지 정렬 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`emoji-sort-${stamp}`,cells:[
    {row:1,column:1,value:'￦100'},{row:2,column:1,value:'😀항목'},
    {row:3,column:1,value:'＀'},{row:4,column:1,value:'🍎사과'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  // 화면이 그린 값은 캔버스에 있으므로 칸을 하나씩 골라 수식 입력줄에서 읽는다.
  const nameBox=page.getByRole('combobox',{name:'이름 상자'})
  const formulaBar=page.getByRole('textbox',{name:'수식 입력창'})
  const drawn=async()=>{
    const values:string[]=[]
    for(const row of [1,2,3,4]){
      await nameBox.fill(`A${row}`)
      await nameBox.press('Enter')
      values.push(await formulaBar.inputValue())
    }
    return values
  }

  // 서버 응답을 붙잡아 둔다. 풀어주기 전까지 화면에는 자기 계산만 남는다.
  let release=()=>{}
  const held=new Promise<void>(resolve=>{release=resolve})
  await page.route(`**/sheets/${sheet}/ranges:sort`,async route=>{await held;await route.continue()})

  await nameBox.fill('A1:A4')
  await nameBox.press('Enter')
  await page.getByRole('button',{name:'범위 정렬'}).click()
  const dialog=page.getByRole('dialog',{name:'범위 정렬'})
  await expect(dialog).toBeVisible()
  await dialog.getByRole('checkbox',{name:'첫 행을 헤더로 고정'}).uncheck()
  await dialog.getByRole('button',{name:'정렬 적용'}).click()

  const optimistic=await drawn()
  release()
  await expect.poll(async()=>(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A4`).then(r=>r.json())).items.map((cell:{value:unknown})=>cell.value))
    .toEqual(['＀','￦100','🍎사과','😀항목'])
  await expect.poll(drawn).toEqual(['＀','￦100','🍎사과','😀항목'])
  // 붙잡아 두었을 때 화면이 그린 차례가 서버가 확정한 차례와 같아야 한다.
  expect(optimistic).toEqual(['＀','￦100','🍎사과','😀항목'])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
