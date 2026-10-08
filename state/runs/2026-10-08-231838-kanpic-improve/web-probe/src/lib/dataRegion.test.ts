import { describe, expect, it } from 'vitest'
import { cellKey } from '../state/editor'
import { dataRegion, looksLikeHeaderRow, populatedCell } from './dataRegion'
import { MAX_GRID_COLUMNS, MAX_GRID_ROWS } from './clipboard'
import type { Cell } from '../types'

function grid(entries:Array<[number,number,unknown]>){
  const cells=new Map<string,Cell>()
  for(const [row,column,value] of entries)cells.set(cellKey(row,column),{sheet_id:'sheet',row,column,value,updated_at:''})
  return cells
}

const bounds={rows:1000,columns:100}

describe('dataRegion',()=>{
  it('expands to the contiguous block around the seed cell',()=>{
    const cells=grid([[2,2,'이름'],[2,3,'매출'],[3,2,'A'],[3,3,10],[4,2,'B'],[4,3,20],[9,9,'멀리']])
    expect(dataRegion(cells,3,3,bounds)).toEqual({startRow:2,startColumn:2,endRow:4,endColumn:3})
  })

  it('keeps a single empty cell as its own region',()=>{
    expect(dataRegion(grid([[1,1,'x']]),5,5,bounds)).toEqual({startRow:5,startColumn:5,endRow:5,endColumn:5})
  })

  it('detects text headers above numeric data',()=>{
    const cells=grid([[1,1,'항목'],[1,2,'값'],[2,1,'A'],[2,2,10]])
    const region=dataRegion(cells,2,2,bounds)
    expect(region).toEqual({startRow:1,startColumn:1,endRow:2,endColumn:2})
    expect(looksLikeHeaderRow(cells,region)).toBe(true)
    expect(looksLikeHeaderRow(cells,{startRow:2,startColumn:1,endRow:2,endColumn:2})).toBe(false)
  })

  it('reports populated cells including formulas',()=>{
    const cells=new Map<string,Cell>([[cellKey(1,1),{sheet_id:'s',row:1,column:1,formula:'=1+1',updated_at:''}]])
    expect(populatedCell(cells,1,1)).toBe(true)
    expect(populatedCell(cells,2,1)).toBe(false)
  })
})

// 편집기의 행 한도는 서버가 담아 두는 시트보다 작으면 안 된다. 10,000이던
// 시절에는 2만 행짜리 표를 정렬하면 앞의 절반만 정렬됐다.
it('resolves a table taller than ten thousand rows', () => {
  const cells=new Map<string,Cell>()
  for(let row=1;row<=15_000;row+=1)cells.set(cellKey(row,1),{sheet_id:'s',row,column:1,value:row,updated_at:''})
  const region=dataRegion(cells,1,1,{rows:MAX_GRID_ROWS,columns:MAX_GRID_COLUMNS})
  expect(region.endRow).toBe(15_000)
})
