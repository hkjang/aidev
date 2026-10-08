import { expect, test, type Page } from '@playwright/test'

// 다른 스프레드시트에서 복사하면 클립보드에는 평문과 HTML 이 함께 올라간다.
// 평문에는 **보이던 글자**만 담기므로 `₩1,234` 를 그대로 저장하면 합계가
// 0이 된다. 서식도 전부 사라졌다.
//
// 이 시나리오는 반드시 실제 브라우저에서 돌아야 한다. 이 앱의 CSP 에는
// `style-src 'unsafe-inline'` 이 없어 크롬이 style 속성을 CSSOM 에 넣지
// 않는다. jsdom 에는 CSP 가 없어 이 차이를 잡지 못한다.
const pasteInto=async(page:Page,html:string,text:string)=>{
  await page.evaluate(([html,text])=>{
    const data=new DataTransfer()
    data.setData('text/html',html)
    data.setData('text/plain',text)
    document.querySelector('.grid-viewport')?.dispatchEvent(new ClipboardEvent('paste',{clipboardData:data,bubbles:true,cancelable:true}))
  },[html,text])
}

const openAtA1=async(page:Page,workbookId:string)=>{
  await page.goto(`/workbooks/${workbookId}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const box=(await page.locator('.grid-canvas').boundingBox())!
  // 행 머리글을 넘어선 자리라야 A1 이 선택된다. 머리글을 누르면 행 전체가
  // 잡혀 복사 한도에 걸린다.
  await page.mouse.click(box.x+80,box.y+42)
}

test('pasting a formatted range from Excel keeps its numbers and its formatting', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`엑셀 붙여넣기 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await openAtA1(page,workbook.id)
  await pasteInto(page,
    '<html xmlns:x="urn:schemas-microsoft-com:office:excel"><body><table><tr>'
    +'<td style="font-weight:700;background:#DBEAFE;text-align:center">제품</td>'
    +'<td style="font-weight:700;background:#DBEAFE">단가</td></tr><tr>'
    +'<td>연필</td><td align=right x:num="1234.5" style="color:#B91C1C">₩1,234.50</td></tr><tr>'
    +'<td>공책</td><td align=right x:num="98000">₩98,000</td></tr></table></body></html>',
    '제품\t단가\n연필\t₩1,234.50\n공책\t₩98,000')

  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:B3`).then(r=>r.json())).items
    return items.length
  }).toBe(6)
  const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:B3`).then(r=>r.json())).items
  const at=(row:number,column:number)=>items.find((cell:{row:number;column:number})=>cell.row===row&&cell.column===column)
  // 보이던 글자가 아니라 셀이 담고 있던 숫자로 들어와야 합계가 성립한다.
  expect(at(2,2)).toMatchObject({value:1234.5})
  expect(at(3,2)).toMatchObject({value:98000})
  expect(at(1,1)).toMatchObject({style:{bold:true,background:'#DBEAFE',horizontal_align:'center'}})
  expect(at(2,2)!.style).toMatchObject({color:'#B91C1C'})

  const current=await request.get(`/api/v1/workbooks/${workbook.id}`).then(r=>r.json())
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:current.version,idempotency_key:`sum-${stamp}`,cells:[{row:5,column:2,formula:'=SUM(B2:B3)'}]}})
  const total=(await request.get(`/api/v1/sheets/${sheet}/ranges/B5:B5`).then(r=>r.json())).items[0]
  expect(total).toMatchObject({value:99234.5})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

