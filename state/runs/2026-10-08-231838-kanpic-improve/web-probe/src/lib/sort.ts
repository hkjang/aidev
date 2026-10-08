import { compareNatural } from './naturalOrder'
import type { Cell } from '../types'
import { cellKey } from '../state/editor'
import { shiftFormula,MAX_SORT_CELLS } from './clipboard'
import { cellMerge,type MergeRange } from './merge'

export type SortKey = { column:number; direction:'asc'|'desc' }
export type SortOptions = { keys:SortKey[]; headerRows:number; caseSensitive:boolean; literalOrder?:boolean }

type Scalar = { rank:number; blank:boolean; number?:number; text?:string; truth?:boolean }

export function materializeSort(cells:Map<string,Cell>,range:MergeRange,options:SortOptions,sheetId:string){
  const rows=range.endRow-range.startRow+1,columns=range.endColumn-range.startColumn+1,dataRows=rows-options.headerRows
  if(rows<2||columns<1||options.headerRows<0||options.headerRows>=rows||dataRows<2)throw new Error('정렬 범위에는 데이터 행이 두 개 이상 있어야 합니다.')
  if(dataRows*columns>MAX_SORT_CELLS)throw new Error(`정렬은 한 번에 최대 ${MAX_SORT_CELLS.toLocaleString()}셀(${Math.floor(MAX_SORT_CELLS/Math.max(1,columns)).toLocaleString()}행 × ${columns}열)까지 가능합니다.`)
  if(options.keys.length<1||options.keys.length>columns)throw new Error('정렬 기준 열을 하나 이상 선택하세요.')
  const used=new Set<number>()
  for(const key of options.keys){if(key.column<range.startColumn||key.column>range.endColumn||used.has(key.column))throw new Error('정렬 기준 열은 선택 범위 안에서 중복 없이 지정해야 합니다.');used.add(key.column)}
  const dataStart=range.startRow+options.headerRows
  for(let row=dataStart;row<=range.endRow;row+=1)for(let column=range.startColumn;column<=range.endColumn;column+=1){const cell=cells.get(cellKey(row,column));if(cell?.spill_source)throw new Error(`${cell.spill_source} 배열 수식의 결과 셀은 정렬할 수 없습니다.`);if(cellMerge(cell))throw new Error('병합된 셀은 병합 해제 후 정렬할 수 있습니다.')}
  const records=Array.from({length:dataRows},(_,offset)=>{const originalRow=dataStart+offset;return{originalRow,values:options.keys.map(key=>scalar(cells.get(cellKey(originalRow,key.column))?.value,options.caseSensitive))}})
  records.sort((left,right)=>{for(let index=0;index<options.keys.length;index+=1){const comparison=compare(left.values[index],right.values[index],options.literalOrder===true);if(comparison!==0){if(left.values[index].blank||right.values[index].blank)return comparison;return comparison*(options.keys[index].direction==='asc'?1:-1)}}return left.originalRow-right.originalRow})
  const updatedAt=new Date().toISOString(),result:Cell[]=[]
  records.forEach((record,offset)=>{const destinationRow=dataStart+offset;for(let column=range.startColumn;column<=range.endColumn;column+=1){const source=cells.get(cellKey(record.originalRow,column));const formula=source?.formula?shiftFormula(source.formula,destinationRow-record.originalRow,0):undefined;result.push({sheet_id:sheetId,row:destinationRow,column,value:formula?undefined:source?.value,formula,style:source?.style?{...source.style}:undefined,updated_at:updatedAt})}})
  return result
}

function scalar(value:unknown,caseSensitive:boolean):Scalar{
  if(value===undefined||value===null)return{rank:4,blank:true}
  if(typeof value==='number')return{rank:0,blank:false,number:value}
  if(typeof value==='string')return{rank:1,blank:false,text:caseSensitive?value:value.toLowerCase()}
  if(typeof value==='boolean')return{rank:2,blank:false,truth:value}
  return{rank:3,blank:false,text:JSON.stringify(value)}
}

function compare(left:Scalar,right:Scalar,literal:boolean){
  if(left.blank!==right.blank)return left.blank?1:-1
  if(left.rank!==right.rank)return left.rank-right.rank
  if(left.rank===0)return left.number!<right.number!?-1:left.number!>right.number!?1:0
  if(left.rank===2)return left.truth===right.truth?0:left.truth?1:-1
  if(literal)return left.text!<right.text!?-1:left.text!>right.text!?1:0
  return compareNatural(left.text??'',right.text??'')
}
