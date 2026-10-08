import { expect, test } from '@playwright/test'

// 다른 곳에서 받은 보고서는 병합투성이다. 부서 이름이 세 행에 걸쳐 한 번만
// 적혀 있으면 사람 눈에는 읽기 좋지만 그 표는 정렬도 피벗도 되지 않는다.
// 풀고 채우면 쓸 수 있는 자료가 되는 것까지 통째로 확인한다.
test('unmerging fills the hidden rows and unblocks the cleanups', async ({ page }) => {
  const alerts:string[] = []
  page.on('dialog', async d => { alerts.push(d.message()); await d.accept() })
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(r => r.json())
  const sheetId = workbook.sheets[0].id as string
  const box = { merge: { start_row: 2, start_column: 1, end_row: 4, end_column: 1 } }
  const write = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: workbook.version, idempotency_key: 'um-seed', cells: [
      {row:1,column:1,value:'부서'},{row:1,column:2,value:'금액'},
      {row:2,column:1,value:'영업1팀',style:box},{row:2,column:2,value:100},
      {row:3,column:1,style:box},{row:3,column:2,value:200},
      {row:4,column:1,style:box},{row:4,column:2,value:300},
    ],
  }})
  expect(write.status(), await write.text()).toBeLessThan(300)

  const canvas = page.locator('canvas.grid-canvas')
  await canvas.click({ position: { x: 70, y: 42 } })
  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '데이터 정리' }).click()
  await page.getByRole('menuitem', { name: '병합 해제하고 채우기…' }).click()
  const dialog = page.getByRole('dialog', { name: '병합 해제하고 채우기' })
  await expect(dialog.getByText('A2:A4')).toBeVisible({ timeout: 15_000 })
  await dialog.getByRole('button', { name: '해제하고 채우기' }).click()

  // 가려져 있던 두 행에 부서가 들어가고 병합은 사라진다.
  await expect.poll(async()=>{
    const result = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A2:A4`).then(r => r.json())
    const rows = result.items as Array<{row:number;value?:unknown;style?:Record<string,unknown>}>
    return rows.map(item=>`${item.value ?? ''}${item.style?.merge?'+병합':''}`).join(',')
  },{timeout:15_000}).toBe('영업1팀,영업1팀,영업1팀')

  // 여기까지 오면 표는 쓸 수 있는 자료다. 병합 때문에 멈추던 정리가 이제 열린다.
  await page.getByRole('menuitem', { name: '데이터' }).click()
  await page.getByRole('menuitem', { name: '데이터 정리' }).click()
  await page.getByRole('menuitem', { name: '중복 항목 삭제…' }).click()
  await expect(page.getByRole('dialog', { name: '중복 항목 삭제' })).toBeVisible({ timeout: 15_000 })
  expect(alerts, '조용히 실패한 것이 있다').toEqual([])
})
