import { expect, test } from '@playwright/test'

// 값 표시와 축 범위는 눈금을 짚어 읽는 수고를 덜자는 것이다. 거는 길이
// 열리고 실제로 그림이 달라지는지 본다.
test('a chart can show values and take a fixed axis range', async ({ page, request }) => {
  const stamp = Date.now()
  const workbook = await request.post('/api/v1/workbooks', { data: { title: `차트 값 표시 ${stamp}` } }).then(response => response.json())
  const sheetId = workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: `chart-${stamp}`,
    cells: [
      { row: 1, column: 1, value: '월' }, { row: 1, column: 2, value: '매출' },
      { row: 2, column: 1, value: '1월' }, { row: 2, column: 2, value: 50 },
      { row: 3, column: 1, value: '2월' }, { row: 3, column: 2, value: 70 },
    ],
  }})
  await page.goto(`/workbooks/${workbook.id}`)
  await page.locator('canvas.grid-canvas').waitFor()

  await page.getByRole('menuitem', { name: '삽입' }).click()
  await page.getByRole('menuitem', { name: '차트…' }).click()
  const dialog = page.getByRole('dialog', { name: '차트 만들기' })
  await dialog.getByLabel('차트 원본 범위').fill('A1:B3')
  await dialog.getByLabel('값 표시').check()
  await dialog.getByLabel('세로축 최소').fill('40')
  await dialog.getByLabel('세로축 최대').fill('80')
  await dialog.getByRole('button', { name: '저장' }).click()
  await expect(dialog).toBeHidden()

  // 저장한 것이 서버에 남아야 한다.
  await expect.poll(async()=>{
    const charts = await request.get(`/api/v1/workbooks/${workbook.id}/charts?sheet_id=${sheetId}`).then(response => response.json())
    const chart = charts.items?.[0]
    return chart ? [chart.data_labels, chart.y_axis_min, chart.y_axis_max] : []
  },{timeout:15_000}).toEqual([true,40,80])

  // 그림에도 값이 적혀야 한다. 저장만 되고 그려지지 않으면 켠 뜻이 없다.
  await expect(page.locator('.chart-svg .chart-label').first()).toBeVisible()
  await expect(page.locator('.chart-svg .chart-label')).toHaveCount(2)

  // 뒤집어 적으면 막는다. 저장되면 그릴 수 없는 차트가 남는다.
  await page.locator('.chart-svg').first().click({ button: 'right' })
  await page.getByRole('menuitem', { name: '차트 설정…' }).click()
  const edit = page.getByRole('dialog', { name: '차트 편집' })
  await edit.getByLabel('세로축 최소').fill('90')
  page.once('dialog', alert => void alert.accept())
  await edit.getByRole('button', { name: '저장' }).click()
  // 막혔으므로 대화상자가 그대로 열려 있어야 한다.
  await expect(edit).toBeVisible()
})
