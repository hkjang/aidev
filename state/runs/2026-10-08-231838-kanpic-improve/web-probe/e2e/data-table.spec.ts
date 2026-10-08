import { expect, test } from '@playwright/test'

// 데이터 표는 "이 값이 이만큼일 때 결과가 어떻게 되나" 를 한 번에 늘어놓는다.
// 손으로 하면 값을 넣고 적고 되돌리기를 되풀이해야 하는 일이다.
test('a data table spells out one and two variable assumptions', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.dismiss() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'dt-seed', cells: [
      {row:1,column:2,value:1000},{row:2,column:2,value:0.05},{row:3,column:2,formula:'=B1*B2'},
    ],
  }})
  expect(write.status(), await write.text()).toBeLessThan(300)

  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '가정 분석' }).click()
  await page.getByRole('menuitem', { name: '데이터 표…' }).click()
  const dialog = page.getByRole('dialog', { name: '데이터 표' })
  await dialog.getByLabel('데이터 표 결과 셀').fill('B3')
  await dialog.getByLabel('데이터 표 세로 입력 셀').fill('B2')
  await dialog.getByLabel('데이터 표 세로 가정').fill('0.03, 0.04, 0.05')
  await dialog.getByRole('button', { name: '표 만들기' }).click()
  // 1000 * 0.03 = 30, 0.04 = 40, 0.05 = 50
  for (const expected of ['30','40','50']) {
    await expect(dialog.locator('td', { hasText: new RegExp(`^${expected}$`) })).toBeVisible({ timeout: 15_000 })
  }

  // 가로 쪽을 더하면 두 방향 표가 된다.
  await dialog.getByLabel('데이터 표 가로 입력 셀').fill('B1')
  await dialog.getByLabel('데이터 표 가로 가정').fill('1000, 2000')
  await dialog.getByRole('button', { name: '표 만들기' }).click()
  await expect(dialog.locator('td', { hasText: /^80$/ })).toBeVisible({ timeout: 15_000 })

  // 시트는 그대로여야 한다. 가정을 넣어 본 것이 남으면 표를 한 번 그렸다는
  // 이유로 사람의 자료가 바뀐다.
  const after = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/B1:B3`).then(r => r.json())
  expect(after.items.map((cell:{value:unknown})=>cell.value)).toEqual([1000, 0.05, 50])
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
