import { expect, test, type Page } from '@playwright/test'

async function unreadableVisibleText(page:Page){
  return page.locator('body *').evaluateAll(elements=>elements.flatMap(element=>{
    const style=getComputedStyle(element),rect=element.getBoundingClientRect()
    const hasOwnText=Array.from(element.childNodes).some(node=>node.nodeType===Node.TEXT_NODE&&Boolean(node.textContent?.trim()))
    if(!hasOwnText||style.display==='none'||style.visibility==='hidden'||Number(style.opacity)===0||rect.width===0||rect.height===0)return[]
    const size=Number.parseFloat(style.fontSize)
    return size<11?[{element:element.tagName.toLowerCase(),className:element.className,text:element.textContent?.trim().slice(0,60),size}]:[]
  }))
}

test('login and profile menus expose the same build version', async ({ page }) => {
  const build = await page.request.get('/api/v1/version').then(response => response.json())
  await page.goto('/login')
  await expect(page.getByText(`kanpic ${build.version}`)).toBeVisible()
  await page.goto('/')
  await page.locator('.profile-trigger').click()
  await expect(page.locator('.version-menu')).toContainText(`kanpic ${build.version}`)
})

test('admin console and personal settings are separate surfaces', async ({ page }) => {
  // The console now opens on the overview and keeps settings on its own tab.
  await page.goto('/admin')
  await expect(page.getByRole('heading', { name: '개요' })).toBeVisible()
  await page.goto('/admin?tab=settings')
  await expect(page.getByRole('heading', { name: '시스템 설정' })).toBeVisible()
  await expect(page.getByText('Keycloak OIDC 간편 연결')).toBeVisible()
  await expect(page.getByText('워크북 자동화 실행 정책')).toBeVisible()
  await page.getByRole('button', { name: /서버 로그/ }).click()
  await expect(page.getByRole('heading', { name: '서버 로그' })).toBeVisible()
  await page.goto('/preferences')
  await expect(page.getByRole('heading', { name: '나만의 작업 환경' })).toBeVisible()
  await page.getByRole('button', { name: 'API 키' }).click()
  await expect(page.getByRole('heading', { name: '개인 API 키' })).toBeVisible()
})

test('creates a workbook and opens the virtual canvas editor', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  await expect(page.locator('canvas.grid-canvas')).toBeVisible()
  await expect(page.locator('.formula-bar')).toBeVisible()
  await expect(page.getByText('AI 도우미', { exact: true })).toBeVisible()
  await page.screenshot({ path: 'test-results/kanpic-editor.png', fullPage: true })
})

test('every editor right panel uses the shared resizer', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`패널 너비 ${Date.now()}`}}).then(response=>response.json())
  try{
    await page.setViewportSize({width:860,height:720})
    await page.goto(`/workbooks/${workbook.id}`)
    await page.waitForSelector('.grid-canvas')
    const toolbar=page.locator('.toolbar')
    const panels=[
      ['AI 도우미','AI 도우미'],
      ['자동화 패널','자동화'],
      ['차트 패널','차트'],
      ['피벗 패널','피벗'],
      ['댓글','댓글'],
      ['편집 충돌','편집 충돌'],
      ['버전 이력','버전 이력'],
    ] as const
    for(const [buttonName,panelName] of panels){
      if(buttonName!=='AI 도우미')await toolbar.getByRole('button',{name:buttonName,exact:true}).evaluate(button=>(button as HTMLButtonElement).click())
      const separator=page.getByRole('separator',{name:`${panelName} 패널 너비 조절`})
      await expect(separator).toBeVisible()
      const before=Number(await separator.getAttribute('aria-valuenow'))
      await separator.press('ArrowRight')
      await expect(separator).toHaveAttribute('aria-valuenow',String(before-16))
    }
  }finally{
    await request.delete(`/api/v1/workbooks/${workbook.id}`)
  }
})

test('keeps visible interface text readable across primary surfaces', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`가독성 검증 ${Date.now()}`}}).then(response=>response.json())
  try{
    for(const path of ['/',`/workbooks/${workbook.id}`,'/admin']){
      await page.goto(path)
      await page.waitForLoadState('networkidle')
      expect(await unreadableVisibleText(page),`${path}에 11px 미만의 표시 텍스트가 있습니다.`).toEqual([])
    }
  }finally{
    await request.delete(`/api/v1/workbooks/${workbook.id}`)
  }
})


test('previews, runs, audits, triggers, and undoes PostgreSQL automations', async ({ page }) => {
  const versions=await page.request.get('/api/v1/admin/settings/versions').then(response=>response.json())
  const restoreRevision=versions.items[0].revision as number
  const putSetting=(key:string,value:unknown)=>page.request.put(`/api/v1/admin/settings/${key}`,{data:{key,value,value_type:typeof value==='boolean'?'boolean':'number',secret:false,description:`E2E ${key}`}})
  let workbookId=''
  let webhookKeyId=''
  try{
    expect((await putSetting('automation.max_cells_per_run',1000)).ok()).toBe(true)
    expect((await putSetting('automation.max_runs_per_hour',1000)).ok()).toBe(true)
    expect((await putSetting('automation.scheduler_poll_seconds',5)).ok()).toBe(true)
    expect((await putSetting('automation.enabled',true)).ok()).toBe(true)
    const tested=await page.request.post('/api/v1/admin/settings:test',{data:{}}).then(response=>response.json())
    expect(tested.items.find((item:{name:string})=>item.name==='자동화 저장소')).toMatchObject({success:true})

    const created=await page.request.post('/api/v1/workbooks',{data:{title:`자동화 ${Date.now()}`,workspace_id:'default'}}).then(response=>response.json())
    workbookId=created.id as string
    const sheetId=created.sheets[0].id as string
    const seeded=await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:1,idempotency_key:`automation-seed-${workbookId}`,cells:[{row:1,column:1,value:5}]}}).then(response=>response.json())
    expect(seeded.server_version).toBe(2)
    await page.goto(`/workbooks/${workbookId}`)
    await page.getByRole('button',{name:'자동화 패널'}).click()
    const panel=page.getByRole('complementary',{name:'자동화 패널'})
    await expect(panel).toBeVisible()
    await panel.getByRole('button',{name:/새 자동화/}).click()
    await panel.getByLabel('자동화 이름').fill('두 배 계산')
    await panel.getByLabel('자동화 작업').selectOption('set_formula')
    await panel.getByLabel('작업 범위').fill('B1')
    await panel.getByLabel('자동화 수식').fill('=A1*2')
    await panel.getByRole('button',{name:/저장 후 검증/}).click()
    await expect(panel.getByText('실행 미리보기')).toBeVisible()
    await expect(panel.getByText('=A1*2')).toBeVisible()
    expect((await page.request.get(`/api/v1/sheets/${sheetId}/ranges/B1`).then(response=>response.json())).items).toHaveLength(0)
    await panel.getByRole('button',{name:/검토한 자동화 실행/}).click()
    await expect.poll(async()=>{const range=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/B1`).then(response=>response.json());return range.items[0]?.value}).toBe(10)
    await expect(panel.getByText('성공',{exact:true})).toBeVisible()
    await panel.getByRole('button',{name:'Undo'}).click()
    await expect.poll(async()=>{const range=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/B1`).then(response=>response.json());return range.items.length}).toBe(0)
    await expect(panel.getByText('Undo 완료',{exact:true})).toBeVisible()

    const current=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
    const triggered=await page.request.post(`/api/v1/workbooks/${workbookId}/automations`,{data:{name:'A2 감시',enabled:true,idempotency_key:`trigger-create-${workbookId}`,trigger:{type:'cell_change',sheet_id:sheetId,range:'A2'},action:{type:'set_value',sheet_id:sheetId,range:'C2',value:'triggered'}}}).then(response=>response.json())
    const changed=await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:current.version,idempotency_key:`trigger-source-${workbookId}`,cells:[{row:2,column:1,value:1}]}}).then(response=>response.json())
    expect(changed.server_version).toBe(current.version+1)
    await expect.poll(async()=>{const range=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/C2`).then(response=>response.json());return range.items[0]?.value}).toBe('triggered')
    const triggerRuns=await page.request.get(`/api/v1/automations/${triggered.id}/runs`).then(response=>response.json())
    expect(triggerRuns.items).toHaveLength(1)
    expect(triggerRuns.items[0]).toMatchObject({trigger_type:'cell_change',status:'succeeded',trigger_operation_id:changed.operation_id})

    const scheduled=await page.request.post(`/api/v1/workbooks/${workbookId}/automations`,{data:{name:'서울 매시간 갱신',enabled:true,idempotency_key:`schedule-create-${workbookId}`,trigger:{type:'schedule',cron:'0 * * * *',timezone:'Asia/Seoul'},action:{type:'set_value',sheet_id:sheetId,range:'D2',value:'scheduled'}}}).then(response=>response.json())
    expect(scheduled).toMatchObject({name:'서울 매시간 갱신',trigger:{type:'schedule',cron:'0 * * * *',timezone:'Asia/Seoul'}})
    expect(Date.parse(scheduled.next_run_at)).not.toBeNaN()
    const schedulePreview=await page.request.post(`/api/v1/automations/${scheduled.id}:test`,{data:{}}).then(response=>response.json())
    expect(schedulePreview.changes).toEqual([expect.objectContaining({address:'D2',after:{value:'scheduled'}})])

    const webhook=await page.request.post(`/api/v1/workbooks/${workbookId}/automations`,{data:{name:'외부 승인 수신',enabled:true,idempotency_key:`webhook-create-${workbookId}`,trigger:{type:'webhook'},action:{type:'set_value',sheet_id:sheetId,range:'E2',value:'received'}}}).then(response=>response.json())
    const unauthenticated=await page.request.post(`/api/v1/automations/${webhook.id}:webhook`,{headers:{'Idempotency-Key':`delivery-${workbookId}`},data:{event:'approved'}})
    expect(unauthenticated.status()).toBe(401)
    const webhookKey=await page.request.post('/api/v1/me/api-keys',{data:{name:'E2E webhook',scopes:['automation.webhook.invoke'],expires_at:null}}).then(response=>response.json())
    webhookKeyId=webhookKey.id as string
    const webhookHeaders={Authorization:`Bearer ${webhookKey.secret}`,'Idempotency-Key':`delivery-${workbookId}`}
    const delivered=await page.request.post(`/api/v1/automations/${webhook.id}:webhook`,{headers:webhookHeaders,data:{event:'approved',sensitive:'not-persisted'}}).then(response=>response.json())
    expect(delivered.run).toMatchObject({trigger_type:'webhook',trigger_key_id:webhookKeyId,status:'succeeded'})
    expect(delivered.run.payload_digest).toMatch(/^[a-f0-9]{64}$/)
    expect(delivered.run.payload_bytes).toBeGreaterThan(0)
    await expect.poll(async()=>{const range=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/E2`).then(response=>response.json());return range.items[0]?.value}).toBe('received')
    const duplicateDelivery=await page.request.post(`/api/v1/automations/${webhook.id}:webhook`,{headers:webhookHeaders,data:{event:'changed'}}).then(response=>response.json())
    expect(duplicateDelivery.run).toMatchObject({id:delivered.run.id,duplicate:true,payload_digest:delivered.run.payload_digest})
    const webhookRuns=await page.request.get(`/api/v1/automations/${webhook.id}/runs`).then(response=>response.json())
    expect(webhookRuns.items).toHaveLength(1)
    expect(webhookRuns.items[0]).toMatchObject({trigger_type:'webhook',trigger_key_id:webhookKeyId,payload_digest:delivered.run.payload_digest})
    expect((await page.request.delete(`/api/v1/me/api-keys/${webhookKeyId}`)).status()).toBe(204)
    const revoked=await page.request.post(`/api/v1/automations/${webhook.id}:webhook`,{headers:{...webhookHeaders,'Idempotency-Key':`delivery-revoked-${workbookId}`},data:{event:'revoked'}})
    expect(revoked.status()).toBe(401)
    webhookKeyId=''
  }finally{
    if(webhookKeyId)await page.request.delete(`/api/v1/me/api-keys/${webhookKeyId}`)
    if(workbookId)await page.request.delete(`/api/v1/workbooks/${workbookId}`)
    await page.request.post(`/api/v1/admin/settings/versions/${restoreRevision}:restore`,{data:{}})
  }
})

