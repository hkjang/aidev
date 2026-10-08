import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Check, Filter, Maximize2, RotateCcw, Trash2 } from 'lucide-react'
import { createPortal } from 'react-dom'
import { useEffect } from 'react'
import { api, address } from '../lib/api'
import { parseFilterRange } from '../lib/filter'
import { columnValues, withColumnValues, type FilterValue } from '../lib/columnFilter'
import type { Cell, FilterCriterion, FilterView, Slicer } from '../types'
import './SlicerOverlay.css'

const columnLabel=(column:number)=>address(1,column).replace(/\d+$/,'')

/**
 * A slicer is the filter a reader can use without knowing there is a filter:
 * a card sitting on the sheet with one column's values as buttons. Ticking a
 * value writes it into the filter view everybody shares, so the rows it hides
 * are hidden exactly the way the column filter menu hides them.
 */
function SlicerCard({slicer,view,sheetId,version,readOnly,onApply,onUpdate,onRemove}:{
  slicer:Slicer
  view:FilterView
  sheetId:string
  version:number
  readOnly:boolean
  onApply:(criteria:FilterCriterion[])=>Promise<void>
  onUpdate:(slicer:Slicer)=>Promise<void>
  onRemove:(slicer:Slicer)=>Promise<void>
}){
  const [position,setPosition]=useState(slicer.position)
  const [busy,setBusy]=useState(false)
  useEffect(()=>setPosition(slicer.position),[slicer.position])
  // The grid holds only the rows on screen, so the value list comes from the
  // server: a slicer built from loaded cells would quietly omit values.
  const range=useMemo(()=>{
    const bounds=parseFilterRange(view.range)
    return bounds?`${address(bounds.startRow,slicer.column)}:${address(bounds.endRow,slicer.column)}`:undefined
  },[view.range,slicer.column])
  const loaded=useQuery({
    queryKey:['slicer-values',sheetId,range,version,view.updated_at],
    queryFn:()=>api<{items:Cell[]}>(`/api/v1/sheets/${sheetId}/ranges/${range}`),
    enabled:Boolean(range),
  })
  const values=useMemo(()=>columnValues(loaded.data?.items??[],view,slicer.column),[loaded.data,view,slicer.column])
  const kept=values.filter(value=>value.checked).length
  const filtered=kept>0&&kept<values.length
  const title=slicer.title||`${columnLabel(slicer.column)}열`
  const toggle=async(target:FilterValue)=>{
    if(readOnly||busy)return
    setBusy(true)
    try{await onApply(withColumnValues(view,slicer.column,values.map(value=>value.label===target.label?{...value,checked:!value.checked}:value)))}
    finally{setBusy(false)}
  }
  const clear=async()=>{
    if(readOnly||busy)return
    setBusy(true)
    try{await onApply(withColumnValues(view,slicer.column,values.map(value=>({...value,checked:true}))))}
    finally{setBusy(false)}
  }
  const startGesture=(kind:'move'|'resize',event:React.PointerEvent)=>{
    if(readOnly)return
    event.preventDefault();event.stopPropagation()
    const startX=event.clientX,startY=event.clientY,start=position
    let latest=start
    const move=(next:PointerEvent)=>{
      const dx=next.clientX-startX,dy=next.clientY-startY
      latest=kind==='move'
        ?{...start,x:Math.round(Math.max(0,start.x+dx)),y:Math.round(Math.max(0,start.y+dy))}
        :{...start,width:Math.round(Math.max(140,Math.min(600,start.width+dx))),height:Math.round(Math.max(96,Math.min(640,start.height+dy)))}
      setPosition(latest)
    }
    const stop=()=>{
      window.removeEventListener('pointermove',move)
      if(JSON.stringify(latest)!==JSON.stringify(slicer.position))
        void onUpdate({...slicer,position:latest}).catch(error=>{setPosition(slicer.position);alert(error instanceof Error?error.message:'슬라이서 배치를 저장하지 못했습니다.')})
    }
    window.addEventListener('pointermove',move)
    window.addEventListener('pointerup',stop,{once:true})
  }
  return <article className="slicer-card" style={{left:position.x,top:position.y,width:position.width,height:position.height}} data-slicer-id={slicer.id} aria-label={`${title} 슬라이서`}>
    <header onPointerDown={event=>startGesture('move',event)}>
      <strong title={title}><Filter/>{title}</strong>
      <span>
        <button aria-label={`${title} 필터 지우기`} title="이 열 필터 지우기" disabled={readOnly||!filtered} onPointerDown={event=>event.stopPropagation()} onClick={()=>void clear()}><RotateCcw/></button>
        <button aria-label={`${title} 슬라이서 삭제`} title="슬라이서 삭제" disabled={readOnly} onPointerDown={event=>event.stopPropagation()} onClick={()=>{if(window.confirm(`'${title}' 슬라이서를 삭제할까요? 열 필터는 그대로 남습니다.`))void onRemove(slicer).catch(error=>alert(error instanceof Error?error.message:'슬라이서를 삭제하지 못했습니다.'))}}><Trash2/></button>
      </span>
    </header>
    <div className="slicer-values" role="group" aria-label={`${title} 값`}>
      {loaded.isPending?<p className="slicer-note">값을 읽는 중…</p>
        :values.length===0?<p className="slicer-note">이 열에는 거를 값이 없습니다.</p>
        :values.map(value=><button key={value.label} type="button" className={value.checked?'on':''} aria-pressed={value.checked} disabled={readOnly||busy} onClick={()=>void toggle(value)}>
          <span className="slicer-tick">{value.checked&&<Check/>}</span>
          <span className="slicer-label" title={value.label}>{value.label}</span>
          <span className="slicer-count">{value.count.toLocaleString()}</span>
        </button>)}
    </div>
    <footer>{filtered?`${kept.toLocaleString()}/${values.length.toLocaleString()}개 값 표시 중`:'모든 값 표시 중'}</footer>
    {!readOnly&&<button className="slicer-resize" aria-label={`${title} 슬라이서 크기 조정`} onPointerDown={event=>startGesture('resize',event)}><Maximize2/></button>}
  </article>
}

export function SlicerOverlay({slicers,views,sheetId,version,readOnly,onApply,onUpdate,onRemove}:{
  slicers:Slicer[]
  views:FilterView[]
  sheetId:string
  version:number
  readOnly:boolean
  onApply:(view:FilterView,criteria:FilterCriterion[])=>Promise<void>
  onUpdate:(slicer:Slicer)=>Promise<void>
  onRemove:(slicer:Slicer)=>Promise<void>
}){
  const [target,setTarget]=useState<Element|null>(null)
  useEffect(()=>setTarget(document.querySelector('.sheet-area')),[slicers.length,sheetId])
  if(!target||slicers.length===0)return null
  return createPortal(<div className="slicer-overlay-layer" aria-label="시트 슬라이서 레이어">
    {slicers.map(slicer=>{
      const view=views.find(item=>item.id===slicer.filter_view_id)
      if(!view)return null
      return <SlicerCard key={slicer.id} slicer={slicer} view={view} sheetId={sheetId} version={version} readOnly={readOnly}
        onApply={criteria=>onApply(view,criteria)} onUpdate={onUpdate} onRemove={onRemove}/>
    })}
  </div>,target)
}
