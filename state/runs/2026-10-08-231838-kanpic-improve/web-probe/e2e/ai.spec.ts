import { expect, test, type APIRequestContext } from '@playwright/test'
import { createServer } from 'node:http'

// Every AI test lives in this one file. The settings the gateway reads are
// global, and Playwright runs separate files in parallel workers, so splitting
// them would let one test switch AI off underneath another.
let aiGatewayPort=0
const requestedBudgets:number[]=[]
const aiGateway=createServer((request,response)=>{
  if(request.method==='GET'&&request.url==='/v1/models'){
    // A vLLM style server publishes the context length, which is what the
    // reply budget is derived from.
    response.writeHead(200,{'Content-Type':'application/json'});response.end(JSON.stringify({data:[{id:'e2e-offline-model',max_model_len:16384}]}));return
  }
  if(request.method==='POST'&&request.url==='/v1/chat/completions'){
    let body=''
    request.on('data',chunk=>{body+=String(chunk)})
    request.on('end',()=>{
      const completion=JSON.parse(body) as {messages:Array<{role:string;content:string}>;max_tokens:number}
      const context=JSON.parse(completion.messages.at(-1)?.content||'{}') as {mode?:string;request?:string;selected_range?:string;workbook_objects?:{charts?:Array<{id:string;type:string}>}}
      requestedBudgets.push((completion as unknown as {max_tokens:number}).max_tokens)
      const currentChart=context.workbook_objects?.charts?.[0]
      const plan=context.request?.includes('선 차트로')&&currentChart
        ?{summary:'선 차트로 변경',explanation:'앞서 만든 차트의 유형을 변경합니다.',findings:[],changes:[],tool_calls:[{name:'update_chart',arguments:{chart_id:currentChart.id,type:'line'}}]}
        :context.request?.includes('막대 차트')
        ?{summary:'막대 차트 생성',explanation:'선택 범위를 막대 차트로 만듭니다.',findings:[],changes:[],tool_calls:[{name:'create_chart',arguments:{type:'bar',title:'선택 범위 차트',source_range:context.selected_range||'A1:B1'}}]}
        :context.mode==='summarize'
        ?{summary:'범위 요약 완료',explanation:'선택 범위의 값이 적습니다.',findings:[],changes:[]}
        :context.mode==='anomaly'
        ?{summary:'이상치 한 건',explanation:'A1은 표본이 적어 검토가 필요합니다.',findings:[{row:1,column:1,severity:'warning',title:'검토 값',description:'비교 표본이 적어 수동 검토가 필요합니다.'}],changes:[]}
        :context.mode==='clean'
          ?{summary:'숫자 형식 정제',explanation:'A1을 문자열 숫자로 표준화합니다.',findings:[],changes:[{row:1,column:1,value:'5'}]}
          :{summary:'A1을 두 배로 계산',explanation:'B1에 A1의 두 배 수식을 제안합니다.',findings:[],changes:[{row:1,column:2,formula:'=A1*2'}]}
      response.writeHead(200,{'Content-Type':'application/json'});response.end(JSON.stringify({choices:[{message:{content:JSON.stringify(plan)},finish_reason:'stop'}],usage:{prompt_tokens:640,completion_tokens:96}}))
    })
    return
  }
  response.writeHead(404);response.end()
})
test.beforeAll(async()=>{await new Promise<void>((resolve,reject)=>{aiGateway.once('error',reject);aiGateway.listen(0,'0.0.0.0',()=>{const address=aiGateway.address();if(typeof address==='object'&&address){aiGatewayPort=address.port;resolve()}else reject(new Error('AI test gateway did not bind'))})})})
test.afterAll(async()=>{await new Promise<void>(resolve=>aiGateway.close(()=>resolve()))})


// The AI settings are global, so every test here sets the values it asserts and
// puts the defaults back afterwards instead of trusting what it inherits.
const put=(request:APIRequestContext,key:string,value:unknown,value_type:'string'|'number'|'boolean')=>
  request.put(`/api/v1/admin/settings/${key}`,{data:{key,value,value_type}})

