import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Bot, Check, ChevronDown, Clipboard, Clock3, Eye, HelpCircle, ListChecks, MessagesSquare, MessageSquarePlus, RefreshCw, RotateCcw, Send, ShieldCheck, Sparkles, Terminal, Wrench, XCircle } from 'lucide-react'
import { FormEvent, useEffect, useRef, useState } from 'react'
import { api, newIdempotencyKey } from '../lib/api'
import { collaborationClientId } from '../lib/client'
import type { AIAction, AIConfig, AIExecutionResult, AICellSnapshot, AIPromptPreview, AIUsage, AgentContext, AgentConversation, AgentExecutionResult, AgentRun, AgentToolCall } from '../types'

type Mode=AIAction['mode']
type Props={workbookId:string;workbookName?:string;sheetId:string;sheetName?:string;selectionRange:string;baseVersion:number;initialMode?:Mode;initialRequest?:string;onClose:()=>void;onExecuted:(result:AIExecutionResult)=>void}

const modeLabel:Record<Mode,string>={formula:'수식 생성',explain:'수식 설명',fix:'수식 오류 수정',summarize:'범위 분석',anomaly:'이상치 탐지',clean:'데이터 정제',format:'자동 서식',chart:'차트 작업',agent:'자동 에이전트'}

/** What each mode does, whether it writes, and a request that works well. */
const MODES:Array<{id:Mode;hint:string;writes:boolean;examples:string[]}>=[
  {id:'agent',hint:'요청 의도를 자동으로 판단하고 이전 대화와 현재 워크북 상태를 이어서 작업합니다.',writes:true,examples:['선택 범위로 막대 차트를 만들어줘','방금 만든 막대 차트를 선 차트로 바꿔줘']},
  {id:'summarize',hint:'선택 범위의 핵심 지표와 패턴을 정리합니다.',writes:false,examples:['핵심 지표와 눈에 띄는 변화를 3줄로 요약해줘','제품별 매출 비중과 상위 항목을 알려줘']},
  {id:'anomaly',hint:'평균이나 추세에서 벗어난 값을 찾아 알려 줍니다.',writes:false,examples:['평균에서 크게 벗어난 값을 찾아줘','전월 대비 급격히 변한 항목을 알려줘']},
  {id:'explain',hint:'선택한 수식의 계산 방식과 참조를 설명합니다.',writes:false,examples:['이 수식이 무엇을 계산하는지 설명해줘','참조하는 범위와 계산 순서를 알려줘']},
  {id:'formula',hint:'요청한 계산을 수행하는 수식을 제안합니다. 승인해야 적용됩니다.',writes:true,examples:['각 행의 합계를 오른쪽 빈 열에 계산하는 수식','전월 대비 증감율을 계산하는 수식을 만들어줘']},
  {id:'fix',hint:'오류가 난 수식의 원인을 찾아 수정안을 제안합니다.',writes:true,examples:['#REF! 오류의 원인을 찾아 고쳐줘','합계가 비어 있는 수식을 수정해줘']},
  {id:'clean',hint:'공백·대소문자·날짜 형식 등을 일관되게 정리합니다.',writes:true,examples:['앞뒤 공백을 없애고 날짜 형식을 YYYY-MM-DD로 통일해줘','전화번호 형식을 010-0000-0000으로 맞춰줘']},
  {id:'format',hint:'헤더·데이터 유형을 분석해 안전한 셀 서식을 제안합니다.',writes:true,examples:['이 데이터를 보기 좋게 정리해줘','헤더와 숫자 형식을 보고서 스타일로 바꿔줘']},
  {id:'chart',hint:'선택 범위와 계열을 검증한 뒤 적합한 차트를 제안합니다.',writes:true,examples:['월별 매출 차트를 만들어줘','카테고리별 값을 막대 차트로 보여줘']},
]

const snapshotText=(snapshot:AICellSnapshot)=>snapshot.formula||(snapshot.value===undefined||snapshot.value===null||snapshot.value===''?'(빈 셀)':typeof snapshot.value==='string'?snapshot.value:JSON.stringify(snapshot.value))

/** Cells a range covers, used to warn before the server refuses the request. */
export function rangeCellCount(range:string){
  const parts=range.split(':')
  const parse=(value:string)=>{
    const match=/^([A-Z]+)(\d+)$/.exec(value.trim().toUpperCase())
    if(!match)return undefined
    let column=0
    for(const letter of match[1])column=column*26+letter.charCodeAt(0)-64
    return {row:Number(match[2]),column}
  }
  const start=parse(parts[0]),end=parse(parts[1]??parts[0])
  if(!start||!end)return 0
  return (Math.abs(end.row-start.row)+1)*(Math.abs(end.column-start.column)+1)
}

