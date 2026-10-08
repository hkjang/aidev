import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { WorkbookMenuBar } from './WorkbookMenuBar'

afterEach(cleanup)

describe('WorkbookMenuBar',()=>{
  it('opens a menu and runs the chosen command',()=>{
    const save=vi.fn()
    render(<WorkbookMenuBar menus={[
      {label:'파일',items:[{kind:'item',label:'저장',shortcut:'Ctrl+S',onSelect:save}]},
      {label:'보기',items:[{kind:'item',label:'수식 표시',checked:true,onSelect:()=>{}}]},
    ]}/>)
    const file=screen.getByRole('menuitem',{name:'파일'})
    expect(file).toHaveAttribute('aria-expanded','false')
    fireEvent.mouseDown(file)
    expect(file).toHaveAttribute('aria-expanded','true')
    fireEvent.click(screen.getByRole('menuitem',{name:/저장/}))
    expect(save).toHaveBeenCalled()
    expect(screen.getByRole('menuitem',{name:'파일'})).toHaveAttribute('aria-expanded','false')
  })

  it('marks toggled entries with a checkbox role',()=>{
    render(<WorkbookMenuBar menus={[{label:'보기',items:[{kind:'item',label:'수식 표시',checked:true,onSelect:()=>{}}]}]}/>)
    fireEvent.mouseDown(screen.getByRole('menuitem',{name:'보기'}))
    expect(screen.getByRole('menuitemcheckbox',{name:/수식 표시/})).toHaveAttribute('aria-checked','true')
  })

  it('moves between menus with the arrow keys',()=>{
    render(<WorkbookMenuBar menus={[
      {label:'파일',items:[{kind:'item',label:'저장',onSelect:()=>{}}]},
      {label:'수정',items:[{kind:'item',label:'실행 취소',onSelect:()=>{}}]},
    ]}/>)
    fireEvent.keyDown(screen.getByRole('menuitem',{name:'파일'}),{key:'ArrowRight'})
    expect(document.activeElement).toBe(screen.getByRole('menuitem',{name:'수정'}))
  })
})
