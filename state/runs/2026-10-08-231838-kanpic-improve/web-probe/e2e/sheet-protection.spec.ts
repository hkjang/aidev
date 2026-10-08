import { expect, test } from '@playwright/test'

// 확정된 모델을 남에게 넘길 때 필요한 것은 "시트 전체 잠금 + 입력 칸만 열기"다.
// 범위 하나씩 보호해서는 만들 수 없는 형태다.
test('a sheet protection locks everything except the ranges it names', async ({ page, request }) => {
  const stamp=Date.now()
  const mate=`protect-mate-${stamp}`
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`시트 보호 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.put(`/api/v1/workbooks/${workbook.id}/shares`,{data:{principal_type:'user',principal_id:mate,role:'editor'}})
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`sp-seed-${stamp}`,cells:[
    {row:1,column:1,value:'요율'},{row:1,column:2,value:0.15},{row:3,column:2,value:'여기에 입력'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'범위 보호…'}).click()
  const dialog=page.getByRole('dialog',{name:'범위 보호'})
  await dialog.getByLabel('보호 대상').selectOption('sheet')
  await dialog.getByLabel('편집 허용 범위').fill('B3:B6')
  await dialog.getByLabel('보호 설명').fill('확정된 요율 모델')
  await dialog.getByRole('button',{name:'이 시트 보호'}).click()
  await expect(dialog.locator('.protected-row strong').filter({hasText:'시트 전체'})).toBeVisible()
  await expect(dialog.getByText(/예외 B3:B6/)).toBeVisible()

  // 협업자는 예외 안에서만 쓸 수 있어야 한다.
  const latest=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  const allowed=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{headers:{'X-Kanpic-Actor':mate},data:{base_version:latest.version,idempotency_key:`sp-in-${stamp}`,cells:[{row:4,column:2,value:12}]}})
  expect(allowed.ok()).toBe(true)
  const current=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  const refused=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{headers:{'X-Kanpic-Actor':mate},data:{base_version:current.version,idempotency_key:`sp-out-${stamp}`,cells:[{row:1,column:2,value:0.99}]}})
  expect(refused.status()).toBe(403)
  expect(await refused.text()).toContain('확정된 요율 모델')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
