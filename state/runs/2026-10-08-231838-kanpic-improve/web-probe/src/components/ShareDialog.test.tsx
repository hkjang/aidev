import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ShareDialog } from './ShareDialog'
import type { Department, SharingResponse, Workbook, WorkbookAccess } from '../types'

afterEach(()=>{cleanup();vi.restoreAllMocks();vi.unstubAllGlobals()})

const workbook={id:'book-1',workspace_id:'default',title:'2026 예산',owner_id:'owner',favorite:false,version:3,created_at:'',updated_at:'',sheets:[]} as unknown as Workbook
const departments:Department[]=[{id:'dept-1',name:'재무팀',path:'경영지원본부 / 재무팀',depth:1,member_count:4,revision:1,created_at:'',updated_at:''}]

function sharingResponse(overrides:Partial<WorkbookAccess>={},shares:SharingResponse['sharing']['shares']=[]):SharingResponse{
  return{
    sharing:{workbook_id:'book-1',owner_id:'owner',link_access:'restricted',link_role:'viewer',sharing_locked:false,viewer_can_copy:true,shares},
    access:{workbook_id:'book-1',actor_id:'owner',role:'owner',source:'owner',can_read:true,can_comment:true,can_write:true,can_manage:true,can_copy:true,link_access:'restricted',link_role:'viewer',owner_id:'owner',...overrides},
  }
}

function stubFetch(response:SharingResponse){
  const calls:Array<{url:string;init?:RequestInit}>=[]
  vi.stubGlobal('fetch',vi.fn(async(url:string,init?:RequestInit)=>{
    calls.push({url,init})
    const payload=url.includes('/departments')?{items:departments}
      :url.includes('/access-requests')?{items:[{id:'req-1',workbook_id:'book-1',requester_id:'minsu',requested_role:'editor',message:'검토 필요',status:'pending',created_at:''}]}
      :response
    return new Response(JSON.stringify(payload),{status:200,headers:{'Content-Type':'application/json'}})
  }))
  return calls
}

function renderDialog(response:SharingResponse){
  const client=new QueryClient({defaultOptions:{queries:{retry:false}}})
  const calls=stubFetch(response)
  render(<QueryClientProvider client={client}><ShareDialog workbook={workbook} onClose={()=>{}}/></QueryClientProvider>)
  return calls
}

describe('ShareDialog',()=>{
  it('grants a department the chosen role',async()=>{
    const calls=renderDialog(sharingResponse())
    await screen.findByText('액세스 권한이 있는 사용자')
    fireEvent.change(screen.getByRole('combobox',{name:'공유 대상 유형'}),{target:{value:'department'}})
    await waitFor(()=>expect(screen.getByRole('combobox',{name:'부서 선택'})).toBeInTheDocument())
    fireEvent.change(screen.getByRole('combobox',{name:'부서 선택'}),{target:{value:'dept-1'}})
    fireEvent.change(screen.getByRole('combobox',{name:'부여할 권한'}),{target:{value:'commenter'}})
    fireEvent.click(screen.getByRole('button',{name:/추가/}))
    await waitFor(()=>expect(calls.some(call=>call.init?.method==='PUT')).toBe(true))
    const put=calls.find(call=>call.init?.method==='PUT')
    expect(JSON.parse(String(put?.init?.body))).toMatchObject({principal_type:'department',principal_id:'dept-1',role:'commenter'})
  })

  it('changes the general link access and shows the matching hint',async()=>{
    const calls=renderDialog(sharingResponse())
    const scope=await screen.findByRole('combobox',{name:'일반 액세스 범위'})
    expect(screen.getByText('추가된 사용자·부서·역할만 열 수 있습니다.')).toBeInTheDocument()
    fireEvent.change(scope,{target:{value:'organization'}})
    await waitFor(()=>expect(calls.some(call=>call.init?.method==='PATCH')).toBe(true))
    expect(JSON.parse(String(calls.find(call=>call.init?.method==='PATCH')?.init?.body))).toEqual({link_access:'organization'})
  })

  it('approves a pending access request',async()=>{
    const calls=renderDialog(sharingResponse())
    await screen.findByText('대기 중인 액세스 요청 1건')
    fireEvent.click(screen.getByRole('button',{name:/승인/}))
    await waitFor(()=>expect(calls.some(call=>call.url.includes('/access-requests/req-1:approve'))).toBe(true))
  })

  it('hides owner controls from an editor and explains the effective role',async()=>{
    renderDialog(sharingResponse({role:'editor',source:'department',source_label:'재무팀',can_manage:false},
      [{id:'s-1',workbook_id:'book-1',principal_type:'user',principal_id:'alice',role:'editor',revision:1,created_at:'',updated_at:''}]))
    await screen.findByText('편집자 · 재무팀 공유')
    expect(screen.queryByRole('combobox',{name:'공유 대상 유형'})).toBeNull()
    expect(screen.queryByText('소유권 이전')).toBeNull()
    expect(screen.getByText('제한됨')).toBeInTheDocument()
  })
})
