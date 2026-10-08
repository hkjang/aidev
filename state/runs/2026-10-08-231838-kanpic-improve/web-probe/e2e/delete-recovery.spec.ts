import { expect, test } from '@playwright/test'

// 행을 지우면 그 행을 가리키던 수식은 `=A3*2` 에서 `=#REF!*2` 로 영구히
// 다시 쓰인다. 서버는 셀 단위로 되돌리지 못하지만 삭제 직전 자동 백업을
// 남긴다. 그 회수 경로가 화면에 없으면 사용자는 지워진 것을 손으로 다시
// 입력해야 한다.
test('a deleted row can be put back from the notice, formula and all', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`복구 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`del-${stamp}`,cells:[
    ...[1,2,3,4,5].map(row=>({row,column:1,value:row*10})),
    {row:1,column:3,formula:'=A3*2'},
  ]}})
  const read=async(range:string)=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/${range}`)).json()).items as Array<{row:number;value?:unknown;formula?:string}>
    return items
  }

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  page.on('dialog',dialog=>void dialog.accept())
  const box=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(box.x+20,box.y+30+2*24+12,{button:'right'})
  await page.getByLabel('행 메뉴').getByRole('menuitem',{name:'행 3 삭제'}).click()

  const notice=page.locator('.formula-issue')
  await expect(notice).toContainText('C1 #REF!')
  // 삭제로 수식 원문이 사라진 것을 확인해 둔다. 되돌리기가 되살리는 것이
  // 행만이 아니라는 뜻이다.
  await expect.poll(async()=>(await read('C1:C1'))[0]?.formula).toBe('=#REF!*2')

  await notice.getByRole('button',{name:'되돌리기'}).click()
  await expect.poll(async()=>(await read('A1:A5')).map(item=>item.value)).toEqual([10,20,30,40,50])
  expect((await read('C1:C1'))[0]?.formula).toBe('=A3*2')
  await expect(notice).toHaveCount(0)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 삽입은 지운 것이 없으니 안내할 것도 없다. 모든 편집마다 안내가 뜨면
// 안내는 배경이 되고 정작 위험한 것을 놓친다.
test('inserting a row says nothing', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`삽입 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`ins-${stamp}`,cells:[
    ...[1,2,3].map(row=>({row,column:1,value:row*10})),
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const box=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(box.x+20,box.y+30+2*24+12,{button:'right'})
  await page.getByLabel('행 메뉴').getByRole('menuitem',{name:'위에 행 1개 삽입'}).click()
  await expect.poll(async()=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A4`)).json()).items as Array<{row:number}>
    return items.map(item=>item.row)
  }).toEqual([1,2,4])
  await expect(page.locator('.formula-issue')).toHaveCount(0)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
