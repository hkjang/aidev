import { describe, expect, it } from 'vitest'
import { columnStats, looksLikeHeader, statNumber } from './columnStats'
import type { Cell } from '../types'

const column=(values:Array<[number,unknown]>,columnIndex=1)=>
  values.map(([row,value])=>({sheet_id:'s',row,column:columnIndex,value,updated_at:''} as Cell))

describe('columnStats', () => {
  it('counts what is filled, empty and distinct', () => {
    const stats=columnStats(column([[1,'서울'],[2,'부산'],[3,'서울'],[5,'']]),1,1,6)
    expect(stats).toMatchObject({scanned:6,filled:3,empty:3,unique:2,numbers:0})
    expect(stats.frequent[0]).toMatchObject({value:'서울',count:2,share:2/3})
  })

  it('summarises a numeric column', () => {
    const stats=columnStats(column([[1,10],[2,20],[3,30],[4,40]]),1,1,4)
    expect(stats).toMatchObject({sum:100,average:25,median:25,min:10,max:40})
    expect(stats.deviation).toBeCloseTo(12.9099,3)
    expect(stats.buckets.reduce((total,bucket)=>total+bucket.count,0)).toBe(4)
  })

  // 쉼표가 붙어 글자로 담긴 것은 세지 않는다. 예전에는 셌지만, 그러면 이
  // 화면에만 나오고 어떤 수식으로도 재현되지 않는 합계가 된다 — =SUM 은 그
  // 칸을 빼고 셈하기 때문이다. 같은 열에서 두 가지 합계가 나오면 사람은
  // 어느 쪽을 믿어야 할지 알 수 없다.
  //
  // 그 자료를 쓰고 싶으면 숫자로 바꾸면 된다. 데이터 정리에 그 자리가 있다.
  it('leaves numbers that arrived as text out, the way a formula does', () => {
    // 셀 수 있는 숫자가 없으면 합계를 아예 내지 않는다. 0 으로 내면 사람은
    // 그것을 답으로 읽는다.
    expect(columnStats(column([[1,'1,500'],[2,'2,500']]),1,1,2).sum).toBeUndefined()
    // 쉼표 없이 담긴 것은 수식도 세므로 여기서도 센다.
    expect(columnStats(column([[1,'1500'],[2,'2500']]),1,1,2).sum).toBe(4000)
  })

  it('leaves other columns out', () => {
    const cells=[...column([[1,10]],1),...column([[1,999]],2)]
    expect(columnStats(cells,1,1,1).sum).toBe(10)
  })

  it('has no spread for a single number and no buckets for none', () => {
    expect(columnStats(column([[1,7]]),1,1,1).deviation).toBe(0)
    expect(columnStats(column([[1,7],[2,7]]),1,1,2).buckets).toEqual([{from:7,to:7,count:2}])
    expect(columnStats(column([[1,'가']]),1,1,1).buckets).toEqual([])
  })

  it('reports an empty column without dividing by zero', () => {
    const stats=columnStats([],1,1,10)
    expect(stats).toMatchObject({scanned:10,filled:0,empty:10,unique:0,numbers:0,frequent:[],buckets:[]})
    expect(stats.average).toBeUndefined()
  })
})

describe('statNumber', () => {
  it('keeps numbers readable and says nothing when there is nothing', () => {
    expect(statNumber(1234567.891)).toBe('1,234,567.89')
    expect(statNumber(undefined)).toBe('—')
  })
})

describe('looksLikeHeader', () => {
  it('sees a label sitting on top of numbers', () => {
    expect(looksLikeHeader(column([[1,'매출'],[2,1200],[3,900]]),1,1,3)).toBe(true)
  })

  it('leaves a column of text alone', () => {
    expect(looksLikeHeader(column([[1,'지역'],[2,'서울'],[3,'부산']]),1,1,3)).toBe(false)
  })

  it('needs the first row to be text', () => {
    expect(looksLikeHeader(column([[1,10],[2,20]]),1,1,2)).toBe(false)
    expect(looksLikeHeader(column([[2,20]]),1,1,2)).toBe(false)
  })

  it('skips blanks between the label and the first number', () => {
    expect(looksLikeHeader(column([[1,'매출'],[3,1200]]),1,1,3)).toBe(true)
  })
})
