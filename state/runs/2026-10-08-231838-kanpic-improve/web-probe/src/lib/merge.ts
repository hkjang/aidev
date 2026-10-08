import type { Cell } from '../types'
import { cellKey } from '../state/editor'

export type MergeRange = { startRow:number; startColumn:number; endRow:number; endColumn:number }

type StoredMerge = { start_row:number; start_column:number; end_row:number; end_column:number }

export function cellMerge(cell:Cell|undefined):MergeRange|undefined {
  const raw=cell?.style?.merge
  if(!raw||typeof raw!=='object')return
  const value=raw as Partial<StoredMerge>
  if(!Number.isInteger(value.start_row)||!Number.isInteger(value.start_column)||!Number.isInteger(value.end_row)||!Number.isInteger(value.end_column))return
  const range={startRow:value.start_row!,startColumn:value.start_column!,endRow:value.end_row!,endColumn:value.end_column!}
  if(range.startRow<1||range.startColumn<1||range.endRow<range.startRow||range.endColumn<range.startColumn||!cell||cell.row<range.startRow||cell.row>range.endRow||cell.column<range.startColumn||cell.column>range.endColumn)return
  return range
}

export function selectedMergedBounds(cells:Map<string,Cell>,selection:MergeRange):MergeRange {
  if(selection.startRow!==selection.endRow||selection.startColumn!==selection.endColumn)return selection
  return cellMerge(cells.get(cellKey(selection.startRow,selection.startColumn)))??selection
}

export function mergeStyle(style:Record<string,unknown>|undefined,range:MergeRange,merge:boolean){
  const next={...(style??{})}
  if(merge)next.merge={start_row:range.startRow,start_column:range.startColumn,end_row:range.endRow,end_column:range.endColumn}
  else delete next.merge
  return Object.keys(next).length?next:undefined
}

export function stripMergeStyle(style:Record<string,unknown>|undefined){
  if(!style||style.merge===undefined)return style
  const next={...style}
  delete next.merge
  return Object.keys(next).length?next:undefined
}

/**
 * 이 범위에 걸쳐 있는 병합 칸의 첫 자리를 찾는다. 없으면 undefined 다.
 *
 * 병합 정보는 덮인 칸마다 적혀 있으므로, 범위 밖에서 시작한 병합도 안쪽
 * 칸 하나만 보면 찾을 수 있다.
 *
 * 자료를 옮기는 정리는 병합을 만나면 그냥 두면 안 된다. 병합된 칸의 서식을
 * 다른 행으로 옮겨 적으면 서버가 통째로 물리친다 — "stored merge metadata is
 * invalid" 라는, 사람이 무엇을 해야 하는지 알 수 없는 말과 함께.
 */
export function mergeInRegion(cells:Map<string,Cell>,region:MergeRange){
  for(let row=region.startRow;row<=region.endRow;row+=1)
    for(let column=region.startColumn;column<=region.endColumn;column+=1){
      const range=cellMerge(cells.get(cellKey(row,column)))
      if(range)return {row,column,range}
    }
  return undefined
}
