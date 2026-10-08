import { Search } from 'lucide-react'
import type { BuildInfo, Session } from '../types'
import { Brand } from './Brand'
import { NotificationMenu } from './NotificationMenu'
import { ProfileMenu } from './ProfileMenu'

export function AppHeader({build,session,children}:{build?:BuildInfo;session?:Session;children?:React.ReactNode}) {
  return <header className="app-header">
    <Brand/>
    {children ?? <div className="global-search"><Search size={17}/><input placeholder="워크북, 시트, 데이터 검색" aria-label="통합 검색"/><kbd>⌘ K</kbd></div>}
    <div className="header-actions"><NotificationMenu session={session}/><ProfileMenu build={build} session={session}/></div>
  </header>
}
