import { describe,expect,it } from 'vitest'
import { cellKey } from '../state/editor'
import type { Cell } from '../types'
import { inspectTable } from './inspectTable'

const cell=(row:number,column:number,value?:unknown,style?:Record<string,unknown>):Cell=>
  ({sheet_id:'s',row,column,updated_at:'',...(value===undefined?{}:{value}),...(style?{style}:{})})
const grid=(rows:unknown[][])=>{
  const cells=new Map<string,Cell>()
  rows.forEach((row,r)=>row.forEach((value,c)=>{if(value!==undefined)cells.set(cellKey(r+1,c+1),cell(r+1,c+1,value))}))
  return {cells,region:{startRow:1,startColumn:1,endRow:rows.length,endColumn:Math.max(...rows.map(row=>row.length))}}
}
const kinds=(findings:ReturnType<typeof inspectTable>)=>findings.map(item=>item.kind)

describe('inspectTable',()=>{
  it('says nothing about a clean table',()=>{
    const {cells,region}=grid([['이름','금액'],['가',1],['나',2]])
    expect(inspectTable(cells,region,1)).toEqual([])
  })

  it('counts numbers that arrived as text',()=>{
    const {cells,region}=grid([['이름','금액'],['가','1,000'],['나','₩2,000']])
    const found=inspectTable(cells,region,1).find(item=>item.kind==='numbers')
    expect(found?.count).toBe(2)
    expect(found?.changesTotals).toBe(true)
  })

  it('counts duplicate rows and stray spaces',()=>{
    const {cells,region}=grid([['이름','금액'],['가 ',1],['가 ',1]])
    const found=inspectTable(cells,region,1)
    expect(kinds(found)).toContain('duplicates')
    expect(kinds(found)).toContain('trim')
  })

  it('counts merged ranges',()=>{
    const {cells,region}=grid([['이름','금액'],['가',1],[undefined,2]])
    const box={merge:{start_row:2,start_column:1,end_row:3,end_column:1}}
    cells.set(cellKey(2,1),cell(2,1,'가',box))
    cells.set(cellKey(3,1),cell(3,1,undefined,box))
    expect(inspectTable(cells,region,1).find(item=>item.kind==='unmerge')?.count).toBe(1)
  })

  it('puts what changes the totals before what only tidies the look',()=>{
    // 합계가 달라지는 것과 눈에만 보이는 것은 급한 정도가 다르다.
    const {cells,region}=grid([['이름','금액'],['가 ','1,000'],['가 ','1,000']])
    const found=inspectTable(cells,region,1)
    const lastChanging=found.map(item=>item.changesTotals).lastIndexOf(true)
    const firstCosmetic=found.map(item=>item.changesTotals).indexOf(false)
    expect(firstCosmetic).toBeGreaterThan(lastChanging)
  })

  it('changes nothing in the sheet it looked at',()=>{
    // 검사는 세기만 한다. 무엇을 고칠지는 사람이 미리보기를 보고 정한다.
    const {cells,region}=grid([['이름','금액'],['가 ','1,000'],['가 ','1,000']])
    const before=JSON.stringify([...cells.entries()])
    inspectTable(cells,region,1)
    expect(JSON.stringify([...cells.entries()])).toBe(before)
  })
})
