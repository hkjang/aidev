import { describe,expect,it } from 'vitest'
import { parseTypedCellValue } from './cellEntry'
import { spreadsheetNumber } from './spreadsheetNumber'

/**
 * 칸에 직접 쳐 넣는 문의 계약. 붙여넣기·파일 가져오기와 겹치는 곳은 같아야
 * 하고, 일부러 다른 곳은 그것이 일부러임을 여기 적어 둔다.
 */
describe('a value typed into a cell',()=>{
  it('keeps what a spreadsheet counts as a number',()=>{
    for(const [raw,value] of [['12',12],['-12',-12],['+12',12],['0',0],['0.5',.5],['.5',.5],['5.',5],
      ['1e3',1000],['1E3',1000],['2.5e-3',.0025],[' 12 ',12]] as const){
      expect(parseTypedCellValue(raw),raw).toEqual({value})
    }
  })

  it('reads the shape a number was written in and keeps it as a format',()=>{
    // 붙여넣기와 같은 자(decomposeNumberText)를 지나므로 값과 서식이 같다.
    expect(parseTypedCellValue('1,234')).toEqual({value:1234,numberFormat:'#,##0'})
    expect(parseTypedCellValue('50%')).toEqual({value:.5,numberFormat:'0%'})
    expect(parseTypedCellValue('₩5,000')).toEqual({value:5000,numberFormat:'"₩"#,##0'})
  })

  it('leaves a hexadecimal, binary or octal literal as the text it is',()=>{
    // Number() 는 스프레드시트보다 넓어 `0x1F` 를 31 로, `0b101` 을 5 로,
    // `0o17` 을 15 로 읽는다. 어느 스프레드시트도 그렇게 읽지 않고, 숫자로
    // 바꾸는 순간 사람이 친 글자는 사라져 되돌릴 길이 없다. 같은 글자가
    // 파일이나 붙여넣기로 들어오면 글자로 남는데 손으로 치면 수가 되는 것은
    // 한 표 안에서 같은 값이 두 가지로 저장된다는 뜻이다.
    for(const raw of ['0x10','0X1F','0xdeadbeef','0b101','0B11','0o17','0O7']){
      expect(parseTypedCellValue(raw),raw).toEqual({value:raw})
    }
  })

  it('leaves alone what is not a number at all',()=>{
    for(const raw of ['abc','1,2','--5','1 234','１２３','5%원','1e400','2026-01-01',' ']){
      expect(parseTypedCellValue(raw),raw).toEqual({value:raw})
    }
  })

  it('reads true and false as booleans and an empty box as empty',()=>{
    expect(parseTypedCellValue('true')).toEqual({value:true})
    expect(parseTypedCellValue('TRUE')).toEqual({value:true})
    expect(parseTypedCellValue('False')).toEqual({value:false})
    expect(parseTypedCellValue('')).toEqual({value:undefined})
  })

  it('still reads a typed leading zero and a too-long number as a number',()=>{
    // 파일·붙여넣기와 일부러 다른 곳이다. 파일에서 온 `00123` 은 우편번호일
    // 수 있어 글자로 두지만, 사람이 칸에 직접 친 것은 엑셀·시트가 그러듯
    // 숫자로 받는다 — 되돌릴 길은 사람이 다시 치는 것이라 파일과 다르다.
    expect(parseTypedCellValue('00123')).toEqual({value:123})
    expect(parseTypedCellValue('007')).toEqual({value:7})
    expect(parseTypedCellValue('12345678901234567890')).toEqual({value:12345678901234567890})
  })

  it('never stores a plain number the sum would count differently',()=>{
    // 서식 없이 숫자로 담기는 것은 셈하는 자가 같은 값으로 읽는 것뿐이어야
    // 한다. 그렇지 않으면 상태 줄 합계와 =SUM 이 갈린다.
    for(const raw of ['12','-12','0.5','.5','5.','1e3',' 12 ','00123','12345678901234567890',
      '0x10','0b101','abc','1,234','50%','₩5,000','','true']){
      const parsed=parseTypedCellValue(raw)
      if(typeof parsed.value!=='number'||parsed.numberFormat!==undefined)continue
      expect(spreadsheetNumber(raw),raw).toBe(parsed.value)
    }
  })
})
