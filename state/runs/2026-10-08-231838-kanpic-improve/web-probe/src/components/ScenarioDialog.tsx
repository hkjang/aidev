import { Layers,Plus } from 'lucide-react'
import { useEffect,useRef,useState } from 'react'
import { useDialog } from '../lib/useDialog'
import type { Scenario,ScenarioComparison } from '../types'
import './ScenarioDialog.css'

const looksLikeCell=(text:string)=>/^[A-Za-z]{1,3}[0-9]{1,7}$/.test(text.trim())
const show=(value:number|null)=>value==null?'—':Number(value.toPrecision(12)).toLocaleString('ko-KR',{maximumFractionDigits:10})
// 가정은 "B1=12000" 처럼 한 줄에 하나씩 적는다. 사람이 표를 보며 옮겨 적기
// 좋은 꼴이다.
const parseInputs=(text:string)=>text.split('\n').flatMap(line=>{
  const [cell,value]=line.split('=')
  if(cell===undefined||value===undefined)return []
  const number=Number(value.replace(/,/g,'').trim())
  if(!looksLikeCell(cell)||!Number.isFinite(number))return []
  return [{cell:cell.trim().toUpperCase(),value:number}]
})
const formatInputs=(item:Scenario)=>item.inputs.map(input=>`${input.cell}=${input.value ?? ''}`).join('\n')

export function ScenarioDialog({scenarios,activeSheetId,defaultTarget,onClose,onCreate,onUpdate,onDelete,onCompare}:{
  scenarios:Scenario[]
  activeSheetId:string
  defaultTarget:string
  onClose:()=>void
  onCreate:(input:Record<string,unknown>)=>Promise<Scenario>
  onUpdate:(id:string,input:Record<string,unknown>)=>Promise<Scenario>
  onDelete:(item:Scenario)=>Promise<void>
  onCompare:(targets:string[])=>Promise<ScenarioComparison>
}){
  const [selectedId,setSelectedId]=useState<string>()
  const [name,setName]=useState('')
  const [inputText,setInputText]=useState('')
  const [targets,setTargets]=useState(defaultTarget)
  const [comparison,setComparison]=useState<ScenarioComparison>()
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  const dialog=useDialog<HTMLElement>(onClose)
  const resultRef=useRef<HTMLElement>(null)
  useEffect(()=>{if(comparison)resultRef.current?.scrollIntoView({block:'nearest',behavior:'smooth'})},[comparison])

  const selected=scenarios.find(item=>item.id===selectedId)
  const choose=(item?:Scenario)=>{setSelectedId(item?.id);setName(item?.name??'');setInputText(item?formatInputs(item):'')}
  const inputs=parseInputs(inputText)
  const targetList=targets.split(/[\s,]+/).map(part=>part.trim().toUpperCase()).filter(looksLikeCell)
  const save=async()=>{
    if(!name.trim())return setError('시나리오 이름을 적으세요.')
    if(inputs.length===0)return setError('가정을 한 줄에 하나씩 B1=12000 처럼 적으세요.')
    setBusy(true);setError('')
    try{
      const body={name:name.trim(),sheet_id:activeSheetId,inputs}
      const saved=selected?await onUpdate(selected.id,{...body,expected_revision:selected.revision}):await onCreate(body)
      choose(saved)
    }catch(problem){setError(problem instanceof Error?problem.message:'시나리오를 저장하지 못했습니다.')}
    finally{setBusy(false)}
  }
  const remove=async(item:Scenario)=>{
    if(!confirm(`${item.name} 시나리오를 삭제할까요?`))return
    setBusy(true);setError('')
    try{await onDelete(item);choose()}catch(problem){setError(problem instanceof Error?problem.message:'삭제하지 못했습니다.')}
    finally{setBusy(false)}
  }
  const compare=async()=>{
    if(targetList.length===0)return setError('견줄 결과 셀을 적으세요.')
    setBusy(true);setError('');setComparison(undefined)
    try{setComparison(await onCompare(targetList))}
    catch(problem){setError(problem instanceof Error?problem.message:'견주지 못했습니다.')}
    finally{setBusy(false)}
  }
  const failureFor=(rowIndex:number,target:string)=>comparison?.rows[rowIndex]?.failures?.find(item=>item.target.endsWith(`!${target}`)||item.target===target)
  return <div className="modal-backdrop"><div className="modal scenario-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="시나리오">
    <header><div><Layers/><div><h2>시나리오</h2><p>가정 한 벌에 이름을 붙여 두고 나란히 놓고 견줍니다. 시트는 그대로 둡니다.</p></div></div><button aria-label="시나리오 닫기" onClick={onClose}>×</button></header>
    <div className="scenario-body">
      <div className="scenario-layout">
        <aside>
          <button className={!selected?'active':''} onClick={()=>choose()}><Plus/> 새 시나리오</button>
          {scenarios.map(item=><button key={item.id} className={selected?.id===item.id?'active':''} onClick={()=>choose(item)}>
            <span>{item.name}</span><em>가정 {item.inputs.length}개 · r{item.revision}</em>
          </button>)}
        </aside>
        <section>
          <label>이름<input aria-label="시나리오 이름" value={name} onChange={event=>setName(event.target.value)} placeholder="낙관"/></label>
          <label>가정<textarea aria-label="시나리오 가정" rows={4} value={inputText} onChange={event=>setInputText(event.target.value)} placeholder={'B1=12000\nB3=1500'}/></label>
          <p className="scenario-hint">한 줄에 하나씩 <code>셀=값</code> 으로 적습니다. 지금 읽은 가정은 {inputs.length}개입니다.</p>
          <div className="scenario-actions">
            {selected&&<button className="danger" disabled={busy} onClick={()=>remove(selected)}>삭제</button>}
            <span/>
            <button className="primary" disabled={busy||!name.trim()||inputs.length===0} onClick={save}>{busy?'저장 중…':'저장'}</button>
          </div>
        </section>
      </div>
      <section className="scenario-compare">
        <label>견줄 결과 셀<input aria-label="시나리오 결과 셀" value={targets} onChange={event=>setTargets(event.target.value)} placeholder="B4, B5"/></label>
        {error&&<p className="scenario-error" role="alert">{error}</p>}
      </section>
      {comparison&&<section className="scenario-result" ref={resultRef as React.RefObject<any>}>
        <table><tbody>
          <tr><th/>{targetList.map(target=><th key={target}>{target}</th>)}</tr>
          <tr className="scenario-now"><th>지금</th>{comparison.current.map((value,index)=><td key={index}>{show(value)}</td>)}</tr>
          {comparison.rows.map((row,rowIndex)=><tr key={row.name}>
            <th>{row.name}</th>
            {row.values.map((value,index)=>{
              const failure=failureFor(rowIndex,targetList[index])
              return <td key={index} className={failure?'scenario-failed':''} title={failure?.reason}>{failure?failure.reason:show(value)}</td>
            })}
          </tr>)}
        </tbody></table>
      </section>}
    </div>
    <div className="modal-actions"><span/><button className="secondary" onClick={onClose}>닫기</button><button className="primary" disabled={busy||scenarios.length===0} onClick={compare}>{busy?'셈하는 중…':'견주기'}</button></div>
  </div></div>
}
