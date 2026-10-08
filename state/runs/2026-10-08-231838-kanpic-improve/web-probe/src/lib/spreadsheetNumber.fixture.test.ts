import {describe,expect,it} from 'vitest'
import {readFileSync} from 'node:fs'
import {spreadsheetNumber} from './spreadsheetNumber'

// 글자로 담긴 값이 셈에 들어가는 규칙은 서버의 수식 엔진이 정하고 격자가 따른다.
// 두 곳이 다르면 =SUM 이 내는 값과 상태 줄에 보이는 합계가 어긋나고, 사람은
// 어느 쪽을 믿어야 할지 알 수 없다.
//
// testdata/numeric-text.json 을 internal/formula/numeric_text_test.go 와 함께
// 읽는다. 한쪽만 고치면 양쪽 다 걸린다.
type NumericTextCase={text:string;counts:boolean;value?:number}

describe('the grid counts what the engine counts',()=>{
  it('agrees on every case in testdata/numeric-text.json',()=>{
    const fixture=JSON.parse(readFileSync('../testdata/numeric-text.json','utf8')) as {cases:NumericTextCase[]}
    expect(fixture.cases.length).toBeGreaterThan(40)
    const wrong:string[]=[]
    for(const item of fixture.cases){
      const counted=spreadsheetNumber(item.text)
      if(item.counts){
        if(counted!==item.value)wrong.push(`${JSON.stringify(item.text)} -> ${counted}, 엔진은 ${item.value}`)
      }else if(counted!==undefined)wrong.push(`${JSON.stringify(item.text)} -> ${counted}, 엔진은 세지 않는다`)
    }
    expect(wrong.join('\n')).toBe('')
  })
})
