import { expect, test } from '@playwright/test'

// 색 눈금은 색맹인 사람에게 아무 말도 하지 않고, 흑백으로 인쇄하면 모두에게
// 같은 회색이 된다. 엑셀이 아이콘 집합을 두는 이유이고, kanpic 에는 없었다.
//
// 아이콘은 그려져야 뜻이 있으므로 여기서는 캔버스에 실제로 칠해졌는지 본다.
// 격자는 평소 짙은 빨강이나 초록을 쓰지 않으므로, 그런 점이 생겼다는 것은
// 아이콘이 그려졌다는 뜻이다.
const countColoredPixels=`(() => {
  const canvas=document.querySelector('.grid-canvas')
  const context=canvas.getContext('2d')
  const {data}=context.getImageData(0,0,canvas.width,canvas.height)
  let count=0
  for(let at=0;at<data.length;at+=4){
    const [red,green,blue]=[data[at],data[at+1],data[at+2]]
    const high=Math.max(red,green,blue),low=Math.min(red,green,blue)
    if(high-low>70&&high>90)count+=1
  }
  return count
})()`

test('an icon set draws a readable mark next to the value', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`아이콘 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`icons-${stamp}`,cells:[
    {row:1,column:1,value:0},{row:2,column:1,value:25},{row:3,column:1,value:50},
    {row:4,column:1,value:75},{row:5,column:1,value:100},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const plain=await page.evaluate(countColoredPixels)

  await page.getByRole('button',{name:'조건부 서식'}).click()
  const dialog=page.getByRole('dialog',{name:'조건부 서식'})
  await expect(dialog).toBeVisible()
  await dialog.getByLabel('조건부 서식 범위').fill('A1:A5')
  await dialog.getByLabel('조건부 서식 유형').selectOption('icon_set')
  await dialog.getByLabel('아이콘 종류').selectOption('3Arrows')
  await dialog.getByRole('button',{name:/저장|추가|만들기/}).first().click()

  // 서버가 셋으로 나눈다. 엑셀과 같은 33%·67% 자리다.
  await expect.poll(async()=>{
    const evaluated=await request.get(`/api/v1/sheets/${sheet}/conditional-formats:evaluate?range=A1:A5`).then(r=>r.json())
    return (evaluated.items??[]).map((item:{row:number;icon?:{index:number}})=>[item.row,item.icon?.index])
  }).toEqual([[1,0],[2,0],[3,1],[4,2],[5,2]])

  await dialog.getByRole('button',{name:'조건부 서식 닫기'}).click()
  await expect.poll(()=>page.evaluate(countColoredPixels),{timeout:10000}).toBeGreaterThan(plain+100)

  // 아이콘이 값을 밀어낼 뿐 가리거나 대신하지는 않는다.
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A5')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await expect(page.locator('.formula-input')).toHaveValue('100')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
