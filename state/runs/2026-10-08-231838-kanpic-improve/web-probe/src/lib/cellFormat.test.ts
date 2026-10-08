import {describe,expect,it} from 'vitest'
import {formatCellValue,wrapText} from './cellFormat'

describe('cell display formats',()=>{
  it('formats grouped numbers, percentages, currencies and scientific values',()=>{
    expect(formatCellValue(1234.5,{number_format:'#,##0.00'},'en-US')).toBe('1,234.50')
    expect(formatCellValue(.125,{number_format:'0.0%'},'en-US')).toBe('12.5%')
    expect(formatCellValue(2500,{number_format:'₩#,##0'},'en-US')).toBe('₩2,500')
    // 서식의 E+00 은 지수를 두 자리로 적으라는 뜻이다. 예전에는 자리를
    // 채우지 않아 1.23E+3 이었고, 이 시험이 그것을 고정하고 있었다.
    expect(formatCellValue(1234,{number_format:'0.00E+00'},'en-US')).toBe('1.23E+03')
    expect(formatCellValue(42,{number_format:'00000'},'en-US')).toBe('00042')
  })
  it('formats Excel serial dates and times without timezone drift',()=>{
    expect(formatCellValue(45292,{number_format:'yyyy-mm-dd'},'en-CA')).toBe('2024-01-01')
    expect(formatCellValue(.5,{number_format:'hh:mm'},'en-GB')).toBe('12:00')
    expect(formatCellValue(.75,{number_format:'h:mm AM/PM'},'en-US')).toBe('6:00 PM')
    expect(formatCellValue(1234,{number_format:'#,##0 ;[red](#,##0)'},'en-US')).toBe('1,234')
  })
  it('wraps words and long unbroken text',()=>{
    expect(wrapText('one two three',7,text=>text.length)).toEqual(['one two','three'])
    expect(wrapText('abcdefgh',3,text=>text.length)).toEqual(['abc','def','gh'])
  })
})

// 격자에 보이는 자릿수는 사람이 적은 십진수를 따라야 한다. 1.005 는 이진
// 실수로 1.00499999999999989… 로 담기지만 엑셀과 시트는 1.01 로 보여준다.
// Intl.NumberFormat 은 가장 짧은 십진수 표기를 기준으로 반올림하므로 이미
// 맞게 나온다 — (1.005).toFixed(2) 는 "1.00" 이라 쓸 수 없다.
//
// 서버의 TEXT 는 Go 의 FormatFloat 에 그대로 맡기고 있어서 "1.00" 을 냈다.
// 격자와 TEXT 가 다른 답을 내고 있었던 것이다. 아래 값은 서버의
// internal/formula/library_extended_test.go 와 **같은 값** 을 고정한다.
describe('the grid rounds the way the server does',()=>{
  it('rounds the decimal people typed, not the binary float',()=>{
    expect(formatCellValue(1.005,{number_format:'0.00'},'en-US')).toBe('1.01')
    expect(formatCellValue(2.675,{number_format:'0.00'},'en-US')).toBe('2.68')
    expect(formatCellValue(8.475,{number_format:'0.00'},'en-US')).toBe('8.48')
    expect(formatCellValue(1234.5,{number_format:'#,##0.00'},'en-US')).toBe('1,234.50')
    // toFixed 를 쓰면 아래가 "1.00" 이 된다. 쓰지 않는 이유다.
    expect((1.005).toFixed(2)).toBe('1.00')
  })
})

// 엑셀 서식은 값의 부호마다 다른 구역을 담고, 자리 기호 뒤의 쉼표는 자릿점이
// 아니라 천 단위 축약이다. 둘 다 재무 자료에서 온 파일에 흔하다.
//
// 예전에는 구역이 둘뿐인 줄 알고 0 을 늘 양수 구역으로 보냈고(회계 서식의 0 은
// "-" 인데 "0" 이었다), 비워 둔 구역을 감추지 않았으며, 쉼표가 있기만 하면
// 자릿점으로 보아 "#,##0,," 이 백만 배로 보였다. 아래 값은 서버의
// internal/formula/cell_formats_test.go 와 **같은 글자** 를 고정한다.
describe('the grid picks the format section the value belongs to',()=>{
  const shown=(value:number,format:string)=>formatCellValue(value,{number_format:format},'en-US')
  it('draws zero with the third section and hides an empty one',()=>{
    expect(shown(0,'#,##0;(#,##0)')).toBe('0')
    expect(shown(0,'#,##0;(#,##0);"-"')).toBe('-')
    expect(shown(-1234,'#,##0;(#,##0);"-"')).toBe('(1,234)')
    expect(shown(0,'0;-0;"영"')).toBe('영')
    expect(shown(-5,'0;;')).toBe('')
    expect(shown(0,'0;;')).toBe('')
    expect(shown(5,'0;;')).toBe('5')
    expect(shown(5,'"판매"')).toBe('판매')
    // 따옴표 안의 쌍반점은 구역을 가르지 않는다.
    expect(shown(-5,'#,##0;"내려감;"')).toBe('내려감;')
  })
  it('shrinks by a thousand for every comma behind the digits',()=>{
    expect(shown(1234567,'#,##0,')).toBe('1,235')
    expect(shown(1234567,'#,##0,,')).toBe('1')
    expect(shown(1234567,'0.0,,')).toBe('1.2')
    expect(shown(-1234567,'#,##0,,')).toBe('-1')
    // 자리 기호 사이의 쉼표는 그대로 자릿점이다.
    expect(shown(1234567,'#,##0')).toBe('1,234,567')
    expect(shown(1234567,'0')).toBe('1234567')
  })
})

