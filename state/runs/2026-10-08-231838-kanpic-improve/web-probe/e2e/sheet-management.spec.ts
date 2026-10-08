import { expect, test, type APIRequestContext, type Browser, type Page } from '@playwright/test'

async function seedWorkbook(request:APIRequestContext,title:string,sheetNames:string[]=[]){
  const workbook=await request.post('/api/v1/workbooks',{data:{title}}).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${workbook.sheets[0].id}/cells:batch`,{data:{
    base_version:workbook.version,idempotency_key:`sheet-seed-${workbook.id}`,
    cells:[{row:1,column:1,value:'항목'},{row:2,column:1,value:'임대료'},{row:2,column:2,value:1200},{row:3,column:2,formula:'=B2*12'}],
  }})
  for(const name of sheetNames)await request.post(`/api/v1/workbooks/${workbook.id}/sheets`,{data:{name}})
  return workbook
}

async function openEditor(page:Page,workbookId:string){
  await page.goto(`/workbooks/${workbookId}`)
  await page.waitForSelector('.grid-canvas')
  await page.waitForTimeout(800)
}

test('the sheet manager reports data, hides a sheet and reorders tabs', async ({ page, request }) => {
  const workbook=await seedWorkbook(request,`시트 관리 ${Date.now()}`,['보조','보관'])
  await openEditor(page,workbook.id)

  await page.getByRole('button',{name:'모든 시트 관리'}).click()
  const manager=page.getByRole('dialog',{name:'시트 관리'})
  await expect(manager).toContainText('3개 시트')
  const firstRow=manager.locator('.sheet-manager-row').first()
  await expect(firstRow).toContainText('4셀')
  await expect(firstRow).toContainText('A1:B3')

  // Hiding a sheet from the manager removes its tab and offers it back.
  await manager.getByRole('button',{name:'보관 숨기기'}).click()
  await expect(manager.locator('.sheet-manager-row',{hasText:'보관'})).toContainText('숨김')
  await manager.getByRole('button',{name:'시트 관리 닫기'}).click()
  await expect(page.locator('.sheet-tab-main',{hasText:'보관'})).toHaveCount(0)
  await page.getByRole('button',{name:/숨긴 시트 1/}).click()
  await page.getByRole('button',{name:'보관 숨김 해제'}).click()
  await expect(page.locator('.sheet-tab-main',{hasText:'보관'})).toHaveCount(1)

  // Reordering from the manager moves the sheet in the strip too.
  await page.getByRole('button',{name:'모든 시트 관리'}).click()
  await page.getByRole('button',{name:'보관 왼쪽으로 이동'}).click()
  await expect(page.locator('.sheet-manager-open')).toHaveText(['Sheet1','보관','보조'])
  await page.getByRole('button',{name:'시트 관리 닫기'}).click()
  const order=await page.locator('.sheet-tab-main span').allInnerTexts()
  expect(order).toEqual(['Sheet1','보관','보조'])
})

test('the last visible sheet cannot be hidden', async ({ page, request }) => {
  const workbook=await seedWorkbook(request,`숨김 검증 ${Date.now()}`)
  await openEditor(page,workbook.id)
  await page.locator('.sheet-tab-menu-trigger').click()
  await expect(page.getByRole('menuitem',{name:'시트 숨기기'})).toBeDisabled()

  const rejected=await page.request.patch(`/api/v1/sheets/${workbook.sheets[0].id}`,{data:{hidden:true}})
  expect(rejected.status()).toBe(400)
})

test('a sheet copies into another workbook the user can edit', async ({ page, request }) => {
  const source=await seedWorkbook(request,`복사 원본 ${Date.now()}`)
  const target=await request.post('/api/v1/workbooks',{data:{title:`복사 대상 ${Date.now()}`}}).then(response=>response.json())
  await openEditor(page,source.id)

  await page.locator('.sheet-tab-menu-trigger').click()
  await page.getByRole('menuitem',{name:'다른 워크북으로 복사'}).click()
  const dialog=page.getByRole('dialog',{name:'다른 워크북으로 복사'})
  await dialog.getByLabel('대상 워크북').selectOption(target.id)
  page.once('dialog',confirmation=>confirmation.dismiss())
  await dialog.getByRole('button',{name:'복사'}).click()
  await expect(dialog).toHaveCount(0)

  const copied=await request.get(`/api/v1/workbooks/${target.id}`).then(response=>response.json())
  expect(copied.sheets).toHaveLength(2)
  const stats=await request.get(`/api/v1/workbooks/${target.id}/sheet-stats`).then(response=>response.json())
  const copiedStats=stats.items.find((item:{sheet_id:string})=>item.sheet_id!==target.sheets[0].id)
  expect(copiedStats.non_empty_cells).toBe(4)
})

test('a deleted workbook can be restored from the trash and purged', async ({ page, request }) => {
  const workbook=await seedWorkbook(request,`휴지통 ${Date.now()}`)
  expect((await request.delete(`/api/v1/workbooks/${workbook.id}`)).ok()).toBeTruthy()

  await page.goto('/')
  await page.getByRole('button',{name:/휴지통/}).click()
  const row=page.locator('.trash-row',{hasText:workbook.title})
  await expect(row).toBeVisible()
  await row.getByRole('button',{name:'복원'}).click()
  await expect(page.locator('.trash-row',{hasText:workbook.title})).toHaveCount(0)

  const restored=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
  expect(restored.sheets).toHaveLength(1)

  await request.delete(`/api/v1/workbooks/${workbook.id}`)
  await page.reload()
  await page.getByRole('button',{name:/휴지통/}).click()
  page.once('dialog',confirmation=>confirmation.accept())
  await page.locator('.trash-row',{hasText:workbook.title}).getByRole('button',{name:'완전 삭제'}).click()
  await expect(page.locator('.trash-row',{hasText:workbook.title})).toHaveCount(0)
  expect((await request.get(`/api/v1/workbooks/${workbook.id}`)).status()).toBe(404)
})

test('a shared viewer keeps a personal star and no sheet management', async ({ page, request, browser }) => {
  const workbook=await seedWorkbook(request,`개인 즐겨찾기 ${Date.now()}`)
  await request.put(`/api/v1/workbooks/${workbook.id}/shares`,{data:{principal_type:'user',principal_id:'e2e-star',role:'viewer'}})

  const context=await browser.newContext({extraHTTPHeaders:{'X-Kanpic-Actor':'e2e-star'},viewport:{width:1400,height:900}})
  const viewer=await context.newPage()
  await viewer.goto('/')
  const card=viewer.locator('.workbook-card',{hasText:workbook.title})
  await card.getByRole('button',{name:new RegExp(`${workbook.title} 더보기`)}).click()
  await viewer.getByRole('menuitem',{name:'즐겨찾기'}).click()
  await viewer.getByRole('button',{name:/즐겨찾기$/}).click()
  await expect(viewer.locator('.workbook-card',{hasText:workbook.title})).toBeVisible()

  // The owner's own list is unaffected by the viewer's star.
  await page.goto('/')
  await page.getByRole('button',{name:/즐겨찾기$/}).click()
  await expect(page.locator('.workbook-card',{hasText:workbook.title})).toHaveCount(0)

  await openEditor(viewer,workbook.id)
  await expect(viewer.getByRole('button',{name:'시트 추가'})).toHaveCount(0)
  await viewer.locator('.sheet-tab-menu-trigger').click()
  await expect(viewer.getByRole('menuitem',{name:'이름 변경'})).toBeDisabled()
  await expect(viewer.getByRole('menuitem',{name:'다른 워크북으로 복사'})).toBeEnabled()
  await context.close()
})
