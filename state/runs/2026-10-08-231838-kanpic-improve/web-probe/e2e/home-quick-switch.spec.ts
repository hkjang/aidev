import { expect, test } from '@playwright/test'

// 빠른 이동은 편집기에만 있었다. 워크북 목록 화면에서 Ctrl/⌘+K 를 누르면
// 아무 일도 일어나지 않아, 단축키가 고장 난 것처럼 보였다.
//
// 목록에 보이는 워크북과 이 화면에서 할 수 있는 일을 모두 찾을 수 있어야
// 한다.
test('the workbook list opens quick switch and can jump to a workbook', async ({ page, request }) => {
  const stamp=Date.now()
  const shown=await request.post('/api/v1/workbooks',{data:{title:`빠른이동 ${stamp}`}}).then(r=>r.json())
  await page.goto('/')
  await expect(page.getByRole('heading',{name:'최근 워크북'})).toBeVisible()

  // 목록 화면에서 단축키가 창을 연다.
  await page.keyboard.press('ControlOrMeta+k')
  const box=page.getByPlaceholder('워크북 또는 명령 검색')
  await expect(box).toBeVisible()

  // 이 화면에서 할 수 있는 일도 함께 찾힌다.
  await box.fill('가져오기')
  await expect(page.getByRole('option',{name:/파일 가져오기/})).toBeVisible()

  // 워크북 이름으로 찾아 연다.
  await box.fill(`빠른이동 ${stamp}`)
  await page.keyboard.press('Enter')
  await expect(page).toHaveURL(new RegExp(`/workbooks/${shown.id}`))

  await request.delete(`/api/v1/workbooks/${shown.id}`)
})
