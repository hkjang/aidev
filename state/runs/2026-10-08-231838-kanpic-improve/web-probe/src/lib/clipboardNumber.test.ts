import { describe, expect, it } from 'vitest'
import { parsePastedNumber } from './clipboardNumber'

describe('parsePastedNumber',()=>{
  it('reads the number a spreadsheet was showing',()=>{
    expect(parsePastedNumber('1234')).toEqual({value:1234})
    expect(parsePastedNumber('-12.5')).toEqual({value:-12.5})
    expect(parsePastedNumber('1,234')).toEqual({value:1234,numberFormat:'#,##0'})
    expect(parsePastedNumber('₩1,234.50')).toEqual({value:1234.5,numberFormat:'"₩"#,##0.00'})
    expect(parsePastedNumber('98,000원')).toEqual({value:98000,numberFormat:'#,##0"원"'})
    expect(parsePastedNumber('12.5%')).toEqual({value:0.125,numberFormat:'0.0%'})
    // 회계 표기의 괄호는 음수를 뜻한다.
    // 회계 표기의 괄호는 음수를 적는 꼴이다. 떼어 버리면 (1,234) 가
    // -1,234 로 보이고, 사람은 자기가 옮긴 표와 다른 것을 보게 된다.
    expect(parsePastedNumber('(1,234)')).toEqual({value:-1234,numberFormat:'#,##0;(#,##0)'})
  })

  it('leaves alone anything a spreadsheet would keep as text',()=>{
    // 쉼표를 소수점으로 쓰는 표기를 자릿수 구분으로 읽으면 1,2가 12가 된다.
    expect(parsePastedNumber('1,2')).toBeUndefined()
    expect(parsePastedNumber('12,34')).toBeUndefined()
    expect(parsePastedNumber('010-1234-5678')).toBeUndefined()
    expect(parsePastedNumber('2026-08-23')).toBeUndefined()
    expect(parsePastedNumber('1,234 개')).toBeUndefined()
    expect(parsePastedNumber('제품')).toBeUndefined()
    expect(parsePastedNumber('')).toBeUndefined()
    expect(parsePastedNumber('1.2.3')).toBeUndefined()
  })
})
