import { expect, test } from '@playwright/test'

// 서버가 500을 돌려주면 저장 대기줄은 그 항목을 지우지 않고 멈춘다(outbox.ts).
// 다음 저장이 같은 요청을 그대로 다시 보내고 또 500을 받는다. 한 칸에 적은
// 수식 하나가 워크북 전체의 저장을 영영 막는다.
//
// EXP(1000)은 무한대가 되고, Go의 JSON 인코더는 무한대를 담지 못해 500을
// 낸다. 엑셀과 시트는 #NUM! 을 돌려준다.
test('a formula that overflows does not wedge the save queue', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`무한대 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  const nameBox=page.getByRole('combobox',{name:'이름 상자'})
  const type=async(address:string,text:string)=>{
    await nameBox.fill(address)
    await nameBox.press('Enter')
    await page.locator('.grid-canvas').press('Enter')
    await page.keyboard.type(text)
    await page.keyboard.press('Enter')
  }
  await type('A1','=EXP(1000)')
  // 넘치는 수식 뒤에 적은 값이 서버에 닿아야 한다. 대기줄이 막히면 닿지 않는다.
  await type('B1','42')

  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A1:B1`).then(r=>r.json())
    const byColumn=new Map(range.items.map((cell:{column:number;value:unknown})=>[cell.column,cell.value]))
    return {a:byColumn.get(1),b:byColumn.get(2)}
  },{timeout:15000}).toEqual({a:'#NUM!',b:42})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
