import { expect, test } from '@playwright/test'

// 1월을 끌면 2월이 나와야 한다. 어느 스프레드시트에서든 되는 일인데
// kanpic 은 1월만 계속 복사했다.
test('dragging the fill handle carries months and weekdays forward', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`채우기 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`fill-${stamp}`,cells:[
    {row:1,column:1,value:'11월'},{row:1,column:2,value:'월요일'},{row:1,column:3,value:'3반'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const box=(await page.locator('.grid-canvas').boundingBox())!
  // A1 을 고르고 채우기 손잡이를 세 칸 아래로 끈다.
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A1')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await expect(page.locator('.name-box')).toHaveValue('A1')
  const handle={x:box.x+46+108-2,y:box.y+27+27-2}
  await page.mouse.move(handle.x,handle.y)
  await page.mouse.down()
  await page.mouse.move(handle.x,handle.y+27*3,{steps:6})
  await page.mouse.up()

  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A4`).then(r=>r.json())).items
    return items.map((cell:{value:unknown})=>cell.value)
  // 12월 다음은 13월이 아니라 1월이다.
  }).toEqual(['11월','12월','1월','2월'])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
