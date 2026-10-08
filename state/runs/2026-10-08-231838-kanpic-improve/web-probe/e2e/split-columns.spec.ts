import { expect, test } from '@playwright/test'

// 분할은 오른쪽 열을 덮어쓴다. 미리보기가 없으면 무엇이 어떻게 나뉘는지도,
// 무엇을 잃는지도 모른 채 실행하게 된다.
test('splitting text previews the columns and warns about what it overwrites', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`분할 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`sp-seed-${stamp}`,cells:[
    {row:1,column:1,value:'주소'},{row:1,column:2,value:'비고'},
    {row:2,column:1,value:'"서울, 강남구",100,완료'},{row:2,column:2,value:'기존값'},
    {row:3,column:1,value:'"부산, 해운대구",250,진행'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A2:A3')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'텍스트를 열로 분할…'}).click()
  const dialog=page.getByRole('dialog',{name:'텍스트를 열로 분할'})

  // 미리보기는 따옴표 안의 쉼표를 나누지 않는다.
  await expect(dialog.locator('.split-preview td').first()).toHaveText('서울, 강남구')
  await expect(dialog.locator('.split-summary')).toContainText('자동 감지: 쉼표 · 3개 열로 나뉩니다.')
  await expect(dialog.locator('.split-summary')).toContainText('오른쪽 2개 열의 기존 값을 덮어씁니다.')

  // 나눌 수 없는 구분 기호를 고르면 실행을 막는다.
  await dialog.getByRole('radio',{name:'세미콜론'}).click()
  await expect(dialog.getByRole('button',{name:'분할'})).toBeDisabled()
  await dialog.getByRole('radio',{name:'자동 감지'}).click()
  await dialog.getByRole('button',{name:'분할'}).click()

  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A2:C2`).then(response=>response.json())
    return range.items.sort((first:{column:number},second:{column:number})=>first.column-second.column).map((cell:{value:unknown})=>cell.value)
  },{timeout:15_000}).toEqual(['서울, 강남구',100,'완료'])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
