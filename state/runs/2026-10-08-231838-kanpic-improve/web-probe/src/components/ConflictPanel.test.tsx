import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { CellConflict, CellConflictResolutionResult, MutationResult, Sheet } from '../types'
import { ConflictPanel } from './ConflictPanel'

const sheet:Sheet={id:'sheet-1',workbook_id:'book-1',name:'매출',position:0,hidden:false,layout:{revision:1,frozen_rows:0,frozen_columns:0},created_at:'2026-08-02T00:00:00Z'}
const conflict:CellConflict={id:'conflict-1',workbook_id:'book-1',sheet_id:'sheet-1',operation_id:'operation-1',row:2,column:3,base_version:1,changed_at_version:2,server_version:3,actor_id:'bob',client_id:'browser-b',conflicting_actor_id:'alice',base_cell:{value:'original'},conflicting_cell:{value:'first',style:{bold:true}},submitted_cell:{value:'second',style:{italic:true}},applied_cell:{value:'second',style:{italic:true}},current_cell:{value:'second',style:{italic:true}},status:'open',revision:1,created_at:'2026-08-02T00:01:00Z',updated_at:'2026-08-02T00:01:00Z'}

afterEach(()=>{cleanup();vi.restoreAllMocks();vi.unstubAllGlobals()})

function renderPanel(props:Partial<React.ComponentProps<typeof ConflictPanel>>={}){
  const client=new QueryClient({defaultOptions:{queries:{retry:false,gcTime:0}}})
  const defaults={workbookId:'book-1',sheets:[sheet],currentActor:'bob',onClose:vi.fn(),onNavigate:vi.fn(()=>true),onResolved:vi.fn()}
  return{...render(<QueryClientProvider client={client}><ConflictPanel {...defaults} {...props}/></QueryClientProvider>),defaults}
}

describe('ConflictPanel',()=>{
  it('compares the conflict timeline, navigates, and restores the prior server value',async()=>{
    const operation:MutationResult={operation_id:'resolution-1',workbook_id:'book-1',sheet_id:'sheet-1',base_version:3,server_version:4,applied_cells:1,recalculated_cells:[],formula_errors:[],validation_warnings:[],duplicate:false,conflicts:[]}
    const resolved:CellConflictResolutionResult={conflict:{...conflict,status:'resolved',resolution:'restore_previous',revision:2,current_cell:conflict.conflicting_cell,resolution_operation_id:'resolution-1',resolution_server_version:4,resolved_by:'bob'},operation}
    let open=true
    const fetchMock=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
      const path=String(input)
      if(path==='/api/v1/conflicts/conflict-1:resolve'&&init?.method==='POST'){
        const body=JSON.parse(String(init.body)) as Record<string,unknown>
        expect(body).toMatchObject({client_id:expect.any(String),expected_revision:1,resolution:'restore_previous'})
        expect(body.idempotency_key).toBeTruthy()
        open=false
        return new Response(JSON.stringify(resolved),{status:200,headers:{'Content-Type':'application/json'}})
      }
      return new Response(JSON.stringify({items:open?[conflict]:[]}),{status:200,headers:{'Content-Type':'application/json'}})
    })
    vi.stubGlobal('fetch',fetchMock)
    const navigated=vi.fn(()=>true),onResolved=vi.fn()
    renderPanel({onNavigate:navigated,onResolved})
    expect(await screen.findByText('충돌 전 기준')).toBeInTheDocument()
    expect(screen.getByText('original')).toBeInTheDocument()
    expect(screen.getByText('first')).toBeInTheDocument()
    expect(screen.getByText('second')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button',{name:/매출!C2/}))
    expect(navigated).toHaveBeenCalledWith('sheet-1','C2')
    fireEvent.click(screen.getByRole('button',{name:/먼저 반영된 값 복원/}))
    await waitFor(()=>expect(onResolved).toHaveBeenCalledWith(resolved))
    expect(await screen.findByText('열린 충돌이 없습니다.')).toBeInTheDocument()
  })

  it('blocks restoration when the current cell changed after the conflict',async()=>{
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response(JSON.stringify({items:[{...conflict,current_cell:{value:'third'}}]}),{status:200,headers:{'Content-Type':'application/json'}})))
    renderPanel()
    expect(await screen.findByText(/충돌 뒤 값이 다시 변경되어 복원할 수 없습니다/)).toBeInTheDocument()
    expect(screen.getByRole('button',{name:/먼저 반영된 값 복원/})).toBeDisabled()
    expect(screen.getByRole('button',{name:/현재 값 유지/})).toBeEnabled()
  })
})
