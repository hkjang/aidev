import { expect, test, type APIRequestContext } from '@playwright/test'

const seed=async (request:APIRequestContext)=>{
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`열 필터 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','매출'],['서울',4200000],['부산',1850000],['서울',3100000],['대구',970000],['서울',5600000]]
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}`,
    cells:rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))}})
  await request.post(`/api/v1/sheets/${sheet}/filter-views`,{data:{
    name:'기본 필터',range:'A1:B6',header_rows:1,active:true,criteria:[],idempotency_key:`filter-${Date.now()}`}})
  return {workbook,sheet}
}

// Ticking values in the header is how a filter is actually used, so it has to
// work without opening the filter dialog at all.
test('the column filter button filters by the values it lists', async ({ page, request }) => {
  const {workbook,sheet}=await seed(request)
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  const canvas=(await page.locator('.grid-canvas').boundingBox())!
  // The funnel sits at the right edge of the header of every filtered column.
  await page.mouse.click(canvas.x+46+108-14,canvas.y+13)

  const menu=page.getByRole('dialog',{name:'A열 필터'})
  // Most frequent first, then Korean alphabetical order for the ties.
  await expect(menu.getByRole('checkbox')).toHaveText([/서울\s*3/,/대구\s*1/,/부산\s*1/])

  await menu.getByRole('button',{name:'선택 해제'}).click()
  await menu.getByRole('checkbox',{name:/서울/}).click()
  await menu.getByRole('button',{name:'적용'}).click()

  // The filter now keeps only the 서울 rows, which the status bar counts.
  await expect.poll(async()=>{
    const views=(await (await request.get(`/api/v1/sheets/${sheet}/filter-views`)).json()).items
    return views[0]?.criteria
  }).toEqual([{column:1,operator:'values',values:['서울']}])
  // Three data rows survive; the header row is not counted.
  await expect(page.locator('.sheet-status')).toContainText('필터 3행')

  // Reopening shows what is kept, and putting everything back clears the rule.
  await page.mouse.click(canvas.x+46+108-14,canvas.y+13)
  const reopened=page.getByRole('dialog',{name:'A열 필터'})
  await expect(reopened.getByRole('checkbox',{name:/서울/})).toHaveAttribute('aria-checked','true')
  await expect(reopened.getByRole('checkbox',{name:/부산/})).toHaveAttribute('aria-checked','false')
  await reopened.getByRole('button',{name:'전체 선택'}).click()
  await reopened.getByRole('button',{name:'적용'}).click()
  await expect.poll(async()=>{
    const views=(await (await request.get(`/api/v1/sheets/${sheet}/filter-views`)).json()).items
    return views[0]?.criteria?.length??0
  }).toBe(0)
})

// The search narrows a long list, and the bulk buttons act on what it shows.
test('searching inside the filter menu narrows what select all applies to', async ({ page, request }) => {
  const {workbook,sheet}=await seed(request)
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  const canvas=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(canvas.x+46+108-14,canvas.y+13)

  const menu=page.getByRole('dialog',{name:'A열 필터'})
  await menu.getByRole('button',{name:'선택 해제'}).click()
  await menu.getByLabel('값 검색').fill('부')
  await expect(menu.getByRole('checkbox')).toHaveCount(1)
  await menu.getByRole('button',{name:'전체 선택'}).click()
  await menu.getByRole('button',{name:'적용'}).click()

  await expect.poll(async()=>{
    const views=(await (await request.get(`/api/v1/sheets/${sheet}/filter-views`)).json()).items
    return views[0]?.criteria
  }).toEqual([{column:1,operator:'values',values:['부산']}])
})
