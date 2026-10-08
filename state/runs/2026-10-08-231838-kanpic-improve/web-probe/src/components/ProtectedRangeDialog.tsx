import { useState } from 'react'
import { Lock, Plus, Trash2 } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { address } from '../lib/api'
import type { MergeRange } from '../lib/merge'
import type { ProtectedRange } from '../types'

/**
 * Sharing decides who can open a workbook; this decides who can change the
 * cells a model depends on. The dialog is deliberately a list plus one form:
 * a range, who may edit it, and whether it blocks or only warns.
 */
export function ProtectedRangeDialog({range,rules,onClose,onCreate,onDelete}:{
  range:MergeRange
  rules:ProtectedRange[]
  onClose:()=>void
  onCreate:(input:Record<string,unknown>)=>Promise<void>
  onDelete:(rule:ProtectedRange)=>Promise<void>
}){
  const selection=`${address(range.startRow,range.startColumn)}:${address(range.endRow,range.endColumn)}`
  const [scope,setScope]=useState<'range'|'sheet'>('range')
  const [target,setTarget]=useState(selection)
  const [exceptions,setExceptions]=useState('')
  const [description,setDescription]=useState('')
  const [editors,setEditors]=useState('')
  const [warningOnly,setWarningOnly]=useState(false)
  const [saving,setSaving]=useState(false)
  const dialog=useDialog<HTMLElement>(onClose)
  const save=async()=>{
    setSaving(true)
    try{
      await onCreate({
        scope,range:scope==='sheet'?undefined:target.toUpperCase(),description:description.trim(),warning_only:warningOnly,
        exceptions:scope==='sheet'?exceptions.split(/[,\n]/).map(item=>item.trim().toUpperCase()).filter(Boolean):undefined,
        editors:editors.split(/[,\n]/).map(item=>item.trim()).filter(Boolean),
      })
      setDescription('');setEditors('');setExceptions('')
    }catch(error){alert(error instanceof Error?error.message:'범위를 보호하지 못했습니다.')}
    finally{setSaving(false)}
  }
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal protected-modal" ref={dialog as React.RefObject<any>} role="dialog" aria-modal="true" aria-label="범위 보호">
      <h2><Lock/> 시트·범위 보호</h2>
      <p>보호한 곳은 아래에 적은 사람과 워크북 소유자만 수정할 수 있습니다. 공유 권한과 별개로 셀 단위를 지킵니다.</p>
      <div className="protected-form">
        <label>보호 대상<select aria-label="보호 대상" value={scope} onChange={event=>setScope(event.target.value as 'range'|'sheet')}><option value="range">선택한 범위</option><option value="sheet">시트 전체</option></select></label>
        {scope==='range'
          ?<label>범위<input aria-label="보호할 범위" value={target} onChange={event=>setTarget(event.target.value)}/></label>
          :<label>편집 허용 범위<input aria-label="편집 허용 범위" placeholder="예: B2:C10, E1 (비우면 시트 전체 잠금)" value={exceptions} onChange={event=>setExceptions(event.target.value)}/></label>}
        <label>설명<input aria-label="보호 설명" placeholder="예: 요율표 — 재무팀만 수정" value={description} onChange={event=>setDescription(event.target.value)}/></label>
        <label className="wide">편집 허용 사용자<input aria-label="편집 허용 사용자" placeholder="쉼표로 구분한 사용자 ID (비우면 소유자만)" value={editors} onChange={event=>setEditors(event.target.value)}/></label>
        <label className="check-label"><input type="checkbox" aria-label="경고만 표시" checked={warningOnly} onChange={event=>setWarningOnly(event.target.checked)}/> 막지 않고 경고만 남기기</label>
      </div>
      <div className="modal-actions">
        <button className="primary" disabled={saving||(scope==='range'&&!target.trim())} onClick={()=>void save()}><Plus/> {scope==='sheet'?'이 시트 보호':'이 범위 보호'}</button>
      </div>
      <div className="protected-list">
        {rules.length===0&&<p className="empty-hint">보호된 범위가 없습니다.</p>}
        {rules.map(rule=>{
          const label=rule.scope==='sheet'?'시트 전체':rule.range
          return <div className="protected-row" key={rule.id}>
            <div>
              <strong>{label}{rule.warning_only?' · 경고만':''}</strong>
              <small>{rule.description||'설명 없음'} · {rule.editors.length>0?`편집 허용 ${rule.editors.join(', ')}`:'소유자만'}{rule.exceptions?.length?` · 예외 ${rule.exceptions.join(', ')}`:''}</small>
            </div>
            <button aria-label={`${label} 보호 해제`} onClick={()=>void onDelete(rule)}><Trash2/></button>
          </div>
        })}
      </div>
      <div className="modal-actions"><button onClick={onClose}>닫기</button></div>
    </section>
  </div>
}
