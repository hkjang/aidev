import { expect, test } from '@playwright/test'

// 수동 갱신 피벗은 만들어진 시점의 숫자를 계속 보여 준다. 그 숫자가 아직
// 유효한지 알 방법이 없으면, 오래된 합계를 최신으로 착각하게 된다.
test('a manually refreshed pivot says when its source has moved on', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`피벗 신선도 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','매출'],['서울',100],['부산',50],['서울',80]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`pf-seed-${stamp}`,cells}})
  const pivot=await request.post(`/api/v1/workbooks/${workbook.id}/pivots`,{data:{
    idempotency_key:`pf-pivot-${stamp}`,sheet_id:sheet,source_sheet_id:sheet,name:'지역별 매출',source_range:'A1:B4',
    rows:[{column:1}],values:[{column:2,aggregation:'sum'}],refresh_mode:'manual',
  }}).then(response=>response.json())
  const built=await request.get(`/api/v1/pivots/${pivot.id}/data`).then(response=>response.json())
  expect(built.stale).toBeFalsy()
  expect(built.grand_totals).toEqual([230])

  const current=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:current.version,idempotency_key:`pf-change-${stamp}`,cells:[{row:2,column:2,value:900}]}})
  const afterChange=await request.get(`/api/v1/pivots/${pivot.id}/data`).then(response=>response.json())
  // 캐시된 숫자는 그대로 두고, 오래되었다는 사실만 알린다.
  expect(afterChange.stale).toBe(true)
  expect(afterChange.grand_totals).toEqual([230])

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('button',{name:'피벗 패널'}).click()
  await page.getByTitle('결과 열기').first().click()
  const dialog=page.getByRole('dialog',{name:'피벗 결과'})
  await expect(dialog.getByText('원본이 바뀌었습니다 · 지금 갱신하면 반영됩니다')).toBeVisible()
  await dialog.getByRole('button',{name:'지금 갱신'}).click()
  await expect(dialog.getByText('원본이 바뀌었습니다 · 지금 갱신하면 반영됩니다')).toBeHidden()

  const refreshed=await request.get(`/api/v1/pivots/${pivot.id}/data`).then(response=>response.json())
  expect(refreshed.stale).toBeFalsy()
  expect(refreshed.grand_totals).toEqual([1030])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
