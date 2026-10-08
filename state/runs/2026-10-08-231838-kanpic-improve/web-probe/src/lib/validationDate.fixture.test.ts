import { describe,expect,it } from 'vitest'
import { readFileSync } from 'node:fs'
import { parseValidationDate } from './validationDate'

// testdata/validation-dates.json 을 서버의
// internal/workbook/validation_date_test.go 와 함께 읽는다. 한쪽만 고치면
// 양쪽 다 걸린다.
//
// 목록에는 브라우저가 기꺼이 읽어 주지만 서버가 물리치는 것들이 들어 있다.
// 그런 값을 화면이 "괜찮다" 하고 적어 넣으면 서버가 묶음을 물리쳐, 사람은
// 값이 나타났다 사라지는 것을 보게 된다.
describe('the grid reads dates the way the server does',()=>{
  it('agrees on every text in testdata/validation-dates.json',()=>{
    const fixture=JSON.parse(readFileSync('../testdata/validation-dates.json','utf8')) as {cases:Array<{text:string;iso:string|null}>}
    expect(fixture.cases.length).toBeGreaterThan(20)
    for(const item of fixture.cases){
      const parsed=parseValidationDate(item.text)
      if(item.iso===null){
        expect(parsed,`${JSON.stringify(item.text)} 는 날짜가 아니어야 한다`).toBeUndefined()
        continue
      }
      expect(parsed,`${JSON.stringify(item.text)}`).toBeDefined()
      expect(new Date(parsed!).toISOString().replace('.000',''),`${JSON.stringify(item.text)}`).toBe(item.iso)
    }
  })
})