export function AIPanel({workbookId,workbookName,sheetId,sheetName,selectionRange,baseVersion,initialMode='agent',initialRequest='',onClose,onExecuted}:Props){
  const client=useQueryClient()
  const [mode,setMode]=useState<Mode>(initialMode)
  const [request,setRequest]=useState(initialRequest)
  const [run,setRun]=useState<AgentRun>()
  const [conversationId,setConversationId]=useState<string>()
  const [pendingMessage,setPendingMessage]=useState('')
  const [guideOpen,setGuideOpen]=useState(false)
  const [historyOpen,setHistoryOpen]=useState(false)
  const [promptOpen,setPromptOpen]=useState(false)
  const [elapsed,setElapsed]=useState(0)
  const [sessionError,setSessionError]=useState('')
  const requestRef=useRef<HTMLTextAreaElement>(null)
  const chatRef=useRef<HTMLDivElement>(null)
  const restoredWorkbook=useRef('')
  const conversationStorageKey=`kanpic:agent-conversation:${workbookId}`
  const config=useQuery({queryKey:['ai-config'],queryFn:()=>api<AIConfig>('/api/v1/ai/config')})
  const context=useQuery({queryKey:['agent-context',workbookId,sheetId,selectionRange],queryFn:()=>api<AgentContext>(`/api/v1/workbooks/${workbookId}/agent/context?sheet_id=${encodeURIComponent(sheetId)}&selection=${encodeURIComponent(selectionRange)}`),enabled:Boolean(config.data?.enabled)})
  const conversations=useQuery({queryKey:['agent-conversations',workbookId],queryFn:()=>api<{items:AgentConversation[]}>(`/api/v1/workbooks/${workbookId}/agent/conversations?limit=20`),enabled:Boolean(config.data?.enabled)})
  const refresh=async()=>Promise.all([client.invalidateQueries({queryKey:['agent-conversations',workbookId]}),client.invalidateQueries({queryKey:['agent-runs',workbookId]}),client.invalidateQueries({queryKey:['ai-actions',workbookId]})])
  const selectedCells=rangeCellCount(selectionRange)
  const overLimit=Boolean(config.data&&selectedCells>config.data.max_input_cells)
  const active=MODES.find(item=>item.id===mode)??MODES[0]
  const plan=useMutation({
    mutationFn:()=>{const key=newIdempotencyKey();return api<AgentRun>(`/api/v1/workbooks/${workbookId}/agent/messages`,{method:'POST',headers:{'Idempotency-Key':key},body:JSON.stringify({sheet_id:sheetId,selection:selectionRange,message:request.trim(),mode:mode==='agent'?undefined:mode,conversation_id:conversationId,base_version:baseVersion,idempotency_key:key,client_id:collaborationClientId()})})},
    onMutate:()=>setPendingMessage(request.trim()),
    onSuccess:async item=>{setRun(item);setConversationId(item.conversation_id);window.localStorage.setItem(conversationStorageKey,item.conversation_id);setRequest('');setMode('agent');setPendingMessage('');setSessionError('');await refresh()},
    onError:()=>setPendingMessage(''),
  })
  // The prompt preview is fetched on demand so opening the panel never reads
  // cells that nobody asked to send.
  const preview=useQuery({
    queryKey:['ai-prompt-preview',workbookId,sheetId,selectionRange,mode,request.trim()],
    queryFn:()=>api<AIPromptPreview>('/api/v1/ai/prompt:preview',{method:'POST',body:JSON.stringify({workbook_id:workbookId,sheet_id:sheetId,range:selectionRange,request:request.trim()||'(요청 없음)',mode,base_version:baseVersion})}),
    enabled:promptOpen&&!overLimit,
    staleTime:15_000,
  })
  const approve=useMutation({
    mutationFn:(item:AgentRun)=>api<AgentExecutionResult>(`/api/v1/agent/runs/${item.id}/approve`,{method:'POST',body:JSON.stringify({idempotency_key:newIdempotencyKey(),client_id:collaborationClientId(),expected_revision:item.action.revision})}),
    onSuccess:async result=>{setRun(result.run);if(result.operation)onExecuted({action:result.run.action,operation:result.operation,changes:result.changes});await Promise.all([refresh(),client.invalidateQueries({queryKey:['workbook',workbookId]}),client.invalidateQueries({queryKey:['charts',workbookId]})])},
  })
  const undo=useMutation({
    mutationFn:(item:AgentRun)=>api<AgentExecutionResult>(`/api/v1/changesets/${item.change_set_id}/rollback`,{method:'POST',body:JSON.stringify({idempotency_key:newIdempotencyKey(),client_id:collaborationClientId(),expected_revision:item.action.revision})}),
    onSuccess:async result=>{setRun(result.run);if(result.operation)onExecuted({action:result.run.action,operation:result.operation,changes:result.changes});await Promise.all([refresh(),client.invalidateQueries({queryKey:['workbook',workbookId]}),client.invalidateQueries({queryKey:['charts',workbookId]})])},
  })
  const cancel=useMutation({
    mutationFn:(item:AgentRun)=>api<AgentRun>(`/api/v1/agent/runs/${item.id}/cancel`,{method:'POST',body:JSON.stringify({idempotency_key:newIdempotencyKey(),client_id:collaborationClientId(),expected_revision:item.action.revision})}),
    onSuccess:async item=>{setRun(item);await refresh()},
  })
  const actionPending=approve.isPending||undo.isPending||cancel.isPending
  // A gateway round trip takes seconds, so the wait is counted out loud.
  useEffect(()=>{
    if(!plan.isPending){setElapsed(0);return}
    const started=Date.now()
    const timer=window.setInterval(()=>setElapsed(Math.round((Date.now()-started)/1000)),500)
    return()=>window.clearInterval(timer)
  },[plan.isPending])
  const loadRun=async(id:string)=>{
    setSessionError('')
    try{
      const item=await api<AgentRun>(`/api/v1/agent/runs/${id}`)
      setRun(item)
      setConversationId(item.conversation_id)
      setMode('agent')
      setHistoryOpen(false)
      window.localStorage.setItem(conversationStorageKey,item.conversation_id)
    }catch(error){
      window.localStorage.removeItem(conversationStorageKey)
      setSessionError(error instanceof Error?error.message:'대화를 불러오지 못했습니다.')
    }
  }
  useEffect(()=>{
    if(!conversations.data||restoredWorkbook.current===workbookId)return
    restoredWorkbook.current=workbookId
    const saved=window.localStorage.getItem(conversationStorageKey)
    const item=conversations.data.items.find(candidate=>candidate.id===saved)
    if(item?.latest_run_id){void loadRun(item.latest_run_id);return}
    if(saved)window.localStorage.removeItem(conversationStorageKey)
    setRun(undefined)
    setConversationId(undefined)
  },[conversationStorageKey,conversations.data,workbookId])
  useEffect(()=>{
    const element=chatRef.current
    if(element)window.requestAnimationFrame(()=>{element.scrollTop=element.scrollHeight})
  },[pendingMessage,run?.id,run?.messages.length,run?.updated_at])
  const canSend=Boolean(request.trim()&&config.data?.enabled&&!overLimit&&!plan.isPending&&!actionPending)
  const send=()=>{if(canSend)plan.mutate()}
  const submit=(event:FormEvent)=>{event.preventDefault();send()}
  const startConversation=()=>{window.localStorage.removeItem(conversationStorageKey);restoredWorkbook.current=workbookId;setRun(undefined);setConversationId(undefined);setRequest('');setMode('agent');setPromptOpen(false);setPendingMessage('');setSessionError('');setHistoryOpen(false);window.requestAnimationFrame(()=>requestRef.current?.focus())}
  const chooseFollowUp=(suggestion:string)=>{setMode('agent');setRequest(suggestion);window.requestAnimationFrame(()=>requestRef.current?.focus())}
  const editLastRequest=()=>{if(!run)return;setMode(run.action.mode);setRequest(run.action.request);window.requestAnimationFrame(()=>requestRef.current?.focus())}
  const error=plan.error||approve.error||undo.error||cancel.error
  return <aside className="ai-panel managed-ai-panel" aria-label="AI 도우미 패널">
    <div className="ai-panel-head"><span><Bot/><span><strong>AI 도우미</strong><small>멀티턴 Workbook Agent</small></span></span><div><button className={`ai-head-help${historyOpen?' active':''}`} aria-label="AI 대화 목록 열기" aria-expanded={historyOpen} title="대화 목록" onClick={()=>setHistoryOpen(current=>!current)}><MessagesSquare/></button><button className="ai-head-help" aria-label="새 AI 대화 시작" title="새 대화" disabled={plan.isPending||actionPending} onClick={startConversation}><MessageSquarePlus/></button><button className="ai-head-help" aria-label="사용 가이드 열기" aria-expanded={guideOpen} title="사용 가이드" onClick={()=>setGuideOpen(current=>!current)}><HelpCircle/></button><button onClick={onClose} aria-label="Workbook Agent 닫기">×</button></div></div>
    {config.data?.enabled&&<section className={`ai-chat-scope${overLimit?' over':''}`} aria-label="현재 채팅 범위"><div><strong>{run?.context.active_sheet.name||context.data?.active_sheet?.name||sheetName||sheetId}</strong><span>{selectionRange}</span></div><small>{selectedCells.toLocaleString()}셀 · {config.data.model}</small></section>}
    <div className="ai-chat-scroll" ref={chatRef}>
      {historyOpen&&<ConversationList items={conversations.data?.items??[]} activeId={conversationId} loading={conversations.isLoading} disabled={plan.isPending||actionPending} onOpen={item=>item.latest_run_id&&void loadRun(item.latest_run_id)}/>}
      {guideOpen&&<UsageGuide config={config.data} onClose={()=>setGuideOpen(false)}/>}
      {config.isLoading&&<div className="ai-state"><RefreshCw className="spin"/>AI 설정을 확인하는 중…</div>}
      {config.isError&&<div className="ai-state error"><AlertTriangle/>AI 설정을 불러오지 못했습니다.</div>}
      {config.data&&!config.data.enabled&&<div className="ai-disabled"><Bot/><strong>AI가 아직 비활성화되어 있습니다.</strong><p>관리자가 사내 LLM Gateway를 검증하고 활성화하면 선택 범위 기반 계획을 사용할 수 있습니다.</p><a href="/admin?tab=settings">관리자 AI 설정 열기</a><button className="ai-inline-link" onClick={()=>setGuideOpen(true)}>사용 가이드 먼저 보기</button></div>}
      {config.data?.enabled&&<>
        {overLimit&&<div className="ai-error" role="alert"><AlertTriangle/>한 번에 최대 {config.data.max_input_cells.toLocaleString()}셀까지 보낼 수 있습니다. 범위를 좁혀 주세요.</div>}
        {run&&<AgentTimeline run={run}/>}
        {pendingMessage&&<section className="agent-timeline pending" aria-live="polite"><article className="user"><strong>사용자</strong><p>{pendingMessage}</p></article><article className="assistant thinking"><RefreshCw className="spin"/><p>{run?.action.status==='planned'?'이전 승인 대기 계획을 대체할 새 계획을 만드는 중':'대화와 현재 워크북 상태를 읽고 계획하는 중'} · {elapsed}초</p></article></section>}
        {!run&&!pendingMessage&&<div className="ai-chat-welcome"><div className="ai-glyph"><Sparkles/></div><h3>무엇을 도와드릴까요?</h3><p>선택한 범위를 기준으로 자연스럽게 요청하세요. 작업 유형은 Agent가 판단하고, 변경은 검토 후에만 적용됩니다.</p><div><span>예시</span><p>“이 범위의 핵심 추세를 설명해줘”</p><p>“선택한 부분을 막대 차트로 만들어줘”</p><p>“방금 만든 차트를 선 차트로 바꿔줘”</p></div></div>}
        {run&&<section className="ai-chat-result" aria-label="현재 Agent 작업"><AgentPlanView run={run}/><ActionPreview action={run.action} onApprove={()=>approve.mutate(run)} onCancel={()=>cancel.mutate(run)} onUndo={()=>undo.mutate(run)} pending={actionPending||plan.isPending} onRetry={editLastRequest}/></section>}
          {run&&(run.suggested_follow_ups?.length??0)>0&&<section className="ai-followup-suggestions" aria-label="추천 후속 요청"><strong><Sparkles/> 다음 작업 제안</strong><div>{run.suggested_follow_ups?.map(suggestion=><button type="button" key={suggestion} disabled={plan.isPending||actionPending} onClick={()=>chooseFollowUp(suggestion)}>{suggestion}</button>)}</div></section>}
          {promptOpen&&<PromptDisclosure open={promptOpen} onToggle={()=>setPromptOpen(false)} preview={preview.data} loading={preview.isFetching} error={preview.isError} disabled={overLimit}/>}
        {(error||sessionError)&&<div className="ai-error" role="alert"><AlertTriangle/>{sessionError||(error instanceof Error?error.message:'AI 작업을 처리하지 못했습니다.')}</div>}
      </>}
    </div>
    {config.data?.enabled&&<form className="ai-chat-composer" onSubmit={submit}>
      <div className="ai-chat-options"><label><Bot/><select aria-label="AI 작업 방식" value={mode} title={active.hint} onChange={event=>setMode(event.target.value as Mode)}>{MODES.map(item=><option value={item.id} key={item.id}>{modeLabel[item.id]}</option>)}</select></label><button type="button" aria-label="모델에 보내는 내용 보기" aria-expanded={promptOpen} onClick={()=>setPromptOpen(current=>!current)}><Terminal/></button></div>
      <div className="ai-chat-input"><textarea ref={requestRef} aria-label="AI 요청" value={request} maxLength={4000} rows={2} onChange={event=>setRequest(event.target.value)} onKeyDown={event=>{if(event.nativeEvent.isComposing)return;if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();send()}}} placeholder={run?'이어서 요청하세요…':active.examples[0]}/><button className="primary" aria-label="AI 메시지 보내기" title="메시지 보내기" disabled={!canSend}>{plan.isPending?<RefreshCw className="spin"/>:<Send/>}</button></div>
      <div className="ai-chat-composer-foot"><small>{run?'같은 대화에 이어서 전송':'새 대화 시작'} · Enter 전송 · Shift+Enter 줄바꿈</small><span>{request.length.toLocaleString()} / 4,000</span></div>
      <small className="ai-chat-safety"><Eye/> 변경 작업은 계획을 확인하고 승인해야 적용됩니다.</small>
    </form>}
  </aside>
}

