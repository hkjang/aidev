import type { Cell } from '../types'
import { spreadsheetNumber } from './spreadsheetNumber'

export type ValueCount={value:string;count:number;share:number}

export type ColumnStats={
  /** Cells inspected, including the empty ones. */
  scanned:number
  filled:number
  empty:number
  unique:number
  numbers:number
  /** Present only when the column holds numbers. */
  sum?:number
  average?:number
  median?:number
  min?:number
  max?:number
  deviation?:number
  /** The values that appear most often, longest bar first. */
  frequent:ValueCount[]
  /** Equal-width buckets over the numbers, for the distribution chart. */
  buckets:Array<{from:number;to:number;count:number}>
}

const MAX_FREQUENT=8
const BUCKETS=10

const text=(value:unknown)=>value==null?'':typeof value==='number'?String(value):String(value).trim()

// 셈하는 규칙은 서버의 수식 엔진이 정한다. 여기서 따로 세면 =SUM 이 내는
// 값과 화면에 보이는 합계가 달라진다.
const numeric=spreadsheetNumber

/**
 * Whether the first row is a label rather than data: text on top of a column
 * that holds numbers. Counting a header as a value would make every column
 * look like it has one more distinct entry than it does.
 */
export function looksLikeHeader(cells:Cell[],column:number,startRow:number,endRow:number){
  const byRow=new Map<number,Cell>()
  for(const cell of cells)if(cell.column===column)byRow.set(cell.row,cell)
  const first=byRow.get(startRow)?.value
  if(typeof first!=='string'||first.trim()==='')return false
  for(let row=startRow+1;row<=endRow;row+=1){
    const value=byRow.get(row)?.value
    if(value===undefined||value===null||value==='')continue
    return numeric(value)!==undefined
  }
  return false
}

/**
 * Summarises one column the way a data review starts: how much is filled, how
 * many distinct values there are, which values repeat, and — when the column
 * is numeric — where the numbers sit.
 */
export function columnStats(cells:Cell[],column:number,startRow:number,endRow:number):ColumnStats{
  const byRow=new Map<number,Cell>()
  for(const cell of cells)if(cell.column===column)byRow.set(cell.row,cell)
  const counts=new Map<string,number>()
  const numbers:number[]=[]
  let filled=0
  for(let row=startRow;row<=endRow;row+=1){
    const value=byRow.get(row)?.value
    const label=text(value)
    if(label===''){continue}
    filled+=1
    counts.set(label,(counts.get(label)??0)+1)
    const parsed=numeric(value)
    if(parsed!==undefined)numbers.push(parsed)
  }
  const scanned=Math.max(0,endRow-startRow+1)
  const stats:ColumnStats={
    scanned,filled,empty:scanned-filled,unique:counts.size,numbers:numbers.length,
    frequent:[...counts.entries()]
      .sort((first,second)=>second[1]-first[1]||first[0].localeCompare(second[0],'ko-KR'))
      .slice(0,MAX_FREQUENT)
      .map(([value,count])=>({value,count,share:filled>0?count/filled:0})),
    buckets:[],
  }
  if(numbers.length===0)return stats
  const sorted=[...numbers].sort((first,second)=>first-second)
  const sum=numbers.reduce((total,value)=>total+value,0)
  const average=sum/numbers.length
  const middle=Math.floor(sorted.length/2)
  stats.sum=sum
  stats.average=average
  stats.median=sorted.length%2===1?sorted[middle]:(sorted[middle-1]+sorted[middle])/2
  stats.min=sorted[0]
  stats.max=sorted[sorted.length-1]
  // The sample deviation needs two numbers; one number has no spread.
  stats.deviation=numbers.length>1
    ? Math.sqrt(numbers.reduce((total,value)=>total+(value-average)**2,0)/(numbers.length-1))
    : 0
  stats.buckets=distribute(sorted)
  return stats
}

/** Equal-width buckets across the range, so the shape of the data is visible. */
function distribute(sorted:number[]){
  const low=sorted[0],high=sorted[sorted.length-1]
  if(low===high)return [{from:low,to:high,count:sorted.length}]
  const width=(high-low)/BUCKETS
  const buckets=Array.from({length:BUCKETS},(_unused,index)=>({from:low+index*width,to:low+(index+1)*width,count:0}))
  for(const value of sorted){
    const index=Math.min(BUCKETS-1,Math.floor((value-low)/width))
    buckets[index].count+=1
  }
  return buckets
}

/** Formats a number for the panel without drowning it in decimals. */
export function statNumber(value:number|undefined){
  if(value===undefined)return '—'
  return value.toLocaleString('ko-KR',{maximumFractionDigits:2})
}
