import { expect, test } from '@playwright/test'

// 폭포는 앞줄까지의 누계에서 시작해 값만큼 자란다. 매출에서 원가와 판관비를
// 빼 영업이익에 이르는 길이 한눈에 보여야 한다. 늘어난 것과 줄어든 것을
// 색으로 가르고, 합계 줄은 바닥부터 그린다.
test('a waterfall chart stacks each step on the running total', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.dismiss() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'waterfall-seed', cells: [
      {row:1,column:1,value:'항목'},{row:1,column:2,value:'금액'},{row:1,column:3,value:'합계'},
      {row:2,column:1,value:'매출'},{row:2,column:2,value:1000},
      {row:3,column:1,value:'원가'},{row:3,column:2,value:-600},
      {row:4,column:1,value:'판관비'},{row:4,column:2,value:-150},
      {row:5,column:1,value:'영업이익'},{row:5,column:2,value:250},{row:5,column:3,value:true},
    ],
  }})
  expect(write.status(), await write.text()).toBeLessThan(300)

  await page.getByRole('menuitem', { name: '삽입' }).click()
  await page.getByRole('menuitem', { name: '차트…' }).click()
  const dialog = page.getByRole('dialog', { name: '차트 만들기' })
  await dialog.getByLabel('차트 원본 범위').fill('A1:C5')
  await dialog.getByLabel('차트 유형').selectOption('waterfall')
  // 종류를 고르면 열을 어떻게 읽는지 알려 준다.
  await expect(dialog.getByText(/이름 · 값 · 합계 여부/)).toBeVisible()
  await dialog.getByLabel('차트 제목').fill('손익 구조')
  await dialog.getByRole('button', { name: '저장' }).click()

  const svg = page.locator('svg.chart-svg').first()
  await expect(svg).toBeVisible({ timeout: 15_000 })
  for (const name of ['매출','원가','판관비','영업이익']) await expect(svg.getByText(name, { exact:true })).toBeVisible({ timeout: 15_000 })

  // 막대 넷. 늘어난 것과 줄어든 것과 합계가 서로 다른 색이어야 한다.
  const bars = svg.locator('rect[rx="1"]')
  await expect(bars).toHaveCount(4)
  const colours = await bars.evaluateAll(nodes => nodes.map(node => node.getAttribute('fill')))
  expect(colours[0]).not.toBe(colours[1])          // 매출(증가) ≠ 원가(감소)
  expect(colours[1]).toBe(colours[2])              // 원가·판관비 둘 다 감소
  expect(colours[3]).not.toBe(colours[0])          // 합계는 증감과 다른 색
  expect(new Set(colours).size).toBe(3)

  // 원가 막대는 매출 꼭대기(1000)에서 시작해 400까지 내려온다. 바닥까지
  // 내려오면 폭포가 아니라 그냥 막대 차트다.
  const geometry = await bars.evaluateAll(nodes => nodes.map(node => ({
    y: Number(node.getAttribute('y')), h: Number(node.getAttribute('height')),
  })))
  const bottom = (bar:{y:number;h:number}) => bar.y + bar.h
  expect(geometry[1].y, '원가는 매출이 끝난 자리에서 시작한다').toBe(geometry[0].y)
  expect(bottom(geometry[1]), '원가가 바닥까지 내려오면 폭포가 아니다').toBeLessThan(bottom(geometry[0]))
  // 합계 줄은 바닥부터 그린다.
  expect(bottom(geometry[3]), '영업이익은 바닥에서 그린다').toBe(bottom(geometry[0]))
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
