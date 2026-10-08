import { describe, expect, it } from 'vitest'
import { cellKey } from '../state/editor'
import { convertTextNumbers, detectDelimiter, removeDuplicateRows, splitLine, splitTextToColumns, splitWouldOverwrite, trimWhitespace, unmergeAndFill } from './dataCleanup'
import type { Cell } from '../types'

function grid(entries:Array<[number,number,unknown]>){
  const cells=new Map<string,Cell>()
  for(const [row,column,value] of entries)cells.set(cellKey(row,column),{sheet_id:'sheet',row,column,value,updated_at:''})
  return cells
}
const region=(startRow:number,startColumn:number,endRow:number,endColumn:number)=>({startRow,startColumn,endRow,endColumn})

describe('removeDuplicateRows',()=>{
  it('keeps the first occurrence and compacts the rows below it',()=>{
    const cells=grid([[1,1,'이름'],[1,2,'부서'],[2,1,'박지민'],[2,2,'영업'],[3,1,'박지민'],[3,2,'영업'],[4,1,'이서준'],[4,2,'개발']])
    const result=removeDuplicateRows(cells,region(1,1,4,2),1)
    expect(result.removed).toBe(1)
    // The unique third row moves up into the duplicate's place.
    expect(result.writes).toContainEqual({row:3,column:1,value:'이서준',formula:undefined,style:undefined})
    // The vacated last row is emptied instead of keeping a stale copy.
    expect(result.writes).toContainEqual({row:4,column:1,value:undefined,formula:undefined,style:undefined})
  })

  it('reports nothing to do when every row is unique',()=>{
    const cells=grid([[1,1,'가'],[2,1,'나']])
    expect(removeDuplicateRows(cells,region(1,1,2,1),0)).toEqual({writes:[],removed:0})
  })
})

describe('trimWhitespace',()=>{
  it('trims edges, collapses inner runs and leaves formulas alone',()=>{
    const cells=grid([[1,1,'  박지민  '],[2,1,'영업   1팀'],[3,1,'정상']])
    cells.set(cellKey(4,1),{sheet_id:'sheet',row:4,column:1,formula:'=A1',value:' 값 ',updated_at:''})
    const result=trimWhitespace(cells,region(1,1,4,1))
    expect(result.changed).toBe(2)
    expect(result.writes[0].value).toBe('박지민')
    expect(result.writes[1].value).toBe('영업 1팀')
  })
})

describe('splitTextToColumns',()=>{
  it('splits on the detected delimiter and keeps numbers numeric',()=>{
    const cells=grid([[1,1,'박지민,영업,1200'],[2,1,'이서준,개발,980']])
    const result=splitTextToColumns(cells,region(1,1,2,1),'auto')
    expect(result.separator).toBe(',')
    expect(result.columns).toBe(3)
    expect(result.writes).toContainEqual({row:1,column:3,value:1200,formula:undefined,style:undefined})
    expect(result.writes).toContainEqual({row:2,column:2,value:'개발',formula:undefined,style:undefined})
  })

  it('does nothing when the delimiter never appears',()=>{
    const cells=grid([[1,1,'박지민'],[2,1,'이서준']])
    expect(splitTextToColumns(cells,region(1,1,2,1),',').columns).toBe(0)
  })

  it('detects the delimiter that appears in the most rows',()=>{
    expect(detectDelimiter(['a;b','c;d','e f'])).toBe(';')
  })

  it('warns before overwriting neighbouring data',()=>{
    const cells=grid([[1,1,'a,b'],[1,2,'기존']])
    expect(splitWouldOverwrite(cells,region(1,1,1,1),2)).toBe(true)
    expect(splitWouldOverwrite(cells,region(1,1,1,1),1)).toBe(false)
  })
})

describe('splitLine',()=>{
  it('keeps a quoted field together',()=>{
    expect(splitLine('"서울, 강남",100,완료',',')).toEqual(['서울, 강남','100','완료'])
  })

  it('reads a doubled quote inside a quoted field as one quote',()=>{
    expect(splitLine('"그는 ""안녕"" 이라고",2',',')).toEqual(['그는 "안녕" 이라고','2'])
  })

  it('leaves quotes that are part of the text alone',()=>{
    expect(splitLine('15" 모니터,3',',')).toEqual(['15" 모니터','3'])
  })

  it('splits on a multi-character separator',()=>{
    expect(splitLine('서울 :: 100 :: 완료',' :: ')).toEqual(['서울','100','완료'])
  })
})

