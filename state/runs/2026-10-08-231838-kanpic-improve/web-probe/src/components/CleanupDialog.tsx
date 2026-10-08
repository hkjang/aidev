import { useMemo, useState } from 'react'
import { AlertTriangle, Eraser, Rows3 } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { cleanupText, convertTextNumbers, removeDuplicateRows, trimWhitespace, unmergeAndFill } from '../lib/dataCleanup'
import { cellMerge } from '../lib/merge'
import { planRemoveSubtotals } from '../lib/subtotal'
import { cellKey } from '../state/editor'
import type { Cell } from '../types'
import type { GridRegion } from '../lib/dataRegion'
import { address } from '../lib/api'
import './CleanupDialog.css'

const PREVIEW_ROWS=6
const label=(region:GridRegion)=>`${address(region.startRow,region.startColumn)}:${address(region.endRow,region.endColumn)}`

export type CleanupTarget={
  region:GridRegion
  cells:Map<string,Cell>
  /** The data block around the selection, when it is wider than the selection. */
  block?:GridRegion
  blockCells?:Map<string,Cell>
  headerRows:number
}

/**
 * Both cleanups rewrite cells in place, and one of them moves rows. The dialog
 * exists to show what goes before it goes — and, for duplicates, to catch the
 * case where the selection is narrower than the table: shifting rows up in
 * some columns and not others quietly misaligns the data.
 */
