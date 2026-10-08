import { expect, test } from '@playwright/test'

// 좁은 열에 안 들어가는 숫자를 눌러 담으면 읽을 수 없고, 잘라 보여 주면
// 1,234,567 이 1,234 로 읽힌다. 그래서 격자는 `####` 로 값이 아니라 **열이
// 좁다** 는 것을 알린다.
//
// 그 그림 자체는 캔버스라 여기서 확인할 수 없다(칠해진 양도 글자 사이 간격도
// 눌러 담은 숫자와 구별되지 않는다). 판단 자체는 src/lib/cellWidth.test.ts 가
// 본다. 여기서는 `####` 가 **값을 가리지 않는지** 를 확인한다. 저장된 값은
// 그대로여야 하고, 수식 입력줄에는 전부 보여야 한다.
test('hashing a too-narrow number never hides the value itself', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`좁은 열 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`narrow-${stamp}`,cells:[
    {row:1,column:1,value:123456789012345},{row:1,column:2,value:'막음'},
    {row:2,column:1,formula:'=A1*2'},
  ]}})
  const created=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${sheet}/layout:apply`,{headers:{'Idempotency-Key':`size-${stamp}`},data:{
    expected_revision:created.sheets[0].layout.revision,action:'resize',axis:'column',start:1,count:1,size:70}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  // 열이 좁아도 값은 수식 입력줄에서 온전히 읽을 수 있어야 한다.
  await expect(page.locator('.formula-input')).toHaveValue('123456789012345')

  const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A2`).then(r=>r.json())).items
  expect(items.find((cell:{row:number})=>cell.row===1)).toMatchObject({value:123456789012345})
  // 좁은 열은 그림의 문제이지 값의 문제가 아니다. 참조하는 수식은 그대로 센다.
  expect(items.find((cell:{row:number})=>cell.row===2)).toMatchObject({value:246913578024690})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
