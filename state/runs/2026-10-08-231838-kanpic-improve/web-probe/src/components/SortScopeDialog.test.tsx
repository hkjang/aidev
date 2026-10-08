import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { SortScopeDialog } from './SortScopeDialog'
import { cellKey } from '../state/editor'
import type { Cell } from '../types'

afterEach(()=>cleanup())

const block=(endRow:number,endColumn:number)=>{
  const cells=new Map<string,Cell>()
  for(let row=1;row<=Math.min(endRow,3);row+=1)cells.set(cellKey(row,1),{sheet_id:'s',row,column:1,value:row,updated_at:''})
  return {region:{startRow:1,startColumn:1,endRow,endColumn},cells}
}

describe('SortScopeDialog', () => {
  it('sorts a range inside the limit', () => {
    const onSort=vi.fn().mockResolvedValue(undefined)
    render(<SortScopeDialog column={1} direction="asc" block={block(100,3)} selection={{startRow:1,startColumn:1,endRow:1,endColumn:1}} onClose={()=>{}} onSort={onSort}/>)
    const button=screen.getByRole('button',{name:'정렬'})
    expect(button).toBeEnabled()
    fireEvent.click(button)
    expect(onSort).toHaveBeenCalled()
  })

  // 눌러 본 뒤에 알려 주면 무엇을 줄여야 할지 모른 채 실패만 겪는다.
  it('says why a range is too large before the button is pressed', () => {
    const onSort=vi.fn()
    render(<SortScopeDialog column={1} direction="asc" block={block(100,2_000)} selection={{startRow:1,startColumn:1,endRow:1,endColumn:1}} onClose={()=>{}} onSort={onSort}/>)
    expect(screen.getByRole('dialog')).toHaveTextContent('200,000셀')
    expect(screen.getByRole('dialog')).toHaveTextContent('60,000셀을 넘습니다')
    expect(screen.getByRole('button',{name:'정렬'})).toBeDisabled()
    expect(onSort).not.toHaveBeenCalled()
  })
})
