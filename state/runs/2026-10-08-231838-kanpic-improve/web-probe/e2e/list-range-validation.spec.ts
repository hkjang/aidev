import { expect, test } from '@playwright/test'

// 코드 목록은 보통 별도 시트에 산다. 드롭다운이 그 범위를 가리켜야 목록을
// 한 곳에서 고치고 모든 셀에 반영할 수 있다.
test('a range dropdown offers what its source range holds right now', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`범위 목록 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`lr-seed-${stamp}`,cells:[
    {row:1,column:5,value:'서울'},{row:2,column:5,value:'부산'},{row:3,column:5,value:'서울'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1:A5')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('button',{name:'데이터 검증'}).click()
  const dialog=page.getByRole('dialog',{name:'데이터 검증'})
  await dialog.getByLabel('검증 유형').selectOption('list_range')
  await dialog.getByLabel('드롭다운 목록 범위').fill('E1:E10')
  await dialog.getByRole('button',{name:'규칙 저장'}).click()

  // 서버가 목록을 지금 값으로 풀어 준다: 중복은 한 번만, 빈 칸은 제외.
  await expect.poll(async()=>{
    const rules=await request.get(`/api/v1/sheets/${sheet}/data-validations`).then(response=>response.json())
    return rules.items[0]?.source_options?.map((option:{value:unknown})=>option.value)
  },{timeout:15_000}).toEqual(['서울','부산'])

  const current=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  const refused=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:current.version,idempotency_key:`lr-bad-${stamp}`,cells:[{row:2,column:1,value:'대구'}]}})
  expect(refused.status()).toBe(422)

  // 목록에 값을 더하면 같은 입력이 통과한다. 규칙은 손대지 않았다.
  const beforeAdd=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:beforeAdd.version,idempotency_key:`lr-extend-${stamp}`,cells:[{row:4,column:5,value:'대구'}]}})
  const afterAdd=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  const accepted=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:afterAdd.version,idempotency_key:`lr-good-${stamp}`,cells:[{row:2,column:1,value:'대구'}]}})
  expect(accepted.ok()).toBe(true)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
