import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Pivot, Sheet } from '../types'
import { PivotDialog } from './PivotDialog'

const sheet:Sheet={id:'sheet-1',workbook_id:'book-1',name:'Data',position:0,hidden:false,layout:{revision:1,frozen_rows:0,frozen_columns:0},created_at:'2026-08-01T00:00:00Z'}
const pivot:Pivot={id:'pivot-1',workbook_id:'book-1',workbook_version:3,sheet_id:sheet.id,source_sheet_id:sheet.id,name:'지역별 매출',source_range:'A1:C8',first_row_headers:true,rows:[{column:1,name:'지역',group:'none'}],columns:[],values:[{column:3,name:'매출',aggregation:'sum'}],filters:[],calculated_fields:[],refresh_mode:'manual',source_version:2,revision:1,created_by:'alice',updated_by:'alice',created_at:'2026-08-01T00:00:00Z',updated_at:'2026-08-01T00:00:00Z'}

function renderDialog(node:ReactNode){
  const client=new QueryClient({defaultOptions:{queries:{retry:false}}})
  return render(<QueryClientProvider client={client}>{node}</QueryClientProvider>)
}

afterEach(()=>{cleanup();vi.unstubAllGlobals()})

describe('PivotDialog',()=>{
  it('creates a managed pivot from the selected range',async()=>{
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response(JSON.stringify({items:[]}),{status:200,headers:{'Content-Type':'application/json'}})))
    const create=vi.fn().mockResolvedValue(pivot)
    renderDialog(<PivotDialog activeSheetId={sheet.id} selectionRange="A1:C8" sheets={[sheet]} onClose={vi.fn()} onCreate={create} onUpdate={vi.fn()} onDelete={vi.fn()}/>)
    fireEvent.change(screen.getByLabelText('피벗 이름'),{target:{value:'분기 요약'}})
    fireEvent.change(screen.getByLabelText('피벗 갱신 방식'),{target:{value:'manual'}})
    fireEvent.click(screen.getByText('피벗 저장'))
    await waitFor(()=>expect(create).toHaveBeenCalledTimes(1))
    expect(create.mock.calls[0][0]).toMatchObject({sheet_id:sheet.id,source_sheet_id:sheet.id,source_range:'A1:C8',name:'분기 요약',refresh_mode:'manual',first_row_headers:true,values:[{column:1,aggregation:'sum'}]})
  })

  it('sends the revision and configured dimensions when editing',async()=>{
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response(JSON.stringify({items:[]}),{status:200,headers:{'Content-Type':'application/json'}})))
    const update=vi.fn().mockResolvedValue({...pivot,name:'변경',revision:2})
    renderDialog(<PivotDialog pivot={pivot} activeSheetId={sheet.id} selectionRange="D1:E4" sheets={[sheet]} onClose={vi.fn()} onCreate={vi.fn()} onUpdate={update} onDelete={vi.fn()}/>)
    fireEvent.change(screen.getByLabelText('피벗 이름'),{target:{value:'변경'}})
    fireEvent.click(screen.getByText('피벗 저장'))
    await waitFor(()=>expect(update).toHaveBeenCalledTimes(1))
    expect(update.mock.calls[0][0]).toEqual(pivot)
    expect(update.mock.calls[0][1]).toMatchObject({name:'변경',expected_revision:1,rows:[{column:1,name:'지역',group:'none'}],values:[{column:3,name:'매출',aggregation:'sum'}]})
  })

  it('uses source headers for generated dimension and value names',async()=>{
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response(JSON.stringify({items:[{sheet_id:sheet.id,row:1,column:1,value:'지역',updated_at:''},{sheet_id:sheet.id,row:1,column:2,value:'매출',updated_at:''}]}),{status:200,headers:{'Content-Type':'application/json'}})))
    const create=vi.fn().mockResolvedValue(pivot)
    renderDialog(<PivotDialog activeSheetId={sheet.id} selectionRange="A1:B8" sheets={[sheet]} onClose={vi.fn()} onCreate={create} onUpdate={vi.fn()} onDelete={vi.fn()}/>)
    await screen.findByRole('option',{name:'지역 · 1열'})
    fireEvent.click(screen.getByLabelText('행 그룹 추가'))
    fireEvent.change(screen.getByLabelText('값 필드 1'),{target:{value:'2'}})
    fireEvent.click(screen.getByText('피벗 저장'))
    await waitFor(()=>expect(create).toHaveBeenCalledTimes(1))
    expect(create.mock.calls[0][0]).toMatchObject({rows:[{column:1,name:'지역'}],values:[{column:2,aggregation:'sum',name:'합계 매출'}]})
  })
})
