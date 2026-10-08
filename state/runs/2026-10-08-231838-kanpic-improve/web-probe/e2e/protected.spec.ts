import { expect, test, type APIRequestContext } from '@playwright/test'

const write=(request:APIRequestContext,sheet:string,actor:string,row:number,column:number)=>
  request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{headers:{'X-Kanpic-Actor':actor},
    data:{idempotency_key:`write-${actor}-${row}-${Date.now()}`,cells:[{row,column,value:'수정'}]}})

// Sharing decides who opens a workbook; a protected range decides who may
// change the cells a model depends on.
test('a protected range refuses writes from everyone but its editors', async ({ request }) => {
  const workbook=await request.post('/api/v1/workbooks',{headers:{'X-Kanpic-Actor':'model.owner@corp.example'},
    data:{title:`범위 보호 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  // Both collaborators can edit the workbook; only one is on the protection.
  for(const person of ['model.reader@corp.example','model.analyst@corp.example']){
    await request.put(`/api/v1/workbooks/${workbook.id}/shares`,{headers:{'X-Kanpic-Actor':'model.owner@corp.example'},
      data:{principal_type:'user',principal_id:person,role:'editor'}})
  }
  const created=await request.post(`/api/v1/sheets/${sheet}/protected-ranges`,{headers:{'X-Kanpic-Actor':'model.owner@corp.example'},
    data:{range:'B2:C5',description:'요율표',editors:['model.analyst@corp.example'],idempotency_key:`p-${Date.now()}`}})
  expect(created.status()).toBe(201)

  // A workbook editor without a place on the list is refused, with the reason.
  const blocked=await write(request,sheet,'model.reader@corp.example',3,2)
  expect(blocked.status()).toBe(403)
  const failure=await blocked.json()
  expect(failure.error.code).toBe('range_protected')
  expect(failure.error.message).toContain('요율표')

  // The listed editor and the owner may write, and so may anyone outside it.
  expect((await write(request,sheet,'model.analyst@corp.example',3,2)).ok()).toBeTruthy()
  expect((await write(request,sheet,'model.owner@corp.example',4,3)).ok()).toBeTruthy()
  expect((await write(request,sheet,'model.reader@corp.example',9,9)).ok()).toBeTruthy()
})

// The console for it is one list and one form, reachable from the data menu.
test('the data menu protects the selected range and lists it', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`보호 화면 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  await page.locator('.name-box').fill('B2:C4')
  await page.keyboard.press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'범위 보호…'}).click()

  const dialog=page.getByRole('dialog',{name:'범위 보호'})
  await expect(dialog.getByLabel('보호할 범위')).toHaveValue('B2:C4')
  await dialog.getByLabel('보호 설명').fill('가정 입력값')
  await dialog.getByRole('button',{name:'이 범위 보호'}).click()
  await expect(dialog.locator('.protected-row')).toContainText('B2:C4')
  await expect(dialog.locator('.protected-row')).toContainText('가정 입력값')

  await expect.poll(async()=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/protected-ranges`)).json()).items
    return items.length
  }).toBe(1)

  // Removing it puts the range back in everyone's hands.
  await dialog.getByLabel('B2:C4 보호 해제').click()
  await expect.poll(async()=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/protected-ranges`)).json()).items
    return items.length
  }).toBe(0)
})