export function CleanupDialog({mode,target,onClose,onApply}:{
  mode:'duplicates'|'trim'|'subtotals'|'numbers'|'unmerge'
  target:CleanupTarget
  onClose:()=>void
  onApply:(region:GridRegion,cells:Map<string,Cell>,headerRows:number)=>Promise<void>
}){
  const narrower=mode==='duplicates'&&Boolean(target.block&&target.blockCells&&
    (target.block.startColumn<target.region.startColumn||target.block.endColumn>target.region.endColumn))
  const [expand,setExpand]=useState(narrower)
  const [header,setHeader]=useState(target.headerRows>0)
  const [busy,setBusy]=useState(false)
  const dialog=useDialog<HTMLElement>(onClose)
  const region=expand&&target.block?target.block:target.region
  const cells=expand&&target.blockCells?target.blockCells:target.cells
  const headerRows=mode==='duplicates'&&header?1:0

  const duplicates=useMemo(()=>{
    if(mode!=='duplicates')return undefined
    const seen=new Set<string>(),removed:Array<{row:number;text:string}>=[]
    for(let row=region.startRow+headerRows;row<=region.endRow;row+=1){
      const parts:string[]=[]
      for(let column=region.startColumn;column<=region.endColumn;column+=1)parts.push(cleanupText(cells.get(cellKey(row,column))))
      const signature=parts.join(' ')
      if(seen.has(signature)){removed.push({row,text:parts.filter(Boolean).join(' · ')});continue}
      seen.add(signature)
    }
    return removed
  },[mode,region,cells,headerRows])

  const subtotals=useMemo(()=>mode==='subtotals'?planRemoveSubtotals(cells,region).rows:undefined,[mode,cells,region])

  const trims=useMemo(()=>{
    if(mode!=='trim')return undefined
    return trimWhitespace(cells,region).writes.map(write=>({
      row:write.row,column:write.column,
      before:cleanupText(cells.get(cellKey(write.row,write.column))),
      after:String(write.value??''),
    }))
  },[mode,region,cells])

  const numbers=useMemo(()=>{
    if(mode!=='numbers')return undefined
    return convertTextNumbers(cells,region).writes.map(write=>({
      row:write.row,column:write.column,
      before:cleanupText(cells.get(cellKey(write.row,write.column))),
      after:String(write.value??''),
    }))
  },[mode,region,cells])

  const unmerges=useMemo(()=>{
    if(mode!=='unmerge')return undefined
    const result=unmergeAndFill(cells,region)
    const seen=new Set<string>(),rows:Array<{row:number;column:number;before:string;after:string}>=[]
    for(const write of result.writes){
      const range=cellMerge(cells.get(cellKey(write.row,write.column)))
      if(!range)continue
      const key=`${range.startRow}:${range.startColumn}`
      if(seen.has(key))continue
      seen.add(key)
      rows.push({row:range.startRow,column:range.startColumn,
        before:`${address(range.startRow,range.startColumn)}:${address(range.endRow,range.endColumn)}`,
        after:cleanupText(cells.get(cellKey(range.startRow,range.startColumn)))||'(빈 값)'})
    }
    return {rows,filled:result.filled,kept:result.kept}
  },[mode,region,cells])

  const count=mode==='duplicates'?(duplicates?.length??0):mode==='subtotals'?(subtotals?.length??0):mode==='numbers'?(numbers?.length??0):mode==='unmerge'?(unmerges?.rows.length??0):(trims?.length??0)
  const apply=async()=>{
    setBusy(true)
    try{await onApply(region,cells,headerRows);onClose()}
    catch(error){alert(error instanceof Error?error.message:'정리하지 못했습니다.')}
    finally{setBusy(false)}
  }
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal cleanup-modal" ref={dialog as React.RefObject<never>} role="dialog" aria-modal="true" aria-label={mode==='unmerge'?'병합 해제하고 채우기':mode==='numbers'?'텍스트로 저장된 숫자':mode==='duplicates'?'중복 항목 삭제':mode==='subtotals'?'부분합 제거':'공백 제거'}>
      <h2>{mode==='trim'||mode==='numbers'?<Eraser/>:<Rows3/>} {mode==='unmerge'?'병합 해제하고 채우기':mode==='numbers'?'텍스트로 저장된 숫자':mode==='duplicates'?'중복 항목 삭제':mode==='subtotals'?'부분합 제거':'공백 제거'}</h2>
      <p>{label(region)} 범위를 정리합니다.</p>
      {narrower&&<label className="cleanup-expand">
        <input type="checkbox" aria-label="표 전체로 확장" checked={expand} onChange={event=>setExpand(event.target.checked)}/>
        <span>
          <strong>표 전체({label(target.block!)})로 확장</strong>
          {mode==='duplicates'&&<em><AlertTriangle/> 선택한 열만 정리하면 남은 행이 위로 올라가면서 옆 열과 어긋납니다.</em>}
        </span>
      </label>}
      {mode==='duplicates'&&<label className="cleanup-check">
        <input type="checkbox" aria-label="첫 행은 머리글" checked={header} onChange={event=>setHeader(event.target.checked)}/> 첫 행은 머리글로 유지
      </label>}
      <div className="cleanup-preview">
        {count===0
          ?<p className="cleanup-empty">{mode==='unmerge'?'이 범위에 병합된 셀이 없습니다.':mode==='numbers'?'글자로 담긴 숫자가 없습니다.':mode==='duplicates'?'중복된 행이 없습니다.':mode==='subtotals'?'이 표에는 부분합 행이 없습니다.':'제거할 공백이 없습니다.'}</p>
          :mode==='subtotals'
            ?<ul>{subtotals!.slice(0,PREVIEW_ROWS).map(item=><li key={item.row}><b>{item.row}행</b><span>{item.label||'(소계 행)'}</span></li>)}</ul>
          :mode==='duplicates'
            ?<ul>{duplicates!.slice(0,PREVIEW_ROWS).map(item=><li key={item.row}><b>{item.row}행</b><span>{item.text||'(빈 행)'}</span></li>)}</ul>
          :mode==='unmerge'
            ?<ul>{unmerges!.rows.slice(0,PREVIEW_ROWS).map(item=><li key={`${item.row}:${item.column}`}>
              <b>{item.before}</b><span><code>{item.after}</code></span>
            </li>)}</ul>
            :<ul>{(mode==='numbers'?numbers!:trims!).slice(0,PREVIEW_ROWS).map(item=><li key={`${item.row}:${item.column}`}>
              <b>{address(item.row,item.column)}</b><span><code>{item.before}</code> → <code>{item.after}</code></span>
            </li>)}</ul>}
      </div>
      {count>0&&<p className="cleanup-summary">
        {mode==='unmerge'?`병합 ${count.toLocaleString()}개를 풀고 ${(unmerges?.filled??0).toLocaleString()}칸을 채웁니다.${unmerges?.kept?` 이미 값이 있는 ${unmerges.kept.toLocaleString()}칸은 그대로 둡니다.`:''} 병합된 표는 정렬도 피벗도 되지 않습니다.`:mode==='numbers'?`글자로 담긴 숫자 ${count.toLocaleString()}칸을 숫자로 바꿉니다. 지금은 =SUM 이 이 칸들을 빼고 셈합니다. 칸에 담기는 값은 아래와 같고, 보이는 모양은 ₩ 와 % 를 서식으로 옮겨 그대로 둡니다.`:mode==='duplicates'?`중복된 ${count.toLocaleString()}개 행을 삭제합니다.`:mode==='subtotals'?`부분합 ${count.toLocaleString()}개 행을 지우고 그룹을 풉니다.`:`${count.toLocaleString()}개 셀의 공백을 정리합니다.`}
        {count>PREVIEW_ROWS?` 위에는 앞의 ${PREVIEW_ROWS}건만 표시했습니다.`:''}
      </p>}
      <div className="modal-actions">
        <button onClick={onClose}>취소</button>
        <button className="primary" disabled={busy||count===0} onClick={()=>void apply()}>{mode==='unmerge'?'해제하고 채우기':mode==='numbers'?'숫자로 바꾸기':mode==='trim'?'정리':'삭제'}</button>
      </div>
    </section>
  </div>
}

/** The rows a dedupe would remove, exposed so callers can reuse the rule. */
export function duplicateWrites(cells:Map<string,Cell>,region:GridRegion,headerRows:number){
  return removeDuplicateRows(cells,region,headerRows)
}
