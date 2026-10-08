import { describe, expect, it } from 'vitest'
import { planRemoveSubtotals, planSubtotals } from './subtotal'
import type { Cell } from '../types'
import { cellKey } from '../state/editor'

const sheet=(rows:Array<[string,number]>):Map<string,Cell>=>{
  const cells=new Map<string,Cell>()
  cells.set(cellKey(1,1),{sheet_id:'s',row:1,column:1,value:'지역'} as Cell)
  cells.set(cellKey(1,2),{sheet_id:'s',row:1,column:2,value:'매출'} as Cell)
  rows.forEach(([region,amount],index)=>{
    cells.set(cellKey(index+2,1),{sheet_id:'s',row:index+2,column:1,value:region} as Cell)
    cells.set(cellKey(index+2,2),{sheet_id:'s',row:index+2,column:2,value:amount} as Cell)
  })
  return cells
}
const region=(endRow:number)=>({startRow:1,startColumn:1,endRow,endColumn:2})

describe('planSubtotals',()=>{
  it('adds one subtotal row per run and folds the rows it covers',()=>{
    const cells=sheet([['서울',10],['서울',20],['부산',30]])
    const plan=planSubtotals(cells,region(4),{groupColumn:1,valueColumns:[2],aggregation:'sum',headerRows:1,grandTotal:false})
    expect(plan.runs.map(run=>run.label)).toEqual(['서울','부산'])
    expect(plan.addedRows).toBe(2)
    // 서울 covers rows 2-3 and its total lands on 4; 부산 moves down one row.
    expect(plan.groups).toEqual([{start:2,end:3},{start:5,end:5}])
    const formulas=plan.writes.filter(write=>write.formula).map(write=>`${write.row}:${write.formula}`)
    expect(formulas).toEqual(['4:=SUBTOTAL(109,B2:B3)','6:=SUBTOTAL(109,B5:B5)'])
  })

  it('labels each subtotal with its group',()=>{
    const cells=sheet([['서울',10],['부산',30]])
    const plan=planSubtotals(cells,region(3),{groupColumn:1,valueColumns:[2],aggregation:'average',headerRows:1,grandTotal:false})
    const labels=plan.writes.filter(write=>typeof write.value==='string'&&write.value.endsWith('평균')).map(write=>write.value)
    expect(labels).toEqual(['서울 평균','부산 평균'])
  })

  // A grand total over subtotals must not count the same numbers twice, which
  // is what the 1xx SUBTOTAL codes are for.
  it('adds a grand total that ignores the subtotals above it',()=>{
    const cells=sheet([['서울',10],['부산',30]])
    const plan=planSubtotals(cells,region(3),{groupColumn:1,valueColumns:[2],aggregation:'sum',headerRows:1,grandTotal:true})
    const grand=plan.writes.filter(write=>write.formula).pop()
    // The data runs are named directly. A span covering the subtotal rows
    // would add their numbers to the ones they already summarise.
    expect(grand?.formula).toBe('=SUBTOTAL(109,B2:B2,B4:B4)')
    expect(plan.addedRows).toBe(3)
  })

  it('reports a group column that is not sorted',()=>{
    const scattered=sheet([['서울',10],['부산',30],['서울',20]])
    expect(planSubtotals(scattered,region(4),{groupColumn:1,valueColumns:[2],aggregation:'sum',headerRows:1,grandTotal:false}).unsorted).toBe(true)
    const sorted=sheet([['부산',30],['서울',10],['서울',20]])
    expect(planSubtotals(sorted,region(4),{groupColumn:1,valueColumns:[2],aggregation:'sum',headerRows:1,grandTotal:false}).unsorted).toBe(false)
  })
})

