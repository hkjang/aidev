import { describe,expect,it } from 'vitest'
import { createRowVisibility } from './rowVisibility'

describe('filtered row visibility mapping',()=>{
  it('compresses hidden rows and maps display positions back to source rows',()=>{
    const rows=createRowVisibility([3,4,7],10)
    expect(rows.hidden).toEqual([3,4,7])
    expect(rows.visibleCount).toBe(7)
    expect([1,2,3,4,5,6,7].map(rows.displayToActual)).toEqual([1,2,5,6,8,9,10])
    expect(rows.actualToDisplay(8)).toBe(5)
    expect(rows.countVisible(2,8)).toBe(4)
  })

  it('skips consecutive hidden rows during navigation',()=>{
    const rows=createRowVisibility([2,3,9],10)
    expect(rows.nextVisible(1,1)).toBe(4)
    expect(rows.nextVisible(4,-1)).toBe(1)
    expect(rows.firstVisibleAtOrAfter(2)).toBe(4)
    expect(rows.lastVisibleAtOrBefore(9)).toBe(8)
  })

  it('normalizes duplicate and invalid hidden rows',()=>{
    const rows=createRowVisibility([0,2,2,11,Number.NaN],10)
    expect(rows.hidden).toEqual([2])
    expect(rows.displayToActual(2)).toBe(3)
  })
})
