import { expect, test } from '@playwright/test'

const printedText=async(page:import('@playwright/test').Page)=>{
  // The print frame is removed a second after it opens, so the document is
  // read as soon as it has rows rather than after a fixed wait.
  let text=''
  for(let attempt=0;attempt<40&&text==='';attempt+=1){
    text=await page.evaluate(()=>{
      for(const frame of [...document.querySelectorAll('iframe')]){
        const doc=frame.contentDocument
        if(doc&&doc.querySelectorAll('tr').length>1)return doc.body.innerText
      }
      return ''
    })
    if(text==='')await page.waitForTimeout(100)
  }
  return text
}

// 화면에서 걸러 낸 행이 종이에 나오면, 인쇄한 사람이 방금 본 화면과 다른
// 문서를 들고 있게 된다.
test('printing leaves out rows a filter hides', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`인쇄 필터 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','매출'],['서울',100],['부산',50],['대구',30]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`ph-seed-${stamp}`,cells}})
  await request.post(`/api/v1/sheets/${sheet}/filter-views`,{data:{
    idempotency_key:`ph-filter-${stamp}`,name:'서울만',range:'A1:B4',header_rows:1,active:true,
    criteria:[{column:1,operator:'values',values:['서울']}],
  }})

  await page.addInitScript(()=>{window.print=()=>{}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menuitem',{name:'파일'}).click()
  await page.getByRole('menuitem',{name:/^인쇄(?!\s*영역)/}).click()

  const text=await printedText(page)
  expect(text).toContain('서울')
  expect(text).not.toContain('부산')
  expect(text).not.toContain('대구')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 접어 둔 그룹도 마찬가지다: 소계만 남기고 접었으면 인쇄물도 소계만 나와야 한다.
test('printing leaves out rows folded into a collapsed group', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`인쇄 접기 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','매출'],['부산',80],['부산',95],['서울',120]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`pg-seed-${stamp}`,cells}})
  await request.patch(`/api/v1/sheets/${sheet}/layout:apply`,{data:{expected_revision:1,idempotency_key:`pg-group-${stamp}`,action:'group',axis:'row',start:2,count:2}})
  await request.patch(`/api/v1/sheets/${sheet}/layout:apply`,{data:{expected_revision:2,idempotency_key:`pg-collapse-${stamp}`,action:'collapse',axis:'row',start:2,count:2}})

  await page.addInitScript(()=>{window.print=()=>{}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menuitem',{name:'파일'}).click()
  await page.getByRole('menuitem',{name:/^인쇄(?!\s*영역)/}).click()

  const text=await printedText(page)
  expect(text).toContain('서울')
  expect(text).not.toContain('부산')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
