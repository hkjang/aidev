import { useState, type FormEvent } from 'react'
import { ArrowRight, CheckCircle2, ChevronDown, Database, LockKeyhole, LogIn, ShieldCheck, Sparkles } from 'lucide-react'
import { api } from '../lib/api'
import type { AuthConfig, BuildInfo, Session } from '../types'
import { Brand } from '../components/Brand'

export function LoginPage({build,auth,returnTo='/'}:{build?:BuildInfo;auth?:AuthConfig;returnTo?:string}) {
  const [id,setID]=useState(''),[password,setPassword]=useState(''),[error,setError]=useState(''),[pending,setPending]=useState(false),[localLoginOpen,setLocalLoginOpen]=useState(false)
  const oidcEnabled=auth?.oidc_enabled===true,bootstrapEnabled=auth?.bootstrap_login_enabled===true
  // The OIDC callback lands here with sso=none when a silent (prompt=none)
  // attempt found no Keycloak session; the marker only explains the screen
  // and never changes where the buttons go.
  const ssoRefused=oidcEnabled&&new URLSearchParams(window.location.search).get('sso')==='none'
  // The local administrator stays reachable after SSO is configured as the
  // recovery path when Keycloak is unreachable, but folded away so the
  // organisation account is the obvious door.
  const bootstrapIsFallback=bootstrapEnabled&&oidcEnabled
  const bootstrapOpen=bootstrapEnabled&&(!bootstrapIsFallback||localLoginOpen)
  const oidcUrl=`/auth/login?return_to=${encodeURIComponent(returnTo)}`
  const bootstrapLogin=async(event:FormEvent)=>{event.preventDefault();setError('');setPending(true);try{await api<Session>('/auth/bootstrap/login',{method:'POST',body:JSON.stringify({id,password})});window.location.href=returnTo}catch(reason){setError(reason instanceof Error?reason.message:'로그인하지 못했습니다.');setPending(false)}}
  return <main className="login-page">
    <section className="login-visual">
      <div className="login-orb one"/><div className="login-orb two"/>
      <div className="login-story">
        <Brand/>
        <span className="eyebrow"><Sparkles size={15}/> AI spreadsheet workspace</span>
        <h1>데이터가 모이고,<br/>팀의 판단이 빨라집니다.</h1>
        <p>익숙한 스프레드시트 경험에 실시간 협업, 사내 데이터와 안전한 AI를 하나로 연결하세요.</p>
        <div className="login-benefits"><span><CheckCircle2/> 서버 권위 자동 저장</span><span><CheckCircle2/> 폐쇄망 완전 지원</span><span><CheckCircle2/> API · MCP 우선 설계</span></div>
      </div>
      <div className="login-version">kanpic {build?.version ?? '…'} · {build?.commit && build.commit !== 'unknown' ? build.commit.slice(0,8) : 'development'}</div>
    </section>
    <section className="login-form-wrap">
      <div className="login-form">
        <div className="login-icon"><LockKeyhole/></div>
        <h2>kanpic에 로그인</h2>
        <p>{oidcEnabled?'인증 후 원래 화면으로 안전하게 돌아갑니다.':bootstrapEnabled?'bootstrap 관리자 계정으로 로그인하세요.':'최초 설치 설정을 시작하세요.'}</p>
        {ssoRefused&&<div className="notice" role="status"><ShieldCheck size={20}/><span>회사 계정 세션이 없어 자동으로 로그인하지 않았습니다. 아래 버튼으로 로그인하세요.</span></div>}
        {oidcEnabled&&<a className="primary large-button sso-button" href={oidcUrl}><ShieldCheck size={19}/> 회사 계정으로 SSO 로그인</a>}
        {bootstrapIsFallback&&<button type="button" className="local-login-toggle" aria-expanded={localLoginOpen} aria-controls="local-login-form" onClick={()=>setLocalLoginOpen(open=>!open)}><LockKeyhole size={17}/><span>관리자 계정으로 로그인</span><ChevronDown size={17} className={localLoginOpen?'rotate-180':''}/></button>}
        {bootstrapOpen&&<form id="local-login-form" className={`bootstrap-login-form${oidcEnabled?' fallback':''}`} onSubmit={bootstrapLogin}>
          <div className="notice"><LockKeyhole size={20}/><span>{oidcEnabled?'SSO를 사용할 수 없을 때를 위한 복구용 관리자 계정입니다. 평소에는 회사 계정으로 로그인하세요.':'최초 설치 관리자 계정입니다. SSO를 설정한 뒤에도 복구용으로 계속 사용할 수 있습니다.'}</span></div>
          <label>Bootstrap 관리자<input autoFocus autoComplete="username" value={id} onChange={event=>setID(event.target.value)} required/></label><label>비밀번호<input type="password" autoComplete="current-password" value={password} onChange={event=>setPassword(event.target.value)} required/></label>{error&&<div className="login-error" role="alert">{error}</div>}<button className={`${oidcEnabled?'secondary':'primary'} large-button`} type="submit" disabled={pending}><LogIn size={18}/> {pending?'확인 중…':'관리자 로그인'}</button></form>}
        {!bootstrapEnabled&&!oidcEnabled&&<a className="primary large-button" href="/admin">로컬 관리자 설정 시작 <ArrowRight size={18}/></a>}
        <div className="login-security"><Database size={17}/><div><strong>데이터는 조직 내부에 유지됩니다</strong><small>Keycloak OIDC와 내부 PostgreSQL을 사용합니다.</small></div></div>
      </div>
      <footer>© {new Date().getFullYear()} kanpic · Private data collaboration</footer>
    </section>
  </main>
}
