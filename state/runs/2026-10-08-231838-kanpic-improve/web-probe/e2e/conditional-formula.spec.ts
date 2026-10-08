import { expect, test } from '@playwright/test'

// 맞춤 수식 규칙은 규칙이 덮지 않는 열의 값으로 행 전체를 칠할 수 있어야
// 한다. 그것이 구글 시트에서 조건부 서식을 실제로 쓰게 만드는 형태다.
test('a custom formula rule highlights whole rows from another column', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`맞춤 수식 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['업무','상태'],['견적 검토','완료'],['계약 작성','진행'],['입금 확인','완료']]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`cf-seed-${workbook.id}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A2:A4')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('button',{name:'조건부 서식'}).click()
  const dialog=page.getByRole('dialog',{name:'조건부 서식'})
  await dialog.getByLabel('조건부 서식 유형').selectOption('custom_formula')
  await dialog.getByLabel('조건부 서식 맞춤 수식').fill('=$B2="완료"')
  await dialog.getByRole('button',{name:'규칙 저장'}).click()

  // 저장된 규칙과 서버 평가 결과가 같은 셀을 가리켜야 한다.
  await expect.poll(async()=>{
    const rules=await request.get(`/api/v1/sheets/${sheet}/conditional-formats`).then(response=>response.json())
    return rules.items[0]?.formula
  },{timeout:15_000}).toBe('=$B2="완료"')
  const evaluation=await request.get(`/api/v1/sheets/${sheet}/conditional-formats:evaluate?range=A1:B4`).then(response=>response.json())
  const matched=evaluation.items.filter((item:{style?:Record<string,unknown>})=>item.style).map((item:{row:number;column:number})=>`${item.row}:${item.column}`)
  expect(matched.sort()).toEqual(['2:1','4:1'])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
