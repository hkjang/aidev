import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { WorkbookSearchResult } from '../types'
import { WorkbookSearchDialog } from './WorkbookSearchDialog'

afterEach(()=>{cleanup();vi.restoreAllMocks();vi.unstubAllGlobals()})

describe('WorkbookSearchDialog',()=>{
  it('searches the server and navigates to the selected cell',async()=>{
    const response:WorkbookSearchResult={workbook_id:'book-1',workbook_version:7,query:'매출',items:[{sheet_id:'sheet-2',sheet_name:'분기 실적',address:'C120',row:120,column:3,value:'매출 합계',formula:'=SUM(C2:C119)',matched_fields:['value']}],next_offset:50}
    const fetchMock=vi.fn().mockResolvedValue(new Response(JSON.stringify(response),{status:200,headers:{'Content-Type':'application/json'}}))
    vi.stubGlobal('fetch',fetchMock)
    const close=vi.fn(),navigate=vi.fn()
    render(<WorkbookSearchDialog open workbookId="book-1" version={7} onClose={close} onNavigate={navigate}/>)
    fireEvent.change(screen.getByRole('textbox',{name:'검색어'}),{target:{value:'매출'}})
    const option=await screen.findByRole('option',{name:/분기 실적/},{timeout:2000})
    expect(option).toHaveTextContent('C120')
    expect(screen.getByText('워크북 v7')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/workbooks/book-1/search?q=%EB%A7%A4%EC%B6%9C&limit=50',expect.objectContaining({signal:expect.any(AbortSignal)}))
    fireEvent.click(option)
    expect(navigate).toHaveBeenCalledWith(response.items[0])
    expect(close).toHaveBeenCalled()
  })

  it('opens a keyboard-selected result with Enter',async()=>{
    const response:WorkbookSearchResult={workbook_id:'book-1',workbook_version:8,query:'sum',items:[{sheet_id:'sheet-1',sheet_name:'Sheet1',address:'A1',row:1,column:1,value:3,formula:'=SUM(A2:A3)',matched_fields:['formula']}]}
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response(JSON.stringify(response),{status:200,headers:{'Content-Type':'application/json'}})))
    const navigate=vi.fn()
    render(<WorkbookSearchDialog open workbookId="book-1" version={8} onClose={()=>{}} onNavigate={navigate}/>)
    const input=screen.getByRole('textbox',{name:'검색어'})
    fireEvent.change(input,{target:{value:'sum'}})
    await screen.findByRole('option',{name:/Sheet1/},{timeout:2000})
    fireEvent.keyDown(input,{key:'Enter'})
    await waitFor(()=>expect(navigate).toHaveBeenCalledWith(response.items[0]))
  })
})

describe('WorkbookSearchDialog 찾기 및 바꾸기',()=>{
  const match={sheet_id:'sheet-1',sheet_name:'Sheet1',address:'B2',row:2,column:2,value:'구 사명',matched_fields:['value'] as Array<'value'|'formula'>}
  const searchResponse={workbook_id:'book-1',workbook_version:9,query:'구 사명',items:[match]}

  function stubFetch(replaceResponses:unknown[]){
    const calls:Array<{url:string;init?:RequestInit}>=[]
    const fetchMock=vi.fn(async(url:string,init?:RequestInit)=>{
      calls.push({url,init})
      const payload=url.includes('search:replace')?replaceResponses.shift():searchResponse
      return new Response(JSON.stringify(payload),{status:200,headers:{'Content-Type':'application/json'}})
    })
    vi.stubGlobal('fetch',fetchMock)
    return calls
  }

  it('sends the selected search options to the server',async()=>{
    const calls=stubFetch([])
    render(<WorkbookSearchDialog open workbookId="book-1" version={9} sheetId="sheet-1" sheetName="Sheet1" onClose={()=>{}} onNavigate={()=>{}}/>)
    fireEvent.change(screen.getByRole('textbox',{name:'검색어'}),{target:{value:'구 사명'}})
    await screen.findByRole('option',{name:/Sheet1/},{timeout:2000})
    fireEvent.click(screen.getByRole('button',{name:'대소문자 구분'}))
    fireEvent.click(screen.getByRole('button',{name:'Sheet1 시트만 검색'}))
    await waitFor(()=>expect(calls.at(-1)?.url).toContain('match_case=true'),{timeout:2000})
    expect(calls.at(-1)?.url).toContain('sheet_id=sheet-1')
  })

  it('previews then applies a replacement and reports the count',async()=>{
    const preview={workbook_id:'book-1',workbook_version:9,query:'구 사명',replacement:'새 사명',preview:true,matched_cells:3,planned_cells:3,replaced_cells:0,skipped_cells:0,items:[],sheets:[],server_version:9}
    const applied={...preview,preview:false,replaced_cells:3,server_version:10}
    const calls=stubFetch([preview,applied])
    vi.stubGlobal('confirm',vi.fn().mockReturnValue(true))
    const replaced=vi.fn()
    render(<WorkbookSearchDialog open replaceMode workbookId="book-1" version={9} sheetId="sheet-1" onClose={()=>{}} onNavigate={()=>{}} onReplaced={replaced}/>)
    fireEvent.change(screen.getByRole('textbox',{name:'검색어'}),{target:{value:'구 사명'}})
    fireEvent.change(screen.getByRole('textbox',{name:'바꿀 내용'}),{target:{value:'새 사명'}})
    fireEvent.click(screen.getByRole('button',{name:'모두 바꾸기'}))
    await waitFor(()=>expect(replaced).toHaveBeenCalledWith(applied),{timeout:2000})
    expect(screen.getByRole('status')).toHaveTextContent('3개 셀을 바꿨습니다.')
    const replaceCalls=calls.filter(call=>call.url.includes('search:replace'))
    expect(JSON.parse(String(replaceCalls[0].init?.body))).toMatchObject({query:'구 사명',replacement:'새 사명',preview:true})
    expect(JSON.parse(String(replaceCalls[1].init?.body))).toMatchObject({idempotency_key:expect.any(String)})
  })

  it('replaces only the highlighted cell when 바꾸기 is used',async()=>{
    const single={workbook_id:'book-1',workbook_version:9,query:'구 사명',replacement:'새 사명',preview:true,matched_cells:1,planned_cells:1,replaced_cells:0,skipped_cells:0,items:[],sheets:[],server_version:9}
    const calls=stubFetch([single,{...single,preview:false,replaced_cells:1,server_version:10}])
    render(<WorkbookSearchDialog open replaceMode workbookId="book-1" version={9} sheetId="sheet-1" onClose={()=>{}} onNavigate={()=>{}}/>)
    fireEvent.change(screen.getByRole('textbox',{name:'검색어'}),{target:{value:'구 사명'}})
    await screen.findByRole('option',{name:/Sheet1/},{timeout:2000})
    fireEvent.click(screen.getByRole('button',{name:'바꾸기'}))
    await waitFor(()=>expect(calls.filter(call=>call.url.includes('search:replace')).length).toBe(2),{timeout:2000})
    expect(JSON.parse(String(calls.filter(call=>call.url.includes('search:replace'))[0].init?.body))).toMatchObject({sheet_id:'sheet-1',range:'B2'})
  })
})
