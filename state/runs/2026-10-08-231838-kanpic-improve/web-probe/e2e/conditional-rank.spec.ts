import { expect, test } from '@playwright/test'

// "상위 10개 항목" 은 엑셀 조건부 서식에서 가장 많이 쓰는 규칙 축에 든다.
// 한 칸만 봐서는 답할 수 없어 kanpic 에는 없던 종류다.
test('a top-N rule highlights the largest values in its range', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`상위 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`rank-${stamp}`,cells:[
    {row:1,column:1,value:10},{row:2,column:1,value:50},{row:3,column:1,value:30},
    {row:4,column:1,value:50},{row:5,column:1,value:20},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('button',{name:'조건부 서식'}).click()
  const dialog=page.getByRole('dialog',{name:'조건부 서식'})
  await expect(dialog).toBeVisible()
  await dialog.getByLabel('조건부 서식 범위').fill('A1:A5')
  await dialog.getByLabel('조건부 서식 유형').selectOption('rank')
  await dialog.getByLabel('순위 기준').selectOption('top')
  await dialog.getByLabel('순위 개수').fill('2')
  await dialog.getByRole('button',{name:/저장|추가|만들기/}).first().click()

  await expect.poll(async()=>{
    const evaluated=await request.get(`/api/v1/sheets/${sheet}/conditional-formats:evaluate?range=A1:A5`).then(r=>r.json())
    return (evaluated.items??[]).map((item:{row:number})=>item.row).sort((a:number,b:number)=>a-b)
  // 50 이 둘이므로 상위 2개는 그 둘이다.
  }).toEqual([2,4])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
