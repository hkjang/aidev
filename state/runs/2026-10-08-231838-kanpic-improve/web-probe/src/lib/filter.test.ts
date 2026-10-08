import { describe,expect,it } from 'vitest'
import { filterCriterionInput,parseFilterInput,parseFilterRange } from './filter'

describe('filter form helpers',()=>{
  it('parses normalized A1 ranges',()=>expect(parseFilterRange('C8:A2')).toEqual({startRow:2,startColumn:1,endRow:8,endColumn:3}))
  it('parses typed values',()=>expect(['12','true','null','text'].map(parseFilterInput)).toEqual([12,true,null,'text']))
  it('materializes values and color criteria',()=>{
    expect(filterCriterionInput(1,'values','Seoul, 10, true','#ffffff',false)).toEqual({column:1,operator:'values',values:['Seoul',10,true]})
    expect(filterCriterionInput(2,'background_color','', '#FEF3C7',false)).toEqual({column:2,operator:'background_color',color:'#FEF3C7'})
  })
})
