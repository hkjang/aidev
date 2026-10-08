import { useState } from 'react'
import { Link2 } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { parseFilterRange } from '../lib/filter'
import { safeLinkTarget, workbookRangeLink } from '../lib/hyperlink'
import type { Sheet } from '../types'
import './LinkDialog.css'

/**
 * Two kinds of link go into a sheet: one to somewhere on the web, and one to
 * another place in this workbook. The second is the one a spreadsheet needs
 * and the one nobody can write by hand, so it gets its own tab rather than
 * asking people to paste a URL they would have to build themselves.
 */
export function LinkDialog({workbookId,sheets,activeSheetId,selectionRange,onClose,onApply}:{
  workbookId:string
  sheets:Sheet[]
  activeSheetId:string
  selectionRange:string
  onClose:()=>void
  onApply:(formula:string)=>void
}){
  const [mode,setMode]=useState<'url'|'range'>('url')
  const [url,setUrl]=useState('https://')
  const [sheetId,setSheetId]=useState(activeSheetId)
  const [range,setRange]=useState(selectionRange)
  const [label,setLabel]=useState('')
  const dialog=useDialog<HTMLElement>(onClose)
  const sheetName=sheets.find(sheet=>sheet.id===sheetId)?.name??''
  const apply=()=>{
    let href=''
    if(mode==='url'){
      const target=safeLinkTarget(url)
      if(!target)return alert('http, https, mailto 주소만 넣을 수 있습니다.')
      href=target
    }else{
      // A jump target is usually one cell, so C20 counts as much as B2:C10.
      const target=range.trim().toUpperCase()
      if(!parseFilterRange(target.includes(':')?target:`${target}:${target}`))return alert('올바른 A1 주소나 범위를 입력하세요. 예: C20 또는 B2:C10')
      href=workbookRangeLink(workbookId,sheetId,target)
    }
    const text=label.trim()||(mode==='range'?`${sheetName}!${range.toUpperCase()}`:href)
    const escape=(value:string)=>value.replace(/"/g,'""')
    onApply(`=HYPERLINK("${escape(href)}","${escape(text)}")`)
    onClose()
  }
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal link-modal" ref={dialog as React.RefObject<never>} role="dialog" aria-modal="true" aria-label="링크 삽입">
      <h2><Link2/> 링크 삽입</h2>
      <div className="link-tabs" role="tablist">
        <button role="tab" aria-selected={mode==='url'} className={mode==='url'?'active':''} onClick={()=>setMode('url')}>웹 주소</button>
        <button role="tab" aria-selected={mode==='range'} className={mode==='range'?'active':''} onClick={()=>setMode('range')}>이 워크북의 범위</button>
      </div>
      {mode==='url'
        ?<label>주소<input aria-label="링크 주소" value={url} onChange={event=>setUrl(event.target.value)} placeholder="https://example.com"/></label>
        :<div className="link-range-fields">
          <label>시트<select aria-label="링크 시트" value={sheetId} onChange={event=>setSheetId(event.target.value)}>{sheets.map(sheet=><option key={sheet.id} value={sheet.id}>{sheet.name}</option>)}</select></label>
          <label>범위<input aria-label="링크 범위" value={range} onChange={event=>setRange(event.target.value)} placeholder="예: B2:C10"/></label>
        </div>}
      <label>표시할 텍스트<input aria-label="링크 표시 텍스트" value={label} onChange={event=>setLabel(event.target.value)} placeholder={mode==='range'?`${sheetName}!${range.toUpperCase()}`:'비우면 주소가 그대로 보입니다'}/></label>
      <div className="modal-actions">
        <button onClick={onClose}>취소</button>
        <button className="primary" onClick={apply}>링크 넣기</button>
      </div>
    </section>
  </div>
}
