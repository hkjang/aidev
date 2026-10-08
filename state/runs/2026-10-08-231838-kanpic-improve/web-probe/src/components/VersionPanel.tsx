import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Clock3, History, RotateCcw, Save } from 'lucide-react'
import { FormEvent, useState } from 'react'
import { api } from '../lib/api'
import { useUserDirectory, userLabel, userTooltip } from '../state/directory'
import type { MutationResult, WorkbookVersion } from '../types'

export function VersionPanel({workbookId,currentVersion,onClose,onRestored}:{workbookId:string;currentVersion:number;onClose:()=>void;onRestored:(result:MutationResult)=>void}) {
  const client=useQueryClient()
  const [name,setName]=useState('')
  const versions=useQuery({queryKey:['workbook-versions',workbookId],queryFn:()=>api<{items:WorkbookVersion[]}>(`/api/v1/workbooks/${workbookId}/versions`)})
  const directory=useUserDirectory((versions.data?.items??[]).map(version=>version.actor_id))
  const create=useMutation({
    mutationFn:(versionName:string)=>api<WorkbookVersion>(`/api/v1/workbooks/${workbookId}/versions`,{method:'POST',body:JSON.stringify({name:versionName})}),
    onSuccess:()=>{setName('');client.invalidateQueries({queryKey:['workbook-versions',workbookId]})},
  })
  const restore=useMutation({
    mutationFn:(versionId:string)=>api<MutationResult>(`/api/v1/versions/${versionId}:restore`,{method:'POST',body:'{}'}),
    onSuccess:(result)=>{onRestored(result);client.invalidateQueries({queryKey:['workbook-versions',workbookId]})},
  })
  const submit=(event:FormEvent)=>{event.preventDefault();const value=name.trim();if(value)create.mutate(value)}
  const restoreVersion=(version:WorkbookVersion)=>{
    if(confirm(`“${version.name||`v${version.workbook_version}`}” 버전으로 복원할까요? 현재 상태는 자동 백업됩니다.`))restore.mutate(version.id)
  }
  return <aside className="version-panel">
    <div className="version-panel-head"><span><History/> 버전 이력</span><button onClick={onClose} aria-label="버전 이력 닫기">×</button></div>
    <div className="version-current"><Clock3/><div><small>현재 워크북 버전</small><strong>v{currentVersion}</strong></div></div>
    <form className="named-version-form" onSubmit={submit}><label>이름 있는 버전<input value={name} onChange={event=>setName(event.target.value)} placeholder="예: 2026년 3분기 확정" maxLength={120}/></label><button className="primary" disabled={!name.trim()||create.isPending}><Save/> 저장</button></form>
    <div className="workbook-version-list">
      {versions.isLoading&&<p className="empty-version">버전 이력을 불러오는 중…</p>}
      {versions.isError&&<p className="empty-version error-text">버전 이력을 불러오지 못했습니다.</p>}
      {versions.data?.items.length===0&&<p className="empty-version">아직 저장된 버전이 없습니다.</p>}
      {versions.data?.items.map(version=><article key={version.id} className="workbook-version"><span className="version-node"/><div><strong>{version.name||`버전 ${version.workbook_version}`}</strong><small>v{version.workbook_version} · <span title={userTooltip(version.actor_id,directory)}>{userLabel(version.actor_id,directory)}</span></small><time>{new Date(version.created_at).toLocaleString('ko-KR')}</time></div><button className="ghost" onClick={()=>restoreVersion(version)} disabled={restore.isPending} title="이 버전 복원"><RotateCcw/> 복원</button></article>)}
    </div>
    <p className="version-safety">복원은 기존 버전을 덮어쓰지 않고 새로운 최신 버전으로 기록됩니다.</p>
  </aside>
}
