import { useMemo, useState } from 'react'
import { AlertTriangle, Table2 } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { cleanupText, detectDelimiter, splitLine, type SplitDelimiter } from '../lib/dataCleanup'
import type { Cell } from '../types'
import type { GridRegion } from '../lib/dataRegion'
import { cellKey } from '../state/editor'
import './SplitDialog.css'

const PRESETS:Array<{id:string;label:string;value:string}>=[
  {id:'auto',label:'자동 감지',value:'auto'},
  {id:'comma',label:'쉼표',value:','},
  {id:'semicolon',label:'세미콜론',value:';'},
  {id:'tab',label:'탭',value:'\t'},
  {id:'space',label:'공백',value:' '},
  {id:'custom',label:'맞춤',value:''},
]
const PREVIEW_ROWS=6
const separatorName=(value:string)=>value==='\t'?'탭':value===' '?'공백':value===','?'쉼표':value===';'?'세미콜론':`"${value}"`

/**
 * Splitting overwrites whatever sits to the right, so the preview exists to
 * answer the two questions that decide whether to go ahead: how the rows come
 * apart, and what the split is about to land on.
 */
export function SplitDialog({cells,region,onClose,onApply}:{
  cells:Map<string,Cell>
  region:GridRegion
  onClose:()=>void
  onApply:(delimiter:SplitDelimiter)=>Promise<void>
}){
  const [preset,setPreset]=useState('auto')
  const [custom,setCustom]=useState('')
  const [busy,setBusy]=useState(false)
  const dialog=useDialog<HTMLElement>(onClose)
  const texts=useMemo(()=>{
    const result:string[]=[]
    for(let row=region.startRow;row<=region.endRow;row+=1){
      const cell=cells.get(cellKey(row,region.startColumn))
      if(cell?.formula)continue
      result.push(cleanupText(cell))
    }
    return result
  },[cells,region])
  const chosen=preset==='custom'?custom:PRESETS.find(item=>item.id===preset)?.value??','
  const separator=chosen==='auto'?detectDelimiter(texts):chosen
  const rows=useMemo(()=>texts.slice(0,PREVIEW_ROWS).map(text=>text===''?[]:splitLine(text,separator).map(part=>part.trim())),[texts,separator])
  const width=rows.reduce((max,parts)=>Math.max(max,parts.length),0)
  const occupied=useMemo(()=>{
    for(let row=region.startRow;row<=region.endRow;row+=1)
      for(let column=region.startColumn+1;column<region.startColumn+width;column+=1)
        if(cleanupText(cells.get(cellKey(row,column)))!=='')return true
    return false
  },[cells,region,width])
  const apply=async()=>{
    setBusy(true)
    try{await onApply(preset==='custom'?custom:(PRESETS.find(item=>item.id===preset)?.value??',') as SplitDelimiter);onClose()}
    catch(error){alert(error instanceof Error?error.message:'분할하지 못했습니다.')}
    finally{setBusy(false)}
  }
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal split-modal" ref={dialog as React.RefObject<never>} role="dialog" aria-modal="true" aria-label="텍스트를 열로 분할">
      <h2><Table2/> 텍스트를 열로 분할</h2>
      <p>선택한 열의 값을 구분 기호로 나눠 오른쪽 열에 채웁니다.</p>
      <div className="split-presets" role="radiogroup" aria-label="구분 기호">
        {PRESETS.map(item=><button key={item.id} role="radio" aria-checked={preset===item.id} className={preset===item.id?'active':''} onClick={()=>setPreset(item.id)}>{item.label}</button>)}
      </div>
      {preset==='custom'&&<label className="split-custom">구분 기호<input aria-label="맞춤 구분 기호" value={custom} onChange={event=>setCustom(event.target.value)} placeholder="예: | 또는 ::"/></label>}
      <div className="split-preview">
        {width<2
          ?<p className="split-empty">{preset==='custom'&&custom===''?'구분 기호를 입력하세요.':`${separatorName(separator)}(으)로는 나눌 수 있는 값이 없습니다.`}</p>
          :<table><tbody>{rows.map((parts,index)=><tr key={index}>{Array.from({length:width},(_,column)=><td key={column}>{parts[column]??''}</td>)}</tr>)}</tbody></table>}
      </div>
      {width>=2&&<p className="split-summary">
        {chosen==='auto'?`자동 감지: ${separatorName(separator)} · `:''}{width}개 열로 나뉩니다.
        {occupied&&<span className="split-warning"><AlertTriangle/> 오른쪽 {width-1}개 열의 기존 값을 덮어씁니다.</span>}
      </p>}
      <div className="modal-actions">
        <button onClick={onClose}>취소</button>
        <button className="primary" disabled={busy||width<2} onClick={()=>void apply()}>분할</button>
      </div>
    </section>
  </div>
}
