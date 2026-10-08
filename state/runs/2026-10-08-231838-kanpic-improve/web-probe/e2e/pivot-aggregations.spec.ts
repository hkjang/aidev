import { expect, test } from '@playwright/test'

// 서버가 열세 가지 집계를 내주어도 목록에 없으면 고를 수 없다. 서버만
// 고치고 화면을 그대로 두면 아무도 쓰지 못하는 기능이 된다.
//
// 개수를 세는 셋은 이름만으로 무엇이 다른지 알기 어려워 괄호로 갈라
// 적는다. count 는 숫자만, counta 는 비어 있지 않은 것을 센다.
test('the pivot dialog offers every aggregation the server computes', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`피벗 집계 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`agg-${stamp}`,cells:[
    {row:1,column:1,value:'지역'},{row:1,column:2,value:'수량'},
    {row:2,column:1,value:'서울'},{row:2,column:2,value:10},
    {row:3,column:1,value:'서울'},{row:3,column:2,value:'미정'},
    {row:4,column:1,value:'서울'},{row:4,column:2,value:20},
    {row:5,column:1,value:'부산'},{row:5,column:2,value:5},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1:B5')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('button',{name:'피벗 패널'}).click()
  await page.getByRole('button',{name:'새 피벗'}).click()
  const dialog=page.getByRole('dialog')
  await expect(dialog).toBeVisible()

  // 서버가 셈할 수 있는 열세 가지가 모두 목록에 있어야 한다.
  const aggregation=dialog.getByRole('combobox',{name:'집계 1'})
  const offered=await aggregation.locator('option').evaluateAll(items=>items.map(item=>(item as HTMLOptionElement).value))
  expect(offered.sort()).toEqual([
    'average','count','counta','countunique','max','median','min',
    'product','stdev','stdevp','sum','var','varp',
  ])

  // 골라서 저장하면 서버가 그대로 받아 셈한다. "미정" 은 숫자가 아니므로
  // count 는 2, counta 는 3 이다.
  await dialog.getByRole('combobox',{name:'행 필드 1'}).count().catch(()=>0)
  await aggregation.selectOption('counta')
  await expect(aggregation).toHaveValue('counta')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