function ConversationList({items,activeId,loading,disabled,onOpen}:{items:AgentConversation[];activeId?:string;loading:boolean;disabled:boolean;onOpen:(item:AgentConversation)=>void}){
  return <section className="ai-conversation-list" aria-label="AI 대화 목록"><header><MessagesSquare/><div><strong>대화 목록</strong><small>이 워크북에서 이어서 작업할 대화</small></div></header>{loading&&<p><RefreshCw className="spin"/>대화를 불러오는 중…</p>}{!loading&&items.length===0&&<p>아직 저장된 대화가 없습니다.</p>}{items.map(item=><button type="button" key={item.id} className={item.id===activeId?'active':''} disabled={!item.latest_run_id||disabled} onClick={()=>onOpen(item)} aria-label={`대화 열기: ${item.title}`}><span>{item.title}</span><small>{item.message_count>0?`${Math.ceil(item.message_count/2)}턴 · `:''}{item.latest_state?agentStateLabel(item.latest_state):'대화 시작됨'} · {new Date(item.updated_at).toLocaleString('ko-KR')}</small></button>)}</section>
}

/** Shows the exact request that would leave the building, on demand. */
function PromptDisclosure({open,onToggle,preview,loading,error,disabled}:{open:boolean;onToggle:()=>void;preview?:AIPromptPreview;loading:boolean;error:boolean;disabled:boolean}){
  const copy=async(text:string)=>{try{await navigator.clipboard?.writeText(text)}catch{window.prompt('복사할 내용',text)}}
  return <section className="ai-prompt-view">
    <button className="ai-disclosure" aria-expanded={open} onClick={onToggle}><Terminal/><span>모델에 보내는 내용 보기</span><ChevronDown className={open?'open':''}/></button>
    {open&&<div className="ai-disclosure-body">
      {disabled&&<p className="ai-note">선택 범위가 상한을 넘어 미리 볼 수 없습니다.</p>}
      {!disabled&&loading&&<p className="ai-note"><RefreshCw className="spin"/> 보낼 내용을 계산하는 중…</p>}
      {!disabled&&error&&<p className="ai-note error">보낼 내용을 불러오지 못했습니다.</p>}
      {!disabled&&preview&&<>
        <div className="ai-prompt-meta"><span>모델 {preview.model}</span><span>셀 {preview.cell_count.toLocaleString()}개</span><span>temperature {preview.temperature}</span><span>max_tokens {preview.max_tokens.toLocaleString()}</span>{preview.context_window?<span>컨텍스트 {preview.context_window.toLocaleString()}</span>:null}{preview.estimated_prompt_tokens?<span>입력 추정 {preview.estimated_prompt_tokens.toLocaleString()}</span>:null}</div>
        <p className="ai-note">{preview.context_window?`모델의 컨텍스트 ${preview.context_window.toLocaleString()}토큰에서 입력 추정치를 빼고 응답 상한을 자동으로 잡았습니다.`:'게이트웨이가 컨텍스트 길이를 알려 주지 않아 보수적인 상한을 사용합니다. 응답이 잘리면 자동으로 더 크게 다시 요청합니다.'}</p>
        <div className="ai-prompt-block"><header>시스템 프롬프트<button aria-label="시스템 프롬프트 복사" onClick={()=>void copy(preview.system_prompt)}><Clipboard/></button></header><pre>{preview.system_prompt}</pre></div>
        <div className="ai-prompt-block"><header>전달되는 데이터<button aria-label="전달 데이터 복사" onClick={()=>void copy(preview.user_content)}><Clipboard/></button></header><pre>{preview.user_content}</pre></div>
        <p className="ai-note">값이 비어 있는 셀과 선택 범위 밖의 데이터는 전송하지 않습니다.</p>
      </>}
    </div>}
  </section>
}

