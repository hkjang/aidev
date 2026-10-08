import { expect, test } from '@playwright/test'

test('typing a formula suggests functions and shows the argument hint', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`수식 자동완성 ${Date.now()}`}}).then(response=>response.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await page.getByLabel('A1 셀 입력').press('=')
  await page.getByLabel('A1 셀 입력').fill('=SUMI')
  const suggestions=page.locator('.formula-suggest')
  await expect(suggestions.getByRole('option').first()).toContainText('SUMIF')
  // Tab accepts the highlighted function and leaves the caret inside the call.
  await page.getByLabel('A1 셀 입력').press('Tab')
  await expect(page.getByLabel('A1 셀 입력')).toHaveValue('=SUMIF(')
  await expect(suggestions.locator('.formula-signature .current')).toHaveText('범위')
})
