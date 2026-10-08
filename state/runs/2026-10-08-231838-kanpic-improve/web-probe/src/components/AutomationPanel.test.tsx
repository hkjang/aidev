import { QueryClient,QueryClientProvider } from '@tanstack/react-query'
import { cleanup,fireEvent,render,screen,waitFor } from '@testing-library/react'
import { afterEach,describe,expect,it,vi } from 'vitest'
import type { Automation,AutomationExecutionResult,AutomationPreview,AutomationRun,MutationResult,Sheet } from '../types'
import { AutomationPanel } from './AutomationPanel'

const sheet:Sheet={id:'sheet-1',workbook_id:'book-1',name:'Data',position:0,hidden:false,layout:{revision:1,frozen_rows:0,frozen_columns:0},created_at:'2026-08-02T00:00:00Z'}
const automation:Automation={id:'automation-1',workbook_id:'book-1',name:'상태 완료',enabled:true,trigger:{type:'manual'},action:{type:'set_value',sheet_id:sheet.id,range:'B1',value:'완료'},revision:1,created_by:'alice',updated_by:'alice',created_at:'2026-08-02T00:00:00Z',updated_at:'2026-08-02T00:00:00Z'}
const preview:AutomationPreview={automation_id:automation.id,automation_name:automation.name,workbook_id:automation.workbook_id,automation_revision:automation.revision,base_version:2,action:automation.action,changes:[{row:1,column:2,address:'B1',before:{},after:{value:'완료'}}]}
const operation:MutationResult={operation_id:'operation-1',workbook_id:'book-1',sheet_id:sheet.id,base_version:2,server_version:3,applied_cells:1,recalculated_cells:[],formula_errors:[],validation_warnings:[],duplicate:false,conflicts:[]}
const run:AutomationRun={id:'run-1',automation_id:automation.id,workbook_id:'book-1',actor_id:'alice',trigger_type:'manual',status:'succeeded',base_version:2,action:automation.action,operation_id:operation.operation_id,operation,started_at:'2026-08-02T00:01:00Z',completed_at:'2026-08-02T00:01:01Z',updated_at:'2026-08-02T00:01:01Z'}
const executed:AutomationExecutionResult={run,operation,changes:[{row:1,column:2,value:'완료'}]}

afterEach(()=>{cleanup();vi.restoreAllMocks();vi.unstubAllGlobals()})

function renderPanel(onExecuted=vi.fn(),options:{workbookVersion?:number;prepareExecution?:()=>Promise<number>}={}){
  const client=new QueryClient({defaultOptions:{queries:{retry:false,gcTime:0}}})
  render(<QueryClientProvider client={client}><AutomationPanel workbookId="book-1" workbookVersion={options.workbookVersion??2} sheets={[sheet]} activeSheetId={sheet.id} selectionRange="B1" prepareExecution={options.prepareExecution??vi.fn(async()=>options.workbookVersion??2)} onClose={vi.fn()} onExecuted={onExecuted}/></QueryClientProvider>)
  return onExecuted
}