test('pasting from Google Sheets uses the value it declares, not the text it shows', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`시트 붙여넣기 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await openAtA1(page,workbook.id)
  await pasteInto(page,
    '<meta charset="utf-8"><table><tbody><tr>'
    +'<td data-sheets-value=\'{"1":2,"2":"분기"}\' style="font-style:italic">분기</td>'
    +'<td data-sheets-value=\'{"1":3,"3":0.125}\' style="text-align:right">12.5%</td></tr><tr>'
    +'<td colspan="2" style="background:rgb(254, 226, 226)">두 칸을 차지하는 셀</td></tr><tr>'
    +'<td>다음 줄</td><td>1,234</td></tr></tbody></table>',
    '분기\t12.5%\n두 칸을 차지하는 셀\t\n다음 줄\t1,234')

  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:B3`).then(r=>r.json())).items
    return items.length
  }).toBe(5)
  const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:B3`).then(r=>r.json())).items
  const at=(row:number,column:number)=>items.find((cell:{row:number;column:number})=>cell.row===row&&cell.column===column)
  expect(at(1,1)).toMatchObject({value:'분기',style:{italic:true}})
  expect(at(1,2)).toMatchObject({value:0.125})
  expect(at(2,1)).toMatchObject({value:'두 칸을 차지하는 셀',style:{background:'#FEE2E2'}})
  // colspan 이 걸린 셀이 두 칸을 차지하므로 다음 줄이 왼쪽으로 밀리면 안 된다.
  expect(at(3,1)).toMatchObject({value:'다음 줄'})
  expect(at(3,2)).toMatchObject({value:1234,style:{number_format:'#,##0'}})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 복사할 때 평문만 올리면 엑셀·구글 시트·워드 어디에 붙여넣어도 굵게·색·
// 정렬이 사라진 글자만 남는다. 스프레드시트끼리 서식이 오가는 길은
// `text/html` 표뿐이다.
test('copying puts a formatted table on the clipboard, not only plain text', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`복사 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`copy-${stamp}`,cells:[
    {row:1,column:1,value:'제품',style:{bold:true,background:'#DBEAFE',horizontal_align:'center'}},
    {row:1,column:2,value:'단가',style:{bold:true,background:'#DBEAFE'}},
    {row:2,column:1,value:'연필',style:{italic:true}},
    {row:2,column:2,value:1234.5,style:{color:'#B91C1C',number_format:'#,##0.00'}},
  ]}})
  await openAtA1(page,workbook.id)
  // 데이터 영역 끝까지 확장하는 단축키는 이미 들어온 값으로 끝을 찾는다.
  // 여기서는 A1:B2 만 필요하므로 한 칸씩 넓힌다.
  await page.keyboard.press('Shift+ArrowRight')
  await page.keyboard.press('Shift+ArrowDown')

  const copy=()=>page.evaluate(()=>{
    const data=new DataTransfer()
    document.querySelector('.grid-viewport')?.dispatchEvent(new ClipboardEvent('copy',{clipboardData:data,bubbles:true,cancelable:true}))
    return {html:data.getData('text/html'),text:data.getData('text/plain')}
  })
  // 셀 값이 도착하기 전에 복사하면 빈 표가 나온다.
  await expect.poll(async()=>(await copy()).text).toContain('제품')
  const copied=await copy()
  expect(copied.text).toContain('제품\t단가')
  expect(copied.html).toContain('<table')
  expect(copied.html).toContain('font-weight:700')
  expect(copied.html).toContain('background-color:#DBEAFE')
  expect(copied.html).toContain('font-style:italic')
  expect(copied.html).toContain('color:#B91C1C')
  // 받는 쪽이 통화 표기를 못 읽어도 숫자는 숫자로 들어가야 한다.
  expect(copied.html).toContain('x:num="1234.5"')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 취소선은 이 앱 어디에서나 `strike` 다. 클립보드로 내보내는 쪽만 `strikethrough`
// 라는, 아무도 쓰지 않는 이름을 읽고 있었다. 읽는 쪽에는 아예 없었다. 그래서
// 취소선과 글꼴은 엑셀과 오갈 때 양쪽 모두에서 조용히 사라졌다.
test('a strikethrough and a font name survive the trip from Excel and back', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`취소선 글꼴 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await openAtA1(page,workbook.id)
  await pasteInto(page,
    '<html xmlns:x="urn:schemas-microsoft-com:office:excel"><body><table><tr>'
    +'<td style="text-decoration:line-through;font-family:맑은 고딕">지난 계획</td>'
    // 서버는 72pt 까지 받는다. 더 큰 것을 그대로 보내면 붙여넣기 전체가 400 이 된다.
    +'<td style="font-size:80pt;font-family:\'Malgun Gothic\', sans-serif">제목</td></tr></table></body></html>',
    '지난 계획\t제목')

  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:B1`).then(r=>r.json())).items
    return items.length
  }).toBe(2)
  const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/A1:B1`).then(r=>r.json())).items
  const at=(row:number,column:number)=>items.find((cell:{row:number;column:number})=>cell.row===row&&cell.column===column)
  expect(at(1,1)!.style).toMatchObject({strike:true,font_family:'맑은 고딕'})
  expect(at(1,2)!.style).toMatchObject({font_size:72,font_family:'Malgun Gothic'})

  // 다시 복사하면 클립보드의 HTML 이 그 둘을 다시 담아야 한다.
  const box=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(box.x+80,box.y+42)
  const copied=await page.evaluate(()=>{
    const data=new DataTransfer()
    document.querySelector('.grid-viewport')?.dispatchEvent(new ClipboardEvent('copy',{clipboardData:data,bubbles:true,cancelable:true}))
    return data.getData('text/html')
  })
  expect(copied).toContain('line-through')
  expect(copied).toContain('font-family:맑은 고딕')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
