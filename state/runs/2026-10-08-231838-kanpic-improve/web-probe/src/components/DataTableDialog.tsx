import { Grid3x3 } from 'lucide-react'
import { useEffect,useRef,useState } from 'react'
import { useDialog } from '../lib/useDialog'
import './DataTableDialog.css'

export type DataTableOutcome={
  values:Array<Array<number|null>>
  failures?:Array<{row:number;column:number;reason:string}>
}

const looksLikeCell=(text:string)=>/^[A-Za-z]{1,3}[0-9]{1,7}$/.test(text.trim())
// 가정은 쉼표나 줄바꿈으로 나누어 적는다. 사람이 어느 쪽으로 적든 받는다.
const parseValues=(text:string)=>text.split(/[\n,]/).map(part=>Number(part.replace(/,/g,'').trim())).filter(value=>Number.isFinite(value))
const show=(value:number|null)=>value==null?'':Number(value.toPrecision(12)).toLocaleString('ko-KR',{maximumFractionDigits:10})

export function DataTableDialog({defaultTarget,onClose,onCompute}:{
  defaultTarget:string
  onClose:()=>void
  onCompute:(input:{target:string;columnInput:string;columnValues:number[];rowInput:string;rowValues:number[]})=>Promise<DataTableOutcome>
}){
  const [target,setTarget]=useState(defaultTarget)
  const [columnInput,setColumnInput]=useState('')
  const [columnText,setColumnText]=useState('')
  const [rowInput,setRowInput]=useState('')
  const [rowText,setRowText]=useState('')
  const [outcome,setOutcome]=useState<DataTableOutcome>()
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  const dialog=useDialog<HTMLElement>(onClose)
  // 결과는 폼 아래에 붙으므로 창이 작으면 접힌 자리에 생긴다. 눌렀는데
  // 화면이 그대로면 사람은 안 된 줄 안다.
  const resultRef=useRef<HTMLTableSectionElement>(null)
  useEffect(()=>{if(outcome)resultRef.current?.scrollIntoView({block:'nearest',behavior:'smooth'})},[outcome])

  const columnValues=parseValues(columnText),rowValues=parseValues(rowText)
  const twoWay=looksLikeCell(rowInput)&&rowValues.length>0
  const ready=looksLikeCell(target)&&looksLikeCell(columnInput)&&columnValues.length>0
  const compute=async()=>{
    setBusy(true);setError('');setOutcome(undefined)
    try{
      setOutcome(await onCompute({
        target:target.trim().toUpperCase(),
        columnInput:columnInput.trim().toUpperCase(),columnValues,
        rowInput:twoWay?rowInput.trim().toUpperCase():'',rowValues:twoWay?rowValues:[],
      }))
    }catch(problem){
      setError(problem instanceof Error?problem.message:'데이터 표를 만들지 못했습니다.')
    }finally{setBusy(false)}
  }
  const failureAt=(row:number,column:number)=>outcome?.failures?.find(item=>item.row===row&&item.column===column)
  return <div className="modal-backdrop"><div className="modal data-table-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="데이터 표">
    <header><div><Grid3x3/><div><h2>데이터 표</h2><p>가정을 하나씩 넣어 보며 결과가 어떻게 달라지는지 한 번에 봅니다. 시트는 그대로 둡니다.</p></div></div><button aria-label="데이터 표 닫기" onClick={onClose}>×</button></header>
    <div className="data-table-body"><section className="data-table-form">
      <label>결과 셀<input aria-label="데이터 표 결과 셀" value={target} onChange={event=>setTarget(event.target.value)} placeholder="B3"/></label>
      <label>세로로 바꿀 셀<input aria-label="데이터 표 세로 입력 셀" value={columnInput} onChange={event=>setColumnInput(event.target.value)} placeholder="B2"/></label>
      <label>세로 가정<textarea aria-label="데이터 표 세로 가정" rows={3} value={columnText} onChange={event=>setColumnText(event.target.value)} placeholder="0.03, 0.04, 0.05"/></label>
      <label>가로로 바꿀 셀 (선택)<input aria-label="데이터 표 가로 입력 셀" value={rowInput} onChange={event=>setRowInput(event.target.value)} placeholder="B1"/></label>
      <label>가로 가정 (선택)<textarea aria-label="데이터 표 가로 가정" rows={2} value={rowText} onChange={event=>setRowText(event.target.value)} placeholder="1000, 2000"/></label>
      <p className="data-table-hint">가정은 쉼표나 줄바꿈으로 나누어 적습니다. 가로 쪽을 비우면 한 방향 표가 됩니다.</p>
      {error&&<p className="data-table-error" role="alert">{error}</p>}
    </section>
    {outcome&&<section className="data-table-result" ref={resultRef as React.RefObject<any>}>
      <table><tbody>
        {twoWay&&<tr><th/>{rowValues.map((value,index)=><th key={index}>{show(value)}</th>)}</tr>}
        {outcome.values.map((line,rowIndex)=><tr key={rowIndex}>
          <th>{show(columnValues[rowIndex] ?? null)}</th>
          {line.map((value,columnIndex)=>{
            const failure=failureAt(rowIndex,columnIndex)
            // 셈하지 못한 자리는 비워 두고 까닭을 적는다. 0 으로 채우면
            // 사람은 그것을 답으로 읽는다.
            return <td key={columnIndex} className={failure?'data-table-failed':''} title={failure?.reason}>{failure?failure.reason:show(value)}</td>
          })}
        </tr>)}
      </tbody></table>
    </section>}
    </div>
    <div className="modal-actions"><span/><button className="secondary" onClick={onClose}>닫기</button><button className="primary" disabled={busy||!ready} onClick={compute}>{busy?'셈하는 중…':'표 만들기'}</button></div>
  </div></div>
}