test('compares and resolves a persisted same-cell conflict', async ({ page }) => {
  const created=await page.request.post('/api/v1/workbooks',{data:{title:`충돌 검증 ${Date.now()}`,workspace_id:'default'}}).then(response=>response.json())
  const sheetId=created.sheets[0].id as string
  for(const actor of ['alice','bob'])await page.request.put(`/api/v1/workbooks/${created.id}/shares`,{data:{principal_type:'user',principal_id:actor,role:'editor'}})
  const first=await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{headers:{'X-Kanpic-Actor':'alice'},data:{base_version:1,idempotency_key:`conflict-first-${created.id}`,client_id:'alice-browser',cells:[{row:2,column:3,value:'first',style:{bold:true}}]}})
  expect(first.ok()).toBe(true)
  const stale=await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{headers:{'X-Kanpic-Actor':'bob'},data:{base_version:1,idempotency_key:`conflict-stale-${created.id}`,client_id:'bob-browser',cells:[{row:2,column:3,value:'second',style:{italic:true}}]}})
  expect(stale.ok()).toBe(true)
  expect((await stale.json()).conflicts).toHaveLength(1)

  await page.goto(`/workbooks/${created.id}`)
  const conflictButton=page.getByRole('button',{name:'편집 충돌',exact:true})
  await expect(conflictButton).toHaveAttribute('title','편집 충돌 1건')
  await conflictButton.click()
  const panel=page.getByRole('complementary',{name:'편집 충돌 패널'})
  await expect(panel).toBeVisible()
  await expect(panel.getByText('충돌 전 기준')).toBeVisible()
  await expect(panel.getByText('first',{exact:true})).toBeVisible()
  await expect(panel.getByText('second',{exact:true})).toBeVisible()
  await panel.getByRole('button',{name:/먼저 반영된 값 복원/}).click()
  await expect(panel.getByText('열린 충돌이 없습니다.')).toBeVisible()
  await expect.poll(async()=>{const range=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/C2`).then(response=>response.json());return range.items[0]?.value}).toBe('first')
  const history=await page.request.get(`/api/v1/workbooks/${created.id}/conflicts?include_resolved=true`).then(response=>response.json())
  expect(history.items).toHaveLength(1)
  expect(history.items[0]).toMatchObject({status:'resolved',resolution:'restore_previous',revision:2})
  expect(history.items[0].resolution_operation_id).toBeTruthy()
  await page.request.delete(`/api/v1/workbooks/${created.id}`)
})

test('searches workbook values and formulas and scrolls to a result on another sheet', async ({ page }) => {
  const title=`통합 검색 ${Date.now()}`
  const created=await page.request.post('/api/v1/workbooks',{data:{title,workspace_id:'default'}}).then(response=>response.json())
  const second=await page.request.post(`/api/v1/workbooks/${created.id}/sheets`,{data:{name:'보관 데이터'}}).then(response=>response.json())
  const current=await page.request.get(`/api/v1/workbooks/${created.id}`).then(response=>response.json())
  const seeded=await page.request.patch(`/api/v1/sheets/${second.id}/cells:batch`,{data:{base_version:current.version,idempotency_key:`search-seed-${created.id}`,cells:[{row:120,column:3,value:'원거리 매출 검색 대상'},{row:121,column:3,formula:'=CONCAT("매출", " 수식")'}]}})
  expect(seeded.ok()).toBe(true)

  await page.goto(`/workbooks/${created.id}`)
  await expect(page.locator('canvas.grid-canvas')).toBeVisible()
  await page.keyboard.press('Control+f')
  const dialog=page.getByRole('dialog',{name:'워크북 찾기 및 바꾸기'})
  await expect(dialog).toBeVisible()
  await page.getByRole('textbox',{name:'검색어'}).fill('원거리 매출')
  await page.getByRole('option',{name:/보관 데이터.*C120/}).click()
  await expect(page.locator('.sheet-tabs button.active')).toContainText('보관 데이터')
  await expect(page.locator('.name-box')).toHaveValue('C120')
  await expect.poll(()=>page.locator('.grid-viewport').evaluate(element=>element.scrollTop)).toBeGreaterThan(2000)
  const apiResult=await page.request.get(`/api/v1/workbooks/${created.id}/search?q=${encodeURIComponent('매출')}`).then(response=>response.json())
  expect(apiResult.items.map((item:{address:string})=>item.address)).toEqual(['C120','C121'])
  await page.request.delete(`/api/v1/workbooks/${created.id}`)
})

test('collaborates with anchored comments, replies, and mention notifications', async ({ page, context }) => {
  const created=await page.request.post('/api/v1/workbooks',{data:{title:`댓글 협업 ${Date.now()}`,workspace_id:'default'}}).then(response=>response.json())
  const sheetId=created.sheets[0].id as string
  const anchorRow=100+(Date.now()%9800),anchorRange=`B${anchorRow}:C${anchorRow+1}`
  await page.goto(`/workbooks/${created.id}`)
  const nameBox=page.getByRole('combobox',{name:'이름 상자'})
  await nameBox.fill(anchorRange)
  await nameBox.press('Enter')
  await page.getByRole('button',{name:'댓글',exact:true}).click()
  const panel=page.getByRole('complementary',{name:'댓글 패널'})
  await expect(panel).toBeVisible()
  await panel.getByRole('textbox',{name:'새 댓글 내용'}).fill('@bob@example.com 이 범위를 검토해 주세요.')
  await panel.getByRole('button',{name:'등록'}).click()
  await expect(panel.getByText('@bob@example.com 이 범위를 검토해 주세요.')).toBeVisible()
  await expect.poll(async()=>{const response=await page.request.get(`/api/v1/workbooks/${created.id}/comments?sheet_id=${sheetId}`).then(result=>result.json());return response.items.length}).toBe(1)
  const comments=await page.request.get(`/api/v1/workbooks/${created.id}/comments?sheet_id=${sheetId}`).then(response=>response.json())
  expect(comments.items).toHaveLength(1)
  expect(comments.items[0].range).toBe(anchorRange)

  await page.request.put(`/api/v1/workbooks/${created.id}/shares`,{data:{principal_type:'user',principal_id:'bob@example.com',role:'commenter'}})
  const reviewer=await context.newPage()
  await reviewer.setExtraHTTPHeaders({'X-Kanpic-Actor':'bob@example.com'})
  await reviewer.goto('/')
  const bell=reviewer.getByRole('button',{name:/알림 \d+개/})
  await expect(bell).toBeVisible()
  await bell.click()
  const notification=reviewer.locator('.notification-list>button').filter({hasText:anchorRange})
  await expect(notification).toContainText('local-user')
  await notification.click()
  await reviewer.waitForURL(new RegExp(`/workbooks/${created.id}.*comment_id=`))
  const reviewerPanel=reviewer.getByRole('complementary',{name:'댓글 패널'})
  await expect(reviewerPanel).toBeVisible()
  await expect(reviewer.locator('.name-box')).toHaveValue(anchorRange)
  await reviewerPanel.getByRole('button',{name:'답글'}).click()
  await reviewerPanel.getByRole('textbox',{name:'답글 내용'}).fill('확인했습니다 @local-user')
  await reviewerPanel.getByRole('button',{name:'답글'}).last().click()
  await expect(reviewerPanel.getByText('확인했습니다 @local-user')).toBeVisible()
  await expect(panel.getByText('확인했습니다 @local-user')).toBeVisible()
  await expect(page.getByRole('button',{name:/알림 \d+개/})).toBeVisible()

  const notifications=await reviewer.request.get('/api/v1/me/notifications?unread_only=true',{headers:{'X-Kanpic-Actor':'bob@example.com'}}).then(response=>response.json())
  expect(notifications.items.filter((item:{workbook_id:string})=>item.workbook_id===created.id)).toHaveLength(0)
  await reviewer.close()
  await page.request.delete(`/api/v1/workbooks/${created.id}`)
})

test('manages workbook favorite, rename, duplicate, and delete from home', async ({ page }) => {
  await page.goto('/')
  const title=`홈 수명주기 ${Date.now()}`,renamed=`${title} 변경`,copyTitle=`${renamed} 복사본`
  const source = await page.request.post('/api/v1/workbooks', { data:{ title, workspace_id:'default' } }).then(response=>response.json())
  await page.request.patch(`/api/v1/sheets/${source.sheets[0].id}/cells:batch`, { data:{ base_version:1, idempotency_key:`home-seed-${source.id}`, cells:[{ row:1, column:1, value:42, style:{ bold:true } }] } })
  await page.reload()
  await expect(page.getByText(title, { exact:true })).toBeVisible()

  await page.getByRole('button', { name:`${title} 더보기` }).click()
  await page.getByRole('menuitem', { name:'즐겨찾기에 추가' }).click()
  await page.locator('.segmented').getByRole('button', { name:'즐겨찾기' }).click()
  await expect(page.getByText(title, { exact:true })).toBeVisible()
  await page.locator('.segmented').getByRole('button', { name:'최근' }).click()

  await page.getByRole('button', { name:`${title} 더보기` }).click()
  await page.getByRole('menuitem', { name:'이름 변경' }).click()
  await page.getByRole('textbox', { name:'워크북 이름' }).fill(renamed)
  await page.getByRole('button', { name:'워크북 이름 저장' }).click()
  await expect(page.getByText(renamed, { exact:true })).toBeVisible()

  await page.getByRole('button', { name:`${renamed} 더보기` }).click()
  await page.getByRole('menuitem', { name:'복제' }).click()
  await expect(page.getByText(copyTitle, { exact:true })).toBeVisible()
  const books=await page.request.get('/api/v1/workbooks').then(response=>response.json())
  const copied=books.items.find((workbook:{title:string})=>workbook.title===copyTitle)
  const copiedRange=await page.request.get(`/api/v1/sheets/${copied.sheets[0].id}/ranges/A1`).then(response=>response.json())
  expect(copied.version).toBe(1)
  expect(copiedRange.items[0]?.value).toBe(42)
  expect(copiedRange.items[0]?.style?.bold).toBe(true)

  await page.getByRole('button', { name:`${renamed} 더보기` }).click()
  page.once('dialog',dialog=>dialog.accept())
  await page.getByRole('menuitem', { name:'휴지통으로 이동' }).click()
  await expect(page.getByText(renamed, { exact:true })).toHaveCount(0)
  await expect(page.getByText(copyTitle, { exact:true })).toBeVisible()
  await page.request.delete(`/api/v1/workbooks/${copied.id}`)
})

test('undoes and redoes an acknowledged cell operation', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response => response.json())
  const sheetId = workbook.sheets[0].id as string
  const valueAtA1 = async () => {
    const response = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1`)
    const body = await response.json()
    return body.items[0]?.value
  }

  const canvas = page.locator('canvas.grid-canvas')
  await canvas.dblclick({ position: { x: 70, y: 42 } })
  await page.locator('.cell-editor').fill('2')
  await page.locator('.cell-editor').press('Enter')
  await expect.poll(valueAtA1).toBe(2)

  await canvas.dblclick({ position: { x: 70, y: 42 } })
  await page.locator('.cell-editor').fill('3')
  await page.locator('.cell-editor').press('Enter')
  await expect.poll(valueAtA1).toBe(3)

  const undo = page.getByRole('button', { name: '실행 취소' })
  await expect(undo).toBeEnabled()
  await undo.click()
  await expect.poll(valueAtA1).toBe(2)

  const redo = page.getByRole('button', { name: '다시 실행' })
  await expect(redo).toBeEnabled()
  await redo.click()
  await expect.poll(valueAtA1).toBe(3)
})

