import { describe, expect, it } from 'vitest'
import { BLANK_LABEL, columnFiltered, columnValues, valuesCriterion, withColumnValues } from './columnFilter'
import type { Cell, FilterView } from '../types'

const view=(criteria:FilterView['criteria']=[]):FilterView=>({
  id:'f1',sheet_id:'s',actor_id:'a',name:'필터',range:'A1:C6',header_rows:1,criteria,active:true,
  created_at:'',updated_at:'',
} as FilterView)

const cells=(values:Array<[number,number,unknown]>)=>
  values.map(([row,column,value])=>({sheet_id:'s',row,column,value,updated_at:''} as Cell))

const data=cells([
  [1,1,'지역'],[2,1,'서울'],[3,1,'부산'],[4,1,'서울'],[5,1,'대구'],
])

describe('columnValues', () => {
  it('lists the distinct values under the header, most frequent first', () => {
    expect(columnValues(data,view(),1).map(item=>[item.label,item.count,item.checked])).toEqual([
      ['서울',2,true],['대구',1,true],['부산',1,true],['(빈 값)',1,true],
    ])
  })

  it('counts rows with nothing in the column as blanks', () => {
    expect(columnValues(data,view(),1).find(item=>item.label===BLANK_LABEL)?.count).toBe(1)
  })

  it('marks only the values the criterion keeps', () => {
    const filtered=view([{column:1,operator:'values',values:['서울']}])
    expect(columnValues(data,filtered,1).filter(item=>item.checked).map(item=>item.label)).toEqual(['서울'])
  })

  it('has nothing to offer outside the filter range', () => {
    expect(columnValues(data,view(),9)).toEqual([])
  })
})

describe('withColumnValues', () => {
  it('drops the criterion when every value is kept', () => {
    const values=columnValues(data,view(),1)
    expect(withColumnValues(view(),1,values)).toEqual([])
  })

  it('writes the kept values and leaves other columns alone', () => {
    const existing=view([{column:2,operator:'contains',value:'x'}])
    const values=columnValues(data,existing,1).map(item=>({...item,checked:item.label==='서울'}))
    expect(withColumnValues(existing,1,values)).toEqual([
      {column:2,operator:'contains',value:'x'},
      {column:1,operator:'values',values:['서울']},
    ])
  })

  it('replaces the previous list for the same column', () => {
    const existing=view([{column:1,operator:'values',values:['부산']}])
    const values=columnValues(data,existing,1).map(item=>({...item,checked:item.label==='대구'}))
    expect(withColumnValues(existing,1,values)).toEqual([{column:1,operator:'values',values:['대구']}])
  })
})

describe('criterion helpers', () => {
  it('finds the value list and reports whether a column is filtered', () => {
    const filtered=view([{column:2,operator:'values',values:[1]}])
    expect(valuesCriterion(filtered,2)?.values).toEqual([1])
    expect(columnFiltered(filtered,2)).toBe(true)
    expect(columnFiltered(filtered,1)).toBe(false)
  })
})
