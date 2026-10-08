import { expect, test } from '@playwright/test'

const actor=(name:string)=>({'X-Kanpic-Actor':name})

// IMPORTRANGE는 한 워크북의 데이터를 다른 워크북에 넘긴다. 그래서 흥미로운
// 것은 성공보다 거절이다: 가져오는 워크북의 소유자가 원본을 읽을 수 있어야
// 하고, 그렇지 않으면 빈 값이 아니라 이유를 말해야 한다.
test('IMPORTRANGE pulls another workbook only when the owner may read it', async ({ request }) => {
  const stamp=Date.now()
  const bob=`import-bob-${stamp}`,alice=`import-alice-${stamp}`
  const source=await request.post('/api/v1/workbooks',{headers:actor(bob),data:{title:`원본 ${stamp}`}}).then(response=>response.json())
  const sourceSheet=source.sheets[0].id
  await request.patch(`/api/v1/sheets/${sourceSheet}/cells:batch`,{headers:actor(bob),data:{base_version:source.version,idempotency_key:`import-source-${stamp}`,cells:[
    {row:1,column:1,value:10},{row:2,column:1,value:20},{row:3,column:1,value:30},
  ]}})

  const report=await request.post('/api/v1/workbooks',{headers:actor(alice),data:{title:`보고서 ${stamp}`}}).then(response=>response.json())
  const reportSheet=report.sheets[0].id
  const applied=await request.patch(`/api/v1/sheets/${reportSheet}/cells:batch`,{headers:actor(alice),data:{base_version:report.version,idempotency_key:`import-formula-${stamp}`,cells:[
    {row:1,column:1,formula:`=SUM(IMPORTRANGE("${source.id}","Sheet1!A1:A3"))`},
  ]}}).then(response=>response.json())
  expect(applied.formula_errors?.[0]?.message).toContain('읽기 권한')

  // 원본을 공유하면 다음 계산에서 값이 들어온다. IMPORTRANGE는 휘발성이므로
  // 관계없는 셀 하나를 고치는 것만으로 다시 계산된다.
  await request.put(`/api/v1/workbooks/${source.id}/shares`,{headers:actor(bob),data:{principal_type:'user',principal_id:alice,role:'viewer'}})
  const current=await request.get(`/api/v1/workbooks/${report.id}`,{headers:actor(alice)}).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${reportSheet}/cells:batch`,{headers:actor(alice),data:{base_version:current.version,idempotency_key:`import-touch-${stamp}`,cells:[{row:5,column:1,value:1}]}})
  const range=await request.get(`/api/v1/sheets/${reportSheet}/ranges/A1:A1`,{headers:actor(alice)}).then(response=>response.json())
  expect(range.items[0].value).toBe(60)

  // 원본이 바뀌면 가져온 쪽도 다음 계산에서 따라온다.
  const sourceNow=await request.get(`/api/v1/workbooks/${source.id}`,{headers:actor(bob)}).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${sourceSheet}/cells:batch`,{headers:actor(bob),data:{base_version:sourceNow.version,idempotency_key:`import-change-${stamp}`,cells:[{row:1,column:1,value:100}]}})
  const reportNow=await request.get(`/api/v1/workbooks/${report.id}`,{headers:actor(alice)}).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${reportSheet}/cells:batch`,{headers:actor(alice),data:{base_version:reportNow.version,idempotency_key:`import-touch2-${stamp}`,cells:[{row:6,column:1,value:1}]}})
  const refreshed=await request.get(`/api/v1/sheets/${reportSheet}/ranges/A1:A1`,{headers:actor(alice)}).then(response=>response.json())
  expect(refreshed.items[0].value).toBe(150)

  await request.delete(`/api/v1/workbooks/${report.id}`,{headers:actor(alice)})
  await request.delete(`/api/v1/workbooks/${source.id}`,{headers:actor(bob)})
})
