import { expect, test } from '@playwright/test'

// 행 하나를 지워 다른 곳의 수식 열두 개가 깨져도 화면은 아무 말이 없었다.
// 서버는 이미 어디가 깨졌는지 알려 주고 있었다.
test('an edit that breaks formulas says so and leads to the first one', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`참조 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`ref-${stamp}`,cells:[
    ...[1,2,3,4,5].map(row=>({row,column:1,value:row*10})),
    {row:1,column:3,formula:'=A3*2'},
    {row:2,column:3,formula:'=A3+1'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await expect(page.locator('.formula-issue')).toHaveCount(0)

  page.on('dialog',dialog=>void dialog.accept())
  const box=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(box.x+20,box.y+30+2*24+12,{button:'right'})
  await page.getByLabel('행 메뉴').getByRole('menuitem',{name:'행 3 삭제'}).click()

  const notice=page.locator('.formula-issue')
  await expect(notice).toContainText('수식 2곳이 오류가 되었습니다')
  // 몇 곳인지만 알려 주고 어디인지 말하지 않으면 찾아다녀야 한다.
  await expect(notice).toContainText('C1 #REF!')

  await notice.getByRole('button',{name:'보기'}).click()
  await expect(page.getByLabel('이름 상자')).toHaveValue('C1')
  await expect(notice).toHaveCount(0)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
