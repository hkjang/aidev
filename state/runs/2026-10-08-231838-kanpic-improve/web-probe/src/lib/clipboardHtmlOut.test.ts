import { describe, expect, it } from 'vitest'
import type { KanpicClipboard } from './clipboard'
import { clipboardHtml } from './clipboardHtmlOut'
import { parseClipboardHtml } from './clipboardHtml'

const payload=(cells:KanpicClipboard['cells'],rows=1,columns=1):KanpicClipboard=>(
  {version:1,sourceRow:1,sourceColumn:1,rows,columns,cells})

describe('clipboardHtml',()=>{
  it('carries the formatting a plain-text copy loses',()=>{
    const html=clipboardHtml(payload([
      {rowOffset:0,columnOffset:0,value:'제품',style:{bold:true,background:'#DBEAFE',horizontal_align:'center'}},
      {rowOffset:0,columnOffset:1,value:1234.5,style:{color:'#B91C1C',number_format:'#,##0.00'}},
    ],1,2))
    expect(html).toContain('font-weight:700')
    expect(html).toContain('background-color:#DBEAFE')
    expect(html).toContain('text-align:center')
    expect(html).toContain('color:#B91C1C')
    // 엑셀은 붙여넣을 때 이 표시 형식을 셀에 적용한다.
    expect(html).toContain("mso-number-format:'#,##0.00'")
    // 보이던 글자와 함께 실제 숫자도 넘긴다.
    expect(html).toContain('x:num="1234.5"')
    expect(html).toContain('1,234.50')
  })

  it('escapes text that would otherwise become markup',()=>{
    const html=clipboardHtml(payload([{rowOffset:0,columnOffset:0,value:'<b>주의</b> & "인용"'}]))
    expect(html).toContain('&lt;b&gt;주의&lt;/b&gt; &amp; &quot;인용&quot;')
    expect(html).not.toContain('<b>주의')
  })

  it('leaves a format Excel could not read out of the declaration list',()=>{
    // 세미콜론이 들어간 조건부 표시 형식은 CSS 선언을 끊어 버린다.
    const html=clipboardHtml(payload([{rowOffset:0,columnOffset:0,value:5,style:{number_format:'#,##0;[Red]-#,##0'}}]))
    expect(html).not.toContain('mso-number-format')
    expect(html).toContain('x:num="5"')
  })

  it('survives its own round trip through the reader',()=>{
    const html=clipboardHtml(payload([
      {rowOffset:0,columnOffset:0,value:'합계',style:{bold:true,italic:true,horizontal_align:'right'}},
      {rowOffset:0,columnOffset:1,value:98000,style:{number_format:'#,##0"원"'}},
    ],1,2))
    const read=parseClipboardHtml(html,100)!
    expect(read[0]).toMatchObject({value:'합계',style:{bold:true,italic:true,horizontal_align:'right'}})
    expect(read[1]).toMatchObject({value:98000})
  })
})

describe('formatting that has to survive the trip to Excel', () => {
  const cell=(style:Record<string,unknown>)=>clipboardHtml({version:1,sourceRow:1,sourceColumn:1,rows:1,columns:1,
    cells:[{rowOffset:0,columnOffset:0,value:'값',style}]})

  // 이 앱은 취소선을 어디서나 `strike` 라 부른다. 쓰는 쪽만 다른 이름을 보면
  // 그은 줄이 조용히 사라진다.
  it('writes the strikethrough under the name the app stores it as', () => {
    expect(cell({strike:true})).toContain('line-through')
    expect(cell({underline:true,strike:true})).toContain('text-decoration:underline line-through')
  })

  it('writes a Korean font name instead of dropping it', () => {
    expect(cell({font_family:'맑은 고딕'})).toContain('font-family:맑은 고딕')
    expect(cell({font_family:'Arial'})).toContain('font-family:Arial')
  })

  it('drops a font name that would break out of the declaration', () => {
    expect(cell({font_family:'evil;color:red'})).not.toContain('font-family')
  })
})
