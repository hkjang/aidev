import { expect, test } from '@playwright/test'

// 병합된 칸이 섞인 자료에서 중복 행을 지우려 하면, 병합 서식이 다른 행으로
// 딸려 가고 서버가 묶음을 통째로 물리쳤다. 사람이 본 것은 "stored merge
// metadata is invalid" 였다 — 무엇을 해야 하는지 알 수 없는 말이다.
test('cleaning duplicate rows over a merged cell says what to do', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  const merged = { merge: { start_row: 2, start_column: 2, end_row: 2, end_column: 3 } }
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'cm-seed', cells: [
      {row:1,column:1,value:'이름'},{row:1,column:2,value:'값'},
      {row:2,column:1,value:'사과'},{row:2,column:2,value:1,style:merged},{row:2,column:3,style:merged},
      {row:3,column:1,value:'사과'},{row:3,column:2,value:1},
      {row:4,column:1,value:'배'},{row:4,column:2,value:2},
    ],
  }})
  expect(write.status(), await write.text()).toBeLessThan(300)

  const canvas = page.locator('canvas.grid-canvas')
  await canvas.click({ position: { x: 70, y: 42 } })
  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '데이터 정리' }).click()
  await page.getByRole('menuitem', { name: '중복 항목 삭제…' }).click()

  // 어느 칸이 걸렸는지 짚어 주고, 미리보기는 아예 열리지 않는다.
  await expect.poll(()=>alerts.join('|'),{timeout:15_000}).toContain('병합된 셀')
  expect(alerts.join('|')).toContain('B2')
  expect(alerts.join('|')).not.toContain('merge metadata')
  await expect(page.getByRole('dialog', { name: '중복 항목 삭제' })).toHaveCount(0)

  // 자료는 그대로다. 물리쳐진 정리가 절반만 남는 일은 없어야 한다.
  const after = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A2:B4`).then(r => r.json())
  const values = (after.items as Array<{row:number;column:number;value?:unknown}>)
    .filter(item=>item.column===1).map(item=>item.value)
  expect(values).toEqual(['사과','사과','배'])
})
