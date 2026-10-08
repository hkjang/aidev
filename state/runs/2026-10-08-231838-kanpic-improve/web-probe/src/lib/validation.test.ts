import { describe,expect,it } from 'vitest'
import { checkboxState,checkboxValues,optionForValue,validateClientInputs,validateClientValue,validationForCell,validationOptionInput } from './validation'
import type { DataValidation } from '../types'

const list:DataValidation={id:'list',workbook_id:'w',workbook_version:2,sheet_id:'s',range:'B2:B4',rule_type:'list',operator:'in_list',options:[{value:'open',label:'Open',color:'#dcfce7'},{value:2}],allow_blank:true,reject_input:true,show_dropdown:true,display_style:'chip',revision:1,created_by:'a',updated_by:'a',created_at:'',updated_at:''}

describe('data validation helpers',()=>{
  it('finds a rule and preserves typed dropdown values',()=>{expect(validationForCell([list],3,2)?.id).toBe('list');expect(validationForCell([list],3,3)).toBeUndefined();expect(optionForValue(list,2)?.value).toBe(2);expect(optionForValue(list,'2')).toBeUndefined();expect(validationOptionInput('true','Yes','#ffffff')).toEqual({value:true,label:'Yes',color:'#ffffff'})})
  it('validates list, number and date conditions',()=>{expect(validateClientValue(list,'open').valid).toBe(true);expect(validateClientValue(list,'other').valid).toBe(false);const number={...list,id:'number',range:'A1:A2',rule_type:'number' as const,operator:'between' as const,value:10,value2:20,options:undefined,display_style:'plain' as const,show_dropdown:false};expect(validateClientValue(number,15).valid).toBe(true);expect(validateClientValue(number,'15').valid).toBe(false);const date={...number,id:'date',rule_type:'date' as const,value:'2026-01-01',value2:'2026-12-31'};expect(validateClientValue(date,'2026-07-31').valid).toBe(true)})
  it('separates rejected values from warning-only values and defers formulas',()=>{const warning={...list,id:'warning',range:'C1:C1',reject_input:false};const result=validateClientInputs([list,warning],[{row:2,column:2,value:'bad'},{row:1,column:3,value:'bad'},{row:3,column:2,formula:'=A1'}]);expect(result.rejected).toHaveLength(1);expect(result.warnings).toHaveLength(1)})
})

describe('checkbox rules', () => {
  const rule=(options:Array<{value:unknown}>)=>({
    id:'v1',workbook_id:'w',workbook_version:1,sheet_id:'s',range:'B2:B10',rule_type:'checkbox',operator:'in_list',
    options,allow_blank:true,reject_input:false,show_dropdown:false,display_style:'plain',revision:1,
    created_by:'a',updated_by:'a',created_at:'',updated_at:'',
  }) as unknown as DataValidation

  it('reads the checked state and knows what the other state is', () => {
    const boolean=rule([{value:true},{value:false}])
    expect(checkboxState(boolean,true)).toEqual({checked:true,next:false})
    expect(checkboxState(boolean,false)).toEqual({checked:false,next:true})
    // An empty cell counts as unchecked, so the first click checks it.
    expect(checkboxState(boolean,undefined)).toEqual({checked:false,next:true})
  })

  it('works with the sheet own pair of values', () => {
    const korean=rule([{value:'예'},{value:'아니오'}])
    expect(checkboxState(korean,'예')).toEqual({checked:true,next:'아니오'})
    expect(checkboxState(korean,'아니오')).toEqual({checked:false,next:'예'})
  })

  it('accepts only the two values it toggles between', () => {
    const boolean=rule([{value:true},{value:false}])
    expect(validateClientValue(boolean,true).valid).toBe(true)
    expect(validateClientValue(boolean,'예').valid).toBe(false)
  })

  it('is not a checkbox without exactly two values', () => {
    expect(checkboxValues(rule([{value:true}]))).toBeUndefined()
  })
})
