import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { Activity, ArrowLeft, Building2, Bot, LineChart, Mail, Send, CheckCircle2, ChevronRight, Database, FileClock, KeyRound, ListFilter, Plus, RefreshCw, RotateCcw, Save, Search, ServerCog, Settings2, ShieldCheck, SlidersHorizontal, Trash2, Users, Workflow, XCircle ,UploadCloud} from 'lucide-react'
import { Brand } from '../components/Brand'
import { ProfileMenu } from '../components/ProfileMenu'
import { api } from '../lib/api'
import { PromptDialog, type PromptRequest } from '../components/PromptDialog'
import { HandoverDialog } from '../components/HandoverDialog'
import { UserImportDialog } from '../components/UserImportDialog'
import { useDialog } from '../lib/useDialog'
import type { AdminOverview, AIAction, MailDeliveryPage, AIHistoryItem, AIHistoryPage, BuildInfo, Department, DirectoryUser, GovernedWorkbook, LogEntry, Session, SettingVersion, SystemSetting } from '../types'

type Tab='overview'|'settings'|'users'|'departments'|'workbooks'|'ai'|'mail'|'analytics'|'logs'|'keys'|'system'
const tabFromURL=():Tab=>{const value=new URLSearchParams(location.search).get('tab');return ['settings','users','departments','workbooks','ai','mail','analytics','logs','keys','system'].includes(value||'')?value as Tab:'overview'}

export function AdminPage({build,session}:{build?:BuildInfo;session?:Session}) {
  const [tab,setTab]=useState<Tab>(tabFromURL())
  const navigate=(next:Tab)=>{history.replaceState(null,'',`/admin?tab=${next}`);setTab(next)}
  return <div className="console-shell"><aside className="console-sidebar"><Brand/><div className="console-label">ADMIN CONSOLE</div><nav>
    <button className={tab==='overview'?'active':''} onClick={()=>navigate('overview')}><Activity/> 개요 <ChevronRight/></button>
    <button className={tab==='settings'?'active':''} onClick={()=>navigate('settings')}><Settings2/> 시스템 설정 <ChevronRight/></button>
    <button className={tab==='users'?'active':''} onClick={()=>navigate('users')}><Users/> 사용자 및 역할 <ChevronRight/></button>
    <button className={tab==='workbooks'?'active':''} onClick={()=>navigate('workbooks')}><Database/> 워크북 거버넌스 <ChevronRight/></button>
    <button className={tab==='departments'?'active':''} onClick={()=>navigate('departments')}><Building2/> 부서 및 공유 <ChevronRight/></button>
    <button className={tab==='ai'?'active':''} onClick={()=>navigate('ai')}><Bot/> AI 호출 이력 <ChevronRight/></button>
    <button className={tab==='mail'?'active':''} onClick={()=>navigate('mail')}><Mail/> 알림 메일 <ChevronRight/></button>
    <button className={tab==='analytics'?'active':''} onClick={()=>navigate('analytics')}><LineChart/> 방문자 추적 <ChevronRight/></button>
    <button className={tab==='logs'?'active':''} onClick={()=>navigate('logs')}><Activity/> 서버 로그 <ChevronRight/></button>
    <button className={tab==='keys'?'active':''} onClick={()=>navigate('keys')}><KeyRound/> API 키 현황 <ChevronRight/></button>
    <button className={tab==='system'?'active':''} onClick={()=>navigate('system')}><ServerCog/> 시스템 상태 <ChevronRight/></button>
  </nav><div className="console-nav-group"><span>바로가기</span><a href="/preferences"><ShieldCheck/> 개인 환경설정</a><a href="/"><Database/> 워크스페이스로</a></div><a className="back-link" href="/"><ArrowLeft/> 워크스페이스로</a></aside>
    <div className="console-main"><header className="console-header"><div><span className="status-pill"><i/> 시스템 정상</span></div><ProfileMenu build={build} session={session}/></header>{tab==='overview'&&<OverviewPanel onNavigate={navigate}/>}{tab==='settings'&&<SettingsPanel/>}{tab==='users'&&<UsersPanel/>}{tab==='departments'&&<DepartmentsPanel/>}{tab==='workbooks'&&<WorkbookGovernancePanel/>}{tab==='ai'&&<AIHistoryPanel/>}{tab==='mail'&&<MailPanel/>}{tab==='analytics'&&<AnalyticsPanel/>}{tab==='logs'&&<LogsPanel/>}{tab==='keys'&&<AdminKeysPanel/>}{tab==='system'&&<SystemPanel build={build}/>}</div>
  </div>
}

const LINK_ACCESS_LABEL:Record<string,string>={restricted:'제한됨',organization:'조직 전체',anyone:'링크 공개'}

/** The console landing page: scale, sharing exposure and what needs attention. */
function OverviewPanel({onNavigate}:{onNavigate:(tab:'users'|'departments'|'workbooks'|'settings')=>void}){
  const overview=useQuery({queryKey:['admin-overview'],queryFn:()=>api<{overview:AdminOverview;policy:Record<string,string>}>('/api/v1/admin/overview')})
  const data=overview.data?.overview
  const policy=overview.data?.policy
  const attention=[
    {label:'링크가 있는 모든 사용자에게 공개',value:data?.anyone_shared??0,filter:'anyone',tone:'error-text'},
    {label:'조직 전체 공개',value:data?.organization_shared??0,filter:'organization',tone:'warn-text'},
    {label:'소유자가 없거나 정지된 워크북',value:data?.orphan_workbooks??0,filter:'orphan',tone:'warn-text'},
    {label:'1년 이상 손대지 않은 워크북',value:data?.dormant_workbooks??0,filter:'dormant',tone:'muted-text'},
    {label:'대기 중인 액세스 요청',value:data?.pending_access_requests??0,filter:'all',tone:''},
  ]
  return <main className="console-content">
    <div className="content-title"><div><span className="eyebrow">OPERATIONS OVERVIEW</span><h1>개요</h1><p>조직의 사용량과 공유 노출 상태를 한눈에 확인하고 관리 화면으로 이동합니다.</p></div><button className="secondary" onClick={()=>overview.refetch()}><RefreshCw/> 새로고침</button></div>
    <div className="metric-row"><div><small>사용자</small><strong>{(data?.users??0).toLocaleString()}</strong><span className="metric-note">활성 {data?.active_users??0} · 정지 {data?.suspended_users??0}</span></div><div><small>부서</small><strong>{(data?.departments??0).toLocaleString()}</strong><span className="metric-note">워크북 공유 대상</span></div><div><small>워크북</small><strong>{(data?.workbooks??0).toLocaleString()}</strong><span className="metric-note">휴지통 {data?.trashed_workbooks??0}개</span></div></div>
    <div className="metric-row"><div><small>공유된 워크북</small><strong>{(data?.shared_workbooks??0).toLocaleString()}</strong><span className="metric-note">공유 항목 {data?.shares??0}개</span></div><div><small>조직 전체 공개</small><strong className={(data?.organization_shared??0)>0?'warn-text':''}>{(data?.organization_shared??0).toLocaleString()}</strong><span className="metric-note">링크로 열람 가능</span></div><div><small>링크 공개</small><strong className={(data?.anyone_shared??0)>0?'error-text':''}>{(data?.anyone_shared??0).toLocaleString()}</strong><span className="metric-note">가장 넓은 공개 범위</span></div></div>
    <section className="admin-card">
      <div className="card-heading"><div className="card-icon"><ShieldCheck/></div><div><h2>점검이 필요한 항목</h2><p>항목을 선택하면 해당 조건으로 걸러진 워크북 목록이 열립니다.</p></div></div>
      <div className="settings-table">
        {attention.map(item=><div className="settings-row attention-row" key={item.label}>
          <div><strong>{item.label}</strong></div>
          <strong className={item.tone}>{item.value.toLocaleString()}건</strong>
          <button className="secondary" onClick={()=>{history.replaceState(null,'',`/admin?tab=workbooks&filter=${item.filter}`);onNavigate('workbooks')}}>목록 보기</button>
        </div>)}
      </div>
    </section>
    <section className="admin-card">
      <div className="card-heading"><div className="card-icon"><Settings2/></div><div><h2>공유 정책</h2><p>허용 최대 링크 액세스 <b>{LINK_ACCESS_LABEL[policy?.max_link_access??'anyone']}</b> · 새 워크북 기본값 <b>{LINK_ACCESS_LABEL[policy?.default_link_access??'restricted']}</b></p></div><button className="secondary" onClick={()=>onNavigate('settings')}>시스템 설정 열기</button></div>
    </section>
    {overview.isLoading&&<div className="loading-card">요약을 불러오는 중…</div>}
  </main>
}

/**
 * Administrators can see every workbook, not only their own, so over-sharing and
 * orphaned data can be found and corrected in one place.
 */
