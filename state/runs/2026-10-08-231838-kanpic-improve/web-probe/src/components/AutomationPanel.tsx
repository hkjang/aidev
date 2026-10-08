import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Check, Clock3, Eye, Pencil, Play, Plus, RefreshCw, RotateCcw, Save, ShieldAlert, ShieldCheck, Trash2, Workflow } from 'lucide-react'
import { FormEvent, useEffect, useRef, useState } from 'react'
import { ApiError, api, newIdempotencyKey } from '../lib/api'
import { collaborationClientId } from '../lib/client'
import type { Automation, AutomationAction, AutomationExecutionResult, AutomationOverview, AutomationPreview, AutomationRun, AutomationTrigger, Sheet } from '../types'
import './AutomationPanel.css'

type Props={
  workbookId:string
  workbookVersion:number
  sheets:Sheet[]
  activeSheetId:string
  selectionRange:string
  prepareExecution:()=>Promise<number>
  onClose:()=>void
  onExecuted:(result:AutomationExecutionResult)=>void
}
type ValueKind='string'|'number'|'boolean'

const sheetLabel=(sheets:Sheet[],id?:string)=>sheets.find(sheet=>sheet.id===id)?.name??'알 수 없는 시트'
const triggerLabel=(trigger:AutomationTrigger,sheets:Sheet[])=>trigger.type==='manual'?'수동 실행':trigger.type==='schedule'?`일정 · ${trigger.cron} · ${trigger.timezone}`:trigger.type==='webhook'?'인바운드 웹훅':`셀 변경 · ${sheetLabel(sheets,trigger.sheet_id)}!${trigger.range}`
const actionLabel=(action:AutomationAction)=>action.type==='set_formula'?'수식 설정':action.type==='set_value'?'값 설정':'내용 지우기'
const snapshotText=(snapshot:{value?:unknown;formula?:string})=>snapshot.formula||(snapshot.value===undefined||snapshot.value===null||snapshot.value===''?'(빈 셀)':typeof snapshot.value==='string'?snapshot.value:JSON.stringify(snapshot.value))

