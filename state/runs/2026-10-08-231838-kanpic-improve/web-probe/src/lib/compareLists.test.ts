import { describe,expect,it } from 'vitest'
import { cellKey } from '../state/editor'
import type { Cell } from '../types'
import { compareKey, compareLists, splitSheetRange } from './compareLists'

const cell=(row:number,column:number,value?:unknown):Cell=>({sheet_id:'s',row,column,updated_at:'',...(value===undefined?{}:{value})})
const side=(values:unknown[],headerRows=0)=>({
  cells:new Map(values.map((value,index)=>[cellKey(index+1,1),cell(index+1,1,value)])),
  region:{startRow:1,startColumn:1,endRow:values.length,endColumn:1},
  keyColumn:1,headerRows,
})

describe('compareKey',()=>{
  it('reads a number written as text as the same key',()=>{
    // 은행 내역은 "1,234" 로 오고 장부에는 1234 로 적혀 있다. 글자 그대로
    // 견주면 하나도 맞지 않는다.
    expect(compareKey('1,234')).toBe(compareKey(1234))
    expect(compareKey('₩5,000')).toBe(compareKey(5000))
    expect(compareKey(' 300 ')).toBe(compareKey(300))
  })
  it('ignores surrounding spaces and letter case',()=>{
    expect(compareKey(' ABC ')).toBe(compareKey('abc'))
  })
  it('has no key for a blank cell',()=>{
    expect(compareKey('')).toBeUndefined()
    expect(compareKey(null)).toBeUndefined()
    expect(compareKey('   ')).toBeUndefined()
  })
  it('does not force text that is not a number',()=>{
    expect(compareKey('1,2,3')).toBe('1,2,3')
  })
})

describe('compareLists',()=>{
  it('says what is on one side only',()=>{
    const result=compareLists(side(['가','나','다']),side(['나','다','라']))
    expect(result.onlyLeft.map(row=>row.label)).toEqual(['가'])
    expect(result.onlyRight.map(row=>row.label)).toEqual(['라'])
    expect(result.both).toBe(2)
  })

  it('matches a bank list written as text against a ledger of numbers',()=>{
    const result=compareLists(side(['1,000','2,000','3,000']),side([1000,2000,4000]))
    expect(result.both).toBe(2)
    expect(result.onlyLeft.map(row=>row.label)).toEqual(['3,000'])
    expect(result.onlyRight.map(row=>row.label)).toEqual(['4000'])
  })

  it('reports a key that appears twice instead of hiding it',()=>{
    // 대사에서 같은 번호가 두 번 나오는 것은 발견이지 잡음이 아니다.
    const result=compareLists(side(['가','가','나']),side(['가','나']))
    expect(result.duplicated).toEqual([{side:'left',key:'가',label:'가',count:2}])
    expect(result.both).toBe(2)
  })

  it('counts rows whose key cell is empty rather than dropping them',()=>{
    // 조용히 빼면 "다 맞았다" 는 대사표가 나오고 수는 맞지 않는다.
    const cells=new Map([[cellKey(1,1),cell(1,1,'가')],[cellKey(2,1),cell(2,1,'')],[cellKey(2,2),cell(2,2,'금액만 있음')]])
    const left={cells,region:{startRow:1,startColumn:1,endRow:2,endColumn:2},keyColumn:1,headerRows:0}
    expect(compareLists(left,side(['가'])).blank.left).toBe(1)
  })

  it('skips the header row when told to',()=>{
    const result=compareLists(side(['이름','가'],1),side(['이름','가'],1))
    expect(result.both).toBe(1)
    expect(result.onlyLeft).toEqual([])
  })
})

describe('splitSheetRange',()=>{
  it('splits a sheet name from a range',()=>{
    expect(splitSheetRange('장부!A1:C100')).toEqual({sheetName:'장부',region:{startRow:1,startColumn:1,endRow:100,endColumn:3}})
  })
  it('splits at the last mark so a name may contain one',()=>{
    // "1분기!실적" 이라는 이름의 시트를 앞에서 자르면 영영 못 고른다.
    expect(splitSheetRange('1분기!실적!A1:B2')?.sheetName).toBe('1분기!실적')
  })
  it('unwraps the quotes a name with spaces gets',()=>{
    expect(splitSheetRange("'상반기 장부'!A1:B2")?.sheetName).toBe('상반기 장부')
  })
  it('leaves the sheet unnamed when there is no mark',()=>{
    expect(splitSheetRange('A1:B2')).toEqual({region:{startRow:1,startColumn:1,endRow:2,endColumn:2}})
  })
  it('refuses what is not a range',()=>{
    expect(splitSheetRange('장부!')).toBeUndefined()
    expect(splitSheetRange('장부!A1')).toBeUndefined()
    expect(splitSheetRange('')).toBeUndefined()
    expect(splitSheetRange('!A1:B2')).toBeUndefined()
  })
})

describe('keys that must not be read as numbers',()=>{
  // 대사에서 틀리게 "맞았다" 고 하는 것은 "안 맞았다" 고 하는 것보다 나쁘다.
  // 안 맞으면 사람이 들여다보지만, 맞았다고 하면 거기서 끝난다.
  it('keeps a leading zero apart from the bare number',()=>{
    expect(compareKey('007')).not.toBe(compareKey('7'))
    expect(compareKey('007')).toBe('007')
  })
  it('keeps two long account numbers apart',()=>{
    // 배정밀도로 읽으면 뒤가 뭉개져 둘이 같은 키가 된다.
    expect(compareKey('12345678901234567890')).not.toBe(compareKey('12345678901234567891'))
  })
  it('still matches amounts written with separators',()=>{
    expect(compareKey('1,000')).toBe(compareKey(1000))
    expect(compareKey('0')).toBe(compareKey(0))
    expect(compareKey('0.5')).toBe(compareKey(0.5))
  })
  it('finds the difference two long numbers really have',()=>{
    const result=compareLists(side(['12345678901234567890']),side(['12345678901234567891']))
    expect(result.both).toBe(0)
    expect(result.onlyLeft).toHaveLength(1)
    expect(result.onlyRight).toHaveLength(1)
  })
})
