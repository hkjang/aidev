import { expect, test } from '@playwright/test'

// `=0.1+0.2` 가 0.30000000000000004 로 보이는 것은 스프레드시트가 고장 난
// 것처럼 보이는 가장 오래된 방법이다. 금액을 더한 열에 이런 값이 뜨면
// 사람은 합계를 믿지 않는다.
test('a sum of decimals reads as the number a person expects', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`정밀도 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`prec-${stamp}`,cells:[
    {row:1,column:1,value:0.1},{row:2,column:1,value:0.2},
    {row:3,column:1,formula:'=SUM(A1:A2)'},
    {row:4,column:1,formula:'=IF(A3=0.3,"맞음","틀림")'},
    {row:5,column:1,formula:'=1.5E+3'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A3:A5`).then(r=>r.json())).items
  const at=(row:number)=>items.find((cell:{row:number})=>cell.row===row)
  expect(at(3)).toMatchObject({value:0.3})
  // 이 비교가 틀리면 대조용 수식이 있지도 않은 불일치를 보고한다.
  expect(at(4)).toMatchObject({value:'맞음'})
  expect(at(5)).toMatchObject({value:1500})

  // 넘쳐 버린 결과는 빈 응답이 아니라 오류로 돌아와야 한다.
  const overflow=await request.post('/api/v1/formulas:evaluate',{data:{formula:'=1E308*10',cells:{}}})
  expect(overflow.ok()).toBe(true)
  expect(await overflow.json()).toMatchObject({error:{code:'#NUM!'}})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