// 엑셀 서식의 넷째 구역은 글자 값의 서식이다 — 회계 서식의 마지막 토막
// `_-@_-` 가 그것이고, `#,##0;(#,##0);"-";"["@"]"` 는 글자 칸을 [미정] 으로
// 그린다. 예전에는 그 구역을 아예 읽지 않아 글자 칸이 서식 없이 그대로
// 나왔고, 이름 뒤에 말을 붙이는 `@" 님"` 은 "@ 님" 이라고 그렸다.
// 아래 값은 서버의 internal/formula/cell_formats_test.go 와 **같은 글자** 다.
describe('the grid draws text with the fourth section',()=>{
  const shown=(value:unknown,format:string)=>formatCellValue(value,{number_format:format},'en-US')
  it('gives text values the fourth section and leaves shorter formats alone',()=>{
    expect(shown('미정','#,##0;(#,##0);"-";"["@"]"')).toBe('[미정]')
    expect(shown('미정','_-* #,##0_-;-* #,##0_-;_-* "-"_-;_-@_-')).toBe('미정')
    expect(shown('미정','0;;;"기타"')).toBe('기타')
    // 비워 둔 넷째 구역은 글자를 감춘다.
    expect(shown('미정','#,##0;;;')).toBe('')
    // 구역이 넷이 안 되면 글자를 손대지 않는다.
    expect(shown('미정','#,##0;(#,##0);"-"')).toBe('미정')
    expect(shown('미정','#,##0')).toBe('미정')
  })
  it('reads a lone section with @ as the text section',()=>{
    expect(shown('홍길동','@" 님"')).toBe('홍길동 님')
    expect(shown('홍길동','"("@")"')).toBe('(홍길동)')
    // @ 는 수에도 걸린다. 값을 있는 그대로 적는 자리다.
    expect(shown(5,'@" 님"')).toBe('5 님')
  })
  it('still draws numbers with the sections in front',()=>{
    expect(shown(1234,'#,##0;(#,##0);"-";"["@"]"')).toBe('1,234')
    expect(shown(-1234,'#,##0;(#,##0);"-";"["@"]"')).toBe('(1,234)')
    expect(shown(0,'#,##0;(#,##0);"-";"["@"]"')).toBe('-')
  })
})

// 분수 서식은 값을 소수가 아니라 분수로 적는다. 엑셀 기본 서식 12·13 번
// (`# ?/?`, `# ??/??`)이라 XLSX 로 그냥 들어오는데, 예전에는 자리 기호만 세고
// 빗금을 글자로 흘려 보내 `# ?/?` 가 0.5 를 "1/2" 가 아니라 "1/" 로 그렸다.
// 아래 값은 서버의 internal/formula/cell_formats_test.go 와 **같은 글자** 다.
describe('the grid draws fraction formats as fractions',()=>{
  const shown=(value:number,format:string)=>formatCellValue(value,{number_format:format},'en-US')
  it('picks the closest fraction the placeholders allow',()=>{
    expect(shown(.3125,'# ?/?')).toBe('1/3')
    expect(shown(.3125,'# ??/??')).toBe('5/16')
    expect(shown(3.14159,'# ???/???')).toBe('3 16/113')
    // 같은 만큼 어긋나면 분모가 작은 쪽이다.
    expect(shown(.5,'# ?/?')).toBe('1/2')
    expect(shown(.5,'# ??/??')).toBe('1/2')
    // 분모를 숫자로 못 박으면 약분하지 않는다.
    expect(shown(.5,'?/8')).toBe('4/8')
    expect(shown(.375,'?/8')).toBe('3/8')
  })
  it('splits off a whole part only when the format has one',()=>{
    expect(shown(2.75,'# ?/?')).toBe('2 3/4')
    expect(shown(5.25,'#/#')).toBe('21/4')
    expect(shown(.99,'# ?/?')).toBe('1')
    expect(shown(1,'# ?/?')).toBe('1')
    expect(shown(.5,'0 ?/?')).toBe('0 1/2')
    expect(shown(1234.5,'#,##0 ?/?')).toBe('1,234 1/2')
  })
  it('keeps the sign and the literals, and never draws -0',()=>{
    expect(shown(-2.75,'# ?/?')).toBe('-2 3/4')
    expect(shown(-.02,'# ?/?')).toBe('0')
    expect(shown(2.75,'# ?/?" 개"')).toBe('2 3/4 개')
    // 날짜 서식의 빗금은 자리 기호 사이에 있지 않으므로 분수가 아니다.
    expect(shown(45000.5,'m/d/yyyy')).toBe('3/15/2023')
  })
  // bestFraction 은 분모를 1 부터 상한까지 하나씩 재고 이 루프는 메인
  // 스레드에서 돈다. 자리 기호를 아홉 개 적으면 칸 하나가 10억 번 돌아
  // 그리드를 멈췄으므로 상한을 여섯 자리로 좁혔다 — 서버의
  // internal/formula/cell_formats_test.go 가 같은 글자를 못 박는다.
  it('stops counting denominator placeholders at six digits',()=>{
    expect(shown(1/3,'?????????/?????????')).toBe('1/3')
    expect(shown(1/3,'?????????/?????????')).toBe(shown(1/3,'??????/??????'))
    // 무리수는 자리를 넓힐수록 조금 더 가까워지므로 여섯 자리에서 멈춘
    // 값이 적힌다. 일부러 좁힌 것이다 — 아홉 자리를 끝까지 재던 때는
    // "3 192583/1360120" 이었다.
    expect(shown(Math.PI,'# ?????????/?????????')).toBe('3 51669/364913')
  })
})

