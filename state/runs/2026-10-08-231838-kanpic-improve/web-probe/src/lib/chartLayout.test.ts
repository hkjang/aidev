import { describe, expect, it } from 'vitest'
import { chartShape, shapeExtent, stackedBases } from './chartLayout'
import type { ChartSeries } from '../types'

const series=(name:string,values:number[]):ChartSeries=>
  ({name,points:values.map((value,index)=>({category:`C${index+1}`,value}))} as ChartSeries)

describe('chartShape', () => {
  it('draws a combination chart as bars plus a line for the last series', () => {
    const shape=chartShape('combo',[series('매출',[1,2]),series('비용',[1,1]),series('이익률',[0.1,0.2])],true)
    expect(shape.bars.map(item=>item.name)).toEqual(['매출','비용'])
    expect(shape.lines.map(item=>item.name)).toEqual(['이익률'])
    expect(shape.secondary).toBe(true)
  })

  it('falls back to a line when a combination chart has one series', () => {
    const shape=chartShape('combo',[series('매출',[1,2])])
    expect(shape.bars).toEqual([])
    expect(shape.lines).toHaveLength(1)
    expect(shape.secondary).toBe(false)
  })

  it('marks the stacked and filled kinds', () => {
    expect(chartShape('stacked_bar',[series('a',[1])])).toMatchObject({stacked:true,filled:false})
    expect(chartShape('stacked_area',[series('a',[1])])).toMatchObject({stacked:true,filled:true})
    expect(chartShape('area',[series('a',[1])])).toMatchObject({stacked:false,filled:true})
    expect(chartShape('bar',[series('a',[1])]).bars).toHaveLength(1)
  })
})

describe('stackedBases', () => {
  it('puts each series on the sum of the ones before it', () => {
    const {bases,totals}=stackedBases([series('a',[10,20]),series('b',[5,5])],2)
    expect(bases[0]).toEqual([0,0])
    expect(bases[1]).toEqual([10,20])
    expect(totals).toEqual([15,25])
  })
})

describe('shapeExtent', () => {
  it('reaches the top of the stack, not the tallest single value', () => {
    expect(shapeExtent([series('a',[10,20]),series('b',[5,5])],true).max).toBe(25)
    expect(shapeExtent([series('a',[10,20]),series('b',[5,5])],false).max).toBe(20)
  })

  it('always includes the baseline and never collapses to a point', () => {
    expect(shapeExtent([series('a',[5,5])],false)).toEqual({min:0,max:5})
    expect(shapeExtent([],false)).toEqual({min:0,max:1})
  })
})

describe('축 범위', () => {
  const series=[{name:'매출',points:[{category:'1월',value:50},{category:'2월',value:70}]}]
  // 0 에서 시작하지 않으면 작은 차이가 크게 보인다. 사람이 그렇게 정했으면
  // 자료에 맞춰 다시 정해 주면 안 된다.
  it('사람이 정한 범위를 따른다', () => {
    expect(shapeExtent(series,false)).toEqual({min:0,max:70})
    expect(shapeExtent(series,false,{min:40,max:80})).toEqual({min:40,max:80})
    // 한쪽만 정해도 된다.
    expect(shapeExtent(series,false,{min:40})).toEqual({min:40,max:70})
    expect(shapeExtent(series,false,{max:100})).toEqual({min:0,max:100})
  })

  it('비워 두면 자료에 맞춘다', () => {
    expect(shapeExtent(series,false,{min:null,max:null})).toEqual({min:0,max:70})
    expect(shapeExtent(series,false,{})).toEqual({min:0,max:70})
  })

  // 저장할 때 막지만 예전 자료가 뒤집혀 있을 수 있다. 그래도 그려야 한다.
  it('뒤집혀 들어와도 그릴 수 있는 범위를 낸다', () => {
    const extent=shapeExtent(series,false,{min:80,max:40})
    expect(extent.max).toBeGreaterThan(extent.min)
  })
})
