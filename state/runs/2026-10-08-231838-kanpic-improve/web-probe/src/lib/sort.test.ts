import { describe,expect,it } from 'vitest'
import type { Cell } from '../types'
import { materializeSort } from './sort'

const cell=(row:number,column:number,value:unknown,formula?:string,style?:Record<string,unknown>):Cell=>({sheet_id:'sheet',row,column,value,formula,style,updated_at:'now'})

describe('range sorting',()=>{
  it('applies stable multi-key sorting and shifts formulas with styles',()=>{
    const cells=new Map<string,Cell>([
      ['1:1',cell(1,1,'Name')],['1:2',cell(1,2,'Quantity')],
      ['2:1',cell(2,1,'beta')],['2:2',cell(2,2,2)],['2:3',cell(2,3,20,'=B2*10',{bold:true})],
      ['3:1',cell(3,1,'Alpha')],['3:2',cell(3,2,10)],['3:3',cell(3,3,100,'=B3*10')],
      ['4:1',cell(4,1,'alpha')],['4:2',cell(4,2,5)],['4:3',cell(4,3,50,'=B4*10')],
    ])
    const result=materializeSort(cells,{startRow:1,startColumn:1,endRow:4,endColumn:3},{headerRows:1,caseSensitive:false,keys:[{column:1,direction:'asc'},{column:2,direction:'desc'}]},'sheet')
    expect([result[0].value,result[3].value,result[6].value]).toEqual(['Alpha','alpha','beta'])
    expect([result[2].formula,result[5].formula,result[8].formula]).toEqual(['=B2*10','=B3*10','=B4*10'])
    expect(result[8].style).toEqual({bold:true})
  })

  it('keeps blanks last for descending sort',()=>{
    const cells=new Map<string,Cell>([['1:1',cell(1,1,2)],['3:1',cell(3,1,9)],['4:1',cell(4,1,1)]])
    const result=materializeSort(cells,{startRow:1,startColumn:1,endRow:4,endColumn:1},{headerRows:0,caseSensitive:false,keys:[{column:1,direction:'desc'}]},'sheet')
    expect(result.map(item=>item.value)).toEqual([9,2,1,undefined])
  })

  it('rejects duplicate keys, merged cells, and oversized operations',()=>{
    const range={startRow:1,startColumn:1,endRow:3,endColumn:2}
    expect(()=>materializeSort(new Map(),range,{headerRows:0,caseSensitive:false,keys:[{column:1,direction:'asc'},{column:1,direction:'desc'}]},'sheet')).toThrow('중복 없이')
    const merged=cell(1,1,'x',undefined,{merge:{start_row:1,start_column:1,end_row:2,end_column:1}})
    expect(()=>materializeSort(new Map([['1:1',merged]]),range,{headerRows:0,caseSensitive:false,keys:[{column:1,direction:'asc'}]},'sheet')).toThrow('병합 해제')
    // 정렬은 붙여넣기보다 넓은 한도를 가진다. 한도를 넘으면 몇 셀까지인지와
    // 그것이 몇 행인지를 함께 알려 준다.
    expect(()=>materializeSort(new Map(),{startRow:1,startColumn:1,endRow:5001,endColumn:2},{headerRows:0,caseSensitive:false,keys:[{column:1,direction:'asc'}]},'sheet')).not.toThrow()
    expect(()=>materializeSort(new Map(),{startRow:1,startColumn:1,endRow:30001,endColumn:2},{headerRows:0,caseSensitive:false,keys:[{column:1,direction:'asc'}]},'sheet')).toThrow('60,000셀(30,000행 × 2열)')
  })
})

// 대소문자를 무시하는 정렬은 화면과 서버가 **같은 방식** 으로 글자를 낮춰야
// 한다. 자바스크립트의 toLowerCase 는 유니코드 규칙을 그대로 따르지만 Go 의
// strings.ToLower 는 글자 하나씩만 보는 단순 변환이라 두 군데에서 갈렸다.
// 서버가 x/text 의 cases.Lower 를 쓰도록 고쳐 맞추었다.
//
// 아래 목록은 서버의 internal/workbook/sort_test.go 와 **같은 값** 을 고정한다.
// 한쪽만 고치면 양쪽 다 걸린다. 넣는 차례를 일부러 답과 다르게 두어, 안정
// 정렬이 손대지 않고 지나가는 것만으로는 통과하지 못하게 했다.
describe('ignoring case lowers letters the way the server does',()=>{
  it('keeps the Turkish dotted İ and the Greek final sigma in the same place',()=>{
    const values=['İD','id','ID','οδοσ','ΟΔΟΣ','ΟΔΟΤ']
    const cells=new Map<string,Cell>(values.map((value,index)=>[`${index+1}:1`,cell(index+1,1,value)]))
    const sorted=materializeSort(cells,{startRow:1,startColumn:1,endRow:values.length,endColumn:1},{keys:[{column:1,direction:'asc'}],headerRows:0,caseSensitive:false},'sheet')
    const got:string[]=[]
    for(const item of sorted)got[item.row-1]=item.value as string
    expect(got).toEqual(['id','ID','İD','ΟΔΟΣ','οδοσ','ΟΔΟΤ'])
  })
})