function WorkbookGovernancePanel(){
  const client=useQueryClient()
  const [filter,setFilter]=useState<string>(()=>new URLSearchParams(location.search).get('filter')||'all')
  const [message,setMessage]=useState(''),[error,setError]=useState(''),[prompt,setPrompt]=useState<PromptRequest>()
  const workbooks=useQuery({queryKey:['admin-workbooks',filter],queryFn:()=>api<{items:GovernedWorkbook[]}>(`/api/v1/admin/workbooks?filter=${filter}`)})
  const items=workbooks.data?.items??[]
  const run=async(action:()=>Promise<unknown>,success:string)=>{
    setError('');setMessage('')
    try{await action();await client.invalidateQueries({queryKey:['admin-workbooks']});await client.invalidateQueries({queryKey:['admin-overview']});setMessage(success)}
    catch(reason){setError(reason instanceof Error?reason.message:'요청을 처리하지 못했습니다.')}
  }
  const restrict=(item:GovernedWorkbook)=>{
    if(!window.confirm(`'${item.title}'의 링크 액세스를 제한됨으로 되돌릴까요? 링크로만 접근하던 사용자는 열 수 없게 됩니다.`))return
    void run(()=>api(`/api/v1/workbooks/${item.id}/sharing`,{method:'PATCH',body:JSON.stringify({link_access:'restricted'})}),'링크 액세스를 제한했습니다.')
  }
  const transfer=(item:GovernedWorkbook)=>setPrompt({
    title:'소유권 이전',label:'새 소유자',placeholder:'사용자 ID 또는 이메일',confirmLabel:'이전',
    hint:`'${item.title}'의 소유자가 바뀝니다. 이전 소유자는 편집 권한도 잃습니다.`,
    validate:value=>value.trim()===''?'새 소유자를 입력하세요.':undefined,
    onSubmit:value=>void run(()=>api(`/api/v1/workbooks/${item.id}/sharing:transfer-ownership`,{method:'POST',body:JSON.stringify({new_owner_id:value.trim(),keep_as_editor:false})}),'소유권을 이전했습니다.'),
  })
  const trash=(item:GovernedWorkbook)=>{
    if(!window.confirm(`'${item.title}'을(를) 휴지통으로 옮길까요? 소유자가 복원할 수 있습니다.`))return
    void run(()=>api(`/api/v1/workbooks/${item.id}`,{method:'DELETE'}),'휴지통으로 옮겼습니다.')
  }
  const restore=(item:GovernedWorkbook)=>void run(()=>api(`/api/v1/workbooks/${item.id}/restore`,{method:'POST'}),'워크북을 복원했습니다.')
  // "링크 공개 47개" 를 보고 마흔일곱 번 누르는 사람은 없다. 남은 수를
  // 함께 알려 주어, 한 번에 다 닫지 못한 것을 사람이 알게 한다.
  const restrictAll=(scope:'anyone'|'organization')=>{
    const label=scope==='anyone'?'링크가 있는 모든 사용자에게 공개':'조직 전체 공개'
    if(!window.confirm(`${label} 상태인 워크북의 링크 액세스를 한 번에 제한할까요? 링크로만 접근하던 사용자는 열 수 없게 됩니다.`))return
    setError('');setMessage('')
    void api<{restricted:unknown[];failed:unknown[];remaining:number}>('/api/v1/admin/workbooks:restrict-links',
      {method:'POST',body:JSON.stringify({filter:scope})})
      .then(async result=>{
        await workbooks.refetch()
        const parts=[`${result.restricted.length}개를 제한했습니다.`]
        if(result.failed.length>0)parts.push(`${result.failed.length}개는 실패했습니다.`)
        if(result.remaining>0)parts.push(`아직 ${result.remaining}개 남았습니다. 다시 눌러 주세요.`)
        setMessage(parts.join(' '))
      })
      .catch(reason=>setError(reason instanceof Error?reason.message:'공개를 해제하지 못했습니다.'))
  }
  const filters=[{id:'all',label:'전체'},{id:'anyone',label:'링크 공개'},{id:'organization',label:'조직 전체'},{id:'orphan',label:'소유자 문제'},{id:'dormant',label:'잠든 워크북'},{id:'trashed',label:'휴지통'}]
  return <main className="console-content">
    {prompt&&<PromptDialog request={prompt} onClose={()=>setPrompt(undefined)}/>}
    <div className="content-title"><div><span className="eyebrow">WORKBOOK GOVERNANCE</span><h1>워크북 거버넌스</h1><p>모든 워크북의 소유자와 공유 범위를 점검하고 과도한 공개를 되돌리거나 소유권을 이전합니다.</p></div><div className="log-actions">{(filter==='anyone'||filter==='organization')&&<button className="secondary" onClick={()=>restrictAll(filter as 'anyone'|'organization')}><ShieldCheck/> 이 목록 전체 공개 해제</button>}<button className="secondary" onClick={()=>workbooks.refetch()}><RefreshCw/> 새로고침</button></div></div>
    {message&&<div className="result-banner" role="status"><CheckCircle2/><pre>{message}</pre><button onClick={()=>setMessage('')}>×</button></div>}
    {error&&<div className="result-banner error" role="alert"><XCircle/><pre>{error}</pre><button onClick={()=>setError('')}>×</button></div>}
    <div className="metric-row"><div><small>표시 중</small><strong>{items.length.toLocaleString()}</strong><span className="metric-note">최대 200개</span></div><div><small>공개 범위 확장</small><strong className={items.filter(item=>item.link_access!=='restricted').length>0?'warn-text':''}>{items.filter(item=>item.link_access!=='restricted').length}</strong><span className="metric-note">조직 전체 또는 링크 공개</span></div><div><small>소유자 문제</small><strong className={items.filter(item=>!item.owner_id||item.owner_status==='suspended').length>0?'error-text':''}>{items.filter(item=>!item.owner_id||item.owner_status==='suspended').length}</strong><span className="metric-note">이전이 필요합니다</span></div></div>
    <section className="admin-card">
      <div className="log-filters">
        {filters.map(item=><button key={item.id} className={filter===item.id?'filter-chip active':'filter-chip'} onClick={()=>{setFilter(item.id);history.replaceState(null,'',`/admin?tab=workbooks&filter=${item.id}`)}}>{item.label}</button>)}
      </div>
      <div className="settings-table">
        <div className="settings-row governance-row head"><span>워크북</span><span>소유자</span><span>공유 범위</span><span>시트</span><span>최근 수정</span><span>조치</span></div>
        {workbooks.isLoading?<div className="loading-card">워크북을 불러오는 중…</div>
          :items.length===0?<div className="table-empty"><Database/><strong>해당하는 워크북이 없습니다.</strong><span>다른 필터를 선택해 보세요.</span></div>
          :items.map(item=><div className="settings-row governance-row" key={item.id}>
            <div><strong><a href={`/workbooks/${item.id}`}>{item.title}</a></strong><small>{item.pending_access_requests>0?`액세스 요청 ${item.pending_access_requests}건`:`공유 ${item.share_count}개`}</small></div>
            <span className={!item.owner_id||item.owner_status==='suspended'?'error-text':''}>{item.owner_name||item.owner_id||'소유자 없음'}{item.owner_status==='suspended'&&' (정지됨)'}</span>
            <span className={item.link_access==='restricted'?'disabled-badge':item.link_access==='anyone'?'danger-badge':'warn-badge'}>{LINK_ACCESS_LABEL[item.link_access]}</span>
            <small>{item.sheet_count}개</small>
            <small>{new Date(item.updated_at).toLocaleDateString('ko-KR')}</small>
            <span className="row-actions">
              {item.deleted_at
                ?<button onClick={()=>restore(item)}><RotateCcw/> 복원</button>
                :<>{item.link_access!=='restricted'&&<button onClick={()=>restrict(item)}><ShieldCheck/> 공개 해제</button>}
                  <button onClick={()=>transfer(item)}><Users/> 소유권</button>
                  <button className="danger" onClick={()=>trash(item)}><Trash2/> 휴지통</button></>}
            </span>
          </div>)}
      </div>
    </section>
  </main>
}

/**
 * Identity comes from the identity provider or the bootstrap login, so the
 * console manages what kanpic owns: account status, kanpic roles that
 * role-based sharing can target, notes, administrator rights and sessions.
 */
// 서버의 workbook.DormantUserDays 와 같은 수다. 화면과 서버가 다른 날 수를
// 쓰면 개요에 적힌 수와 목록의 줄 수가 어긋난다.
const DORMANT_USER_DAYS=90

