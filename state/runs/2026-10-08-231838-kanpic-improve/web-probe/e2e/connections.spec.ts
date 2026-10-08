import { expect, test } from '@playwright/test'

// 가져온 데이터는 남의 워크북에 있다. 그래서 읽는 사람이 늘 묻는 두 가지는
// "이 값이 최신인가"와 "아직 볼 수 있는가"다. 연결 패널이 둘 다 답해야 한다.
test('the connection panel lists imports and refreshes them on demand', async ({ page, request }) => {
  const stamp=Date.now()
  const source=await request.post('/api/v1/workbooks',{data:{title:`매출 원장 ${stamp}`}}).then(response=>response.json())
  const sourceSheet=source.sheets[0].id
  await request.patch(`/api/v1/sheets/${sourceSheet}/cells:batch`,{data:{base_version:source.version,idempotency_key:`conn-source-${stamp}`,cells:[
    {row:1,column:1,value:120},{row:2,column:1,value:150},{row:3,column:1,value:180},
  ]}})
  const report=await request.post('/api/v1/workbooks',{data:{title:`보고서 ${stamp}`}}).then(response=>response.json())
  const reportSheet=report.sheets[0].id
  await request.patch(`/api/v1/sheets/${reportSheet}/cells:batch`,{data:{base_version:report.version,idempotency_key:`conn-report-${stamp}`,cells:[
    {row:1,column:1,formula:`=SUM(IMPORTRANGE("${source.id}","Sheet1!A1:A3"))`},
    {row:3,column:1,formula:`=IMPORTRANGE("없는-워크북","A1:A2")`},
  ]}})

  await page.goto(`/workbooks/${report.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'데이터 연결…'}).click()
  const panel=page.getByRole('region',{name:'데이터 연결'})
  await expect(panel.getByText(`매출 원장 ${stamp}`)).toBeVisible()
  await expect(panel.getByText('원본 워크북을 찾을 수 없습니다')).toBeVisible()
  await expect(panel.getByText('1개 연결에 문제가 있습니다')).toBeVisible()

  // 원본이 바뀐 뒤 새로 고침을 누르면 가져온 값이 따라와야 한다.
  const sourceNow=await request.get(`/api/v1/workbooks/${source.id}`).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${sourceSheet}/cells:batch`,{data:{base_version:sourceNow.version,idempotency_key:`conn-change-${stamp}`,cells:[{row:1,column:1,value:1000}]}})
  await panel.getByRole('button',{name:'연결 새로 고침'}).click()
  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${reportSheet}/ranges/A1:A1`).then(response=>response.json())
    return range.items[0]?.value
  },{timeout:15_000}).toBe(1330)
  await expect(panel.getByText(/갱신$/)).toBeVisible()

  await request.delete(`/api/v1/workbooks/${report.id}`)
  await request.delete(`/api/v1/workbooks/${source.id}`)
})
