import type { Cell } from '../types'
import { spreadsheetNumber } from './spreadsheetNumber'

export type SummaryKey='sum'|'average'|'count'|'counta'|'min'|'max'

export const SUMMARY_LABELS:Record<SummaryKey,string>={
  sum:'합계',average:'평균',count:'숫자 개수',counta:'개수',min:'최소',max:'최대',
}

/** What the status bar shows unless somebody picks something else. */
export const DEFAULT_SUMMARY:SummaryKey[]=['sum','average','counta']

export type SelectionStats={
  /** Cells holding a number, which is what the arithmetic runs over. */
  numbers:number
  /** Cells holding anything at all. */
  filled:number
  sum:number
  average:number
  min:number
  max:number
}

// 셈하는 규칙은 서버의 수식 엔진이 정한다. 여기서 따로 세면 =SUM 이 내는
// 값과 화면에 보이는 합계가 달라진다.
const numeric=spreadsheetNumber

/**
 * Summarises the selected cells the way a spreadsheet status bar does: text
 * and empty cells are counted but never averaged, and a formula's calculated
 * value is what counts, not its text.
 */
export function summarizeSelection(cells:Map<string,Cell>,startRow:number,startColumn:number,endRow:number,endColumn:number):SelectionStats{
  let numbers=0,filled=0,sum=0,min=Number.POSITIVE_INFINITY,max=Number.NEGATIVE_INFINITY
  for(let row=startRow;row<=endRow;row+=1){
    for(let column=startColumn;column<=endColumn;column+=1){
      const cell=cells.get(`${row}:${column}`)
      if(!cell)continue
      const value=cell.value
      if(value===null||value===undefined||value==='')continue
      filled+=1
      const parsed=numeric(value)
      if(parsed===undefined)continue
      numbers+=1
      sum+=parsed
      min=Math.min(min,parsed)
      max=Math.max(max,parsed)
    }
  }
  return {
    numbers,filled,sum,
    average:numbers>0?sum/numbers:0,
    min:numbers>0?min:0,
    max:numbers>0?max:0,
  }
}

/** Formats one statistic for the status bar, keeping long numbers readable. */
export function formatStat(key:SummaryKey,stats:SelectionStats){
  if(key==='count')return stats.numbers.toLocaleString('ko-KR')
  if(key==='counta')return stats.filled.toLocaleString('ko-KR')
  const value=key==='sum'?stats.sum:key==='average'?stats.average:key==='min'?stats.min:stats.max
  // Two decimals at most, and none at all when the number is whole, which is
  // what keeps a column of money readable.
  return value.toLocaleString('ko-KR',{maximumFractionDigits:2})
}

/**
 * The summary is only worth showing when there is something to summarise: a
 * single cell repeats what the grid already shows.
 */
export function shouldSummarize(stats:SelectionStats,cellCount:number){
  return cellCount>1&&stats.filled>0
}