function UsersPanel(){
  const client=useQueryClient()
  const [search,setSearch]=useState(''),[selected,setSelected]=useState<string>(),[role,setRole]=useState('')
  const [newUser,setNewUser]=useState({user_id:'',display_name:'',email:''})
  const [handover,setHandover]=useState<DirectoryUser>()
  const [dormantOnly,setDormantOnly]=useState(false)
  const [bulkOpen,setBulkOpen]=useState(false)
  const [showAdd,setShowAdd]=useState(false)
  const [message,setMessage]=useState(''),[error,setError]=useState(''),[prompt,setPrompt]=useState<PromptRequest>()
  const users=useQuery({queryKey:['admin-users'],queryFn:()=>api<{items:DirectoryUser[];admin_roles:string[];default_admin_role:string}>('/api/v1/admin/users')})
  const adminRoles=users.data?.admin_roles??[]
  const defaultAdminRole=users.data?.default_admin_role??'kanpic-admin'
  const isAdmin=(user:DirectoryUser)=>(user.roles??[]).some(item=>adminRoles.some(candidate=>candidate.toLowerCase()===item.toLowerCase()))
  const items=(users.data?.items??[]).filter(user=>{
    // 한 번도 들어온 적 없는 계정도 잠든 것으로 센다. 미리 등록해 두고
    // 아무도 쓰지 않은 계정이 그대로 남는 일이 흔하다.
    if(dormantOnly&&user.status==='active'){
      const seen=user.last_seen_at?Date.parse(user.last_seen_at):Number.NaN
      if(Number.isFinite(seen)&&Date.now()-seen<DORMANT_USER_DAYS*86_400_000)return false
    }else if(dormantOnly)return false
    const needle=search.trim().toLowerCase()
    if(!needle)return true
    return [user.user_id,user.display_name,user.email,...(user.roles??[]),...(user.departments??[])].filter(Boolean).some(value=>String(value).toLowerCase().includes(needle))
  })
  const current=(users.data?.items??[]).find(user=>user.user_id===selected)
  const run=async(action:()=>Promise<unknown>,success:string)=>{
    setError('');setMessage('')
    try{await action();await client.invalidateQueries({queryKey:['admin-users']});setMessage(success)}
    catch(reason){setError(reason instanceof Error?reason.message:'요청을 처리하지 못했습니다.')}
  }
  const create=()=>run(async()=>{
    const created=await api<DirectoryUser>('/api/v1/admin/users',{method:'POST',body:JSON.stringify({...newUser,user_id:newUser.user_id.trim()})})
    setNewUser({user_id:'',display_name:'',email:''});setShowAdd(false);setSelected(created.user_id)
  },'사용자를 등록했습니다.')
  const setStatus=(user:DirectoryUser,status:'active'|'suspended')=>run(
    ()=>api<DirectoryUser>(`/api/v1/admin/users/${encodeURIComponent(user.user_id)}`,{method:'PATCH',body:JSON.stringify({status})}),
    status==='suspended'?'계정을 정지하고 모든 세션을 종료했습니다.':'계정 정지를 해제했습니다.')
  const grant=(user:DirectoryUser)=>{
    const name=role.trim()
    if(!name)return
    void run(async()=>{await api<DirectoryUser>(`/api/v1/admin/users/${encodeURIComponent(user.user_id)}/roles`,{method:'POST',body:JSON.stringify({role:name})});setRole('')},`${name} 역할을 부여했습니다.`)
  }
  const revoke=(user:DirectoryUser,name:string)=>run(
    ()=>api<DirectoryUser>(`/api/v1/admin/users/${encodeURIComponent(user.user_id)}/roles/${encodeURIComponent(name)}`,{method:'DELETE'}),
    `${name} 역할을 회수했습니다.`)
  const signOut=(user:DirectoryUser)=>run(
    ()=>api<{revoked_sessions:number}>(`/api/v1/admin/users/${encodeURIComponent(user.user_id)}/sessions`,{method:'DELETE'}),
    '모든 세션을 종료했습니다.')
  const note=(user:DirectoryUser)=>setPrompt({
    title:`${user.user_id} 메모`,label:'메모',value:user.note??'',multiline:true,
    hint:'관리자만 볼 수 있는 메모입니다. 비우면 삭제됩니다.',
    onSubmit:value=>void run(()=>api<DirectoryUser>(`/api/v1/admin/users/${encodeURIComponent(user.user_id)}`,{method:'PATCH',body:JSON.stringify({note:value})}),'메모를 저장했습니다.'),
  })
  const promote=(user:DirectoryUser)=>{
    if(!window.confirm(`'${user.user_id}' 계정을 관리자로 지정할까요? 모든 워크북과 시스템 설정에 접근할 수 있게 됩니다.`))return
    void run(()=>api<DirectoryUser>(`/api/v1/admin/users/${encodeURIComponent(user.user_id)}/roles`,{method:'POST',body:JSON.stringify({role:defaultAdminRole})}),'관리자로 지정했습니다.')
  }
  const demote=(user:DirectoryUser)=>{
    const held=(user.roles??[]).filter(item=>adminRoles.some(candidate=>candidate.toLowerCase()===item.toLowerCase()))
    if(!window.confirm(`'${user.user_id}' 계정의 관리자 권한(${held.join(', ')})을 회수할까요?`))return
    void run(async()=>{for(const item of held)await api<DirectoryUser>(`/api/v1/admin/users/${encodeURIComponent(user.user_id)}/roles/${encodeURIComponent(item)}`,{method:'DELETE'})},'관리자 권한을 회수했습니다.')
  }
  return <main className="console-content">
    {prompt&&<PromptDialog request={prompt} onClose={()=>setPrompt(undefined)}/>}
    {bulkOpen&&<UserImportDialog onClose={()=>setBulkOpen(false)} onDone={text=>{setBulkOpen(false);setMessage(text);void users.refetch()}}/>}
    {handover&&<HandoverDialog user={handover} onClose={()=>setHandover(undefined)} onDone={text=>{setHandover(undefined);setMessage(text)}}/>}
    <div className="content-title"><div><span className="eyebrow">ACCESS CONTROL</span><h1>사용자 및 역할</h1><p>로그인한 사용자는 자동으로 등록됩니다. 계정 정지, 관리자 지정, kanpic 역할 부여와 세션 종료를 관리합니다.</p></div><div className="log-actions"><button className="secondary" onClick={()=>setBulkOpen(true)}><UploadCloud/> 일괄 등록…</button><button className="primary" onClick={()=>setShowAdd(true)}><Plus/> 사용자 등록</button></div></div>
    {message&&<div className="result-banner" role="status"><CheckCircle2/><pre>{message}</pre><button onClick={()=>setMessage('')}>×</button></div>}
    {error&&<div className="result-banner error" role="alert"><XCircle/><pre>{error}</pre><button onClick={()=>setError('')}>×</button></div>}
    <div className="metric-row"><div><small>전체 사용자</small><strong>{(users.data?.items??[]).length.toLocaleString()}</strong><span className="metric-note">디렉터리 등록 기준</span></div><div><small>관리자</small><strong>{(users.data?.items??[]).filter(isAdmin).length}</strong><span className="metric-note">{adminRoles.join(', ')}</span></div><div><small>정지된 계정</small><strong className={(users.data?.items??[]).some(user=>user.status==='suspended')?'error-text':''}>{(users.data?.items??[]).filter(user=>user.status==='suspended').length}</strong><span className="metric-note">모든 요청이 차단됩니다</span></div></div>
    <section className="admin-card">
      <div className="log-filters"><div><Search/><input aria-label="사용자 검색" value={search} onChange={event=>setSearch(event.target.value)} placeholder="사용자, 이메일, 역할, 부서 검색"/></div><label className="dormant-toggle"><input type="checkbox" aria-label="잠든 계정만" checked={dormantOnly} onChange={event=>setDormantOnly(event.target.checked)}/> 잠든 계정만({DORMANT_USER_DAYS}일)</label></div>
      <div className="settings-table">
        <div className="settings-row user-row head"><span>사용자</span><span>역할</span><span>부서</span><span>워크북</span><span>마지막 접속</span><span>상태</span></div>
        {users.isLoading?<div className="loading-card">사용자를 불러오는 중…</div>
          :items.length===0?<div className="table-empty"><Users/><strong>사용자가 없습니다.</strong><span>사용자가 로그인하거나 직접 등록하면 표시됩니다.</span></div>
          :items.map(user=><div className={selected===user.user_id?'settings-row user-row selected':'settings-row user-row'} key={user.user_id} onClick={()=>setSelected(user.user_id)}>
            <div><strong>{user.display_name||user.user_id}</strong><small>{user.email||user.user_id}</small></div>
            <span className="scope-list">{isAdmin(user)&&<b className="admin-chip">관리자</b>}{(user.roles??[]).join(', ')||'—'}</span>
            <small>{(user.departments??[]).join(', ')||'—'}</small>
            <small>{user.owned_workbooks.toLocaleString()}개</small>
            <small>{user.last_seen_at?new Date(user.last_seen_at).toLocaleDateString('ko-KR'):'기록 없음'}</small>
            <span className={user.status==='suspended'?'disabled-badge':'enabled-badge'}>{user.status==='suspended'?'정지됨':'활성'}</span>
          </div>)}
      </div>
    </section>
    {current&&<section className="admin-card">
      <div className="card-heading"><div className="card-icon"><ShieldCheck/></div><div><h2>{current.display_name||current.user_id}</h2><p>{current.email||current.user_id}{current.note?` · ${current.note}`:''}</p></div><span className={current.status==='suspended'?'disabled-badge':'enabled-badge'}>{current.status==='suspended'?'정지됨':'활성'}</span></div>
      <div className="card-body">
        <div className="field-grid">
          <label>kanpic 역할<div className="chip-row">{(current.roles??[]).length===0?<span className="muted-text">부여된 역할이 없습니다.</span>:(current.roles??[]).map(item=><span className="role-chip" key={item}>{item}<button aria-label={`${item} 역할 회수`} onClick={()=>revoke(current,item)}><XCircle/></button></span>)}</div></label>
          <label>역할 부여<div className="inline-field"><input aria-label="부여할 역할" placeholder="예: kanpic-analyst" value={role} onChange={event=>setRole(event.target.value)} onKeyDown={event=>{if(event.key==='Enter')grant(current)}}/><button className="secondary" disabled={!role.trim()} onClick={()=>grant(current)}>부여</button></div></label>
        </div>
        <p className="field-hint">역할은 워크북 공유 창에서 ‘역할’ 대상으로 선택할 수 있습니다. {adminRoles.join(', ')} 역할은 관리자 권한을 부여합니다.</p>
        <div className="card-actions">
          {isAdmin(current)
            ?<button className="secondary" onClick={()=>demote(current)}><ShieldCheck/> 관리자 해제</button>
            :<button className="secondary" onClick={()=>promote(current)}><ShieldCheck/> 관리자로 지정</button>}
          <button className="secondary" onClick={()=>note(current)}>메모 편집</button>
          <button className="secondary" onClick={()=>signOut(current)}>모든 세션 종료</button><button className="secondary" onClick={()=>setHandover(current)}>가진 워크북 인수인계…</button>
          {current.status==='suspended'
            ?<button className="primary" onClick={()=>setStatus(current,'active')}><CheckCircle2/> 정지 해제</button>
            :<button className="danger" onClick={()=>{if(window.confirm(`'${current.user_id}' 계정을 정지하면 즉시 로그아웃되고 모든 요청이 차단됩니다. 계속할까요?`))setStatus(current,'suspended')}}><XCircle/> 계정 정지</button>}
        </div>
      </div>
    </section>}
    {showAdd&&<AdminModal label="사용자 등록" onClose={()=>setShowAdd(false)}>
      <h2>사용자 등록</h2>
      <p className="field-hint">아직 로그인하지 않은 사용자를 미리 등록해 역할과 부서를 배정할 수 있습니다.</p>
      <label>사용자 ID<input aria-label="사용자 ID" autoFocus placeholder="사용자 ID 또는 이메일" value={newUser.user_id} onChange={event=>setNewUser(current=>({...current,user_id:event.target.value}))}/></label>
      <label>표시 이름<input aria-label="표시 이름" placeholder="선택" value={newUser.display_name} onChange={event=>setNewUser(current=>({...current,display_name:event.target.value}))}/></label>
      <label>이메일<input aria-label="이메일" placeholder="선택" value={newUser.email} onChange={event=>setNewUser(current=>({...current,email:event.target.value}))}/></label>
      <div className="modal-actions"><button className="secondary" onClick={()=>setShowAdd(false)}>취소</button><button className="primary" disabled={!newUser.user_id.trim()} onClick={create}>등록</button></div>
    </AdminModal>}
  </main>
}


const AI_MODE_LABEL:Record<string,string>={formula:'수식 생성',explain:'수식 설명',fix:'오류 수정',summarize:'범위 요약',anomaly:'이상치 탐지',clean:'데이터 정제'}
const AI_STATUS_LABEL:Record<string,string>={planned:'승인 대기',completed:'분석 완료',applying:'적용 중',applied:'적용됨',undoing:'취소 중',undone:'취소됨',failed:'실패'}

/**
 * Every AI call in the organization: who asked, what it changed, what it cost
 * and how long it is kept. The list, the CSV export and the purge all use the
 * same filters so what is exported or deleted is what is on screen.
 */
