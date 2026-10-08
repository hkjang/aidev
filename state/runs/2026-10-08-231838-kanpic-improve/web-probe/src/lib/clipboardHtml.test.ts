import { describe, expect, it } from 'vitest'
import { parseClipboardHtml } from './clipboardHtml'

// 이 테스트는 표의 짜임새와 값 해석만 확인한다. 인라인 스타일을 실제로
// 읽어 내는지는 jsdom 으로 증명할 수 없다: 이 앱의 CSP 에는
// `style-src 'unsafe-inline'` 이 없어 크롬은 style 속성을 CSSOM 에 넣지
// 않는데 jsdom 에는 CSP 가 없다. 그쪽은 e2e/clipboard-paste.spec.ts 가 본다.
describe('parseClipboardHtml',()=>{
  it('ignores anything that is not a table',()=>{
    expect(parseClipboardHtml('<p>그냥 문단</p>',100)).toBeUndefined()
    expect(parseClipboardHtml('',100)).toBeUndefined()
  })

  it('prefers the value a cell declares over the text it shows',()=>{
    const cells=parseClipboardHtml('<table><tr>'
      +'<td x:num="1234.5">₩1,234.50</td>'
      +'<td data-sheets-value=\'{"1":3,"3":0.125}\'>12.5%</td>'
      +'<td x:str="">0012</td></tr></table>',100)!
    expect(cells.map(cell=>cell.value)).toEqual([1234.5,0.125,'0012'])
  })

  it('reads the number a plain cell was showing',()=>{
    const cells=parseClipboardHtml('<table><tr><td>1,234</td><td>연필</td><td>7</td></tr></table>',100)!
    expect(cells[0]).toMatchObject({value:1234,style:{number_format:'#,##0'}})
    expect(cells[1]).toMatchObject({value:'연필'})
    expect(cells[2]).toMatchObject({value:7})
    expect(cells[2].style).toBeUndefined()
  })

  it('keeps the columns lined up under a spanning cell',()=>{
    const cells=parseClipboardHtml('<table>'
      +'<tr><td rowspan="2">왼쪽</td><td colspan="2">두 칸</td></tr>'
      +'<tr><td>가운데</td><td>오른쪽</td></tr>'
      +'<tr><td>아래</td></tr></table>',100)!
    expect(cells.map(cell=>`${cell.rowOffset},${cell.columnOffset}=${String(cell.value)}`)).toEqual([
      '0,0=왼쪽','0,1=두 칸','1,1=가운데','1,2=오른쪽','2,0=아래',
    ])
  })

  it('stops at the cell budget instead of building an unbounded paste',()=>{
    const row='<tr>'+'<td>1</td>'.repeat(10)+'</tr>'
    const cells=parseClipboardHtml(`<table>${row.repeat(10)}</table>`,25)!
    expect(cells).toHaveLength(25)
  })
})

describe('formatting that Excel and Sheets actually send', () => {
  const style=(html:string)=>parseClipboardHtml(`<table><tr><td ${html}>값</td></tr></table>`,10)?.[0]?.style

  it('reads a strikethrough as the key the rest of the app uses', () => {
    expect(style('style="text-decoration:line-through"')?.strike).toBe(true)
    expect(style('style="text-decoration:underline line-through"')).toMatchObject({underline:true,strike:true})
  })

  it('reads a strikethrough written as a tag', () => {
    expect(parseClipboardHtml('<table><tr><td><s>값</s></td></tr></table>',10)?.[0]?.style?.strike).toBe(true)
  })

  it('leaves the strikethrough off when nothing struck the text', () => {
    expect(style('style="text-decoration:underline"')?.strike).toBeUndefined()
  })

  it('keeps the font family, including a Korean one', () => {
    expect(style('style="font-family:맑은 고딕"')?.font_family).toBe('맑은 고딕')
    expect(style(`style="font-family:'Malgun Gothic', sans-serif"`)?.font_family).toBe('Malgun Gothic')
    expect(style('face="Arial"')?.font_family).toBe('Arial')
  })

  // 읽어들인 이름은 다시 내보낼 수 있어야 한다. 두 쪽이 같은 자를 쓰는지 본다.
  it('drops a font family that could not be written back out', () => {
    expect(style('style="font-family:&lt;script&gt;"')?.font_family).toBeUndefined()
    expect(style('style="font-family:a(b)"')?.font_family).toBeUndefined()
  })

  // 서버는 6..72pt 만 받는다. 더 넉넉하게 읽으면 붙여넣기 전체가 400 으로 떨어진다.
  it('holds an oversized font to what the server accepts instead of failing the paste', () => {
    expect(style('style="font-size:80pt"')?.font_size).toBe(72)
    expect(style('style="font-size:14pt"')?.font_size).toBe(14)
    expect(style('style="font-size:4pt"')?.font_size).toBeUndefined()
  })
})
