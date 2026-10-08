import { expect, test } from '@playwright/test'

// 매출과 이익률처럼 단위가 다른 계열을 한 차트에 담으려면 막대와 선을 섞고
// 이익률을 오른쪽 축에 붙여야 읽을 수 있다.
test('a combo chart puts the last series on a line and an optional secondary axis', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`혼합 차트 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['분기','매출','비용','이익률'],['Q1',120,70,0.42],['Q2',150,90,0.4],['Q3',180,95,0.47]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`combo-seed-${workbook.id}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1:D4')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('button',{name:'차트 패널'}).click()
  await page.getByRole('button',{name:'새 차트'}).click()
  const dialog=page.getByRole('dialog',{name:'차트 만들기'})
  await dialog.getByLabel('차트 제목').fill('매출과 이익률')
  await expect(dialog.getByLabel('보조 축 사용')).toBeHidden()
  await dialog.getByLabel('차트 유형').selectOption('combo')
  await dialog.getByLabel('보조 축 사용').check()
  await dialog.getByRole('button',{name:'차트 저장'}).click()

  const card=page.locator('[data-chart-id]')
  await expect(card.getByRole('img',{name:'매출과 이익률'})).toBeVisible()
  const chartId=await card.getAttribute('data-chart-id')
  const saved=await request.get(`/api/v1/charts/${chartId}`).then(response=>response.json())
  expect(saved).toMatchObject({type:'combo',secondary_axis:true})

  // 유형을 누적 막대로 바꾸면 보조 축은 의미가 없어지므로 서버가 스스로 끈다.
  await request.patch(`/api/v1/charts/${chartId}`,{data:{type:'stacked_bar'}})
  const restacked=await request.get(`/api/v1/charts/${chartId}`).then(response=>response.json())
  expect(restacked).toMatchObject({type:'stacked_bar',secondary_axis:false})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
