import { expect, test } from '@playwright/test'

// 넓은 표를 종이 폭에 맞춰 통째로 눌러 담으면 열이 서른 개일 때 한 열이 몇
// 밀리미터로 찌그러져 읽을 수 없다. 들어가는 만큼 찍고 나머지는 다음 장으로
// 넘겨야 한다.
const printedTables=async(page:import('@playwright/test').Page)=>{
  for(let attempt=0;attempt<40;attempt+=1){
    const found=await page.evaluate(()=>{
      for(const frame of [...document.querySelectorAll('iframe')]){
        const doc=frame.contentDocument
        if(doc&&doc.querySelectorAll('tr').length>1){
          const tables=[...doc.querySelectorAll('table')]
          return {
            tables:tables.length,
            headers:tables.filter(table=>table.querySelector('thead')).length,
            rowHeads:tables.filter(table=>table.querySelector('th.row-head')).length,
            text:doc.body.innerText,
            stretched:[...doc.querySelectorAll('style')].some(style=>style.textContent?.includes('width:100%')),
          }
        }
      }
      return undefined
    })
    if(found)return found
    await page.waitForTimeout(100)
  }
  throw new Error('the print document never appeared')
}

test('a table wider than the page carries on over more pages instead of being squeezed', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`넓은 표 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  const cells=[]
  for(let column=1;column<=14;column+=1){
    cells.push({row:1,column,value:`열${column}`})
    cells.push({row:2,column,value:column*10})
  }
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`wide-${stamp}`,cells}})
  const created=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${sheet}/layout:apply`,{headers:{'Idempotency-Key':`w-${stamp}`},data:{
    expected_revision:created.sheets[0].layout.revision,action:'resize',axis:'column',start:1,count:14,size:160}})

  await page.addInitScript(()=>{window.print=()=>{}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menuitem',{name:'파일'}).click()
  await page.getByRole('menuitem',{name:/^인쇄(?!\s*영역)/}).click()

  const printed=await printedTables(page)
  expect(printed.tables).toBeGreaterThan(1)
  // 장마다 열 머리글과 행 번호가 다시 찍혀야 어느 칸인지 알 수 있다.
  expect(printed.headers).toBe(printed.tables)
  expect(printed.rowHeads).toBe(printed.tables)
  expect(printed.stretched).toBe(false)
  // 열은 하나도 빠지지 않는다.
  for(let column=1;column<=14;column+=1)expect(printed.text).toContain(`열${column}`)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
