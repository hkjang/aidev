import { Bell,Plus } from 'lucide-react'
import { useState } from 'react'
import { address } from '../lib/api'
import type { MergeRange } from '../lib/merge'
import type { Sheet,WatchRule } from '../types'
import './WatchRuleDialog.css'
import { useDialog } from '../lib/useDialog'

const rangeText=(range:MergeRange)=>`${address(range.startRow,range.startColumn)}:${address(range.endRow,range.endColumn)}`
const cellPosition=(value:string)=>/^\$?[A-Za-z]{1,3}\$?[1-9]\d*$/.test(value.trim())
const validTarget=(value:string)=>{const text=value.trim();if(!text)return true;const parts=text.split(':');return parts.length<=2&&parts.every(cellPosition)}

export function WatchRuleDialog({selection,sheets,activeSheetId,rules,onClose,onCreate,onUpdate,onDelete}:{
  selection:MergeRange;sheets:Sheet[];activeSheetId:string;rules:WatchRule[];onClose:()=>void
  onCreate:(input:Record<string,unknown>)=>Promise<WatchRule>
  onUpdate:(id:string,input:Record<string,unknown>)=>Promise<WatchRule>
  onDelete:(item:WatchRule)=>Promise<void>
}){
  const [range,setRange]=useState(rangeText(selection)),[label,setLabel]=useState(''),[saving,setSaving]=useState(false)
  const sheetName=(id:string)=>sheets.find(sheet=>sheet.id===id)?.name??'삭제된 시트'
  const add=async()=>{
    if(!validTarget(range))return alert('올바른 A1 셀 또는 범위를 입력하세요. 비워 두면 시트 전체를 지켜봅니다.')
    setSaving(true)
    try{await onCreate({sheet_id:activeSheetId,range:range.trim().toUpperCase(),label:label.trim()});setLabel('')}
    catch(error){alert(error instanceof Error?error.message:'지켜보기를 걸지 못했습니다.')}
    finally{setSaving(false)}
  }
  const dialog=useDialog<HTMLElement>(onClose)
  return <div className="modal-backdrop"><div className="modal watch-rule-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="변경 알림">
    <header><div><Bell/><div><h2>변경 알림</h2><p>지켜보는 범위가 바뀌면 메일로 알려 드립니다.</p></div></div><button aria-label="변경 알림 닫기" onClick={onClose}>×</button></header>
    <div className="watch-rule-body">
      <div className="watch-rule-new">
        <label>범위<input aria-label="지켜볼 범위" value={range} onChange={event=>setRange(event.target.value)} placeholder="A1:B20 (비우면 시트 전체)"/></label>
        <label>이름<input aria-label="지켜보기 이름" value={label} maxLength={200} onChange={event=>setLabel(event.target.value)} placeholder="매출표"/></label>
        <button className="primary" disabled={saving||!validTarget(range)} onClick={add}><Plus/> 지켜보기</button>
      </div>
      <p className="watch-rule-note">내가 고친 것은 나에게 보내지 않습니다. 한 번 저장에 여러 칸이 바뀌어도 메일은 한 통입니다.</p>
      {rules.length===0
        ? <p className="watch-rule-empty">아직 지켜보는 범위가 없습니다.</p>
        : <ul className="watch-rule-list">{rules.map(item=><li key={item.id}>
            <div><strong>{item.label||item.range||'시트 전체'}</strong><em>{sheetName(item.sheet_id)}{item.range?`!${item.range}`:' 전체'}</em></div>
            <label className="watch-rule-toggle"><input type="checkbox" aria-label={`${item.label||item.range||'시트 전체'} 알림 켜기`} checked={item.enabled} onChange={event=>void onUpdate(item.id,{enabled:event.target.checked,expected_revision:item.revision})}/> 켜짐</label>
            <button className="danger" aria-label={`${item.label||item.range||'시트 전체'} 지켜보기 그만두기`} onClick={()=>void onDelete(item)}>그만두기</button>
          </li>)}</ul>}
    </div>
    <div className="modal-actions"><span/><button className="secondary" onClick={onClose}>닫기</button></div>
  </div></div>
}
