import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from './lib/api'
import { beginSilentSso, clearSilentSsoState, safeReturnTo, shouldAttemptSilentSso } from './lib/silent-sso'
import type { AuthConfig, BuildInfo, Session } from './types'
import { LoginPage } from './pages/LoginPage'
import { HomePage } from './pages/HomePage'
import { EditorPage } from './pages/EditorPage'
import { AdminPage } from './pages/AdminPage'
import { PreferencesPage } from './pages/PreferencesPage'

export function App() {
  const path = window.location.pathname
  const build = useQuery({ queryKey:['version'], queryFn:()=>api<BuildInfo>('/api/v1/version') })
  const auth = useQuery({ queryKey:['auth-config'], queryFn:()=>api<AuthConfig>('/api/v1/auth/config') })
  const authRequired = auth.data?.oidc_enabled === true || auth.data?.bootstrap_login_enabled === true
  const session = useQuery({ queryKey:['session'], queryFn:()=>api<Session>('/api/v1/session'), enabled:authRequired })
  const requestedReturnTo = path === '/login' ? new URLSearchParams(location.search).get('return_to') || '/' : `${location.pathname}${location.search}`
  const returnTo = safeReturnTo(requestedReturnTo)
  const authenticated = session.data?.authenticated === true
  const signedOut = authRequired && session.isFetched && !authenticated
  // Someone already signed in at Keycloak lands on the page they opened
  // without seeing a login screen; someone who is not is sent back to
  // /login?sso=none by the callback and never bounced again.
  const silentSso = signedOut && shouldAttemptSilentSso({ config: auth.data, pathname: path, search: location.search })
  useEffect(() => {
    if (authenticated) clearSilentSsoState()
    else if (silentSso) beginSilentSso(returnTo)
  }, [authenticated, silentSso, returnTo])

  if (path === '/login') return <LoginPage build={build.data} auth={auth.data} returnTo={returnTo} />
  if (silentSso) return <SilentSsoSplash />
  if (signedOut) return <LoginPage build={build.data} auth={auth.data} returnTo={returnTo} />
  if (path.startsWith('/admin')) return <AdminPage build={build.data} session={session.data} />
  if (path.startsWith('/preferences')) return <PreferencesPage build={build.data} session={session.data} />
  const workbookMatch = path.match(/^\/workbooks\/([^/]+)/)
  if (workbookMatch) return <EditorPage workbookId={workbookMatch[1]} build={build.data} session={session.data} />
  return <HomePage build={build.data} session={session.data} />
}

// Shown for the moment between deciding to try a silent sign-in and leaving
// for Keycloak, so the login screen does not flash first.
function SilentSsoSplash() {
  return <main className="sso-splash" role="status" aria-live="polite"><div className="sso-splash-spinner" /><p>회사 계정으로 자동 로그인하는 중…</p></main>
}