function AIHistoryPanel(){
  const client=useQueryClient()
  const [status,setStatus]=useState(''),[mode,setMode]=useState(''),[actor,setActor]=useState(''),[query,setQuery]=useState(''),[since,setSince]=useState(''),[until,setUntil]=useState('')
  const [detail,setDetail]=useState<AIAction>()
  const [purgeBefore,setPurgeBefore]=useState('')
  const [result,setResult]=useState('')
  const parameters=new URLSearchParams({status,mode,actor,q:query,since,until,limit:'100'})
  const history=useQuery({queryKey:['ai-history',parameters.toString()],queryFn:()=>api<AIHistoryPage>(`/api/v1/admin/ai/actions?${parameters}`)})
  // There is no single setting route, so the value comes from the same list the
  // settings screen reads.
  const settings=useQuery({queryKey:['settings'],queryFn:()=>api<{items:SystemSetting[]}>('/api/v1/admin/settings')})
  const retentionDays=Number((settings.data?.items??[]).find(item=>item.key==='ai.history_retention_days')?.value??0)
  const summary=history.data?.summary
  const saveRetention=useMutation({
    mutationFn:(days:number)=>api<SystemSetting>('/api/v1/admin/settings/ai.history_retention_days',{method:'PUT',body:JSON.stringify({key:'ai.history_retention_days',value:days,value_type:'number'})}),
    onSuccess:async()=>{setResult('보존 기간을 저장했습니다.');await client.invalidateQueries({queryKey:['settings']})},
  })
  const purge=useMutation({
    mutationFn:(before:string)=>api<{removed:number}>(`/api/v1/admin/ai/actions?before=${encodeURIComponent(before)}`,{method:'DELETE'}),
    onSuccess:async data=>{setResult(`${data.removed.toLocaleString()}건을 삭제했습니다.`);await client.invalidateQueries({queryKey:['ai-history']})},
    onError:error=>setResult(error instanceof Error?error.message:'이력을 삭제하지 못했습니다.'),
  })
  const openDetail=async(item:AIHistoryItem)=>setDetail(await api<AIAction>(`/api/v1/admin/ai/actions/${item.id}`))
  return <main className="console-content">
    <div className="content-title"><div><span className="eyebrow">AI GOVERNANCE</span><h1>AI 호출 이력</h1><p>조직 전체의 AI 요청과 승인, 토큰 사용량을 확인하고 보존 기간을 관리합니다.</p></div>
      <div className="title-actions"><a className="secondary button-link" href={`/api/v1/admin/ai/actions?${parameters}&format=csv`}><FileClock/> CSV 내보내기</a><button className="secondary" onClick={()=>history.refetch()}><RefreshCw/> 새로고침</button></div></div>
    {result&&<div className="result-banner"><CheckCircle2/><pre>{result}</pre><button onClick={()=>setResult('')}>×</button></div>}
    <div className="metric-row">
      <div><small>전체 요청</small><strong>{(summary?.total??0).toLocaleString()}</strong></div>
      <div><small>적용됨 · 실패</small><strong>{(summary?.by_status?.applied??0).toLocaleString()} · <span className={summary?.by_status?.failed?'error-text':''}>{(summary?.by_status?.failed??0).toLocaleString()}</span></strong></div>
      <div><small>토큰 (입력 + 응답)</small><strong>{((summary?.prompt_tokens??0)+(summary?.completion_tokens??0)).toLocaleString()}</strong></div>
    </div>
    <section className="admin-card">
      <div className="log-filters">
        <div><Search/><input aria-label="요청 검색" value={query} onChange={event=>setQuery(event.target.value)} placeholder="요청 문장, 요약 검색"/></div>
        <div><Users/><input aria-label="사용자 검색" value={actor} onChange={event=>setActor(event.target.value)} placeholder="사용자"/></div>
        <select aria-label="상태" value={status} onChange={event=>setStatus(event.target.value)}><option value="">전체 상태</option>{Object.entries(AI_STATUS_LABEL).map(([key,label])=><option key={key} value={key}>{label}</option>)}</select>
        <select aria-label="작업 유형" value={mode} onChange={event=>setMode(event.target.value)}><option value="">전체 유형</option>{Object.entries(AI_MODE_LABEL).map(([key,label])=><option key={key} value={key}>{label}</option>)}</select>
        <input aria-label="시작일" type="date" value={since} onChange={event=>setSince(event.target.value)}/>
        <input aria-label="종료일" type="date" value={until} onChange={event=>setUntil(event.target.value)}/>
      </div>
      <div className="log-table">
        <div className="ai-history-row head"><span>시각</span><span>사용자</span><span>워크북</span><span>유형</span><span>상태</span><span>변경</span><span>토큰</span><span>모델</span></div>
        {history.isLoading&&<div className="ai-history-empty">이력을 불러오는 중…</div>}
        {!history.isLoading&&(history.data?.items.length??0)===0&&<div className="ai-history-empty">조건에 맞는 AI 호출이 없습니다.</div>}
        {history.data?.items.map(item=><button className="ai-history-row" key={item.id} onClick={()=>void openDetail(item)}>
          <time>{new Date(item.created_at).toLocaleString('ko-KR')}</time>
          <span>{item.actor_id}</span>
          <span title={item.request}>{item.workbook_title||item.workbook_id.slice(0,8)}</span>
          <span>{AI_MODE_LABEL[item.mode]??item.mode}</span>
          <em className={`ai-status ${item.status}`}>{AI_STATUS_LABEL[item.status]??item.status}</em>
          <span>{item.change_count>0?`${item.change_count}셀`:item.finding_count>0?`${item.finding_count}건`:'—'}</span>
          <span>{(item.prompt_tokens+item.completion_tokens).toLocaleString()}{item.attempts>1?` · 재시도 ${item.attempts-1}`:''}</span>
          <code>{item.model}</code>
        </button>)}
      </div>
    </section>
    <section className="admin-card">
      <div className="card-heading"><span className="card-icon"><Trash2/></span><div><h2>이력 보존</h2><p>보존 기간이 지난 완료·실패 이력을 매시간 자동으로 삭제합니다. 승인 대기 중인 요청은 지우지 않습니다.</p></div></div>
      <div className="settings-form-grid">
        <label><span>보존 기간 (일)</span><input aria-label="보존 기간" key={retentionDays} type="number" min={0} max={3650} defaultValue={retentionDays} onBlur={event=>{const days=Number(event.target.value);if(days!==retentionDays)saveRetention.mutate(days)}}/><small>0이면 계속 보관합니다. 현재 {retentionDays>0?`${retentionDays}일`:'무기한'} 보관 중입니다.</small></label>
        <label><span>지정 시점 이전 삭제</span><input aria-label="삭제 기준일" type="date" value={purgeBefore} onChange={event=>setPurgeBefore(event.target.value)}/><small>가장 오래된 기록: {summary?.oldest_at?new Date(summary.oldest_at).toLocaleDateString('ko-KR'):'없음'}</small></label>
      </div>
      <div className="card-actions"><button className="secondary" disabled={!purgeBefore||purge.isPending} onClick={()=>{if(window.confirm(`${purgeBefore} 이전의 완료·실패 이력을 삭제할까요?`))purge.mutate(purgeBefore)}}><Trash2/> 선택 시점 이전 삭제</button></div>
    </section>
    {(summary?.top_actors?.length??0)>0&&<section className="admin-card">
      <div className="card-heading"><span className="card-icon"><Users/></span><div><h2>사용자별 사용량</h2><p>현재 필터 기준 상위 사용자입니다.</p></div></div>
      <div className="settings-table">
        <div className="settings-row head"><div>사용자</div><div>요청</div><div>토큰</div><div></div><div></div></div>
        {summary?.top_actors.map(item=><div className="settings-row" key={item.actor_id}><div><strong>{item.actor_id}</strong></div><div>{item.count.toLocaleString()}건</div><div>{item.tokens.toLocaleString()}</div><div></div><div></div></div>)}
      </div>
    </section>}
    {detail&&<AdminModal label="AI 호출 상세" onClose={()=>setDetail(undefined)}>
      <h2>{AI_MODE_LABEL[detail.mode]??detail.mode} · {detail.range}</h2>
      <p className="field-hint">{detail.actor_id} · {new Date(detail.created_at).toLocaleString('ko-KR')} · {detail.model}</p>
      <div className="ai-detail-block"><strong>요청</strong><p>{detail.request}</p></div>
      {detail.summary&&<div className="ai-detail-block"><strong>요약</strong><p>{detail.summary}</p></div>}
      {detail.explanation&&<div className="ai-detail-block"><strong>설명</strong><p>{detail.explanation}</p></div>}
      {detail.error_message&&<div className="ai-detail-block error"><strong>오류</strong><p>{detail.error_message}</p></div>}
      {detail.changes.length>0&&<div className="ai-detail-block"><strong>변경 {detail.changes.length}셀</strong><ul>{detail.changes.slice(0,20).map(change=><li key={change.address}>{change.address}: {String(change.after.formula??change.after.value??'(비움)')}</li>)}</ul></div>}
      {(detail.events?.length??0)>0&&<div className="ai-detail-block"><strong>이벤트</strong><ul>{detail.events?.map(event=><li key={event.id}>{new Date(event.created_at).toLocaleString('ko-KR')} · {event.event_type}{event.tool_name?` · ${event.tool_name}`:''}</li>)}</ul></div>}
      <div className="modal-actions"><a className="secondary button-link" href={`/workbooks/${detail.workbook_id}`} target="_blank" rel="noopener">워크북 열기</a><button className="primary" onClick={()=>setDetail(undefined)}>닫기</button></div>
    </AdminModal>}
  </main>
}


const MAIL_EVENT_LABEL:Record<string,string>={'share.granted':'워크북 공유','comment.created':'댓글','comment.mention':'멘션','access_request.created':'액세스 요청','access_request.decided':'요청 처리','test':'테스트'}
const MAIL_STATUS_LABEL:Record<string,string>={queued:'발송 중',sent:'발송됨',failed:'실패',skipped:'건너뜀'}
const MAIL_FIELDS:Array<{key:string;label:string;hint:string;type?:string}>=[
  {key:'mail.smtp_host',label:'SMTP 서버',hint:'예: smtp.corp.local'},
  {key:'mail.smtp_port',label:'포트',hint:'사내 릴레이 25 · STARTTLS 587 · TLS 465',type:'number'},
  {key:'mail.security',label:'전송 보안',hint:'auto면 서버가 지원할 때만 STARTTLS를 사용합니다'},
  {key:'mail.from_address',label:'보내는 주소',hint:'비우면 kanpic@SMTP서버'},
  {key:'mail.from_name',label:'보내는 이름',hint:'메일 클라이언트에 표시됩니다'},
  {key:'mail.base_url',label:'kanpic 주소',hint:'메일 본문 링크에 사용합니다'},
  {key:'mail.username',label:'사용자 이름',hint:'비우면 인증 없이 발송합니다'},
  {key:'mail.password',label:'비밀번호',hint:'저장 후에는 다시 표시되지 않습니다',type:'password'},
]
const MAIL_TOGGLES:Array<{key:string;label:string}>=[
  {key:'mail.enabled',label:'알림 메일 사용'},
  {key:'mail.notify_share',label:'워크북 공유'},
  {key:'mail.notify_comment',label:'댓글과 답글'},
  {key:'mail.notify_mention',label:'멘션'},
  {key:'mail.notify_access_request',label:'액세스 요청'},
  {key:'mail.skip_tls_verify',label:'사설 인증서 검증 생략'},
]

/**
 * SMTP setup, a live connection test and the delivery log in one screen. The
 * fields write to the same settings store as the system settings page, so this
 * is a friendlier front door rather than a second source of truth.
 */
