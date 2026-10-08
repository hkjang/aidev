import { describe, expect, it } from 'vitest'
import { cellLink, safeLinkTarget, workbookRangeLink, workbookRangeTarget } from './hyperlink'

const cell=(patch:Record<string,unknown>)=>({sheet_id:'s',row:1,column:1,...patch}) as never

describe('cellLink',()=>{
  it('reads the target and label out of a HYPERLINK formula',()=>{
    expect(cellLink(cell({formula:'=HYPERLINK("https://example.com","예시")',value:'예시'})))
      .toMatchObject({href:'https://example.com',label:'예시'})
  })

  it('falls back to the calculated value when the formula has no label',()=>{
    expect(cellLink(cell({formula:'=HYPERLINK("https://example.com")',value:'https://example.com'})))
      .toMatchObject({href:'https://example.com',label:'https://example.com'})
  })

  it('unescapes doubled quotes the way the formula language writes them',()=>{
    expect(cellLink(cell({formula:'=HYPERLINK("https://example.com","말 ""인용"" 표")'}))?.label).toBe('말 "인용" 표')
  })

  it('treats a pasted URL as a link',()=>{
    expect(cellLink(cell({value:'https://example.com/report'}))).toMatchObject({href:'https://example.com/report'})
    expect(cellLink(cell({value:'그냥 텍스트'}))).toBeUndefined()
  })

  // Cell text arrives from imports and other people's workbooks, so a script
  // target is a real possibility rather than a hypothetical one.
  it('refuses a target whose scheme is not one we open',()=>{
    expect(safeLinkTarget('javascript:alert(1)')).toBeUndefined()
    expect(safeLinkTarget('data:text/html,<script>')).toBeUndefined()
    expect(cellLink(cell({formula:'=HYPERLINK("javascript:alert(1)","눌러보세요")'}))).toBeUndefined()
    expect(safeLinkTarget('https://example.com')).toBe('https://example.com')
    expect(safeLinkTarget('/workbooks/abc?range=A1')).toBe('/workbooks/abc?range=A1')
  })

  it('leaves a nested HYPERLINK alone rather than guessing which branch wins',()=>{
    expect(cellLink(cell({formula:'=IF(A1,HYPERLINK("https://a.example"),HYPERLINK("https://b.example"))'}))).toBeUndefined()
  })
})

describe('workbookRangeTarget',()=>{
  it('reads the workbook, sheet and range out of an internal link',()=>{
    const href=workbookRangeLink('wb-1','sheet-1','B2:C5')
    expect(workbookRangeTarget(href)).toEqual({workbookId:'wb-1',sheetId:'sheet-1',range:'B2:C5'})
  })

  it('ignores links to somewhere else',()=>{
    expect(workbookRangeTarget('https://elsewhere.example/workbooks/wb-1?range=A1')).toBeUndefined()
    expect(workbookRangeTarget('/settings')).toBeUndefined()
  })
})

describe('chip labels',()=>{
  it('names an internal range link by its range instead of its URL',()=>{
    const formula=`=HYPERLINK("${workbookRangeLink('wb-1','sheet-1','B2:C5')}","요약으로")`
    expect(cellLink(cell({formula,value:'요약으로'}))?.linkLabel).toBe('이 워크북 · B2:C5')
  })

  it('shows the address itself for an outside link',()=>{
    expect(cellLink(cell({formula:'=HYPERLINK("https://example.com","예시")'}))?.linkLabel).toBe('https://example.com')
  })
})