/** The panel is useless without knowing what it will and will not do. */
function UsageGuide({config,onClose}:{config?:AIConfig;onClose:()=>void}){
  return <section className="ai-guide" aria-label="AI 사용 가이드">
    <header><strong><HelpCircle/> 사용 가이드</strong><button onClick={onClose} aria-label="사용 가이드 닫기">×</button></header>
    <ol>
      <li><strong>범위를 선택합니다.</strong> 시트에서 분석하거나 고칠 범위를 먼저 고릅니다. 선택한 범위 안의 값이 있는 셀만 전송됩니다.</li>
      <li><strong>채팅으로 요청합니다.</strong> 기본 자동 에이전트가 의도를 판단합니다. 꼭 필요한 경우에만 입력창 위 선택 메뉴에서 작업 방식을 고정합니다.</li>
      <li><strong>계획을 검토합니다.</strong> 셀 Diff와 실행할 도구, 위험도, 검증 단계를 적용 전에 보여 줍니다.</li>
      <li><strong>승인해야 적용됩니다.</strong> 승인한 ChangeSet만 반영되고 <em>Undo</em>로 수식·차트·Agent 생성 시트를 함께 되돌릴 수 있습니다.</li>
    </ol>
    <dl>
      <dt>전송되는 것</dt><dd>요청, 워크북·시트 메타데이터, 선택 범위의 주소·값·수식과 의미 프로필{config?`, 최대 ${config.max_input_cells.toLocaleString()}셀`:''}</dd>
      <dt>전송되지 않는 것</dt><dd>다른 시트와 범위 밖의 셀 값, 공유 대상, 댓글, 계정 정보</dd>
      <dt>모델이 할 수 없는 것</dt><dd>승인 없는 변경, 외부 링크·스크립트 실행, 범위 밖 셀 수정{config?`, 한 번에 ${config.max_changes.toLocaleString()}셀 초과 변경`:''}</dd>
    </dl>
    <p className="ai-note">요청은 구체적일수록 좋습니다. “정리해줘”보다 “앞뒤 공백을 없애고 날짜를 YYYY-MM-DD로 통일해줘”처럼 규칙을 적어 주세요.</p>
  </section>
}

