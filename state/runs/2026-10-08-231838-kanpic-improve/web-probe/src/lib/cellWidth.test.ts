import { describe, expect, it } from 'vitest'
import { hashesWhenTooNarrow } from './cellWidth'

// 글자 하나를 6px 로 세는 자. 실제 캔버스 대신 쓴다.
const measure=(value:string)=>value.length*6

describe('hashesWhenTooNarrow',()=>{
  it('leaves a number that fits exactly as it is',()=>{
    expect(hashesWhenTooNarrow('42',60,measure)).toBe('42')
    expect(hashesWhenTooNarrow('123456',36,measure)).toBe('123456')
  })

  it('replaces a number that does not fit so it cannot be misread',()=>{
    // 1,234,567 의 앞부분만 보이면 1,234 로 읽힌다.
    expect(hashesWhenTooNarrow('1,234,567',36,measure)).toBe('######')
    expect(hashesWhenTooNarrow('123456789012345',36,measure)).toBe('######')
  })

  it('shows at least one hash rather than nothing',()=>{
    expect(hashesWhenTooNarrow('123456',3,measure)).toBe('#')
  })

  it('leaves an empty cell and a zero-width column alone',()=>{
    expect(hashesWhenTooNarrow('',10,measure)).toBe('')
    expect(hashesWhenTooNarrow('123',0,measure)).toBe('123')
    expect(hashesWhenTooNarrow('123',-5,measure)).toBe('123')
  })

  it('gives up rather than looping when the ruler measures nothing',()=>{
    expect(hashesWhenTooNarrow('123',10,value=>value==='#'?0:99)).toBe('123')
  })
})
