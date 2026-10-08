import { expect, test } from '@playwright/test'

// 브라우저 기본 prompt는 이름표를 붙일 수도, 값을 검사할 수도 없고, 어떤
// 환경에서는 아예 뜨지 않는다. 이름을 바꿨는데 아무 일도 안 일어나는 것이
// 가장 나쁜 결과라 실제 대화상자로 바꿨다.
test('renaming a workbook and a sheet uses a real dialog that validates', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`이름 변경 ${stamp}`}}).then(response=>response.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  await page.getByRole('menuitem',{name:'파일'}).click()
  await page.getByRole('menuitem',{name:/이름 변경/}).click()
  const rename=page.getByRole('dialog',{name:'워크북 이름 변경'})
  // 빈 이름은 조용히 무시되지 않고 이유가 표시된다.
  await rename.getByRole('textbox',{name:'워크북 이름'}).fill('   ')
  await rename.getByRole('button',{name:'이름 바꾸기'}).click()
  await expect(page.getByRole('alert')).toHaveText('이름을 입력하세요.')
  await rename.getByRole('textbox',{name:'워크북 이름'}).fill(`분기 보고서 ${stamp}`)
  await rename.getByRole('button',{name:'이름 바꾸기'}).click()
  await expect.poll(async()=>(await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())).title,{timeout:15_000})
    .toBe(`분기 보고서 ${stamp}`)

  // 시트 이름도 같은 대화상자를 쓴다.
  await page.getByRole('button',{name:'모든 시트 관리'}).click()
  await page.getByRole('button',{name:'Sheet1 이름 변경'}).click()
  const sheetRename=page.getByRole('dialog',{name:'시트 이름 변경'})
  await sheetRename.getByRole('textbox',{name:'시트 이름'}).fill('요약')
  await sheetRename.getByRole('button',{name:'이름 바꾸기'}).click()
  await expect.poll(async()=>{
    const latest=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
    return latest.sheets[0].name
  },{timeout:15_000}).toBe('요약')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
