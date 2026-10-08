import { Wand2 } from 'lucide-react'
import { useState } from 'react'
import { address } from '../lib/api'
import type { FillPlan } from '../lib/flashFillPlan'
import { useDialog } from '../lib/useDialog'
import './FlashFillDialog.css'

const PREVIEW_ROWS=8

export function FlashFillDialog({plan,column,onClose,onApply}:{
  plan:FillPlan
  column:number
  onClose:()=>void
  onApply:(plan:FillPlan)=>Promise<void>
}){
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  const dialog=useDialog<HTMLElement>(onClose)
  const apply=async()=>{
    setBusy(true);setError('')
    try{await onApply(plan);onClose()}
    catch(problem){setError(problem instanceof Error?problem.message:'값을 채우지 못했습니다.');setBusy(false)}
  }
  const columnName=address(1,column).replace(/\d+$/,'')
  return <div className="modal-backdrop"><div className="modal flash-fill-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="빠른 채우기">
    <header><div><Wand2/><div><h2>빠른 채우기</h2><p>손으로 채워 둔 칸을 보고 규칙을 알아내 {columnName}열의 남은 칸을 채웁니다.</p></div></div><button aria-label="빠른 채우기 닫기" onClick={onClose}>×</button></header>
    <div className="flash-fill-body">
      <section>
        <h3>본보기로 삼은 값</h3>
        <ul className="flash-fill-examples">
          {plan.examples.slice(0,PREVIEW_ROWS).map(example=><li key={example.row}><em>{example.row}행</em><span>{example.value}</span></li>)}
          {plan.examples.length>PREVIEW_ROWS&&<li className="more">외 {plan.examples.length-PREVIEW_ROWS}개</li>}
        </ul>
      </section>
      <section>
        <h3>채울 값 <small>{plan.writes.length.toLocaleString()}칸</small></h3>
        <ul className="flash-fill-writes">
          {plan.writes.slice(0,PREVIEW_ROWS).map(write=><li key={write.row}><em>{write.row}행</em><span>{write.value}</span></li>)}
          {plan.writes.length>PREVIEW_ROWS&&<li className="more">외 {(plan.writes.length-PREVIEW_ROWS).toLocaleString()}칸</li>}
        </ul>
      </section>
    </div>
    {plan.headerSkipped&&<p className="flash-fill-note" role="status">첫 줄은 머리글로 보고 본보기에서 뺐습니다.</p>}
    {plan.unreached>0&&<p className="flash-fill-note" role="status">규칙이 닿지 않는 {plan.unreached.toLocaleString()}칸은 비워 둡니다.</p>}
    <p className="flash-fill-note">이미 값이 있는 칸은 건드리지 않습니다. 채운 뒤에는 실행 취소로 되돌릴 수 있습니다.</p>
    {error&&<div className="flash-fill-error" role="alert">{error}</div>}
    <div className="modal-actions">
      <span/>
      <button className="secondary" onClick={onClose}>닫기</button>
      <button className="primary" disabled={busy} onClick={()=>void apply()}>{busy?'채우는 중…':`${plan.writes.length.toLocaleString()}칸 채우기`}</button>
    </div>
  </div></div>
}
