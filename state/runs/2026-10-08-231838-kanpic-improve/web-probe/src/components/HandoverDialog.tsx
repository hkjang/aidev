import { useEffect, useMemo, useState } from 'react'
import { UserMinus, AlertTriangle } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { api } from '../lib/api'
import type { DirectoryUser, GovernedWorkbook } from '../types'
import './HandoverDialog.css'

type Owned={items:GovernedWorkbook[];trashed:number}
type Outcome={
  transferred:Array<{id:string;title:string}>
  failed:Array<{id:string;title:string;reason:string}>
  skipped_trashed:number
}

/**
 * 퇴사자가 가진 워크북을 한 번에 넘긴다.
 *
 * 워크북 하나씩 넘기는 것은 전부터 있었다. 그런데 마흔 개를 가진 사람이면
 * 마흔 번을 눌러야 하고, 몇 개를 빠뜨렸는지는 아무도 모른다. 빠뜨린 것은
 * 정지된 계정에 묶인 채로 잊힌다.
 */
export function HandoverDialog({user,onClose,onDone}:{user:DirectoryUser;onClose:()=>void;onDone:(message:string)=>void}){
  const dialog=useDialog<HTMLElement>(onClose)
  const [owned,setOwned]=useState<Owned>()
  const [error,setError]=useState('')
  const [newOwner,setNewOwner]=useState('')
  const [keepAsEditor,setKeepAsEditor]=useState(false)
  const [busy,setBusy]=useState(false)
  const [outcome,setOutcome]=useState<Outcome>()

  useEffect(()=>{
    let alive=true
    api<Owned>(`/api/v1/admin/users/${encodeURIComponent(user.user_id)}/workbooks`)
      .then(result=>{if(alive)setOwned(result)})
      .catch(reason=>{if(alive)setError(reason instanceof Error?reason.message:'가진 워크북을 읽지 못했습니다.')})
    return ()=>{alive=false}
  },[user.user_id])

  const live=useMemo(()=>owned?.items.filter(item=>!item.deleted_at)??[],[owned])
  const transfer=async()=>{
    setBusy(true);setError('')
    try{
      const result=await api<Outcome>(`/api/v1/admin/users/${encodeURIComponent(user.user_id)}/workbooks:transfer`,
        {method:'POST',body:JSON.stringify({new_owner_id:newOwner.trim(),keep_as_editor:keepAsEditor})})
      setOutcome(result)
      if(result.failed.length===0)onDone(`${result.transferred.length}개 워크북을 ${newOwner.trim()} 에게 넘겼습니다.`)
    }catch(reason){setError(reason instanceof Error?reason.message:'넘기지 못했습니다.')}
    finally{setBusy(false)}
  }

  return <div className="modal-backdrop"><div className="modal handover-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="가진 워크북 인수인계">
    <header><div><UserMinus/><div><h2>가진 워크북 인수인계</h2><p>{user.display_name||user.user_id} 이(가) 소유한 워크북을 한 번에 넘깁니다.</p></div></div><button aria-label="인수인계 닫기" onClick={onClose}>×</button></header>
    <div className="handover-body">
      {error&&<p className="handover-error"><AlertTriangle/> {error}</p>}
      {!owned&&!error&&<p className="handover-note">읽는 중…</p>}
      {owned&&<>
        <p className="handover-count">소유한 워크북 <b>{owned.items.length.toLocaleString()}개</b>{owned.trashed>0&&<> · 휴지통 <b>{owned.trashed.toLocaleString()}개</b></>}</p>
        {owned.trashed>0&&<p className="handover-note">휴지통에 있는 것은 넘기지 않습니다. 되살린 뒤에 다시 넘기세요.</p>}
        {live.length===0
          ?<p className="handover-note">넘길 워크북이 없습니다.</p>
          :<ul className="handover-list">{live.slice(0,12).map(item=><li key={item.id}>
            <b>{item.title||'(제목 없음)'}</b>
            <span>시트 {item.sheet_count} · 공유 {item.share_count}{item.link_access!=='restricted'&&<> · <em>링크 공개</em></>}</span>
          </li>)}
          {live.length>12&&<li className="handover-more">… 그리고 {(live.length-12).toLocaleString()}개 더</li>}</ul>}
      </>}
      {outcome&&<div className="handover-outcome">
        <p>넘긴 것 <b>{outcome.transferred.length.toLocaleString()}개</b>{outcome.skipped_trashed>0&&<> · 휴지통이라 건너뛴 것 {outcome.skipped_trashed.toLocaleString()}개</>}</p>
        {outcome.failed.length>0&&<>
          <p className="handover-error"><AlertTriangle/> 넘기지 못한 것 {outcome.failed.length.toLocaleString()}개</p>
          <ul className="handover-list">{outcome.failed.map(item=><li key={item.id}><b>{item.title||item.id}</b><span>{item.reason}</span></li>)}</ul>
        </>}
      </div>}
      {live.length>0&&!outcome&&<div className="handover-fields">
        <label>새 소유자<input aria-label="새 소유자" value={newOwner} onChange={event=>setNewOwner(event.target.value)} placeholder="사용자 ID 또는 이메일"/></label>
        <label className="handover-check"><input type="checkbox" checked={keepAsEditor} onChange={event=>setKeepAsEditor(event.target.checked)}/> 이전 소유자를 편집자로 남기기</label>
      </div>}
    </div>
    <div className="modal-actions"><span/>
      <button className="secondary" onClick={onClose}>{outcome?'닫기':'취소'}</button>
      {live.length>0&&!outcome&&<button className="primary" disabled={busy||newOwner.trim()===''} onClick={()=>void transfer()}>{busy?'넘기는 중…':`${live.length.toLocaleString()}개 넘기기`}</button>}
    </div>
  </div></div>
}
