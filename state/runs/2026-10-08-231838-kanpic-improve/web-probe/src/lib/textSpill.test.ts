import { describe, expect, it } from 'vitest'
import { spillRoom } from './textSpill'

const options=(alignment:'left'|'center'|'right',filled:Array<[number,number]>)=>({
  row:2,column:3,alignment,maxColumn:10,sizeOf:()=>100,
  populated:(row:number,column:number)=>filled.some(([r,c])=>r===row&&c===column),
})

describe('spillRoom',()=>{
  it('borrows every empty column to the right of left aligned text',()=>{
    expect(spillRoom({...options('left',[[2,7]]),limit:900})).toEqual({left:0,right:300})
  })

  it('stops at the first neighbour that holds something',()=>{
    expect(spillRoom(options('left',[[2,4]]))).toEqual({left:0,right:0})
  })

  it('borrows to the left for right aligned text',()=>{
    expect(spillRoom(options('right',[[2,1]]))).toEqual({left:100,right:0})
  })

  it('borrows both directions for centred text',()=>{
    expect(spillRoom(options('center',[[2,1],[2,6]]))).toEqual({left:100,right:200})
  })

  it('never borrows more than the limit',()=>{
    expect(spillRoom({...options('left',[]),limit:250})).toEqual({left:0,right:300})
  })

  it('ignores cells in other rows',()=>{
    expect(spillRoom(options('left',[[3,4]]))).toEqual({left:0,right:700})
  })
})
