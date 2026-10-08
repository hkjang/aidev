import { cellKey } from '../state/editor'
import type { Cell } from '../types'

export type GridRegion={startRow:number;startColumn:number;endRow:number;endColumn:number}

export function populatedCell(cells:Map<string,Cell>,row:number,column:number){
  const cell=cells.get(cellKey(row,column))
  return cell?.value!=null||Boolean(cell?.formula)
}

function rowHasData(cells:Map<string,Cell>,row:number,startColumn:number,endColumn:number){
  for(let column=startColumn;column<=endColumn;column+=1)if(populatedCell(cells,row,column))return true
  return false
}

function columnHasData(cells:Map<string,Cell>,column:number,startRow:number,endRow:number){
  for(let row=startRow;row<=endRow;row+=1)if(populatedCell(cells,row,column))return true
  return false
}

/**
 * Grows the rectangle around a seed cell until it is surrounded by empty rows
 * and columns, mirroring the "current region" a spreadsheet sorts or filters by
 * default. Only cells already loaded into the client store are considered, the
 * same data the client-side sort works on.
 */
export function dataRegion(cells:Map<string,Cell>,seedRow:number,seedColumn:number,bounds:{rows:number;columns:number}):GridRegion{
  const region:GridRegion={startRow:seedRow,startColumn:seedColumn,endRow:seedRow,endColumn:seedColumn}
  for(let guard=0;guard<bounds.rows+bounds.columns;guard+=1){
    let grew=false
    while(region.startRow>1&&rowHasData(cells,region.startRow-1,region.startColumn,region.endColumn)){region.startRow-=1;grew=true}
    while(region.endRow<bounds.rows&&rowHasData(cells,region.endRow+1,region.startColumn,region.endColumn)){region.endRow+=1;grew=true}
    while(region.startColumn>1&&columnHasData(cells,region.startColumn-1,region.startRow,region.endRow)){region.startColumn-=1;grew=true}
    while(region.endColumn<bounds.columns&&columnHasData(cells,region.endColumn+1,region.startRow,region.endRow)){region.endColumn+=1;grew=true}
    if(!grew)break
  }
  return region
}

/** True when the region's first row looks like a header instead of data. */
export function looksLikeHeaderRow(cells:Map<string,Cell>,region:GridRegion){
  if(region.endRow<=region.startRow)return false
  let headerText=0,bodyNumbers=0
  for(let column=region.startColumn;column<=region.endColumn;column+=1){
    const header=cells.get(cellKey(region.startRow,column)),body=cells.get(cellKey(region.startRow+1,column))
    if(typeof header?.value==='string'&&header.value.trim()!=='')headerText+=1
    if(typeof body?.value==='number')bodyNumbers+=1
  }
  return headerText>0&&bodyNumbers>0
}