async function enableAI(request:APIRequestContext,maxInputCells:number,gatewayURL='http://127.0.0.1:9/v1'){
  await put(request,'ai.gateway_url',gatewayURL,'string')
  await put(request,'ai.model','corp-llm-8b','string')
  await put(request,'ai.max_input_cells',maxInputCells,'number')
  await put(request,'ai.enabled',true,'boolean')
}

test.afterEach(async ({ request }) => {
  await put(request,'ai.enabled',false,'boolean')
  await put(request,'ai.max_input_cells',200,'number')
})

test('the AI panel scrolls, explains itself and shows the exact prompt', async ({ page, request }) => {
  await enableAI(request,200)
  const workbook=await request.post('/api/v1/workbooks',{data:{template_id:'monthly-sales'}}).then(response=>response.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  await page.waitForTimeout(800)
  await page.locator('.name-box').fill('A3:F9')
  await page.keyboard.press('Enter')

  const panel=page.getByLabel('AI 도우미 패널')
  await expect(panel.getByLabel('현재 채팅 범위')).toContainText('A3:F9')
  await expect(panel.getByLabel('현재 채팅 범위')).toContainText('42셀')

  // The guide says what happens to the data and how approval works.
  await panel.getByRole('button',{name:'사용 가이드 열기'}).click()
  const guide=page.getByLabel('AI 사용 가이드')
  await expect(guide).toContainText('승인해야 적용됩니다.')
  await expect(guide).toContainText('전송되지 않는 것')
  // Long conversations and supporting views scroll while the composer stays fixed.
  const scroller=panel.locator('.ai-chat-scroll')
  await expect.poll(async()=>scroller.evaluate(element=>element.scrollHeight>element.clientHeight)).toBe(true)
  await scroller.evaluate(element=>element.scrollTo(0,element.scrollHeight))
  // The browser reports the new position on the next frame, not in the same
  // tick as the scroll, so this is polled rather than read once.
  await expect.poll(()=>scroller.evaluate(element=>element.scrollTop)).toBeGreaterThan(0)
  await panel.getByRole('button',{name:'사용 가이드 닫기'}).click()

  // The prompt disclosure shows the real system prompt and the real payload.
  await panel.getByLabel('AI 작업 방식').selectOption('clean')
  await panel.getByRole('button',{name:/모델에 보내는 내용 보기/}).click()
  const disclosure=panel.locator('.ai-disclosure-body')
  await expect(disclosure).toContainText('kanpic')
  await expect(disclosure).toContainText('"mode": "clean"')
  await expect(disclosure).toContainText('"selected_range": "A3:F9"')
  await expect(disclosure.locator('.ai-prompt-meta')).toContainText('셀 42개')
  await expect(disclosure.locator('.ai-prompt-meta')).toContainText('corp-llm-8b')
})

test('the AI panel refuses a selection that exceeds the cell budget', async ({ page, request }) => {
  await enableAI(request,10)
  // The settings are global, so the state this test depends on is asserted.
  expect(await request.get('/api/v1/ai/config').then(response=>response.json())).toMatchObject({enabled:true,max_input_cells:10})
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`AI 상한 ${Date.now()}`}}).then(response=>response.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')
  await page.waitForTimeout(800)
  await page.locator('.name-box').fill('A1:Z100')
  await page.keyboard.press('Enter')

  const panel=page.getByLabel('AI 도우미 패널')
  await expect(panel.getByText(/한 번에 최대 10셀까지/)).toBeVisible()
  await panel.getByRole('textbox',{name:'AI 요청'}).fill('요약해줘')
  await expect(panel.getByRole('button',{name:'AI 메시지 보내기'})).toBeDisabled()
})

