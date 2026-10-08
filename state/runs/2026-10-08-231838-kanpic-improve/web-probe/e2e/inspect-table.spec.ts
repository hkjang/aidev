import { expect, test } from '@playwright/test'

// 정리 도구가 여섯 개인데 사람이 "데이터 정리" 를 열어 볼 생각을 해야 알 수
// 있었다. 무엇이 잘못됐는지 모르는 채로 열어 보는 사람은 없다. 표를 보고
// 먼저 세어 말해 주고, 거기서 바로 고치는 자리로 넘어가는 것까지 확인한다.
test('inspecting a table counts what is wrong and hands it to the right tool', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'it-seed', cells: [
      {row:1,column:1,value:'이름'},{row:1,column:2,value:'금액'},
      {row:2,column:1,value:'가 '},{row:2,column:2,value:'1,000'},
      {row:3,column:1,value:'가 '},{row:3,column:2,value:'1,000'},
      {row:4,column:1,value:'나'},{row:4,column:2,value:2000},
    ] }})
  expect(write.status(), await write.text()).toBeLessThan(300)

  await page.reload()
  const canvas = page.locator('canvas.grid-canvas')
  await canvas.click({ position: { x: 70, y: 42 } })
  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '데이터 정리' }).click()
  await page.getByRole('menuitem', { name: '표 검사…' }).click()

  const dialog = page.getByRole('dialog', { name: '표 검사' })
  await expect(dialog.getByText('글자로 담긴 숫자')).toBeVisible({ timeout: 15_000 })
  await expect(dialog.getByText('중복된 행')).toBeVisible()
  await expect(dialog.getByText('앞뒤·가운데 공백')).toBeVisible()

  // 셈이 달라지는 것이 모양만 다듬는 것보다 앞에 온다.
  const titles = await dialog.locator('.inspect-head strong').allTextContents()
  expect(titles.indexOf('글자로 담긴 숫자')).toBeLessThan(titles.indexOf('앞뒤·가운데 공백'))

  // 거기서 바로 고치는 자리로 넘어간다.
  await dialog.locator('li').filter({ hasText: '글자로 담긴 숫자' }).getByRole('button', { name: '고치기…' }).click()
  await expect(page.getByRole('dialog', { name: '텍스트로 저장된 숫자' })).toBeVisible({ timeout: 15_000 })
  await page.getByRole('dialog', { name: '텍스트로 저장된 숫자' }).getByRole('button', { name: '숫자로 바꾸기' }).click()

  await expect.poll(async()=>{
    const result = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/B2:B3`).then(r => r.json())
    return (result.items as Array<{value?:unknown}>).map(item=>item.value)
  },{timeout:15_000}).toEqual([1000,1000])
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
