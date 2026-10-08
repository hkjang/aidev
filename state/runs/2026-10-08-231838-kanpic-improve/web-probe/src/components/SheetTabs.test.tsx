import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SheetTabs } from './SheetTabs'
import type { Sheet } from '../types'

afterEach(()=>{cleanup();vi.restoreAllMocks()})

const sheet=(id:string,name:string,position:number,hidden=false):Sheet=>({id,workbook_id:'wb',name,position,hidden,created_at:'',updated_at:''} as unknown as Sheet)
const sheets=[sheet('s1','요약',0),sheet('s2','원본',1),sheet('s3','보관',2,true)]

function renderTabs(overrides:Partial<Parameters<typeof SheetTabs>[0]>={}){
  const handlers={
    onSelect:vi.fn(),onCreate:vi.fn().mockResolvedValue(undefined),onRename:vi.fn().mockResolvedValue(undefined),
    onDuplicate:vi.fn().mockResolvedValue(undefined),onMove:vi.fn().mockResolvedValue(undefined),onColor:vi.fn().mockResolvedValue(undefined),
    onHidden:vi.fn().mockResolvedValue(undefined),onDelete:vi.fn().mockResolvedValue(undefined),onManage:vi.fn(),onCopyTo:vi.fn(),
  }
  render(<SheetTabs sheets={sheets} activeSheetId="s1" version={1} saveState="saved" saveLabel="저장됨" {...handlers} {...overrides}/>)
  return handlers
}

describe('SheetTabs context menu',()=>{
  it('opens on right click for a tab that is not active and selects it first',()=>{
    const handlers=renderTabs()
    fireEvent.contextMenu(screen.getByText('원본'))
    expect(handlers.onSelect).toHaveBeenCalledWith(expect.objectContaining({id:'s2'}))
    expect(screen.getByRole('menu',{name:'원본 시트 메뉴'})).toBeTruthy()
    expect(screen.getByRole('menuitem',{name:/복제/})).toBeTruthy()
  })

  it('moves a sheet from the menu',async()=>{
    const handlers=renderTabs()
    fireEvent.contextMenu(screen.getByText('원본'))
    fireEvent.click(screen.getByRole('menuitem',{name:/왼쪽으로 이동/}))
    await waitFor(()=>expect(handlers.onMove).toHaveBeenCalledWith(expect.objectContaining({id:'s2'}),0))
  })

  it('offers hidden sheets and sheet creation from the empty strip',async()=>{
    const handlers=renderTabs()
    fireEvent.contextMenu(document.querySelector('.sheet-tabs') as Element)
    expect(screen.getByRole('menu',{name:'시트 탭 메뉴'})).toBeTruthy()
    fireEvent.click(screen.getByRole('menuitem',{name:'보관'}))
    await waitFor(()=>expect(handlers.onHidden).toHaveBeenCalledWith(expect.objectContaining({id:'s3'}),false))
  })

  it('disables editing actions in read only workbooks',()=>{
    renderTabs({readOnly:true})
    fireEvent.contextMenu(screen.getByText('요약'))
    expect(screen.getByRole('menuitem',{name:/시트 삭제/,hidden:true})).toBeDisabled()
  })
})

describe('a menu opened while a sheet operation is running',()=>{
  it('comes back to life when the operation finishes',async()=>{
    // 항목을 열 때 통째로 굳혀 두면, 여는 순간의 pending 이 그대로 남아
    // 작업이 끝난 뒤에도 메뉴가 죽은 채로 보인다. 사람은 닫았다 다시 열어야
    // 하는데 왜 그래야 하는지 알 수 없다. CI 에서 이것 때문에 30초를 기다리다
    // 시험이 깨졌다.
    let finish=()=>{}
    const slow=new Promise<void>(resolve=>{finish=resolve})
    const duplicate=vi.fn().mockReturnValue(slow)
    renderTabs({onDuplicate:duplicate})

    fireEvent.contextMenu(screen.getByText('원본'))
    fireEvent.click(screen.getByRole('menuitem',{name:/복제/}))
    await waitFor(()=>expect(duplicate).toHaveBeenCalled())

    // 작업이 도는 동안 다시 연 메뉴는 잠겨 있어야 한다.
    fireEvent.contextMenu(screen.getByText('원본'))
    await waitFor(()=>expect(screen.getByRole('menuitem',{name:/왼쪽으로 이동/})).toHaveProperty('disabled',true))

    finish()
    // 끝나면 그대로 열어 둔 채로 다시 쓸 수 있어야 한다.
    await waitFor(()=>expect(screen.getByRole('menuitem',{name:/왼쪽으로 이동/})).toHaveProperty('disabled',false))
  })

  it('judges move-left by where the sheet is now, not where it was',async()=>{
    // 열 때의 시트를 들고 있으면 옆 시트가 움직인 뒤에도 옛 자리로 판단한다.
    const moved=[sheet('s2','원본',0),sheet('s1','요약',1),sheet('s3','보관',2,true)]
    const { rerender }=render(<SheetTabs sheets={sheets} activeSheetId="s1" version={1} saveState="saved" saveLabel="저장됨"
      onSelect={vi.fn()} onCreate={vi.fn()} onRename={vi.fn()} onDuplicate={vi.fn()} onMove={vi.fn()} onColor={vi.fn()}
      onHidden={vi.fn()} onDelete={vi.fn()} onManage={vi.fn()}/>)
    fireEvent.contextMenu(screen.getByText('원본'))
    expect(screen.getByRole('menuitem',{name:/왼쪽으로 이동/})).toHaveProperty('disabled',false)
    rerender(<SheetTabs sheets={moved} activeSheetId="s1" version={1} saveState="saved" saveLabel="저장됨"
      onSelect={vi.fn()} onCreate={vi.fn()} onRename={vi.fn()} onDuplicate={vi.fn()} onMove={vi.fn()} onColor={vi.fn()}
      onHidden={vi.fn()} onDelete={vi.fn()} onManage={vi.fn()}/>)
    // 이제 맨 왼쪽이므로 잠겨야 한다.
    await waitFor(()=>expect(screen.getByRole('menuitem',{name:/왼쪽으로 이동/})).toHaveProperty('disabled',true))
  })
})
