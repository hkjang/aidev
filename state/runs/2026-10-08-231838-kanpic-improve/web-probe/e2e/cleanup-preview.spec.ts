import { expect, test } from '@playwright/test'

// 중복 삭제는 남은 행을 위로 당긴다. 선택한 열에서만 당기면 옆 열과 어긋나
// 데이터가 조용히 뒤섞인다. 미리보기는 그것부터 막아야 한다.
test('deduplicating a narrow selection offers to cover the whole table', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`중복 정리 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['이름','부서','점수'],['박지민','영업',90],['박지민','영업',90],['이서준','기획',75]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`cl-seed-${stamp}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1:A4')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'데이터 정리'}).click()
  await page.getByRole('menuitem',{name:'중복 항목 삭제…'}).click()

  const dialog=page.getByRole('dialog',{name:'중복 항목 삭제'})
  // 표보다 좁은 선택이므로 확장이 기본으로 켜지고 이유가 보인다.
  await expect(dialog.getByLabel('표 전체로 확장')).toBeChecked()
  await expect(dialog.getByText('선택한 열만 정리하면 남은 행이 위로 올라가면서 옆 열과 어긋납니다.')).toBeVisible()
  await expect(dialog.locator('.cleanup-preview')).toContainText('박지민 · 영업 · 90')
  await expect(dialog.locator('.cleanup-summary')).toContainText('중복된 1개 행을 삭제합니다.')
  await dialog.getByRole('button',{name:'삭제'}).click()

  // 세 열이 함께 당겨져 행이 어긋나지 않는다.
  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A1:C4`).then(response=>response.json())
    const at=(row:number,column:number)=>range.items.find((cell:{row:number;column:number})=>cell.row===row&&cell.column===column)?.value
    return [at(2,1),at(2,2),at(2,3),at(3,1),at(3,2),at(3,3),at(4,1)]
  },{timeout:15_000}).toEqual(['박지민','영업',90,'이서준','기획',75,undefined])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

test('trimming whitespace shows what it will change before it changes it', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`공백 정리 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`tr-seed-${stamp}`,cells:[
    {row:1,column:1,value:'  서울   본사 '},{row:2,column:1,value:'부산'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1:A2')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'데이터 정리'}).click()
  await page.getByRole('menuitem',{name:'공백 제거…'}).click()

  const dialog=page.getByRole('dialog',{name:'공백 제거'})
  await expect(dialog.locator('.cleanup-preview')).toContainText('서울 본사')
  await expect(dialog.locator('.cleanup-summary')).toContainText('1개 셀의 공백을 정리합니다.')
  await dialog.getByRole('button',{name:'정리'}).click()
  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A1`).then(response=>response.json())
    return range.items[0]?.value
  },{timeout:15_000}).toBe('서울 본사')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
