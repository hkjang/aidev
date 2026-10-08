import type { Cell } from '../types'
import type { PastedCell } from './clipboard'
import type { GridRegion } from './dataRegion'
import { cellKey } from '../state/editor'
import { cleanupText } from './dataCleanup'
import { address } from './api'

export type SubtotalAggregation='sum'|'average'|'count'|'max'|'min'
export type SubtotalRun={label:string;startRow:number;endRow:number}
export type SubtotalPlan={
  runs:SubtotalRun[]
  writes:PastedCell[]
  /** Row spans to fold away, in the coordinates of the rewritten block. */
  groups:Array<{start:number;end:number}>
  addedRows:number
  /** True when the group column repeats a label it already left behind. */
  unsorted:boolean
  lastRow:number
}

// SUBTOTAL codes, using the 1xx family so a grand total never counts the
// subtotals underneath it twice.
const CODES:Record<SubtotalAggregation,number>={sum:109,average:101,count:103,max:104,min:105}
const NAMES:Record<SubtotalAggregation,string>={sum:'합계',average:'평균',count:'개수',max:'최대',min:'최소'}
// SUBTOTAL over a span that contains other subtotals would count those numbers
// twice: the engine sees values, not the formulas that produced them, so it
// cannot tell a subtotal apart from data. Every total therefore names only the
// runs of data it covers.
// A grand total naming every run gets unreadable past a few dozen groups.
export const MaxGrandTotalRuns=30
export const subtotalName=(aggregation:SubtotalAggregation)=>NAMES[aggregation]

/**
 * Plans the subtotal rows for a block grouped by one column. The rows are
 * planned rather than written directly because inserting them changes every
 * row number after the first group, and a plan can be shown before any of
 * that happens.
 */
export function planSubtotals(cells:Map<string,Cell>,region:GridRegion,options:{
  groupColumn:number
  valueColumns:number[]
  aggregation:SubtotalAggregation
  headerRows:number
  grandTotal:boolean
}):SubtotalPlan{
  const {groupColumn,valueColumns,aggregation,headerRows,grandTotal}=options
  const first=region.startRow+headerRows
  const runs:SubtotalRun[]=[]
  const seen=new Set<string>()
  let unsorted=false
  for(let row=first;row<=region.endRow;row+=1){
    const value=cleanupText(cells.get(cellKey(row,groupColumn)))
    const current=runs[runs.length-1]
    if(current&&current.label===value){current.endRow=row;continue}
    if(seen.has(value))unsorted=true
    seen.add(value)
    runs.push({label:value,startRow:row,endRow:row})
  }
  const writes:PastedCell[]=[]
  const groups:Array<{start:number;end:number}>=[]
  let offset=0
  // Rows are copied down as the subtotal rows push them, so the block is
  // rewritten in place rather than relying on structural inserts.
  for(const run of runs){
    for(let row=run.startRow;row<=run.endRow;row+=1){
      const target=row+offset
      if(target!==row)for(let column=region.startColumn;column<=region.endColumn;column+=1){
        const cell=cells.get(cellKey(row,column))
        writes.push({row:target,column,value:cell?.formula?undefined:cell?.value,formula:cell?.formula,style:cell?.style})
      }
    }
    const totalRow=run.endRow+offset+1
    groups.push({start:run.startRow+offset,end:run.endRow+offset})
    // Every cell of the subtotal row is written, including the columns that get
    // no total: the row it lands on used to hold data, and leaving a stray
    // value beside a total reads as part of the total.
    for(let column=region.startColumn;column<=region.endColumn;column+=1){
      if(column===region.startColumn){writes.push({row:totalRow,column,value:`${run.label||'(빈 값)'} ${NAMES[aggregation]}`,formula:undefined});continue}
      if(valueColumns.includes(column)){
        const range=`${address(run.startRow+offset,column)}:${address(run.endRow+offset,column)}`
        writes.push({row:totalRow,column,formula:`=SUBTOTAL(${CODES[aggregation]},${range})`,value:undefined})
        continue
      }
      writes.push({row:totalRow,column,value:undefined,formula:undefined})
    }
    offset+=1
  }
  const lastRow=region.endRow+offset
  const placed=runs.map((run,index)=>({start:run.startRow+index,end:run.endRow+index}))
  if(grandTotal&&runs.length>1&&runs.length<=MaxGrandTotalRuns){
    const grandRow=lastRow+1
    for(let column=region.startColumn;column<=region.endColumn;column+=1){
      if(column===region.startColumn){writes.push({row:grandRow,column,value:`전체 ${NAMES[aggregation]}`,formula:undefined});continue}
      if(!valueColumns.includes(column)){writes.push({row:grandRow,column,value:undefined,formula:undefined});continue}
      const ranges=placed.map(item=>`${address(item.start,column)}:${address(item.end,column)}`).join(',')
      // The grand total names the data runs rather than the span that contains
      // the subtotals, so nothing is counted twice, and it stays a SUBTOTAL
      // call so "remove subtotals" can recognise the row it added.
      writes.push({row:grandRow,column,formula:`=SUBTOTAL(${CODES[aggregation]},${ranges})`,value:undefined})
    }
    offset+=1
  }
  return {runs,writes,groups,addedRows:offset,unsorted,lastRow:region.endRow+offset}
}

/** True when a cell carries one of the formulas a subtotal run writes. */
export function isSubtotalFormula(formula:string|undefined){
  return typeof formula==='string'&&/^=\s*SUBTOTAL\s*\(/i.test(formula.trim())
}

export type SubtotalRemoval={rows:Array<{row:number;label:string}>;writes:PastedCell[]}

/**
 * Plans the removal of the rows a subtotal run added. Rows are recognised by
 * the formula they carry rather than by the label somebody may have edited.
 */
export function planRemoveSubtotals(cells:Map<string,Cell>,region:GridRegion):SubtotalRemoval{
  const removed:Array<{row:number;label:string}>=[]
  for(let row=region.startRow;row<=region.endRow;row+=1){
    let found=false
    for(let column=region.startColumn;column<=region.endColumn&&!found;column+=1)
      if(isSubtotalFormula(cells.get(cellKey(row,column))?.formula))found=true
    if(found)removed.push({row,label:cleanupText(cells.get(cellKey(row,region.startColumn)))})
  }
  if(removed.length===0)return {rows:[],writes:[]}
  const drop=new Set(removed.map(item=>item.row))
  const writes:PastedCell[]=[]
  let target=region.startRow
  for(let row=region.startRow;row<=region.endRow;row+=1){
    if(drop.has(row))continue
    if(target!==row)for(let column=region.startColumn;column<=region.endColumn;column+=1){
      const cell=cells.get(cellKey(row,column))
      writes.push({row:target,column,value:cell?.formula?undefined:cell?.value,formula:cell?.formula,style:cell?.style})
    }
    target+=1
  }
  // The rows freed at the bottom are emptied rather than left repeating the
  // last rows of the table.
  for(let row=target;row<=region.endRow;row+=1)
    for(let column=region.startColumn;column<=region.endColumn;column+=1)
      writes.push({row,column,value:undefined,formula:undefined})
  return {rows:removed,writes}
}
