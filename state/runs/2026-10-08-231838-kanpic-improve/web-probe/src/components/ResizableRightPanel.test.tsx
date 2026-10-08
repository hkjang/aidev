import { cleanup,fireEvent,render,screen,waitFor } from '@testing-library/react'
import { afterEach,describe,expect,it } from 'vitest'
import { ResizableRightPanel } from './ResizableRightPanel'

afterEach(()=>{cleanup();window.localStorage.clear()})

describe('ResizableRightPanel',()=>{
  it('supports keyboard resizing, reset, and restores each panel width',async()=>{
    Object.defineProperty(window,'innerWidth',{configurable:true,value:1200})
    const view=render(<ResizableRightPanel panelKey="ai"><aside>AI</aside></ResizableRightPanel>)
    const separator=screen.getByRole('separator',{name:'AI 도우미 패널 너비 조절'})
    expect(separator.parentElement).toHaveStyle({width:'460px'})
    fireEvent.keyDown(separator,{key:'ArrowLeft'})
    expect(separator.parentElement).toHaveStyle({width:'476px'})
    await waitFor(()=>expect(window.localStorage.getItem('kanpic:right-panel-width:ai')).toBe('476'))

    view.unmount()
    render(<ResizableRightPanel panelKey="ai"><aside>AI 다시 열기</aside></ResizableRightPanel>)
    const restored=screen.getByRole('separator',{name:'AI 도우미 패널 너비 조절'})
    expect(restored.parentElement).toHaveStyle({width:'476px'})
    fireEvent.doubleClick(restored)
    expect(restored.parentElement).toHaveStyle({width:'460px'})
  })

  it('defines a resizable preset for every editor right panel',()=>{
    for(const panelKey of ['ai','automation','history','comments','conflicts','charts','pivots'] as const){
      const view=render(<ResizableRightPanel panelKey={panelKey}><aside>{panelKey}</aside></ResizableRightPanel>)
      expect(screen.getByRole('separator')).toHaveAttribute('aria-label',expect.stringContaining('패널 너비 조절'))
      view.unmount()
    }
  })
})
