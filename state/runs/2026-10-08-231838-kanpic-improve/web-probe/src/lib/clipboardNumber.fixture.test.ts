import {describe,expect,it} from 'vitest'
import {readFileSync} from 'node:fs'
import {materializePaste} from './clipboard'
import {parseClipboardHtml} from './clipboardHtml'

// 밖에서 온 글자를 칸에 수로 담는 규칙은 문이 둘이다 — 파일(업로드
// 가져오기·IMPORTDATA)과 클립보드(평문·HTML 붙여넣기). 같은 표를 CSV 로
// 올리느냐 복사해 붙이느냐에 따라 우편번호의 앞자리 0 이 사라지거나
// 계좌번호의 뒷자리가 뭉개지면, 사람은 자기가 옮긴 표와 다른 것을 보게 되고
// 되돌릴 길이 없다.
//
// testdata/incoming-number.json 을 internal/importexport 의
// incoming_number_fixture_test.go 와 함께 읽는다 — 한쪽 문만 고치면 양쪽 다
// 걸린다. 여기서는 붙여넣기가 실제로 지나는 배선을 통과시킨다.
type IncomingNumberCase={text:string;number:boolean;value?:number}

const fixture=(JSON.parse(readFileSync('../testdata/incoming-number.json','utf8')) as {cases:IncomingNumberCase[]}).cases

function expected(item:IncomingNumberCase){return item.number?item.value:item.text}

describe('the clipboard door stores what the file door stores',()=>{
  it('reads every case in testdata/incoming-number.json', ()=>{
    expect(fixture.length).toBeGreaterThan(25)
  })

  it('agrees on the plain text paste path',()=>{
    const cells=materializePaste(fixture.map(item=>item.text).join('\n'),undefined,1,1)
    const wrong:string[]=[]
    fixture.forEach((item,index)=>{
      const cell=cells.find(candidate=>candidate.row===index+1&&candidate.column===1)
      if(cell?.value!==expected(item))wrong.push(`${JSON.stringify(item.text)} -> ${JSON.stringify(cell?.value)}, 파일 문은 ${JSON.stringify(expected(item))}`)
    })
    expect(wrong.join('\n')).toBe('')
  })

  it('agrees on the HTML paste path',()=>{
    const html='<table>'+fixture.map(item=>`<tr><td>${item.text}</td></tr>`).join('')+'</table>'
    const cells=parseClipboardHtml(html,10000)!
    const wrong:string[]=[]
    fixture.forEach((item,index)=>{
      const cell=cells.find(candidate=>candidate.rowOffset===index&&candidate.columnOffset===0)
      if(cell?.value!==expected(item))wrong.push(`${JSON.stringify(item.text)} -> ${JSON.stringify(cell?.value)}, 파일 문은 ${JSON.stringify(expected(item))}`)
    })
    expect(wrong.join('\n')).toBe('')
  })
})
