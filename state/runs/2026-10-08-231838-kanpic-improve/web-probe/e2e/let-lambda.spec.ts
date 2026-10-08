import { expect, test, type APIRequestContext } from '@playwright/test'

const read=async (request:APIRequestContext,sheet:string,range:string)=>{
  const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/${range}`)).json()).items as Array<{row:number;column:number;value?:unknown}>
  return new Map(items.map(item=>[`${item.row}:${item.column}`,item.value]))
}

// LET과 LAMBDA는 파서에 이름 바인딩을 넣어야 하는 기능이라, 서버가 실제로
// 계산하고 스필까지 시키는지 따로 확인한다.
test('LET names the steps and LAMBDA walks the array through the server', async ({ request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`LET ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[[10,100],[20,200],[30,300]]
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`let-${stamp}`,cells:[
    ...rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value}))),
    {row:1,column:4,formula:'=LET(total,SUM(A1:A3),count,COUNT(A1:A3),total/count)'},
    {row:3,column:4,formula:'=MAP(A1:A3,LAMBDA(x,x*2))'},
    {row:7,column:4,formula:'=BYROW(A1:B3,LAMBDA(row,SUM(row)))'},
    {row:11,column:4,formula:'=REDUCE(0,A1:A3,LAMBDA(acc,x,acc+x))'},
    {row:13,column:4,formula:'=LET(tax,LAMBDA(v,v*0.1),SUM(MAP(B1:B3,tax)))'},
  ]}})

  const grid=await read(request,sheet,'D1:D15')
  expect(grid.get('1:4')).toBe(20)
  expect([grid.get('3:4'),grid.get('4:4'),grid.get('5:4')]).toEqual([20,40,60])
  expect([grid.get('7:4'),grid.get('8:4'),grid.get('9:4')]).toEqual([110,220,330])
  expect(grid.get('11:4')).toBe(60)
  expect(grid.get('13:4')).toBe(60)

  // 이름이 가리키는 셀은 여전히 이 수식의 의존성이므로, 값을 바꾸면 다시
  // 계산된다. 이름을 붙였다고 의존성이 끊기면 화면이 조용히 낡는다.
  const version=(await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())).version
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:version,idempotency_key:`let-edit-${stamp}`,cells:[{row:1,column:1,value:40}]}})
  await expect.poll(async()=>(await read(request,sheet,'D1:D1')).get('1:4')).toBe(30)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
