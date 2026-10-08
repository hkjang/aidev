import { expect, test } from '@playwright/test'

// Reviewing a column starts with the same few questions, so the panel answers
// them without anybody writing a formula.
test('the column stats panel summarises the selected column', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`열 통계 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const cells=[{row:1,column:1,value:'지역'},{row:1,column:2,value:'매출'}]
  const regions=['서울','부산','서울','대구','서울']
  regions.forEach((region,index)=>{
    cells.push({row:index+2,column:1,value:region})
    cells.push({row:index+2,column:2,value:(index+1)*1000})
  })
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  await page.locator('.name-box').fill('B2')
  await page.keyboard.press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'열 통계'}).click()

  const panel=page.getByRole('region',{name:'열 통계 패널'})
  await expect(panel).toContainText('B열 통계')
  // The label on top of the numbers is not counted as one of them.
  await expect(panel).toContainText('1행은 머리글로 보고 제외했습니다')
  await expect(panel.locator('.stats-grid').first()).toContainText('5')
  await expect(panel).toContainText('15,000')
  await expect(panel).toContainText('3,000')

  // A text column reports its repeats instead of arithmetic.
  await page.locator('.name-box').fill('A2')
  await page.keyboard.press('Enter')
  await expect(panel).toContainText('A열 통계')
  await expect(panel.locator('.stats-row').first()).toContainText('서울')
  await expect(panel.locator('.stats-row').first()).toContainText('3')
  await expect(panel).not.toContainText('표준편차')
})