test('inserts and deletes rows with formula references and automatic backup versions', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name:'새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  const workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  const seed=await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:1,idempotency_key:`structure-seed-${workbookId}`,cells:[{row:1,column:1,value:10},{row:2,column:1,value:20},{row:2,column:2,formula:'=A2*2'}]}})
  expect(seed.ok()).toBe(true)
  await page.reload()
  const canvas=page.locator('canvas.grid-canvas')
  await expect(canvas).toBeVisible()
  const cells=async()=>page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1:B3`).then(response=>response.json())

  await canvas.click({position:{x:70,y:69}})
  await page.getByRole('menuitem',{name:'삽입'}).click()
  await page.getByRole('menuitem',{name:'행과 열 관리…'}).click()
  await expect(page.getByRole('dialog',{name:'행과 열 관리'})).toBeVisible()
  await page.getByRole('button',{name:'위에 1개 삽입'}).click()
  await expect(page.getByRole('dialog',{name:'행과 열 관리'})).toHaveCount(0)
  await expect.poll(async()=>{const result=await cells();return result.items.find((cell:{row:number;column:number})=>cell.row===3&&cell.column===2)?.formula}).toBe('=A3*2')
  let result=await cells()
  expect(result.items.find((cell:{row:number;column:number})=>cell.row===3&&cell.column===1)?.value).toBe(20)
  expect(result.items.find((cell:{row:number;column:number})=>cell.row===3&&cell.column===2)?.value).toBe(40)

  await canvas.click({position:{x:70,y:69}})
  await page.getByRole('menuitem',{name:'삽입'}).click()
  await page.getByRole('menuitem',{name:'행과 열 관리…'}).click()
  page.once('dialog',dialog=>dialog.accept())
  await page.getByRole('button',{name:'선택 행 삭제'}).click()
  await expect.poll(async()=>{const current=await cells();return current.items.find((cell:{row:number;column:number})=>cell.row===2&&cell.column===2)?.formula}).toBe('=A2*2')
  result=await cells()
  expect(result.items.find((cell:{row:number;column:number})=>cell.row===2&&cell.column===1)?.value).toBe(20)
  const versions=await page.request.get(`/api/v1/workbooks/${workbookId}/versions`).then(response=>response.json())
  expect(versions.items).toHaveLength(2)
  expect(versions.items.every((version:{name:string})=>version.name.includes('자동 백업'))).toBe(true)
})

test('persists variable dimensions, hidden rows and columns, and frozen panes', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button',{name:'새 워크북'}).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  let workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  const canvas=page.locator('canvas.grid-canvas')
  await expect(canvas).toBeVisible()

  await page.getByRole('menuitem',{name:'보기'}).click()
  await page.getByRole('menuitem',{name:'시트 레이아웃…'}).click()
  const dialog=page.getByRole('dialog',{name:'시트 레이아웃'})
  await expect(dialog).toBeVisible()
  await page.getByLabel('행 높이').fill('60')
  await page.getByRole('button',{name:'높이 적용'}).click()
  await page.getByLabel('열 너비').fill('180')
  await page.getByRole('button',{name:'너비 적용'}).click()
  await page.getByLabel('고정 행 수').fill('1')
  await page.getByLabel('고정 열 수').fill('1')
  await page.getByRole('button',{name:'고정 적용'}).click()
  await page.getByRole('button',{name:'닫기',exact:true}).click()

  workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  let layout=workbook.sheets[0].layout
  expect(layout.row_heights).toEqual([{index:1,size:60}])
  expect(layout.column_widths).toEqual([{index:1,size:180}])
  expect(layout.frozen_rows).toBe(1)
  expect(layout.frozen_columns).toBe(1)

  const viewport=page.locator('.grid-viewport')
  await viewport.evaluate(element=>element.scrollTo({left:700,top:500}))
  await expect.poll(()=>viewport.evaluate(element=>({left:element.scrollLeft,top:element.scrollTop}))).toMatchObject({left:700,top:500})
  await canvas.click({position:{x:70,y:42}})
  await expect(page.locator('.name-box')).toHaveValue('A1')
  await viewport.evaluate(element=>element.scrollTo({left:0,top:0}))
  await canvas.click({position:{x:236,y:97}})
  await expect(page.locator('.name-box')).toHaveValue('B2')
  await page.getByRole('menuitem',{name:'보기'}).click()
  await page.getByRole('menuitem',{name:'시트 레이아웃…'}).click()
  await page.getByRole('button',{name:'선택 행 숨기기'}).click()
  await page.getByRole('button',{name:'선택 열 숨기기'}).click()
  await page.getByRole('button',{name:'닫기',exact:true}).click()
  await expect(page.locator('.name-box')).toHaveValue('C3')
  workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  layout=workbook.sheets[0].layout
  expect(layout.hidden_rows).toEqual([{start:2,end:2}])
  expect(layout.hidden_columns).toEqual([{start:2,end:2}])

  await page.getByRole('menuitem',{name:'보기'}).click()
  await page.getByRole('menuitem',{name:'시트 레이아웃…'}).click()
  await page.getByRole('button',{name:'모든 행 표시'}).click()
  await page.getByRole('button',{name:'모든 열 표시'}).click()
  await page.getByRole('button',{name:'고정 해제'}).click()
  await page.getByRole('button',{name:'닫기',exact:true}).click()
  workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  layout=workbook.sheets[0].layout
  expect(layout.hidden_rows??[]).toEqual([])
  expect(layout.hidden_columns??[]).toEqual([])
  expect(layout.frozen_rows).toBe(0)
  expect(layout.frozen_columns).toBe(0)
})

test('formats a selected range without changing values or formulas and resends offline changes', async ({ page, context }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response => response.json())
  const sheetId = workbook.sheets[0].id as string
  const canvas = page.locator('canvas.grid-canvas')
  const edit = async (position:{x:number;y:number}, value:string) => {
    await canvas.dblclick({ position })
    await page.locator('.cell-editor').fill(value)
    await page.locator('.cell-editor').press('Enter')
  }
  const range = async () => page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1:A2`).then(response => response.json())

  await edit({ x:70, y:42 }, '5')
  await edit({ x:70, y:69 }, '=A1*2')
  await expect.poll(async () => (await range()).items.map((cell:{value:unknown}) => cell.value)).toEqual([5, 10])
  await canvas.click({ position:{ x:70, y:42 } })
  await page.keyboard.press('Shift+ArrowDown')
  await expect(page.locator('.name-box')).toHaveValue('A1:A2')

  await page.getByRole('button', { name:'굵게' }).click()
  await expect.poll(async () => (await range()).items.map((cell:{style?:Record<string,unknown>}) => cell.style?.bold)).toEqual([true, true])
  await page.getByRole('button', { name:'가운데 정렬' }).click()
  await expect.poll(async () => (await range()).items.map((cell:{style?:Record<string,unknown>}) => cell.style?.horizontal_align)).toEqual(['center', 'center'])
  await page.getByLabel('셀 배경색').fill('#fef3c7')
  await expect.poll(async () => (await range()).items.map((cell:{style?:Record<string,unknown>}) => cell.style?.background)).toEqual(['#fef3c7', '#fef3c7'])
  await page.getByLabel('글꼴 크기').selectOption('14')
  await expect.poll(async () => (await range()).items.map((cell:{style?:Record<string,unknown>}) => cell.style?.font_size)).toEqual([14, 14])

  let result = await range()
  expect(result.items.map((cell:{value:unknown}) => cell.value)).toEqual([5, 10])
  expect(result.items[1].formula).toBe('=A1*2')
  await page.getByRole('button', { name:'실행 취소' }).click()
  await expect.poll(async () => (await range()).items.map((cell:{style?:Record<string,unknown>}) => cell.style?.font_size)).toEqual([undefined, undefined])
  result = await range()
  expect(result.items.map((cell:{value:unknown}) => cell.value)).toEqual([5, 10])
  expect(result.items[1].formula).toBe('=A1*2')

  await context.setOffline(true)
  await page.getByRole('button', { name:'기울임' }).click()
  await expect(page.getByText('오프라인 · 로컬 저장', { exact:true })).toBeVisible()
  await context.setOffline(false)
  await expect.poll(async () => (await range()).items.map((cell:{style?:Record<string,unknown>}) => cell.style?.italic), { timeout:15_000 }).toEqual([true, true])
})