// 다른 곳에서 붙여 넣으면 "1,234" 처럼 글자로 들어온다. 사람 눈에는 숫자인데
// =SUM 은 조용히 빼고 셈한다 — 합계가 작게 나오는데 무엇이 빠졌는지는 아무
// 데도 적히지 않는다.
describe('convertTextNumbers',()=>{
  it('글자로 담긴 숫자만 바꾼다',()=>{
    const cells=new Map<string,Cell>()
    const put=(row:number,column:number,value:unknown,formula?:string)=>
      cells.set(`${row}:${column}`,{sheet_id:'s',row,column,value:value as never,formula,updated_at:''} as Cell)
    put(1,1,'1,234')
    put(2,1,'₩5,000')
    put(3,1,'(500)')
    put(4,1,1000)        // 이미 숫자다
    put(5,1,'2000')      // 수식도 세는 꼴이다
    put(6,1,'아무개')     // 숫자가 아니다
    put(7,1,'1,234',  '=A1')  // 수식 칸은 건드리지 않는다
    const result=convertTextNumbers(cells,region(1,1,7,1))
    expect(result.writes.map(write=>[write.row,write.value])).toEqual([[1,1234],[2,5000],[3,-500]])
  })
  it('숫자가 아닌 것을 억지로 만들지 않는다',()=>{
    const cells=new Map<string,Cell>()
    cells.set('1:1',{sheet_id:'s',row:1,column:1,value:'1,2,3' as never,updated_at:''} as Cell)
    expect(convertTextNumbers(cells,region(1,1,1,1)).writes).toHaveLength(0)
  })
})

describe('unmergeAndFill',()=>{
  const merge=(sr:number,sc:number,er:number,ec:number)=>({merge:{start_row:sr,start_column:sc,end_row:er,end_column:ec}})
  const cell=(row:number,column:number,value?:unknown,style?:Record<string,unknown>):Cell=>
    ({sheet_id:'s',row,column,updated_at:'',...(value===undefined?{}:{value}),...(style?{style}:{})})
  const map=(items:Cell[])=>new Map(items.map(item=>[cellKey(item.row,item.column),item]))
  const whole={startRow:1,startColumn:1,endRow:4,endColumn:3}

  it('fills the value the person could see into the cells it covered',()=>{
    const box=merge(2,1,4,1)
    const cells=map([cell(2,1,'영업1팀',box),cell(3,1,undefined,box),cell(4,1,undefined,box)])
    const result=unmergeAndFill(cells,whole)
    expect(result.ranges).toBe(1)
    expect(result.filled).toBe(2)
    expect(result.writes.map(write=>write.value)).toEqual(['영업1팀','영업1팀','영업1팀'])
    // 병합이 남아 있으면 표는 여전히 정렬되지 않는다.
    for(const write of result.writes)expect(write.style?.merge).toBeUndefined()
  })

  it('never overwrites a value the merge was only hiding',()=>{
    // kanpic 의 병합은 값을 지우지 않고 가린다. 덮어 채우면 사람이 보지
    // 못하던 값을 없애는 것이 된다 — 부탁받은 일이 아니다.
    const box=merge(2,1,3,1)
    const cells=map([cell(2,1,'보이는 값',box),cell(3,1,'가려진 값',box)])
    const result=unmergeAndFill(cells,whole)
    expect(result.filled).toBe(0)
    expect(result.kept).toBe(1)
    expect(result.writes.map(write=>write.value)).toEqual(['보이는 값','가려진 값'])
  })

  it('keeps the rest of the formatting',()=>{
    const cells=map([cell(2,1,'제목',{...merge(2,1,2,2),bold:true}),cell(2,2,undefined,{...merge(2,1,2,2),bold:true})])
    const result=unmergeAndFill(cells,whole)
    for(const write of result.writes)expect(write.style?.bold).toBe(true)
  })

  it('unmerges the whole range even where it reaches outside the selection',()=>{
    // 절반만 풀면 어느 곳에도 적히지 않는 반쪽짜리 병합이 남는다.
    const box=merge(2,1,2,3)
    const cells=map([cell(2,1,'하나',box),cell(2,2,undefined,box),cell(2,3,undefined,box)])
    const result=unmergeAndFill(cells,{startRow:2,startColumn:1,endRow:2,endColumn:1})
    expect(result.writes.map(write=>`${write.row}:${write.column}`)).toEqual(['2:1','2:2','2:3'])
  })

  it('does nothing when there is no merge',()=>{
    expect(unmergeAndFill(map([cell(1,1,'가')]),whole).writes).toEqual([])
  })
})
