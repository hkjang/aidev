import { expect, test } from '@playwright/test'

// 긴 수식일수록 수식 입력창에 쓰는데, 함수 제안과 인수 안내는 셀 안에서만
// 나오고 있었다. 두 곳의 조작이 같아야 손이 헷갈리지 않는다.
test('the formula bar offers the same function suggestions as the cell', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`fx ${Date.now()}`}}).then(response=>response.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  const bar=page.getByLabel('수식 입력창')
  await bar.click()
  await bar.type('=TEXTB',{delay:15})
  const list=page.getByLabel('함수 제안')
  await expect(list).toBeVisible()
  await expect(list.getByRole('option',{name:/TEXTBEFORE/})).toBeVisible()
  // 목록이 한 번에 하나만 떠야 한다. 셀 편집기와 겹쳐 두 개가 뜨면 어느
  // 쪽이 Tab을 받는지 알 수 없다.
  await expect(list).toHaveCount(1)

  await page.keyboard.press('Tab')
  await expect(bar).toHaveValue('=TEXTBEFORE(')
  await bar.type('A1,"-"',{delay:15})
  await expect(list.locator('.formula-signature .current')).toHaveText('구분자')

  // 이름을 다 친 사람이 Tab을 눌렀을 때 더 긴 이름이 들어가면 안 된다.
  await page.keyboard.press('Escape')
  await page.keyboard.press('Escape')
  await bar.click()
  await bar.fill('')
  await bar.type('=TEXT',{delay:15})
  await expect(list.getByRole('option').first()).toContainText('TEXT')
  await page.keyboard.press('Tab')
  await expect(bar).toHaveValue('=TEXT(')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
