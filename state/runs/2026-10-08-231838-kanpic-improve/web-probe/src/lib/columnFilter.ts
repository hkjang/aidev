import type { Cell, FilterCriterion, FilterView } from '../types'
import { parseFilterRange } from './filter'

export type FilterValue={label:string;raw:unknown;count:number;checked:boolean}

/** The blank entry is offered like any other value so empty rows can be hidden. */
export const BLANK_LABEL='(빈 값)'

const label=(value:unknown)=>value==null||value===''?BLANK_LABEL:typeof value==='number'?String(value):String(value)

/** The criterion a column's value list is stored in, if it has one. */
export function valuesCriterion(view:FilterView|undefined,column:number){
  return view?.criteria?.find(criterion=>criterion.column===column&&criterion.operator==='values')
}

/** Whether the column is filtered at all, which is what the header glyph shows. */
export function columnFiltered(view:FilterView|undefined,column:number){
  return Boolean(view?.criteria?.some(criterion=>criterion.column===column))
}

/**
 * The distinct values of one column inside the filter range, most frequent
 * first, each marked with whether the current criterion keeps it. With no
 * criterion every value is kept, which is what an unfiltered column means.
 */
export function columnValues(cells:Cell[],view:FilterView,column:number):FilterValue[]{
  const range=parseFilterRange(view.range)
  if(!range||column<range.startColumn||column>range.endColumn)return []
  const firstRow=range.startRow+Math.max(0,view.header_rows??0)
  const counts=new Map<string,{raw:unknown;count:number}>()
  const seen=new Set<number>()
  for(const cell of cells){
    if(cell.column!==column||cell.row<firstRow||cell.row>range.endRow)continue
    seen.add(cell.row)
    const key=label(cell.value)
    const existing=counts.get(key)
    if(existing)existing.count+=1
    else counts.set(key,{raw:cell.value??null,count:1})
  }
  // Rows with nothing in this column still count as blanks.
  const blanks=Math.max(0,range.endRow-firstRow+1-seen.size)
  if(blanks>0){
    const existing=counts.get(BLANK_LABEL)
    if(existing)existing.count+=blanks
    else counts.set(BLANK_LABEL,{raw:null,count:blanks})
  }
  const criterion=valuesCriterion(view,column)
  const kept=criterion?new Set(criterion.values?.map(value=>label(value))??[]):undefined
  return [...counts.entries()]
    .sort((first,second)=>{
      // Blanks belong at the end however many there are; everything else goes
      // by how often it appears.
      if((first[0]===BLANK_LABEL)!==(second[0]===BLANK_LABEL))return first[0]===BLANK_LABEL?1:-1
      return second[1].count-first[1].count||first[0].localeCompare(second[0],'ko-KR')
    })
    .map(([text,item])=>({label:text,raw:item.raw,count:item.count,checked:kept?kept.has(text):true}))
}

/**
 * The criteria to save after a change to one column's value list. Keeping
 * every value means the column has no criterion at all, so the filter stays
 * as small as what it actually restricts.
 */
export function withColumnValues(view:FilterView,column:number,values:FilterValue[]):FilterCriterion[]{
  const others=(view.criteria??[]).filter(criterion=>criterion.column!==column||criterion.operator!=='values')
  const checked=values.filter(value=>value.checked)
  if(checked.length===values.length)return others
  return [...others,{column,operator:'values',values:checked.map(value=>value.raw)}]
}
