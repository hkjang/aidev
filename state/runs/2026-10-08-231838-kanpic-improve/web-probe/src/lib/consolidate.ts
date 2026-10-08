import type { Cell } from '../types'
import { cellKey } from '../state/editor'
import type { GridRegion } from './dataRegion'
import { spreadsheetNumber } from './spreadsheetNumber'
import { compareKey } from './compareLists'
import { mergeInRegion } from './merge'

export type ConsolidateFunction='sum'|'count'|'average'|'max'|'min'
export const CONSOLIDATE_LABELS:Record<ConsolidateFunction,string>={sum:'합계',count:'개수',average:'평균',max:'최대',min:'최소'}

export type ConsolidateSource={sheetName:string;cells:Map<string,Cell>;region:GridRegion}

export type ConsolidateResult={
  rowLabels:string[]
  columnLabels:string[]
  /** `${rowIndex}:${columnIndex}` → 셈한 값. 어느 시트에도 수가 없으면 없다. */
  values:Map<string,number>
  /** 숫자로 세지 않은 글자 칸의 수. =SUM 과 같은 규칙으로 세므로 "1,234" 는 빠진다. */
  skippedText:number
  /** 병합된 칸이 있는 시트. 이름표가 빈 행이 생겨 조용히 어긋난다. */
  mergedSheets:string[]
  /** 어느 시트에 그 이름표가 아예 없었는지. 없는 것과 0인 것은 다르다. */
  missing:Array<{sheetName:string;label:string}>
}

type Bucket={values:number[]}

/**
 * 여러 시트의 같은 모양 표를 이름표 기준으로 합친다.
 *
 * 1월·2월·3월 시트가 부서 × 항목으로 되어 있으면 부서와 항목을 맞춰 더한다.
 * 시트마다 부서 차례가 다르거나 어느 달에만 있는 부서가 있어도 이름표로
 * 맞추므로 자리를 손으로 맞출 필요가 없다.
 *
 * 무엇이 숫자인지는 수식 엔진이 정하는 규칙을 그대로 쓴다. 화면이 더 너그럽게
 * 세면 =SUM 이 내는 값과 통합표의 값이 달라진다. 대신 세지 않은 글자 칸이
 * 몇 개인지 세어 알린다 — 조용히 빼면 합계가 작게 나오고 이유는 아무 데도
 * 적히지 않는다.
 */
export function consolidate(sources:ConsolidateSource[],operation:ConsolidateFunction):ConsolidateResult{
  const rowLabels:string[]=[],columnLabels:string[]=[]
  const rowIndex=new Map<string,number>(),columnIndex=new Map<string,number>()
  const buckets=new Map<string,Bucket>()
  const seenPerSheet=new Map<string,Set<string>>()
  let skippedText=0
  const mergedSheets:string[]=[]

  for(const source of sources){
    if(mergeInRegion(source.cells,source.region))mergedSheets.push(source.sheetName)
    const seen=new Set<string>()
    seenPerSheet.set(source.sheetName,seen)
    for(let column=source.region.startColumn+1;column<=source.region.endColumn;column+=1){
      const key=compareKey(source.cells.get(cellKey(source.region.startRow,column))?.value)
      if(key===undefined)continue
      if(!columnIndex.has(key)){columnIndex.set(key,columnLabels.length);columnLabels.push(displayText(source.cells.get(cellKey(source.region.startRow,column))))}
    }
    for(let row=source.region.startRow+1;row<=source.region.endRow;row+=1){
      const labelCell=source.cells.get(cellKey(row,source.region.startColumn))
      const rowKey=compareKey(labelCell?.value)
      if(rowKey===undefined)continue
      seen.add(rowKey)
      if(!rowIndex.has(rowKey)){rowIndex.set(rowKey,rowLabels.length);rowLabels.push(displayText(labelCell))}
      for(let column=source.region.startColumn+1;column<=source.region.endColumn;column+=1){
        const columnKey=compareKey(source.cells.get(cellKey(source.region.startRow,column))?.value)
        if(columnKey===undefined)continue
        const cell=source.cells.get(cellKey(row,column))
        const number=spreadsheetNumber(cell?.value)
        if(number===undefined){
          if(typeof cell?.value==='string'&&cell.value.trim()!=='')skippedText+=1
          continue
        }
        const slot=`${rowIndex.get(rowKey)}:${columnIndex.get(columnKey)}`
        const bucket=buckets.get(slot)??{values:[]}
        bucket.values.push(number)
        buckets.set(slot,bucket)
      }
    }
  }

  const values=new Map<string,number>()
  for(const [slot,bucket] of buckets)values.set(slot,apply(bucket.values,operation))

  // 어느 시트에 그 이름표가 아예 없었는지. 없는 것과 0인 것은 다르다 —
  // 부서가 통째로 빠진 것을 0원 실적으로 읽으면 안 된다.
  const missing:Array<{sheetName:string;label:string}>=[]
  for(const source of sources){
    const seen=seenPerSheet.get(source.sheetName)
    if(!seen)continue
    for(const [key,index] of rowIndex)if(!seen.has(key))missing.push({sheetName:source.sheetName,label:rowLabels[index]})
  }
  return {rowLabels,columnLabels,values,skippedText,mergedSheets,missing}
}

function apply(values:number[],operation:ConsolidateFunction){
  switch(operation){
    case 'count':return values.length
    case 'average':return values.reduce((total,value)=>total+value,0)/values.length
    case 'max':return values.reduce((best,value)=>value>best?value:best,values[0])
    case 'min':return values.reduce((best,value)=>value<best?value:best,values[0])
    default:return values.reduce((total,value)=>total+value,0)
  }
}

const displayText=(cell?:Cell)=>cell?.value==null?'':typeof cell.value==='string'?cell.value:String(cell.value)
