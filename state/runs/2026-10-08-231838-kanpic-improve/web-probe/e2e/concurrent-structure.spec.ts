import { expect, test } from '@playwright/test'

// 두 사람이 동시에 일한다. 한 명이 행을 지우는 사이 다른 한 명이 그 아래
// 셀을 저장하면, 저장 요청에 담긴 행 번호는 삭제 전의 것이다. 그대로
// 적용하면 값이 한 줄 아래에 찍히고 아무도 건드리지 않은 값이 지워진다.
test('a cell saved during someone else\'s row delete follows its row', async ({ request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`경합 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`seed-${stamp}`,
    cells:[1,2,3,4,5,6].map(row=>({row,column:1,value:`행${row}`}))}})
  const read=async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A6`).then(response=>response.json())).items as Array<{row:number;value?:unknown}>
    return new Map(items.map(item=>[item.row,item.value]))
  }
  // B가 읽은 시점의 버전. 이 시점에 B의 화면에서 A5는 행5다.
  const seen=(await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())).version
  await request.patch(`/api/v1/sheets/${sheet}/structure:apply`,{data:{axis:'row',action:'delete',index:3,count:1,base_version:seen,idempotency_key:`del-${stamp}`}})

  const late=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:seen,idempotency_key:`late-${stamp}`,
    cells:[{row:5,column:1,value:'B가 고친 값'}]}}).then(response=>response.json())
  expect(late.rebased_cells).toBe(1)

  const grid=await read()
  // 행5는 4행으로 올라갔으니 B의 편집도 그리로 따라가야 한다.
  expect(grid.get(4)).toBe('B가 고친 값')
  // 행6은 아무도 건드리지 않았으므로 그대로 남아야 한다.
  expect(grid.get(5)).toBe('행6')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

test('a cell saved into a row someone else deleted is refused, not misplaced', async ({ request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`사라진 행 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`seed-${stamp}`,
    cells:[{row:3,column:1,value:'셋'},{row:4,column:1,value:'넷'}]}})
  const seen=(await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())).version
  await request.patch(`/api/v1/sheets/${sheet}/structure:apply`,{data:{axis:'row',action:'delete',index:3,count:1,base_version:seen,idempotency_key:`del-${stamp}`}})

  const late=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:seen,idempotency_key:`late-${stamp}`,
    cells:[{row:3,column:1,value:'사라진 행에 쓰기'}]}}).then(response=>response.json())
  expect(late.applied_cells).toBe(0)
  expect(late.dropped_cells).toEqual([{row:3,column:1}])
  // 그 자리를 물려받은 값이 덮어써지면 안 된다.
  const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A3:A3`).then(response=>response.json())).items as Array<{value?:unknown}>
  expect(items[0]?.value).toBe('넷')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