function MailPanel(){
  const client=useQueryClient()
  const [result,setResult]=useState('')
  const [recipient,setRecipient]=useState('')
  const [status,setStatus]=useState('')
  const settings=useQuery({queryKey:['settings'],queryFn:()=>api<{items:SystemSetting[]}>('/api/v1/admin/settings')})
  const deliveries=useQuery({queryKey:['mail-deliveries',status],queryFn:()=>api<MailDeliveryPage>(`/api/v1/admin/mail/deliveries?status=${status}&limit=100`),refetchInterval:15000})
  const valueOf=(key:string)=>(settings.data?.items??[]).find(item=>item.key===key)?.value
  const save=useSettingSaver(setResult)
  const test=useMutation({
    mutationFn:()=>api<{items:Array<{name:string;success:boolean;message:string}>}>('/api/v1/admin/settings:test',{method:'POST',body:'{}'}),
    onSuccess:data=>{const smtp=data.items.find(item=>item.name==='사내 SMTP');setResult(smtp?`${smtp.success?'연결 성공':'연결 실패'} · ${smtp.message}`:'메일 발송이 꺼져 있어 연결을 확인하지 않았습니다.')},
    onError:error=>setResult(error instanceof Error?error.message:'연결을 확인하지 못했습니다.'),
  })
  const sendTest=useMutation({
    mutationFn:()=>api<{sent:boolean}>('/api/v1/admin/mail:test',{method:'POST',body:JSON.stringify({recipient:recipient.trim()})}),
    onSuccess:async()=>{setResult(`${recipient} 주소로 테스트 메일을 보냈습니다.`);await client.invalidateQueries({queryKey:['mail-deliveries']})},
    onError:async error=>{setResult(error instanceof Error?error.message:'테스트 메일을 보내지 못했습니다.');await client.invalidateQueries({queryKey:['mail-deliveries']})},
  })
  const summary=deliveries.data?.summary
  return <main className="console-content">
    <div className="content-title"><div><span className="eyebrow">NOTIFICATIONS</span><h1>알림 메일</h1><p>사내 SMTP를 연결하고 공유·댓글·액세스 요청 알림 발송을 관리합니다.</p></div>
      <div className="title-actions"><button className="secondary" onClick={()=>test.mutate()}><ShieldCheck/> 연결 확인</button><button className="secondary" onClick={()=>deliveries.refetch()}><RefreshCw/> 새로고침</button></div></div>
    {result&&<div className="result-banner"><CheckCircle2/><pre>{result}</pre><button onClick={()=>setResult('')}>×</button></div>}
    <div className="metric-row">
      <div><small>발송 성공</small><strong>{(summary?.status?.sent??0).toLocaleString()}</strong></div>
      <div><small>실패</small><strong className={summary?.status?.failed?'error-text':''}>{(summary?.status?.failed??0).toLocaleString()}</strong></div>
      <div><small>상태</small><strong>{valueOf('mail.enabled')===true?'사용 중':'꺼짐'}</strong></div>
    </div>
    <section className="admin-card">
      <div className="card-heading"><span className="card-icon"><Mail/></span><div><h2>SMTP 연결</h2><p>사용자 이름을 비우면 인증 없이 발송합니다. 사내 릴레이는 보통 포트 25에 인증이 필요 없습니다.</p></div>{valueOf('mail.enabled')===true?<span className="enabled-badge">사용</span>:<span className="disabled-badge">중지</span>}</div>
      <div className="settings-form-grid">
        {MAIL_FIELDS.map(field=><label key={field.key}><span>{field.label}</span>
          <input aria-label={field.label} key={String(valueOf(field.key))} type={field.type??'text'} defaultValue={field.type==='password'?'':String(valueOf(field.key)??'')}
            onBlur={event=>{
              const raw=event.target.value
              if(field.type==='password'&&raw==='')return
              const value=field.type==='number'?Number(raw):raw
              if(value!==valueOf(field.key))save.mutate({key:field.key,value,type:field.type==='number'?'number':'string'})
            }}/>
          <small>{field.hint}</small></label>)}
      </div>
      <div className="mail-toggles">
        {MAIL_TOGGLES.map(toggle=><label key={toggle.key} className="mail-toggle">
          <input aria-label={toggle.label} type="checkbox" checked={valueOf(toggle.key)===true} onChange={event=>save.mutate({key:toggle.key,value:event.target.checked,type:'boolean'})}/>
          <span>{toggle.label}</span>
        </label>)}
      </div>
      <div className="card-actions">
        <input aria-label="테스트 수신 주소" className="mail-test-input" placeholder="테스트 받을 주소" value={recipient} onChange={event=>setRecipient(event.target.value)}/>
        <button className="primary" disabled={!recipient.includes('@')||sendTest.isPending} onClick={()=>sendTest.mutate()}><Send/> 테스트 메일 보내기</button>
      </div>
    </section>
    <section className="admin-card">
      <div className="card-heading compact"><div><h2>발송 이력</h2><p>최근 100건입니다. 실패한 메일은 오류 메시지와 함께 남습니다.</p></div>
        <select aria-label="발송 상태" value={status} onChange={event=>setStatus(event.target.value)}><option value="">전체 상태</option>{Object.entries(MAIL_STATUS_LABEL).map(([key,label])=><option key={key} value={key}>{label}</option>)}</select></div>
      <div className="log-table">
        <div className="mail-row head"><span>시각</span><span>이벤트</span><span>수신자</span><span>제목</span><span>상태</span><span>비고</span></div>
        {(deliveries.data?.items.length??0)===0&&<div className="ai-history-empty">발송 이력이 없습니다.</div>}
        {deliveries.data?.items.map(item=><div className="mail-row" key={item.id}>
          <time>{new Date(item.created_at).toLocaleString('ko-KR')}</time>
          <span>{MAIL_EVENT_LABEL[item.event]??item.event}</span>
          <span>{item.recipient}</span>
          <span title={item.subject}>{item.subject}</span>
          <em className={`ai-status ${item.status==='sent'?'applied':item.status==='failed'?'failed':''}`}>{MAIL_STATUS_LABEL[item.status]??item.status}</em>
          <small className={item.error_message?'error-text':''}>{item.error_message||(item.attempts>1?`재시도 ${item.attempts-1}회`:'')}</small>
        </div>)}
      </div>
    </section>
  </main>
}


/**
 * Saves one system setting and updates the cached list immediately, so a toggle
 * flips under the pointer instead of after a server round trip.
 */
function useSettingSaver(onDone:(message:string)=>void){
  const client=useQueryClient()
  return useMutation({
    mutationFn:({key,value,type}:{key:string;value:unknown;type:'string'|'number'|'boolean'})=>
      api<SystemSetting>(`/api/v1/admin/settings/${key}`,{method:'PUT',body:JSON.stringify({key,value,value_type:type})}),
    onMutate:async({key,value})=>{
      await client.cancelQueries({queryKey:['settings']})
      const previous=client.getQueryData<{items:SystemSetting[]}>(['settings'])
      client.setQueryData<{items:SystemSetting[]}>(['settings'],current=>current&&({
        items:current.items.map(item=>item.key===key?{...item,value}:item),
      }))
      return {previous}
    },
    onError:(error,_input,context)=>{
      if(context?.previous)client.setQueryData(['settings'],context.previous)
      onDone(error instanceof Error?error.message:'설정을 저장하지 못했습니다.')
    },
    onSuccess:()=>onDone('저장했습니다.'),
    onSettled:()=>client.invalidateQueries({queryKey:['settings']}),
  })
}

type TrackingViolation={origin:string;directive:string;page:string;count:number;first_seen:string;last_seen:string;allowed:boolean}

const TRACKING_PROVIDERS:Array<{id:string;label:string;hint:string}>=[
  {id:'none',label:'사용 안 함',hint:'추적 코드를 넣지 않습니다.'},
  {id:'ga4',label:'Google Analytics 4',hint:'측정 ID(G-)만 입력하면 로더와 설정 코드를 자동으로 만듭니다.'},
  {id:'gtm',label:'Google Tag Manager',hint:'컨테이너 ID(GTM-)를 입력하면 컨테이너 로더를 넣습니다.'},
  {id:'matomo',label:'Matomo (사내 설치)',hint:'서버 주소와 사이트 ID로 자체 호스팅 추적을 사용합니다.'},
  {id:'custom',label:'직접 입력',hint:'사내 분석 도구의 script 태그를 그대로 붙여 넣습니다.'},
]

/**
 * Visitor tracking is a snippet plus the policy it needs. This screen keeps the
 * two together: pick a provider, fill in the identifier, and the preview shows
 * exactly what will be inserted into every page.
 */