export function AutomationPanel({workbookId,workbookVersion,sheets,activeSheetId,selectionRange,prepareExecution,onClose,onExecuted}:Props){
  const client=useQueryClient()
  const [editing,setEditing]=useState<Automation|null|undefined>()
  const [previewItem,setPreviewItem]=useState<Automation>()
  const [preview,setPreview]=useState<AutomationPreview>()
  const [pending,setPending]=useState('')
  const [error,setError]=useState('')
  const runKeys=useRef<Record<string,string>>({})
  const undoKeys=useRef<Record<string,string>>({})
  const automations=useQuery({queryKey:['automations',workbookId],queryFn:()=>api<AutomationOverview>(`/api/v1/workbooks/${workbookId}/automations`),refetchInterval:15_000})
  // Automation execution is an admin switch that defaults to off. Without this
  // the panel saves and previews normally while every trigger stays silent.
  const executionOff=automations.isSuccess&&automations.data.execution_enabled===false
  const runs=useQuery({queryKey:['automation-runs',previewItem?.id],queryFn:()=>api<{items:AutomationRun[]}>(`/api/v1/automations/${previewItem!.id}/runs?limit=12`),enabled:Boolean(previewItem),refetchInterval:15_000})
  const currentPreviewItem=automations.data?.items.find(item=>item.id===previewItem?.id)
  const previewMissing=Boolean(preview&&automations.isSuccess&&!currentPreviewItem)
  const previewStale=Boolean(preview&&(workbookVersion!==preview.base_version||previewMissing||(currentPreviewItem?.revision??previewItem?.revision??0)!==preview.automation_revision))
  const previewIdentity=preview?`${preview.automation_id}:${preview.automation_revision}:${preview.base_version}`:''
  const hasPendingRunReplay=Boolean(previewIdentity&&runKeys.current[previewIdentity])
  const refresh=async(id?:string)=>{await client.invalidateQueries({queryKey:['automations',workbookId]});if(id)await client.invalidateQueries({queryKey:['automation-runs',id]})}
  const test=async(item:Automation)=>{
    setPending(`test:${item.id}`);setError('');setPreviewItem(item)
    try{
      await prepareExecution()
      const nextPreview=await api<AutomationPreview>(`/api/v1/automations/${item.id}:test`,{method:'POST',body:'{}'})
      await refresh()
      setPreview(nextPreview)
    }catch(reason){setPreview(undefined);setError(reason instanceof Error?reason.message:'자동화를 검증하지 못했습니다.')}
    finally{setPending('')}
  }
  const run=async(item:Automation,replay=false)=>{
    if(!preview||preview.automation_id!==item.id||(!replay&&previewStale)||preview.changes.length===0){setError('최신 변경 내용을 다시 검증한 뒤 실행하세요.');return}
    setPending(`run:${item.id}`);setError('')
    try{
      const currentVersion=await prepareExecution()
      if(!replay&&currentVersion!==preview.base_version){setError('저장된 셀 변경으로 워크북 버전이 바뀌었습니다. 다시 검증하세요.');return}
      const identity=`${item.id}:${preview.automation_revision}:${preview.base_version}`
      const idempotencyKey=runKeys.current[identity]??(runKeys.current[identity]=newIdempotencyKey())
      const result=await api<AutomationExecutionResult>(`/api/v1/automations/${item.id}:run`,{method:'POST',body:JSON.stringify({expected_revision:preview.automation_revision,expected_base_version:preview.base_version,idempotency_key:idempotencyKey,client_id:collaborationClientId()})})
      delete runKeys.current[identity]
      onExecuted(result);setPreview(undefined)
    }catch(reason){if(reason instanceof ApiError&&reason.status>=400&&reason.status<500&&reason.status!==408)delete runKeys.current[`${item.id}:${preview.automation_revision}:${preview.base_version}`];setError(reason instanceof Error?reason.message:'자동화를 실행하지 못했습니다.')}
    finally{await refresh(item.id).catch(()=>{});setPending('')}
  }
  const undo=async(item:AutomationRun)=>{
    const idempotencyKey=undoKeys.current[item.id]??(undoKeys.current[item.id]=newIdempotencyKey())
    setPending(`undo:${item.id}`);setError('')
    try{
      await prepareExecution()
      const result=await api<AutomationExecutionResult>(`/api/v1/automation-runs/${item.id}:undo`,{method:'POST',body:JSON.stringify({idempotency_key:idempotencyKey,client_id:collaborationClientId()})})
      delete undoKeys.current[item.id]
      onExecuted(result)
    }catch(reason){setError(reason instanceof Error?reason.message:'자동화 실행을 되돌리지 못했습니다.')}
    finally{await refresh(item.automation_id).catch(()=>{});setPending('')}
  }
  const remove=async(item:Automation)=>{if(!confirm(`'${item.name}' 자동화를 삭제할까요?`))return;setPending(`delete:${item.id}`);setError('');try{await api(`/api/v1/automations/${item.id}?expected_revision=${item.revision}`,{method:'DELETE'});if(previewItem?.id===item.id){setPreviewItem(undefined);setPreview(undefined)}await refresh()}catch(reason){setError(reason instanceof Error?reason.message:'자동화를 삭제하지 못했습니다.')}finally{setPending('')}}
  const saved=async(item:Automation)=>{setEditing(undefined);await refresh(item.id);await test(item)}
  return <aside className="automation-panel" aria-label="자동화 패널">
    <header><span><Workflow/> 자동화</span><button aria-label="자동화 패널 닫기" onClick={onClose}>×</button></header>
    <div className="automation-scroll">
      <div className="automation-intro"><div><strong>검증 가능한 워크북 자동화</strong><p>실행 전에 최신 셀 기준 변경 내용을 확인하고, 실행 후에는 서버 버전으로 Undo할 수 있습니다.</p></div><button className="primary" onClick={()=>setEditing(null)}><Plus/> 새 자동화</button></div>
      {executionOff&&<div className="automation-disabled-notice" role="status"><ShieldAlert/><div><strong>자동화 실행이 꺼져 있습니다</strong><span>정의를 저장하고 검증할 수는 있지만 수동·셀 변경·일정·웹훅 트리거는 실행되지 않습니다. 관리자 화면의 &lsquo;워크북 자동화 실행 정책&rsquo;에서 켜세요.</span></div></div>}
      {automations.isLoading&&<div className="automation-state" role="status" aria-live="polite"><RefreshCw className="spin"/>자동화를 불러오는 중…</div>}
      {automations.isError&&<div className="automation-error" role="alert"><AlertTriangle/><span>자동화 목록을 불러오지 못했습니다.</span><button aria-label="자동화 목록 다시 시도" onClick={()=>void automations.refetch()}>다시 시도</button></div>}
      {(automations.data?.items.length??0)===0&&!automations.isLoading&&!automations.isError&&<div className="automation-empty"><Workflow/><strong>등록된 자동화가 없습니다</strong><span>수동, 셀 변경, Cron 일정 또는 인증 웹훅 트리거를 추가하세요.</span></div>}
      <div className="automation-list">{automations.data?.items.map(item=><article className={previewItem?.id===item.id?'selected':''} key={item.id}><div className="automation-item-head"><span className={item.enabled?'enabled':'disabled'}>{item.enabled?'사용':'중지'}</span><strong>{item.name}</strong><small>r{item.revision}</small></div><p>{triggerLabel(item.trigger,sheets)} → {actionLabel(item.action)} · {sheetLabel(sheets,item.action.sheet_id)}!{item.action.range}</p>{item.enabled&&item.next_run_at&&!executionOff&&<small className="automation-next-run">다음 실행 {new Date(item.next_run_at).toLocaleString('ko-KR')}</small>}{item.last_failure&&<small className="automation-last-failure"><AlertTriangle/>마지막 실행 실패 · {new Date(item.last_failure.at).toLocaleString('ko-KR')} · {item.last_failure.message}</small>}{item.trigger.type==='webhook'&&<code className="automation-webhook-path">POST /api/v1/automations/{item.id}:webhook</code>}<div><button onClick={()=>void test(item)} disabled={pending!==''}><Eye/> 검증</button><button onClick={()=>setEditing(item)} disabled={pending!==''}><Pencil/> 수정</button><button aria-label={`${item.name} 자동화 삭제`} onClick={()=>void remove(item)} disabled={pending!==''}><Trash2/></button></div></article>)}</div>
      {editing!==undefined&&<AutomationForm item={editing??undefined} workbookId={workbookId} sheets={sheets} activeSheetId={activeSheetId} selectionRange={selectionRange} onClose={()=>setEditing(undefined)} onSaved={saved}/>}
      {previewItem&&preview&&<section className="automation-preview"><div className="automation-section-title"><div><Eye/><span><strong>실행 미리보기</strong><small>{preview.automation_name} · 정의 r{preview.automation_revision} · 기준 v{preview.base_version}</small></span></div><em>{preview.changes.length}셀</em></div>{previewStale&&<div className="automation-preview-notice stale" role="alert"><AlertTriangle/>워크북 또는 자동화 정의가 변경되었습니다. 다시 검증하세요.</div>}{preview.changes.length===0?<div className="automation-preview-notice" role="status"><Check/>현재 셀 값이 이미 자동화 결과와 같습니다. 실행할 변경이 없습니다.</div>:<div className="automation-change-list">{preview.changes.slice(0,20).map(change=><article key={`${change.row}:${change.column}`}><strong>{sheetLabel(sheets,preview.action.sheet_id)}!{change.address}</strong><div><small>현재</small><code>{snapshotText(change.before)}</code></div><span>→</span><div><small>실행 후</small><code>{snapshotText(change.after)}</code></div></article>)}</div>}{preview.changes.length>20&&<small className="automation-truncated">앞 20개 변경만 표시합니다.</small>}<div className="automation-approval"><p><ShieldCheck/> {preview.changes.length>0?`${preview.changes.length}개 셀을 하나의 원자적 작업으로 적용합니다.`:'서버에 적용할 변경이 없습니다.'}</p><button className="primary" title={executionOff?'관리자 설정에서 자동화 실행이 꺼져 있습니다.':undefined} disabled={executionOff||!currentPreviewItem?.enabled||pending!==''||previewStale||preview.changes.length===0} onClick={()=>void run(currentPreviewItem??previewItem)}>{pending===`run:${previewItem.id}`?<RefreshCw className="spin"/>:<Play/>} 검토한 자동화 실행</button>{previewStale&&hasPendingRunReplay&&<button onClick={()=>void run(currentPreviewItem??previewItem,true)} disabled={pending!==''}><RefreshCw/> 이전 실행 응답 확인</button>}{previewStale&&<button onClick={()=>void test(currentPreviewItem??previewItem)} disabled={pending!==''}><RefreshCw/> 다시 검증</button>}</div></section>}
      {previewItem&&<section className="automation-history"><div className="automation-section-title"><div><Clock3/><span><strong>실행 이력</strong><small>성공, 실패와 Undo 상태</small></span></div></div>{runs.isLoading&&<div className="automation-state" role="status" aria-live="polite"><RefreshCw className="spin"/>실행 이력을 불러오는 중…</div>}{runs.isError&&<div className="automation-error" role="alert"><AlertTriangle/><span>실행 이력을 불러오지 못했습니다.</span><button aria-label="실행 이력 다시 시도" onClick={()=>void runs.refetch()}>다시 시도</button></div>}{runs.data?.items.length===0&&!runs.isLoading&&!runs.isError&&<p>실행 이력이 없습니다.</p>}{runs.data?.items.map(item=><article key={item.id}><div><strong>{runStatus(item.status)}</strong><small>{new Date(item.started_at).toLocaleString('ko-KR')} · 기준 v{item.base_version}</small>{item.trigger_type==='webhook'&&<small>payload {item.payload_bytes??0}B · SHA-256 {item.payload_digest?.slice(0,12)}…</small>}{item.error_message&&<em>{item.error_message}</em>}</div>{item.status==='succeeded'&&<button aria-label={`${new Date(item.started_at).toLocaleString('ko-KR')} 자동화 실행 Undo`} disabled={pending!==''} onClick={()=>void undo(item)}><RotateCcw/> Undo</button>}{item.status==='undone'&&<Check aria-label="Undo 완료"/>}</article>)}</section>}
      {error&&<div className="automation-error" role="alert" aria-live="assertive"><AlertTriangle/><span>{error}</span></div>}
    </div>
  </aside>
}

