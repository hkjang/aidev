import { expect, test, type APIRequestContext } from '@playwright/test'

const cellValue=async (request:APIRequestContext,sheet:string,range:string)=>
  ((await (await request.get(`/api/v1/sheets/${sheet}/ranges/${range}`)).json()).items[0]?.value)

// A checkbox is what a task column wants: one click, no typing.
test('a checkbox cell toggles with a click and with the space key', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`체크박스 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}`,cells:[
    {row:1,column:1,value:'할 일'},{row:2,column:1,value:'보고서 초안'},{row:3,column:1,value:'검토 요청'},
  ]}})
  const validation=await request.post(`/api/v1/sheets/${sheet}/data-validations`,{data:{
    range:'B2:B3',rule_type:'checkbox',allow_blank:true,idempotency_key:`rule-${Date.now()}`,
  }})
  expect(validation.ok()).toBeTruthy()
  // The server fills in the pair of values a checkbox toggles between.
  expect((await validation.json()).options).toMatchObject([{value:true},{value:false}])

  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  const canvas=(await page.locator('.grid-canvas').boundingBox())!
  const box=(row:number)=>({x:canvas.x+46+108+54,y:canvas.y+27+(row-1)*27+13})

  // Clicking the box checks it without opening an editor.
  await page.mouse.click(box(2).x,box(2).y)
  await expect.poll(()=>cellValue(request,sheet,'B2:B2')).toBe(true)
  await page.mouse.click(box(2).x,box(2).y)
  await expect.poll(()=>cellValue(request,sheet,'B2:B2')).toBe(false)

  // Space flips the selected cell, which is how a keyboard user works down a
  // column of tasks.
  await page.locator('.name-box').fill('B3')
  await page.keyboard.press('Enter')
  await page.keyboard.press(' ')
  await expect.poll(()=>cellValue(request,sheet,'B3:B3')).toBe(true)

  // A cell outside the rule keeps its ordinary editing behaviour.
  await page.locator('.name-box').fill('C2')
  await page.keyboard.press('Enter')
  await page.keyboard.press('Enter')
  await expect(page.getByLabel('C2 셀 입력')).toBeFocused()
})

test('a checkbox can use the sheet own pair of values', async ({ request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`체크 값 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const created=await request.post(`/api/v1/sheets/${sheet}/data-validations`,{data:{
    range:'A1:A5',rule_type:'checkbox',idempotency_key:`rule-${Date.now()}`,
    options:[{value:'예'},{value:'아니오'}],
  }}).then(response=>response.json())
  expect(created.options).toMatchObject([{value:'예',label:'예'},{value:'아니오',label:'아니오'}])
  expect(created.show_dropdown).toBe(false)

  // Three values are not a checkbox.
  const refused=await request.post(`/api/v1/sheets/${sheet}/data-validations`,{data:{
    range:'B1:B5',rule_type:'checkbox',idempotency_key:`bad-${Date.now()}`,
    options:[{value:1},{value:2},{value:3}],
  }})
  expect(refused.status()).toBe(400)
})