function AnalyticsPanel(){
  const client=useQueryClient()
  const [result,setResult]=useState('')
  const settings=useQuery({queryKey:['settings'],queryFn:()=>api<{items:SystemSetting[]}>('/api/v1/admin/settings')})
  const valueOf=(key:string)=>(settings.data?.items??[]).find(item=>item.key===key)?.value
  const provider=String(valueOf('analytics.provider')??'none')
  const enabled=valueOf('analytics.enabled')===true
  const save=useSettingSaver(message=>setResult(message==='저장했습니다.'?'저장했습니다. 새로 여는 페이지부터 적용됩니다.':message))
  const validate=useMutation({
    mutationFn:()=>api<{issues:Array<{key:string;message:string}>}>('/api/v1/admin/settings:validate',{method:'POST',body:'{}'}),
    onSuccess:data=>{
      const issue=(data.issues??[]).find(item=>item.key.startsWith('analytics.'))
      setResult(issue?`설정 확인 필요 · ${issue.message}`:'설정에 문제가 없습니다.')
    },
    onError:error=>setResult(error instanceof Error?error.message:'설정을 확인하지 못했습니다.'),
  })
  const measurement=String(valueOf('analytics.measurement_id')??'')
  const matomoURL=String(valueOf('analytics.matomo_url')??'')
  const matomoSite=String(valueOf('analytics.matomo_site_id')??'')
  const custom=String(valueOf('analytics.custom_snippet')??'')
  // The same addresses the server derives from the snippet, shown so it is
  // clear which domains were opened without being typed in.
  const detectedHosts=Array.from(new Set((custom.match(/https?:\/\/[^"'`<>\s),;\\+]+/g)??[]).map(found=>{try{return new URL(found).origin}catch{return ''}}).filter(Boolean)))
  const preview=provider==='ga4'&&measurement
    ?`<script async src="https://www.googletagmanager.com/gtag/js?id=${measurement}"></script>\n<script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('js',new Date());gtag('config','${measurement}');</script>`
    :provider==='gtm'&&measurement?`<script>(function(w,d,s,l,i){…})(window,document,'script','dataLayer','${measurement}');</script>`
    :provider==='matomo'&&matomoURL&&matomoSite?`<script>var _paq=window._paq=window._paq||[];_paq.push(['trackPageView']);…_paq.push(['setSiteId','${matomoSite}']);…u="${matomoURL.replace(/\/$/,'')}/"…</script>`
    :provider==='custom'?custom:''
  return <main className="console-content">
    <div className="content-title"><div><span className="eyebrow">ANALYTICS</span><h1>방문자 추적</h1><p>방문자 데이터 수집용 추적 코드를 모든 사용자 화면에 삽입합니다.</p></div>
      <div className="title-actions"><button className="secondary" onClick={()=>validate.mutate()}><ShieldCheck/> 설정 확인</button></div></div>
    {result&&<div className="result-banner"><CheckCircle2/><pre>{result}</pre><button onClick={()=>setResult('')}>×</button></div>}
    <section className="admin-card">
      <div className="card-heading"><span className="card-icon"><LineChart/></span><div><h2>추적 도구</h2><p>도구를 고르고 식별자만 넣으면 코드와 보안 정책이 자동으로 구성됩니다.</p></div>{enabled?<span className="enabled-badge">삽입 중</span>:<span className="disabled-badge">중지</span>}</div>
      <div className="tracking-providers" role="radiogroup" aria-label="추적 도구">
        {TRACKING_PROVIDERS.map(item=><button type="button" key={item.id} role="radio" aria-checked={provider===item.id} className={provider===item.id?'active':''}
          onClick={()=>save.mutate({key:'analytics.provider',value:item.id,type:'string'})}>
          <strong>{item.label}</strong><em>{item.hint}</em>
        </button>)}
      </div>
      <div className="settings-form-grid">
        {(provider==='ga4'||provider==='gtm')&&<label className="wide"><span>{provider==='ga4'?'측정 ID':'컨테이너 ID'}</span>
          <input aria-label="측정 ID" key={measurement} defaultValue={measurement} placeholder={provider==='ga4'?'G-XXXXXXXXXX':'GTM-XXXXXXX'}
            onBlur={event=>{if(event.target.value!==measurement)save.mutate({key:'analytics.measurement_id',value:event.target.value.trim(),type:'string'})}}/>
          <small>Google 계정에서 발급한 식별자입니다.</small></label>}
        {provider==='matomo'&&<>
          <label><span>Matomo 주소</span><input aria-label="Matomo 주소" key={matomoURL} defaultValue={matomoURL} placeholder="https://matomo.corp.local"
            onBlur={event=>{if(event.target.value!==matomoURL)save.mutate({key:'analytics.matomo_url',value:event.target.value.trim(),type:'string'})}}/><small>사내에 설치한 Matomo 주소</small></label>
          <label><span>사이트 ID</span><input aria-label="사이트 ID" key={matomoSite} defaultValue={matomoSite} placeholder="1"
            onBlur={event=>{if(event.target.value!==matomoSite)save.mutate({key:'analytics.matomo_site_id',value:event.target.value.trim(),type:'string'})}}/><small>Matomo에서 발급한 번호</small></label>
        </>}
        {provider==='custom'&&<label className="wide"><span>추적 코드</span>
          <textarea aria-label="추적 코드" key={custom.length} defaultValue={custom} rows={6} placeholder={'<script src="https://stats.corp.local/t.js"></script>'}
            onBlur={event=>{if(event.target.value!==custom)save.mutate({key:'analytics.custom_snippet',value:event.target.value,type:'string'})}}/>
          <small>script 태그를 포함한 HTML을 그대로 넣습니다. 최대 8KB이며 nonce는 자동으로 붙습니다.</small></label>}
        {provider!=='none'&&<label className="wide"><span>추가 허용 도메인</span>
          <input aria-label="추가 허용 도메인" key={String(valueOf('analytics.allowed_hosts'))} defaultValue={String(valueOf('analytics.allowed_hosts')??'')} placeholder="https://stats.corp.local, https://cdn.corp.local"
            onBlur={event=>{if(event.target.value!==String(valueOf('analytics.allowed_hosts')??''))save.mutate({key:'analytics.allowed_hosts',value:event.target.value.trim(),type:'string'})}}/>
          <small>직접 입력한 코드가 접속하는 도메인입니다. 콘텐츠 보안 정책에 함께 허용됩니다.</small></label>}
      </div>
      <div className="mail-toggles">
        <label className="mail-toggle"><input aria-label="추적 코드 삽입" type="checkbox" checked={enabled} onChange={event=>save.mutate({key:'analytics.enabled',value:event.target.checked,type:'boolean'})}/><span>추적 코드 삽입</span></label>
        <label className="mail-toggle"><input aria-label="관리자 화면 포함" type="checkbox" checked={valueOf('analytics.include_admin')===true} onChange={event=>save.mutate({key:'analytics.include_admin',value:event.target.checked,type:'boolean'})}/><span>관리자·개인 설정 화면 포함</span></label>
        <label className="mail-toggle"><input aria-label="body 끝에 삽입" type="checkbox" checked={valueOf('analytics.placement')==='body'} onChange={event=>save.mutate({key:'analytics.placement',value:event.target.checked?'body':'head',type:'string'})}/><span>body 끝에 삽입 (기본은 head)</span></label>
      </div>
    </section>
    <section className="admin-card">
      <div className="card-heading compact"><div><h2>삽입될 코드</h2><p>페이지마다 nonce가 새로 부여되며, 코드에 적힌 도메인은 보안 정책에 자동으로 추가됩니다.</p></div></div>
      <pre className="tracking-preview">{preview||'추적 도구를 선택하고 식별자를 입력하면 여기에 표시됩니다.'}</pre>
      {detectedHosts.length>0&&<p className="tracking-detected">코드에서 찾은 도메인: {detectedHosts.map(host=><code key={host}>{host}</code>)}</p>}
    </section>
    <BlockedRequests onAllowed={message=>setResult(message)}/>
  </main>
}

/**
 * Browsers report what the security policy refused. Showing those reports with
 * an allow button turns a console error nobody sees into a one-click fix.
 */
function BlockedRequests({onAllowed}:{onAllowed:(message:string)=>void}){
  const client=useQueryClient()
  const violations=useQuery({queryKey:['analytics-violations'],queryFn:()=>api<{items:TrackingViolation[]}>('/api/v1/admin/analytics/violations'),refetchInterval:20000})
  const refresh=async()=>{await client.invalidateQueries({queryKey:['analytics-violations']});await client.invalidateQueries({queryKey:['settings']})}
  const allow=useMutation({
    mutationFn:(origin:string)=>api('/api/v1/admin/analytics/violations:allow',{method:'POST',body:JSON.stringify({origin})}),
    onSuccess:async(_data,origin)=>{onAllowed(`${origin} 도메인을 허용했습니다. 새로 여는 페이지부터 적용됩니다.`);await refresh()},
    onError:error=>onAllowed(error instanceof Error?error.message:'도메인을 허용하지 못했습니다.'),
  })
  const clear=useMutation({mutationFn:()=>api('/api/v1/admin/analytics/violations',{method:'DELETE'}),onSuccess:refresh})
  const items=violations.data?.items??[]
  const blocked=items.filter(item=>!item.allowed)
  return <section className="admin-card">
    <div className="card-heading compact">
      <div><h2>차단된 요청</h2><p>추적 코드가 접속하려다 보안 정책에 막힌 주소입니다. 브라우저가 보내온 보고를 그대로 보여 줍니다.</p></div>
      <div className="title-actions"><button className="secondary" onClick={()=>clear.mutate()} disabled={items.length===0}>기록 비우기</button></div>
    </div>
    {items.length===0&&<p className="empty-hint">차단된 요청이 없습니다. 추적 코드를 넣은 뒤에도 값이 수집되지 않으면 이 목록을 확인하세요.</p>}
    {items.map(item=><div className="violation-row" key={`${item.directive} ${item.origin}`}>
      <div><strong>{item.origin}</strong><small>{item.directive} · {item.count.toLocaleString()}회 · {new Date(item.last_seen).toLocaleString('ko-KR')}</small></div>
      {item.allowed
        ?<span className="violation-allowed">허용됨</span>
        :<button className="secondary" onClick={()=>allow.mutate(item.origin)} disabled={allow.isPending}>이 도메인 허용</button>}
    </div>)}
    {blocked.length>0&&<p className="empty-hint">허용하면 <code>analytics.allowed_hosts</code>에 추가되어 스크립트·수집 요청·이미지 전송이 모두 열립니다.</p>}
  </section>
}

/** Small modal wrapper that reuses the shared dialog behaviour in the console. */
function AdminModal({label,onClose,children}:{label:string;onClose:()=>void;children:React.ReactNode}){
  const dialog=useDialog<HTMLDivElement>(onClose)
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <div className="modal" ref={dialog} role="dialog" aria-modal="true" aria-label={label}>{children}</div>
  </div>
}

/**
 * Departments decide which workbooks a person reaches, so the console keeps the
 * hierarchy, membership and the resulting share reach in one place.
 */
function DepartmentsPanel(){
  const client=useQueryClient()
  const [name,setName]=useState(''),[parentID,setParentID]=useState(''),[description,setDescription]=useState('')
  const [selected,setSelected]=useState<string>(),[member,setMember]=useState(''),[manager,setManager]=useState(''),[message,setMessage]=useState(''),[error,setError]=useState('')
  const [showAdd,setShowAdd]=useState(false)
  const departments=useQuery({queryKey:['departments'],queryFn:()=>api<{items:Department[]}>('/api/v1/departments')})
  const items=departments.data?.items??[]
  const current=items.find(item=>item.id===selected)
  const detail=useQuery({queryKey:['department',selected],queryFn:()=>api<Department>(`/api/v1/departments/${selected}`),enabled:Boolean(selected)})
  const run=async(action:()=>Promise<unknown>,success:string)=>{
    setError('');setMessage('')
    try{await action();await client.invalidateQueries({queryKey:['departments']});if(selected)await client.invalidateQueries({queryKey:['department',selected]});setMessage(success)}
    catch(reason){setError(reason instanceof Error?reason.message:'요청을 처리하지 못했습니다.')}
  }
  const create=()=>run(async()=>{
    const created=await api<Department>('/api/v1/departments',{method:'POST',body:JSON.stringify({name:name.trim(),parent_id:parentID||undefined,description:description.trim()})})
    setName('');setDescription('');setShowAdd(false);setSelected(created.id)
  },'부서를 만들었습니다.')
  const addManager=()=>run(async()=>{
    await api<Department>(`/api/v1/departments/${selected}/managers`,{method:'POST',body:JSON.stringify({user_ids:manager.split(/[,\s]+/).filter(Boolean)})})
    setManager('')
  },'부서 관리자를 지정했습니다.')
  const removeManager=(userID:string)=>run(()=>api<Department>(`/api/v1/departments/${selected}/managers/${encodeURIComponent(userID)}`,{method:'DELETE'}),'부서 관리자를 해제했습니다.')
  const addMember=()=>run(async()=>{
    await api<Department>(`/api/v1/departments/${selected}/members`,{method:'POST',body:JSON.stringify({user_ids:member.split(/[,\s]+/).filter(Boolean)})})
    setMember('')
  },'구성원을 추가했습니다.')
  const removeMember=(userID:string)=>run(()=>api<Department>(`/api/v1/departments/${selected}/members/${encodeURIComponent(userID)}`,{method:'DELETE'}),'구성원을 제거했습니다.')
  const remove=(department:Department)=>{
    if(!window.confirm(`'${department.name}' 부서를 삭제하면 이 부서로 공유된 워크북 권한도 함께 사라집니다. 계속할까요?`))return
    void run(async()=>{await api(`/api/v1/departments/${department.id}`,{method:'DELETE'});if(selected===department.id)setSelected(undefined)},'부서를 삭제했습니다.')
  }
  const rename=(department:Department,next:string)=>{
    const trimmed=next.trim()
    if(!trimmed||trimmed===department.name)return
    void run(()=>api<Department>(`/api/v1/departments/${department.id}`,{method:'PATCH',body:JSON.stringify({name:trimmed,expected_revision:department.revision})}),'부서 이름을 변경했습니다.')
  }
  const move=(department:Department,parent:string)=>void run(()=>api<Department>(`/api/v1/departments/${department.id}`,{method:'PATCH',body:JSON.stringify({parent_id:parent,expected_revision:department.revision})}),'상위 부서를 변경했습니다.')
  const members=detail.data?.members??[]
  return <main className="console-content">
    <div className="content-title"><div><span className="eyebrow">ORGANIZATION</span><h1>부서 및 공유</h1><p>부서를 만들고 구성원을 배치하면 워크북을 부서 단위로 공유할 수 있습니다. 상위 부서 공유는 하위 부서까지 상속됩니다.</p></div><button className="primary" onClick={()=>setShowAdd(true)}><Plus/> 부서 추가</button></div>
    {message&&<div className="result-banner" role="status"><CheckCircle2/><pre>{message}</pre><button onClick={()=>setMessage('')}>×</button></div>}
    {error&&<div className="result-banner error" role="alert"><XCircle/><pre>{error}</pre><button onClick={()=>setError('')}>×</button></div>}
    <div className="metric-row"><div><small>부서</small><strong>{items.length.toLocaleString()}</strong><span className="metric-note">최대 8단계 중첩</span></div><div><small>구성원 배치</small><strong>{items.reduce((sum,item)=>sum+item.member_count,0).toLocaleString()}</strong><span className="metric-note">중복 포함</span></div><div><small>최상위 부서</small><strong>{items.filter(item=>!item.parent_id).length}</strong><span className="metric-note">조직 루트</span></div></div>
    <section className="admin-card">
      <div className="settings-table">
        <div className="settings-row department-row head"><span>부서</span><span>설명</span><span>구성원</span><span>조치</span></div>
        {departments.isLoading?<div className="loading-card">부서를 불러오는 중…</div>
          :items.length===0?<div className="table-empty"><Building2/><strong>부서가 없습니다.</strong><span>첫 부서를 만들어 조직 구조를 등록하세요.</span></div>
          :items.map(item=><div className={selected===item.id?'settings-row department-row selected':'settings-row department-row'} key={item.id} onClick={()=>setSelected(item.id)}>
            <div style={{paddingLeft:item.depth*14}}><strong>{item.name}</strong><small>{item.path}</small></div>
            <small>{item.description||'—'}</small>
            <small>{item.member_count}명</small>
            <span className="row-actions"><button className="danger" aria-label={`${item.name} 삭제`} onClick={event=>{event.stopPropagation();remove(item)}}><Trash2/> 삭제</button></span>
          </div>)}
      </div>
    </section>
    {current&&<section className="admin-card">
      <div className="card-heading"><div className="card-icon"><Building2/></div><div><h2>{current.path||current.name}</h2><p>구성원 {members.length}명 · 상위 부서 공유는 이 부서까지 상속됩니다.</p></div></div>
      <div className="card-body">
        <div className="field-grid">
          <label>부서 이름<input aria-label="선택한 부서 이름" defaultValue={current.name} key={`${current.id}-${current.revision}`} onBlur={event=>rename(current,event.target.value)}/></label>
          <label>상위 부서<select aria-label="선택한 부서의 상위 부서" value={current.parent_id??''} onChange={event=>move(current,event.target.value)}>
            <option value="">최상위 부서</option>
            {items.filter(item=>item.id!==current.id).map(item=><option key={item.id} value={item.id}>{item.path||item.name}</option>)}
          </select></label>
          <label>부서 관리자 지정<div className="inline-field"><input aria-label="지정할 부서 관리자" placeholder="사용자 ID (쉼표로 여러 명)" value={manager} onChange={event=>setManager(event.target.value)} onKeyDown={event=>{if(event.key==='Enter'&&manager.trim())addManager()}}/><button className="secondary" disabled={!manager.trim()} onClick={addManager}>지정</button></div></label>
        <label>구성원 추가<div className="inline-field"><input aria-label="추가할 구성원" placeholder="사용자 ID 또는 이메일 (쉼표로 여러 명)" value={member} onChange={event=>setMember(event.target.value)} onKeyDown={event=>{if(event.key==='Enter'&&member.trim())addMember()}}/><button className="secondary" disabled={!member.trim()} onClick={addMember}>추가</button></div></label>
        </div>
        <p className="field-hint">부서 관리자는 <strong>이 부서와 그 아래 부서의 구성원 계정</strong> 만 다룹니다. 워크북 소유권 이전, 개요, 설정·로그·API 키는 전역 관리자만 씁니다.</p>
      <div className="chip-row">{(current.managers??[]).length===0?<span className="muted-text">지정된 부서 관리자가 없습니다.</span>:(current.managers??[]).map(item=><span className="role-chip" key={`manager-${item}`}>{item}<button aria-label={`${item} 부서 관리자 해제`} onClick={()=>removeManager(item)}><XCircle/></button></span>)}</div>
      <div className="chip-row">{members.length===0?<span className="muted-text">아직 구성원이 없습니다.</span>:members.map(userID=><span className="role-chip" key={userID}>{userID}<button aria-label={`${userID} 제거`} onClick={()=>removeMember(userID)}><XCircle/></button></span>)}</div>
      </div>
    </section>}
    {showAdd&&<AdminModal label="부서 추가" onClose={()=>setShowAdd(false)}>
      <h2>부서 추가</h2>
      <label>부서 이름<input aria-label="부서 이름" autoFocus placeholder="예: 재무팀" value={name} onChange={event=>setName(event.target.value)}/></label>
      <label>상위 부서<select aria-label="상위 부서" value={parentID} onChange={event=>setParentID(event.target.value)}>
        <option value="">최상위 부서</option>
        {items.map(item=><option key={item.id} value={item.id}>{item.path||item.name}</option>)}
      </select></label>
      <label>설명<input aria-label="부서 설명" placeholder="선택" value={description} onChange={event=>setDescription(event.target.value)}/></label>
      <div className="modal-actions"><button className="secondary" onClick={()=>setShowAdd(false)}>취소</button><button className="primary" disabled={!name.trim()} onClick={create}>부서 만들기</button></div>
    </AdminModal>}
  </main>
}

function SettingsPanel(){
  const client=useQueryClient();const [showAdd,setShowAdd]=useState(false);const [message,setMessage]=useState('')
  const settings=useQuery({queryKey:['admin-settings'],queryFn:()=>api<{items:SystemSetting[]}>('/api/v1/admin/settings')})
  const versions=useQuery({queryKey:['setting-versions'],queryFn:()=>api<{items:SettingVersion[]}>('/api/v1/admin/settings/versions')})
  const save=async(item:SystemSetting,value:unknown)=>{await api(`/api/v1/admin/settings/${encodeURIComponent(item.key)}`,{method:'PUT',body:JSON.stringify({...item,value})});await client.invalidateQueries({queryKey:['admin-settings']});await client.invalidateQueries({queryKey:['setting-versions']})}
  const validate=useMutation({mutationFn:()=>api<{valid:boolean;issues:Array<{key:string;message:string}>}>('/api/v1/admin/settings:validate',{method:'POST',body:'{}'}),onSuccess:r=>setMessage(r.valid?'모든 설정이 유효합니다.':r.issues.map(i=>`${i.key}: ${i.message}`).join('\n'))})
  const test=useMutation({mutationFn:()=>api<{items:Array<{name:string;success:boolean;message:string}>}>('/api/v1/admin/settings:test',{method:'POST',body:'{}'}),onSuccess:r=>setMessage(r.items.map(i=>`${i.success?'✓':'✕'} ${i.name}: ${i.message}`).join('\n'))})
  const byKey=useMemo(()=>new Map(settings.data?.items.map(item=>[item.key,item])),[settings.data])
  const oidcKeys=['auth.oidc.enabled','auth.oidc.issuer_url','auth.oidc.client_id','auth.oidc.client_secret','auth.oidc.scopes','auth.oidc.admin_roles','auth.oidc.auto_login','auth.oidc.mcp_enabled','auth.oidc.mcp_audiences','auth.oidc.ca_pem','server.public_url']
  const aiKeys=['ai.enabled','ai.gateway_url','ai.model','ai.api_key','ai.timeout_seconds','ai.max_input_cells','ai.max_changes','ai.ca_pem']
  const automationKeys=['automation.enabled','automation.max_cells_per_run','automation.max_runs_per_hour','automation.scheduler_poll_seconds']
  return <main className="console-content"><div className="content-title"><div><span className="eyebrow">PLATFORM CONFIGURATION</span><h1>시스템 설정</h1><p>서비스 설정을 변경하고 저장된 버전을 검증·복원합니다.</p></div><button className="primary" onClick={()=>setShowAdd(true)}><Plus/> 설정 추가</button></div>
    {message&&<div className="result-banner" role="status"><CheckCircle2/><pre>{message}</pre><button onClick={()=>setMessage('')}>×</button></div>}
    <section className="admin-card oidc-card"><div className="card-heading"><div className="card-icon"><ShieldCheck/></div><div><h2>Keycloak OIDC 간편 연결</h2><p>Public Client는 secret 없이, Confidential Client는 Client Secret과 PKCE로 연결합니다.</p></div><span className={byKey.get('auth.oidc.enabled')?.value?'enabled-badge':'disabled-badge'}>{byKey.get('auth.oidc.enabled')?.value?'사용 중':'사용 안 함'}</span></div>
      <div className="settings-form-grid">{oidcKeys.map(key=>{const item=byKey.get(key);return item?<SettingField key={key} item={item} onSave={value=>save(item,value)}/>:null})}</div>
      <div className="card-actions"><button className="secondary" onClick={()=>validate.mutate()} disabled={validate.isPending}><CheckCircle2/> 설정 검증</button><button className="primary" onClick={()=>test.mutate()} disabled={test.isPending}><RefreshCw className={test.isPending?'spin':''}/> 연결 테스트</button></div>
    </section>
    <section className="admin-card oidc-card"><div className="card-heading"><div className="card-icon"><Bot/></div><div><h2>사내 AI Gateway 간편 연결</h2><p>OpenAI 호환 `/v1` Gateway와 모델을 설정합니다. 선택 범위 데이터는 이 주소로만 전송됩니다.</p></div><span className={byKey.get('ai.enabled')?.value?'enabled-badge':'disabled-badge'}>{byKey.get('ai.enabled')?.value?'사용 중':'사용 안 함'}</span></div>
      <div className="settings-form-grid">{aiKeys.map(key=>{const item=byKey.get(key);return item?<SettingField key={key} item={item} onSave={value=>save(item,value)}/>:null})}</div>
      <div className="card-actions"><button className="secondary" onClick={()=>validate.mutate()} disabled={validate.isPending}><CheckCircle2/> 설정 검증</button><button className="primary" onClick={()=>test.mutate()} disabled={test.isPending}><RefreshCw className={test.isPending?'spin':''}/> Gateway 연결 테스트</button></div>
    </section>
    <section className="admin-card oidc-card"><div className="card-heading"><div className="card-icon"><Workflow/></div><div><h2>워크북 자동화 실행 정책</h2><p>PostgreSQL 작업 로그를 사용하는 수동·셀 변경·Cron 스케줄 자동화의 실행 범위, 시간당 한도와 확인 주기를 설정합니다.</p></div><span className={byKey.get('automation.enabled')?.value?'enabled-badge':'disabled-badge'}>{byKey.get('automation.enabled')?.value?'사용 중':'사용 안 함'}</span></div>
      <div className="settings-form-grid">{automationKeys.map(key=>{const item=byKey.get(key);return item?<SettingField key={key} item={item} onSave={value=>save(item,value)}/>:null})}</div>
      <div className="card-actions"><button className="secondary" onClick={()=>validate.mutate()} disabled={validate.isPending}><CheckCircle2/> 정책 검증</button><button className="primary" onClick={()=>test.mutate()} disabled={test.isPending}><RefreshCw className={test.isPending?'spin':''}/> 저장소 준비 상태 테스트</button></div>
    </section>
    <section className="admin-card"><div className="card-heading compact"><div><h2>전체 설정</h2><p>모든 설정은 키 단위로 추가, 수정, 삭제할 수 있습니다.</p></div><span>{settings.data?.items.length||0}개</span></div><div className="settings-table"><div className="settings-row head"><span>키</span><span>값</span><span>유형</span><span>최종 변경</span><span/></div>{settings.data?.items.map(item=><SettingRow key={item.key} item={item} onSave={value=>save(item,value)} onDelete={async()=>{if(confirm(`${item.key} 설정을 삭제할까요?`)){await api(`/api/v1/admin/settings/${encodeURIComponent(item.key)}`,{method:'DELETE'});client.invalidateQueries({queryKey:['admin-settings']});client.invalidateQueries({queryKey:['setting-versions']})}}}/>)}</div></section>
    <section className="admin-card"><div className="card-heading compact"><div><h2>설정 버전 이력</h2><p>변경할 때마다 전체 설정 스냅샷이 생성됩니다.</p></div><FileClock/></div><div className="version-list">{versions.data?.items.slice(0,12).map(version=><div key={version.revision}><span className="revision">r{version.revision}</span><div><strong>{version.change_summary}</strong><small>{version.actor_id} · {new Date(version.created_at).toLocaleString('ko-KR')}</small></div><button className="ghost" onClick={async()=>{if(confirm(`r${version.revision} 설정으로 복원할까요?`)){await api(`/api/v1/admin/settings/versions/${version.revision}:restore`,{method:'POST',body:'{}'});client.invalidateQueries({queryKey:['admin-settings']});client.invalidateQueries({queryKey:['setting-versions']})}}}><RotateCcw/> 복원</button></div>)}</div></section>
    {showAdd&&<AddSettingModal onClose={()=>setShowAdd(false)} onCreated={()=>{setShowAdd(false);client.invalidateQueries({queryKey:['admin-settings']});client.invalidateQueries({queryKey:['setting-versions']})}}/>}
  </main>
}

function SettingField({item,onSave}:{item:SystemSetting;onSave:(value:unknown)=>Promise<void>}) {
  const [value,setValue]=useState(item.secret&&item.value==null?'':Array.isArray(item.value)?item.value.join(', '):String(item.value??''))
  const [saved,setSaved]=useState(false)
  const commit=async()=>{let parsed:unknown=value;if(item.value_type==='boolean')parsed=value==='true';if(item.value_type==='number')parsed=Number(value);if(item.value_type==='string_list')parsed=value.split(',').map(v=>v.trim()).filter(Boolean);await onSave(parsed);setSaved(true);setTimeout(()=>setSaved(false),1500)}
  const caSetting=item.key.endsWith('.ca_pem')
  const placeholder=item.configured?'저장됨 · 변경할 때만 입력':caSetting?'-----BEGIN CERTIFICATE-----':item.secret?'비밀 값 입력':item.key.includes('issuer')?'https://keycloak.internal/realms/company':item.key==='ai.gateway_url'?'http://vllm.internal:8000/v1':''
  return <label className={caSetting?'wide':''}><span>{item.description||item.key}</span>{item.value_type==='boolean'?<select value={value} onChange={e=>setValue(e.target.value)}><option value="false">사용 안 함</option><option value="true">사용</option></select>:caSetting?<textarea value={value} placeholder={placeholder} onChange={e=>setValue(e.target.value)}/>:<input type={item.secret?'password':'text'} value={value} onChange={e=>setValue(e.target.value)} placeholder={placeholder}/>}<button onClick={commit}>{saved?<CheckCircle2/>:<Save/>}</button><small>{item.key}</small></label>
}
function SettingRow({item,onSave,onDelete}:{item:SystemSetting;onSave:(value:unknown)=>Promise<void>;onDelete:()=>void}){const display=item.secret?(item.configured?'••••••••':'설정 안 됨'):Array.isArray(item.value)?item.value.join(', '):JSON.stringify(item.value);return <div className="settings-row"><div><strong>{item.key}</strong><small>{item.description}</small></div><code>{display}</code><span className="type-badge">{item.value_type}</span><small>{new Date(item.updated_at).toLocaleDateString('ko-KR')}<br/>{item.updated_by}</small><div><button onClick={()=>{const raw=prompt(`${item.key}의 새 값`,String(Array.isArray(item.value)?item.value.join(', '):item.value??''));if(raw==null)return;let value:unknown=raw;if(item.value_type==='boolean')value=raw==='true';if(item.value_type==='number')value=Number(raw);if(item.value_type==='string_list')value=raw.split(',').map(v=>v.trim());if(item.value_type==='object')try{value=JSON.parse(raw)}catch{return alert('올바른 JSON을 입력하세요.')};onSave(value)}}><SlidersHorizontal/></button><button className="danger-icon" onClick={onDelete}><Trash2/></button></div></div>}
function AddSettingModal({onClose,onCreated}:{onClose:()=>void;onCreated:()=>void}){const [key,setKey]=useState(''),[description,setDescription]=useState(''),[type,setType]=useState<SystemSetting['value_type']>('string'),[value,setValue]=useState(''),[secret,setSecret]=useState(false);const create=async()=>{let parsed:unknown=value;if(type==='number')parsed=Number(value);if(type==='boolean')parsed=value==='true';if(type==='string_list')parsed=value.split(',').map(v=>v.trim()).filter(Boolean);if(type==='object')try{parsed=JSON.parse(value)}catch{return alert('올바른 JSON을 입력하세요.')};await api(`/api/v1/admin/settings/${encodeURIComponent(key)}`,{method:'PUT',body:JSON.stringify({key,value:parsed,value_type:type,description,secret})});onCreated()};return <div className="modal-backdrop"><div className="modal"><h2>새 시스템 설정</h2><p>점(.)으로 계층을 구분한 키를 권장합니다.</p><label>키<input value={key} onChange={e=>setKey(e.target.value)} placeholder="category.setting_name"/></label><label>설명<input value={description} onChange={e=>setDescription(e.target.value)}/></label><div className="modal-grid"><label>유형<select value={type} onChange={e=>setType(e.target.value as SystemSetting['value_type'])}><option>string</option><option>number</option><option>boolean</option><option>string_list</option><option>object</option></select></label><label className="check-label"><input type="checkbox" checked={secret} onChange={e=>setSecret(e.target.checked)}/> 비밀 값</label></div><label>값<textarea value={value} onChange={e=>setValue(e.target.value)}/></label><div className="modal-actions"><button className="secondary" onClick={onClose}>취소</button><button className="primary" onClick={create}>추가</button></div></div></div>}

function LogsPanel(){
  const [level,setLevel]=useState(''),[query,setQuery]=useState(''),[from,setFrom]=useState(''),[to,setTo]=useState('')
  // 화면과 내보내기가 같은 물음을 쓴다. 따로 만들면 감사에 넘긴 파일과
  // 화면에서 본 것이 달라진다.
  const search=`level=${level}&q=${encodeURIComponent(query)}&from=${from}&to=${to}`
  const logs=useQuery({queryKey:['server-logs',level,query,from,to],queryFn:()=>api<{items:LogEntry[]}>(`/api/v1/admin/logs?${search}&limit=200`),refetchInterval:5000});return <main className="console-content"><div className="content-title"><div><span className="eyebrow">OBSERVABILITY</span><h1>서버 로그</h1><p>구조화 로그와 Trace ID로 운영 상태를 추적합니다.</p></div><div className="log-actions"><a className="secondary" href={`/api/v1/admin/logs.csv?${search}`} download>CSV 내보내기</a><button className="secondary" onClick={()=>logs.refetch()}><RefreshCw/> 새로고침</button></div></div><div className="metric-row"><div><small>최근 로그</small><strong>{logs.data?.items.length||0}</strong></div><div><small>오류</small><strong className="error-text">{logs.data?.items.filter(i=>i.level==='ERROR').length||0}</strong></div><div><small>수집 방식</small><strong>PostgreSQL + stdout</strong></div></div><section className="admin-card log-card"><div className="log-filters"><div><Search/><input value={query} onChange={e=>setQuery(e.target.value)} placeholder="메시지, Trace ID 검색"/></div><select value={level} onChange={e=>setLevel(e.target.value)}><option value="">전체 레벨</option><option>INFO</option><option>WARN</option><option>ERROR</option></select><label className="dormant-toggle"><input type="checkbox" aria-label="관리자 행위만" checked={query==='admin.action'} onChange={e=>setQuery(e.target.checked?'admin.action':'')}/> 관리자 행위만</label><input type="date" aria-label="시작 날짜" value={from} onChange={e=>setFrom(e.target.value)}/><input type="date" aria-label="끝 날짜" value={to} onChange={e=>setTo(e.target.value)}/><button><ListFilter/> 필터</button></div>
      <p className="log-hint"><strong>관리자 행위만</strong> 을 켜면 계정 정지·역할 부여·소유권 이전처럼 사람에게 영향을 준 일만 모아 봅니다. 누가 했는지와, 전역 관리자였는지 위임받은 부서 관리자였는지가 함께 적힙니다. 화면은 최근 200건까지 보여 줍니다. <strong>CSV 내보내기</strong> 는 지금 거른 조건 그대로, <strong>개수를 자르지 않고</strong> 모두 내려받습니다.</p><div className="log-table"><div className="log-row head"><span>시각</span><span>레벨</span><span>메시지</span><span>속성</span><span>Trace ID</span></div>{logs.data?.items.map(log=><div className="log-row" key={log.id}><time>{new Date(log.logged_at).toLocaleString('ko-KR')}</time><span className={`level ${log.level.toLowerCase()}`}>{log.level}</span><strong>{log.message}</strong><code>{JSON.stringify(log.attributes)}</code><code>{log.trace_id?.slice(0,12)}</code></div>)}</div></section></main>}
function AdminKeysPanel(){return <main className="console-content"><div className="content-title"><div><span className="eyebrow">ACCESS CONTROL</span><h1>API 키 현황</h1><p>개인 키의 소유자, scope, 만료 및 폐기 상태를 감사하고, 새어 나간 키를 끊습니다.</p></div></div><AdminKeyTable/></main>}
function AdminKeyTable(){
  const keys=useQuery({queryKey:['admin-api-keys'],queryFn:()=>api<{items:Array<{id:string;user_id:string;name:string;prefix:string;scopes:string[];revoked_at?:string;last_used_at?:string}>}>('/api/v1/admin/api-keys')})
  const [message,setMessage]=useState('')
  // 키 하나가 새어 나가면 그 키만 끊는다. 계정을 통째로 정지하면 키와
  // 상관없는 그 사람의 일까지 멈춘다.
  const revoke=(key:{id:string;name:string;user_id:string})=>{
    if(!window.confirm(`'${key.user_id}' 의 키 '${key.name}' 을(를) 폐기할까요? 이 키를 쓰는 연동은 즉시 끊깁니다.`))return
    void api(`/api/v1/admin/api-keys/${encodeURIComponent(key.id)}`,{method:'DELETE'})
      .then(()=>{setMessage(`'${key.name}' 키를 폐기했습니다.`);void keys.refetch()})
      .catch(reason=>setMessage(reason instanceof Error?reason.message:'키를 폐기하지 못했습니다.'))
  }
  return <section className="admin-card">
  {message&&<div className="result-banner" role="status"><CheckCircle2/><pre>{message}</pre><button onClick={()=>setMessage('')}>×</button></div>}<div className="settings-table"><div className="settings-row head"><span>소유자 / 이름</span><span>Prefix</span><span>Scope</span><span>최근 사용</span><span>상태</span></div>{keys.data?.items.map(key=><div className="settings-row" key={key.id}><div><strong>{key.name}</strong><small>{key.user_id}</small></div><code>{key.prefix}…</code><span className="scope-list">{key.scopes.join(', ')}</span><small>{key.last_used_at?new Date(key.last_used_at).toLocaleString('ko-KR'):'사용 전'}</small><span className={key.revoked_at?'disabled-badge':'enabled-badge'}>{key.revoked_at?'폐기됨':'활성'}{!key.revoked_at&&<button className="secondary" onClick={()=>revoke(key)}>폐기</button>}</span></div>)}</div></section>}
function SystemPanel({build}:{build?:BuildInfo}){const health=useQuery({queryKey:['health'],queryFn:()=>api<{status:string;service:string}>('/healthz'),refetchInterval:10000});return <main className="console-content"><div className="content-title"><div><span className="eyebrow">SYSTEM STATUS</span><h1>시스템 상태</h1><p>초기 모놀리스 서비스와 데이터 저장소 상태입니다.</p></div></div><div className="service-grid"><div className="service-card"><CheckCircle2/><div><strong>kanpic API</strong><small>{health.data?.status||'확인 중'} · {build?.version}</small></div></div><div className="service-card"><CheckCircle2/><div><strong>PostgreSQL</strong><small>연결됨 · 서버 권위 저장소</small></div></div><div className="service-card muted"><XCircle/><div><strong>Redis</strong><small>초기 버전에서 사용하지 않음</small></div></div></div></main>}