test('creates a live native chart and exposes the same definition through REST', async ({ page }) => {
  const created = await page.request.post('/api/v1/workbooks', { data: { title: `차트 ${Date.now()}`, workspace_id: 'default' } }).then(response => response.json())
  const sheetId = created.sheets[0].id as string
  const seeded = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: created.version,
    idempotency_key: `chart-e2e-seed-${created.id}`,
    cells: [
      { row: 1, column: 1, value: '월' }, { row: 1, column: 2, value: '매출' },
      { row: 2, column: 1, value: '1월' }, { row: 2, column: 2, value: 100 },
      { row: 3, column: 1, value: '2월' }, { row: 3, column: 2, formula: '=B2*2' },
    ],
  } })
  expect(seeded.ok()).toBe(true)

  await page.goto(`/workbooks/${created.id}`)
  await page.getByRole('combobox', { name: '이름 상자' }).fill('A1:B3')
  await page.getByRole('combobox', { name: '이름 상자' }).press('Enter')
  await page.getByRole('button', { name: '차트 패널' }).click()
  const panelFrame=page.locator('.right-panel-frame')
  const initialPanelWidth=await panelFrame.evaluate(element=>element.getBoundingClientRect().width)
  const panelResizer=page.getByRole('separator',{name:'차트 패널 너비 조절'})
  await panelResizer.focus()
  await page.keyboard.press('ArrowLeft')
  await expect.poll(async()=>panelFrame.evaluate(element=>element.getBoundingClientRect().width)).toBeGreaterThan(initialPanelWidth)
  await page.getByRole('button', { name: '새 차트' }).click()
  const dialog = page.getByRole('dialog', { name: '차트 만들기' })
  await dialog.getByLabel('차트 제목').fill('월별 매출')
  await dialog.getByLabel('차트 유형').selectOption('line')
  await dialog.getByRole('button', { name: '차트 저장' }).click()

  const card = page.locator('[data-chart-id]')
  await expect(card).toBeVisible()
  await expect(card.getByRole('img', { name: '월별 매출' })).toBeVisible()
  const chartId = await card.getAttribute('data-chart-id')
  const list = await page.request.get(`/api/v1/workbooks/${created.id}/charts?sheet_id=${sheetId}`).then(response => response.json())
  expect(list.items).toHaveLength(1)
  expect(list.items[0]).toMatchObject({ id: chartId, type: 'line', source_range: 'A1:B3', title: '월별 매출' })
  let data = await page.request.get(`/api/v1/charts/${chartId}/data`).then(response => response.json())
  expect(data.series[0].points.map((point: { value: number }) => point.value)).toEqual([100, 200])

  const current = await page.request.get(`/api/v1/workbooks/${created.id}`).then(response => response.json())
  await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: { base_version: current.version, idempotency_key: `chart-e2e-refresh-${created.id}`, cells: [{ row: 2, column: 2, value: 150 }] } })
  await expect.poll(async () => {
    data = await page.request.get(`/api/v1/charts/${chartId}/data`).then(response => response.json())
    return data.series[0].points.map((point: { value: number }) => point.value)
  }).toEqual([150, 300])
  await page.request.delete(`/api/v1/workbooks/${created.id}`)
})

test('creates a managed pivot, opens results, and drills into source rows', async ({ page }) => {
  const created = await page.request.post('/api/v1/workbooks', { data: { title: `피벗 ${Date.now()}`, workspace_id: 'default' } }).then(response => response.json())
  const sheetId = created.sheets[0].id as string
  const seeded = await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`, { data: {
    base_version: created.version,
    idempotency_key: `pivot-e2e-seed-${created.id}`,
    cells: [
      { row: 1, column: 1, value: '지역' }, { row: 1, column: 2, value: '매출' },
      { row: 2, column: 1, value: '동부' }, { row: 2, column: 2, value: 100 },
      { row: 3, column: 1, value: '동부' }, { row: 3, column: 2, value: 50 },
      { row: 4, column: 1, value: '서부' }, { row: 4, column: 2, value: 200 },
    ],
  } })
  expect(seeded.ok()).toBe(true)

  await page.goto(`/workbooks/${created.id}`)
  await page.getByRole('combobox', { name: '이름 상자' }).fill('A1:B4')
  await page.getByRole('combobox', { name: '이름 상자' }).press('Enter')
  await page.getByRole('button', { name: '피벗 패널' }).click()
  await page.getByRole('button', { name: '새 피벗' }).click()
  const dialog = page.getByRole('dialog', { name: '피벗 만들기' })
  await dialog.getByLabel('피벗 이름').fill('지역별 매출')
  await dialog.getByRole('button', { name: '행 그룹 추가' }).click()
  await dialog.getByLabel('값 필드 1').selectOption('2')
  await dialog.getByRole('button', { name: '피벗 저장' }).click()

  const panelItem = page.locator('.pivot-panel-list article').filter({ hasText: '지역별 매출' })
  await expect(panelItem).toBeVisible()
  await panelItem.getByTitle('결과 열기').click()
  const result = page.getByRole('dialog', { name: '피벗 결과' })
  await expect(result.getByText('150', { exact: true })).toBeVisible()
  await expect(result.getByText('200', { exact: true })).toBeVisible()
  await result.getByRole('button', { name: '150' }).click()
  const drilldown = result.getByRole('dialog', { name: '피벗 원본 행' })
  await expect(drilldown.getByText('2개 행')).toBeVisible()

  const list = await page.request.get(`/api/v1/workbooks/${created.id}/pivots?sheet_id=${sheetId}`).then(response => response.json())
  expect(list.items).toHaveLength(1)
  expect(list.items[0]).toMatchObject({ name: '지역별 매출', source_range: 'A1:B4', refresh_mode: 'auto' })
  const data = await page.request.get(`/api/v1/pivots/${list.items[0].id}/data`).then(response => response.json())
  expect(data.rows.map((row: { values: number[] }) => row.values[0])).toEqual([150, 200])
  await page.request.delete(`/api/v1/workbooks/${created.id}`)
})

test('creates, edits, rotates, and revokes a scoped personal API key', async ({ page }) => {
  const initialName = `E2E 에이전트 ${Date.now()}`
  const renamed = `${initialName} 수정`
  await page.goto('/preferences')
  await page.getByRole('button', { name: 'API 키' }).click()
  await page.getByRole('button', { name: '새 키' }).click()
  let dialog = page.getByRole('dialog', { name: '새 API 키' })
  await dialog.getByLabel('키 이름').fill(initialName)
  await expect(dialog.getByText('chart.*')).toBeVisible()
  await expect(dialog.getByText('pivot.*')).toBeVisible()
  await dialog.getByRole('button', { name: '키 생성' }).click()
  await expect(page.getByText('새 키를 지금 복사하세요')).toBeVisible()

  let activeCard = page.locator('.key-card:not(.revoked)').filter({ hasText: initialName })
  await expect(activeCard).toBeVisible()
  await activeCard.getByRole('button', { name: '수정' }).click()
  dialog = page.getByRole('dialog', { name: 'API 키 수정' })
  await dialog.getByLabel('키 이름').fill(renamed)
  await dialog.getByLabel('만료 시점 (선택)').fill('2030-01-02T03:04')
  await dialog.getByRole('button', { name: '변경 저장' }).click()
  activeCard = page.locator('.key-card:not(.revoked)').filter({ hasText: renamed })
  await expect(activeCard).toContainText('2030')

  page.once('dialog', prompt => prompt.accept())
  await activeCard.getByRole('button', { name: '회전' }).click()
  await expect(page.locator('.key-card.revoked').filter({ hasText: renamed })).toBeVisible()
  activeCard = page.locator('.key-card:not(.revoked)').filter({ hasText: renamed })
  await expect(activeCard).toBeVisible()
  page.once('dialog', prompt => prompt.accept())
  await activeCard.getByRole('button', { name: '폐기' }).click()
  await expect(page.locator('.key-card:not(.revoked)').filter({ hasText: renamed })).toHaveCount(0)
})

test('merges cells without data loss and supports undo redo and offline unmerge', async ({ page, context }) => {
  await page.goto('/')
  await page.getByRole('button', { name:'새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  const workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  const canvas=page.locator('canvas.grid-canvas')
  const edit=async(position:{x:number;y:number},value:string)=>{await canvas.dblclick({position});await page.locator('.cell-editor').fill(value);await page.locator('.cell-editor').press('Enter')}
  const cells=async()=>page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1:B2`).then(response=>response.json())

  await edit({x:70,y:42},'merged title')
  await edit({x:170,y:69},'kept')
  await canvas.click({position:{x:70,y:42}})
  await page.keyboard.press('Shift+ArrowRight')
  await page.keyboard.press('Shift+ArrowDown')
  await page.getByRole('button',{name:'셀 병합'}).click()
  await expect.poll(async()=>{const result=await cells();return result.items.filter((cell:{style?:Record<string,unknown>})=>cell.style?.merge).length}).toBe(4)
  let merged=await cells()
  expect(merged.items.find((cell:{row:number;column:number})=>cell.row===1&&cell.column===1).value).toBe('merged title')
  expect(merged.items.find((cell:{row:number;column:number})=>cell.row===2&&cell.column===2).value).toBe('kept')

  await canvas.click({position:{x:170,y:69}})
  await expect(page.locator('.name-box')).toHaveValue('A1:B2')
  await expect(page.getByLabel('수식 입력창')).toHaveValue('merged title')
  await page.getByRole('button',{name:'실행 취소'}).click()
  await expect.poll(async()=>{const result=await cells();return result.items.some((cell:{style?:Record<string,unknown>})=>cell.style?.merge)}).toBe(false)
  await page.getByRole('button',{name:'다시 실행'}).click()
  await expect.poll(async()=>{const result=await cells();return result.items.filter((cell:{style?:Record<string,unknown>})=>cell.style?.merge).length}).toBe(4)

  await canvas.click({position:{x:170,y:69}})
  await context.setOffline(true)
  await page.getByRole('button',{name:'병합 해제'}).click()
  await expect(page.getByText('오프라인 · 로컬 저장',{exact:true})).toBeVisible()
  await context.setOffline(false)
  await expect.poll(async()=>{const result=await cells();return result.items.some((cell:{style?:Record<string,unknown>})=>cell.style?.merge)},{timeout:15_000}).toBe(false)
  merged=await cells()
  expect(merged.items.find((cell:{row:number;column:number})=>cell.row===2&&cell.column===2).value).toBe('kept')
})

