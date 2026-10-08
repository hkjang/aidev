import { expect, test, type APIRequestContext } from '@playwright/test'

const seed=async (request:APIRequestContext,title:string,cells:Array<{row:number;column:number;value?:unknown;formula?:string}>)=>{
  const workbook=await request.post('/api/v1/workbooks',{data:{title}}).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${workbook.sheets[0].id}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}-${title}`,cells}})
  return workbook
}

// The running total of a selection is the number people look at most, so it
// has to appear as soon as more than one cell is selected.
test('selecting a range shows its total, average and count', async ({ page, request }) => {
  const workbook=await seed(request,`요약 ${Date.now()}`,[
    {row:1,column:1,value:1200},{row:2,column:1,value:800},{row:3,column:1,value:'미정'},{row:4,column:1,value:2000},
  ])
  await page.goto(`/workbooks/${workbook.id}`)
  const summary=page.getByLabel('선택 범위 요약')
  // A single cell says nothing the grid does not already show.
  await expect(summary).toBeHidden()

  // 범위 확장 단축키는 시트에 이미 들어온 값으로 끝을 찾는다. 셀이 도착하기
  // 전에 누르면 선택이 짧게 끝나므로, 값이 왔다는 것을 먼저 확인한다.
  await page.getByLabel('A1 셀 입력').press('Shift+ArrowDown')
  await expect(summary).toContainText('합계 2,000')
  await page.keyboard.press('Control+Shift+ArrowDown')
  await expect(summary).toContainText('합계 4,000')
  await expect(summary).toContainText('평균 1,333.33')
  await expect(summary).toContainText('개수 4')

  // The choice of statistics is the reader's, and it sticks.
  await summary.click()
  const maximum=page.getByRole('menuitemcheckbox',{name:'최대'})
  await expect(maximum).toBeVisible()
  await maximum.click()
  await expect(summary).toContainText('최대 2,000')
  await page.reload()
  await page.getByLabel('A1 셀 입력').press('Shift+ArrowDown')
  await expect(summary).toContainText('최대 1,200')
  await page.keyboard.press('Control+Shift+ArrowDown')
  await expect(summary).toContainText('최대 2,000')
})

// Typing the same entry again is the most common thing anybody does in a
// column, so the column's own values are offered.
test('typing in a column suggests the entries already used there', async ({ page, request }) => {
  const workbook=await seed(request,`열 제안 ${Date.now()}`,[
    {row:1,column:1,value:'영업본부'},{row:2,column:1,value:'개발본부'},{row:3,column:1,value:'영업본부'},
  ])
  await page.goto(`/workbooks/${workbook.id}`)
  // 데이터 영역 끝으로 가는 단축키는 이미 들어온 값으로 끝을 찾는다. 값이
  // 도착하기 전에 누르면 시트 맨 아래로 가 버리므로 이름 상자로 옮긴다.
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A4')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  const editor=page.getByLabel('A4 셀 입력')
  await editor.fill('영업')
  const suggestions=page.locator('.value-suggest')
  await expect(suggestions.getByRole('option')).toHaveText(['영업본부'])
  await editor.press('Tab')
  await expect(page.getByLabel('A4 셀 입력')).toHaveValue('영업본부')

  // A formula gets function help instead, not column values.
  await page.getByLabel('A4 셀 입력').fill('=SUMI')
  await expect(page.locator('.formula-suggest')).toBeVisible()
  await expect(suggestions).toBeHidden()
})
