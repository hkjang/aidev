import type { FilterCriterion,FilterOperator } from '../types'

export type ParsedFilterRange={startRow:number;startColumn:number;endRow:number;endColumn:number}

export function columnName(column:number){let value=column,result='';while(value>0){value-=1;result=String.fromCharCode(65+value%26)+result;value=Math.floor(value/26)}return result}
export function columnNumber(name:string){let value=0;for(const character of name.toUpperCase())value=value*26+character.charCodeAt(0)-64;return value}
export function parseFilterRange(input:string):ParsedFilterRange|undefined{
  const match=input.trim().match(/^([A-Za-z]{1,3})([1-9][0-9]*):([A-Za-z]{1,3})([1-9][0-9]*)$/);if(!match)return
  const first={row:Number(match[2]),column:columnNumber(match[1])},second={row:Number(match[4]),column:columnNumber(match[3])}
  return{startRow:Math.min(first.row,second.row),startColumn:Math.min(first.column,second.column),endRow:Math.max(first.row,second.row),endColumn:Math.max(first.column,second.column)}
}
export function parseFilterInput(input:string){const value=input.trim();if(value==='')return'';if(value.toLowerCase()==='true')return true;if(value.toLowerCase()==='false')return false;if(value.toLowerCase()==='null')return null;if(Number.isFinite(Number(value)))return Number(value);return value}
export function filterCriterionInput(column:number,operator:FilterOperator,text:string,color:string,caseSensitive:boolean):FilterCriterion{
  const result:FilterCriterion={column,operator}
  if(operator==='values')result.values=text.split(',').map(parseFilterInput)
  else if(operator==='background_color'||operator==='text_color')result.color=color
  else if(operator!=='is_blank'&&operator!=='is_not_blank')result.value=parseFilterInput(text)
  if(caseSensitive)result.case_sensitive=true
  return result
}
