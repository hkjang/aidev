import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { UserSummary } from '../types'

/** Shortens a raw identifier so a fallback label never fills the panel. */
export function shortenIdentifier(value:string){
  const trimmed=value.trim()
  if(!trimmed)return '알 수 없음'
  const local=trimmed.includes('@')?trimmed.slice(0,trimmed.indexOf('@')):trimmed
  if(local.length<=18)return local
  return `${local.slice(0,16)}…`
}

/** The label shown to people: display name first, then a readable identifier. */
export function userLabel(userID:string,directory?:Map<string,UserSummary>){
  const summary=directory?.get(userID.trim().toLowerCase())
  return summary?.display_name?.trim()||shortenIdentifier(userID)
}

/** The full identity for tooltips, so the exact account stays discoverable. */
export function userTooltip(userID:string,directory?:Map<string,UserSummary>){
  const summary=directory?.get(userID.trim().toLowerCase())
  const parts=[summary?.display_name?.trim(),summary?.email?.trim()||userID.trim()].filter(Boolean)
  return Array.from(new Set(parts)).join(' · ')
}

export function userInitial(userID:string,directory?:Map<string,UserSummary>){
  const label=userLabel(userID,directory)
  return label.slice(0,1).toUpperCase()
}

/**
 * Resolves user identifiers to directory profiles. Identifiers are batched into
 * one request and cached, so a comment thread with many authors still costs a
 * single lookup.
 */
export function useUserDirectory(ids:Array<string|undefined>){
  const unique=Array.from(new Set(ids.map(id=>(id??'').trim()).filter(Boolean).map(id=>id.toLowerCase()))).sort()
  const key=unique.join(',')
  const query=useQuery({
    queryKey:['user-directory',key],
    queryFn:()=>api<{items:UserSummary[]}>(`/api/v1/users:lookup?ids=${encodeURIComponent(key)}`),
    enabled:unique.length>0,
    staleTime:5*60*1000,
  })
  const directory=new Map<string,UserSummary>()
  for(const item of query.data?.items??[]){
    // The response is external input, so entries without an identifier are skipped.
    const id=item?.user_id?.trim?.()
    if(!id)continue
    directory.set(id.toLowerCase(),item)
    if(item.email?.trim())directory.set(item.email.trim().toLowerCase(),item)
  }
  return directory
}

export function useUserSearch(query:string){
  const needle=query.trim()
  const result=useQuery({
    queryKey:['user-search',needle.toLowerCase()],
    queryFn:()=>api<{items:UserSummary[]}>(`/api/v1/users:lookup?q=${encodeURIComponent(needle)}`),
    enabled:needle.length>=1,
    staleTime:60*1000,
  })
  return result.data?.items??[]
}