test('sorts a range by multiple keys with formulas, undo, and offline resend', async ({ page, context }) => {
  await page.goto('/')
  await page.getByRole('button', { name:'새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  const workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  const seed=await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{
    base_version:1,
    idempotency_key:`sort-seed-${workbookId}`,
    cells:[
      {row:1,column:1,value:'Name'},{row:1,column:2,value:'Quantity'},{row:1,column:3,value:'Total'},
      {row:2,column:1,value:'beta',style:{bold:true}},{row:2,column:2,value:2},{row:2,column:3,formula:'=B2*2'},
      {row:3,column:1,value:'Alpha'},{row:3,column:2,value:10},{row:3,column:3,formula:'=B3*2'},
      {row:4,column:1,value:'alpha'},{row:4,column:2,value:5},{row:4,column:3,formula:'=B4*2'},
    ],
  }})
  expect(seed.ok()).toBe(true)
  await page.reload()
  const canvas=page.locator('canvas.grid-canvas')
  await expect(canvas).toBeVisible()
  const range=async()=>page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1:C4`).then(response=>response.json())
  const rows=async()=>{
    const result=await range()
    return Array.from({length:4},(_,offset)=>{
      const row=offset+1
      const at=(column:number)=>result.items.find((cell:{row:number;column:number})=>cell.row===row&&cell.column===column)
      return {name:at(1)?.value,quantity:at(2)?.value,total:at(3)?.value,formula:at(3)?.formula,bold:at(1)?.style?.bold}
    })
  }
  const selectRange=async()=>{
    await canvas.click({position:{x:70,y:42}})
    await page.keyboard.press('Shift+ArrowRight')
    await page.keyboard.press('Shift+ArrowRight')
    await page.keyboard.press('Shift+ArrowDown')
    await page.keyboard.press('Shift+ArrowDown')
    await page.keyboard.press('Shift+ArrowDown')
    await expect(page.locator('.name-box')).toHaveValue('A1:C4')
  }

  await selectRange()
  await page.getByRole('button',{name:'범위 정렬'}).click()
  await expect(page.getByRole('dialog',{name:'범위 정렬'})).toBeVisible()
  await page.getByRole('button',{name:'+ 기준 추가'}).click()
  await page.getByLabel('2차 정렬 방향').selectOption('desc')
  await page.getByRole('button',{name:'정렬 적용'}).click()
  await expect.poll(async()=>(await rows()).map(row=>row.name)).toEqual(['Name','Alpha','alpha','beta'])
  let sorted=await rows()
  expect(sorted.map(row=>row.quantity)).toEqual(['Quantity',10,5,2])
  expect(sorted.slice(1).map(row=>row.formula)).toEqual(['=B2*2','=B3*2','=B4*2'])
  expect(sorted.slice(1).map(row=>row.total)).toEqual([20,10,4])
  expect(sorted[3].bold).toBe(true)

  await page.getByRole('button',{name:'실행 취소'}).click()
  await expect.poll(async()=>(await rows()).map(row=>row.name)).toEqual(['Name','beta','Alpha','alpha'])

  await selectRange()
  await context.setOffline(true)
  await page.getByRole('button',{name:'범위 정렬'}).click()
  await page.getByLabel('1차 정렬 열').selectOption('2')
  await page.getByLabel('1차 정렬 방향').selectOption('desc')
  await page.getByRole('button',{name:'정렬 적용'}).click()
  await expect(page.getByText('오프라인 · 로컬 저장',{exact:true})).toBeVisible()
  await context.setOffline(false)
  await expect.poll(async()=>(await rows()).map(row=>row.quantity),{timeout:15_000}).toEqual(['Quantity',10,5,2])
  sorted=await rows()
  expect(sorted.slice(1).map(row=>row.formula)).toEqual(['=B2*2','=B3*2','=B4*2'])
  expect(sorted.slice(1).map(row=>row.total)).toEqual([20,10,4])
})

test('persists personal filter views and compresses filtered canvas rows', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button',{name:'새 워크북'}).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  const workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  const seed=await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:1,idempotency_key:`filter-seed-${workbookId}`,cells:[
    {row:1,column:1,value:'Region'},{row:1,column:2,value:'Amount'},{row:1,column:3,value:'Status'},
    {row:2,column:1,value:'Seoul'},{row:2,column:2,value:12},{row:2,column:3,value:'open',style:{background:'#fef3c7'}},
    {row:3,column:1,value:'Busan'},{row:3,column:2,value:7},{row:3,column:3,value:'open',style:{background:'#fef3c7'}},
    {row:4,column:1,value:'Daejeon'},{row:4,column:2,value:20},{row:4,column:3,value:'open',style:{background:'#fef3c7'}},
    {row:5,column:1,value:'Seoul'},{row:5,column:2,value:15},{row:5,column:3,value:'closed',style:{background:'#ffffff'}},
  ]}})
  expect(seed.ok()).toBe(true)
  await page.reload()
  const canvas=page.locator('canvas.grid-canvas')
  await expect(canvas).toBeVisible()
  await canvas.click({position:{x:70,y:42}})
  await page.keyboard.press('Shift+ArrowRight');await page.keyboard.press('Shift+ArrowRight')
  for(let index=0;index<4;index++)await page.keyboard.press('Shift+ArrowDown')
  await expect(page.locator('.name-box')).toHaveValue('A1:C5')

  await page.getByRole('button',{name:'필터 보기'}).click()
  await expect(page.getByRole('dialog',{name:'필터 보기'})).toBeVisible()
  await page.getByLabel('필터 보기 이름').fill('qualified')
  await page.getByLabel('1차 필터 값').fill('Seoul, Busan')
  await page.getByRole('button',{name:/기준 추가/}).click()
  await page.getByLabel('2차 필터 조건').selectOption('greater_or_equal')
  await page.getByLabel('2차 필터 값').fill('10')
  await page.getByRole('button',{name:/기준 추가/}).click()
  await page.getByLabel('3차 필터 조건').selectOption('background_color')
  await page.getByLabel('3차 필터 색상').fill('#fef3c7')
  await page.getByRole('button',{name:'저장 및 적용'}).click()
  await expect(page.getByText('전체 4행 중 1행 표시 · 3행 숨김')).toBeVisible()
  const filterViews=async(headers?:Record<string,string>)=>page.request.get(`/api/v1/sheets/${sheetId}/filter-views`,{headers}).then(response=>response.json())
  const filterResult=async(id:string)=>page.request.post(`/api/v1/filter-views/${id}:evaluate`).then(response=>response.json())
  await expect.poll(async()=>{const result=await filterViews();return result.items[0]?.id}).not.toBeUndefined()
  const viewId=(await filterViews()).items[0].id as string
  await expect.poll(async()=>(await filterResult(viewId)).hidden_rows).toEqual([3,4,5])
  await page.request.put(`/api/v1/workbooks/${workbookId}/shares`,{data:{principal_type:'user',principal_id:'other-filter-user',role:'viewer'}})
  const otherUser=await filterViews({'X-Kanpic-Actor':'other-filter-user'})
  expect(otherUser.items).toEqual([])
  await page.getByRole('button',{name:'필터 닫기'}).click()

  await page.reload()
  await expect(canvas).toBeVisible()
  await expect(page.getByRole('button',{name:'필터 보기'})).toHaveClass(/active/)
  await canvas.click({position:{x:70,y:96}})
  await expect(page.locator('.name-box')).toHaveValue('A6')
  await page.getByRole('button',{name:'필터 보기'}).click()
  await page.getByRole('button',{name:/qualified.*적용 중/}).click()
  await page.getByRole('button',{name:'필터 해제'}).click()
  await expect.poll(async()=>Boolean((await filterViews()).items[0]?.active)).toBe(false)
  await page.getByRole('button',{name:'필터 닫기'}).click()
  await canvas.click({position:{x:70,y:96}})
  await expect(page.locator('.name-box')).toHaveValue('A3')

  await page.getByRole('button',{name:'필터 보기'}).click()
  await page.getByRole('button',{name:/qualified/}).click()
  await page.getByRole('button',{name:'필터 적용'}).click()
  await expect.poll(async()=>Boolean((await filterViews()).items[0]?.active)).toBe(true)
  await page.getByRole('button',{name:'필터 닫기'}).click()
  const latest=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:latest.version,idempotency_key:`filter-latest-${workbookId}`,cells:[{row:3,column:2,value:11}]}})
  await expect.poll(async()=>(await filterResult(viewId)).hidden_rows).toEqual([4,5])
  await canvas.click({position:{x:70,y:96}})
  await expect(page.locator('.name-box')).toHaveValue('A3')
})

test('creates colored dropdown validation and rejects invalid writes', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button',{name:'새 워크북'}).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  const workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  const canvas=page.locator('canvas.grid-canvas')
  await canvas.click({position:{x:70,y:42}})
  await page.keyboard.press('Shift+ArrowDown');await page.keyboard.press('Shift+ArrowDown')
  await expect(page.locator('.name-box')).toHaveValue('A1:A3')

  await page.getByRole('button',{name:'데이터 검증'}).click()
  await expect(page.getByRole('dialog',{name:'데이터 검증'})).toBeVisible()
  await page.getByLabel('목록 항목 1 값').fill('open')
  await page.getByLabel('목록 항목 1 라벨').fill('Open')
  await page.getByLabel('목록 항목 1 색상').fill('#dcfce7')
  await page.getByRole('button',{name:/항목 추가/}).click()
  await page.getByLabel('목록 항목 2 값').fill('closed')
  await page.getByLabel('목록 항목 2 라벨').fill('Closed')
  await page.getByLabel('목록 항목 2 색상').fill('#fee2e2')
  await page.getByLabel('검증 도움말').fill('상태 목록에서 선택하세요.')
  await page.getByRole('button',{name:'규칙 저장'}).click()
  const rules=async()=>page.request.get(`/api/v1/sheets/${sheetId}/data-validations`).then(response=>response.json())
  await expect.poll(async()=>{const result=await rules();return result.items[0]?.range}).toBe('A1:A3')
  const rule=(await rules()).items[0]
  expect(rule.options.map((option:{value:string;color:string})=>[option.value,option.color])).toEqual([['open','#dcfce7'],['closed','#fee2e2']])
  await page.getByRole('button',{name:'기존 데이터 검사'}).click()
  await expect(page.getByText('검사 3셀 · 정상 3셀 · 오류 0셀')).toBeVisible()
  await page.getByRole('button',{name:'데이터 검증 닫기'}).click()

  await canvas.click({position:{x:70,y:42}})
  await page.locator('.cell-dropdown-trigger').click()
  await page.getByRole('option',{name:'드롭다운 값 Open'}).click()
  const valueAt=async(row:number)=>page.request.get(`/api/v1/sheets/${sheetId}/ranges/A${row}`).then(response=>response.json()).then(body=>body.items[0]?.value)
  await expect.poll(()=>valueAt(1)).toBe('open')
  await canvas.click({position:{x:70,y:69}})
  await page.locator('.cell-dropdown-trigger').click()
  await page.getByRole('option',{name:'드롭다운 값 Closed'}).click()
  await expect.poll(()=>valueAt(2)).toBe('closed')

  const latest=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const rejected=await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:latest.version,idempotency_key:`validation-invalid-${workbookId}`,cells:[{row:3,column:1,value:'invalid'}]}})
  expect(rejected.status()).toBe(422)
  const rejection=await rejected.json();expect(rejection.error.code).toBe('validation_failed');expect(rejection.error.violations[0].validation_id).toBe(rule.id)
  expect(await valueAt(3)).toBeUndefined()

  await canvas.click({position:{x:70,y:96}})
  const dialogPromise=page.waitForEvent('dialog')
  const pastePromise=page.locator('.grid-viewport').evaluate(element=>{const data=new DataTransfer();data.setData('text/plain','invalid');element.dispatchEvent(new ClipboardEvent('paste',{bubbles:true,cancelable:true,clipboardData:data}))})
  const dialog=await dialogPromise
  expect(dialog.message()).toContain('상태 목록에서 선택하세요.')
  await dialog.accept()
  await pastePromise
  await expect.poll(()=>valueAt(3)).toBeUndefined()

  await page.reload();await expect(canvas).toBeVisible()
  await canvas.click({position:{x:70,y:42}})
  await expect(page.locator('.cell-dropdown-trigger')).toBeVisible()
  await page.getByRole('button',{name:'데이터 검증'}).click()
  await page.locator('.validation-layout>aside button').nth(1).click()
  const deleteDialogPromise=page.waitForEvent('dialog')
  const deletePromise=page.getByRole('button',{name:'삭제',exact:true}).click()
  const deleteDialog=await deleteDialogPromise
  await deleteDialog.accept()
  await deletePromise
  await expect.poll(async()=>(await rules()).items).toEqual([])
})

test('manages conditional formats and renders evaluated cells on canvas', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button',{name:'새 워크북'}).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  let workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`conditional-seed-${workbookId}`,cells:[{row:1,column:1,value:10},{row:2,column:1,value:20},{row:3,column:1,value:20}]}})
  await page.reload()
  const canvas=page.locator('canvas.grid-canvas')
  await expect(canvas).toBeVisible()
  await canvas.click({position:{x:70,y:42}})
  await page.keyboard.press('Shift+ArrowDown');await page.keyboard.press('Shift+ArrowDown')
  await expect(page.locator('.name-box')).toHaveValue('A1:A3')

  await page.getByRole('button',{name:'조건부 서식'}).click()
  await expect(page.getByRole('dialog',{name:'조건부 서식'})).toBeVisible()
  await page.getByLabel('조건부 서식 이름').fill('중복 강조')
  await page.getByLabel('조건부 서식 유형').selectOption('duplicate')
  await page.getByLabel('조건부 배경색').fill('#fef3c7')
  await page.getByLabel('조건부 굵게').check()
  await page.getByRole('button',{name:'규칙 저장'}).click()
  const rules=async()=>page.request.get(`/api/v1/sheets/${sheetId}/conditional-formats`).then(response=>response.json())
  await expect.poll(async()=>(await rules()).items[0]?.name).toBe('중복 강조')
  const created=(await rules()).items[0]
  expect(created.range).toBe('A1:A3');expect(created.rule_type).toBe('duplicate');expect(created.style.bold).toBe(true)
  const evaluated=await page.request.get(`/api/v1/sheets/${sheetId}/conditional-formats:evaluate?range=A1%3AA3`).then(response=>response.json())
  expect(evaluated.items.map((item:{row:number})=>item.row)).toEqual([2,3])
  await page.getByRole('button',{name:'조건부 서식 닫기'}).click()
  await expect.poll(async()=>Number(await canvas.getAttribute('data-conditional-cells'))).toBe(2)
  await expect(page.getByRole('button',{name:'조건부 서식'})).toHaveClass(/active/)

  await page.getByRole('button',{name:'조건부 서식'}).click()
  await page.getByRole('button',{name:/중복 강조/}).click()
  await page.getByLabel('조건부 서식 이름').fill('반복 값')
  await page.getByLabel('조건부 서식 우선순위').fill('4')
  await page.getByRole('button',{name:'규칙 저장'}).click()
  await expect.poll(async()=>{const item=(await rules()).items[0];return `${item.name}:${item.priority}:${item.revision}`}).toBe('반복 값:4:2')
  await page.getByRole('button',{name:'조건부 서식 닫기'}).click()

  await page.reload();await expect(canvas).toBeVisible()
  await expect.poll(async()=>Number(await canvas.getAttribute('data-conditional-cells'))).toBe(2)
  await page.getByRole('button',{name:'조건부 서식'}).click()
  await page.getByRole('button',{name:/반복 값/}).click()
  const deleteDialogPromise=page.waitForEvent('dialog')
  const deletePromise=page.getByRole('button',{name:'삭제',exact:true}).click()
  const deleteDialog=await deleteDialogPromise
  await deleteDialog.accept();await deletePromise
  await expect.poll(async()=>(await rules()).items).toEqual([])
  await page.getByRole('button',{name:'조건부 서식 닫기'}).click()
  await expect.poll(async()=>Number(await canvas.getAttribute('data-conditional-cells'))).toBe(0)
})

test('spills FILTER results and protects generated cells in the editor', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button',{name:'새 워크북'}).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  const workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  const canvas=page.locator('canvas.grid-canvas')
  const edit=async(position:{x:number;y:number},value:string)=>{await canvas.dblclick({position});await page.locator('.cell-editor').fill(value);await page.locator('.cell-editor').press('Enter')}
  const range=async()=>page.request.get(`/api/v1/sheets/${sheetId}/ranges/D1:E2`).then(response=>response.json())

  await edit({x:70,y:42},'a')
  await edit({x:170,y:42},'30')
  await edit({x:70,y:69},'b')
  await edit({x:170,y:69},'10')
  await edit({x:70,y:96},'c')
  await edit({x:170,y:96},'20')
  await edit({x:390,y:42},'=FILTER(A1:B3,B1:B3>=20)')
  await expect.poll(async()=>{
    const result=await range()
    return result.items.map((cell:{value:unknown})=>cell.value)
  }).toEqual(['a',30,'c',20])
  let result=await range()
  expect(result.items.slice(1).every((cell:{spill_source?:string})=>cell.spill_source==='D1')).toBe(true)

  await canvas.click({position:{x:390,y:69}})
  await page.keyboard.press('F2')
  await expect(page.locator('.name-box')).toHaveValue('D1')
  await expect(page.locator('.cell-editor')).toHaveValue('=FILTER(A1:B3,B1:B3>=20)')
  await page.locator('.cell-editor').press('Escape')

  await canvas.click({position:{x:390,y:69}})
  const dialogPromise=page.waitForEvent('dialog')
  const pastePromise=page.locator('.grid-viewport').evaluate(element=>{const data=new DataTransfer();data.setData('text/plain','invalid');element.dispatchEvent(new ClipboardEvent('paste',{bubbles:true,cancelable:true,clipboardData:data}))})
  const dialog=await dialogPromise
  expect(dialog.message()).toContain('D1 배열 수식의 결과')
  await dialog.accept()
  await pastePromise
  expect((await range()).items.map((cell:{value:unknown})=>cell.value)).toEqual(['a',30,'c',20])

  await edit({x:170,y:42},'5')
  await expect.poll(async()=>{
    const shrunk=await range()
    return shrunk.items.map((cell:{value:unknown})=>cell.value)
  }).toEqual(['c',20])
  result=await range()
  expect(result.items[1].spill_source).toBe('D1')
})

test('recalculates cross-sheet formulas entered in the editor and preserves them through rename', async ({ page }) => {
  const workbook=await page.request.post('/api/v1/workbooks',{data:{title:`교차 시트 ${Date.now()}`,workspace_id:'default'}}).then(response=>response.json())
  const inputSheet=workbook.sheets[0]
  const reportSheet=await page.request.post(`/api/v1/workbooks/${workbook.id}/sheets`,{data:{name:'Sales Report'}}).then(response=>response.json())
  await page.request.patch(`/api/v1/sheets/${inputSheet.id}/cells:batch`,{data:{base_version:2,idempotency_key:`cross-seed-${workbook.id}`,cells:[{row:1,column:1,value:10}]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await page.getByRole('button',{name:'Sales Report',exact:true}).click()
  const canvas=page.locator('canvas.grid-canvas')
  await canvas.dblclick({position:{x:208,y:42}})
  await page.locator('.cell-editor').fill(`='Sheet1'!A1*2`)
  await page.locator('.cell-editor').press('Enter')
  const reportValue=async()=>page.request.get(`/api/v1/sheets/${reportSheet.id}/ranges/B1`).then(response=>response.json()).then(body=>body.items[0])
  await expect.poll(async()=>(await reportValue())?.value).toBe(20)

  await page.getByRole('button',{name:'Sheet1',exact:true}).click()
  await canvas.dblclick({position:{x:70,y:42}})
  await page.locator('.cell-editor').fill('25')
  await page.locator('.cell-editor').press('Enter')
  await expect.poll(async()=>(await reportValue())?.value).toBe(50)

  await page.getByRole('button',{name:'Sheet1 시트 메뉴'}).click()
  await page.getByRole('menuitem',{name:'이름 변경'}).click()
  await page.getByRole('textbox',{name:'시트 이름'}).fill('Raw Data')
  await page.getByRole('button',{name:'시트 이름 저장'}).click()
  await expect.poll(async()=>(await reportValue())?.formula).toBe(`='Raw Data'!A1*2`)
})

test('creates named ranges from the name box and keeps formulas valid through rename', async ({ page }) => {
  const workbook=await page.request.post('/api/v1/workbooks',{data:{title:`이름 범위 ${Date.now()}`,workspace_id:'default'}}).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:1,idempotency_key:`named-seed-${workbook.id}`,cells:[{row:1,column:1,value:10},{row:2,column:1,value:20}]}})
  await page.goto(`/workbooks/${workbook.id}`)
  const nameBox=page.getByRole('combobox',{name:'이름 상자'})
  await nameBox.fill('A1:A2')
  await nameBox.press('Enter')
  await expect(nameBox).toHaveValue('A1:A2')
  await page.getByRole('button',{name:'이름 범위 관리'}).click()
  await page.getByRole('textbox',{name:'이름 범위 이름'}).fill('Sales_Data')
  await expect(page.getByRole('textbox',{name:'이름 범위 대상'})).toHaveValue('A1:A2')
  await page.getByRole('button',{name:'저장',exact:true}).click()
  await expect.poll(async()=>page.request.get(`/api/v1/workbooks/${workbook.id}/named-ranges`).then(response=>response.json()).then(body=>body.items[0]?.name)).toBe('Sales_Data')
  await page.getByRole('button',{name:'이름 범위 닫기'}).click()

  await nameBox.fill('B1')
  await nameBox.press('Enter')
  const canvas=page.locator('canvas.grid-canvas')
  await canvas.dblclick({position:{x:208,y:42}})
  await page.locator('.cell-editor').fill('=SUM(Sales_Data)')
  await page.locator('.cell-editor').press('Enter')
  const formulaCell=async()=>page.request.get(`/api/v1/sheets/${sheetId}/ranges/B1`).then(response=>response.json()).then(body=>body.items[0])
  await expect.poll(async()=>(await formulaCell())?.value).toBe(30)

  await page.getByRole('button',{name:'이름 범위 관리'}).click()
  await page.getByRole('button',{name:/Sales_Data/}).click()
  await page.getByRole('textbox',{name:'이름 범위 이름'}).fill('Revenue')
  await page.getByRole('textbox',{name:'이름 범위 대상'}).fill('A1')
  await page.getByRole('button',{name:'저장',exact:true}).click()
  await expect.poll(async()=>({formula:(await formulaCell())?.formula,value:(await formulaCell())?.value})).toEqual({formula:'=SUM(Revenue)',value:10})
  await page.getByRole('button',{name:'이름 범위 닫기'}).click()
  await nameBox.fill('Revenue')
  await nameBox.press('Enter')
  await expect(nameBox).toHaveValue('A1')

  await page.getByRole('button',{name:'이름 범위 관리'}).click()
  await page.getByRole('button',{name:/Revenue/}).click()
  page.once('dialog',dialog=>dialog.accept())
  await page.getByRole('button',{name:'삭제',exact:true}).click()
  await expect.poll(async()=>(await formulaCell())?.value).toBe('#NAME?')
})

test('synchronizes presence and edits between two browser tabs', async ({ page, context }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const editorURL = page.url()
  await expect(page.locator('.collaboration-count')).toContainText('1명 접속')

  const second = await context.newPage()
  await second.goto(editorURL)
  await expect(second.locator('.collaboration-count')).toContainText('2명 접속')
  await expect(page.locator('.collaboration-count')).toContainText('2명 접속')

  const firstCanvas = page.locator('canvas.grid-canvas')
  await firstCanvas.dblclick({ position: { x: 70, y: 42 } })
  await page.locator('.cell-editor').fill('17')
  await page.locator('.cell-editor').press('Enter')
  await expect(second.getByLabel('수식 입력창')).toHaveValue('17')

  await second.close()
  await expect(page.locator('.collaboration-count')).toContainText('1명 접속')
})

test('creates a named version and restores it with an automatic backup', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response => response.json())
  const sheetId = workbook.sheets[0].id as string
  const valueAtA1 = async () => {
    const body = await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1`).then(response => response.json())
    return body.items[0]?.value
  }
  const editA1 = async (value:string) => {
    const canvas = page.locator('canvas.grid-canvas')
    await canvas.dblclick({ position: { x: 70, y: 42 } })
    await page.locator('.cell-editor').fill(value)
    await page.locator('.cell-editor').press('Enter')
  }

  await editA1('10')
  await expect.poll(valueAtA1).toBe(10)
  await page.locator('.toolbar').getByRole('button', { name: '버전 이력', exact: true }).click()
  await expect(page.getByText('버전 이력', { exact: true })).toBeVisible()
  await page.getByPlaceholder('예: 2026년 3분기 확정').fill('기준 버전')
  await page.getByRole('button', { name: '저장', exact: true }).click()
  await expect(page.locator('.workbook-version').filter({ hasText: '기준 버전' })).toBeVisible()

  await page.locator('.toolbar').getByRole('button', { name: '버전 이력', exact: true }).click()
  await editA1('20')
  await expect.poll(valueAtA1).toBe(20)
  await page.locator('.toolbar').getByRole('button', { name: '버전 이력', exact: true }).click()
  page.once('dialog', dialog => dialog.accept())
  await page.locator('.workbook-version').filter({ hasText: '기준 버전' }).getByRole('button', { name: '복원' }).click()
  await expect.poll(valueAtA1).toBe(10)
  await expect(page.getByLabel('수식 입력창')).toHaveValue('10')
  await expect(page.locator('.workbook-version').filter({ hasText: '복원 전 자동 백업' })).toBeVisible()
})

