import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { compareNatural } from './naturalOrder'

const ordered=(values:string[])=>[...values].sort(compareNatural)

// 서버의 internal/workbook/natural_order.go 와 같은 답을 내야 한다. 정렬은
// 화면에서 먼저 반영하고 서버가 다시 확정하므로, 둘이 어긋나면 값이 튄다.
describe('compareNatural',()=>{
  it('reads the digits in a value as a number',()=>{
    expect(ordered(['2월','10월','1월','12월','3월'])).toEqual(['1월','2월','3월','10월','12월'])
    expect(ordered(['항목10','항목2','항목1'])).toEqual(['항목1','항목2','항목10'])
    expect(ordered(['a10b','a2b','a2a'])).toEqual(['a2a','a2b','a10b'])
  })

  it('leaves text without digits alone',()=>{
    expect(ordered(['나','가','다'])).toEqual(['가','나','다'])
    expect(compareNatural('가','가')).toBe(0)
  })

  it('keeps a steady order for the same number written differently',()=>{
    expect(ordered(['7호','07호'])).toEqual(['07호','7호'])
    expect(compareNatural('007','7')).toBeLessThan(0)
  })

  it('orders numbers too long for a float exactly',()=>{
    expect(ordered(['9999999999999999999','1111111111111111111']))
      .toEqual(['1111111111111111111','9999999999999999999'])
    expect(compareNatural('10000000000000000000','9999999999999999999')).toBeGreaterThan(0)
  })

  it('puts a digit before a letter at the same position',()=>{
    expect(compareNatural('1가','가1')).toBeLessThan(0)
  })

  it('treats a longer value as greater when it starts the same',()=>{
    expect(compareNatural('가','가나')).toBeLessThan(0)
    expect(compareNatural('가나','가')).toBeGreaterThan(0)
  })
})

// 서버와 화면이 같은 차례를 내야 한다. 정렬은 화면에서 먼저 반영하고 서버가
// 다시 확정하므로, 둘이 어긋나면 줄이 눈앞에서 한 번 튄다.
//
// 아래 목록은 서버의 internal/workbook/natural_order.go 시험과 **같은 값** 을
// 고정한다. 한쪽만 고치면 양쪽 다 걸린다.
describe('the browser orders text the way the server does', () => {
  it('puts characters in code point order, surrogates included', () => {
    // 자바스크립트의 기본 문자열 비교는 UTF-16 조각을 견주므로 이모지를
    // ￦(U+FFE6) 앞에 놓는다. 서버는 UTF-8 바이트를 견주어 뒤에 놓는다.
    const sorted=['￦100','😀항목','가나다','항목2','항목10','ZZ','＀','�끝','🍎사과','abc'].sort(compareNatural)
    expect(sorted).toEqual(['ZZ','abc','가나다','항목2','항목10','＀','￦100','�끝','🍎사과','😀항목'])
  })

  it('does not fall back to the built-in comparison', () => {
    // 이 한 쌍이 두 방식을 가른다.
    expect(compareNatural('😀','￦')).toBe(1)
    expect('😀'<'￦').toBe(true)
  })
})

// 정렬은 화면에서 먼저 반영하고 서버가 다시 확정한다. 두 곳의 비교가
// 어긋나면 줄이 눈앞에서 한 번 튄다. 실제로 두 번 어긋났었다 — UTF-16
// 조각과 UTF-8 바이트를 견주던 것, 대소문자를 낮추는 방식이 다르던 것.
//
// testdata/sort-order.json 을 서버의 internal/workbook/sort_order_test.go
// 와 함께 읽는다. 한쪽만 고치면 양쪽 다 걸린다.
//
// 목록에는 그리스어 낱말 끝 시그마, 튀르키예어 İ, 독일어 ß, 미리 합친
// 글자와 나눠 적은 글자, 이어붙인 이모지, 아랍 숫자가 섞여 있다.
describe('the grid sorts the way the server does',()=>{
  it('agrees on every string in testdata/sort-order.json',()=>{
    const fixture=JSON.parse(readFileSync('../testdata/sort-order.json','utf8')) as {corpus:string[];sorted:string[];sortedCaseSensitive:string[]}
    expect(fixture.corpus.length).toBeGreaterThan(100)
    for(const [sensitive,want] of [[false,fixture.sorted],[true,fixture.sortedCaseSensitive]] as [boolean,string[]][]){
      const items=[...fixture.corpus]
      items.sort((left,right)=>compareNatural(sensitive?left:left.toLowerCase(),sensitive?right:right.toLowerCase()))
      const first=items.findIndex((value,index)=>value!==want[index])
      expect({sensitive,index:first,got:first<0?null:items[first],want:first<0?null:want[first]})
        .toEqual({sensitive,index:-1,got:null,want:null})
    }
  })
})
