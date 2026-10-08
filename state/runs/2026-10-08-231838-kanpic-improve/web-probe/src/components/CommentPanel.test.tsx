import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { CommentThread } from '../types'
import { CommentPanel } from './CommentPanel'

afterEach(()=>{cleanup();vi.restoreAllMocks();vi.unstubAllGlobals()})

function renderPanel(props:Partial<React.ComponentProps<typeof CommentPanel>>={}){
  const client=new QueryClient({defaultOptions:{queries:{retry:false,gcTime:0}}})
  const defaults={workbookId:'book-1',sheetId:'sheet-1',selectionRange:'B2:C3',currentActor:'alice',onNavigate:vi.fn(()=>true),onClose:vi.fn()}
  return{...render(<QueryClientProvider client={client}><CommentPanel {...defaults} {...props}/></QueryClientProvider>),client,defaults}
}

describe('CommentPanel',()=>{
  it('renders comments as text and navigates to their anchored range',async()=>{
    const thread:CommentThread={id:'thread-1',workbook_id:'book-1',sheet_id:'sheet-1',sheet_name:'Sheet1',range:'C7:D8',resolved:false,revision:1,created_by:'alice',created_at:'2026-08-01T10:00:00Z',updated_at:'2026-08-01T10:00:00Z',messages:[{id:'message-1',thread_id:'thread-1',author_id:'alice',content:'<img src=x onerror=alert(1)> @bob',mentions:['bob'],revision:1,created_at:'2026-08-01T10:00:00Z',updated_at:'2026-08-01T10:00:00Z'}]}
    vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response(JSON.stringify({items:[thread]}),{status:200,headers:{'Content-Type':'application/json'}})))
    const navigate=vi.fn(()=>true)
    renderPanel({onNavigate:navigate,focusThreadId:'thread-1'})
    expect(await screen.findByText('<img src=x onerror=alert(1)> @bob')).toBeInTheDocument()
    expect(document.querySelector('img')).toBeNull()
    fireEvent.click(screen.getByRole('button',{name:/Sheet1 · C7:D8/}))
    expect(navigate).toHaveBeenCalledWith('sheet-1','C7:D8')
    expect(screen.getByText('@bob')).toBeInTheDocument()
  })

  it('creates a range comment with an idempotency key and refreshes the list',async()=>{
    let items:CommentThread[]=[]
    const fetchMock=vi.fn(async(input:RequestInfo|URL,init?:RequestInit)=>{
      const path=String(input)
      if(path==='/api/v1/workbooks/book-1/comments'&&init?.method==='POST'){
        const body=JSON.parse(String(init.body)) as {sheet_id:string;range:string;content:string;idempotency_key:string}
        expect(body).toEqual(expect.objectContaining({sheet_id:'sheet-1',range:'B2:C3',content:'검토 부탁드립니다 @bob'}))
        expect(body.idempotency_key).toBeTruthy()
        const now='2026-08-01T10:00:00Z'
        const created:CommentThread={id:'thread-2',workbook_id:'book-1',sheet_id:'sheet-1',sheet_name:'Sheet1',range:body.range,resolved:false,revision:1,created_by:'alice',created_at:now,updated_at:now,messages:[{id:'message-2',thread_id:'thread-2',author_id:'alice',content:body.content,mentions:['bob'],revision:1,created_at:now,updated_at:now}]}
        items=[created]
        return new Response(JSON.stringify(created),{status:201,headers:{'Content-Type':'application/json'}})
      }
      return new Response(JSON.stringify({items}),{status:200,headers:{'Content-Type':'application/json'}})
    })
    vi.stubGlobal('fetch',fetchMock)
    renderPanel()
    await screen.findByText('표시할 댓글이 없습니다.')
    fireEvent.change(screen.getByRole('textbox',{name:'새 댓글 내용'}),{target:{value:'검토 부탁드립니다 @bob'}})
    fireEvent.click(screen.getByRole('button',{name:'등록'}))
    expect(await screen.findByText('검토 부탁드립니다 @bob')).toBeInTheDocument()
    await waitFor(()=>expect(fetchMock).toHaveBeenCalledWith('/api/v1/workbooks/book-1/comments',expect.objectContaining({method:'POST'})))
  })
})
