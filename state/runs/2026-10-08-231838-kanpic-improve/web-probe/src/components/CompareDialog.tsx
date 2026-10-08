import { useMemo, useState } from 'react'
import { GitCompare } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { compareLists, type CompareResult } from '../lib/compareLists'
import type { Cell, Sheet } from '../types'
import type { GridRegion } from '../lib/dataRegion'
import { address } from '../lib/api'
import './CompareDialog.css'

const PREVIEW_ROWS=8

export type CompareSource={sheet:Sheet;region:GridRegion;cells:Map<string,Cell>}

const columnName=(column:number)=>address(1,column).replace(/\d+$/,'')

function KeyColumn({label,region,value,onChange}:{label:string;region:GridRegion;value:number;onChange:(column:number)=>void}){
  const columns=[]
  for(let column=region.startColumn;column<=region.endColumn;column+=1)columns.push(column)
  return <label>{label}
    <select aria-label={label} value={value} onChange={event=>onChange(Number(event.target.value))}>
      {columns.map(column=><option value={column} key={column}>{columnName(column)}열</option>)}
    </select>
  </label>
}

function Bucket({title,rows,empty}:{title:string;rows:Array<{key:string;label:string;row:number}>;empty:string}){
  return <section className="compare-bucket">
    <h3>{title} <b>{rows.length.toLocaleString()}</b></h3>
    {rows.length===0
      ?<p className="compare-empty">{empty}</p>
      :<ul>{rows.slice(0,PREVIEW_ROWS).map(row=><li key={`${row.key}:${row.row}`}><b>{row.row}행</b><code>{row.label||'(빈 값)'}</code></li>)}
        {rows.length>PREVIEW_ROWS&&<li className="compare-more">… 그리고 {(rows.length-PREVIEW_ROWS).toLocaleString()}개 더</li>}</ul>}
  </section>
}

export function CompareDialog({left,right,onClose,onReport}:{
  left:CompareSource;right:CompareSource;onClose:()=>void
  onReport:(result:CompareResult,keys:{left:number;right:number},headerRows:number)=>Promise<void>
}){
  const dialog=useDialog<HTMLElement>(onClose)
  const [headerRows,setHeaderRows]=useState(1)
  const [leftKey,setLeftKey]=useState(left.region.startColumn)
  const [rightKey,setRightKey]=useState(right.region.startColumn)
  const [busy,setBusy]=useState(false)
  const result=useMemo(()=>compareLists(
    {cells:left.cells,region:left.region,keyColumn:leftKey,headerRows},
    {cells:right.cells,region:right.region,keyColumn:rightKey,headerRows},
  ),[left,right,leftKey,rightKey,headerRows])

  const report=async()=>{
    setBusy(true)
    try{await onReport(result,{left:leftKey,right:rightKey},headerRows)}
    finally{setBusy(false)}
  }
  const label=(source:CompareSource)=>`${source.sheet.name}!${address(source.region.startRow,source.region.startColumn)}:${address(source.region.endRow,source.region.endColumn)}`
  const nothing=result.onlyLeft.length===0&&result.onlyRight.length===0

  return <div className="modal-backdrop"><div className="modal compare-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="두 목록 비교">
    <header><div><GitCompare/><div><h2>두 목록 비교</h2><p>키 열을 맞춰 어느 쪽에만 있는 항목인지 찾습니다.</p></div></div><button aria-label="두 목록 비교 닫기" onClick={onClose}>×</button></header>
    <div className="compare-body">
      <div className="compare-ranges">
        <div><strong>왼쪽</strong><span>{label(left)}</span><KeyColumn label="왼쪽 키 열" region={left.region} value={leftKey} onChange={setLeftKey}/></div>
        <div><strong>오른쪽</strong><span>{label(right)}</span><KeyColumn label="오른쪽 키 열" region={right.region} value={rightKey} onChange={setRightKey}/></div>
      </div>
      <label className="compare-header-rows"><input type="checkbox" checked={headerRows>0} onChange={event=>setHeaderRows(event.target.checked?1:0)}/> 첫 행은 머리글</label>
      <p className="compare-counts">양쪽에 <b>{result.both.toLocaleString()}</b> · 왼쪽에만 <b>{result.onlyLeft.length.toLocaleString()}</b> · 오른쪽에만 <b>{result.onlyRight.length.toLocaleString()}</b></p>
      <div className="compare-buckets">
        <Bucket title="왼쪽에만" rows={result.onlyLeft} empty="왼쪽 항목은 모두 오른쪽에 있습니다."/>
        <Bucket title="오른쪽에만" rows={result.onlyRight} empty="오른쪽 항목은 모두 왼쪽에 있습니다."/>
      </div>
      {result.duplicated.length>0&&<p className="compare-note">같은 키가 거듭 나온 곳이 {result.duplicated.length.toLocaleString()}개 있습니다 ({result.duplicated.slice(0,3).map(item=>`${item.side==='left'?'왼쪽':'오른쪽'} ${item.label}×${item.count}`).join(', ')}). 대사에서는 이것 자체가 발견입니다.</p>}
      {(result.blank.left>0||result.blank.right>0)&&<p className="compare-note">키 칸이 비어 견주지 못한 행이 왼쪽 {result.blank.left.toLocaleString()}개, 오른쪽 {result.blank.right.toLocaleString()}개 있습니다.</p>}
      {nothing&&result.both>0&&<p className="compare-ok">두 목록이 키 기준으로 완전히 맞습니다.</p>}
    </div>
    <div className="modal-actions"><span/><button className="secondary" onClick={onClose}>닫기</button>
      <button className="primary" disabled={busy} onClick={()=>void report()}>{busy?'만드는 중…':'결과를 새 시트로'}</button></div>
  </div></div>
}
