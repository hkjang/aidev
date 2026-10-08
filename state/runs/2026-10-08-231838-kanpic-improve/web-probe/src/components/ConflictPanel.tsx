import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Check, MapPin, RefreshCw, RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { address, api, newIdempotencyKey } from '../lib/api'
import { collaborationClientId } from '../lib/client'
import { useUserDirectory, userLabel, userTooltip } from '../state/directory'
import type { CellConflict, CellConflictResolutionResult, CellConflictSnapshot, Sheet } from '../types'

function snapshotKey(snapshot:CellConflictSnapshot){return JSON.stringify({value:snapshot.value,formula:snapshot.formula??'',style:snapshot.style??{},spill_source:snapshot.spill_source??''})}
function snapshotText(snapshot:CellConflictSnapshot){
  if(snapshot.formula)return snapshot.formula
  if(snapshot.value===undefined||snapshot.value===null||snapshot.value==='')return '(빈 셀)'
  return typeof snapshot.value==='string'?snapshot.value:JSON.stringify(snapshot.value)
}
function styleText(snapshot:CellConflictSnapshot){
  const style=snapshot.style??{},labels:string[]=[]
  if(style.bold)labels.push('굵게')
  if(style.italic)labels.push('기울임')
  if(typeof style.background==='string')labels.push(`배경 ${style.background}`)
  if(typeof style.color==='string')labels.push(`글자 ${style.color}`)
  return labels.join(' · ')
}

function Snapshot({label,snapshot,tone}:{label:string;snapshot:CellConflictSnapshot;tone?:string}){
  const style=styleText(snapshot)
  return <div className={`conflict-snapshot ${tone??''}`}><small>{label}</small><code>{snapshotText(snapshot)}</code>{snapshot.formula&&snapshot.value!==undefined&&<span>계산 결과 {snapshotText({value:snapshot.value})}</span>}{style&&<span>{style}</span>}</div>
}

type Props={
  workbookId:string
  sheets:Sheet[]
  currentActor?:string
  onClose:()=>void
  onNavigate:(sheetId:string,range:string)=>boolean
  onResolved:(result:CellConflictResolutionResult)=>void
}

export function ConflictPanel({workbookId,sheets,currentActor,onClose,onNavigate,onResolved}:Props){
  const client=useQueryClient()
  const [history,setHistory]=useState(false)
  const [error,setError]=useState('')
  const conflicts=useQuery({queryKey:['cell-conflicts',workbookId,history],queryFn:()=>api<{items:CellConflict[]}>(`/api/v1/workbooks/${workbookId}/conflicts${history?'?include_resolved=true':''}`)})
  const resolve=useMutation({
    mutationFn:({item,resolution}:{item:CellConflict;resolution:'keep_current'|'restore_previous'})=>api<CellConflictResolutionResult>(`/api/v1/conflicts/${item.id}:resolve`,{method:'POST',body:JSON.stringify({idempotency_key:newIdempotencyKey(),client_id:collaborationClientId(),expected_revision:item.revision,resolution})}),
    onMutate:()=>setError(''),
    onSuccess:async result=>{onResolved(result);await client.invalidateQueries({queryKey:['cell-conflicts',workbookId]})},
    onError:reason=>setError(reason instanceof Error?reason.message:'충돌을 해소하지 못했습니다.'),
  })
  const items=conflicts.data?.items??[]
  const directory=useUserDirectory(items.flatMap(item=>[item.actor_id,item.conflicting_actor_id,item.resolved_by]))
  return <aside className="conflict-panel" aria-label="편집 충돌 패널">
    <div className="conflict-panel-head"><span><AlertTriangle/> 편집 충돌</span><button onClick={onClose} aria-label="편집 충돌 닫기">×</button></div>
    <label className="conflict-history-toggle"><input type="checkbox" checked={history} onChange={event=>setHistory(event.target.checked)}/> 해소된 충돌 이력 포함</label>
    {error&&<div className="conflict-error" role="alert">{error}</div>}
    <div className="conflict-list">
      {conflicts.isLoading&&<div className="conflict-empty"><RefreshCw/>충돌 기록을 불러오는 중…</div>}
      {conflicts.isError&&<div className="conflict-empty error-text"><AlertTriangle/>충돌 기록을 불러오지 못했습니다.</div>}
      {!conflicts.isLoading&&!conflicts.isError&&items.length===0&&<div className="conflict-empty"><Check/><strong>{history?'충돌 이력이 없습니다.':'열린 충돌이 없습니다.'}</strong><span>동일 셀 동시 수정이 감지되면 여기서 값을 비교하고 결정할 수 있습니다.</span></div>}
      {items.map(item=>{
        const sheet=sheets.find(candidate=>candidate.id===item.sheet_id),location=address(item.row,item.column)
        const canRestore=snapshotKey(item.current_cell)===snapshotKey(item.applied_cell)
        const resolved=item.status==='resolved'
        return <article className={`conflict-card ${resolved?'resolved':''}`} key={item.id}>
          <div className="conflict-card-head"><button onClick={()=>onNavigate(item.sheet_id,location)}><MapPin/>{sheet?.name??'삭제된 시트'}!{location}</button><span>{resolved?'해소됨':'확인 필요'}</span></div>
          <div className="conflict-meta">v{item.base_version} 기준 · v{item.changed_at_version}에서 충돌 · {new Date(item.created_at).toLocaleString('ko-KR')}</div>
          <Snapshot label="충돌 전 기준" snapshot={item.base_cell}/>
          <Snapshot label={`먼저 반영된 값${item.conflicting_actor_id?` · ${item.conflicting_actor_id===currentActor?'나':userLabel(item.conflicting_actor_id,directory)}`:''}`} snapshot={item.conflicting_cell} tone="other"/>
          <Snapshot label="현재 서버 값" snapshot={item.current_cell} tone="current"/>
          {snapshotKey(item.submitted_cell)!==snapshotKey(item.current_cell)&&<Snapshot label="당시 내 제출 값" snapshot={item.submitted_cell}/>}
          {resolved?<div className="conflict-resolution"><Check/>{item.resolution==='restore_previous'?'먼저 반영된 값으로 복원':'현재 값 유지'} · <span title={userTooltip(item.resolved_by??'',directory)}>{userLabel(item.resolved_by??'',directory)}</span></div>:<>
            {!canRestore&&<p className="conflict-stale"><AlertTriangle/>충돌 뒤 값이 다시 변경되어 복원할 수 없습니다. 현재 값을 확인한 뒤 유지하거나 새 편집을 적용하세요.</p>}
            <div className="conflict-actions"><button disabled={resolve.isPending||!canRestore} onClick={()=>resolve.mutate({item,resolution:'restore_previous'})}><RotateCcw/> 먼저 반영된 값 복원</button><button className="primary" disabled={resolve.isPending} onClick={()=>resolve.mutate({item,resolution:'keep_current'})}><Check/> 현재 값 유지</button></div>
          </>}
        </article>
      })}
    </div>
    <p className="conflict-safety">모든 결정은 새 워크북 버전과 작업 이력으로 기록됩니다. 복원은 이후 셀 변경이 없을 때만 허용됩니다.</p>
  </aside>
}
