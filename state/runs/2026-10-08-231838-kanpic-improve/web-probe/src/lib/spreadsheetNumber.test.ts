import { describe, expect, it } from 'vitest'
import { looksLikeNumberStoredAsText, numberFormatForText, spreadsheetNumber, textToNumber } from './spreadsheetNumber'
import { formatCellValue } from './cellFormat'

// 셈하는 규칙이 세 곳에 따로 적혀 있었다. 선택 요약과 열 통계와 수식 엔진이
// 셋 다 달라, 같은 열에서 세 가지 합계가 나올 수 있었다. 이제 한 곳에서
// 나오되, 그 한 곳은 엔진과 같아야 한다.
describe('spreadsheetNumber',()=>{
  it('엔진과 같은 것만 센다',()=>{
    expect(spreadsheetNumber(1234)).toBe(1234)
    expect(spreadsheetNumber('2000')).toBe(2000)
    expect(spreadsheetNumber(' 300 ')).toBe(300)
    expect(spreadsheetNumber(true)).toBe(1)
    expect(spreadsheetNumber(false)).toBe(0)
  })
  it('엔진이 글자로 보는 것은 세지 않는다',()=>{
    // 쉼표가 붙은 것을 세면 화면에는 더해지는데 수식에서는 빠져, 두 숫자가
    // 어긋난 채로 남는다.
    for(const value of ['1,234','50%','1234원','(500)','','   ',null,undefined,{}]){
      expect(spreadsheetNumber(value)).toBeUndefined()
    }
  })
})

describe('textToNumber',()=>{
  it('사람이 적는 꼴을 숫자로 바꾼다',()=>{
    expect(textToNumber('1,234')).toBe(1234)
    expect(textToNumber('1,234,567')).toBe(1234567)
    expect(textToNumber('₩1,234')).toBe(1234)
    expect(textToNumber('1,234원')).toBe(1234)
    expect(textToNumber('-1,234')).toBe(-1234)
    // 괄호는 회계에서 음수를 적는 꼴이다.
    expect(textToNumber('(500)')).toBe(-500)
    expect(textToNumber('(1,500)')).toBe(-1500)
    // 백분율은 100 으로 나눈다. 화면에 보이던 것과 셈하는 값이 같아야 한다.
    expect(textToNumber('50%')).toBe(0.5)
    expect(textToNumber('1,234.56')).toBe(1234.56)
  })
  it('숫자가 아닌 것을 억지로 만들지 않는다',()=>{
    // 쉼표가 아무 데나 찍힌 것은 숫자가 아니다. 억지로 바꾸면 사람이 적은
    // 것과 다른 값이 칸에 들어간다.
    for(const value of ['1,2,3','12,34','아무개','1234abc','','--5','1.2.3']){
      expect(textToNumber(value)).toBeUndefined()
    }
  })
})

describe('looksLikeNumberStoredAsText',()=>{
  it('이미 셈에 들어가는 것은 고칠 것이 없다',()=>{
    for(const value of ['2000',' 300 ',1234,true,null]){
      expect(looksLikeNumberStoredAsText(value)).toBe(false)
    }
  })
  it('사람 눈에는 숫자인데 수식이 빼고 세는 것을 찾는다',()=>{
    for(const value of ['1,234','₩1,234','1,234원','(500)','50%']){
      expect(looksLikeNumberStoredAsText(value)).toBe(true)
    }
  })
  it('숫자가 아닌 글자는 건드리지 않는다',()=>{
    for(const value of ['아무개','','1,2,3']){
      expect(looksLikeNumberStoredAsText(value)).toBe(false)
    }
  })
})

describe('numberFormatForText',()=>{
  // 서식만 따로 보는 것으로는 부족하다. 고른 서식이 실제로 그리는 글자가
  // 원래 보이던 것과 같은지가 지켜야 할 것이다.
  const shown=(text:string)=>formatCellValue(textToNumber(text),{number_format:numberFormatForText(text)})
  it('keeps what the person saw',()=>{
    expect(shown('₩5,000')).toBe('₩5,000')
    expect(shown('50%')).toBe('50%')
    expect(shown('12.5%')).toBe('12.5%')
    expect(shown('1,234')).toBe('1,234')
    expect(shown('(500)')).toBe('(500)')
    expect(shown('$1,234.50')).toBe('$1,234.50')
  })
  it('leaves plain digits without a format',()=>{
    // 꾸밈이 없으면 굳힐 것도 없다. General 이 사람이 적은 자리를 그대로 낸다.
    expect(numberFormatForText('1234')).toBeUndefined()
  })
  it('keeps a currency symbol and a trailing unit',()=>{
    // 서식 말이 따옴표로 묶은 글자를 그리게 된 뒤로는 놓아줄 이유가 없다.
    expect(shown('£5,000')).toBe('£5,000')
    expect(shown('1,234원')).toBe('1,234원')
    // 기호는 따옴표로 묶는다. 붙여넣기가 내는 서식과 같은 글자여야, 같은
    // 값이 어떻게 들어왔는지에 따라 다르게 저장되지 않는다.
    expect(numberFormatForText('£5,000')).toBe('"£"#,##0')
  })
  it('has no format for text that is not a number',()=>{
    expect(numberFormatForText('1,2,3')).toBeUndefined()
    expect(numberFormatForText('사과')).toBeUndefined()
  })
})

describe('numbers too long to hold exactly',()=>{
  it('refuses to convert what a double cannot carry',()=>{
    // 스무 자리는 계좌번호이지 금액이 아니다. 숫자로 바꾸면 뒤가 뭉개져
    // 사람이 적은 것과 다른 값이 칸에 들어간다.
    expect(textToNumber('₩12345678901234567890')).toBeUndefined()
    expect(looksLikeNumberStoredAsText('₩12345678901234567890')).toBe(false)
  })
  it('still converts amounts of an ordinary size',()=>{
    expect(textToNumber('₩123,456,789,012')).toBe(123456789012)
  })
})
