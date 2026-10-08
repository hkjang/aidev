import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import {
  ArrowLeft,
  Check,
  Copy,
  Eye,
  KeyRound,
  Monitor,
  Moon,
  Palette,
  Pencil,
  Plus,
  RotateCw,
  Settings,
  Sun,
  Trash2,
} from 'lucide-react'
import { AppHeader } from '../components/AppHeader'
import { api } from '../lib/api'
import type { ApiKey, BuildInfo, CreatedApiKey, Session } from '../types'

const agentScopes = [
  'mcp.use',
  'ai.use',
  'workbook.*',
  'range.*',
  'format.*',
  'formula.*',
  'chart.*',
  'pivot.*',
  'automation.*',
  'automation.webhook.invoke',
  'version.*',
  'comment.*',
  'presentation.*',
  'profile.read',
]

export function PreferencesPage({ build, session }: { build?: BuildInfo; session?: Session }) {
  const [section, setSection] = useState<'appearance' | 'keys'>('appearance')
  return (
    <div className="page-shell preferences-shell">
      <AppHeader build={build} session={session} />
      <main className="preferences-layout">
        <aside>
          <a href="/"><ArrowLeft /> 워크스페이스</a>
          <div className="preference-person">
            <span>{(session?.user?.display_name || '로컬').slice(0, 1)}</span>
            <div>
              <strong>{session?.user?.display_name || '로컬 관리자'}</strong>
              <small>{session?.user?.email || '개인 설정'}</small>
            </div>
          </div>
          <nav>
            <button className={section === 'appearance' ? 'active' : ''} onClick={() => setSection('appearance')}><Palette /> 개인화</button>
            <button className={section === 'keys' ? 'active' : ''} onClick={() => setSection('keys')}><KeyRound /> API 키</button>
          </nav>
        </aside>
        {section === 'appearance' ? <AppearancePanel /> : <ApiKeysPanel />}
      </main>
    </div>
  )
}

function AppearancePanel() {
  const preferences = useQuery({ queryKey: ['preferences'], queryFn: () => api<{ values: Record<string, unknown> }>('/api/v1/me/preferences') })
  const [theme, setTheme] = useState('system')
  const [locale, setLocale] = useState('ko-KR')
  const [density, setDensity] = useState('comfortable')
  const save = useMutation({
    mutationFn: () => api('/api/v1/me/preferences', {
      method: 'PUT',
      body: JSON.stringify({ values: { ...preferences.data?.values, theme, locale, density } }),
    }),
  })
  return (
    <section className="preference-content">
      <span className="eyebrow">PERSONALIZATION</span>
      <h1>나만의 작업 환경</h1>
      <p>이 설정은 현재 계정에만 적용되며 시스템 설정과 분리됩니다.</p>
      <div className="preference-card">
        <div className="preference-heading"><Monitor /><div><h2>화면 테마</h2><p>kanpic 화면의 밝기를 선택하세요.</p></div></div>
        <div className="theme-options">
          {([['light', Sun, '라이트'], ['dark', Moon, '다크'], ['system', Monitor, '시스템 설정']] as const).map(([value, Icon, label]) => (
            <button className={theme === value ? 'active' : ''} onClick={() => setTheme(value)} key={value}>
              <span><Icon /></span><strong>{label}</strong>{theme === value && <Check />}
            </button>
          ))}
        </div>
      </div>
      <div className="preference-card rows">
        <label><div><Settings /><span><strong>언어 및 로케일</strong><small>날짜와 숫자 표시 방식</small></span></div><select value={locale} onChange={event => setLocale(event.target.value)}><option value="ko-KR">한국어 (대한민국)</option><option value="en-US">English (United States)</option></select></label>
        <label><div><Eye /><span><strong>화면 밀도</strong><small>도구와 목록의 간격</small></span></div><select value={density} onChange={event => setDensity(event.target.value)}><option value="comfortable">여유롭게</option><option value="compact">조밀하게</option></select></label>
      </div>
      <button className="primary save-preference" onClick={() => save.mutate()}>{save.isSuccess ? <Check /> : null} 개인 설정 저장</button>
    </section>
  )
}

