import { useMemo, useState } from 'react'
import { AlertTriangle, Sigma } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { address } from '../lib/api'
import { cleanupText } from '../lib/dataCleanup'
import { cellKey } from '../state/editor'
import { planSubtotals, subtotalName, type SubtotalAggregation, type SubtotalPlan } from '../lib/subtotal'
import type { Cell } from '../types'
import type { GridRegion } from '../lib/dataRegion'
import './SubtotalDialog.css'

const AGGREGATIONS:SubtotalAggregation[]=['sum','average','count','max','min']
const columnLabel=(column:number)=>address(1,column).replace(/\d+$/,'')

/**
 * Subtotals rewrite the block and push every later row down, so the dialog
 * settles what will happen — which groups, how many rows, and whether the
 * grouping column is even sorted — before anything moves.
 */
export function SubtotalDialog({region,cells,headerRows,occupiedBelow,onClose,onApply}:{
  region:GridRegion
  cells:Map<string,Cell>
  headerRows:number
  /** Rows under the block that the added rows would land on. */
  occupiedBelow:number
  onClose:()=>void
  onApply:(plan:SubtotalPlan)=>Promise<void>
}){
  const columns=useMemo(()=>{
    const result:Array<{column:number;name:string;numeric:boolean}>=[]
    for(let column=region.startColumn;column<=region.endColumn;column+=1){
      const head=headerRows>0?cleanupText(cells.get(cellKey(region.startRow,column))):''
      let numeric=false
      for(let row=region.startRow+headerRows;row<=region.endRow&&!numeric;row+=1)
        if(typeof cells.get(cellKey(row,column))?.value==='number')numeric=true
      result.push({column,name:head||`${columnLabel(column)}열`,numeric})
    }
    return result
  },[cells,region,headerRows])
  const [groupColumn,setGroupColumn]=useState(columns.find(item=>!item.numeric)?.column??region.startColumn)
  const [aggregation,setAggregation]=useState<SubtotalAggregation>('sum')
  const [values,setValues]=useState<number[]>(()=>columns.filter(item=>item.numeric).map(item=>item.column))
  const [grandTotal,setGrandTotal]=useState(true)
  const [busy,setBusy]=useState(false)
  const dialog=useDialog<HTMLElement>(onClose)
  const plan=useMemo(()=>planSubtotals(cells,region,{groupColumn,valueColumns:values,aggregation,headerRows,grandTotal}),
    [cells,region,groupColumn,values,aggregation,headerRows,grandTotal])
  const toggleValue=(column:number)=>setValues(current=>current.includes(column)?current.filter(item=>item!==column):[...current,column].sort((a,b)=>a-b))
  const apply=async()=>{
    setBusy(true)
    try{await onApply(plan);onClose()}
    catch(error){alert(error instanceof Error?error.message:'부분합을 넣지 못했습니다.')}
    finally{setBusy(false)}
  }
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal subtotal-modal" ref={dialog as React.RefObject<never>} role="dialog" aria-modal="true" aria-label="부분합">
      <h2><Sigma/> 부분합</h2>
      <p>{`${address(region.startRow,region.startColumn)}:${address(region.endRow,region.endColumn)}`} 범위를 한 열로 묶어 그룹마다 소계 행을 넣습니다.</p>
      <div className="subtotal-fields">
        <label>그룹 기준 열<select aria-label="그룹 기준 열" value={groupColumn} onChange={event=>setGroupColumn(Number(event.target.value))}>
          {columns.map(item=><option key={item.column} value={item.column}>{item.name}</option>)}
        </select></label>
        <label>집계<select aria-label="집계 함수" value={aggregation} onChange={event=>setAggregation(event.target.value as SubtotalAggregation)}>
          {AGGREGATIONS.map(item=><option key={item} value={item}>{subtotalName(item)}</option>)}
        </select></label>
      </div>
      <fieldset className="subtotal-values">
        <legend>집계할 열</legend>
        {columns.filter(item=>item.column!==groupColumn).map(item=>
          <label key={item.column}><input type="checkbox" aria-label={`${item.name} 집계`} checked={values.includes(item.column)} onChange={()=>toggleValue(item.column)}/> {item.name}</label>)}
      </fieldset>
      <label className="subtotal-check"><input type="checkbox" aria-label="전체 합계 행 추가" checked={grandTotal} onChange={event=>setGrandTotal(event.target.checked)}/> 마지막에 전체 {subtotalName(aggregation)} 행 추가</label>
      {plan.unsorted&&<p className="subtotal-warning"><AlertTriangle/> 그룹 기준 열이 정렬되어 있지 않아 같은 값이 여러 그룹으로 나뉩니다. 먼저 그 열로 정렬하세요.</p>}
      {occupiedBelow>0&&<p className="subtotal-warning"><AlertTriangle/> 표 아래 {occupiedBelow}개 행의 기존 값을 밀어내고 덮어씁니다.</p>}
      <p className="subtotal-summary">{plan.runs.length.toLocaleString()}개 그룹 · {plan.addedRows.toLocaleString()}개 행이 추가됩니다.{values.length===0?' 집계할 열을 하나 이상 고르세요.':''}</p>
      <div className="modal-actions">
        <button onClick={onClose}>취소</button>
        <button className="primary" disabled={busy||values.length===0||plan.runs.length===0} onClick={()=>void apply()}>부분합 넣기</button>
      </div>
    </section>
  </div>
}
