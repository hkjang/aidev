import { describe, expect, it } from 'vitest'
import { explainFormulaError, formulaErrorCode, formulaErrorCodes } from './formulaError'

describe('formulaErrorCode', () => {
  it('recognises an error code held in a cell', () => {
    expect(formulaErrorCode('#VALUE!')).toBe('#VALUE!')
    expect(formulaErrorCode(' #n/a ')).toBe('#N/A')
  })

  it('leaves ordinary values alone', () => {
    expect(formulaErrorCode('합계')).toBeUndefined()
    expect(formulaErrorCode(12)).toBeUndefined()
    expect(formulaErrorCode(undefined)).toBeUndefined()
    // 오류처럼 보이지만 엔진이 내지 않는 코드는 설명할 것이 없다.
    expect(formulaErrorCode('#WHAT?')).toBeUndefined()
  })
})

describe('explainFormulaError', () => {
  it('says what happened and what to do next', () => {
    const explanation=explainFormulaError('#REF!')
    expect(explanation?.summary).toContain('사라졌습니다')
    expect(explanation?.next).toContain('실행 취소')
  })

  // 엔진이 구체적인 이유를 준 경우에는 총론보다 그것이 낫다.
  it('prefers the engine reason when there is one', () => {
    expect(explainFormulaError('#SPILL!','아래 3칸에 이미 값이 있습니다')?.summary).toBe('아래 3칸에 이미 값이 있습니다')
    // 그래도 다음에 할 일은 계속 안내한다.
    expect(explainFormulaError('#SPILL!','아래 3칸에 이미 값이 있습니다')?.next).toContain('비우세요')
  })

  it('explains every code the engine can produce', () => {
    for(const code of ['#CIRC!','#DIV/0!','#ERROR!','#NAME?','#NULL!','#NUM!','#REF!','#SPILL!','#VALUE!','#N/A'])
      expect(explainFormulaError(code),code).toBeDefined()
    expect(formulaErrorCodes).toHaveLength(10)
  })
})
