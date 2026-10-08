import { expect, test } from '@playwright/test'

// A note annotates a cell for whoever reads it later. It is not a comment:
// nobody replies, and it must not disturb the value it sits on.
test('a note is written from the cell menu, shown on hover and removed again', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`메모 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}`,cells:[
    {row:2,column:2,value:1200,style:{bold:true}},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  await page.locator('.name-box').fill('B2')
  await page.keyboard.press('Enter')
  const canvas=(await page.locator('.grid-canvas').boundingBox())!
  const cell={x:canvas.x+46+108+50,y:canvas.y+27+27+13}
  await page.mouse.click(cell.x,cell.y,{button:'right'})
  // The note sits beside the comment, in the insert submenu.
  await page.getByLabel('셀 메뉴').getByRole('menuitem',{name:'삽입',exact:true}).click()
  await page.getByRole('menuitem',{name:'메모 삽입…'}).click()
  await page.getByLabel('메모 내용').fill('협력사 확정 단가')
  await page.getByRole('button',{name:'저장',exact:true}).click()

  // The note reaches the server without touching the value or the formatting.
  await expect.poll(async()=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/B2:B2`)).json()).items
    return items[0]
  }).toMatchObject({value:1200,note:'협력사 확정 단가',style:{bold:true}})

  // Hovering the cell shows it.
  await page.mouse.move(cell.x-20,cell.y)
  await page.mouse.move(cell.x,cell.y)
  await expect(page.getByRole('tooltip')).toHaveText('협력사 확정 단가')
  await page.mouse.move(canvas.x+46+30,canvas.y+27+27*5)
  await expect(page.getByRole('tooltip')).toBeHidden()

  // Removing the note leaves the cell exactly as it was.
  await page.mouse.click(cell.x,cell.y,{button:'right'})
  await page.getByLabel('셀 메뉴').getByRole('menuitem',{name:'삽입',exact:true}).click()
  await page.getByRole('menuitem',{name:'메모 편집…'}).click()
  await page.getByRole('button',{name:'메모 삭제'}).click()
  await expect.poll(async()=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/B2:B2`)).json()).items
    return items[0]?.note??''
  }).toBe('')
  const cellAfter=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/B2:B2`)).json()).items[0]
  expect(cellAfter).toMatchObject({value:1200,style:{bold:true}})
})
