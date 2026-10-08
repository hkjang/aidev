import { expect, test } from '@playwright/test'

// 200행짜리 표를 인쇄하면 둘째 장부터는 어느 칸이 무슨 뜻인지 알 수 없었다.
// 화면에서 고정해 둔 행이 곧 머리글이므로 장마다 다시 찍어야 한다.
test('a frozen row is repeated at the top of every printed page', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`인쇄 머리글 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  const cells=[{row:1,column:1,value:'제품'},{row:1,column:2,value:'단가'}]
  for(let row=2;row<=60;row+=1){cells.push({row,column:1,value:`품목${row}`});cells.push({row,column:2,value:row*100})}
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`frozen-${stamp}`,cells}})
  const created=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${sheet}/layout:apply`,{headers:{'Idempotency-Key':`fr-${stamp}`},data:{
    expected_revision:created.sheets[0].layout.revision,action:'freeze',frozen_rows:1,frozen_columns:0}})

  await page.addInitScript(()=>{window.print=()=>{}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menuitem',{name:'파일'}).click()
  await page.getByRole('menuitem',{name:/^인쇄(?!\s*영역)/}).click()

  const printed=await (async()=>{
    for(let attempt=0;attempt<40;attempt+=1){
      const found=await page.evaluate(()=>{
        for(const frame of [...document.querySelectorAll('iframe')]){
          const doc=frame.contentDocument
          if(doc&&doc.querySelectorAll('tr').length>1)return {
            headText:[...doc.querySelectorAll('thead')].map(head=>head.textContent??'').join('|'),
            frozenRows:doc.querySelectorAll('thead tr.frozen').length,
            // 브라우저는 thead 를 장마다 다시 찍는다. 반복은 이 표시에 달렸다.
            repeats:doc.defaultView?.getComputedStyle(doc.querySelector('thead')!).display,
            bodyStarts:doc.querySelector('tbody tr')?.textContent??'',
          }
        }
        return undefined
      })
      if(found)return found
      await page.waitForTimeout(100)
    }
    throw new Error('the print document never appeared')
  })()

  expect(printed.frozenRows).toBe(1)
  expect(printed.headText).toContain('제품')
  expect(printed.headText).toContain('단가')
  expect(printed.repeats).toBe('table-header-group')
  // 머리글로 올라간 행이 본문 첫 줄로 또 나오면 안 된다.
  expect(printed.bodyStarts).toContain('품목2')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
