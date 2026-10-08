import { describe, expect, it } from 'vitest'
import { formatStat, shouldSummarize, summarizeSelection } from './selectionSummary'
import type { Cell } from '../types'

const grid=(values:Array<[number,number,unknown]>)=>{
  const cells=new Map<string,Cell>()
  for(const [row,column,value] of values)cells.set(`${row}:${column}`,{sheet_id:'s',row,column,value,updated_at:''} as Cell)
  return cells
}

describe('summarizeSelection', () => {
  it('adds the numbers and counts everything that is filled', () => {
    const stats=summarizeSelection(grid([[1,1,10],[2,1,20],[3,1,'미정'],[4,1,''],[5,1,null]]),1,1,5,1)
    expect(stats).toMatchObject({numbers:2,filled:3,sum:30,average:15,min:10,max:20})
  })

  it('reads numbers that arrived as text, as an imported sheet has', () => {
    expect(summarizeSelection(grid([[1,1,'1500'],[2,1,'2500']]),1,1,2,1).sum).toBe(4000)
  })

  it('reports zero for a selection with no numbers instead of infinity', () => {
    const stats=summarizeSelection(grid([[1,1,'가'],[2,1,'나']]),1,1,2,1)
    expect(stats).toMatchObject({numbers:0,filled:2,sum:0,average:0,min:0,max:0})
  })

  it('only covers the selected rectangle', () => {
    expect(summarizeSelection(grid([[1,1,10],[1,2,99],[2,1,20]]),1,1,2,1).sum).toBe(30)
  })
})

describe('formatStat', () => {
  it('keeps money readable and small averages precise', () => {
    const stats=summarizeSelection(grid([[1,1,1234567],[2,1,2]]),1,1,2,1)
    expect(formatStat('sum',stats)).toBe('1,234,569')
    expect(formatStat('average',stats)).toBe('617,284.5')
    expect(formatStat('counta',stats)).toBe('2')
  })
})

describe('shouldSummarize', () => {
  it('stays quiet for a single cell and for an empty selection', () => {
    const stats=summarizeSelection(grid([[1,1,10]]),1,1,1,1)
    expect(shouldSummarize(stats,1)).toBe(false)
    expect(shouldSummarize(summarizeSelection(new Map(),1,1,3,3),9)).toBe(false)
    expect(shouldSummarize(stats,4)).toBe(true)
  })
})
