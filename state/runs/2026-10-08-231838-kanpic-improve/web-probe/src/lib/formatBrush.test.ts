import { describe, expect, it } from 'vitest'
import { brushIsEmpty, brushPatch, BRUSH_STYLE_KEYS } from './formatBrush'

describe('brushPatch', () => {
  it('carries the keys the source sets', () => {
    const patch=brushPatch({bold:true,background:'#fee2e2',number_format:'#,##0'})
    expect(patch).toMatchObject({bold:true,background:'#fee2e2',number_format:'#,##0'})
  })

  it('clears the keys the source does not set, so the target matches it', () => {
    const patch=brushPatch({bold:true})
    expect(patch.italic).toBeNull()
    expect(patch.background).toBeNull()
    expect(Object.keys(patch).sort()).toEqual([...BRUSH_STYLE_KEYS].sort())
  })

  it('never carries merge metadata', () => {
    expect(brushPatch({bold:true,merge:{start_row:1}} as Record<string,unknown>)).not.toHaveProperty('merge')
  })

  it('treats a cell with no style as an empty brush', () => {
    expect(brushIsEmpty(undefined)).toBe(true)
    expect(brushIsEmpty({})).toBe(true)
    expect(brushIsEmpty({bold:false})).toBe(false)
  })
})
