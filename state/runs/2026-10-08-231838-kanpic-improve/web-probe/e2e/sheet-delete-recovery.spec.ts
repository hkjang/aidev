import { expect, test } from '@playwright/test'

// 시트 삭제는 그 안의 모든 셀을 버리고, 행 삭제와 달리 셀 단위로 되돌릴
// 수도 없다. 지금까지는 회수 경로가 아예 없었다.
test('a deleted sheet can be brought back with everything in it', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`시트 ${stamp}`}}).then(response=>response.json())
  const detail=await request.post(`/api/v1/workbooks/${workbook.id}/sheets`,{data:{name:'상세'}}).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${detail.id}/cells:batch`,{data:{idempotency_key:`sd-${stamp}`,cells:[
    {row:1,column:1,value:'중요한 값'},{row:2,column:1,value:1234},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  page.on('dialog',dialog=>void dialog.accept())
  await page.getByRole('button',{name:'상세',exact:true}).click({button:'right'})
  await page.getByRole('menuitem',{name:/삭제/}).first().click()

  const notice=page.locator('.formula-issue')
  await expect(notice).toContainText('시트 상세을(를) 삭제했습니다')
  await expect.poll(async()=>{
    const book=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
    return book.sheets.length
  }).toBe(1)

  await notice.getByRole('button',{name:'되돌리기'}).click()
  // 탭만 돌아오면 소용이 없다. 그 안에 있던 것이 함께 돌아와야 한다.
  await expect.poll(async()=>{
    const body=await request.get(`/api/v1/sheets/${detail.id}/ranges/A1:A2`).then(response=>response.json())
    const items=(body.items??[]) as Array<{value?:unknown}>
    return items.map(item=>item.value)
  }).toEqual(['중요한 값',1234])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
