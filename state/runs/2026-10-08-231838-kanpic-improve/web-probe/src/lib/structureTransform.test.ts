import { describe, expect, it } from 'vitest'
import { survivesChange, transformPosition, transformSelection } from './structureTransform'

describe('transformPosition', () => {
  it('moves down when rows are inserted above', () => {
    expect(transformPosition(5,{axis:'row',action:'insert',index:3,count:2})).toBe(7)
    expect(transformPosition(2,{axis:'row',action:'insert',index:3,count:2})).toBe(2)
  })

  it('moves up when rows above are deleted', () => {
    expect(transformPosition(5,{axis:'row',action:'delete',index:3,count:1})).toBe(4)
    expect(transformPosition(2,{axis:'row',action:'delete',index:3,count:1})).toBe(2)
  })

  // 보고 있던 행이 사라지면 갈 곳이 없다. 그 자리를 물려받은 행에 머무르는
  // 편이 화면이 맨 위로 튀는 것보다 낫다.
  it('stays where the deleted row was', () => {
    expect(transformPosition(3,{axis:'row',action:'delete',index:3,count:2})).toBe(3)
    expect(transformPosition(4,{axis:'row',action:'delete',index:3,count:2})).toBe(3)
  })

  it('follows a moved band', () => {
    // 2행을 5행 자리로 옮기면 2행은 4행이 된다.
    expect(transformPosition(2,{axis:'row',action:'move',index:2,count:1,destination:5})).toBe(4)
    // 그 사이의 행들은 한 칸씩 올라온다.
    expect(transformPosition(3,{axis:'row',action:'move',index:2,count:1,destination:5})).toBe(2)
    // 범위 밖은 그대로다.
    expect(transformPosition(9,{axis:'row',action:'move',index:2,count:1,destination:5})).toBe(9)
  })
})

describe('transformSelection', () => {
  it('moves only the axis that changed', () => {
    const selection={activeRow:5,activeColumn:2,anchorRow:5,anchorColumn:2}
    expect(transformSelection(selection,{axis:'row',action:'delete',index:1,count:1})).toEqual({activeRow:4,activeColumn:2,anchorRow:4,anchorColumn:2})
    expect(transformSelection(selection,{axis:'column',action:'insert',index:1,count:3})).toEqual({activeRow:5,activeColumn:5,anchorRow:5,anchorColumn:5})
  })
})

describe('survivesChange', () => {
  it('says a cell outside the deleted band survives', () => {
    expect(survivesChange(5,1,{axis:'row',action:'delete',index:3,count:1})).toBe(true)
    expect(survivesChange(2,1,{axis:'row',action:'delete',index:3,count:1})).toBe(true)
  })

  // 사라진 셀에 입력을 되돌리면 그 자리를 물려받은 남의 값을 덮어쓴다.
  it('says a cell inside the deleted band is gone', () => {
    expect(survivesChange(3,1,{axis:'row',action:'delete',index:3,count:2})).toBe(false)
    expect(survivesChange(4,1,{axis:'row',action:'delete',index:3,count:2})).toBe(false)
    expect(survivesChange(1,2,{axis:'column',action:'delete',index:2,count:1})).toBe(false)
  })

  it('never loses a cell to an insert or a move', () => {
    expect(survivesChange(3,1,{axis:'row',action:'insert',index:3,count:2})).toBe(true)
    expect(survivesChange(3,1,{axis:'row',action:'move',index:3,count:1,destination:6})).toBe(true)
  })
})
