import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { ChevronRight } from 'lucide-react'
import './ContextMenu.css'

export type MenuItem =
  | {kind:'separator'}
  | {kind:'label';label:string}
  | {kind:'item';label:string;shortcut?:string;icon?:ReactNode;disabled?:boolean;danger?:boolean;checked?:boolean;onSelect:()=>void}
  | {kind:'submenu';label:string;icon?:ReactNode;disabled?:boolean;items:MenuItem[]}

const MENU_WIDTH=252,MENU_MARGIN=8
export function selectableItems(items:MenuItem[]){return items.map((item,index)=>({item,index})).filter(entry=>(entry.item.kind==='item'||entry.item.kind==='submenu')&&!entry.item.disabled)}

/** Positions a menu inside the viewport, flipping it when it would overflow. */
function clampPosition(x:number,y:number,height:number){
  const left=Math.max(MENU_MARGIN,Math.min(x,window.innerWidth-MENU_WIDTH-MENU_MARGIN))
  const top=Math.max(MENU_MARGIN,Math.min(y,window.innerHeight-height-MENU_MARGIN))
  return{left,top}
}

/**
 * Places a submenu next to the row that opened it. The submenu is rendered in a
 * portal because the parent list scrolls: nesting it would clip the submenu and
 * turn a simple pointer move into a scroll hunt. It opens to the right and
 * flips to the left only when there is no room.
 */
export function submenuPosition(anchor:DOMRect,size:{width:number;height:number},viewport:{width:number;height:number}){
  const right=anchor.right-4
  const left=right+size.width+MENU_MARGIN<=viewport.width?right:Math.max(MENU_MARGIN,anchor.left-size.width+4)
  const top=Math.max(MENU_MARGIN,Math.min(anchor.top-5,viewport.height-size.height-MENU_MARGIN))
  return{left,top}
}

function Submenu({anchor,items,autoFocus,onClose,onBack}:{anchor:HTMLElement|null;items:MenuItem[];autoFocus:boolean;onClose:()=>void;onBack:()=>void}){
  const box=useRef<HTMLDivElement>(null)
  const [position,setPosition]=useState<{left:number;top:number}>()
  useLayoutEffect(()=>{
    if(!anchor||!box.current)return
    const size={width:box.current.offsetWidth,height:box.current.offsetHeight}
    setPosition(submenuPosition(anchor.getBoundingClientRect(),size,{width:window.innerWidth,height:window.innerHeight}))
  },[anchor,items])
  return createPortal(
    <div className="context-submenu" data-kanpic-menu ref={box} style={position?{left:position.left,top:position.top}:{visibility:'hidden'}} onContextMenu={event=>event.preventDefault()}>
      <MenuList items={items} onClose={onClose} onBack={onBack} autoFocus={autoFocus}/>
    </div>,
    document.body,
  )
}