test('selects a range and pastes copied formulas with relative references', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response => response.json())
  const sheetId = workbook.sheets[0].id as string
  const canvas = page.locator('canvas.grid-canvas')
  const edit = async (position:{x:number;y:number},value:string) => {
    await canvas.dblclick({ position })
    await page.locator('.cell-editor').fill(value)
    await page.locator('.cell-editor').press('Enter')
  }
  await edit({x:70,y:42},'2')
  await edit({x:170,y:42},'=A1*2')

  // 복사는 화면 저장소에 있는 것을 담는다. Enter 를 누른 뒤 그 저장소가
  // 채워지기까지는 비동기로 몇 단계를 거치는데, Playwright 의 press 는 키를
  // 보낸 순간 돌아온다. 그 사이에 Ctrl+C 가 끼면 빈 칸을 복사하고,
  // 붙여넣기는 한 칸만 쓴다. 붙여넣은 쪽을 15초 기다려도 복사한 것이
  // 비어 있었으면 영영 채워지지 않는다 — CI 에서 드물게 실패한 까닭이다.
  //
  // 서버에 닿았는지로 기다린다. 저장소는 서버로 보내기 **전에** 채워지므로,
  // 서버가 알고 있다면 저장소는 이미 채워져 있다.
  const source = async () => page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1:B1`).then(response => response.json())
  await expect.poll(async()=>{
    const result=await source()
    return result.items.map((cell:{value:unknown})=>cell.value)
  },{timeout:15_000}).toEqual([2,4])

  await canvas.click({position:{x:70,y:42}})
  await page.keyboard.press('Shift+ArrowRight')
  await expect(page.locator('.name-box')).toHaveValue('A1:B1')
  await page.keyboard.press('Control+C')
  await canvas.click({position:{x:390,y:120}})
  // 붙여넣기는 눌린 순간의 활성 셀에 들어간다. 선택이 옮겨진 것을 확인하지
  // 않고 누르면 이전 셀이나 옆 칸에 붙는다.
  await expect(page.locator('.name-box')).toHaveValue('D4')
  await page.keyboard.press('Control+V')

  const pasted = async () => page.request.get(`/api/v1/sheets/${sheetId}/ranges/D4:E4`).then(response => response.json())
  // 붙여넣기는 클립보드에서 저장 대기줄을 거쳐 서버까지 간다. 내 자리에서는
  // 눈 깜짝할 새지만 CI 는 느려서 기본 5초를 넘기는 때가 있다.
  await expect.poll(async()=>{
    const result=await pasted()
    return result.items.map((cell:{value:unknown})=>cell.value)
  },{timeout:15_000}).toEqual([2,4])
  const result=await pasted()
  expect(result.items[1].formula).toBe('=D4*2')
})

test('drags the fill handle for numeric series and relative formulas with undo and offline resend', async ({ page, context }) => {
  await page.goto('/')
  await page.getByRole('button', { name:'새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  const workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  const canvas=page.locator('canvas.grid-canvas')
  const edit=async(position:{x:number;y:number},value:string)=>{await canvas.dblclick({position});await page.locator('.cell-editor').fill(value);await page.locator('.cell-editor').press('Enter')}
  const dragFill=async(handle:{x:number;y:number},target:{x:number;y:number})=>{const box=await canvas.boundingBox();if(!box)throw new Error('canvas is not visible');await page.mouse.move(box.x+handle.x,box.y+handle.y);await page.mouse.down();await page.mouse.move(box.x+target.x,box.y+target.y,{steps:6});await page.mouse.up()}
  const values=async(range:string)=>page.request.get(`/api/v1/sheets/${sheetId}/ranges/${range}`).then(response=>response.json())

  await edit({x:70,y:42},'1')
  await edit({x:70,y:69},'2')
  await canvas.click({position:{x:70,y:42}})
  await page.keyboard.press('Shift+ArrowDown')
  await dragFill({x:153,y:80},{x:100,y:149})
  await expect.poll(async()=>(await values('A1:A5')).items.map((cell:{value:unknown})=>cell.value)).toEqual([1,2,3,4,5])

  await edit({x:170,y:42},'=A1*10')
  await canvas.click({position:{x:170,y:42}})
  await dragFill({x:261,y:53},{x:208,y:149})
  await expect.poll(async()=>(await values('B1:B5')).items.map((cell:{value:unknown})=>cell.value)).toEqual([10,20,30,40,50])
  const formulas=await values('B1:B5')
  expect(formulas.items.map((cell:{formula?:string})=>cell.formula)).toEqual(['=A1*10','=A2*10','=A3*10','=A4*10','=A5*10'])

  await page.getByRole('button',{name:'실행 취소'}).click()
  await expect.poll(async()=>(await values('B1:B5')).items.map((cell:{value:unknown})=>cell.value)).toEqual([10])

  await canvas.click({position:{x:70,y:42}})
  await page.keyboard.press('Shift+ArrowDown')
  await context.setOffline(true)
  await dragFill({x:153,y:80},{x:100,y:176})
  await expect(page.getByText('오프라인 · 로컬 저장',{exact:true})).toBeVisible()
  await context.setOffline(false)
  await expect.poll(async()=>(await values('A6')).items[0]?.value,{timeout:15_000}).toBe(6)
})

test('pastes more than 1000 cells without truncation in one version', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId = page.url().split('/workbooks/')[1]
  const workbook = await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response => response.json())
  const sheetId = workbook.sheets[0].id as string
  const text=Array.from({length:1001},(_,index)=>String(index+1)).join('\n')
  await page.locator('.grid-viewport').evaluate((element,pasteText)=>{
    const clipboardData=new DataTransfer()
    clipboardData.setData('text/plain',pasteText)
    element.dispatchEvent(new ClipboardEvent('paste',{bubbles:true,cancelable:true,clipboardData}))
  },text)
  await expect.poll(async()=>{
    const result=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1001`).then(response=>response.json())
    return result.items[0]?.value
  }).toBe(1001)
  const updated=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  expect(updated.version).toBe(2)
})

