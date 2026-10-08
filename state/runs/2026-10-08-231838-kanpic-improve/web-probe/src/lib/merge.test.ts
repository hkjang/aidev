import { describe,expect,it } from 'vitest'
import type { Cell } from '../types'
import { cellKey } from '../state/editor'
import type { MergeRange } from './merge'
import { cellMerge,mergeInRegion,mergeStyle,selectedMergedBounds,stripMergeStyle } from './merge'

const merged:Cell={sheet_id:'sheet',row:2,column:2,value:9,style:{bold:true,merge:{start_row:1,start_column:1,end_row:2,end_column:2}},updated_at:'now'}

describe('merged cell metadata',()=>{
  it('resolves a covered cell to the complete range',()=>{
    expect(cellMerge(merged)).toEqual({startRow:1,startColumn:1,endRow:2,endColumn:2})
    expect(selectedMergedBounds(new Map([['2:2',merged]]),{startRow:2,startColumn:2,endRow:2,endColumn:2})).toEqual({startRow:1,startColumn:1,endRow:2,endColumn:2})
  })

  it('adds and removes merge metadata without changing ordinary styles',()=>{
    const range={startRow:3,startColumn:4,endRow:4,endColumn:5}
    expect(mergeStyle({italic:true},range,true)).toEqual({italic:true,merge:{start_row:3,start_column:4,end_row:4,end_column:5}})
    expect(mergeStyle(merged.style,range,false)).toEqual({bold:true})
    expect(stripMergeStyle(merged.style)).toEqual({bold:true})
  })

  it('rejects malformed or non-containing metadata',()=>{
    expect(cellMerge({...merged,style:{merge:{start_row:1,start_column:1,end_row:1,end_column:1}}})).toBeUndefined()
    expect(cellMerge({...merged,style:{merge:{start_row:'1'}}})).toBeUndefined()
  })
})

describe('mergeInRegion',()=>{
  const cell=(row:number,column:number,merge?:MergeRange):Cell=>({sheet_id:'s',row,column,updated_at:'',
    ...(merge?{style:{merge:{start_row:merge.startRow,start_column:merge.startColumn,end_row:merge.endRow,end_column:merge.endColumn}}}:{})})
  const map=(items:Cell[])=>new Map(items.map(item=>[cellKey(item.row,item.column),item]))
  const box={startRow:2,startColumn:1,endRow:2,endColumn:2}

  it('finds a merge inside the region',()=>{
    const cells=map([cell(1,1),cell(2,1,box),cell(2,2,box)])
    expect(mergeInRegion(cells,{startRow:1,startColumn:1,endRow:3,endColumn:3})).toMatchObject({row:2,column:1})
  })
  it('finds a merge that started outside it',()=>{
    // 병합 정보는 덮인 칸마다 적혀 있다. 시작 칸이 범위 밖이어도 안쪽 칸
    // 하나만 보면 찾을 수 있어야 한다 — 못 찾으면 그 서식이 딸려 간다.
    const cells=map([cell(2,1,box),cell(2,2,box)])
    expect(mergeInRegion(cells,{startRow:2,startColumn:2,endRow:2,endColumn:2})).toMatchObject({row:2,column:2})
  })
  it('says nothing when the region is clear',()=>{
    expect(mergeInRegion(map([cell(1,1),cell(2,1)]),{startRow:1,startColumn:1,endRow:2,endColumn:1})).toBeUndefined()
  })
})
