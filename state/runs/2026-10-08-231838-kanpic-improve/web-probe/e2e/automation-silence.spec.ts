import { expect, test } from '@playwright/test'

// 자동화 실행은 관리자 설정이고 기본값이 꺼짐이다. 그런데 워크북 쪽에는
// 그 사실이 어디에도 없었다. 정의를 만들고 검증까지 통과한 다음 실행을
// 눌러야 비로소 503이 돌아왔고, 셀 변경·일정·웹훅 트리거는 아무 말 없이
// 영영 돌지 않았다.
test('the panel says when automation execution is switched off', async ({ page }) => {
  const versions=await page.request.get('/api/v1/admin/settings/versions').then(response=>response.json())
  const restoreRevision=versions.items[0].revision as number
  const putSetting=(key:string,value:unknown)=>page.request.put(`/api/v1/admin/settings/${key}`,{data:{key,value,value_type:typeof value==='boolean'?'boolean':'number',secret:false,description:`E2E ${key}`}})
  let workbookId=''
  try{
    expect((await putSetting('automation.enabled',false)).ok()).toBe(true)
    const created=await page.request.post('/api/v1/workbooks',{data:{title:`자동화 중지 ${Date.now()}`,workspace_id:'default'}}).then(response=>response.json())
    workbookId=created.id as string
    const sheetId=created.sheets[0].id as string
    await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:1,idempotency_key:`off-seed-${workbookId}`,cells:[{row:1,column:1,value:4}]}})
    await page.request.post(`/api/v1/workbooks/${workbookId}/automations`,{data:{name:'두 배',enabled:true,idempotency_key:`off-create-${workbookId}`,trigger:{type:'manual'},action:{type:'set_formula',sheet_id:sheetId,range:'B1',formula:'=A1*2'}}})

    await page.goto(`/workbooks/${workbookId}`)
    await page.getByRole('button',{name:'자동화 패널'}).click()
    const panel=page.getByRole('complementary',{name:'자동화 패널'})
    await expect(panel).toContainText('자동화 실행이 꺼져 있습니다')
    await panel.getByRole('button',{name:/검증/}).click()
    await expect(panel.getByText('실행 미리보기')).toBeVisible()
    // 검증은 통과한다. 실행만 서버에서 거절당하므로 버튼이 그 이유를 들고 있어야 한다.
    await expect(panel.getByRole('button',{name:/검토한 자동화 실행/})).toBeDisabled()

    expect((await putSetting('automation.enabled',true)).ok()).toBe(true)
    await page.reload()
    await page.getByRole('button',{name:'자동화 패널'}).click()
    await expect(panel).not.toContainText('자동화 실행이 꺼져 있습니다')
    await panel.getByRole('button',{name:/검증/}).click()
    await expect(panel.getByRole('button',{name:/검토한 자동화 실행/})).toBeEnabled()
  }finally{
    if(workbookId)await page.request.delete(`/api/v1/workbooks/${workbookId}`)
    await page.request.post(`/api/v1/admin/settings/versions/${restoreRevision}:restore`,{data:{}})
  }
})

// 셀 변경 트리거가 실패하면 서버 로그에만 남았다. 편집한 사람에게도,
// 실행 이력에도 아무 흔적이 없어 자동화가 조용히 죽은 줄 알 수 없었다.
test('a cell-change automation that fails tells the editor and leaves a record', async ({ page }) => {
  const versions=await page.request.get('/api/v1/admin/settings/versions').then(response=>response.json())
  const restoreRevision=versions.items[0].revision as number
  const putSetting=(key:string,value:unknown)=>page.request.put(`/api/v1/admin/settings/${key}`,{data:{key,value,value_type:typeof value==='boolean'?'boolean':'number',secret:false,description:`E2E ${key}`}})
  let workbookId=''
  try{
    expect((await putSetting('automation.enabled',true)).ok()).toBe(true)
    expect((await putSetting('automation.max_cells_per_run',1000)).ok()).toBe(true)
    const created=await page.request.post('/api/v1/workbooks',{data:{title:`자동화 실패 ${Date.now()}`,workspace_id:'default'}}).then(response=>response.json())
    workbookId=created.id as string
    const sheetId=created.sheets[0].id as string
    const automation=await page.request.post(`/api/v1/workbooks/${workbookId}/automations`,{data:{name:'A열 감시',enabled:true,idempotency_key:`fail-create-${workbookId}`,trigger:{type:'cell_change',sheet_id:sheetId,range:'A1:A5'},action:{type:'set_value',sheet_id:sheetId,range:'C1:C5',value:'복사됨'}}}).then(response=>response.json())
    // 정의를 저장한 뒤에 관리자가 한도를 낮추면 그 자동화는 실행할 때마다 실패한다.
    expect((await putSetting('automation.max_cells_per_run',1)).ok()).toBe(true)

    await page.goto(`/workbooks/${workbookId}`)
    await expect(page.locator('.grid-canvas')).toBeVisible()
    const box=(await page.locator('.grid-canvas').boundingBox())!
    await page.mouse.click(box.x+40,box.y+42)
    await page.keyboard.type('12')
    await page.keyboard.press('Enter')

    const notice=page.locator('.formula-issue')
    await expect(notice).toContainText('자동화가 실패했습니다')
    await expect(notice).toContainText('1 cell limit')

    await page.getByRole('button',{name:'자동화 패널'}).click()
    const panel=page.getByRole('complementary',{name:'자동화 패널'})
    await expect(panel).toContainText('마지막 실행 실패')
    const runs=await page.request.get(`/api/v1/automations/${automation.id}/runs`).then(response=>response.json())
    expect(runs.items[0]).toMatchObject({trigger_type:'cell_change',status:'failed'})
    expect(runs.items[0].error_message).toContain('1 cell limit')
  }finally{
    if(workbookId)await page.request.delete(`/api/v1/workbooks/${workbookId}`)
    await page.request.post(`/api/v1/admin/settings/versions/${restoreRevision}:restore`,{data:{}})
  }
})
