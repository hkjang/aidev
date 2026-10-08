import { Check, ChevronLeft, ChevronRight, Cloud, CloudOff, Copy, Eye, EyeOff, LayoutList, MoreHorizontal, Palette, Plus, Save, SquareArrowOutUpRight, Trash2 } from 'lucide-react'
import { useRef, useState } from 'react'
import { ContextMenu, type MenuItem } from './ContextMenu'
import { SelectionSummary } from './SelectionSummary'
import type { Sheet } from '../types'

const colors:Array<{value:string;label:string}>=[
  {value:'',label:'색상 없음'},
  {value:'#ef4444',label:'빨강'},
  {value:'#f59e0b',label:'주황'},
  {value:'#22c55e',label:'초록'},
  {value:'#06b6d4',label:'청록'},
  {value:'#3b82f6',label:'파랑'},
  {value:'#8b5cf6',label:'보라'},
  {value:'#ec4899',label:'분홍'},
]

type Props={
  sheets:Sheet[]
  activeSheetId:string
  /** Bumped by every mutation, so the summary re-reads what it needs. */
  version:number
  saveState:'saved'|'saving'|'offline'|'conflict'|'error'
  saveLabel:string
  readOnly?:boolean
  onStatusClick?:()=>void
  onSelect:(sheet:Sheet)=>void
  onCreate:()=>Promise<void>
  onRename:(sheet:Sheet,name:string)=>Promise<void>
  onDuplicate:(sheet:Sheet)=>Promise<void>
  onMove:(sheet:Sheet,position:number)=>Promise<void>
  onColor:(sheet:Sheet,color:string)=>Promise<void>
  onHidden:(sheet:Sheet,hidden:boolean)=>Promise<void>
  onDelete:(sheet:Sheet)=>Promise<void>
  onCopyTo?:(sheet:Sheet)=>void
  onManage?:()=>void
}

