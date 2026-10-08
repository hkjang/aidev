import { describe, expect, it } from 'vitest'
import { leadingNumberSeriesValue, listSeriesValue } from './fillSeries'

const walk=(values:unknown[],count:number,run:(values:unknown[],position:number)=>string|undefined)=>
  Array.from({length:count},(_unused,index)=>run(values,index+values.length))

describe('listSeriesValue',()=>{
  it('carries on through the names a spreadsheet knows',()=>{
    expect(walk(['1월'],3,listSeriesValue)).toEqual(['2월','3월','4월'])
    expect(walk(['월요일'],3,listSeriesValue)).toEqual(['화요일','수요일','목요일'])
    expect(walk(['Jan'],3,listSeriesValue)).toEqual(['Feb','Mar','Apr'])
    expect(walk(['Monday'],2,listSeriesValue)).toEqual(['Tuesday','Wednesday'])
    expect(walk(['1분기'],2,listSeriesValue)).toEqual(['2분기','3분기'])
  })

  it('comes back round instead of inventing a thirteenth month',()=>{
    expect(walk(['11월'],3,listSeriesValue)).toEqual(['12월','1월','2월'])
    expect(walk(['토'],3,listSeriesValue)).toEqual(['일','월','화'])
    expect(walk(['4분기'],2,listSeriesValue)).toEqual(['1분기','2분기'])
  })

  it('keeps the gap between two seeds',()=>{
    expect(walk(['월','수'],4,listSeriesValue)).toEqual(['금','일','화','목'])
    expect(walk(['1월','4월'],2,listSeriesValue)).toEqual(['7월','10월'])
  })

  it('writes the answer the way the seed was written',()=>{
    expect(walk(['mon'],2,listSeriesValue)).toEqual(['tue','wed'])
    expect(walk(['MON'],2,listSeriesValue)).toEqual(['TUE','WED'])
  })

  it('refuses anything that is not one steady walk through one list',()=>{
    // 다른 목록끼리는 한 줄로 보지 않는다.
    expect(listSeriesValue(['월요일','Jan'],2)).toBeUndefined()
    // 간격이 들쭉날쭉하면 무엇을 이어야 할지 알 수 없다.
    expect(listSeriesValue(['월','수','목'],3)).toBeUndefined()
    // 제자리걸음은 늘어나는 것이 아니다.
    expect(listSeriesValue(['월','월'],2)).toBeUndefined()
    expect(listSeriesValue(['제품A'],1)).toBeUndefined()
    expect(listSeriesValue([1,2],2)).toBeUndefined()
    expect(listSeriesValue([],0)).toBeUndefined()
  })
})

describe('leadingNumberSeriesValue',()=>{
  it('counts on when the number comes first',()=>{
    expect(walk(['3반'],2,leadingNumberSeriesValue)).toEqual(['4반','5반'])
    expect(walk(['1차 검토'],2,leadingNumberSeriesValue)).toEqual(['2차 검토','3차 검토'])
    expect(walk(['2팀','4팀'],2,leadingNumberSeriesValue)).toEqual(['6팀','8팀'])
  })

  it('keeps the padding the seed was written with',()=>{
    expect(walk(['08호'],3,leadingNumberSeriesValue)).toEqual(['09호','10호','11호'])
  })

  it('leaves alone what is not a number followed by a name',()=>{
    expect(leadingNumberSeriesValue(['항목 1'],1)).toBeUndefined()
    expect(leadingNumberSeriesValue(['123'],1)).toBeUndefined()
    expect(leadingNumberSeriesValue(['2팀','3반'],1)).toBeUndefined()
    expect(leadingNumberSeriesValue([42],1)).toBeUndefined()
  })
})
