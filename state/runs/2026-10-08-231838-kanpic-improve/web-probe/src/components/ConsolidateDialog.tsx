import { useMemo, useState } from 'react'
import { Layers } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { consolidate, CONSOLIDATE_LABELS, type ConsolidateFunction, type ConsolidateResult, type ConsolidateSource } from '../lib/consolidate'
import './ConsolidateDialog.css'

const PREVIEW_ROWS=6
const FUNCTIONS:ConsolidateFunction[]=['sum','count','average','max','min']

export function ConsolidateDialog({sources,skipped=[],onClose,onApply}:{
  sources:ConsolidateSource[];skipped?:string[];onClose:()=>void
  onApply:(result:ConsolidateResult,operation:ConsolidateFunction)=>Promise<void>
}){
  const dialog=useDialog<HTMLElement>(onClose)
  const [operation,setOperation]=useState<ConsolidateFunction>('sum')
  const [chosen,setChosen]=useState<string[]>(()=>sources.map(source=>source.sheetName))
  const [busy,setBusy]=useState(false)
  const picked=useMemo(()=>sources.filter(source=>chosen.includes(source.sheetName)),[sources,chosen])
  const result=useMemo(()=>consolidate(picked,operation),[picked,operation])
  const toggle=(name:string)=>setChosen(current=>current.includes(name)?current.filter(item=>item!==name):[...current,name])

  const apply=async()=>{setBusy(true);try{await onApply(result,operation)}finally{setBusy(false)}}
  const missingSheets=[...new Set(result.missing.map(item=>item.sheetName))]

  return <div className="modal-backdrop"><div className="modal consolidate-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="여러 시트 합치기">
    <header><div><Layers/><div><h2>여러 시트 합치기</h2><p>첫 행은 항목 이름, 첫 열은 줄 이름표로 보고 이름표를 맞춰 합칩니다.</p></div></div><button aria-label="여러 시트 합치기 닫기" onClick={onClose}>×</button></header>
    <div className="consolidate-body">
      <div className="consolidate-controls">
        <fieldset><legend>합칠 시트</legend>
          <div className="consolidate-sheets">{sources.map(source=>
            <label key={source.sheetName}><input type="checkbox" checked={chosen.includes(source.sheetName)} onChange={()=>toggle(source.sheetName)}/> {source.sheetName}</label>)}
          </div>
        </fieldset>
        <label className="consolidate-function">함수
          <select aria-label="합치는 함수" value={operation} onChange={event=>setOperation(event.target.value as ConsolidateFunction)}>
            {FUNCTIONS.map(item=><option value={item} key={item}>{CONSOLIDATE_LABELS[item]}</option>)}
          </select>
        </label>
      </div>
      {picked.length<2&&<p className="consolidate-note">시트를 두 개 이상 고르세요.</p>}
      {skipped.length>0&&<p className="consolidate-note">표를 찾지 못해 목록에 없는 시트: {skipped.join(', ')}. 첫 행에 항목 이름, 첫 열에 줄 이름표가 있어야 합니다.</p>}
      {result.mergedSheets.length>0&&<p className="consolidate-warn">병합된 셀이 있는 시트: {result.mergedSheets.join(', ')}. 병합된 이름표 열은 빈 이름표 행을 만들어 그 행이 통째로 빠집니다. <b>데이터 › 데이터 정리 › 병합 해제하고 채우기</b> 를 먼저 하세요.</p>}
      {result.skippedText>0&&<p className="consolidate-warn">숫자로 세지 않은 글자 칸이 {result.skippedText.toLocaleString()}개 있습니다. `=SUM` 과 같은 규칙으로 세므로 <b>"1,234"</b> 처럼 글자로 담긴 숫자는 빠집니다. <b>텍스트로 저장된 숫자</b> 로 먼저 고치세요.</p>}
      {missingSheets.length>0&&<p className="consolidate-note">어떤 시트에는 없는 이름표가 있습니다 ({result.missing.length.toLocaleString()}건). 없는 것은 0이 아니라 <b>세지 않은 것</b> 입니다.</p>}
      {result.rowLabels.length>0&&<table className="consolidate-preview"><thead><tr><th/>{result.columnLabels.map(label=><th key={label}>{label}</th>)}</tr></thead>
        <tbody>{result.rowLabels.slice(0,PREVIEW_ROWS).map((label,rowIndex)=><tr key={label}><th>{label}</th>
          {result.columnLabels.map((column,columnIndex)=><td key={column}>{formatCell(result.values.get(`${rowIndex}:${columnIndex}`))}</td>)}</tr>)}</tbody></table>}
      {result.rowLabels.length>PREVIEW_ROWS&&<p className="consolidate-note">… 그리고 {(result.rowLabels.length-PREVIEW_ROWS).toLocaleString()}줄 더</p>}
      {result.rowLabels.length===0&&picked.length>=2&&<p className="consolidate-note">이름표를 찾지 못했습니다. 첫 열에 줄 이름표가 있는지 확인하세요.</p>}
    </div>
    <div className="modal-actions"><span/><button className="secondary" onClick={onClose}>닫기</button>
      <button className="primary" disabled={busy||picked.length<2||result.rowLabels.length===0} onClick={()=>void apply()}>{busy?'만드는 중…':'결과를 새 시트로'}</button></div>
  </div></div>
}

const formatCell=(value?:number)=>value===undefined?'':Number.isInteger(value)?value.toLocaleString():value.toLocaleString(undefined,{maximumFractionDigits:2})
