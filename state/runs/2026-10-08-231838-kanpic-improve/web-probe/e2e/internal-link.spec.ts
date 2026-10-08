import { expect, test } from '@playwright/test'

// 목차에서 요약으로 뛰는 링크는 스프레드시트에서 가장 흔한 링크인데, 주소를
// 손으로 만들 수 있는 사람은 없다. 대화상자가 만들어 주고, 누르면 페이지를
// 새로 읽지 않고 그 자리로 이동해야 한다.
test('a link to a range in this workbook moves the selection without reloading', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`내부 링크 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`il-seed-${stamp}`,cells:[
    {row:1,column:1,value:'목차'},{row:20,column:3,value:'여기가 요약'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const nameBox=page.getByRole('combobox',{name:'이름 상자'})
  await nameBox.fill('A2')
  await nameBox.press('Enter')
  await page.getByRole('menuitem',{name:'삽입'}).click()
  await page.getByRole('menuitem',{name:'링크…'}).click()
  const dialog=page.getByRole('dialog',{name:'링크 삽입'})
  await dialog.getByRole('tab',{name:'이 워크북의 범위'}).click()
  await dialog.getByLabel('링크 범위').fill('C20')
  await dialog.getByLabel('링크 표시 텍스트').fill('요약으로 이동')
  await dialog.getByRole('button',{name:'링크 넣기'}).click()

  // 링크는 셀에 바로 들어간다. 반쯤 입력된 수식으로 남지 않는다.
  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A2:A2`).then(response=>response.json())
    return range.items[0]?.value
  },{timeout:15_000}).toBe('요약으로 이동')

  await nameBox.fill('A2')
  await nameBox.press('Enter')
  const chip=page.locator('.cell-link a')
  await expect(chip).toHaveText('이 워크북 · C20')
  const before=page.url()
  await chip.click()
  await expect(nameBox).toHaveValue('C20')
  expect(page.url()).toBe(before)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
