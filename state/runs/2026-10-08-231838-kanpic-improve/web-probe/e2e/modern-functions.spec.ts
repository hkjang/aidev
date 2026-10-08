import { expect, test, type APIRequestContext } from '@playwright/test'

const read=async (request:APIRequestContext,sheet:string,range:string)=>{
  const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/${range}`)).json()).items as Array<{row:number;column:number;value?:unknown}>
  return new Map(items.map(item=>[`${item.row}:${item.column}`,item.value]))
}

// 새 배열·텍스트 함수는 서버에서 계산되어 여러 셀로 펼쳐진다. 엔진 단위
// 테스트만으로는 펼쳐진 결과가 시트에 실제로 쓰이는지 알 수 없다.
test('the new array and text functions spill through the server', async ({ request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`함수 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','매출'],['부산',80],['서울',120],['대구',95]]
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`fn-${stamp}`,cells:[
    ...rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value}))),
    {row:1,column:4,value:'이름: 홍길동'},
    {row:1,column:5,formula:'=TEXTAFTER(D1,": ")'},
    {row:3,column:4,formula:'=SORTBY(A2:B4,B2:B4,-1)'},
    {row:7,column:4,formula:'=TEXTSPLIT("a,b;c,d",",",";")'},
    // 가운데 인수를 비워 두면 그 인수의 기본값이 쓰인다.
    {row:10,column:4,formula:'=SEQUENCE(2,,7)'},
    {row:12,column:4,formula:'=DROP(A1:B4,1)'},
  ]}})

  const grid=await read(request,sheet,'D1:E14')
  expect(grid.get('1:5')).toBe('홍길동')
  // SORTBY는 매출 내림차순으로 세 행을 펼친다.
  expect([grid.get('3:4'),grid.get('3:5'),grid.get('5:4'),grid.get('5:5')]).toEqual(['서울',120,'부산',80])
  // TEXTSPLIT은 2×2 표가 된다.
  expect([grid.get('7:4'),grid.get('7:5'),grid.get('8:4'),grid.get('8:5')]).toEqual(['a','b','c','d'])
  // SEQUENCE(2,,7)은 열 수를 기본값 1로 두고 7부터 센다.
  expect([grid.get('10:4'),grid.get('11:4')]).toEqual([7,8])
  // DROP은 머리글 행을 덜어낸 데이터만 남긴다.
  expect([grid.get('12:4'),grid.get('12:5'),grid.get('14:4')]).toEqual(['부산',80,'대구'])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
