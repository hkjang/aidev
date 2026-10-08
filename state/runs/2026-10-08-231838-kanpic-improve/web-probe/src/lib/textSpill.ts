export type SpillRoom={left:number;right:number}

/**
 * Extra width a cell may borrow from its empty neighbours. Spreadsheets let
 * long text run over the cells next to it and cut it off at the first cell that
 * holds something, which reads far better than squeezing the glyphs to fit.
 */
export function spillRoom(options:{
  row:number
  column:number
  alignment:'left'|'center'|'right'
  populated:(row:number,column:number)=>boolean
  sizeOf:(column:number)=>number
  maxColumn:number
  limit?:number
}):SpillRoom{
  const limit=options.limit??900
  const room={left:0,right:0}
  if(options.alignment!=='right'){
    for(let column=options.column+1;column<=options.maxColumn&&room.right<limit;column+=1){
      if(options.populated(options.row,column))break
      room.right+=options.sizeOf(column)
    }
  }
  if(options.alignment!=='left'){
    for(let column=options.column-1;column>=1&&room.left<limit;column-=1){
      if(options.populated(options.row,column))break
      room.left+=options.sizeOf(column)
    }
  }
  // The limit only bounds the scan; the room already found is real space.
  return room
}
