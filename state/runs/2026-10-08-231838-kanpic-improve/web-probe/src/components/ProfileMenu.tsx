import { useEffect, useRef, useState } from 'react'
import { ChevronDown, LogOut, Settings, ShieldCheck, UserRound } from 'lucide-react'
import type { BuildInfo, Session } from '../types'
import { markSignedOut } from '../lib/silent-sso'

export function ProfileMenu({build,session}:{build?:BuildInfo;session?:Session}) {
  const [open,setOpen]=useState(false)
  const root=useRef<HTMLDivElement>(null)
  useEffect(()=>{const close=(event:MouseEvent)=>{if(!root.current?.contains(event.target as Node))setOpen(false)};document.addEventListener('mousedown',close);return()=>document.removeEventListener('mousedown',close)},[])
  const name=session?.user?.display_name || '로컬 관리자'
  const initial=name.slice(0,1).toUpperCase()
  // Marked before the session goes away: a silent SSO attempt right after
  // signing out would make the sign-out look broken.
  const logout=async()=>{markSignedOut();await fetch('/auth/logout',{method:'POST'});window.location.href='/login'}
  return <div className="profile-menu" ref={root}>
    <button className="profile-trigger" onClick={()=>setOpen(!open)} aria-expanded={open}>
      <span className="avatar">{initial}</span><span className="profile-name">{name}</span><ChevronDown size={15}/>
    </button>
    {open && <div className="profile-popover">
      <div className="profile-summary"><span className="avatar large">{initial}</span><div><strong>{name}</strong><small>{session?.user?.email || (session?.authenticated?'로컬 bootstrap 관리자':'초기 설정 모드')}</small></div></div>
      <div className="menu-separator"/>
      <a href="/preferences"><UserRound size={16}/> 개인화 설정</a>
      {(session?.admin ?? true) && <a href="/admin"><ShieldCheck size={16}/> 관리자 콘솔</a>}
      <div className="menu-separator"/>
      <div className="version-menu"><Settings size={15}/><div><span>kanpic {build?.version ?? '…'}</span><small>{build?.commit && build.commit !== 'unknown' ? build.commit.slice(0,8) : 'development build'}</small></div></div>
      {session?.authenticated && <button className="menu-action" onClick={logout}><LogOut size={16}/> 로그아웃</button>}
    </div>}
  </div>
}
