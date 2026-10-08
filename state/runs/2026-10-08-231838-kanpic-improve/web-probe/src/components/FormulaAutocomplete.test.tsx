import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { FormulaAutocomplete, formulaHint } from './FormulaAutocomplete'
import type { FunctionDoc } from '../lib/formulaSuggest'

const catalog:FunctionDoc[]=[
  {name:'SUM',category:'수학',syntax:'SUM(값1, 값2, …)',summary:'합계'},
  {name:'TAKE',category:'배열',syntax:'TAKE(배열, 행 수, [열 수])',summary:'앞뒤에서 남기기'},
]

const hintFor=(text:string)=>{
  const hint=formulaHint(catalog,text,text.length)
  if(!hint)throw new Error(`no hint for ${text}`)
  return hint
}

describe('FormulaAutocomplete', () => {
  it('marks the argument the caret is on', () => {
    render(<FormulaAutocomplete hint={hintFor('=TAKE(A1:B3,1,')} active={0} left={0} top={0} onChoose={()=>{}}/>)
    expect(screen.getByText('[열 수]')).toHaveClass('current')
  })

  // 값을 계속 받는 함수는 이름 붙은 인수보다 많이 받는다. 네 번째 값에서
  // 힌트가 꺼지면, 정작 가장 필요할 때 안내가 사라진다.
  it('keeps marking the repeating argument past the named ones', () => {
    render(<FormulaAutocomplete hint={hintFor('=SUM(1,2,3,4,')} active={0} left={0} top={0} onChoose={()=>{}}/>)
    expect(screen.getByText('…')).toHaveClass('current')
  })
})