describe('AutomationPanel',()=>{
  it('creates an automation, previews exact changes, then explicitly runs it',async()=>{
    let items:Automation[]=[]
    let runs:AutomationRun[]=[]
    const fetchMock=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
      const path=String(input),method=init?.method??'GET'
      if(path==='/api/v1/workbooks/book-1/automations'&&method==='GET')return json({items})
      if(path==='/api/v1/workbooks/book-1/automations'&&method==='POST'){
        const body=JSON.parse(String(init?.body)) as Record<string,unknown>
        expect(body).toMatchObject({name:'상태 완료',enabled:true,trigger:{type:'manual'},action:{type:'set_value',sheet_id:sheet.id,range:'B1',value:'완료'}})
        expect(body.idempotency_key).toBeTruthy()
        items=[automation]
        return json(automation,201)
      }
      if(path==='/api/v1/automations/automation-1:test')return json(preview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:runs})
      if(path==='/api/v1/automations/automation-1:run'){
        const body=JSON.parse(String(init?.body)) as Record<string,unknown>
        expect(body).toMatchObject({expected_revision:1,expected_base_version:2,client_id:expect.any(String)})
        expect(body.idempotency_key).toBeTruthy()
        runs=[run]
        return json(executed)
      }
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    const onExecuted=renderPanel()
    expect(await screen.findByText('등록된 자동화가 없습니다')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button',{name:/새 자동화/}))
    fireEvent.change(screen.getByLabelText('자동화 이름'),{target:{value:'상태 완료'}})
    fireEvent.click(screen.getByRole('button',{name:/저장 후 검증/}))
    expect(await screen.findByText('실행 미리보기')).toBeInTheDocument()
    expect(screen.getByText('(빈 셀)')).toBeInTheDocument()
    expect(screen.getAllByText('완료').length).toBeGreaterThan(0)
    expect(screen.getAllByText(/Data!B1/).length).toBeGreaterThan(0)
    expect(onExecuted).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button',{name:/검토한 자동화 실행/}))
    await waitFor(()=>expect(onExecuted).toHaveBeenCalledWith(executed))
  })

  it('keeps the same Undo idempotency key until a retry succeeds',async()=>{
    let currentRun=run
    let undoAttempts=0
    const undoKeys:string[]=[]
    const undoOperation:MutationResult={...operation,operation_id:'operation-undo',base_version:3,server_version:4}
    const undone:AutomationExecutionResult={run:{...run,status:'undone',undo_operation_id:undoOperation.operation_id,undo_operation:undoOperation},operation:undoOperation}
    const fetchMock=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations')return json({items:[automation]})
      if(path==='/api/v1/automations/automation-1:test')return json(preview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:[currentRun]})
      if(path==='/api/v1/automation-runs/run-1:undo'){
        const body=JSON.parse(String(init?.body)) as Record<string,unknown>
        expect(body.idempotency_key).toBeTruthy()
        expect(body.client_id).toBeTruthy()
        undoKeys.push(String(body.idempotency_key))
        undoAttempts+=1
        if(undoAttempts===1)throw new TypeError('응답을 받지 못했습니다.')
        currentRun=undone.run
        return json(undone)
      }
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    const onExecuted=renderPanel()
    const validate=await screen.findByRole('button',{name:/검증/})
    fireEvent.click(validate)
    expect(await screen.findByText('실행 이력')).toBeInTheDocument()
    const undoButton=await screen.findByRole('button',{name:/Undo/})
    fireEvent.click(undoButton)
    await screen.findByText('응답을 받지 못했습니다.')
    fireEvent.click(screen.getByRole('button',{name:/Undo/}))
    await waitFor(()=>expect(onExecuted).toHaveBeenCalledWith(undone))
    expect(undoKeys).toHaveLength(2)
    expect(undoKeys[1]).toBe(undoKeys[0])
  })

  it('keeps the preview pinned and reuses the run key after a lost response',async()=>{
    let attempts=0
    const keys:string[]=[]
    const prepareExecution=vi.fn(async()=>2)
    const fetchMock=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations')return json({items:[automation]})
      if(path==='/api/v1/automations/automation-1:test')return json(preview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:[]})
      if(path==='/api/v1/automations/automation-1:run'){
        const body=JSON.parse(String(init?.body)) as Record<string,unknown>
        expect(body).toMatchObject({expected_revision:1,expected_base_version:2})
        keys.push(String(body.idempotency_key));attempts+=1
        if(attempts===1)throw new TypeError('네트워크 응답이 유실되었습니다.')
        return json(executed)
      }
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    const onExecuted=renderPanel(vi.fn(),{prepareExecution})
    fireEvent.click(await screen.findByRole('button',{name:/검증/}))
    const runButton=await screen.findByRole('button',{name:/검토한 자동화 실행/})
    fireEvent.click(runButton)
    await screen.findByText('네트워크 응답이 유실되었습니다.')
    fireEvent.click(screen.getByRole('button',{name:/검토한 자동화 실행/}))
    await waitFor(()=>expect(onExecuted).toHaveBeenCalledWith(executed))
    expect(keys).toHaveLength(2)
    expect(keys[1]).toBe(keys[0])
    expect(prepareExecution).toHaveBeenCalledTimes(3)
  })

  it('treats an empty preview as a successful no-op and disables execution',async()=>{
    const emptyPreview:AutomationPreview={...preview,changes:[]}
    const fetchMock=vi.fn(async(input:RequestInfo|URL)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations')return json({items:[automation]})
      if(path==='/api/v1/automations/automation-1:test')return json(emptyPreview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:[]})
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel()
    fireEvent.click(await screen.findByRole('button',{name:/검증/}))
    expect(await screen.findByText(/실행할 변경이 없습니다/)).toBeInTheDocument()
    expect(screen.getByRole('button',{name:/검토한 자동화 실행/})).toBeDisabled()
    expect(fetchMock.mock.calls.some(([input])=>String(input).endsWith(':run'))).toBe(false)
  })

  it('waits for local edits and stops preview when preparation fails',async()=>{
    const prepareExecution=vi.fn(async()=>{throw new Error('저장 대기 중인 변경이 있습니다.')})
    const fetchMock=vi.fn(async(input:RequestInfo|URL)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations')return json({items:[automation]})
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:[]})
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel(vi.fn(),{prepareExecution})
    fireEvent.click(await screen.findByRole('button',{name:/검증/}))
    expect(await screen.findByText('저장 대기 중인 변경이 있습니다.')).toBeInTheDocument()
    expect(prepareExecution).toHaveBeenCalledOnce()
    expect(fetchMock.mock.calls.some(([input])=>String(input).endsWith(':test'))).toBe(false)
  })

  it('disables a preview when the workbook version has advanced',async()=>{
    const fetchMock=vi.fn(async(input:RequestInfo|URL)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations')return json({items:[automation]})
      if(path==='/api/v1/automations/automation-1:test')return json(preview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:[]})
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel(vi.fn(),{workbookVersion:3})
    fireEvent.click(await screen.findByRole('button',{name:/검증/}))
    expect(await screen.findByText(/워크북 또는 자동화 정의가 변경되었습니다/)).toBeInTheDocument()
    expect(screen.getByRole('button',{name:/검토한 자동화 실행/})).toBeDisabled()
    expect(screen.getByRole('button',{name:/다시 검증/})).toBeEnabled()
  })

  it('blocks execution when preparing pending edits advances the workbook version',async()=>{
    let preparations=0
    const prepareExecution=vi.fn(async()=>{preparations+=1;return preparations===1?2:3})
    const fetchMock=vi.fn(async(input:RequestInfo|URL)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations')return json({items:[automation]})
      if(path==='/api/v1/automations/automation-1:test')return json(preview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:[]})
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel(vi.fn(),{prepareExecution})
    fireEvent.click(await screen.findByRole('button',{name:/검증/}))
    fireEvent.click(await screen.findByRole('button',{name:/검토한 자동화 실행/}))
    expect(await screen.findByText(/저장된 셀 변경으로 워크북 버전이 바뀌었습니다/)).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([input])=>String(input).endsWith(':run'))).toBe(false)
    expect(screen.queryByRole('button',{name:/이전 실행 응답 확인/})).not.toBeInTheDocument()
  })

  it('pins labels to the preview snapshot and blocks a newer server definition missing from the list cache',async()=>{
    const newer:Automation={...automation,name:'서버의 새 정의',revision:2,action:{...automation.action,range:'C1'}}
    const newerPreview:AutomationPreview={...preview,automation_name:newer.name,automation_revision:newer.revision,action:newer.action,changes:[{...preview.changes[0],column:3,address:'C1'}]}
    const fetchMock=vi.fn(async(input:RequestInfo|URL)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations')return json({items:[automation]})
      if(path==='/api/v1/automations/automation-1:test')return json(newerPreview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:[]})
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel()
    fireEvent.click(await screen.findByRole('button',{name:/검증/}))
    expect(await screen.findByText(/서버의 새 정의 · 정의 r2/)).toBeInTheDocument()
    expect(screen.getByText('Data!C1')).toBeInTheDocument()
    expect(screen.getByText(/워크북 또는 자동화 정의가 변경되었습니다/)).toBeInTheDocument()
    expect(screen.getByRole('button',{name:/검토한 자동화 실행/})).toBeDisabled()
  })

  it('normalizes literal kinds, rejects non-finite numbers, and reuses the create key',async()=>{
    const booleanAutomation:Automation={...automation,action:{...automation.action,value:true}}
    let attempts=0
    const keys:string[]=[]
    const fetchMock=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
      const path=String(input),method=init?.method??'GET'
      if(path==='/api/v1/workbooks/book-1/automations'&&method==='GET')return json({items:attempts>1?[booleanAutomation]:[]})
      if(path==='/api/v1/workbooks/book-1/automations'&&method==='POST'){
        const body=JSON.parse(String(init?.body)) as {idempotency_key:string;action:{value:unknown}}
        keys.push(body.idempotency_key)
        expect(body.action.value).toBe(true)
        attempts+=1
        if(attempts===1)throw new TypeError('저장 응답이 유실되었습니다.')
        return json(booleanAutomation,201)
      }
      if(path==='/api/v1/automations/automation-1:test')return json(preview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:[]})
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel()
    await screen.findByText('등록된 자동화가 없습니다')
    fireEvent.click(screen.getByRole('button',{name:/새 자동화/}))
    fireEvent.change(screen.getByLabelText('자동화 값 유형'),{target:{value:'number'}})
    expect(screen.getByLabelText('자동화 값')).toHaveValue('0')
    fireEvent.change(screen.getByLabelText('자동화 값'),{target:{value:'Infinity'}})
    expect(screen.getByText('유한한 숫자를 입력하세요.')).toBeInTheDocument()
    expect(screen.getByRole('button',{name:/저장 후 검증/})).toBeDisabled()
    fireEvent.change(screen.getByLabelText('자동화 값 유형'),{target:{value:'boolean'}})
    expect(screen.getByLabelText('자동화 값')).toHaveValue('true')
    fireEvent.click(screen.getByRole('button',{name:/저장 후 검증/}))
    await screen.findByText('저장 응답이 유실되었습니다.')
    fireEvent.click(screen.getByRole('button',{name:/저장 후 검증/}))
    expect(await screen.findByText('실행 미리보기')).toBeInTheDocument()
    expect(keys).toHaveLength(2)
    expect(keys[1]).toBe(keys[0])
  })

  it('shows list failures without an empty-state contradiction and retries them',async()=>{
    let attempts=0
    const fetchMock=vi.fn(async(input:RequestInfo|URL)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations'){
        attempts+=1
        return attempts===1?json({error:{message:'목록 오류'}},500):json({items:[automation]})
      }
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel()
    expect(await screen.findByText('자동화 목록을 불러오지 못했습니다.')).toBeInTheDocument()
    expect(screen.queryByText('등록된 자동화가 없습니다')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button',{name:'자동화 목록 다시 시도'}))
    expect(await screen.findByRole('button',{name:/검증/})).toBeInTheDocument()
  })

  it('reports and retries run-history loading failures',async()=>{
    let historyAttempts=0
    const fetchMock=vi.fn(async(input:RequestInfo|URL)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations')return json({items:[automation]})
      if(path==='/api/v1/automations/automation-1:test')return json(preview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12'){
        historyAttempts+=1
        return historyAttempts===1?json({error:{message:'이력 오류'}},500):json({items:[]})
      }
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel()
    fireEvent.click(await screen.findByRole('button',{name:/검증/}))
    const historyError=await screen.findByText('실행 이력을 불러오지 못했습니다.')
    fireEvent.click(historyError.parentElement!.querySelector('button')!)
    expect(await screen.findByText('실행 이력이 없습니다.')).toBeInTheDocument()
  })

  it('creates a timezone-aware Cron schedule and shows its next run',async()=>{
    const scheduled:Automation={...automation,id:'schedule-1',name:'매시간 갱신',trigger:{type:'schedule',cron:'0 * * * *',timezone:'Asia/Seoul'},next_run_at:'2026-08-02T01:00:00Z'}
    let items:Automation[]=[]
    const fetchMock=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
      const path=String(input),method=init?.method??'GET'
      if(path==='/api/v1/workbooks/book-1/automations'&&method==='GET')return json({items})
      if(path==='/api/v1/workbooks/book-1/automations'&&method==='POST'){
        const body=JSON.parse(String(init?.body)) as Record<string,unknown>
        expect(body).toMatchObject({name:'매시간 갱신',trigger:{type:'schedule',cron:'0 * * * *',timezone:'Asia/Seoul'}})
        items=[scheduled]
        return json(scheduled,201)
      }
      if(path==='/api/v1/automations/schedule-1:test')return json({...preview,automation_id:scheduled.id})
      if(path==='/api/v1/automations/schedule-1/runs?limit=12')return json({items:[]})
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel()
    await screen.findByText('등록된 자동화가 없습니다')
    fireEvent.click(screen.getByRole('button',{name:/새 자동화/}))
    fireEvent.change(screen.getByLabelText('자동화 이름'),{target:{value:'매시간 갱신'}})
    fireEvent.change(screen.getByLabelText('자동화 트리거'),{target:{value:'schedule'}})
    fireEvent.change(screen.getByLabelText('스케줄 프리셋'),{target:{value:'0 * * * *'}})
    fireEvent.click(screen.getByRole('button',{name:/저장 후 검증/}))
    expect(await screen.findByText(/다음 실행/)).toBeInTheDocument()
    expect(screen.getByText(/0 \* \* \* \*/)).toBeInTheDocument()
  })

  it('creates an API-key authenticated inbound webhook and shows its endpoint',async()=>{
    const webhook:Automation={...automation,id:'webhook-1',name:'승인 수신',trigger:{type:'webhook'}}
    let items:Automation[]=[]
    const fetchMock=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
      const path=String(input),method=init?.method??'GET'
      if(path==='/api/v1/workbooks/book-1/automations'&&method==='GET')return json({items})
      if(path==='/api/v1/workbooks/book-1/automations'&&method==='POST'){
        const body=JSON.parse(String(init?.body)) as Record<string,unknown>
        expect(body).toMatchObject({name:'승인 수신',trigger:{type:'webhook'}})
        items=[webhook]
        return json(webhook,201)
      }
      if(path==='/api/v1/automations/webhook-1:test')return json({...preview,automation_id:webhook.id})
      if(path==='/api/v1/automations/webhook-1/runs?limit=12')return json({items:[]})
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel()
    await screen.findByText('등록된 자동화가 없습니다')
    fireEvent.click(screen.getByRole('button',{name:/새 자동화/}))
    fireEvent.change(screen.getByLabelText('자동화 이름'),{target:{value:'승인 수신'}})
    fireEvent.change(screen.getByLabelText('자동화 트리거'),{target:{value:'webhook'}})
    expect(screen.getByText(/JSON 원문은 저장하지 않습니다/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button',{name:/저장 후 검증/}))
    expect(await screen.findByText(/POST \/api\/v1\/automations\/webhook-1:webhook/)).toBeInTheDocument()
  })

  // 자동화 실행은 관리자 설정이고 기본값이 꺼짐이다. 패널이 그 사실을 말하지
  // 않으면 저장도 검증도 통과한 자동화가 왜 안 도는지 알 길이 없었다.
  it('says that execution is switched off and refuses to offer the run button',async()=>{
    const failing:Automation={...automation,id:'automation-2',name:'예약 갱신',trigger:{type:'schedule',cron:'0 * * * *',timezone:'Asia/Seoul'},next_run_at:'2026-08-02T01:00:00Z',last_failure:{run_id:'run-9',trigger_type:'schedule',message:'automation exceeds the 1 cell limit',at:'2026-08-02T00:30:00Z'}}
    const fetchMock=vi.fn(async(input:RequestInfo|URL)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/automations')return json({items:[automation,failing],execution_enabled:false})
      if(path==='/api/v1/automations/automation-1:test')return json(preview)
      if(path==='/api/v1/automations/automation-1/runs?limit=12')return json({items:[]})
      return json({},404)
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel()
    expect(await screen.findByText('자동화 실행이 꺼져 있습니다')).toBeInTheDocument()
    // 돌지 않는 자동화의 "다음 실행 …"은 사실이 아니다.
    expect(screen.queryByText(/다음 실행/)).not.toBeInTheDocument()
    expect(screen.getByText(/마지막 실행 실패/)).toHaveTextContent('1 cell limit')
    fireEvent.click(screen.getAllByRole('button',{name:/검증/})[0])
    await screen.findByText('실행 미리보기')
    expect(screen.getByRole('button',{name:/검토한 자동화 실행/})).toBeDisabled()
  })
})

function json(value:unknown,status=200){return new Response(JSON.stringify(value),{status,headers:{'Content-Type':'application/json'}})}
