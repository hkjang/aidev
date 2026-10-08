import { expect, test } from '@playwright/test'

// 엑셀로 내보냈다가 다시 가져오면 시트 꼬리가 숨겨지고 메모가 사라졌다.
// 그리고 가져오기가 무엇을 버리는지 아무 말도 하지 않았다.
test('a workbook survives a round trip through XLSX, and the import says what it drops', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`왕복 ${stamp}`,workspace_id:'default'}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:1,idempotency_key:`rt-cells-${stamp}`,cells:[
    {row:1,column:1,value:'제품'},{row:1,column:3,value:1500},
    {row:2,column:1,value:'연필'},{row:2,column:3,value:3200},
  ]}})
  const seeded=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${sheet}/ranges:note`,{data:{base_version:seeded.version,idempotency_key:`rt-note-${stamp}`,range:'A2',note:'협력사 확정 단가'}})
  const noted=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  // 마지막으로 값이 쓰인 행보다 아래에 있는 병합. 여기서 꼬리가 숨겨졌다.
  await request.patch(`/api/v1/sheets/${sheet}/ranges:merge`,{data:{base_version:noted.version,idempotency_key:`rt-merge-${stamp}`,range:'A9:C9'}})
  await request.post(`/api/v1/workbooks/${workbook.id}/named-ranges`,{data:{idempotency_key:`rt-name-${stamp}`,sheet_id:sheet,name:'단가',range:'C1:C2'}})

  const exported=await request.post('/api/v1/exports',{data:{workbook_id:workbook.id,format:'xlsx'}})
  expect(exported.ok()).toBe(true)
  const bytes=await exported.body()

  await page.goto('/')
  await page.locator('input[type="file"]').setInputFiles({name:`왕복 ${stamp}.xlsx`,mimeType:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',buffer:bytes})
  const modal=page.locator('.import-modal')
  await expect(modal).toBeVisible()
  // kanpic이 다 읽을 수 있는 파일이므로 버린다고 경고할 것이 없어야 한다.
  await expect(modal.locator('.import-warnings')).toHaveCount(0)
  await modal.getByRole('button',{name:'워크북으로 가져오기'}).click()
  await expect(page.locator('.grid-canvas')).toBeVisible()

  const url=new URL(page.url())
  const importedId=url.pathname.split('/').pop()!
  const imported=await request.get(`/api/v1/workbooks/${importedId}`).then(response=>response.json())
  const importedSheet=imported.sheets[0]
  // 값이 없는 꼬리 행을 숨겨진 것으로 만들면 병합 셀이 화면에서 사라진다.
  expect(importedSheet.layout.hidden_rows??[]).toEqual([])
  const cells=await request.get(`/api/v1/sheets/${importedSheet.id}/ranges/A1:C2`).then(response=>response.json())
  expect(cells.items.find((cell:{row:number;column:number})=>cell.row===2&&cell.column===1)).toMatchObject({note:'협력사 확정 단가'})
  const names=await request.get(`/api/v1/workbooks/${importedId}/named-ranges`).then(response=>response.json())
  expect(names.items).toMatchObject([{name:'단가',range:'C1:C2',sheet_id:importedSheet.id}])

  await request.delete(`/api/v1/workbooks/${workbook.id}`)
  await request.delete(`/api/v1/workbooks/${importedId}`)
})
