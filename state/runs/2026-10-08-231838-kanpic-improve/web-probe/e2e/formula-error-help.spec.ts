import { expect, test } from '@playwright/test'

// 셀에 `#VALUE!` 만 찍히면 무엇이 잘못됐는지도, 어떻게 고치는지도 알 수
// 없다. 마우스와 키보드 양쪽에서 이유가 닿아야 한다.
test('an error cell explains itself on hover and in the formula bar', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`오류 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`err-${stamp}`,cells:[
    {row:1,column:1,value:10},{row:2,column:1,value:0},
    {row:1,column:2,formula:'=A1/A2'},
    {row:2,column:2,formula:'=NOTAFUNCTION(1)'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const box=(await page.locator('.grid-canvas').boundingBox())!
  // 값이 멀쩡한 셀에는 안내가 없다.
  await page.mouse.click(box.x+48+53,box.y+30+12)
  await expect(page.locator('.formula-error')).toHaveCount(0)

  // 방향키로 옮겨도 이유가 보인다. 마우스를 올려야만 보이면 키보드로
  // 다니는 사람에게는 없는 것과 같다.
  await page.keyboard.press('ArrowRight')
  await expect(page.locator('.formula-error')).toContainText('#DIV/0!')
  await expect(page.locator('.formula-error')).toContainText('0으로 나눴습니다')
  await expect(page.locator('.formula-error')).toHaveAttribute('title',/IFERROR/)

  // 마우스를 올리면 다음에 할 일까지 함께 나온다.
  await page.mouse.move(box.x+48+53+94,box.y+30+40)
  const tip=page.locator('.cell-note.cell-error')
  await expect(tip).toBeVisible()
  await expect(tip).toContainText('#NAME?')
  await expect(tip).toContainText('함수 목록')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
