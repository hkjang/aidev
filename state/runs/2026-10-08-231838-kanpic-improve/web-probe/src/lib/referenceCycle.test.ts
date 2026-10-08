import { describe, expect, it } from 'vitest'
import { cycleReference } from './referenceCycle'

const at=(text:string,caret:number)=>cycleReference(text,caret,caret)

describe('cycleReference',()=>{
  it('walks one reference through the four forms and back',()=>{
    let text='=A1'
    const steps:string[]=[]
    for(let round=0;round<5;round+=1){
      const result=cycleReference(text,text.length,text.length)!
      text=result.text
      steps.push(text)
    }
    expect(steps).toEqual(['=$A$1','=A$1','=$A1','=A1','=$A$1'])
  })

  it('picks the reference the caret is sitting in',()=>{
    // =SUM(A1:B2) 에서 B2 안에 캐럿이 있으면 B2 만 바뀐다.
    expect(at('=SUM(A1:B2)',9)?.text).toBe('=SUM(A1:$B$2)')
    expect(at('=SUM(A1:B2)',7)?.text).toBe('=SUM($A$1:B2)')
  })

  it('picks the reference just typed when the caret is past it',()=>{
    expect(at('=SUM(A1',7)?.text).toBe('=SUM($A$1')
  })

  it('moves every reference inside a selection together',()=>{
    const result=cycleReference('=SUM(A1:B2)',5,10)!
    expect(result.text).toBe('=SUM($A$1:$B$2)')
    // 고른 자리를 그대로 두어야 F4 를 다시 눌러 함께 이어 돌릴 수 있다.
    expect(result.text.slice(result.start,result.end)).toBe('$A$1:$B$2')
    const again=cycleReference(result.text,result.start,result.end)!
    expect(again.text).toBe('=SUM(A$1:B$2)')
  })

  it('follows the first reference when a selection mixes forms',()=>{
    expect(cycleReference('=$A$1+B2',0,8)?.text).toBe('=A$1+B$2')
  })

  it('leaves a sheet name alone and only anchors the cell',()=>{
    expect(at("='2분기'!C5",10)?.text).toBe("='2분기'!$C$5")
  })

  it('does not mistake part of a name for a reference',()=>{
    // LOG10 의 G10, A1B 의 A1 은 참조가 아니다.
    expect(at('=LOG10(2)',6)).toBeUndefined()
    expect(at('=A1B',4)).toBeUndefined()
    expect(at('=SUM(1,2)',5)).toBeUndefined()
    expect(at('그냥 글자',3)).toBeUndefined()
  })
})
