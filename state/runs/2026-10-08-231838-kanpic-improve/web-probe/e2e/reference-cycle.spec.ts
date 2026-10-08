import { expect, test } from '@playwright/test'

// 조회표를 붙들어 두려면 `$` 를 매번 손으로 쳐야 했다. F4 로 참조 고정을
// 돌리는 것은 수식을 쓰는 사람의 손버릇이다.
test('F4 cycles a reference through the four forms, in the cell and in the formula bar', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`참조 고정 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`ref-${stamp}`,cells:[
    {row:1,column:1,value:2},{row:2,column:1,value:3},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  // 셀 안에서 수식을 쓰다 누른다.
  await page.getByRole('combobox',{name:'이름 상자'}).fill('C1')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.keyboard.type('=A1*2')
  await page.keyboard.press('F4')
  await expect(page.locator('.formula-input')).toHaveValue('=$A$1*2')
  await page.keyboard.press('F4')
  await expect(page.locator('.formula-input')).toHaveValue('=A$1*2')
  await page.keyboard.press('Enter')
  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/C1:C1`).then(r=>r.json())).items
    return items[0]?.formula
  }).toBe('=A$1*2')
  // 고정이 붙어도 값은 그대로 계산된다.
  const stored=(await request.get(`/api/v1/sheets/${sheet}/ranges/C1:C1`).then(r=>r.json())).items[0]
  expect(stored).toMatchObject({value:4})

  // 수식 입력줄에서도 같아야 한다.
  await page.getByRole('combobox',{name:'이름 상자'}).fill('C2')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  const bar=page.locator('.formula-input')
  await bar.click()
  await bar.fill('=A2+1')
  await bar.press('F4')
  await expect(bar).toHaveValue('=$A$2+1')
  await bar.press('Enter')
  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/C2:C2`).then(r=>r.json())).items
    return items[0]?.formula
  }).toBe('=$A$2+1')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
