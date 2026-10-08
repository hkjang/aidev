import { describe, expect, it, vi } from 'vitest'
import { drawSparkline, parseSparkline } from './sparkline'

describe('parseSparkline', () => {
  it('recognises the value a SPARKLINE formula produces', () => {
    expect(parseSparkline({kanpic:'sparkline',chart:'column',values:[1,2,3],color:'#123456'})).toMatchObject({
      chart:'column',values:[1,2,3],color:'#123456',
    })
  })

  it('leaves ordinary cell values alone', () => {
    for(const value of [null,undefined,42,'문자',[1,2],{kanpic:'other',values:[1]},{kanpic:'sparkline',values:[]}]){
      expect(parseSparkline(value)).toBeUndefined()
    }
  })

  it('drops values that are not finite numbers and falls back to a line', () => {
    expect(parseSparkline({kanpic:'sparkline',values:[1,'x',Number.NaN,4]})).toMatchObject({chart:'line',values:[1,4]})
  })
})

describe('drawSparkline', () => {
  const context=()=>({
    save:vi.fn(),restore:vi.fn(),beginPath:vi.fn(),rect:vi.fn(),clip:vi.fn(),moveTo:vi.fn(),lineTo:vi.fn(),
    stroke:vi.fn(),fillRect:vi.fn(),strokeStyle:'',fillStyle:'',lineWidth:0,lineJoin:'',
  }) as unknown as CanvasRenderingContext2D&{moveTo:ReturnType<typeof vi.fn>;lineTo:ReturnType<typeof vi.fn>;fillRect:ReturnType<typeof vi.fn>}

  it('draws one line segment per value', () => {
    const canvas=context()
    drawSparkline(canvas,{chart:'line',values:[1,5,3],color:'#0f766e'},0,0,60,20)
    expect(canvas.moveTo).toHaveBeenCalledTimes(1)
    expect(canvas.lineTo).toHaveBeenCalledTimes(2)
  })

  it('draws one bar per value for a column chart', () => {
    const canvas=context()
    drawSparkline(canvas,{chart:'column',values:[1,-2,3,4],color:'#0f766e'},0,0,60,20)
    expect(canvas.fillRect).toHaveBeenCalledTimes(4)
  })

  it('survives a flat series without dividing by zero', () => {
    const canvas=context()
    drawSparkline(canvas,{chart:'line',values:[5,5,5],color:'#0f766e'},0,0,60,20)
    const coordinates=canvas.lineTo.mock.calls.flat()
    expect(coordinates.every(value=>Number.isFinite(value))).toBe(true)
  })
})