function AutomationForm({item,workbookId,sheets,activeSheetId,selectionRange,onClose,onSaved}:{item?:Automation;workbookId:string;sheets:Sheet[];activeSheetId:string;selectionRange:string;onClose:()=>void;onSaved:(item:Automation)=>void|Promise<void>}){
  const [name,setName]=useState('새 자동화')
  const [enabled,setEnabled]=useState(true)
  const [triggerType,setTriggerType]=useState<AutomationTrigger['type']>('manual')
  const [triggerSheet,setTriggerSheet]=useState(activeSheetId)
  const [triggerRange,setTriggerRange]=useState(selectionRange)
  const [cron,setCron]=useState('0 9 * * 1-5')
  const [timezone,setTimezone]=useState('Asia/Seoul')
  const [actionType,setActionType]=useState<AutomationAction['type']>('set_value')
  const [actionSheet,setActionSheet]=useState(activeSheetId)
  const [actionRange,setActionRange]=useState(selectionRange)
  const [formula,setFormula]=useState('=A1')
  const [valueKind,setValueKind]=useState<ValueKind>('string')
  const [value,setValue]=useState('완료')
  const [saving,setSaving]=useState(false)
  const [error,setError]=useState('')
  const [createKey]=useState(newIdempotencyKey)
  // Selection changes outside an open form must not overwrite the user's draft.
  useEffect(()=>{if(!item)return;setName(item.name);setEnabled(item.enabled);setTriggerType(item.trigger.type);setTriggerSheet(item.trigger.sheet_id||activeSheetId);setTriggerRange(item.trigger.range||selectionRange);setCron(item.trigger.cron||'0 9 * * 1-5');setTimezone(item.trigger.timezone||'Asia/Seoul');setActionType(item.action.type);setActionSheet(item.action.sheet_id);setActionRange(item.action.range);setFormula(item.action.formula||'=A1');const next=item.action.value;setValueKind(typeof next==='number'?'number':typeof next==='boolean'?'boolean':'string');setValue(next===undefined?'완료':String(next))},[item?.id,item?.revision])
  const numberInvalid=actionType==='set_value'&&valueKind==='number'&&(value.trim()===''||!Number.isFinite(Number(value)))
  const changeValueKind=(next:ValueKind)=>{setValueKind(next);setError('');setValue(current=>next==='boolean'?(current==='false'?'false':'true'):next==='number'?(current.trim()!==''&&Number.isFinite(Number(current))?String(Number(current)):'0'):current)}
  const submit=async(event:FormEvent)=>{
    event.preventDefault();setError('')
    if(numberInvalid){setError('숫자 값은 비어 있지 않은 유한한 숫자여야 합니다.');return}
    setSaving(true)
    const trigger:AutomationTrigger=triggerType==='manual'?{type:'manual'}:triggerType==='schedule'?{type:'schedule',cron,timezone}:triggerType==='webhook'?{type:'webhook'}:{type:'cell_change',sheet_id:triggerSheet,range:triggerRange}
    const action:AutomationAction={type:actionType,sheet_id:actionSheet,range:actionRange,...(actionType==='set_formula'?{formula}:actionType==='set_value'?{value:literalValue(valueKind,value)}:{})}
    try{
      const saved=await api<Automation>(item?`/api/v1/automations/${item.id}`:`/api/v1/workbooks/${workbookId}/automations`,{method:item?'PATCH':'POST',body:JSON.stringify({...(!item?{idempotency_key:createKey}:{}),name,enabled,trigger,action,...(item?{expected_revision:item.revision}:{})})})
      await onSaved(saved)
    }catch(reason){setError(reason instanceof Error?reason.message:'자동화를 저장하지 못했습니다.')}
    finally{setSaving(false)}
  }
  const formInvalid=saving||!name.trim()||!actionRange.trim()||(triggerType==='cell_change'&&!triggerRange.trim())||(triggerType==='schedule'&&(!cron.trim()||!timezone.trim()))||numberInvalid
  return <form className="automation-form" onSubmit={event=>void submit(event)}><div className="automation-form-title"><strong>{item?'자동화 수정':'새 자동화'}</strong><button aria-label="자동화 편집 닫기" type="button" onClick={onClose}>×</button></div><label>이름<input aria-label="자동화 이름" value={name} maxLength={120} onChange={event=>setName(event.target.value)}/></label><label className="automation-check"><input type="checkbox" checked={enabled} onChange={event=>setEnabled(event.target.checked)}/> 저장 후 사용</label><fieldset><legend>트리거</legend><label>실행 조건<select aria-label="자동화 트리거" value={triggerType} onChange={event=>setTriggerType(event.target.value as AutomationTrigger['type'])}><option value="manual">수동 실행</option><option value="cell_change">셀 변경 시</option><option value="schedule">일정(Cron)</option><option value="webhook">인바운드 웹훅</option></select></label>{triggerType==='cell_change'&&<div className="automation-inline"><label>시트<select aria-label="트리거 시트" value={triggerSheet} onChange={event=>setTriggerSheet(event.target.value)}>{sheets.map(sheet=><option value={sheet.id} key={sheet.id}>{sheet.name}</option>)}</select></label><label>감시 범위<input aria-label="트리거 범위" value={triggerRange} onChange={event=>setTriggerRange(event.target.value.toUpperCase())}/></label></div>}{triggerType==='schedule'&&<><label>실행 일정<select aria-label="스케줄 프리셋" value={schedulePreset(cron)} onChange={event=>{if(event.target.value!=='custom')setCron(event.target.value)}}><option value="0 9 * * 1-5">평일 오전 9시</option><option value="0 * * * *">매시간</option><option value="*/15 * * * *">15분마다</option><option value="0 0 * * *">매일 자정</option><option value="custom">직접 입력</option></select></label><div className="automation-inline"><label>Cron<input aria-label="자동화 Cron" value={cron} onChange={event=>setCron(event.target.value)}/><small>분 시 일 월 요일</small></label><label>시간대<input aria-label="자동화 시간대" list="automation-timezones" value={timezone} onChange={event=>setTimezone(event.target.value)}/><datalist id="automation-timezones"><option value="Asia/Seoul"/><option value="UTC"/><option value="Asia/Tokyo"/><option value="America/New_York"/><option value="Europe/London"/></datalist></label></div></>}{triggerType==='webhook'&&<p className="automation-webhook-help">저장 후 표시되는 endpoint를 <code>automation.webhook.invoke</code> scope의 개인 API 키와 <code>Idempotency-Key</code> 헤더로 호출합니다. JSON 원문은 저장하지 않습니다.</p>}</fieldset><fieldset><legend>작업</legend><label>작업 유형<select aria-label="자동화 작업" value={actionType} onChange={event=>setActionType(event.target.value as AutomationAction['type'])}><option value="set_value">값 설정</option><option value="set_formula">수식 설정</option><option value="clear">내용 지우기</option></select></label><div className="automation-inline"><label>시트<select aria-label="작업 시트" value={actionSheet} onChange={event=>setActionSheet(event.target.value)}>{sheets.map(sheet=><option value={sheet.id} key={sheet.id}>{sheet.name}</option>)}</select></label><label>대상 범위<input aria-label="작업 범위" value={actionRange} onChange={event=>setActionRange(event.target.value.toUpperCase())}/></label></div>{actionType==='set_formula'&&<label>기준 수식<input aria-label="자동화 수식" value={formula} onChange={event=>setFormula(event.target.value)}/><small>대상 범위의 왼쪽 위 셀을 기준으로 상대 참조가 이동합니다.</small></label>}{actionType==='set_value'&&<div className="automation-inline value"><label>값 유형<select aria-label="자동화 값 유형" value={valueKind} onChange={event=>changeValueKind(event.target.value as ValueKind)}><option value="string">문자열</option><option value="number">숫자</option><option value="boolean">불리언</option></select></label><label>값{valueKind==='boolean'?<select aria-label="자동화 값" value={value} onChange={event=>setValue(event.target.value)}><option value="true">TRUE</option><option value="false">FALSE</option></select>:<input aria-label="자동화 값" aria-invalid={numberInvalid||undefined} inputMode={valueKind==='number'?'decimal':undefined} value={value} onChange={event=>setValue(event.target.value)}/>} {numberInvalid&&<small className="automation-field-error">유한한 숫자를 입력하세요.</small>}</label></div>}</fieldset>{error&&<div className="automation-error" role="alert"><AlertTriangle/><span>{error}</span></div>}<div className="automation-form-actions"><button type="button" onClick={onClose}>취소</button><button className="primary" disabled={formInvalid} type="submit">{saving?<RefreshCw className="spin"/>:<Save/>} 저장 후 검증</button></div></form>
}

function literalValue(kind:ValueKind,value:string){if(kind==='number'){const number=Number(value);if(value.trim()===''||!Number.isFinite(number))throw new Error('숫자 값은 유한한 숫자여야 합니다.');return number}if(kind==='boolean')return value==='true';return value}
function runStatus(status:AutomationRun['status']){return {running:'실행 중',succeeded:'성공',skipped:'변경 없음',failed:'실패',undoing:'Undo 중',undone:'Undo 완료'}[status]}
function schedulePreset(cron:string){return ['0 9 * * 1-5','0 * * * *','*/15 * * * *','0 0 * * *'].includes(cron)?cron:'custom'}