function AgentTimeline({run}:{run:AgentRun}){
  return <section className="agent-timeline" aria-label="Agent 대화">
    {run.messages.map(message=><article className={message.role} key={message.id}><strong>{message.role==='user'?'사용자':'Agent'}</strong><p>{message.content}</p></article>)}
  </section>
}

function AgentPlanView({run}:{run:AgentRun}){
  return <section className="agent-plan" aria-label="Agent 실행 계획">
    <header><ListChecks/><div><strong>{run.plan.goal}</strong><small>{riskLabel(run.risk)} · {agentStateLabel(run.state)}</small></div></header>
    <ol>{run.plan.steps.map(step=><li className={step.status} key={step.id||step.position}><span>{step.status==='completed'?<Check/>:step.status==='failed'?<AlertTriangle/>:step.status==='cancelled'?<XCircle/>:<Clock3/>}</span><div><strong>{step.description}</strong><small><code>{step.tool}</code> · {stepStatusLabel(step.status)}</small></div></li>)}</ol>
    {run.action.tool_calls?.length>0&&<div className="agent-tools"><Wrench/><span>{run.action.tool_calls.map(tool=>`${toolLabel(tool.name)} (${stepStatusLabel(tool.status)})`).join(', ')}</span></div>}
  </section>
}

function ActionPreview({action,onApprove,onCancel,onUndo,pending,onRetry}:{action:AIAction;onApprove:()=>void;onCancel:()=>void;onUndo:()=>void;pending:boolean;onRetry:()=>void}){
  const readOnly=action.mode==='explain'||action.mode==='summarize'||action.mode==='anomaly'
  const findings=action.findings??[]
  // Usage arrives with a fresh plan and is kept in the audit event afterwards.
  const usage=action.usage??(action.events??[]).map(event=>event.payload?.usage as AIUsage|undefined).find(Boolean)
  return <div className="ai-action-preview">
    <div className="ai-action-title"><div><span>{modeLabel[action.mode]}</span><strong>{action.summary}</strong></div><em className={`ai-status ${action.status}`}>{statusLabel(action.status,action.mode)}</em></div>
    <p className="ai-action-request">요청: {action.request}</p>
    <p>{action.explanation}</p>
    {findings.length>0&&<div className="ai-finding-list">{findings.map((finding,index)=><article className={finding.severity} key={`${finding.address||'range'}:${index}`}><header><span>{finding.address||'전체 범위'}</span><em>{severityLabel(finding.severity)}</em></header><strong>{finding.title}</strong><p>{finding.description}</p>{finding.address&&<code>현재: {snapshotText(finding.cell??{})}</code>}</article>)}</div>}
    {action.changes.length>0&&<div className="ai-change-list">{action.changes.map(change=><article key={`${change.row}:${change.column}`}><strong>{change.address}</strong><div><small>현재</small><code>{action.mode==='format'?JSON.stringify(change.before.style??{}):snapshotText(change.before)}</code></div><span>→</span><div><small>제안</small><code>{action.mode==='format'?JSON.stringify(change.after.style??{}):snapshotText(change.after)}</code></div></article>)}</div>}
    {action.tool_calls?.map((tool,index)=><article className="agent-tool-preview" key={tool.id||tool.idempotency_key||`${tool.name}:${index}`}><header><Wrench/><strong>{toolLabel(tool.name)}</strong><em>{riskLabel(tool.risk)}</em></header>{toolSummary(tool)&&<p className="agent-tool-summary">{toolSummary(tool)}</p>}<pre>{JSON.stringify(tool.arguments,null,2)}</pre></article>)}
    <div className="ai-plan-meta"><span>기준 버전 v{action.base_version}</span>{action.changes.length>0&&<span>{action.changes.length}셀 변경</span>}{findings.length>0&&<span>{findings.length}개 발견</span>}<span>{action.model}</span>{usage?.prompt_tokens?<span>입력 {usage.prompt_tokens.toLocaleString()}토큰</span>:null}{usage?.completion_tokens?<span>응답 {usage.completion_tokens.toLocaleString()}토큰</span>:null}{usage&&(usage.attempts??1)>1?<span>재시도 {(usage.attempts??1)-1}회</span>:null}</div>
    {action.status==='failed'&&<div className="ai-error"><AlertTriangle/>{action.error_message||'작업이 실패했습니다.'}</div>}
    {action.status==='planned'&&!readOnly&&<div className="ai-approval"><p><ShieldCheck/> 위 변경만 하나의 ChangeSet으로 적용되고 전체 Undo할 수 있습니다.</p><div><button onClick={onCancel} disabled={pending}><XCircle/> 취소</button><button className="primary" aria-label="검토한 계획 승인 및 변경 적용" onClick={onApprove} disabled={pending}><Check/> 변경 적용</button></div></div>}
    {(action.status==='completed'||(action.status==='planned'&&readOnly))&&<div className="ai-explanation-done"><Check/> 읽기 전용 분석이 완료됐으며 워크북 변경은 없습니다.</div>}
    {(action.status==='applying'||action.status==='undoing')&&<div className="ai-state"><RefreshCw className="spin"/>서버 작업을 마무리하는 중…</div>}
    {action.status==='applied'&&<div className="ai-applied"><Check/><div><strong>승인한 변경이 적용되었습니다.</strong><small>{action.operation?.server_version?`서버 버전 v${action.operation.server_version}`:'현재 워크북에 실시간 반영됨'}</small></div><button onClick={onUndo} disabled={pending}><RotateCcw/> Undo</button></div>}
    {action.status==='undone'&&<div className="ai-explanation-done"><RotateCcw/> AI 변경을 새 서버 버전으로 되돌렸습니다.</div>}
    <div className="ai-action-actions"><button onClick={onRetry} disabled={pending}>이 요청 수정</button></div>
  </div>
}