function MenuList({items,onClose,autoFocus,label,onBack,onSwitch}:{items:MenuItem[];onClose:()=>void;autoFocus:boolean;label?:string;onBack?:()=>void;onSwitch?:(delta:1|-1)=>void}){
  const [active,setActive]=useState(()=>selectableItems(items)[0]?.index??-1)
  const [submenu,setSubmenu]=useState<{index:number;keyboard:boolean}>()
  const list=useRef<HTMLDivElement>(null),rows=useRef<Array<HTMLButtonElement|null>>([])
  useEffect(()=>{if(autoFocus)list.current?.focus()},[autoFocus])
  const movable=selectableItems(items)
  const move=(direction:1|-1)=>{
    if(movable.length===0)return
    const position=movable.findIndex(entry=>entry.index===active)
    const next=movable[(position+direction+movable.length*2)%movable.length]
    setActive(next.index);setSubmenu(undefined)
  }
  const closeSubmenu=()=>{setSubmenu(undefined);list.current?.focus()}
  const choose=(item:MenuItem,index:number,keyboard=false)=>{
    // Hovering already opens a submenu, so clicking the same row must keep it
    // open instead of toggling it shut under the pointer.
    if(item.kind==='submenu'){setSubmenu({index,keyboard});setActive(index);return}
    if(item.kind!=='item'||item.disabled)return
    onClose();item.onSelect()
  }
  return <div className="context-menu-list" ref={list} tabIndex={-1} role="menu" aria-label={label} onKeyDown={event=>{
    if(event.key==='ArrowDown'){move(1);event.preventDefault()}
    else if(event.key==='ArrowUp'){move(-1);event.preventDefault()}
    else if(event.key==='Home'){setActive(movable[0]?.index??-1);event.preventDefault()}
    else if(event.key==='End'){setActive(movable.at(-1)?.index??-1);event.preventDefault()}
    else if(event.key==='ArrowRight'){
      if(items[active]?.kind==='submenu')setSubmenu({index:active,keyboard:true})
      else if(onSwitch)onSwitch(1)
      else return
      event.preventDefault()
    }
    else if(event.key==='ArrowLeft'){
      if(submenu)closeSubmenu()
      else if(onBack)onBack()
      else if(onSwitch)onSwitch(-1)
      else return
      event.preventDefault()
    }
    else if(event.key==='Enter'||event.key===' '){const item=items[active];if(item)choose(item,active,true);event.preventDefault()}
    else if(event.key==='Escape'){onClose();event.preventDefault()}
  }}>
    {items.map((item,index)=>{
      if(item.kind==='separator')return <hr key={index}/>
      if(item.kind==='label')return <strong key={index} className="context-menu-label">{item.label}</strong>
      const isSubmenu=item.kind==='submenu',open=isSubmenu&&submenu?.index===index
      return <div key={index} className={`context-menu-row${open?' open':''}`}>
        <button
          ref={node=>{rows.current[index]=node}}
          role={isSubmenu?'menuitem':item.checked===undefined?'menuitem':'menuitemcheckbox'}
          aria-haspopup={isSubmenu||undefined} aria-expanded={isSubmenu?open:undefined}
          aria-checked={!isSubmenu&&item.kind==='item'&&item.checked!==undefined?item.checked:undefined}
          className={`${index===active?'active':''}${!isSubmenu&&item.kind==='item'&&item.danger?' danger':''}`}
          disabled={item.disabled}
          // Hovering a parent row opens its submenu straight away, the way a
          // desktop menu behaves, and leaving for another row closes it.
          onMouseEnter={()=>{setActive(index);if(isSubmenu&&!item.disabled)setSubmenu({index,keyboard:false});else setSubmenu(undefined)}}
          onClick={()=>choose(item,index)}>
          <span className="context-menu-icon">{item.kind==='item'&&item.checked?'✓':item.icon}</span>
          <span className="context-menu-text">{item.label}</span>
          {isSubmenu?<ChevronRight className="context-menu-arrow"/>:item.kind==='item'&&item.shortcut?<kbd>{item.shortcut}</kbd>:null}
        </button>
        {open&&item.kind==='submenu'&&<Submenu anchor={rows.current[index]} items={item.items} autoFocus={submenu?.keyboard===true} onClose={onClose} onBack={closeSubmenu}/>}
      </div>
    })}
  </div>
}

/** A pointer- and keyboard-driven menu rendered at viewport coordinates. */
export function ContextMenu({x,y,items,label,onClose,onSwitchMenu}:{x:number;y:number;items:MenuItem[];label:string;onClose:()=>void;onSwitchMenu?:(delta:1|-1)=>void}){
  const container=useRef<HTMLDivElement>(null),[position,setPosition]=useState(()=>clampPosition(x,y,240))
  useLayoutEffect(()=>{setPosition(clampPosition(x,y,container.current?.offsetHeight??240))},[x,y,items])
  useEffect(()=>{
    // Submenus live in portals, so anything inside a menu surface counts as
    // inside this menu.
    const dismiss=(event:MouseEvent)=>{if(!(event.target as HTMLElement|null)?.closest?.('[data-kanpic-menu]'))onClose()}
    const close=()=>onClose()
    window.addEventListener('mousedown',dismiss)
    window.addEventListener('resize',close)
    window.addEventListener('blur',close)
    return()=>{window.removeEventListener('mousedown',dismiss);window.removeEventListener('resize',close);window.removeEventListener('blur',close)}
  },[onClose])
  return <div className="context-menu" data-kanpic-menu ref={container} style={{left:position.left,top:position.top}} onContextMenu={event=>event.preventDefault()}>
    <MenuList items={items} onClose={onClose} autoFocus label={label} onSwitch={onSwitchMenu}/>
  </div>
}
