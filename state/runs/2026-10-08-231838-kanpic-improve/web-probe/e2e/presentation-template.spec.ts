import { expect, test } from '@playwright/test'

// 디자인 목록은 쉰 가지다. 한 줄로 늘어서면 고르기가 아니라 훑기가 되고,
// 결국 아무도 바꾸지 않아 만드는 덱이 모두 같은 모습이 된다.
//
// 이름이 "Ptium <색> <배치>" 이므로 색으로 묶어 준다. 그리고 한 번 고른
// 것을 기억한다 — 기억하지 않으면 열 때마다 "기본 디자인" 으로 돌아가고,
// 그러면 아무것도 보내지 않아 서비스 기본값으로 만들어진다.
test('the presentation dialog groups designs and remembers the last choice', async ({ page, request }) => {
  const config=await request.get('/api/v1/presentation/config').then(r=>r.json())
  test.skip(!config.enabled,'프레젠테이션 설정이 꺼져 있습니다')
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`디자인 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`tpl-${stamp}`,cells:[
    {row:1,column:1,value:'부서'},{row:1,column:2,value:'매출'},
    {row:2,column:1,value:'영업'},{row:2,column:2,value:100},
    {row:3,column:1,value:'개발'},{row:3,column:2,value:80},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1:B3')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'프레젠테이션'}).click()
  await page.getByRole('menuitem',{name:'프레젠테이션 만들기…'}).click()

  const picker=page.getByRole('combobox',{name:'프레젠테이션 템플릿'})
  await expect(picker).toBeVisible()

  // 색 묶음으로 나뉘어 있어야 한다. 목록은 프레젠테이션 서비스에서 오므로
  // 도착할 때까지 기다린다. evaluateAll 은 기다려 주지 않아, 곧바로 부르면
  // 아직 비어 있는 목록을 재고 만다.
  await expect(picker.locator('optgroup').first()).toBeAttached({timeout:15_000})
  const families=await picker.locator('optgroup').evaluateAll(groups=>groups.map(g=>(g as HTMLOptGroupElement).label))
  expect(families.length).toBeGreaterThan(1)

  // 고른 값이 다음에 열 때도 남아 있어야 한다.
  const chosen=await picker.locator('optgroup option').first().getAttribute('value')
  await picker.selectOption(chosen!)
  await page.getByRole('button',{name:'프레젠테이션 닫기'}).click()

  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'프레젠테이션'}).click()
  await page.getByRole('menuitem',{name:'프레젠테이션 만들기…'}).click()
  await expect(page.getByRole('combobox',{name:'프레젠테이션 템플릿'})).toHaveValue(chosen!)

  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
