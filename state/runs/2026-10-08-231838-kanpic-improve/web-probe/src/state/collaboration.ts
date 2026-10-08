import { create } from 'zustand'
import { collaborationClientId } from '../lib/client'

export type Coordinate = { row:number; column:number }
export type Cursor = Coordinate & { sheet_id:string }
export type Selection = { sheet_id:string; start:Coordinate; end:Coordinate }
export type Presence = { actor_id:string; client_id:string; cursor?:Cursor; selection?:Selection }
type ConnectionState = 'connecting'|'connected'|'reconnecting'|'offline'
export type ServerEvent = { type:string; workbook_id:string; actor_id?:string; client_id?:string; server_version?:number; data?:unknown }

type CollaborationState = {
  status:ConnectionState
  users:Record<string,Presence>
  connect:(workbookId:string,onVersion:(version:number,event:ServerEvent)=>void,onEvent?:(event:ServerEvent)=>void)=>void
  disconnect:()=>void
  sendCursor:(cursor:Cursor)=>void
  sendSelection:(selection:Selection)=>void
}

let socket:WebSocket|undefined
let retryTimer:number|undefined
let generation=0
let retryCount=0

function eventData<T>(event:ServerEvent) { return event.data as T }
function eventId() { return crypto.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}` }

export const useCollaborationStore=create<CollaborationState>((set,get)=>({
  status:'offline',users:{},
  connect:(workbookId,onVersion,onEvent)=>{
    generation+=1
    const currentGeneration=generation
    retryCount=0
    if(retryTimer)window.clearTimeout(retryTimer)
    socket?.close(1000,'switching workbook')
    set({status:'connecting',users:{}})
    const open=()=>{
      if(currentGeneration!==generation)return
      const scheme=window.location.protocol==='https:'?'wss':'ws'
      const url=`${scheme}://${window.location.host}/ws/v1/workbooks/${encodeURIComponent(workbookId)}?client_id=${encodeURIComponent(collaborationClientId())}`
      const next=new WebSocket(url)
      socket=next
      next.onopen=()=>{if(currentGeneration===generation){retryCount=0;set({status:'connected'})}}
      next.onmessage=(message)=>{
        if(currentGeneration!==generation)return
        let event:ServerEvent
        try{event=JSON.parse(String(message.data)) as ServerEvent}catch{return}
        if(event.workbook_id!==workbookId)return
        onEvent?.(event)
        if(event.server_version)onVersion(event.server_version,event)
        if(event.type==='presence.snapshot'){
          const users=eventData<{users:Presence[]}>(event)?.users??[]
          set({users:Object.fromEntries(users.map(user=>[user.client_id,user]))})
        }else if(event.type==='presence.join'){
          const user=eventData<Presence>(event);if(user?.client_id)set(state=>({users:{...state.users,[user.client_id]:user}}))
        }else if(event.type==='presence.leave'){
          const clientId=event.client_id??eventData<Presence>(event)?.client_id
          if(clientId)set(state=>{const users={...state.users};delete users[clientId];return{users}})
        }else if(event.type==='cursor.update'&&event.client_id){
          const cursor=eventData<Cursor>(event)
          set(state=>({users:{...state.users,[event.client_id!]:{...(state.users[event.client_id!]??{actor_id:event.actor_id??'사용자',client_id:event.client_id!}),cursor}}}))
        }else if(event.type==='selection.update'&&event.client_id){
          const selection=eventData<Selection>(event)
          set(state=>({users:{...state.users,[event.client_id!]:{...(state.users[event.client_id!]??{actor_id:event.actor_id??'사용자',client_id:event.client_id!}),selection}}}))
        }
      }
      next.onclose=()=>{
        if(currentGeneration!==generation)return
        socket=undefined
        set({status:navigator.onLine?'reconnecting':'offline'})
        const delay=Math.min(10_000,500*2**Math.min(retryCount++,5))
        retryTimer=window.setTimeout(open,delay)
      }
      next.onerror=()=>next.close()
    }
    open()
  },
  disconnect:()=>{
    generation+=1
    if(retryTimer)window.clearTimeout(retryTimer)
    retryTimer=undefined
    socket?.close(1000,'editor closed')
    socket=undefined
    set({status:'offline',users:{}})
  },
  sendCursor:(cursor)=>{
    if(socket?.readyState===WebSocket.OPEN)socket.send(JSON.stringify({id:eventId(),type:'cursor.update',data:cursor}))
  },
  sendSelection:(selection)=>{
    if(socket?.readyState===WebSocket.OPEN)socket.send(JSON.stringify({id:eventId(),type:'selection.update',data:selection}))
  },
}))

export function presenceColor(value:string){
  const palette=['#2563eb','#9333ea','#db2777','#ea580c','#0891b2','#65a30d','#7c3aed','#dc2626']
  let hash=0
  for(let index=0;index<value.length;index++)hash=(hash*31+value.charCodeAt(index))|0
  return palette[Math.abs(hash)%palette.length]
}
