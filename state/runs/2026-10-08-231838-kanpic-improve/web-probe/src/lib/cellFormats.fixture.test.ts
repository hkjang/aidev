import {describe,expect,it} from 'vitest'
import {readFileSync} from 'node:fs'
import {formatCellValue} from './cellFormat'

// 격자와 서버의 TEXT 는 같은 값을 같은 글자로 적어야 한다. 두 곳에서 따로
// 셈하므로 어긋나기 쉽고, 실제로 여러 번 어긋나 있었다 — 시각의 mm 을 달로
// 읽거나, 지수 자리를 채우지 않거나, 음수를 날짜로 그리거나.
//
// testdata/cell-formats.json 을 서버의 internal/formula/cell_formats_test.go
// 와 함께 읽는다. 한쪽만 고치면 양쪽 다 걸린다.
// 값은 수만이 아니다. 넷째 구역(글자 구역)은 글자 값에만 걸린다.
type FormatCase={value:unknown;format:string;text:string}

describe('the grid writes what the server writes',()=>{
  it('agrees on every format in testdata/cell-formats.json',()=>{
    const fixture=JSON.parse(readFileSync('../testdata/cell-formats.json','utf8')) as {locale:string;cases:FormatCase[]}
    expect(fixture.cases.length).toBeGreaterThan(100)
    const wrong:string[]=[]
    for(const item of fixture.cases){
      const drawn=formatCellValue(item.value,{number_format:item.format},fixture.locale)
      if(drawn!==item.text)wrong.push(`${JSON.stringify(item.value)} + "${item.format}" -> ${JSON.stringify(drawn)}, 서버는 ${JSON.stringify(item.text)}`)
    }
    expect(wrong.slice(0,20)).toEqual([])
  })
})
