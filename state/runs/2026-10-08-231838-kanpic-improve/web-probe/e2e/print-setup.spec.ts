import { expect, test } from '@playwright/test'

// 넓은 표를 세로로만 찍으면 열이 자꾸 다음 장으로 넘어간다. 방향과 여백을
// 고르고 한 장 너비에 맞추는 길이 실제로 열리는지 본다.
test('page setup chooses paper direction and fits a wide table to one page', async ({ page, request }) => {
  const stamp = Date.now()
  const workbook = await request.post('/api/v1/workbooks', { data: { title: `페이지 설정 ${stamp}` } }).then(response => response.json())
  const sheetId = workbook.sheets[0].id as string
  const cells = []
  for (let column = 1; column <= 20; column += 1) cells.push({ row: 1, column, value: `열${column}` })
  for (let column = 1; column <= 20; column += 1) cells.push({ row: 2, column, value: column })
  await request.post(`/api/v1/sheets/${sheetId}/cells:batch`, {
    data: { base_version: workbook.version, idempotency_key: `setup-${stamp}`, cells },
  })
  await page.goto(`/workbooks/${workbook.id}`)
  await page.locator('canvas.grid-canvas').waitFor()

  await page.getByRole('menuitem', { name: '파일' }).click()
  await page.getByRole('menuitem', { name: '페이지 설정…' }).click()
  const dialog = page.getByRole('dialog', { name: '인쇄 설정' })
  await expect(dialog).toBeVisible()
  // 고른 것이 다음에도 기억되어야 한다. 같은 표를 여러 번 찍는 일이 흔하다.
  await dialog.getByLabel('가로').check()
  await dialog.getByLabel('좁게').check()
  await dialog.getByLabel('한 장 너비에 맞추기').check()
  await dialog.getByRole('button', { name: '인쇄', exact: true }).click()
  await expect(dialog).toBeHidden()

  await page.getByRole('menuitem', { name: '파일' }).click()
  await page.getByRole('menuitem', { name: '페이지 설정…' }).click()
  await expect(dialog.getByLabel('가로')).toBeChecked()
  await expect(dialog.getByLabel('좁게')).toBeChecked()
  await expect(dialog.getByLabel('한 장 너비에 맞추기')).toBeChecked()
  await dialog.getByRole('button', { name: '취소' }).click()
  await expect(dialog).toBeHidden()
})
