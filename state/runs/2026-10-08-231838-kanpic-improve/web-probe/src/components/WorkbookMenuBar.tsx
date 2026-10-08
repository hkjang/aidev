import { useRef, useState } from 'react'
import { ContextMenu, type MenuItem } from './ContextMenu'

export type WorkbookMenu={label:string;items:MenuItem[]}

/**
 * The workbook menu bar reuses the grid context menu so both surfaces share
 * keyboard navigation, submenu behaviour and shortcut hints.
 */
export function WorkbookMenuBar({menus}:{menus:WorkbookMenu[]}){
  const [open,setOpen]=useState<number>()
  const buttons=useRef<Array<HTMLButtonElement|null>>([])
  const focusMenu=(index:number)=>{const next=(index+menus.length)%menus.length;buttons.current[next]?.focus();if(open!==undefined)setOpen(next)}
  const anchor=()=>{
    const rect=open===undefined?undefined:buttons.current[open]?.getBoundingClientRect()
    return{x:rect?.left??0,y:(rect?.bottom??0)+3}
  }
  return <div className="menu-strip" role="menubar" aria-label="워크북 메뉴">
    {menus.map((menu,index)=><button key={menu.label} ref={node=>{buttons.current[index]=node}} type="button" role="menuitem"
      aria-haspopup="menu" aria-expanded={open===index} className={open===index?'active':''}
      // The menu itself takes the keyboard, so the button must not steal focus
      // back on mouse down.
      onMouseDown={event=>{event.stopPropagation();event.preventDefault();setOpen(current=>current===index?undefined:index)}}
      onMouseEnter={()=>setOpen(current=>current===undefined?current:index)}
      onKeyDown={event=>{
        if(event.key==='ArrowRight'){focusMenu(index+1);event.preventDefault()}
        else if(event.key==='ArrowLeft'){focusMenu(index-1);event.preventDefault()}
        else if(event.key==='ArrowDown'||event.key==='Enter'||event.key===' '){setOpen(index);event.preventDefault()}
        else if(event.key==='Escape')setOpen(undefined)
      }}>{menu.label}</button>)}
    {open!==undefined&&<ContextMenu {...anchor()} items={menus[open].items} label={`${menus[open].label} 메뉴`} onClose={()=>{setOpen(undefined);buttons.current[open]?.focus()}} onSwitchMenu={delta=>setOpen(current=>current===undefined?current:(current+delta+menus.length)%menus.length)}/>}
  </div>
}