test('the prompt preview refuses a workbook the caller cannot read', async ({ request }) => {
  await enableAI(request,200)
  const workbook=await request.post('/api/v1/workbooks',{
    headers:{'X-Kanpic-Actor':'ai.owner@corp.example'},data:{title:`AI 권한 ${Date.now()}`},
  }).then(response=>response.json())
  const body={workbook_id:workbook.id,sheet_id:workbook.sheets[0].id,range:'A1:B2',mode:'summarize',request:'요약',base_version:workbook.version}

  const owner=await request.post('/api/v1/ai/prompt:preview',{headers:{'X-Kanpic-Actor':'ai.owner@corp.example'},data:body})
  expect(owner.status()).toBe(200)
  const stranger=await request.post('/api/v1/ai/prompt:preview',{headers:{'X-Kanpic-Actor':'ai.stranger@corp.example'},data:body})
  expect(stranger.status()).toBe(403)
  const plan=await request.post('/api/v1/ai/actions:plan',{
    headers:{'X-Kanpic-Actor':'ai.stranger@corp.example'},data:{...body,idempotency_key:`stranger-${Date.now()}`},
  })
  expect(plan.status()).toBe(403)
})

test('plans, analyzes, cleans, audits, and undoes offline AI actions', async ({ page }) => {
  const versions=await page.request.get('/api/v1/admin/settings/versions').then(response=>response.json())
  const restoreRevision=versions.items[0].revision as number
  const gatewayHost=process.env.KANPIC_E2E_GATEWAY_HOST||'127.0.0.1'
  const putSetting=(key:string,value:unknown,value_type:'string'|'number'|'boolean',secret=false)=>page.request.put(`/api/v1/admin/settings/${key}`,{data:{key,value,value_type,secret,description:`E2E ${key}`}})
  let workbookId=''
  try{
    expect((await putSetting('ai.gateway_url',`http://${gatewayHost}:${aiGatewayPort}/v1`,'string')).ok()).toBe(true)
    expect((await putSetting('ai.model','e2e-offline-model','string')).ok()).toBe(true)
    expect((await putSetting('ai.max_input_cells',20,'number')).ok()).toBe(true)
    expect((await putSetting('ai.enabled',true,'boolean')).ok()).toBe(true)
    const tested=await page.request.post('/api/v1/admin/settings:test',{data:{}}).then(response=>response.json())
    expect(tested.items.find((item:{name:string})=>item.name==='사내 LLM Gateway')).toMatchObject({success:true})

    const created=await page.request.post('/api/v1/workbooks',{data:{title:`AI 안전 실행 ${Date.now()}`,workspace_id:'default'}}).then(response=>response.json())
    workbookId=created.id as string
    const sheetId=created.sheets[0].id as string
    const seeded=await page.request.patch(`/api/v1/sheets/${sheetId}/cells:batch`,{data:{base_version:1,idempotency_key:`ai-e2e-seed-${workbookId}`,cells:[{row:1,column:1,value:5}]}}).then(response=>response.json())
    await page.goto(`/workbooks/${workbookId}`)
    await page.getByRole('combobox',{name:'이름 상자'}).fill('A1:B1')
    await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
    const panel=page.getByRole('complementary',{name:'AI 도우미 패널'})
    await expect(panel.getByLabel('현재 채팅 범위')).toContainText('A1:B1')
    await expect(panel.getByLabel('현재 채팅 범위')).toContainText('2셀')
    await panel.getByLabel('AI 작업 방식').selectOption('formula')
    await panel.getByRole('textbox',{name:'AI 요청'}).fill('B1에 A1의 두 배 수식을 넣어줘')
    await panel.getByRole('button',{name:'AI 메시지 보내기'}).click()
    await expect(panel.getByLabel('Agent 실행 계획').getByText('A1을 두 배로 계산',{exact:true})).toBeVisible()
    await expect(panel.getByText('=A1*2')).toBeVisible()
    let before=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/B1`).then(response=>response.json())
    expect(before.items).toHaveLength(0)
    await panel.locator('.ai-approval button.primary').click()
    await expect(panel.getByText('승인한 변경이 적용되었습니다.')).toBeVisible()
    // The reply budget comes from the model's published context length instead
    // of a fixed guess, and the token cost is reported back.
    expect(requestedBudgets[0]).toBeGreaterThan(8_192)
    expect(requestedBudgets[0]).toBeLessThan(16_384)
    await expect(panel.getByText(/응답 96토큰/)).toBeVisible()
    await expect.poll(async()=>{const range=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/B1`).then(response=>response.json());return range.items[0]?.formula}).toBe('=A1*2')
    const actions=await page.request.get(`/api/v1/workbooks/${workbookId}/ai/actions`).then(response=>response.json())
    expect(actions.items).toHaveLength(1)
    // 감사 기록은 도구가 끝난 뒤에 쌓인다. 곧바로 읽으면 마지막 한 줄이
    // 아직 없을 수 있어, 실행 전체가 남을 때까지 기다린다.
    await expect.poll(async()=>{
      const audited=await page.request.get(`/api/v1/ai/actions/${actions.items[0].id}`).then(response=>response.json())
      return (audited.events??[]).map((event:{tool_name:string})=>event.tool_name)
    }).toEqual(['range.read','formula.set'])
    await panel.getByRole('button',{name:'Undo'}).click()
    await expect(panel.getByText('AI 변경을 새 서버 버전으로 되돌렸습니다.')).toBeVisible()
    await expect.poll(async()=>{const range=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/B1`).then(response=>response.json());return range.items.length}).toBe(0)

    await panel.getByRole('button',{name:'새 AI 대화 시작'}).click()
    await panel.getByLabel('AI 작업 방식').selectOption('anomaly')
    await panel.getByRole('textbox',{name:'AI 요청'}).fill('선택 범위의 이상치를 찾아줘')
    await panel.getByRole('button',{name:'AI 메시지 보내기'}).click()
    await expect(panel.getByText('탐지 완료',{exact:true})).toBeVisible()
    await expect(panel.getByText('검토 값')).toBeVisible()
    await expect(panel.getByText('현재: 5')).toBeVisible()
    await expect(panel.locator('.ai-approval button.primary')).toHaveCount(0)

    await panel.getByRole('button',{name:'새 AI 대화 시작'}).click()
    await panel.getByLabel('AI 작업 방식').selectOption('clean')
    await panel.getByRole('textbox',{name:'AI 요청'}).fill('A1의 숫자 형식을 정제해줘')
    await panel.getByRole('button',{name:'AI 메시지 보내기'}).click()
    await expect(panel.getByLabel('Agent 실행 계획').getByText('숫자 형식 정제',{exact:true})).toBeVisible()
    expect((await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1`).then(response=>response.json())).items[0].value).toBe(5)
    await panel.locator('.ai-approval button.primary').click()
    await expect(panel.getByText('승인한 변경이 적용되었습니다.')).toBeVisible()
    await expect.poll(async()=>{const range=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1`).then(response=>response.json());return range.items[0]?.value}).toBe('5')
    const latestActions=await page.request.get(`/api/v1/workbooks/${workbookId}/ai/actions`).then(response=>response.json())
    const cleanAction=latestActions.items.find((item:{mode:string})=>item.mode==='clean')
    const cleanAudit=await page.request.get(`/api/v1/ai/actions/${cleanAction.id}`).then(response=>response.json())
    expect(cleanAudit.events.map((event:{tool_name:string})=>event.tool_name)).toEqual(['range.read','data.clean'])
    await panel.getByRole('button',{name:'Undo'}).click()
    await expect.poll(async()=>{const range=await page.request.get(`/api/v1/sheets/${sheetId}/ranges/A1`).then(response=>response.json());return range.items[0]?.value}).toBe(5)

    await panel.getByRole('button',{name:'새 AI 대화 시작'}).click()
    await panel.getByRole('textbox',{name:'AI 요청'}).fill('선택 범위로 막대 차트를 만들어줘')
    await panel.getByRole('button',{name:'AI 메시지 보내기'}).click()
    await expect(panel.getByLabel('Agent 실행 계획').getByText('막대 차트 생성',{exact:true})).toBeVisible()
    await panel.locator('.ai-approval button.primary').click()
    await expect.poll(async()=>{const result=await page.request.get(`/api/v1/workbooks/${workbookId}/charts`).then(response=>response.json());return result.items[0]?.type}).toBe('bar')
    await panel.getByRole('textbox',{name:'AI 요청'}).fill('막대 차트를 선 차트로 바꿔줘')
    await panel.getByRole('button',{name:'AI 메시지 보내기'}).click()
    await expect(panel.getByLabel('Agent 실행 계획').getByText('선 차트로 변경',{exact:true})).toBeVisible()
    await expect(panel.getByLabel('Agent 대화')).toContainText('선택 범위로 막대 차트를 만들어줘')
    await expect(panel.getByLabel('Agent 대화')).toContainText('막대 차트를 선 차트로 바꿔줘')
    await panel.locator('.ai-approval button.primary').click()
    await expect.poll(async()=>{const result=await page.request.get(`/api/v1/workbooks/${workbookId}/charts`).then(response=>response.json());return result.items[0]?.type}).toBe('line')
    expect(seeded.server_version).toBe(2)
  }finally{
    if(workbookId)await page.request.delete(`/api/v1/workbooks/${workbookId}`)
    await page.request.post(`/api/v1/admin/settings/versions/${restoreRevision}:restore`,{data:{}})
  }
})

test('the console lists AI history, exports it and prunes it', async ({ page, request }) => {
  // This test needs a gateway that answers, so it uses the one this file runs.
  await enableAI(request,200,`http://${process.env.KANPIC_E2E_GATEWAY_HOST||'127.0.0.1'}:${aiGatewayPort}/v1`)
  const workbook=await request.post('/api/v1/workbooks',{
    headers:{'X-Kanpic-Actor':'history.owner@corp.example'},data:{title:`AI 이력 ${Date.now()}`},
  }).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${workbook.sheets[0].id}/cells:batch`,{
    headers:{'X-Kanpic-Actor':'history.owner@corp.example'},
    data:{base_version:workbook.version,idempotency_key:`hist-${Date.now()}`,cells:[{row:1,column:1,value:5}]},
  })
  // The gateway in this file answers plans, so a real action lands in history.
  const planned=await request.post('/api/v1/ai/actions:plan',{
    headers:{'X-Kanpic-Actor':'history.owner@corp.example'},
    data:{workbook_id:workbook.id,sheet_id:workbook.sheets[0].id,range:'A1:B1',mode:'summarize',request:'콘솔 이력 확인용 요약',base_version:workbook.version+1,idempotency_key:`console-${Date.now()}`},
  })
  expect(planned.status()).toBe(201)

  await page.goto('/admin?tab=ai')
  await expect(page.getByRole('heading',{name:'AI 호출 이력'})).toBeVisible()
  const row=page.locator('.ai-history-row',{hasText:'history.owner@corp.example'}).first()
  await expect(row).toBeVisible()
  await expect(row).toContainText('범위 요약')

  // The detail dialog shows the request and the event trail.
  await row.click()
  const detail=page.getByRole('dialog',{name:'AI 호출 상세'})
  await expect(detail).toContainText('콘솔 이력 확인용 요약')
  await expect(detail).toContainText('이벤트')
  await page.keyboard.press('Escape')

  // A filter that excludes the row empties the table.
  await page.getByRole('combobox',{name:'상태'}).selectOption('failed')
  await expect(page.getByText('조건에 맞는 AI 호출이 없습니다.')).toBeVisible()
  await page.getByRole('combobox',{name:'상태'}).selectOption('')

  const csv=await request.get('/api/v1/admin/ai/actions?format=csv')
  expect(csv.headers()['content-type']).toContain('text/csv')
  expect(await csv.text()).toContain('history.owner@corp.example')

  // Pruning removes finished actions before the chosen day.
  const purged=await request.delete('/api/v1/admin/ai/actions?before=2099-01-01').then(response=>response.json())
  expect(purged.removed).toBeGreaterThanOrEqual(1)
  await page.reload()
  await expect(page.locator('.ai-history-row',{hasText:'history.owner@corp.example'})).toHaveCount(0)
})
