import { describe,expect,it } from 'vitest'
import { cellKey } from '../state/editor'
import type { Cell } from '../types'
import { consolidate, type ConsolidateSource } from './consolidate'

/** 첫 행은 항목 이름, 첫 열은 줄 이름표인 표 하나를 만든다. */
function sheet(sheetName:string,grid:unknown[][]):ConsolidateSource{
  const cells=new Map<string,Cell>()
  grid.forEach((row,rowIndex)=>row.forEach((value,columnIndex)=>{
    if(value===undefined)return
    cells.set(cellKey(rowIndex+1,columnIndex+1),{sheet_id:sheetName,row:rowIndex+1,column:columnIndex+1,value,updated_at:''})
  }))
  return {sheetName,cells,region:{startRow:1,startColumn:1,endRow:grid.length,endColumn:Math.max(...grid.map(row=>row.length))}}
}
const at=(result:ReturnType<typeof consolidate>,row:string,column:string)=>
  result.values.get(`${result.rowLabels.indexOf(row)}:${result.columnLabels.indexOf(column)}`)

const january=sheet('1월',[['부서','매출','건수'],['영업1팀',100,2],['영업2팀',200,3]])
const february=sheet('2월',[['부서','매출','건수'],['영업2팀',20,1],['영업1팀',10,4]])

describe('consolidate',()=>{
  it('adds by label even when the sheets order rows differently',()=>{
    // 시트마다 부서 차례가 달라도 이름표로 맞춘다. 자리로 맞추면 1월 영업1팀에
    // 2월 영업2팀이 더해진다.
    const result=consolidate([january,february],'sum')
    expect(at(result,'영업1팀','매출')).toBe(110)
    expect(at(result,'영업2팀','매출')).toBe(220)
    expect(at(result,'영업1팀','건수')).toBe(6)
  })

  it('keeps a label that only one sheet has, and says which sheet lacked it',()=>{
    // 없는 것과 0인 것은 다르다. 부서가 통째로 빠진 것을 0원 실적으로 읽으면 안 된다.
    const march=sheet('3월',[['부서','매출'],['영업3팀',7]])
    const result=consolidate([january,march],'sum')
    expect(at(result,'영업3팀','매출')).toBe(7)
    expect(result.missing).toContainEqual({sheetName:'1월',label:'영업3팀'})
    expect(result.missing).toContainEqual({sheetName:'3월',label:'영업1팀'})
  })

  it('counts the text cells it did not add instead of hiding them',()=>{
    // 조용히 빼면 합계가 작게 나오고 이유는 아무 데도 적히지 않는다.
    const messy=sheet('3월',[['부서','매출'],['영업1팀','1,000']])
    const result=consolidate([january,messy],'sum')
    expect(at(result,'영업1팀','매출')).toBe(100)
    expect(result.skippedText).toBe(1)
  })

  it('applies each function over the values it gathered',()=>{
    expect(at(consolidate([january,february],'count'),'영업1팀','매출')).toBe(2)
    expect(at(consolidate([january,february],'average'),'영업1팀','매출')).toBe(55)
    expect(at(consolidate([january,february],'max'),'영업1팀','매출')).toBe(100)
    expect(at(consolidate([january,february],'min'),'영업1팀','매출')).toBe(10)
  })

  it('names a sheet that holds merged cells',()=>{
    // 병합된 이름표 열은 빈 이름표 행을 만든다. 그 행은 통합에서 통째로 빠진다.
    const merged=sheet('3월',[['부서','매출'],['영업1팀',5],['',6]])
    merged.cells.get(cellKey(2,1))!.style={merge:{start_row:2,start_column:1,end_row:3,end_column:1}}
    merged.cells.set(cellKey(3,1),{sheet_id:'3월',row:3,column:1,value:'',updated_at:'',style:{merge:{start_row:2,start_column:1,end_row:3,end_column:1}}})
    expect(consolidate([merged],'sum').mergedSheets).toEqual(['3월'])
  })

  it('does not confuse a numbered label with a plain number',()=>{
    // 007 부서와 7 부서는 다른 부서다. 대사에서 쓰는 규칙을 그대로 쓴다.
    const left=sheet('가',[['코드','값'],['007',1]])
    const right=sheet('나',[['코드','값'],['7',2]])
    const result=consolidate([left,right],'sum')
    expect(result.rowLabels).toEqual(['007','7'])
  })
})
