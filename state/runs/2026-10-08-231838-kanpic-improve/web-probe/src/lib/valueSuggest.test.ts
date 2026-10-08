import { describe, expect, it } from 'vitest'
import { suggestColumnValues } from './valueSuggest'
import type { Cell } from '../types'

const grid=(values:Array<[number,number,unknown,string?]>)=>{
  const cells=new Map<string,Cell>()
  for(const [row,column,value,formula] of values)cells.set(`${row}:${column}`,{sheet_id:'s',row,column,value,formula,updated_at:''} as Cell)
  return cells
}

describe('suggestColumnValues', () => {
  const cells=grid([
    [1,1,'영업본부'],[2,1,'영업본부'],[3,1,'영업지원'],[4,1,'개발본부'],
    [5,2,'영업본부'],[6,1,'',''],[7,1,120],[8,1,'영업합계','=A1'],
  ])

  it('offers the column entries that start with what was typed', () => {
    expect(suggestColumnValues(cells,1,9,'영업')).toEqual(['영업본부','영업지원'])
  })

  it('puts the value used most often first', () => {
    expect(suggestColumnValues(cells,1,9,'영')[0]).toBe('영업본부')
  })

  it('leaves other columns, formulas, numbers and the current row alone', () => {
    expect(suggestColumnValues(cells,1,9,'개발')).toEqual(['개발본부'])
    expect(suggestColumnValues(cells,1,1,'영업본부')).toEqual([])
    expect(suggestColumnValues(cells,1,9,'12')).toEqual([])
    expect(suggestColumnValues(cells,1,9,'영업합')).toEqual([])
  })

  it('stays out of the way for formulas and empty input', () => {
    expect(suggestColumnValues(cells,1,9,'=영업')).toEqual([])
    expect(suggestColumnValues(cells,1,9,'   ')).toEqual([])
  })

  it('does not suggest what has already been typed in full', () => {
    expect(suggestColumnValues(cells,1,9,'영업본부')).toEqual([])
  })
})
