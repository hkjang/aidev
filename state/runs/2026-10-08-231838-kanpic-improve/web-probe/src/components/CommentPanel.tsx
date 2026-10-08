import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, CornerDownRight, MapPin, MessageSquare, Pencil, RotateCcw, Send, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { api, newIdempotencyKey } from '../lib/api'
import { useUserDirectory, userInitial, userLabel, userTooltip } from '../state/directory'
import type { CommentMessage, CommentThread } from '../types'
import './CommentPanel.css'

type Props = {
  workbookId:string
  sheetId:string
  selectionRange:string
  currentActor:string
  focusThreadId?:string
  onNavigate:(sheetId:string,range:string)=>boolean
  onClose:()=>void
}

function formatTime(value:string){return new Intl.DateTimeFormat('ko-KR',{month:'short',day:'numeric',hour:'2-digit',minute:'2-digit'}).format(new Date(value))}

export function CommentPanel({workbookId,sheetId,selectionRange,currentActor,focusThreadId,onNavigate,onClose}:Props){
  const client=useQueryClient()
  const [showResolved,setShowResolved]=useState(false),[content,setContent]=useState(''),[replyThread,setReplyThread]=useState<string>(),[reply,setReply]=useState(''),[editing,setEditing]=useState<{id:string;content:string}>(),[busy,setBusy]=useState(false),[error,setError]=useState('')
  const focused=useRef<HTMLDivElement>(null)
  const comments=useQuery({queryKey:['comments',workbookId,sheetId],queryFn:()=>api<{items:CommentThread[]}>(`/api/v1/workbooks/${workbookId}/comments?sheet_id=${encodeURIComponent(sheetId)}&include_resolved=true`)})
  const threads=useMemo(()=>{
    const items=(comments.data?.items??[]).filter(thread=>showResolved||!thread.resolved)
    if(!focusThreadId)return items
    return [...items].sort((a,b)=>a.id===focusThreadId?-1:b.id===focusThreadId?1:0)
  },[comments.data?.items,focusThreadId,showResolved])
  // Comment authors and mentions are shown by name, with the account in the
  // tooltip, so a long identifier never hides who wrote something.
  const directory=useUserDirectory(threads.flatMap(thread=>[thread.created_by,...thread.messages.flatMap(message=>[message.author_id,...message.mentions])]))
  useEffect(()=>{if(focusThreadId&&focused.current)focused.current.scrollIntoView?.({block:'nearest'})},[focusThreadId,threads.length])
  const refresh=()=>{client.invalidateQueries({queryKey:['comments',workbookId]});client.invalidateQueries({queryKey:['mention-notifications']})}
  const execute=async(action:()=>Promise<unknown>)=>{setBusy(true);setError('');try{await action();await refresh()}catch(value){setError(value instanceof Error?value.message:'댓글 작업을 완료하지 못했습니다.');await refresh()}finally{setBusy(false)}}
  const create=()=>execute(async()=>{await api<CommentThread>(`/api/v1/workbooks/${workbookId}/comments`,{method:'POST',body:JSON.stringify({sheet_id:sheetId,range:selectionRange,content,idempotency_key:newIdempotencyKey()})});setContent('')})
  const addReply=(thread:CommentThread)=>execute(async()=>{await api<CommentThread>(`/api/v1/comments/${thread.id}/replies`,{method:'POST',body:JSON.stringify({content:reply,idempotency_key:newIdempotencyKey()})});setReply('');setReplyThread(undefined)})
  const toggleResolved=(thread:CommentThread)=>execute(()=>api<CommentThread>(`/api/v1/comments/${thread.id}`,{method:'PATCH',body:JSON.stringify({resolved:!thread.resolved,expected_revision:thread.revision})}))
  const removeThread=(thread:CommentThread)=>{if(!confirm('이 댓글 스레드와 모든 답글을 삭제할까요?'))return;void execute(()=>api(`/api/v1/comments/${thread.id}`,{method:'DELETE'}))}
  const updateMessage=(message:CommentMessage)=>execute(async()=>{await api<CommentThread>(`/api/v1/comment-messages/${message.id}`,{method:'PATCH',body:JSON.stringify({content:editing?.content??'',expected_revision:message.revision})});setEditing(undefined)})
  const removeMessage=(message:CommentMessage)=>{if(!confirm('이 댓글을 삭제할까요?'))return;void execute(()=>api<CommentThread>(`/api/v1/comment-messages/${message.id}?expected_revision=${message.revision}`,{method:'DELETE'}))}
  return <aside className="comment-panel" aria-label="댓글 패널">
    <div className="comment-panel-head"><span><MessageSquare/> 댓글</span><button onClick={onClose} aria-label="댓글 닫기">×</button></div>
    <div className="comment-compose"><div><strong>새 댓글</strong><code>{selectionRange}</code></div><textarea aria-label="새 댓글 내용" value={content} onChange={event=>setContent(event.target.value)} placeholder="댓글을 입력하세요. @이메일로 멘션할 수 있습니다." maxLength={10000}/><button className="primary" disabled={busy||!content.trim()} onClick={()=>void create()}><Send/> 등록</button></div>
    <label className="comment-resolved-toggle"><input type="checkbox" checked={showResolved} onChange={event=>setShowResolved(event.target.checked)}/> 해결된 댓글 표시</label>
    {error&&<div className="comment-error" role="alert">{error}</div>}
    <div className="comment-list">
      {comments.isLoading&&<p className="comment-empty">댓글을 불러오는 중…</p>}
      {!comments.isLoading&&threads.length===0&&<div className="comment-empty"><MessageSquare/><strong>표시할 댓글이 없습니다.</strong><span>셀이나 범위를 선택해 첫 댓글을 남겨보세요.</span></div>}
      {threads.map(thread=><div key={thread.id} ref={thread.id===focusThreadId?focused:undefined} className={`comment-thread ${thread.resolved?'resolved':''} ${thread.id===focusThreadId?'focused':''}`}>
        <div className="comment-thread-head"><button className="comment-location" onClick={()=>onNavigate(thread.sheet_id,thread.range)} disabled={thread.range==='#REF!'}><MapPin/>{thread.sheet_name} · {thread.range}</button><span>{thread.resolved?'해결됨':`${thread.messages.filter(message=>!message.deleted_at).length}개`}</span></div>
        <div className="comment-messages">{thread.messages.map(message=><div className={`comment-message ${message.deleted_at?'deleted':''}`} key={message.id}>
          <div className="comment-message-meta"><span className="comment-avatar" title={userTooltip(message.author_id,directory)}>{userInitial(message.author_id,directory)}</span><strong title={userTooltip(message.author_id,directory)}>{userLabel(message.author_id,directory)}{message.author_id===currentActor?' (나)':''}</strong><time>{formatTime(message.updated_at)}</time></div>
          {message.deleted_at?<p>삭제된 댓글입니다.</p>:editing?.id===message.id?<div className="comment-edit"><textarea aria-label="댓글 수정 내용" value={editing.content} onChange={event=>setEditing({...editing,content:event.target.value})}/><div><button onClick={()=>setEditing(undefined)}>취소</button><button disabled={busy||!editing.content.trim()} onClick={()=>void updateMessage(message)}>저장</button></div></div>:<><p>{message.content}</p>{message.mentions.length>0&&<small className="comment-mentions">{message.mentions.map(value=>`@${userLabel(value,directory)}`).join(' ')}</small>}</>}
          {!message.deleted_at&&message.author_id===currentActor&&editing?.id!==message.id&&<div className="comment-message-actions"><button aria-label="댓글 수정" onClick={()=>setEditing({id:message.id,content:message.content})}><Pencil/></button><button aria-label="댓글 삭제" onClick={()=>removeMessage(message)}><Trash2/></button></div>}
        </div>)}</div>
        {replyThread===thread.id?<div className="comment-reply"><textarea autoFocus aria-label="답글 내용" value={reply} onChange={event=>setReply(event.target.value)} placeholder="답글 또는 @멘션"/><div><button onClick={()=>{setReplyThread(undefined);setReply('')}}>취소</button><button disabled={busy||!reply.trim()} onClick={()=>void addReply(thread)}><Send/> 답글</button></div></div>:<button className="comment-reply-trigger" onClick={()=>setReplyThread(thread.id)}><CornerDownRight/> 답글</button>}
        <div className="comment-thread-actions"><button onClick={()=>void toggleResolved(thread)}>{thread.resolved?<><RotateCcw/> 재열기</>:<><Check/> 해결</>}</button>{thread.created_by===currentActor&&<button className="danger" onClick={()=>removeThread(thread)}><Trash2/> 스레드 삭제</button>}</div>
      </div>)}
    </div>
  </aside>
}
