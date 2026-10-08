import { Columns3, Rows3 } from 'lucide-react'
import { useState } from 'react'
import type { MergeRange } from '../lib/merge'
import { useDialog } from '../lib/useDialog'

export type StructureCommand = { axis:'row'|'column'; action:'insert'|'delete'|'move'; index:number; count:number; destination?:number }

export function StructureDialog({range,onClose,onApply}:{range:MergeRange;onClose:()=>void;onApply:(command:StructureCommand)=>Promise<void>}){
  const [saving,setSaving]=useState(false)
  const rows=range.endRow-range.startRow+1,columns=range.endColumn-range.startColumn+1
  const run=async(command:StructureCommand)=>{
    if(command.action==='delete'&&!window.confirm(`선택한 ${command.count}개 ${command.axis==='row'?'행':'열'}을 삭제할까요? 삭제 전 복구 버전이 자동 생성됩니다.`))return
    setSaving(true)
    try{await onApply(command);onClose()}finally{setSaving(false)}
  }
  const dialog=useDialog<HTMLElement>(onClose)
  return <div className="modal-backdrop"><div className="modal structure-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="행과 열 관리">
    <h2>행과 열 관리</h2><p>선택 범위를 기준으로 구조를 변경합니다. 수식과 이름 범위 등 관련 참조도 함께 갱신됩니다.</p>
    <section className="structure-section"><div><Rows3/><span><strong>행</strong><small>{range.startRow}–{range.endRow}행 선택</small></span></div><div className="structure-actions"><button disabled={saving} onClick={()=>void run({axis:'row',action:'insert',index:range.startRow,count:rows})}>위에 {rows}개 삽입</button><button className="danger" disabled={saving} onClick={()=>void run({axis:'row',action:'delete',index:range.startRow,count:rows})}>선택 행 삭제</button></div></section>
    <section className="structure-section"><div><Columns3/><span><strong>열</strong><small>{range.startColumn}–{range.endColumn}열 선택</small></span></div><div className="structure-actions"><button disabled={saving} onClick={()=>void run({axis:'column',action:'insert',index:range.startColumn,count:columns})}>왼쪽에 {columns}개 삽입</button><button className="danger" disabled={saving} onClick={()=>void run({axis:'column',action:'delete',index:range.startColumn,count:columns})}>선택 열 삭제</button></div></section>
    <div className="structure-note">구조 변경은 온라인에서만 실행됩니다. 변경 직전 버전은 자동으로 보관되어 버전 이력에서 복원할 수 있습니다.</div>
    <div className="modal-actions"><button className="secondary" disabled={saving} onClick={onClose}>닫기</button></div>
  </div></div>
}
