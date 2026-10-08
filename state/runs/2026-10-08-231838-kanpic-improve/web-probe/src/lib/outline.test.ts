import { describe, expect, it } from 'vitest'
import { collapsedIndexes, controlAt, controlIndexFor, groupsAt, innermostGroup, outlineSize } from './outline'
import type { DimensionGroup } from '../types'

const group=(start:number,end:number,collapsed=false,depth=0):DimensionGroup=>({start,end,collapsed,depth})

describe('collapsedIndexes', () => {
  it('folds away only the rows of collapsed groups', () => {
    const hidden=collapsedIndexes([group(3,6,true),group(10,12)])
    expect([...hidden].sort((a,b)=>a-b)).toEqual([3,4,5,6])
  })

  it('has nothing to hide without groups', () => {
    expect(collapsedIndexes(undefined).size).toBe(0)
  })
})

describe('outlineSize', () => {
  it('grows one step per level of nesting', () => {
    expect(outlineSize(undefined)).toBe(0)
    expect(outlineSize([group(2,5)])).toBe(15)
    expect(outlineSize([group(2,9),group(3,5,false,1)])).toBe(26)
  })
})

describe('controls', () => {
  it('puts the control one past the end of the group', () => {
    expect(controlIndexFor(group(3,6))).toBe(7)
    expect(controlAt([group(3,6)],7,0)).toBeDefined()
    expect(controlAt([group(3,6)],6,0)).toBeUndefined()
    // A nested control belongs to its own level only.
    expect(controlAt([group(3,9),group(4,6,false,1)],7,0)).toBeUndefined()
    expect(controlAt([group(3,9),group(4,6,false,1)],7,1)).toBeDefined()
  })

  it('reports every bracket passing through a row', () => {
    const groups=[group(2,9),group(4,6,false,1)]
    expect(groupsAt(groups,5)).toHaveLength(2)
    // Row 7 carries the inner group's control box and the outer bracket.
    expect(groupsAt(groups,7)).toHaveLength(2)
    expect(groupsAt(groups,10)).toHaveLength(1)
    expect(groupsAt(groups,11)).toHaveLength(0)
  })
})

describe('innermostGroup', () => {
  it('picks the smallest group that covers the selection', () => {
    const groups=[group(2,9),group(4,6,false,1)]
    expect(innermostGroup(groups,4,6)).toMatchObject({start:4,end:6})
    expect(innermostGroup(groups,3,8)).toMatchObject({start:2,end:9})
    expect(innermostGroup(groups,20,21)).toBeUndefined()
  })
})
