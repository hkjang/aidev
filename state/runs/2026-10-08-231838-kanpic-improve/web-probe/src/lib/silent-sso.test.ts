import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { AuthConfig } from '../types'
import { beginSilentSso, clearSilentSsoState, isSilentSsoPath, markSignedOut, shouldAttemptSilentSso, silentSsoUrl } from './silent-sso'

const autoLogin: AuthConfig = { oidc_enabled: true, bootstrap_login_enabled: false, issuer_url: 'https://id.example/realms/company', client_id: 'kanpic', client_secret_configured: false, auto_login: true }

function decide(overrides: Partial<AuthConfig> = {}, path = '/workbooks/abc') {
  const [pathname = '/', search = ''] = path.split('?')
  return shouldAttemptSilentSso({ config: { ...autoLogin, ...overrides }, pathname, search: search ? `?${search}` : '' })
}

describe('silent SSO rules', () => {
  beforeEach(() => sessionStorage.clear())
  afterEach(() => { sessionStorage.clear(); vi.restoreAllMocks() })

  it('attempts only when the administrator turned auto_login on', () => {
    expect(decide()).toBe(true)
    expect(decide({ auto_login: false })).toBe(false)
    expect(decide({ auto_login: undefined })).toBe(false)
    expect(decide({ oidc_enabled: false })).toBe(false)
    expect(shouldAttemptSilentSso({ config: undefined, pathname: '/', search: '' })).toBe(false)
  })

  it('never retries in the same tab session once an attempt was made', () => {
    const assign = vi.fn()
    vi.spyOn(window, 'location', 'get').mockReturnValue({ ...window.location, assign } as Location)
    expect(decide()).toBe(true)
    beginSilentSso('/workbooks/abc?sheet=2')
    expect(assign).toHaveBeenCalledWith('/auth/login?prompt=none&return_to=%2Fworkbooks%2Fabc%3Fsheet%3D2')
    // A reload after Keycloak refused must not bounce again.
    expect(decide()).toBe(false)
    clearSilentSsoState()
    expect(decide()).toBe(true)
  })

  it('honours the refusal marker the callback leaves in the address', () => {
    expect(decide({}, '/?sso=none')).toBe(false)
    expect(decide({}, '/?sso=error')).toBe(false)
    expect(decide({}, '/?tab=1')).toBe(true)
  })

  it('stays quiet after a deliberate sign-out until a session exists again', () => {
    markSignedOut()
    expect(decide()).toBe(false)
    clearSilentSsoState()
    expect(decide()).toBe(true)
  })

  it('never starts from the login screen or from non-page paths', () => {
    expect(isSilentSsoPath('/')).toBe(true)
    expect(isSilentSsoPath('/workbooks/x')).toBe(true)
    for (const path of ['/login', '/api/v1/session', '/auth/callback', '/mcp', '/ws/workbooks/x', '/healthz', '/print-frame', '/.well-known/oauth-protected-resource']) {
      expect(isSilentSsoPath(path), path).toBe(false)
    }
  })

  it('fails closed when session storage is unavailable', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('blocked') })
    expect(decide()).toBe(false)
  })

  it('only returns to same-origin paths', () => {
    expect(silentSsoUrl('//evil.example/')).toBe('/auth/login?prompt=none&return_to=%2F')
    expect(silentSsoUrl('https://evil.example/')).toBe('/auth/login?prompt=none&return_to=%2F')
    expect(silentSsoUrl('/admin')).toBe('/auth/login?prompt=none&return_to=%2Fadmin')
  })
})
