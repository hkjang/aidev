import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

async function seedWorkbook(request:APIRequestContext,title:string,sheets:string[]=[]){
  const workbook=await request.post('/api/v1/workbooks',{data:{title}}).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${workbook.sheets[0].id}/cells:batch`,{data:{
    base_version:workbook.version,idempotency_key:`nav-seed-${workbook.id}`,
    cells:[{row:1,column:1,value:'항목'},{row:2,column:1,value:'임대료'},{row:2,column:2,value:1200}],
  }})
  for(const name of sheets)await request.post(`/api/v1/workbooks/${workbook.id}/sheets`,{data:{name}})
  return workbook
}

async function openEditor(page:Page,workbookId:string){
  await page.goto(`/workbooks/${workbookId}`)
  await page.waitForSelector('.grid-canvas')
  await page.waitForTimeout(800)
}

test('the quick switcher jumps to sheets, cells and commands', async ({ page, request }) => {
  const workbook=await seedWorkbook(request,`빠른 이동 ${Date.now()}`,['월별 추이','보관'])
  await openEditor(page,workbook.id)

  // Ctrl/⌘+K opens the palette and filters by sheet name.
  await page.keyboard.press('Control+k')
  const palette=page.getByRole('dialog',{name:'빠른 이동'})
  await expect(palette).toBeVisible()
  await palette.getByRole('textbox',{name:'빠른 이동 검색'}).fill('월별')
  // Workbook suggestions load asynchronously, so the target option is clicked
  // once it is on screen instead of relying on the highlighted row.
  await palette.getByRole('option',{name:/월별 추이/}).first().click()
  await expect(palette).toHaveCount(0)
  await expect(page.locator('.sheet-tab-main.active')).toContainText('월별 추이')

  // A typed A1 address becomes a jump target.
  await page.keyboard.press('Control+k')
  await page.getByRole('textbox',{name:'빠른 이동 검색'}).fill('C7')
  await expect(page.getByRole('option',{name:/C7/})).toBeVisible()
  await page.keyboard.press('Enter')
  await expect(page.locator('.name-box')).toHaveValue('C7')

  // Commands run from the same list.
  await page.keyboard.press('Control+k')
  await page.getByRole('textbox',{name:'빠른 이동 검색'}).fill('모든 시트 관리')
  // Other tests leave similarly named workbooks behind, so the command entry is
  // chosen explicitly instead of relying on ranking.
  await page.getByRole('option',{name:/모든 시트 관리/}).first().click()
  await expect(page.getByRole('dialog',{name:'시트 관리'})).toBeVisible()
})

test('every dialog closes with Escape and restores focus', async ({ page, request }) => {
  const workbook=await seedWorkbook(request,`모달 UX ${Date.now()}`)
  await openEditor(page,workbook.id)

  const shareButton=page.getByRole('button',{name:/공유/})
  await shareButton.click()
  const share=page.getByRole('dialog',{name:/공유/})
  await expect(share).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(share).toHaveCount(0)
  // Focus returns to the control that opened the dialog.
  await expect(shareButton).toBeFocused()

  await page.getByRole('button',{name:'모든 시트 관리'}).click()
  await expect(page.getByRole('dialog',{name:'시트 관리'})).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog',{name:'시트 관리'})).toHaveCount(0)

  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'정렬',exact:true}).click()
  await page.getByRole('menuitem',{name:'범위 정렬…'}).click()
  await expect(page.getByRole('dialog',{name:'범위 정렬'})).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog',{name:'범위 정렬'})).toHaveCount(0)
})

test('the admin console summarises sharing exposure and restricts a workbook', async ({ page, request }) => {
  const workbook=await seedWorkbook(request,`거버넌스 ${Date.now()}`)
  await request.patch(`/api/v1/workbooks/${workbook.id}/sharing`,{data:{link_access:'anyone',link_role:'editor'}})

  await page.goto('/admin?tab=overview')
  await expect(page.getByRole('heading',{name:'개요'})).toBeVisible()
  await page.locator('.attention-row',{hasText:'링크가 있는 모든 사용자에게 공개'}).getByRole('button',{name:'목록 보기'}).click()
  await expect(page.getByRole('heading',{name:'워크북 거버넌스'})).toBeVisible()

  const row=page.locator('.governance-row',{hasText:workbook.title})
  await expect(row).toContainText('링크 공개')
  page.once('dialog',dialog=>dialog.accept())
  await row.getByRole('button',{name:'공개 해제'}).click()
  await expect(page.getByRole('status')).toContainText('링크 액세스를 제한했습니다')

  const sharing=await request.get(`/api/v1/workbooks/${workbook.id}/sharing`).then(response=>response.json())
  expect(sharing.sharing.link_access).toBe('restricted')
})

test('comments show the author display name instead of a long identifier', async ({ page, request }) => {
  const actor=`comment.author.${Date.now()}@corp.example`
  await request.post('/api/v1/admin/users',{data:{user_id:actor,display_name:'댓글 작성자',email:actor}})
  const workbook=await seedWorkbook(request,`이름 표시 ${Date.now()}`)
  await request.put(`/api/v1/workbooks/${workbook.id}/shares`,{data:{principal_type:'user',principal_id:actor,role:'commenter'}})
  await request.post(`/api/v1/workbooks/${workbook.id}/comments`,{
    headers:{'X-Kanpic-Actor':actor},
    data:{sheet_id:workbook.sheets[0].id,range:'A1',content:'확인 부탁드립니다',idempotency_key:`comment-${Date.now()}`},
  })

  await openEditor(page,workbook.id)
  await page.getByRole('button',{name:'댓글',exact:true}).click()
  const panel=page.getByRole('complementary',{name:'댓글 패널'})
  await expect(panel).toContainText('댓글 작성자')
  await expect(panel).not.toContainText(actor)
})
