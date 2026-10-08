import { expect, test } from '@playwright/test'

// 다른 곳에서 붙여 넣으면 "1,234" 처럼 글자로 들어온다. 사람 눈에는 숫자인데
// =SUM 은 조용히 빼고 셈한다 — 합계가 작게 나오는데 무엇이 빠졌는지는 아무
// 데도 적히지 않는다. 찾아서 알려 주고 한 번에 고치는 길을 통째로 확인한다.
test('numbers stored as text are found and fixed, and the total changes', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'tn-seed', cells: [
      {row:1,column:1,value:'금액'},
      {row:2,column:1,value:'1,234'},
      {row:3,column:1,value:'₩5,000'},
      {row:4,column:1,value:1000},
      {row:5,column:1,formula:'=SUM(A2:A4)'},
    ],
  }})
  expect(write.status(), await write.text()).toBeLessThan(300)
  // 고치기 전: 글자로 담긴 둘은 빠지고 1000 만 더해진다.
  await expect.poll(async()=>{
    const result = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A5:A5`).then(r => r.json())
    return result.items[0]?.value
  },{timeout:15_000}).toBe(1000)

  const canvas = page.locator('canvas.grid-canvas')
  // 대화상자가 선택한 칸 둘레의 자료 덩어리를 스스로 찾는다.
  await canvas.click({ position: { x: 70, y: 42 } })
  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '데이터 정리' }).click()
  await page.getByRole('menuitem', { name: '텍스트로 저장된 숫자…' }).click()
  const dialog = page.getByRole('dialog', { name: '텍스트로 저장된 숫자' })
  await expect(dialog.getByText(/1,234/)).toBeVisible({ timeout: 15_000 })
  await dialog.getByRole('button', { name: '숫자로 바꾸기' }).click()

  // 고친 뒤: 셋이 다 더해진다.
  await expect.poll(async()=>{
    const result = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A5:A5`).then(r => r.json())
    return result.items[0]?.value
  },{timeout:15_000}).toBe(7234)

  // 값만 맞고 화면이 달라지면 사람은 자기 자료가 망가졌다고 읽는다. ₩ 는
  // 서식으로 옮겨 두었으므로 칸은 그대로 ₩5,000 으로 보여야 한다.
  const after = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A3:A3`).then(r => r.json())
  expect(after.items[0]?.value).toBe(5000)
  // 기호는 따옴표로 묶는다. 붙여넣기가 내는 서식과 같은 글자다.
  expect(after.items[0]?.style?.number_format).toBe('"₩"#,##0')
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