function ApiKeysPanel() {
  const client = useQueryClient()
  const [secret, setSecret] = useState('')
  const [editing, setEditing] = useState<ApiKey | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const keys = useQuery({ queryKey: ['my-api-keys'], queryFn: () => api<{ items: ApiKey[] }>('/api/v1/me/api-keys') })
  const refresh = () => client.invalidateQueries({ queryKey: ['my-api-keys'] })
  const rotate = async (id: string) => {
    if (!confirm('기존 키는 즉시 폐기됩니다. 회전할까요?')) return
    const item = await api<CreatedApiKey>(`/api/v1/me/api-keys/${id}:rotate`, { method: 'POST', body: '{}' })
    setSecret(item.secret)
    await refresh()
  }
  const revoke = async (id: string) => {
    if (!confirm('이 키를 폐기할까요?')) return
    await api(`/api/v1/me/api-keys/${id}`, { method: 'DELETE' })
    await refresh()
  }
  return (
    <section className="preference-content">
      <div className="preference-title-row">
        <div><span className="eyebrow">DEVELOPER ACCESS</span><h1>개인 API 키</h1><p>REST와 MCP 에이전트에서 사용할 최소 권한 키를 관리합니다.</p></div>
        <button className="primary" onClick={() => setShowCreate(true)}><Plus /> 새 키</button>
      </div>
      {secret && (
        <div className="secret-banner"><KeyRound /><div><strong>새 키를 지금 복사하세요</strong><p>보안을 위해 이 원문은 다시 표시되지 않습니다.</p><code>{secret}</code></div><button onClick={() => navigator.clipboard.writeText(secret)}><Copy /> 복사</button></div>
      )}
      <div className="key-list">
        {keys.data?.items.map(key => (
          <div className={`key-card ${key.revoked_at ? 'revoked' : ''}`} key={key.id}>
            <div className="key-icon"><KeyRound /></div>
            <div className="key-info">
              <div><strong>{key.name}</strong><span className={key.revoked_at ? 'disabled-badge' : 'enabled-badge'}>{key.revoked_at ? '폐기' : '활성'}</span></div>
              <code>{key.prefix}••••••••••••</code>
              <div className="scope-list">{key.scopes.map(scope => <span key={scope}>{scope}</span>)}</div>
              <small>생성 {new Date(key.created_at).toLocaleDateString('ko-KR')} · 만료 {key.expires_at ? new Date(key.expires_at).toLocaleString('ko-KR') : '없음'} · {key.last_used_at ? `최근 사용 ${new Date(key.last_used_at).toLocaleString('ko-KR')}` : '아직 사용하지 않음'}</small>
            </div>
            {!key.revoked_at && (
              <div className="key-actions">
                <button onClick={() => setEditing(key)}><Pencil /> 수정</button>
                <button onClick={() => rotate(key.id)}><RotateCw /> 회전</button>
                <button className="danger" onClick={() => revoke(key.id)}><Trash2 /> 폐기</button>
              </div>
            )}
          </div>
        ))}
      </div>
      {showCreate && <ApiKeyDialog onClose={() => setShowCreate(false)} onSaved={item => { if ('secret' in item) setSecret(item.secret); setShowCreate(false); void refresh() }} />}
      {editing && <ApiKeyDialog item={editing} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); void refresh() }} />}
    </section>
  )
}

function ApiKeyDialog({ item, onClose, onSaved }: { item?: ApiKey; onClose: () => void; onSaved: (item: CreatedApiKey | ApiKey) => void }) {
  const [name, setName] = useState(item?.name || '새 에이전트 키')
  const [scopes, setScopes] = useState<string[]>(item?.scopes || agentScopes)
  const [expiresAt, setExpiresAt] = useState(item?.expires_at ? item.expires_at.slice(0, 16) : '')
  useEffect(() => {
    setName(item?.name || '새 에이전트 키')
    setScopes(item?.scopes || agentScopes)
    setExpiresAt(item?.expires_at ? item.expires_at.slice(0, 16) : '')
  }, [item])
  const toggleScope = (scope: string) => setScopes(current => current.includes(scope) ? current.filter(value => value !== scope) : [...current, scope])
  const save = useMutation({
    mutationFn: () => api<CreatedApiKey | ApiKey>(item ? `/api/v1/me/api-keys/${item.id}` : '/api/v1/me/api-keys', {
      method: item ? 'PATCH' : 'POST',
      body: JSON.stringify({ name, scopes, expires_at: expiresAt ? new Date(expiresAt).toISOString() : null }),
    }),
    onSuccess: onSaved,
  })
  return (
    <div className="modal-backdrop">
      <div className="modal api-key-modal" role="dialog" aria-modal="true" aria-label={item ? 'API 키 수정' : '새 API 키'}>
        <h2>{item ? 'API 키 수정' : '새 API 키'}</h2>
        <p>이름, 만료 시점과 에이전트가 사용할 최소 scope를 선택하세요.</p>
        <label>키 이름<input value={name} onChange={event => setName(event.target.value)} /></label>
        <label>만료 시점 (선택)<input type="datetime-local" value={expiresAt} onChange={event => setExpiresAt(event.target.value)} /></label>
        <fieldset className="scope-selector">
          <legend>허용 scope</legend>
          {agentScopes.map(scope => <label key={scope}><input type="checkbox" checked={scopes.includes(scope)} onChange={() => toggleScope(scope)} /><code>{scope}</code></label>)}
        </fieldset>
        {save.error && <p className="error-text">{save.error instanceof Error ? save.error.message : 'API 키를 저장하지 못했습니다.'}</p>}
        <div className="modal-actions"><button className="secondary" onClick={onClose}>취소</button><button className="primary" disabled={!name.trim() || scopes.length === 0 || save.isPending} onClick={() => save.mutate()}>{item ? '변경 저장' : '키 생성'}</button></div>
      </div>
    </div>
  )
}
