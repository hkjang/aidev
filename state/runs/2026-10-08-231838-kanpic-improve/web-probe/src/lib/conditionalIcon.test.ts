import { describe,expect,it } from 'vitest'
import { ICON_STYLES,iconGlyph,iconGutter,iconSetPreview } from './conditionalIcon'
import type { ConditionalIconStyle } from '../types'

describe('conditionalIcon', () => {
  it('has as many icons as the style name promises', () => {
    for(const style of ICON_STYLES){
      expect(iconSetPreview(style)).toHaveLength(Number(style[0]))
    }
  })

  // 색만으로 구분되는 집합은 신호등뿐이다. 나머지는 방향이나 모양이 함께
  // 달라야 색을 못 보는 사람도 순서를 읽을 수 있다.
  it('separates its icons by more than colour', () => {
    for(const style of ICON_STYLES){
      if(style==='3TrafficLights1')continue
      const shapes=iconSetPreview(style).map(glyph=>JSON.stringify({...glyph,color:''}))
      expect(new Set(shapes).size).toBe(shapes.length)
    }
  })

  it('reads the icon the server picked', () => {
    expect(iconGlyph({style:'3Arrows',index:0,count:3})).toEqual({kind:'arrow',color:'#dc2626',angle:90})
    expect(iconGlyph({style:'3Arrows',index:2,count:3})).toEqual({kind:'arrow',color:'#16a34a',angle:-90})
  })

  // 서버가 새 아이콘 집합을 알게 되어도 화면이 죽지는 않는다.
  it('survives an index or a style it does not know', () => {
    expect(iconGlyph({style:'3Arrows',index:9,count:3})).toEqual({kind:'arrow',color:'#16a34a',angle:-90})
    expect(iconGlyph({style:'3Arrows',index:-1,count:3})).toEqual({kind:'arrow',color:'#dc2626',angle:90})
    expect(iconGlyph({style:'7Hearts' as ConditionalIconStyle,index:0,count:7})).toBeNull()
  })

  it('leaves room for the value beside the icon', () => {
    expect(iconGutter(12)).toBe(18)
  })
})
