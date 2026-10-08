import { useEffect, useState } from 'react'
import { ContextMenu, type MenuItem } from './ContextMenu'
import { api, address } from '../lib/api'
import { DEFAULT_SUMMARY, formatStat, shouldSummarize, SUMMARY_LABELS, summarizeSelection, type SummaryKey } from '../lib/selectionSummary'
import { selectedBounds, useEditorStore } from '../state/editor'
import type { Cell } from '../types'

const STORAGE_KEY='kanpic.selection-summary'
/** Beyond this the summary is not worth the read, and neither is the answer. */
const MAX_SUMMARY_CELLS=100_000

function storedChoice():SummaryKey[]{
  try{
    const raw=JSON.parse(localStorage.getItem(STORAGE_KEY)??'null')
    if(Array.isArray(raw)&&raw.every(item=>item in SUMMARY_LABELS)&&raw.length>0)return raw as SummaryKey[]
  }catch{/* a browser that blocks storage just gets the default */}
  return DEFAULT_SUMMARY
}

/**
 * The running total of whatever is selected, in the corner every spreadsheet
 * keeps it. The numbers come from the server rather than the loaded tiles: the
 * grid only holds the rows on screen, so summing what is in memory would put a
 * confidently wrong total under a large selection.
 */
export function SelectionSummary({sheetId,version}:{sheetId?:string;version:number}){
  const [chosen,setChosen]=useState<SummaryKey[]>(storedChoice)
  const [menu,setMenu]=useState<{x:number;y:number}>()
  const [cells,setCells]=useState<Cell[]>()
  const [pending,setPending]=useState(false)
  const activeRow=useEditorStore(state=>state.activeRow)
  const activeColumn=useEditorStore(state=>state.activeColumn)
  const anchorRow=useEditorStore(state=>state.anchorRow)
  const anchorColumn=useEditorStore(state=>state.anchorColumn)
  const bounds=selectedBounds({activeRow,activeColumn,anchorRow,anchorColumn})
  const cellCount=(bounds.endRow-bounds.startRow+1)*(bounds.endColumn-bounds.startColumn+1)
  const range=`${address(bounds.startRow,bounds.startColumn)}:${address(bounds.endRow,bounds.endColumn)}`
  const tooLarge=cellCount>MAX_SUMMARY_CELLS
  useEffect(()=>{
    if(!sheetId||cellCount<2||tooLarge){setCells(undefined);return}
    const controller=new AbortController()
    setPending(true)
    // The selection changes as the pointer drags, so the read waits for it to
    // settle rather than firing on every intermediate rectangle.
    const timer=window.setTimeout(()=>{
      api<{items:Cell[]}>(`/api/v1/sheets/${sheetId}/ranges/${range}`,{signal:controller.signal})
        .then(result=>{setCells(result.items);setPending(false)})
        .catch(()=>undefined)
    },220)
    return()=>{window.clearTimeout(timer);controller.abort()}
  },[sheetId,range,cellCount,tooLarge,version])
  const loaded=new Map((cells??[]).map(cell=>[`${cell.row}:${cell.column}`,cell]))
  const stats=summarizeSelection(loaded,bounds.startRow,bounds.startColumn,bounds.endRow,bounds.endColumn)
  const toggle=(key:SummaryKey)=>{
    const next=chosen.includes(key)?chosen.filter(item=>item!==key):[...chosen,key]
    if(next.length===0)return
    setChosen(next)
    try{localStorage.setItem(STORAGE_KEY,JSON.stringify(next))}catch{/* nothing to do */}
  }
  const items:MenuItem[]=(Object.keys(SUMMARY_LABELS) as SummaryKey[]).map(key=>({
    kind:'item',label:SUMMARY_LABELS[key],checked:chosen.includes(key),onSelect:()=>toggle(key),
  }))
  if(cellCount<2)return null
  if(tooLarge)return <span className="selection-summary as-text">선택 {cellCount.toLocaleString()}셀</span>
  if(!cells||pending)return <span className="selection-summary as-text">계산 중…</span>
  if(!shouldSummarize(stats,cellCount))return null
  return <>
    <button className="selection-summary" aria-label="선택 범위 요약" title="클릭해 표시할 항목을 고릅니다"
      onClick={event=>setMenu({x:event.clientX,y:event.clientY})}>
      {chosen.map(key=><span key={key}><small>{SUMMARY_LABELS[key]}</small> {formatStat(key,stats)}</span>)}
    </button>
    {menu&&<ContextMenu x={menu.x} y={menu.y} items={items} label="요약 항목" onClose={()=>setMenu(undefined)}/>}
  </>
}
