import { describe, expect, it } from 'vitest'
import { cellKey } from '../state/editor'
import { clearTableStyleCells, DEFAULT_TABLE_OPTIONS, TABLE_THEMES, tableStyleCells } from './tableStyle'
import type { Cell } from '../types'

const theme=TABLE_THEMES[0]
const region={startRow:1,startColumn:1,endRow:4,endColumn:3}

function grid(entries:Array<[number,number,unknown,Record<string,unknown>?]>){
  const cells=new Map<string,Cell>()
  for(const [row,column,value,style] of entries)cells.set(cellKey(row,column),{sheet_id:'sheet',row,column,value,style,updated_at:''})
  return cells
}
const at=(writes:ReturnType<typeof tableStyleCells>,row:number,column:number)=>writes.find(cell=>cell.row===row&&cell.column===column)

describe('tableStyleCells',()=>{
  it('paints the header band and alternating body rows',()=>{
    const writes=tableStyleCells(grid([]),region,theme,DEFAULT_TABLE_OPTIONS)
    expect(writes).toHaveLength(12)
    expect(at(writes,1,1)?.style).toMatchObject({background:theme.header,color:theme.headerText,bold:true,horizontal_align:'center'})
    // The first body row stays white and the next one is banded.
    expect(at(writes,2,1)?.style?.background).toBeUndefined()
    expect(at(writes,3,1)?.style?.background).toBe(theme.band)
  })

  it('outlines the edges and rules the inside more lightly',()=>{
    const writes=tableStyleCells(grid([]),region,theme,DEFAULT_TABLE_OPTIONS)
    expect(at(writes,1,1)?.style?.borders).toMatchObject({top:{style:'thin',color:theme.outline},left:{style:'thin',color:theme.outline}})
    expect(at(writes,3,2)?.style?.borders).toMatchObject({top:{style:'thin',color:theme.inner},left:{style:'thin',color:theme.inner}})
    expect((at(writes,4,3)?.style?.borders as Record<string,unknown>).bottom).toMatchObject({color:theme.outline})
    expect((at(writes,4,3)?.style?.borders as Record<string,unknown>).right).toMatchObject({color:theme.outline})
  })

  it('carries values and formulas through untouched',()=>{
    const cells=grid([[2,1,'항목'],[2,2,10]])
    cells.set(cellKey(2,3),{sheet_id:'sheet',row:2,column:3,formula:'=B2*2',value:20,updated_at:''})
    const writes=tableStyleCells(cells,region,theme,DEFAULT_TABLE_OPTIONS)
    expect(at(writes,2,1)?.value).toBe('항목')
    expect(at(writes,2,3)).toMatchObject({formula:'=B2*2',value:undefined})
  })

  it('keeps unrelated formatting such as number formats',()=>{
    const cells=grid([[2,2,1200,{number_format:'₩#,##0',italic:true}]])
    expect(at(tableStyleCells(cells,region,theme,DEFAULT_TABLE_OPTIONS),2,2)?.style).toMatchObject({number_format:'₩#,##0',italic:true})
  })

  it('emphasises the last row when a total row is requested',()=>{
    const writes=tableStyleCells(grid([]),region,theme,{...DEFAULT_TABLE_OPTIONS,totalRow:true})
    expect(at(writes,4,1)?.style).toMatchObject({bold:true,background:theme.band})
  })

  it('draws borders only when asked and skips the header for the borders only theme',()=>{
    const plain=TABLE_THEMES.find(item=>item.id==='plain')!
    const writes=tableStyleCells(grid([]),region,plain,DEFAULT_TABLE_OPTIONS)
    expect(at(writes,1,1)?.style?.background).toBeUndefined()
    expect(at(writes,1,1)?.style?.borders).toBeTruthy()
    const withoutBorders=tableStyleCells(grid([]),region,theme,{...DEFAULT_TABLE_OPTIONS,borders:false})
    expect(at(withoutBorders,1,1)?.style?.borders).toBeUndefined()
  })
})

describe('clearTableStyleCells',()=>{
  it('removes the table look and keeps the data and number format',()=>{
    const cells=grid([[1,1,'머리글',{background:'#0f766e',color:'#ffffff',bold:true,borders:{top:{style:'thin',color:'#0f766e'}},number_format:'#,##0'}]])
    const cleared=clearTableStyleCells(cells,region)
    expect(at(cleared,1,1)?.value).toBe('머리글')
    expect(at(cleared,1,1)?.style).toEqual({number_format:'#,##0'})
  })
})
