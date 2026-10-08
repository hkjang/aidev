import { axisIndexAtViewport, axisViewportPosition, type DimensionAxis } from './dimensionAxis'

export type GridGeometry={
  rowAxis:DimensionAxis
  columnAxis:DimensionAxis
  scroll:{left:number;top:number}
  frozenRows:number
  frozenColumns:number
  headerWidth:number
  headerHeight:number
}
export type GridPointerRegion={kind:'cell';row:number;column:number}|{kind:'row';index:number}|{kind:'column';index:number}|{kind:'corner'}
export type ResizeTarget={axis:'row'|'column';index:number}

/** Maps a canvas coordinate to the header, corner or cell it belongs to. */
export function pointerRegion(geometry:GridGeometry,x:number,y:number):GridPointerRegion{
  const{rowAxis,columnAxis,scroll,frozenRows,frozenColumns,headerWidth,headerHeight}=geometry
  if(x<headerWidth&&y<headerHeight)return{kind:'corner'}
  if(y<headerHeight)return{kind:'column',index:axisIndexAtViewport(columnAxis,Math.max(0,x-headerWidth),scroll.left,frozenColumns)}
  if(x<headerWidth)return{kind:'row',index:axisIndexAtViewport(rowAxis,Math.max(0,y-headerHeight),scroll.top,frozenRows)}
  return{
    kind:'cell',
    row:axisIndexAtViewport(rowAxis,y-headerHeight,scroll.top,frozenRows),
    column:axisIndexAtViewport(columnAxis,x-headerWidth,scroll.left,frozenColumns),
  }
}

/**
 * Returns the row or column a header boundary near the pointer would resize.
 * Grabbing the leading edge targets the previous visible dimension, which is
 * how spreadsheets let you widen the column left of the cursor.
 */
export function resizeHandleAt(geometry:GridGeometry,x:number,y:number,tolerance:number):ResizeTarget|undefined{
  const{rowAxis,columnAxis,scroll,frozenRows,frozenColumns,headerWidth,headerHeight}=geometry
  if(y<headerHeight&&x>=headerWidth){
    const index=axisIndexAtViewport(columnAxis,x-headerWidth,scroll.left,frozenColumns)
    const left=headerWidth+axisViewportPosition(columnAxis,index,scroll.left,frozenColumns)
    if(x>=left+columnAxis.sizeOf(index)-tolerance)return{axis:'column',index}
    if(x<=left+tolerance){const previous=columnAxis.nextVisible(index,-1);if(previous<index)return{axis:'column',index:previous}}
    return undefined
  }
  if(x<headerWidth&&y>=headerHeight){
    const index=axisIndexAtViewport(rowAxis,y-headerHeight,scroll.top,frozenRows)
    const top=headerHeight+axisViewportPosition(rowAxis,index,scroll.top,frozenRows)
    if(y>=top+rowAxis.sizeOf(index)-tolerance)return{axis:'row',index}
    if(y<=top+tolerance){const previous=rowAxis.nextVisible(index,-1);if(previous<index)return{axis:'row',index:previous}}
  }
  return undefined
}

/** Clamps a dragged dimension size to the range the server accepts. */
export function clampDimensionSize(axis:'row'|'column',size:number){
  const minimum=axis==='column'?32:16,maximum=axis==='column'?600:400
  return Math.round(Math.max(minimum,Math.min(maximum,size)))
}

/**
 * A1:B10 을 좌표로 바꾼다. 표의 범위는 저장할 때 이 꼴로 다듬어 두므로 두
 * 칸을 이은 것만 읽는다. 읽지 못하면 그리지 않는다 — 어림잡아 그린 테두리는
 * 표가 아닌 곳을 표라고 말하는 것이라 없느니만 못하다.
 */
export function parseTableRange(value:string){
  const parts=value.split(':')
  if(parts.length!==2)return
  const position=(text:string)=>{
    const match=/^\$?([A-Za-z]+)\$?([1-9]\d*)$/.exec(text.trim())
    if(!match)return
    let column=0
    for(const letter of match[1].toUpperCase())column=column*26+letter.charCodeAt(0)-64
    const row=Number(match[2])
    if(column>16384||row>1048576)return
    return {row,column}
  }
  const start=position(parts[0]),end=position(parts[1])
  if(!start||!end||start.row>end.row||start.column>end.column)return
  return {startRow:start.row,startColumn:start.column,endRow:end.row,endColumn:end.column}
}
