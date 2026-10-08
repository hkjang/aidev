import { useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent, type ReactNode } from 'react'
import './ResizableRightPanel.css'

const MIN_WIDTH=320
const MAX_WIDTH=960
const GRID_WIDTH=320

export type RightPanelKey='ai'|'automation'|'history'|'comments'|'conflicts'|'charts'|'pivots'|'stats'|'connections'|'presentations'

export const RIGHT_PANEL_CONFIG:Record<RightPanelKey,{label:string;defaultWidth:number}>={
  ai:{label:'AI 도우미',defaultWidth:460},
  automation:{label:'자동화',defaultWidth:430},
  history:{label:'버전 이력',defaultWidth:380},
  comments:{label:'댓글',defaultWidth:400},
  conflicts:{label:'편집 충돌',defaultWidth:440},
  charts:{label:'차트',defaultWidth:400},
  pivots:{label:'피벗',defaultWidth:420},
  stats:{label:'열 통계',defaultWidth:380},
  connections:{label:'데이터 연결',defaultWidth:400},
  presentations:{label:'프레젠테이션',defaultWidth:400},
}

function availableMaxWidth(){
  if(typeof window==='undefined')return MAX_WIDTH
  return Math.max(MIN_WIDTH,Math.min(MAX_WIDTH,window.innerWidth-GRID_WIDTH))
}
function clampWidth(width:number){return Math.min(availableMaxWidth(),Math.max(MIN_WIDTH,Math.round(width)))}

export function ResizableRightPanel({panelKey,children}:{panelKey:RightPanelKey;children:ReactNode}){
  const {label,defaultWidth}=RIGHT_PANEL_CONFIG[panelKey]
  const storageKey=`kanpic:right-panel-width:${panelKey}`
  const [width,setWidth]=useState(()=>{
    const saved=typeof window==='undefined'?null:window.localStorage.getItem(storageKey)
    const stored=saved===null?Number.NaN:Number(saved)
    return clampWidth(Number.isFinite(stored)?stored:defaultWidth)
  })
  const drag=useRef<{x:number;width:number}|undefined>(undefined)
  const updateWidth=useCallback((next:number)=>setWidth(clampWidth(next)),[])

  useEffect(()=>{window.localStorage.setItem(storageKey,String(width))},[storageKey,width])
  useEffect(()=>()=>document.body.classList.remove('right-panel-resizing'),[])
  useEffect(()=>{
    const resize=()=>updateWidth(width)
    window.addEventListener('resize',resize)
    return()=>window.removeEventListener('resize',resize)
  },[updateWidth,width])

  const startResize=(event:ReactPointerEvent<HTMLDivElement>)=>{
    drag.current={x:event.clientX,width}
    event.currentTarget.setPointerCapture(event.pointerId)
    document.body.classList.add('right-panel-resizing')
  }
  const moveResize=(event:ReactPointerEvent<HTMLDivElement>)=>{
    if(!drag.current)return
    updateWidth(drag.current.width+drag.current.x-event.clientX)
  }
  const stopResize=(event:ReactPointerEvent<HTMLDivElement>)=>{
    drag.current=undefined
    if(event.currentTarget.hasPointerCapture(event.pointerId))event.currentTarget.releasePointerCapture(event.pointerId)
    document.body.classList.remove('right-panel-resizing')
  }
  const reset=()=>updateWidth(defaultWidth)

  return <section className="right-panel-frame" style={{width}} data-panel-key={panelKey}>
    <div
      className="right-panel-resizer"
      role="separator"
      aria-label={`${label} 패널 너비 조절`}
      aria-orientation="vertical"
      aria-valuemin={MIN_WIDTH}
      aria-valuemax={availableMaxWidth()}
      aria-valuenow={width}
      tabIndex={0}
      title={`${label} 패널 · 드래그하거나 화살표 키로 너비 조절 · 두 번 클릭해 초기화`}
      onPointerDown={startResize}
      onPointerMove={moveResize}
      onPointerUp={stopResize}
      onPointerCancel={stopResize}
      onDoubleClick={reset}
      onKeyDown={event=>{
        if(event.key==='ArrowLeft'){event.preventDefault();updateWidth(width+16)}
        else if(event.key==='ArrowRight'){event.preventDefault();updateWidth(width-16)}
        else if(event.key==='Home'){event.preventDefault();updateWidth(MIN_WIDTH)}
        else if(event.key==='End'){event.preventDefault();updateWidth(availableMaxWidth())}
      }}
    ><span/><output aria-hidden="true">{width}px</output></div>
    <div className="right-panel-content">{children}</div>
  </section>
}