function statusLabel(status:AIAction['status'],mode:AIAction['mode']){if(status==='completed')return mode==='summarize'?'요약 완료':mode==='anomaly'?'탐지 완료':'설명 완료';return {planned:'승인 대기',applying:'적용 중',applied:'적용됨',undoing:'취소 중',undone:'취소됨',failed:'실패',cancelled:'취소됨'}[status]}
function severityLabel(severity:'info'|'warning'|'critical'){return {info:'정보',warning:'주의',critical:'심각'}[severity]}
function agentStateLabel(state:AgentRun['state']){return {THINKING:'생각 중',READING_WORKBOOK:'워크북 읽는 중',PLANNING:'계획 중',WAITING_APPROVAL:'승인 대기',EXECUTING:'실행 중',VALIDATING:'검증 중',COMPLETED:'완료',FAILED:'실패',CANCELLED:'취소됨'}[state]}
function stepStatusLabel(status:string){return {completed:'완료',waiting_approval:'승인 대기',pending:'대기',executing:'진행',validating:'검증',failed:'실패',cancelled:'취소',planned:'계획됨'}[status]||status}
function toolLabel(name:string){return {create_chart:'차트 만들기',update_chart:'차트 바꾸기',create_report_sheet:'보고서 시트 만들기',create_conditional_format:'조건부 서식 만들기',create_pivot:'피벗 요약 만들기',create_data_validation:'입력 규칙 만들기',create_filter_view:'필터 보기 만들기',sort_range:'범위 정렬하기'}[name]||name}
// 승인 화면에 도구 이름과 JSON 만 있었다. 무엇이 어디에 생기는지 한 줄로
// 먼저 말해 주어야 사람이 승인할지 판단할 수 있다. JSON 은 그대로 남긴다 —
// 한 줄로 줄이면서 빠뜨린 것을 확인할 곳이 있어야 한다.
function toolSummary(tool:AgentToolCall){
  const argument=(tool.arguments||{}) as Record<string,any>
  const field=(item:any)=>String(item?.name||`열 ${item?.column??'?'}`)
  if(tool.name==='create_pivot'){
    const rows=(argument.rows||[]).map(field).join(', ')
    const values=(argument.values||[]).map((item:any)=>`${field(item)} ${aggregationLabel(String(item?.aggregation||''))}`).join(', ')
    return `${argument.source_range||''}${rows?` · ${rows}별`:''}${values?` · ${values}`:''}`
  }
  if(tool.name==='sort_range'){
    const keys=(argument.keys||[]).map((item:any)=>`${columnName(Number(item?.column))}열 ${item?.direction==='desc'?'내림차순':'오름차순'}`).join(', ')
    const header=argument.header_rows===0?'머리글 없음':`머리글 ${argument.header_rows??1}줄`
    return `${argument.range||''}${keys?` · ${keys}`:''} · ${header}`
  }
  if(tool.name==='create_filter_view'){
    const conditions=(argument.criteria||[]).map((item:any)=>`${columnName(Number(item?.column))}열 ${filterOperatorLabel(String(item?.operator||''))}${item?.value!==undefined?` ${JSON.stringify(item.value)}`:''}${Array.isArray(item?.values)?` ${item.values.length}개 값`:''}`).join(', ')
    return `${argument.range||''}${conditions?` · ${conditions}`:''}`
  }
  if(tool.name==='create_data_validation'){
    const kind=validationLabel(String(argument.rule_type||''))
    const detail=argument.rule_type==='list'?(argument.options||[]).map((item:any)=>String(item?.label??JSON.stringify(item?.value))).join(', '):argument.rule_type==='list_range'?String(argument.source_range||''):argument.rule_type==='custom_formula'?String(argument.formula||''):[argument.operator,argument.value,argument.value2].filter(v=>v!==undefined&&v!==null&&v!=='').join(' ')
    return `${argument.range||''} · ${kind}${detail?` · ${detail}`:''}${argument.reject_input?' · 어긋나면 입력 거부':''}`
  }
  if(tool.name==='create_conditional_format')return `${argument.range||''}${argument.rule_type?` · ${String(argument.rule_type)}`:''}`
  if(tool.name==='create_chart'||tool.name==='update_chart')return [argument.title,argument.source_range,argument.type].filter(Boolean).join(' · ')
  if(tool.name==='create_report_sheet')return `${argument.name||'새 시트'} · ${(argument.cells||[]).length}셀${argument.chart?' · 차트 포함':''}`
  return ''
}
function columnName(column:number){let name='',current=column
  while(current>0){const remainder=(current-1)%26;name=String.fromCharCode(65+remainder)+name;current=Math.floor((current-1)/26)}
  return name||'?'}
function filterOperatorLabel(name:string){return {values:'값 목록',equals:'같음',not_equals:'다름',contains:'포함',not_contains:'미포함',starts_with:'시작',ends_with:'끝남',greater_than:'초과',greater_or_equal:'이상',less_than:'미만',less_or_equal:'이하',is_blank:'빈 값',is_not_blank:'값 있음',background_color:'배경색',text_color:'글자색'}[name]||name}
function validationLabel(name:string){return {list:'목록에서 고르기',list_range:'범위의 값에서 고르기',checkbox:'체크박스',number:'숫자 조건',date:'날짜 조건',custom_formula:'수식 조건'}[name]||name}
function aggregationLabel(name:string){return {sum:'합계',average:'평균',count:'숫자 개수',counta:'개수',countunique:'고유 개수',min:'최소',max:'최대',median:'중앙값',product:'곱',stdev:'표본 표준편차',stdevp:'표준편차',var:'표본 분산',varp:'분산'}[name]||name}
function riskLabel(risk:AIAction['risk']){return {READ:'읽기',LOW:'낮은 위험',MEDIUM:'중간 위험',HIGH:'높은 위험',CRITICAL:'매우 높은 위험'}[risk]||risk}