export function SheetTabs({sheets,activeSheetId,version,saveState,saveLabel,readOnly=false,onStatusClick,onSelect,onCreate,onRename,onDuplicate,onMove,onColor,onHidden,onDelete,onCopyTo,onManage}:Props){
  const scroller=useRef<HTMLDivElement>(null)
  const dragged=useRef<string|undefined>(undefined)
  // 메뉴에 무엇을 담을지는 열 때 정하고, 각 줄을 쓸 수 있는지는 그릴 때
  // 정한다. 항목을 통째로 굳혀 두면 여는 순간의 pending 이 그대로 남아,
  // 시트 작업이 끝난 뒤에도 메뉴가 죽은 채로 보인다. 사람은 닫았다 다시
  // 열어야 하는데 왜 그래야 하는지 알 수 없다.
  const [menu,setMenu]=useState<{x:number;y:number;sheetId?:string;label:string}>()
  const [renaming,setRenaming]=useState<string>(),[name,setName]=useState(''),[pending,setPending]=useState(false),[error,setError]=useState(''),[dropTarget,setDropTarget]=useState<string>(),[hiddenOpen,setHiddenOpen]=useState(false)
  const run=async(action:()=>Promise<void>)=>{setPending(true);setError('');try{await action();setRenaming(undefined)}catch(reason){setError(reason instanceof Error?reason.message:'시트 작업을 완료하지 못했습니다.')}finally{setPending(false)}}
  const startRename=(sheet:Sheet)=>{setRenaming(sheet.id);setName(sheet.name);setError('')}
  const submitRename=(sheet:Sheet)=>{const next=name.trim();if(!next||next===sheet.name){setRenaming(undefined);return}void run(()=>onRename(sheet,next))}
  const scroll=(direction:number)=>scroller.current?.scrollBy({left:direction*220,behavior:'smooth'})
  const visible=sheets.filter(sheet=>!sheet.hidden)
  const hidden=sheets.filter(sheet=>sheet.hidden)
  // Dropping a tab moves it in front of the tab it was dropped on, matching the
  // drag behaviour of a spreadsheet tab strip.
  const drop=(target:Sheet)=>{
    const sourceID=dragged.current
    dragged.current=undefined
    setDropTarget(undefined)
    if(!sourceID||sourceID===target.id)return
    const source=sheets.find(sheet=>sheet.id===sourceID)
    if(!source)return
    void run(()=>onMove(source,target.position))
  }

  /** The sheet menu is the same whether it is opened by right click or button. */
  const sheetMenuItems=(sheet:Sheet):MenuItem[]=>[
    {kind:'label',label:sheet.name},
    {kind:'item',label:'이름 변경',icon:<Save/>,disabled:readOnly,onSelect:()=>startRename(sheet)},
    {kind:'item',label:'복제',icon:<Copy/>,disabled:pending||readOnly,onSelect:()=>void run(()=>onDuplicate(sheet))},
    ...(onCopyTo?[{kind:'item',label:'다른 워크북으로 복사…',icon:<SquareArrowOutUpRight/>,onSelect:()=>onCopyTo(sheet)} as MenuItem]:[]),
    {kind:'separator'},
    {kind:'item',label:'왼쪽으로 이동',icon:<ChevronLeft/>,disabled:pending||readOnly||sheet.position===0,onSelect:()=>void run(()=>onMove(sheet,sheet.position-1))},
    {kind:'item',label:'오른쪽으로 이동',icon:<ChevronRight/>,disabled:pending||readOnly||sheet.position===sheets.length-1,onSelect:()=>void run(()=>onMove(sheet,sheet.position+1))},
    {kind:'submenu',label:'탭 색상',icon:<Palette/>,disabled:readOnly,items:colors.map(color=>({
      kind:'item',label:color.label,checked:(sheet.color??'')===color.value,
      icon:color.value?<i className="sheet-color-dot" style={{background:color.value}}/>:undefined,
      onSelect:()=>void run(()=>onColor(sheet,color.value)),
    }))},
    {kind:'separator'},
    {kind:'item',label:'시트 숨기기',icon:<EyeOff/>,disabled:pending||readOnly||visible.length===1,onSelect:()=>void run(()=>onHidden(sheet,true))},
    {kind:'item',label:'시트 삭제',icon:<Trash2/>,danger:true,disabled:pending||readOnly||sheets.length===1,onSelect:()=>{
      if(window.confirm(`'${sheet.name}' 시트와 모든 셀을 삭제할까요?`))void run(()=>onDelete(sheet))
    }},
    ...(onManage?[{kind:'separator'} as MenuItem,{kind:'item',label:'모든 시트 관리…',icon:<LayoutList/>,onSelect:onManage} as MenuItem]:[]),
  ]
  // Right clicking the empty part of the strip offers the strip level actions.
  const stripMenuItems=():MenuItem[]=>[
    {kind:'item',label:'새 시트',icon:<Plus/>,shortcut:'Shift+F11',disabled:readOnly||pending,onSelect:()=>void run(onCreate)},
    ...(onManage?[{kind:'item',label:'모든 시트 관리…',icon:<LayoutList/>,onSelect:onManage} as MenuItem]:[]),
    ...(hidden.length>0?[{kind:'separator'} as MenuItem,{kind:'label',label:`숨긴 시트 ${hidden.length}개`} as MenuItem,
      ...hidden.map(sheet=>({kind:'item',label:sheet.name,icon:<Eye/>,disabled:readOnly,onSelect:()=>void run(()=>onHidden(sheet,false))} as MenuItem))]:[]),
  ]
  const openSheetMenu=(sheet:Sheet,event:{clientX:number;clientY:number})=>{
    if(sheet.id!==activeSheetId&&!sheet.hidden)onSelect(sheet)
    setMenu({x:event.clientX,y:event.clientY,sheetId:sheet.id,label:`${sheet.name} 시트 메뉴`})
  }

  return <div className="sheet-tabs" onContextMenu={event=>{
    if(event.target!==event.currentTarget&&!(event.target as HTMLElement).classList.contains('sheet-tab-scroller'))return
    event.preventDefault()
    setMenu({x:event.clientX,y:event.clientY,label:'시트 탭 메뉴'})
  }}>
    <button className="tab-nav" onClick={()=>scroll(-1)} aria-label="시트 탭 왼쪽으로"><ChevronLeft/></button>
    <button className="tab-nav" onClick={()=>scroll(1)} aria-label="시트 탭 오른쪽으로"><ChevronRight/></button>
    {!readOnly&&<button className="tab-add" disabled={pending} onClick={()=>run(onCreate)} aria-label="시트 추가"><Plus/></button>}
    {onManage&&<button className="tab-nav" onClick={onManage} aria-label="모든 시트 관리" title="모든 시트 관리"><LayoutList/></button>}
    <div className="sheet-tab-scroller" ref={scroller}>
      {visible.map(sheet=><div className={`sheet-tab-wrap${dropTarget===sheet.id?' drop-target':''}`} key={sheet.id}
        draggable={!readOnly&&renaming!==sheet.id}
        onDragStart={()=>{dragged.current=sheet.id}}
        onDragOver={event=>{if(dragged.current&&dragged.current!==sheet.id){event.preventDefault();setDropTarget(sheet.id)}}}
        onDragLeave={()=>setDropTarget(current=>current===sheet.id?undefined:current)}
        onDrop={event=>{event.preventDefault();drop(sheet)}}
        onDragEnd={()=>{dragged.current=undefined;setDropTarget(undefined)}}
        onContextMenu={event=>{event.preventDefault();event.stopPropagation();openSheetMenu(sheet,event)}}>
        {renaming===sheet.id?<form className="sheet-tab-rename" onSubmit={event=>{event.preventDefault();submitRename(sheet)}}><input autoFocus value={name} maxLength={100} aria-label="시트 이름" onChange={event=>setName(event.target.value)} onKeyDown={event=>{if(event.key==='Escape'){event.preventDefault();setRenaming(undefined)}}}/><button disabled={pending} aria-label="시트 이름 저장"><Check/></button></form>:<button className={`sheet-tab-main ${sheet.id===activeSheetId?'active':''}`} onClick={()=>onSelect(sheet)} onDoubleClick={()=>{if(!readOnly)startRename(sheet)}}><i className={sheet.color?'':'empty'} style={sheet.color?{background:sheet.color}:undefined}/><span>{sheet.name}</span></button>}
        {sheet.id===activeSheetId&&renaming!==sheet.id&&<button className="sheet-tab-menu-trigger" aria-label={`${sheet.name} 시트 메뉴`} aria-haspopup="menu"
          onClick={event=>{const rect=event.currentTarget.getBoundingClientRect();openSheetMenu(sheet,{clientX:rect.left,clientY:rect.top})}}><MoreHorizontal/></button>}
      </div>)}
    </div>
    {hidden.length>0&&<div className="sheet-hidden-group">
      <button className="sheet-hidden-trigger" aria-expanded={hiddenOpen} onClick={()=>setHiddenOpen(current=>!current)}><EyeOff/> 숨긴 시트 {hidden.length}</button>
      {hiddenOpen&&<div className="sheet-hidden-list" role="menu">
        {hidden.map(sheet=><div key={sheet.id}>
          <span><i className={sheet.color?'':'empty'} style={sheet.color?{background:sheet.color}:undefined}/>{sheet.name}</span>
          <button disabled={pending||readOnly} aria-label={`${sheet.name} 숨김 해제`} onClick={()=>void run(async()=>{await onHidden(sheet,false);setHiddenOpen(false)})}><Eye/> 표시</button>
        </div>)}
      </div>}
    </div>}
    {menu&&(()=>{
      // 자리 번호도 그릴 때 다시 본다. 열 때의 시트를 그대로 들고 있으면
      // 옆 시트가 움직인 뒤에도 옛 자리로 "왼쪽으로 이동" 을 판단한다.
      const target=menu.sheetId?sheets.find(sheet=>sheet.id===menu.sheetId):undefined
      if(menu.sheetId&&!target)return null
      return <ContextMenu x={menu.x} y={menu.y} items={target?sheetMenuItems(target):stripMenuItems()} label={menu.label} onClose={()=>setMenu(undefined)}/>
    })()}
    {error&&<span className="sheet-tabs-error">{error}</span>}
    <SelectionSummary sheetId={activeSheetId} version={version}/>
    <button className="sheet-status" disabled={!onStatusClick} onClick={onStatusClick}>{saveState==='offline'?<CloudOff/>:<Cloud/>} {saveLabel}</button>
  </div>
}
