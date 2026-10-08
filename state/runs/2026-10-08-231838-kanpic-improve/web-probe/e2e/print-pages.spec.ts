import { expect, test } from '@playwright/test'

// 종이가 나온 뒤에 몇 장인지 아는 것은 늦다. 방향과 여백을 고르는 자리에서
// 몇 장으로 나뉘는지 그때그때 보여 주는 것까지 확인한다.
test('page setup says how many sheets wide the table prints, and reacts', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  // 기본 열 너비 108px 로 스무 열이면 A4 세로 한 장에 들어가지 않는다.
  // 일부러 D열에서 시작한다. 미리보기가 인쇄와 다른 범위를 세면 A열부터
  // 세어 있지도 않은 열을 장에 넣는다.
  const cells = Array.from({length:20},(_,index)=>({row:1,column:index+4,value:`열${index+1}`}))
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, {
    data: { base_version: workbook.version, idempotency_key: 'pp-seed', cells } })
  expect(write.status(), await write.text()).toBeLessThan(300)

  await page.getByRole('menuitem', { name: '파일' }).click()
  await page.getByRole('menuitem', { name: '페이지 설정…' }).click()
  const dialog = page.getByRole('dialog', { name: '인쇄 설정' })
  const pages = dialog.locator('.print-options-pages')
  await expect(pages).toContainText('가로로', { timeout: 15_000 })
  const portrait = Number((await pages.textContent())?.match(/가로로\s*(\d+)장/)?.[1])
  expect(portrait).toBeGreaterThan(1)
  // 어느 열이 어느 장에 가는지도 적혀 있어야 고르는 데 쓸모가 있다.
  // 자료가 있는 D열에서 시작해야 한다. A라면 빈 열을 세고 있는 것이다.
  await expect(pages.locator('li').first()).toHaveText(/^D/)

  // 가로로 돌리면 그 자리에서 장수가 줄어든다.
  await dialog.getByText('가로', { exact: true }).click()
  await expect.poll(async()=>Number((await pages.textContent())?.match(/가로로\s*(\d+)장/)?.[1] ?? '1'),
    { timeout: 10_000 }).toBeLessThan(portrait)

  // 한 장 너비에 맞추면 나뉘지 않는다.
  await dialog.getByText('한 장 너비에 맞추기').click()
  await expect(pages).toContainText('한 장 너비에 맞춰')
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
