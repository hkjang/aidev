import { expect, test, type Page } from '@playwright/test'

// 스프레드시트는 `1,234` 나 `12%` 를 글자가 아니라 숫자로 받는다. kanpic 은
// 글자로 저장해 합계에 들어가지 않았다. 붙여넣기는 이미 숫자로 읽으므로
// 같은 값을 손으로 치면 결과가 달라졌다.
const typeAt=async(page:Page,address:string,text:string)=>{
  await page.getByRole('combobox',{name:'이름 상자'}).fill(address)
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.keyboard.type(text)
  await page.keyboard.press('Enter')
}

test('a typed 1,234 is a number that sums, and keeps the look it was typed in', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`입력 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await typeAt(page,'A1','1,234')
  await typeAt(page,'A2','₩1,000')
  await typeAt(page,'A3','12.5%')
  await typeAt(page,'A4','(500)')
  await typeAt(page,'A5','=SUM(A1:A2)')
  // 숫자로 볼 수 없는 것은 글자로 남아야 한다.
  await typeAt(page,'B1','1,2')
  await typeAt(page,'B2','010-1234-5678')
  await typeAt(page,'B3','2026-08-23')

  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:B5`).then(r=>r.json())).items
    return items.length
  }).toBe(8)
  const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:B5`).then(r=>r.json())).items
  const at=(address:string)=>{
    const column=address.charCodeAt(0)-64,row=Number(address.slice(1))
    return items.find((cell:{row:number;column:number})=>cell.row===row&&cell.column===column)
  }
  expect(at('A1')).toMatchObject({value:1234,style:{number_format:'#,##0'}})
  expect(at('A2')).toMatchObject({value:1000,style:{number_format:'"₩"#,##0'}})
  expect(at('A3')).toMatchObject({value:0.125,style:{number_format:'0.0%'}})
  expect(at('A4')).toMatchObject({value:-500})
  // 글자였다면 합계가 0이 된다.
  expect(at('A5')).toMatchObject({value:2234})
  expect(at('B1')).toMatchObject({value:'1,2'})
  expect(at('B2')).toMatchObject({value:'010-1234-5678'})
  expect(at('B3')).toMatchObject({value:'2026-08-23'})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 이미 서식을 지정해 둔 칸에 값을 넣는 것은 서식을 바꾸겠다는 뜻이 아니다.
test('typing into a cell that already has a format leaves that format alone', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`서식 유지 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`fmt-${stamp}`,cells:[
    {row:1,column:1,value:1,style:{number_format:'0.00',bold:true}},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await typeAt(page,'A1','1,234')
  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A1`).then(r=>r.json())).items
    return items[0]?.value
  }).toBe(1234)
  const cell=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A1`).then(r=>r.json())).items[0]
  expect(cell.style).toMatchObject({number_format:'0.00',bold:true})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
