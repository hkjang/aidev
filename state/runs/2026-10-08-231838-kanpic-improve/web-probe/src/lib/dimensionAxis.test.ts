import { describe,expect,it } from 'vitest'
import { axisIndexAtViewport,axisViewportPosition,createDimensionAxis,rowHeaderWidth,presenceLabelTop} from './dimensionAxis'

describe('dimension axis',()=>{
  it('combines sparse sizes and hidden ranges without materializing every row',()=>{
    const axis=createDimensionAxis({total:10,defaultSize:20,sizes:[{index:2,size:40},{index:7,size:30}],hiddenRanges:[{start:3,end:5}],hiddenIndexes:[8]})
    expect(axis.sizeOf(2)).toBe(40)
    expect(axis.sizeOf(4)).toBe(0)
    expect(axis.offsetOf(6)).toBe(60)
    expect(axis.extent).toBe(150)
    expect(axis.indexAtOffset(61)).toBe(6)
    expect(axis.countVisible(1,10)).toBe(6)
    expect(axis.rangeSize(2,6)).toBe(60)
  })
  it('maps frozen and scrolling dimensions to viewport coordinates',()=>{
    const axis=createDimensionAxis({total:100,defaultSize:10})
    expect(axisViewportPosition(axis,2,50,2)).toBe(10)
    expect(axisViewportPosition(axis,8,50,2)).toBe(20)
    expect(axisIndexAtViewport(axis,15,50,2)).toBe(2)
    expect(axisIndexAtViewport(axis,25,50,2)).toBe(8)
  })
})

describe('rowHeaderWidth', () => {
  // 작은 시트에서 자리를 낭비하지 않는다.
  it('keeps the base width while row numbers are short', () => {
    expect(rowHeaderWidth(46,1)).toBe(46)
    expect(rowHeaderWidth(46,9_999)).toBe(46)
  })

  // 15만 행짜리 시트를 내려가면 번호의 앞자리가 잘려 나갔다.
  it('grows a step for every extra digit', () => {
    expect(rowHeaderWidth(46,10_000)).toBe(54)
    expect(rowHeaderWidth(46,150_001)).toBe(62)
  })

  it('scales the step with the zoom', () => {
    expect(rowHeaderWidth(46,150_001,2)).toBe(78)
  })
})

describe('presenceLabelTop', () => {
  it('sits above the cell when there is room', () => {
    expect(presenceLabelTop(200,27,14,27)).toBe(185)
  })

  // 맨 윗줄에서 머리글 아래로 밀어 넣으면 그 셀의 내용을 가린다.
  it('drops below the cell when the header is in the way', () => {
    expect(presenceLabelTop(27,27,14,27)).toBe(55)
    expect(presenceLabelTop(35,27,14,27)).toBe(63)
  })
})
