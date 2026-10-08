import { useState } from 'react'
import { StickyNote } from 'lucide-react'
import { useDialog } from '../lib/useDialog'

const MAX_NOTE=1000

/**
 * A cell note is a short annotation the reader sees on hover. It is not a
 * comment: nobody replies to it and it never appears in the comment panel, so
 * the editor is deliberately one box and two buttons.
 */
export function NoteDialog({address,note,onClose,onApply}:{
  address:string
  note:string
  onClose:()=>void
  onApply:(note:string)=>Promise<void>
}){
  const [text,setText]=useState(note)
  const [saving,setSaving]=useState(false)
  const dialog=useDialog<HTMLElement>(onClose)
  const run=async(value:string)=>{setSaving(true);try{await onApply(value);onClose()}finally{setSaving(false)}}
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal" ref={dialog as React.RefObject<any>} role="dialog" aria-modal="true" aria-label="셀 메모">
      <h2><StickyNote/> 메모</h2>
      <p>{address} 셀에 붙는 짧은 설명입니다. 셀 위에 마우스를 올리면 보이고, 댓글과 달리 답글이 달리지 않습니다.</p>
      <label><span className="sr-only">메모 내용</span>
        <textarea autoFocus aria-label="메모 내용" maxLength={MAX_NOTE} value={text} placeholder="예: 협력사 확정 단가. 분기마다 재확인" onChange={event=>setText(event.target.value)}
          onKeyDown={event=>{if(event.key==='Enter'&&(event.ctrlKey||event.metaKey)){event.preventDefault();void run(text.trim())}}}/>
      </label>
      <small className="note-count">{text.length.toLocaleString()} / {MAX_NOTE.toLocaleString()}자 · Ctrl+Enter로 저장</small>
      <div className="modal-actions">
        {note&&<button className="danger-text" disabled={saving} onClick={()=>void run('')}>메모 삭제</button>}
        <span style={{flex:1}}/>
        <button onClick={onClose}>취소</button>
        <button className="primary" disabled={saving||text.trim()===note} onClick={()=>void run(text.trim())}>저장</button>
      </div>
    </section>
  </div>
}