test('applies display, wrapping, alignment and atomic outer borders from the format dialog', async ({page})=>{
  await page.goto('/')
  await page.getByRole('button',{name:'새 워크북'}).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]
  const workbook=await page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  const sheetId=workbook.sheets[0].id as string
  const canvas=page.locator('canvas.grid-canvas')
  await canvas.dblclick({position:{x:70,y:42}})
  await page.locator('.cell-editor').fill('0.125')
  await page.locator('.cell-editor').press('Enter')
  const nameBox=page.getByLabel('이름 상자')
  await nameBox.fill('A1:B2')
  await nameBox.press('Enter')
  await page.getByRole('menuitem',{name:'서식'}).click()
  await page.getByRole('menuitem',{name:'서식 세부 설정…'}).click()
  const dialog=page.getByRole('dialog',{name:'셀 서식'})
  await expect(dialog).toBeVisible()
  await page.getByLabel('표시 형식').selectOption('0.00%')
  await page.getByLabel('세로 정렬').selectOption('bottom')
  await page.getByLabel('텍스트 배치').selectOption('wrap')
  await page.getByLabel('테두리도 적용').check()
  await dialog.locator('.format-border select').nth(0).selectOption('outer')
  await dialog.locator('.format-border select').nth(1).selectOption('double')
  await page.getByRole('button',{name:'적용',exact:true}).click()

  const cells=async()=>page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1:B2`).then(response=>response.json())
  await expect.poll(async()=>(await cells()).items.length).toBe(4)
  const result=await cells()
  expect(result.items.find((cell:{row:number;column:number})=>cell.row===1&&cell.column===1)?.value).toBe(.125)
  for(const cell of result.items as Array<{style:Record<string,unknown>}>){
    expect(cell.style.number_format).toBe('0.00%')
    expect(cell.style.vertical_align).toBe('bottom')
    expect(cell.style.text_mode).toBe('wrap')
    expect(Object.keys(cell.style.borders as Record<string,unknown>)).toHaveLength(2)
  }
})

test('manages the complete sheet lifecycle without losing copied cells', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: '새 워크북' }).click()
  await page.waitForURL(/\/workbooks\//)
  const workbookId=page.url().split('/workbooks/')[1]

  await page.getByRole('button',{name:'시트 추가'}).click()
  await expect(page.getByRole('button',{name:'Sheet2 시트 메뉴'})).toBeVisible()
  await page.getByRole('button',{name:'Sheet2 시트 메뉴'}).click()
  await page.getByRole('menuitem',{name:'이름 변경'}).click()
  await page.getByRole('textbox',{name:'시트 이름'}).fill('Raw Data')
  await page.getByRole('button',{name:'시트 이름 저장'}).click()
  await expect(page.getByRole('button',{name:'Raw Data 시트 메뉴'})).toBeVisible()

  const canvas=page.locator('canvas.grid-canvas')
  await canvas.dblclick({position:{x:70,y:42}})
  await page.locator('.cell-editor').fill('9')
  await page.locator('.cell-editor').press('Enter')
  await page.getByRole('button',{name:'Raw Data 시트 메뉴'}).click()
  await page.getByRole('menuitem',{name:'복제'}).click()
  await expect(page.getByRole('button',{name:'Raw Data 복사본 시트 메뉴'})).toBeVisible()

  const currentWorkbook=async()=>page.request.get(`/api/v1/workbooks/${workbookId}`).then(response=>response.json())
  await expect.poll(async()=>((await currentWorkbook()).sheets as Array<{name:string}>).map(sheet=>sheet.name)).toEqual(['Sheet1','Raw Data','Raw Data 복사본'])
  let book=await currentWorkbook()
  let copied=book.sheets.find((sheet:{name:string})=>sheet.name==='Raw Data 복사본')
  const copiedCell=await page.request.get(`/api/v1/sheets/${copied.id}/ranges/A1`).then(response=>response.json())
  expect(copiedCell.items[0]?.value).toBe(9)

  await page.getByRole('button',{name:'Raw Data 복사본 시트 메뉴'}).click()
  await page.getByRole('menuitem',{name:'탭 색상'}).click()
  await page.getByRole('menuitemcheckbox',{name:'파랑'}).click()
  await expect.poll(async()=>((await currentWorkbook()).sheets as Array<{name:string;color:string}>).find(sheet=>sheet.name==='Raw Data 복사본')?.color).toBe('#3b82f6')
  await page.getByRole('button',{name:'Raw Data 복사본 시트 메뉴'}).click()
  await page.getByRole('menuitem',{name:'왼쪽으로 이동'}).click()
  await expect.poll(async()=>((await currentWorkbook()).sheets as Array<{name:string;position:number}>).map(sheet=>`${sheet.position}:${sheet.name}`)).toEqual(['0:Sheet1','1:Raw Data 복사본','2:Raw Data'])

  await page.getByRole('button',{name:'Raw Data 복사본 시트 메뉴'}).click()
  page.once('dialog',dialog=>dialog.accept())
  await page.getByRole('menuitem',{name:'시트 삭제'}).click()
  await expect(page.getByRole('button',{name:'Sheet1 시트 메뉴'})).toBeVisible()
  book=await currentWorkbook()
  expect(book.sheets.map((sheet:{name:string;position:number})=>`${sheet.position}:${sheet.name}`)).toEqual(['0:Sheet1','1:Raw Data'])
})
