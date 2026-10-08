import { describe,expect,it } from 'vitest'
import { cellKey } from '../state/editor'
import type { Cell } from '../types'
import { planFlashFill,type FillPlan } from './flashFillPlan'

const sheet=(rows:Array<Array<string|number|undefined>>)=>{
  const cells=new Map<string,Cell>()
  rows.forEach((row,rowIndex)=>row.forEach((value,columnIndex)=>{
    if(value===undefined)return
    cells.set(cellKey(rowIndex+1,columnIndex+1),{sheet_id:'s',row:rowIndex+1,column:columnIndex+1,value,updated_at:''})
  }))
  return cells
}
const region=(rows:number,columns:number)=>({startRow:1,startColumn:1,endRow:rows,endColumn:columns})

describe('planFlashFill', () => {
  it('takes the filled cells as examples and plans the empty ones', () => {
    const cells=sheet([
      ['이메일','아이디'],
      ['hong@example.com','hong'],
      ['kim@sample.co.kr',undefined],
      ['lee@x.io',undefined],
    ])
    const plan=planFlashFill(cells,region(4,2),2) as FillPlan
    expect(plan.examples).toEqual([{row:2,value:'hong'}])
    expect(plan.writes).toEqual([{row:3,value:'kim'},{row:4,value:'lee'}])
    expect(plan.unreached).toBe(0)
  })

  // 이미 쓴 값을 덮어쓰면 되돌리기 전에는 무엇이 있었는지 알 수 없다.
  it('never writes over a cell that already says something', () => {
    const cells=sheet([
      ['hong@example.com','hong'],
      ['kim@sample.co.kr','kim'],
      ['lee@x.io',undefined],
    ])
    const plan=planFlashFill(cells,region(3,2),2) as FillPlan
    expect(plan.examples.map(example=>example.row)).toEqual([1,2])
    expect(plan.writes).toEqual([{row:3,value:'lee'}])
  })

  // 손으로 써 넣은 엉뚱한 값도 예시로 읽는다. 그것과 어긋나는 규칙을 밀어붙이면
  // 사람이 일부러 적어 둔 것을 무시하는 셈이 된다 — 엑셀도 여기서 포기한다.
  it('gives up when one filled cell contradicts the others', () => {
    const cells=sheet([
      ['hong@example.com','hong'],
      ['kim@sample.co.kr','손으로 쓴 값'],
      ['lee@x.io',undefined],
    ])
    expect(planFlashFill(cells,region(3,2),2)).toBe('no-rule')
  })

  // 규칙이 닿지 못한 줄을 조용히 비워 두면 채워진 줄만 보고 다 됐다고 여긴다.
  it('counts the rows the rule could not reach', () => {
    const cells=sheet([
      ['hong@example.com','hong'],
      ['구분자 없음',undefined],
      ['lee@x.io',undefined],
    ])
    const plan=planFlashFill(cells,region(3,2),2) as FillPlan
    expect(plan.writes).toEqual([{row:3,value:'lee'}])
    expect(plan.unreached).toBe(1)
  })

  it('says why it cannot plan', () => {
    const empty=sheet([['hong@example.com',undefined],['kim@sample.co.kr',undefined]])
    expect(planFlashFill(empty,region(2,2),2)).toBe('no-examples')

    const full=sheet([['hong@example.com','hong'],['kim@sample.co.kr','kim']])
    expect(planFlashFill(full,region(2,2),2)).toBe('nothing-to-fill')

    const unexplained=sheet([['hong@example.com','전혀 다른 값'],['kim@sample.co.kr',undefined]])
    expect(planFlashFill(unexplained,region(2,2),2)).toBe('no-rule')
  })

  // 머리글은 예시가 아니다. 예시로 삼으면 "아이디" 라는 낱말에서 규칙을 찾으려
  // 든다. 숫자가 하나도 없는 표에서도 알아봐야 한다 — 이름과 주소만 있는 표가
  // 빠른 채우기를 가장 많이 쓰는 표다.
  it('leaves the header row out of the examples', () => {
    const cells=sheet([
      ['이메일','아이디'],
      ['hong@example.com','hong'],
      ['kim@sample.co.kr',undefined],
    ])
    const plan=planFlashFill(cells,region(3,2),2) as FillPlan
    expect(plan.examples.map(example=>example.row)).toEqual([2])
    expect(plan.headerSkipped).toBe(true)
    expect(plan.writes).toEqual([{row:3,value:'kim'}])
  })

  // 머리글이 없는 표에서는 첫 줄을 빼면 안 된다.
  it('keeps the first row when it is an example rather than a heading', () => {
    const cells=sheet([
      ['hong@example.com','hong'],
      ['kim@sample.co.kr',undefined],
    ])
    const plan=planFlashFill(cells,region(2,2),2) as FillPlan
    expect(plan.headerSkipped).toBe(false)
    expect(plan.examples.map(example=>example.row)).toEqual([1])
  })

  it('reads more than one source column', () => {
    const cells=sheet([
      ['홍','길동','홍 길동'],
      ['김','철수',undefined],
    ])
    const plan=planFlashFill(cells,region(2,3),3) as FillPlan
    expect(plan.writes).toEqual([{row:2,value:'김 철수'}])
  })
})
