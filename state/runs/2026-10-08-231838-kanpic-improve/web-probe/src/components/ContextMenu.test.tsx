import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ContextMenu, submenuPosition, type MenuItem } from './ContextMenu'

afterEach(()=>{cleanup();vi.restoreAllMocks()})

function items(onCopy:()=>void,onDelete:()=>void,onSort:()=>void):MenuItem[]{
  return[
    {kind:'item',label:'복사',shortcut:'Ctrl+C',onSelect:onCopy},
    {kind:'item',label:'붙여넣기',disabled:true,onSelect:()=>{throw new Error('disabled item must not run')}},
    {kind:'separator'},
    {kind:'item',label:'행 삭제',danger:true,onSelect:onDelete},
    {kind:'submenu',label:'데이터',items:[{kind:'item',label:'범위 정렬…',onSelect:onSort}]},
  ]
}

describe('ContextMenu',()=>{
  it('runs the clicked action and closes',()=>{
    const copy=vi.fn(),close=vi.fn()
    render(<ContextMenu x={20} y={30} label="셀 메뉴" items={items(copy,()=>{},()=>{})} onClose={close}/>)
    fireEvent.click(screen.getByRole('menuitem',{name:/복사/}))
    expect(copy).toHaveBeenCalled()
    expect(close).toHaveBeenCalled()
  })

  it('keeps disabled items inert and skips them while navigating',()=>{
    const remove=vi.fn()
    render(<ContextMenu x={0} y={0} label="셀 메뉴" items={items(()=>{},remove,()=>{})} onClose={()=>{}}/>)
    expect(screen.getByRole('menuitem',{name:/붙여넣기/})).toBeDisabled()
    const menu=screen.getByRole('menu')
    fireEvent.keyDown(menu,{key:'ArrowDown'})
    fireEvent.keyDown(menu,{key:'Enter'})
    expect(remove).toHaveBeenCalled()
  })

  it('opens a submenu and runs its action',()=>{
    const sort=vi.fn()
    render(<ContextMenu x={0} y={0} label="셀 메뉴" items={items(()=>{},()=>{},sort)} onClose={()=>{}}/>)
    const submenu=screen.getByRole('menuitem',{name:/데이터/})
    expect(submenu).toHaveAttribute('aria-expanded','false')
    fireEvent.click(submenu)
    expect(submenu).toHaveAttribute('aria-expanded','true')
    fireEvent.click(screen.getByRole('menuitem',{name:/범위 정렬/}))
    expect(sort).toHaveBeenCalled()
  })

  it('closes on Escape and on an outside click',()=>{
    const close=vi.fn()
    render(<ContextMenu x={0} y={0} label="셀 메뉴" items={items(()=>{},()=>{},()=>{})} onClose={close}/>)
    fireEvent.keyDown(screen.getByRole('menu'),{key:'Escape'})
    expect(close).toHaveBeenCalledTimes(1)
    fireEvent.mouseDown(document.body)
    expect(close).toHaveBeenCalledTimes(2)
  })
})

describe('submenuPosition',()=>{
  const anchor=(left:number,top:number)=>({left,top,right:left+240,bottom:top+26,width:240,height:26,x:left,y:top,toJSON:()=>''}) as DOMRect
  const viewport={width:1280,height:800}

  it('opens to the right of the row that owns it',()=>{
    expect(submenuPosition(anchor(300,200),{width:220,height:180},viewport)).toEqual({left:536,top:195})
  })

  it('flips to the left when the right side has no room',()=>{
    expect(submenuPosition(anchor(1000,200),{width:220,height:180},viewport).left).toBe(1004-220)
  })

  it('lifts a tall submenu so it stays inside the viewport',()=>{
    expect(submenuPosition(anchor(300,700),{width:220,height:400},viewport).top).toBe(392)
  })

  it('never leaves the top or left edge',()=>{
    const position=submenuPosition(anchor(4,2),{width:900,height:900},viewport)
    expect(position.left).toBeGreaterThanOrEqual(8)
    expect(position.top).toBeGreaterThanOrEqual(8)
  })
})