describe('subtotal rows',()=>{
  it('clears the columns it does not total, because the row held data',()=>{
    const cells=new Map<string,Cell>()
    cells.set(cellKey(1,1),{sheet_id:'s',row:1,column:1,value:'지역'} as Cell)
    cells.set(cellKey(1,2),{sheet_id:'s',row:1,column:2,value:'제품'} as Cell)
    cells.set(cellKey(1,3),{sheet_id:'s',row:1,column:3,value:'매출'} as Cell)
    const rows:Array<[string,string,number]>=[['부산','공책',80],['서울','연필',120]]
    rows.forEach(([area,item,amount],index)=>{
      cells.set(cellKey(index+2,1),{sheet_id:'s',row:index+2,column:1,value:area} as Cell)
      cells.set(cellKey(index+2,2),{sheet_id:'s',row:index+2,column:2,value:item} as Cell)
      cells.set(cellKey(index+2,3),{sheet_id:'s',row:index+2,column:3,value:amount} as Cell)
    })
    const plan=planSubtotals(cells,{startRow:1,startColumn:1,endRow:3,endColumn:3},
      {groupColumn:1,valueColumns:[3],aggregation:'sum',headerRows:1,grandTotal:false})
    // Row 3 is 부산's total and used to hold 서울/연필/120.
    const middle=plan.writes.filter(write=>write.row===3)
    expect(middle.find(write=>write.column===2)).toMatchObject({value:undefined,formula:undefined})
    expect(middle.find(write=>write.column===1)?.value).toBe('부산 합계')
  })
})

describe('planRemoveSubtotals',()=>{
  const withSubtotals=()=>{
    const cells=new Map<string,Cell>()
    const put=(row:number,column:number,patch:Partial<Cell>)=>cells.set(cellKey(row,column),{sheet_id:'s',row,column,...patch} as Cell)
    put(1,1,{value:'지역'});put(1,2,{value:'매출'})
    put(2,1,{value:'부산'});put(2,2,{value:80})
    put(3,1,{value:'부산 합계'});put(3,2,{value:80,formula:'=SUBTOTAL(109,B2:B2)'})
    put(4,1,{value:'서울'});put(4,2,{value:120})
    put(5,1,{value:'서울 합계'});put(5,2,{value:120,formula:'=SUBTOTAL(109,B4:B4)'})
    put(6,1,{value:'전체 합계'});put(6,2,{value:200,formula:'=SUBTOTAL(109,B2:B2,B4:B4)'})
    return cells
  }

  it('removes every row a subtotal run added and pulls the data up',()=>{
    const plan=planRemoveSubtotals(withSubtotals(),{startRow:1,startColumn:1,endRow:6,endColumn:2})
    expect(plan.rows.map(item=>item.label)).toEqual(['부산 합계','서울 합계','전체 합계'])
    const at=(row:number,column:number)=>plan.writes.find(write=>write.row===row&&write.column===column)
    expect(at(3,1)?.value).toBe('서울')
    expect(at(3,2)?.value).toBe(120)
    // The rows the table no longer needs are cleared, not left as leftovers.
    expect(at(4,1)).toMatchObject({value:undefined,formula:undefined})
    expect(at(6,2)).toMatchObject({value:undefined,formula:undefined})
  })

  it('recognises the rows by their formula, not by the label',()=>{
    const cells=withSubtotals()
    cells.set(cellKey(3,1),{sheet_id:'s',row:3,column:1,value:'부산 소계(수정됨)'} as Cell)
    const plan=planRemoveSubtotals(cells,{startRow:1,startColumn:1,endRow:6,endColumn:2})
    expect(plan.rows).toHaveLength(3)
  })

  it('does nothing when the table has no subtotal rows',()=>{
    const cells=new Map<string,Cell>()
    cells.set(cellKey(1,1),{sheet_id:'s',row:1,column:1,value:'지역'} as Cell)
    cells.set(cellKey(2,1),{sheet_id:'s',row:2,column:1,value:'부산'} as Cell)
    expect(planRemoveSubtotals(cells,{startRow:1,startColumn:1,endRow:2,endColumn:1})).toEqual({rows:[],writes:[]})
  })
})
