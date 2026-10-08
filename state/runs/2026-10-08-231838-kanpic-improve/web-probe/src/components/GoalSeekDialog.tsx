import { Target } from 'lucide-react'
import { useState } from 'react'
import { useDialog } from '../lib/useDialog'
import './GoalSeekDialog.css'

export type GoalSeekOutcome={
  value:number; result:number; before:number
  converged:boolean; iterations:number; reason?:string
}

const asNumber=(text:string)=>Number(text.replace(/,/g,'').trim())
const show=(value:number)=>{
  if(!Number.isFinite(value))return String(value)
  // 목표값 찾기의 답은 대개 딱 떨어지지 않는다. 반올림해서 보여 주되 값 자체는
  // 그대로 넣는다 — 보이는 것과 들어가는 것이 다르면 안 되므로 자릿수를 넉넉히
  // 남긴다.
  return Number(value.toPrecision(12)).toLocaleString('ko-KR',{maximumFractionDigits:10})
}
const looksLikeCell=(text:string)=>/^[A-Za-z]{1,3}[0-9]{1,7}$/.test(text.trim())

export function GoalSeekDialog({defaultTarget,canWrite,onClose,onSeek,onApply}:{
  defaultTarget:string
  canWrite:boolean
  onClose:()=>void
  onSeek:(input:{target:string;changing:string;goal:number})=>Promise<GoalSeekOutcome>
  onApply:(cell:string,value:number)=>Promise<void>
}){
  const [target,setTarget]=useState(defaultTarget)
  const [changing,setChanging]=useState('')
  const [goal,setGoal]=useState('')
  const [outcome,setOutcome]=useState<GoalSeekOutcome>()
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  const [applied,setApplied]=useState(false)
  const dialog=useDialog<HTMLElement>(onClose)

  const ready=looksLikeCell(target)&&looksLikeCell(changing)&&goal.trim()!==''&&Number.isFinite(asNumber(goal))
  const seek=async()=>{
    setBusy(true);setError('');setOutcome(undefined);setApplied(false)
    try{
      setOutcome(await onSeek({target:target.trim().toUpperCase(),changing:changing.trim().toUpperCase(),goal:asNumber(goal)}))
    }catch(problem){
      setError(problem instanceof Error?problem.message:'목표값을 찾지 못했습니다.')
    }finally{setBusy(false)}
  }
  const apply=async()=>{
    if(!outcome)return
    setBusy(true);setError('')
    try{await onApply(changing.trim().toUpperCase(),outcome.value);setApplied(true)}
    catch(problem){setError(problem instanceof Error?problem.message:'값을 넣지 못했습니다.')}
    finally{setBusy(false)}
  }

  return <div className="modal-backdrop"><div className="modal goal-seek-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="목표값 찾기">
    <header><div><Target/><div><h2>목표값 찾기</h2><p>수식이 원하는 값이 되려면 어떤 칸이 얼마여야 하는지 되짚어 찾습니다.</p></div></div><button aria-label="목표값 찾기 닫기" onClick={onClose}>×</button></header>
    <div className="goal-seek-fields">
      <label>수식 셀<input aria-label="수식 셀" value={target} placeholder="B4" onChange={event=>{setTarget(event.target.value);setOutcome(undefined)}}/></label>
      <label>찾는 값<input aria-label="찾는 값" value={goal} placeholder="900000" onChange={event=>{setGoal(event.target.value);setOutcome(undefined)}}/></label>
      <label>바꿀 셀<input aria-label="바꿀 셀" value={changing} placeholder="B2" onChange={event=>{setChanging(event.target.value);setOutcome(undefined)}}/></label>
    </div>
    <p className="goal-seek-note">바꿀 셀에는 값이 들어 있어야 합니다. 수식이 든 칸을 바꾸면 그 수식이 사라집니다.</p>
    {outcome&&<div className={`goal-seek-outcome${outcome.converged?'':' short'}`} role="status">
      {outcome.converged
        ?<>
          <strong>{changing.toUpperCase()} → <em>{show(outcome.value)}</em></strong>
          <strong className="goal-seek-effect">{target.toUpperCase()} → {show(outcome.result)}<small> (지금 {show(outcome.before)})</small></strong>
          <small>{outcome.iterations}번 계산해 찾았습니다.</small>
        </>
        :<>
          <strong>목표에 이르지 못했습니다.</strong>
          <small>{outcome.reason||'값이 목표에 가까워지지 않습니다.'} · 가장 가까운 값은 {show(outcome.result)} 입니다.</small>
        </>}
    </div>}
    {applied&&<div className="goal-seek-applied" role="status">{changing.toUpperCase()} 셀에 값을 넣었습니다.</div>}
    {error&&<div className="goal-seek-error" role="alert">{error}</div>}
    <div className="modal-actions">
      <span/>
      <button className="secondary" onClick={onClose}>닫기</button>
      {outcome?.converged&&canWrite&&<button className="primary" disabled={busy||applied} onClick={()=>void apply()}>{applied?'넣었습니다':'값 넣기'}</button>}
      <button className={outcome?.converged?'secondary':'primary'} disabled={busy||!ready} onClick={()=>void seek()}>{busy?'찾는 중…':'찾기'}</button>
    </div>
  </div></div>
}
