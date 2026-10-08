import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { StructureDialog } from './StructureDialog'

afterEach(()=>{cleanup();vi.restoreAllMocks()})

describe('StructureDialog',()=>{
  const range={startRow:2,startColumn:3,endRow:4,endColumn:4}

  it('creates a row insertion command from the selected row span',async()=>{
    const apply=vi.fn().mockResolvedValue(undefined),close=vi.fn()
    render(<StructureDialog range={range} onClose={close} onApply={apply}/>)
    fireEvent.click(screen.getByRole('button',{name:'위에 3개 삽입'}))
    await waitFor(()=>expect(apply).toHaveBeenCalledWith({axis:'row',action:'insert',index:2,count:3}))
    expect(close).toHaveBeenCalled()
  })

  it('requires confirmation before deleting selected columns',async()=>{
    const apply=vi.fn().mockResolvedValue(undefined),close=vi.fn()
    vi.spyOn(window,'confirm').mockReturnValue(false)
    render(<StructureDialog range={range} onClose={close} onApply={apply}/>)
    fireEvent.click(screen.getByRole('button',{name:'선택 열 삭제'}))
    expect(apply).not.toHaveBeenCalled()
    vi.mocked(window.confirm).mockReturnValue(true)
    fireEvent.click(screen.getByRole('button',{name:'선택 열 삭제'}))
    await waitFor(()=>expect(apply).toHaveBeenCalledWith({axis:'column',action:'delete',index:3,count:2}))
  })
})
