import { describe, expect, it } from 'vitest'
import { applySuggestion, formulaContext, suggestFunctions, type FunctionDoc } from './formulaSuggest'

const catalog:FunctionDoc[]=[
  {name:'SUM',category:'수학',syntax:'SUM(값1, 값2, …)',summary:'합계'},
  {name:'SUMIF',category:'집계',syntax:'SUMIF(범위, 조건, [합계 범위])',summary:'조건 합계'},
  {name:'SUMPRODUCT',category:'집계',syntax:'SUMPRODUCT(범위1, 범위2, …)',summary:'곱의 합'},
  {name:'XLOOKUP',category:'조회',syntax:'XLOOKUP(찾을 값, 찾을 범위, 반환 범위)',summary:'조회'},
  {name:'DSUM',category:'기타',syntax:'DSUM(범위, 필드, 조건)',summary:'포함 검색 확인용'},
  // TEXTJOIN는 실제 목록에서 TEXT보다 앞에 있다. 이름을 다 친 사람이
  // Tab을 눌렀을 때 다른 함수가 들어가지 않는지 확인하기 위한 순서다.
  {name:'TEXTJOIN',category:'텍스트',syntax:'TEXTJOIN(구분자, 빈 값 무시, 값1, …)',summary:'이어 붙이기'},
  {name:'TEXT',category:'텍스트',syntax:'TEXT(값, 형식)',summary:'표시 형식'},
  {name:'TEXTSPLIT',category:'텍스트',syntax:'TEXTSPLIT(텍스트, 열 구분자)',summary:'나누기'},
]

describe('formulaContext', () => {
  it('ignores anything that is not a formula', () => {
    expect(formulaContext('합계',2)).toBeUndefined()
  })

  it('reads the partial name at the caret', () => {
    const context=formulaContext('=SUM',4)
    expect(context?.token).toBe('SUM')
    expect(context?.start).toBe(1)
  })

  it('does not suggest inside a text literal', () => {
    expect(formulaContext('="SUM',5)).toBeUndefined()
    // A closed string leaves the caret back in formula territory.
    expect(formulaContext('="SUM",SU',9)?.token).toBe('SU')
  })

  it('reports the call and argument the caret sits in', () => {
    expect(formulaContext('=SUM(A1:A3',10)?.call).toEqual({name:'SUM',argument:0})
    expect(formulaContext('=SUMIF(A1:A3,">2",',18)?.call).toEqual({name:'SUMIF',argument:2})
    // A finished inner call does not become the context.
    expect(formulaContext('=SUM(MAX(A1,A2),',16)?.call).toEqual({name:'SUM',argument:1})
    expect(formulaContext('=SUM(MAX(A1,',12)?.call).toEqual({name:'MAX',argument:1})
  })

  it('offers nothing to complete right after a bracket', () => {
    expect(formulaContext('=SUM(',5)?.token).toBe('')
  })
})

describe('suggestFunctions', () => {
  it('puts prefix matches before contained matches', () => {
    expect(suggestFunctions(catalog,'sum').map(item=>item.name)).toEqual(['SUM','SUMIF','SUMPRODUCT','DSUM'])
  })

  it('returns nothing without a partial name', () => {
    expect(suggestFunctions(catalog,'')).toEqual([])
  })

  it('offers the fully typed name first even when a longer one is listed earlier', () => {
    // 이름을 다 치고 Tab을 눌렀는데 TEXTJOIN이 들어가면, 사용자가 만든
    // 수식이 조용히 다른 함수가 된다.
    expect(suggestFunctions(catalog,'TEXT').map(item=>item.name)).toEqual(['TEXT','TEXTJOIN','TEXTSPLIT'])
    expect(suggestFunctions(catalog,'SUM').map(item=>item.name)[0]).toBe('SUM')
  })
})

describe('applySuggestion', () => {
  it('replaces the partial name and opens the bracket', () => {
    const context=formulaContext('=SU',3)!
    expect(applySuggestion('=SU',context,'SUMIF')).toEqual({text:'=SUMIF(',caret:7})
  })

  it('keeps an existing bracket and the rest of the formula', () => {
    const context=formulaContext('=SU(A1:A3)+1',3)!
    expect(applySuggestion('=SU(A1:A3)+1',context,'SUM')).toEqual({text:'=SUM(A1:A3)+1',caret:4})
  })
})
