import { parseFilterInput,parseFilterRange } from './filter'
import type { DataValidation,ValidationOperator,ValidationOption,ValidationViolation } from '../types'
import { spreadsheetDate } from './cellFormat'
import { parseValidationDate } from './validationDate'

export type ValidationInputCell={row:number;column:number;value?:unknown;formula?:string}

export function validationForCell(rules:DataValidation[],row:number,column:number){
  return rules.find(rule=>{const range=parseFilterRange(rule.range);return Boolean(range&&row>=range.startRow&&row<=range.endRow&&column>=range.startColumn&&column<=range.endColumn)})
}

export function ruleOptions(rule:DataValidation){return rule.rule_type==='list_range'?rule.source_options:rule.options}
export function optionForValue(rule:DataValidation,value:unknown){return ruleOptions(rule)?.find(option=>sameValue(option.value,value))}
export function optionLabel(option:ValidationOption){return option.label?.trim()||String(option.value)}
export function validationDraftValue(value:unknown){return value==null?'':String(value)}
export function validationOptionInput(value:string,label:string,color:string):ValidationOption{return{value:parseFilterInput(value),...(label.trim()?{label:label.trim()}:{}),...(color?{color}: {})}}

export function validateClientValue(rule:DataValidation,value:unknown):{valid:boolean;deferred:boolean;message:string}{
  if(value==null||value==='')return{valid:rule.allow_blank,deferred:false,message:rule.help_text||'빈 값은 허용되지 않습니다.'}
  let valid=false,deferred=false
  if(rule.rule_type==='list'||rule.rule_type==='list_range'||rule.rule_type==='checkbox')valid=Boolean(optionForValue(rule,value))
  else if(rule.rule_type==='number')valid=typeof value==='number'&&Number.isFinite(value)&&compare(value,Number(rule.value),Number(rule.value2),rule.operator)
  else if(rule.rule_type==='date'){const actual=dateValue(value),first=dateValue(rule.value),second=dateValue(rule.value2);valid=actual!==undefined&&first!==undefined&&compare(actual,first,second??0,rule.operator)}
  else{valid=true;deferred=true}
  const fallback=rule.rule_type==='checkbox'?'체크 상태를 나타내는 두 값 중 하나여야 합니다.':rule.rule_type==='list'||rule.rule_type==='list_range'?'목록에 있는 값을 선택해야 합니다.':rule.rule_type==='number'?'숫자 검증 조건을 만족하지 않습니다.':rule.rule_type==='date'?'날짜 검증 조건을 만족하지 않습니다.':'사용자 지정 수식은 서버에서 검사됩니다.'
  return{valid,deferred,message:rule.help_text||fallback}
}

export function validateClientInputs(rules:DataValidation[],inputs:ValidationInputCell[]){
  const rejected:ValidationViolation[]=[],warnings:ValidationViolation[]=[]
  for(const input of inputs){if(input.formula)continue;const rule=validationForCell(rules,input.row,input.column);if(!rule)continue;const result=validateClientValue(rule,input.value);if(result.valid||result.deferred)continue;const violation={validation_id:rule.id,row:input.row,column:input.column,message:result.message};(rule.reject_input?rejected:warnings).push(violation)}
  return{rejected,warnings}
}

export function comparisonNeedsSecond(operator:ValidationOperator){return operator==='between'||operator==='not_between'}

function sameValue(left:unknown,right:unknown){return typeof left===typeof right&&Object.is(left,right)}
function compare(actual:number,first:number,second:number,operator:ValidationOperator){if(!Number.isFinite(first))return false;switch(operator){case'between':return Number.isFinite(second)&&actual>=first&&actual<=second;case'not_between':return Number.isFinite(second)&&(actual<first||actual>second);case'equal':return actual===first;case'not_equal':return actual!==first;case'greater_than':return actual>first;case'greater_or_equal':return actual>=first;case'less_than':return actual<first;case'less_or_equal':return actual<=first;default:return false}}
// 날 수를 날짜로 바꾸는 셈은 격자와 같은 것을 쓴다. 여기서 따로 세던
// 시절에는 1900년 윤년 어긋남을 보지 않아, 격자가 1900-01-01로 그리는 칸을
// 검증은 1899-12-31로 읽었다.
function dateValue(value:unknown){
  if(typeof value==='number'){const moment=spreadsheetDate(value);return moment?moment.getTime():undefined}
  if(typeof value!=='string')return
  return parseValidationDate(value)
}

/**
 * The two values a checkbox toggles between. The first option is the checked
 * state, which is the order the server normalises them into.
 */
export function checkboxValues(rule:DataValidation){
  const options=rule.options??[]
  if(rule.rule_type!=='checkbox'||options.length!==2)return undefined
  return {checked:options[0].value,unchecked:options[1].value}
}

/** Whether a cell reads as checked, and what writing the opposite means. */
export function checkboxState(rule:DataValidation,value:unknown){
  const values=checkboxValues(rule)
  if(!values)return undefined
  const checked=sameValue(values.checked,value)
  return {checked,next:checked?values.unchecked:values.checked}
}
