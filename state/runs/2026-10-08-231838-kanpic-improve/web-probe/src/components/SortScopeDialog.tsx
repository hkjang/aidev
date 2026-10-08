import { MAX_SORT_CELLS } from '../lib/clipboard'
import { useState } from 'react'
import { AlertTriangle, ArrowDownAZ, ArrowUpAZ } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { address } from '../lib/api'
import { cleanupText } from '../lib/dataCleanup'
import { looksLikeHeaderRow } from '../lib/dataRegion'
import { cellKey } from '../state/editor'
import type { Cell } from '../types'
import type { GridRegion } from '../lib/dataRegion'
import './SortScopeDialog.css'

const label=(region:GridRegion)=>`${address(region.startRow,region.startColumn)}:${address(region.endRow,region.endColumn)}`
const columnName=(column:number)=>address(1,column).replace(/\d+$/,'')

/**
 * Sorting one column of a table and leaving the rest where they are pairs each
 * value with somebody else's row. The dialog therefore states which block is
 * about to move, and offers the narrow sort only as a deliberate choice.
 */
export function SortScopeDialog({column,direction,block,selection,onClose,onSort}:{
  column:number
  direction:'asc'|'desc'
  block:{region:GridRegion;cells:Map<string,Cell>}
  selection:GridRegion
  onClose:()=>void
  onSort:(region:GridRegion,cells:Map<string,Cell>)=>Promise<void>
}){
  // A single cell is a cursor, not a chosen range: offering to sort it alone
  // would be noise. The choice only matters when somebody deliberately
  // selected part of a table.
  const spans=selection.endRow>selection.startRow||selection.endColumn>selection.startColumn
  const narrower=spans&&(selection.startColumn>block.region.startColumn||selection.endColumn<block.region.endColumn)
  const [scope,setScope]=useState<'block'|'selection'>('block')
  const [busy,setBusy]=useState(false)
  const dialog=useDialog<HTMLElement>(onClose)
  const region=scope==='selection'?selection:block.region
  const header=looksLikeHeaderRow(block.cells,block.region)?1:0
  const rows=Math.max(0,region.endRow-region.startRow+1-(scope==='block'?header:0))
  const sample=[] as string[]
  for(let row=region.startRow+(scope==='block'?header:0);row<=region.endRow&&sample.length<3;row+=1){
    const text=cleanupText(block.cells.get(cellKey(row,column)))
    if(text!=='')sample.push(text)
  }
  // 정렬은 범위 전체를 다시 쓰므로 한 번에 처리할 수 있는 크기가 있다.
  // 눌러 본 뒤에 알려 주면 사용자는 무엇을 줄여야 할지 모른 채 실패를
  // 겪는다. 여기서 미리 말하고 버튼을 잠근다.
  const cells=Math.max(0,region.endRow-region.startRow+1)*Math.max(0,region.endColumn-region.startColumn+1)
  const tooLarge=cells>MAX_SORT_CELLS
  const run=async()=>{
    setBusy(true)
    // 실패는 정렬을 수행하는 쪽이 이미 알린다. 여기서 또 알리면 같은 말이
    // 두 번 뜬다.
    try{await onSort(region,block.cells);onClose()}
    catch{/* already reported */}
    finally{setBusy(false)}
  }
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal sort-scope-modal" ref={dialog as React.RefObject<never>} role="dialog" aria-modal="true" aria-label="정렬 범위 확인">
      <h2>{direction==='asc'?<ArrowUpAZ/>:<ArrowDownAZ/>} {columnName(column)}열 기준 {direction==='asc'?'오름차순':'내림차순'} 정렬</h2>
      {narrower
        ?<div className="sort-scope-options" role="radiogroup" aria-label="정렬 범위">
          <label><input type="radio" name="sort-scope" aria-label="표 전체 정렬" checked={scope==='block'} onChange={()=>setScope('block')}/>
            <span><strong>표 전체 {label(block.region)}</strong><em>행 전체가 함께 움직여 다른 열과 짝이 유지됩니다.</em></span></label>
          <label><input type="radio" name="sort-scope" aria-label="선택 범위만 정렬" checked={scope==='selection'} onChange={()=>setScope('selection')}/>
            <span><strong>선택 범위만 {label(selection)}</strong><em className="sort-scope-warning"><AlertTriangle/> 다른 열은 그대로 있어 값의 짝이 어긋납니다.</em></span></label>
        </div>
        :<p className="sort-scope-single">{label(region)} 범위를 정렬합니다.{header?' 첫 행은 머리글로 유지합니다.':''}</p>}
      <p className="sort-scope-summary">{rows.toLocaleString()}행이 정렬됩니다.{sample.length>0?` 예: ${sample.join(', ')}`:''}</p>
      {tooLarge&&<p className="sort-scope-warning"><AlertTriangle/> 이 범위는 {cells.toLocaleString()}셀로, 한 번에 정렬할 수 있는 {MAX_SORT_CELLS.toLocaleString()}셀을 넘습니다. 더 좁은 범위를 선택해 정렬하세요.</p>}
      <div className="modal-actions">
        <button onClick={onClose}>취소</button>
        <button className="primary" disabled={busy||rows<2||tooLarge} onClick={()=>void run()}>정렬</button>
      </div>
    </section>
  </div>
}
