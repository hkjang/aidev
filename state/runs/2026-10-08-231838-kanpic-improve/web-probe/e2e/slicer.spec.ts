import { expect, test } from '@playwright/test'

// 보고서를 읽는 사람은 필터 메뉴가 어디 있는지 모른다. 슬라이서는 시트 위에
// 놓인 버튼 묶음이라 누르기만 하면 되고, 누른 결과는 모두가 공유하는 필터에
// 그대로 반영되어야 한다.
test('a slicer filters the sheet and survives a column move', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`슬라이서 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','제품','매출'],['서울','연필',120],['부산','공책',80],['서울','공책',150],['대구','연필',60]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`slicer-seed-${workbook.id}`,cells}})
  await request.post(`/api/v1/sheets/${sheet}/filter-views`,{data:{idempotency_key:`slicer-view-${workbook.id}`,name:'매출 필터',range:'A1:C5',header_rows:1,active:true}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A2')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'슬라이서 추가'}).click()

  const card=page.locator('[data-slicer-id]')
  await expect(card).toBeVisible()
  await expect(card.getByRole('button',{name:/서울/})).toHaveAttribute('aria-pressed','true')
  await expect(card.locator('footer')).toHaveText('모든 값 표시 중')
  await card.getByRole('button',{name:/서울/}).click()
  await expect(card.locator('footer')).toHaveText('2/3개 값 표시 중')

  // 슬라이서가 손대는 것은 공용 필터 보기이므로, 걸러진 행 수가 서버에서도 같아야 한다.
  const views=await request.get(`/api/v1/sheets/${sheet}/filter-views`).then(response=>response.json())
  const view=views.items[0]
  expect(view.criteria.find((criterion:{column:number})=>criterion.column===1).operator).toBe('values')
  const evaluated=await request.post(`/api/v1/filter-views/${view.id}:evaluate`,{data:{}}).then(response=>response.json())
  expect(evaluated.visible_count).toBe(2)

  // A열을 옮겨도 슬라이서는 같은 열을 계속 가리켜야 한다.
  const latest=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${sheet}/structure:apply`,{data:{base_version:latest.version,idempotency_key:`slicer-move-${workbook.id}`,axis:'column',action:'move',index:1,count:1,destination:3}})
  const moved=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  expect(moved.sheets[0].layout.slicers[0].column).toBe(2)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
