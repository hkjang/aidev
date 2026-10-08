import {it,expect} from 'vitest'
import {render,screen,cleanup} from '@testing-library/react'
import {compareKey,compareLists,splitSheetRange} from './compareLists'
import {CompareDialog} from '../components/CompareDialog'
const side=(values:unknown[])=>({cells:new Map(values.map((value,i)=>[`${i+1}:1`,{sheet_id:'s',row:i+1,column:1,value,updated_at:''}])),region:{startRow:1,startColumn:1,endRow:values.length,endColumn:1},keyColumn:1,headerRows:0})
it('observes current nondecimal collisions through production comparison',()=>{
  for(const [text,num] of [['0x10',16],['0b10',2],['0o10',8]] as const){
    const r=compareLists(side([text]),side([num]))
    console.log(JSON.stringify({text,num,key:compareKey(text),result:r}))
    expect(r.both).toBe(1)
    expect(r.onlyLeft).toEqual([])
  }
  const dup=compareLists(side(['0x10',16]),side([16]))
  console.log('false duplicate',JSON.stringify(dup))
  expect(dup.duplicated[0].count).toBe(2)
})
it('observes false complete match rendered by production dialog',()=>{
  const source=(values:unknown[])=>({...side(values),sheet:{id:'s',name:'자료',workbook_id:'w',position:0,row_count:100,column_count:10} as any})
  render(<CompareDialog left={source(['번호','0x10'])} right={source(['번호',16])} onClose={()=>{}} onReport={async()=>{}}/>)
  expect(screen.getByText('두 목록이 키 기준으로 완전히 맞습니다.')).toBeTruthy()
  console.log('production UI renders false complete match')
  cleanup()
})
it('observes two independent new candidates',()=>{
  console.log('boolean keys',compareKey(true),compareKey('TRUE'))
  expect(compareLists(side([true]),side(['TRUE'])).both).toBe(0)
  console.log('quoted sheet',JSON.stringify(splitSheetRange("'O''Brien'!A1:B2")))
  expect(splitSheetRange("'O''Brien'!A1:B2")?.sheetName).toBe("O''Brien")
})
