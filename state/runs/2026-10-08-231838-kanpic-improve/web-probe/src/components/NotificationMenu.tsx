import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Bell, Check, MapPin } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import { useUserDirectory, userInitial, userLabel, userTooltip } from '../state/directory'
import type { MentionNotification, Session } from '../types'

function formatTime(value:string){return new Intl.DateTimeFormat('ko-KR',{month:'short',day:'numeric',hour:'2-digit',minute:'2-digit'}).format(new Date(value))}

export function NotificationMenu({session}:{session?:Session}){
  const [open,setOpen]=useState(false),root=useRef<HTMLDivElement>(null),client=useQueryClient()
  const notifications=useQuery({queryKey:['mention-notifications',session?.user?.id??'local-user'],queryFn:()=>api<{items:MentionNotification[]}>('/api/v1/me/notifications?unread_only=true&limit=50'),refetchInterval:30_000})
  useEffect(()=>{const close=(event:MouseEvent)=>{if(!root.current?.contains(event.target as Node))setOpen(false)};document.addEventListener('mousedown',close);return()=>document.removeEventListener('mousedown',close)},[])
  const items=notifications.data?.items??[]
  const directory=useUserDirectory(items.map(item=>item.actor_id))
  const openNotification=async(item:MentionNotification)=>{try{await api(`/api/v1/me/notifications/${item.id}`,{method:'PATCH',body:'{}'});await client.invalidateQueries({queryKey:['mention-notifications']})}finally{const query=new URLSearchParams({sheet_id:item.sheet_id,comment_id:item.thread_id});if(item.range!=='#REF!')query.set('range',item.range);window.location.href=`/workbooks/${item.workbook_id}?${query.toString()}`}}
  return <div className="notification-menu" ref={root}>
    <button className="icon-button" aria-label={`알림${items.length?` ${items.length}개`:''}`} aria-expanded={open} onClick={()=>setOpen(value=>!value)}><Bell size={19}/>{items.length>0&&<b>{items.length>99?'99+':items.length}</b>}</button>
    {open&&<div className="notification-popover" role="dialog" aria-label="멘션 알림">
      <header><div><strong>알림</strong><small>@멘션된 댓글</small></div><span>{items.length}개 안 읽음</span></header>
      <div className="notification-list">{notifications.isLoading?<p>알림을 불러오는 중…</p>:items.length===0?<div className="notification-empty"><Check/><strong>모두 확인했습니다.</strong><span>새로운 멘션이 없습니다.</span></div>:items.map(item=><button key={item.id} onClick={()=>void openNotification(item)}><span className="notification-avatar" title={userTooltip(item.actor_id,directory)}>{userInitial(item.actor_id,directory)}</span><span><strong title={userTooltip(item.actor_id,directory)}>{userLabel(item.actor_id,directory)}</strong><em>님이 댓글에서 나를 멘션했습니다.</em><small><MapPin/>{item.sheet_name} · {item.range} · {formatTime(item.created_at)}</small></span></button>)}</div>
    </div>}
  </div>
}
