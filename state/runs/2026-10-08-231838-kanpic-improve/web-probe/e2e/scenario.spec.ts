import { expect, test } from '@playwright/test'

// 시나리오는 가정 한 벌에 이름을 붙여 두고 나란히 놓고 견주는 것이다.
// 회의에서 두 안을 놓고 보는 그 일이다.
test('scenarios are saved and compared side by side', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'sc-seed', cells: [
      {row:1,column:2,value:10000},{row:2,column:2,value:6000},{row:3,column:2,value:1000},
      {row:4,column:2,formula:'=(B1-B2)*B3'},{row:5,column:2,formula:'=B4/B3'},
    ],
  }})
  expect(write.status(), await write.text()).toBeLessThan(300)

  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '가정 분석' }).click()
  await page.getByRole('menuitem', { name: '시나리오…' }).click()
  const dialog = page.getByRole('dialog', { name: '시나리오' })
  for (const [name, inputs] of [['낙관','B1=12000\nB3=1500'],['보수','B1=9000\nB3=800']] as const) {
    await dialog.getByRole('button', { name: '새 시나리오' }).click()
    await dialog.getByLabel('시나리오 이름').fill(name)
    await dialog.getByLabel('시나리오 가정').fill(inputs)
    await dialog.getByRole('button', { name: '저장', exact: true }).click()
    await expect(dialog.locator('aside').getByRole('button', { name: new RegExp(name) })).toBeVisible({ timeout: 15_000 })
  }
  await dialog.getByLabel('시나리오 결과 셀').fill('B4')
  await dialog.getByRole('button', { name: '견주기' }).click()
  // 지금 4,000,000 / 낙관 9,000,000 / 보수 2,400,000
  for (const expected of ['4,000,000','9,000,000','2,400,000']) {
    await expect(dialog.locator('td', { hasText: new RegExp(`^${expected.replace(/,/g,',')}$`) })).toBeVisible({ timeout: 15_000 })
  }
  // 셈하지 못한 자리는 까닭을 그 자리에 적어야 한다. 비워 두기만 하면
  // 사람은 왜 비었는지 알 수 없고, 지금 값을 적으면 가정이 먹힌 것처럼 보인다.
  await dialog.getByRole('button', { name: '새 시나리오' }).click()
  await dialog.getByLabel('시나리오 이름').fill('0으로')
  await dialog.getByLabel('시나리오 가정').fill('B3=0')
  await dialog.getByRole('button', { name: '저장', exact: true }).click()
  await expect(dialog.locator('aside').getByRole('button', { name: /0으로/ })).toBeVisible({ timeout: 15_000 })
  await dialog.getByLabel('시나리오 결과 셀').fill('B5')
  await dialog.getByRole('button', { name: '견주기' }).click()
  await expect(dialog.locator('td', { hasText: '#DIV/0!' })).toBeVisible({ timeout: 15_000 })

  // 시트는 그대로여야 한다.
  const after = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/B1:B4`).then(r => r.json())
  expect(after.items.map((cell:{value:unknown})=>cell.value)).toEqual([10000, 6000, 1000, 4000000])
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
