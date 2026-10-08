import { expect, test } from '@playwright/test'

// 일정표는 열을 앞에서부터 이름·시작·끝으로 읽어 가로 막대로 그린다. 날짜는
// 대개 글자다 — DATE() 가 글자를 내기 때문이다. 막대가 실제로 그려지는지,
// 눈금에 날짜가 적히는지 통째로 확인한다.
test('a timeline chart draws one bar per task from name, start and end columns', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.dismiss() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'timeline-seed', cells: [
      {row:1,column:1,value:'일감'},{row:1,column:2,value:'시작'},{row:1,column:3,value:'끝'},
      {row:2,column:1,value:'설계'},{row:2,column:2,formula:'=DATE(2026,1,5)'},{row:2,column:3,formula:'=DATE(2026,2,10)'},
      {row:3,column:1,value:'개발'},{row:3,column:2,formula:'=DATE(2026,2,11)'},{row:3,column:3,formula:'=DATE(2026,4,30)'},
      // 끝을 적지 않으면 그 날 하루짜리 이정표다.
      {row:4,column:1,value:'출시'},{row:4,column:2,formula:'=DATE(2026,6,1)'},
    ],
  }})
  expect(write.status(), await write.text()).toBeLessThan(300)

  await page.getByRole('menuitem', { name: '삽입' }).click()
  await page.getByRole('menuitem', { name: '차트…' }).click()
  const dialog = page.getByRole('dialog', { name: '차트 만들기' })
  await dialog.getByLabel('차트 원본 범위').fill('A1:C4')
  await dialog.getByLabel('차트 유형').selectOption('timeline')
  // 종류를 고르면 열을 어떻게 읽는지 알려 준다.
  await expect(dialog.getByText(/이름 · 시작 · 끝/)).toBeVisible()
  await dialog.getByLabel('차트 제목').fill('출시 일정')
  await dialog.getByRole('button', { name: '저장' }).click()

  const svg = page.locator('svg.chart-svg').first()
  await expect(svg).toBeVisible({ timeout: 15_000 })
  // 일감 셋이므로 막대도 셋이다.
  await expect(svg.locator('rect')).toHaveCount(3, { timeout: 15_000 })
  // 세로축은 눈금이 아니라 일감의 이름이다.
  for (const name of ['설계','개발','출시']) await expect(svg.getByText(name, { exact:true })).toBeVisible()
  // 가로축에는 날짜가 적힌다.
  await expect(svg.getByText('2026-01-05')).toBeVisible()
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