// 엑셀 파일에서 읽어 온 날짜는 1899-12-30부터 센 날 수로 담긴다. 격자는
// 이 번호를 날짜로 보여주어야 하고, 서버의 수식도 같은 날짜로 읽어야 한다.
// 아래 값은 서버의 internal/formula/library_extended_test.go 와 **같은
// 날짜** 를 고정한다. 한쪽만 고치면 양쪽 다 걸린다.
describe('the grid reads the date serials an import brings in',()=>{
  it('turns a serial into the same day the server does',()=>{
    expect(formatCellValue(45306,{number_format:'yyyy-mm-dd'},'en-US')).toBe('2024-01-15')
    // 엑셀은 1900년을 윤년으로 잘못 센다. 60번은 없는 날이라, 그보다 작은
    // 번호는 하루 뒤에서 세야 1번이 1900-01-01이 된다.
    expect(formatCellValue(1,{number_format:'yyyy-mm-dd'},'en-US')).toBe('1900-01-01')
    expect(formatCellValue(59,{number_format:'yyyy-mm-dd'},'en-US')).toBe('1900-02-28')
    expect(formatCellValue(61,{number_format:'yyyy-mm-dd'},'en-US')).toBe('1900-03-01')
  })
})

// 표 서식에서 m은 앞뒤를 봐야 뜻이 정해진다. 시 뒤에 오거나 초 앞에 오면
// 분이고, 그 밖에는 달이다.
//
// 예전에는 서식을 공백으로 잘라 "날짜 한 토막"과 "시각 한 토막"만 그렸다.
// 그래서 "yyyy년 m월 d일" 처럼 토막이 셋인 한국어 날짜 서식이 앞 토막만
// 그려져 "2024년 m월 d일" 이 되었다. 요일과 달 이름도 그리지 못했다.
//
// 아래 값은 서버의 internal/formula/library_extended_test.go 와 **같은
// 글자** 를 고정한다. 한쪽만 고치면 양쪽 다 걸린다.
describe('the grid tells months from minutes in a date format',()=>{
  const shown=(value:number,format:string)=>formatCellValue(value,{number_format:format},'en-US')
  it('reads m by what sits beside it',()=>{
    expect(shown(0.5,'hh:mm')).toBe('12:00')
    expect(shown(0.5,'h:mm:ss')).toBe('12:00:00')
    expect(shown(0.5104166666666666,'hh:mm:ss')).toBe('12:15:00')
    expect(shown(0.5104166666666666,'h:m')).toBe('12:15')
    expect(shown(0.5104166666666666,'m:ss')).toBe('15:00')
    expect(shown(45306.25,'yyyy-mm-dd hh:mm')).toBe('2024-01-15 06:00')
    expect(shown(45306,'mm/dd/yyyy')).toBe('01/15/2024')
  })
  it('draws every piece of a format, not just the first',()=>{
    expect(shown(45306,'yyyy년 m월 d일')).toBe('2024년 1월 15일')
    expect(shown(45306,'mmmm')).toBe('January')
    expect(shown(45306.5,'mmm d')).toBe('Jan 15')
    expect(shown(45306,'dddd')).toBe('Monday')
    expect(shown(45306,'ddd')).toBe('Mon')
    expect(shown(45306.75,'h:mm am/pm')).toBe('6:00 PM')
    expect(shown(45306.25,'h:mm am/pm')).toBe('6:00 AM')
  })
})
