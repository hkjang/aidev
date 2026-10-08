import { expect, test } from '@playwright/test'

// "이 범위가 바뀌면 알려줘" 를 걸고 끄고 그만두는 길이 실제로 열리는지 본다.
// 거는 길이 없으면 있는 기능이 아니다.
test('a watch rule can be added, switched off and dropped', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  await page.locator('canvas.grid-canvas').waitFor()

  await page.getByRole('menuitem', { name: '삽입' }).click()
  await page.getByRole('menuitem', { name: '변경 알림…' }).click()
  const dialog = page.getByRole('dialog', { name: '변경 알림' })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('아직 지켜보는 범위가 없습니다.')).toBeVisible()

  await dialog.getByLabel('지켜볼 범위').fill('A1:B10')
  await dialog.getByLabel('지켜보기 이름').fill('매출표')
  await dialog.getByRole('button', { name: '지켜보기' }).click()
  await expect(dialog.getByText('매출표')).toBeVisible()

  // 켜고 끄는 것이 남아야 한다. 끈 규칙은 셀이 바뀌어도 알리지 않는다.
  const toggle = dialog.getByLabel('매출표 알림 켜기')
  await expect(toggle).toBeChecked()
  // 체크박스는 서버가 받아들인 뒤에 움직인다. 눌러 놓고 기다린다.
  await toggle.click()
  await expect(toggle).not.toBeChecked()

  await dialog.getByRole('button', { name: '매출표 지켜보기 그만두기' }).click()
  await expect(dialog.getByText('아직 지켜보는 범위가 없습니다.')).toBeVisible()
})
