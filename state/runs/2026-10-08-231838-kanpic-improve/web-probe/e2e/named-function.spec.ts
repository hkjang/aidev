import { expect, test } from '@playwright/test'

// 이름 있는 수식은 팀에서 쓰는 셈을 한 번 정의해 두고 함수처럼 부르는
// 것이다. 만들어 두고 아무도 못 찾으면 없는 것과 같으므로, 메뉴에서
// 만들고 셀에서 부르는 길을 통째로 확인한다.
test('a named function is defined once and called from a cell', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response => response.json())
  const sheetId = workbook.sheets[0].id as string
  const canvas = page.locator('canvas.grid-canvas')
  const edit = async (position:{x:number;y:number}, value:string) => {
    await canvas.dblclick({ position })
    await page.locator('.cell-editor').fill(value)
    await page.locator('.cell-editor').press('Enter')
  }
  await edit({x:70,y:42},'100')
  await edit({x:170,y:42},'60')
  const cells = async (range:string) =>
    page.request.get(`/api/v1/sheets/${sheetId}/ranges/${range}`).then(response => response.json())
  await expect.poll(async()=>{
    const result = await cells('A1:B1')
    return result.items.map((cell:{value:unknown})=>cell.value)
  },{timeout:15_000}).toEqual([100,60])

  // 이름 범위 옆, 삽입 메뉴에서 정의한다.
  await page.getByRole('menuitem', { name: '삽입' }).click()
  await page.getByRole('menuitem', { name: '이름 있는 수식…' }).click()
  const dialog = page.getByRole('dialog', { name: '이름 있는 수식' })
  await dialog.getByLabel('수식 이름').fill('마진율')
  await dialog.getByLabel('매개변수').fill('매출, 원가')
  await dialog.getByLabel('수식 본문').fill('(매출-원가)/매출')
  await dialog.getByLabel('수식 설명').fill('매출에서 원가를 뺀 몫')
  await dialog.getByRole('button', { name: '저장' }).click()
  // 저장하면 왼쪽 목록에 나타난다.
  await expect(dialog.getByText('마진율(매출, 원가)')).toBeVisible()
  await dialog.getByRole('button', { name: '이름 있는 수식 닫기' }).click()

  // 셀에서 함수처럼 부른다.
  await edit({x:270,y:42},'=마진율(A1,B1)')
  await expect.poll(async()=>{
    const result = await cells('C1:C1')
    return result.items.map((cell:{value:unknown})=>cell.value)
  },{timeout:15_000}).toEqual([0.4])

  // 정의를 고치면 그것을 쓰는 칸이 따라 바뀐다. 이것이 되지 않으면 한 번
  // 정의해 두는 뜻이 없다.
  await page.getByRole('menuitem', { name: '삽입' }).click()
  await page.getByRole('menuitem', { name: '이름 있는 수식…' }).click()
  await dialog.getByRole('button', { name: /마진율/ }).click()
  await dialog.getByLabel('수식 본문').fill('(매출-원가)/원가')
  await dialog.getByRole('button', { name: '저장' }).click()
  await dialog.getByRole('button', { name: '이름 있는 수식 닫기' }).click()
  await expect.poll(async()=>{
    const result = await cells('C1:C1')
    return result.items.map((cell:{value:unknown})=>Number(cell.value).toFixed(4))
  },{timeout:15_000}).toEqual(['0.6667'])
})
