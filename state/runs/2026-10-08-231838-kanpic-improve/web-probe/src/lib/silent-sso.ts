import type { AuthConfig } from '../types'

// A silent attempt is scoped to this tab's browsing session, so the flags live
// in sessionStorage rather than localStorage: a fresh tab tries again, while a
// reload after a refusal does not.
const ATTEMPTED_KEY = 'kanpic.sso.silentAttempted'
const SIGNED_OUT_KEY = 'kanpic.sso.signedOut'

// Screens that must never start a silent attempt: the login screen is where a
// loop would come from, and everything under /api, /auth, /mcp and friends is
// not a browser page at all.
const EXCLUDED_PATHS = ['/login']
const EXCLUDED_PREFIXES = ['/api/', '/auth/', '/mcp', '/ws/', '/healthz', '/print-frame', '/handoff', '/.well-known/']

function readFlag(key: string): boolean {
  try {
    return window.sessionStorage.getItem(key) === 'true'
  } catch {
    // Private modes and blocked site data throw. Reading that as "already
    // attempted" is the safe answer: the alternative is a redirect loop.
    return true
  }
}

function writeFlag(key: string, value: boolean) {
  try {
    if (value) window.sessionStorage.setItem(key, 'true')
    else window.sessionStorage.removeItem(key)
  } catch {
    // Nothing to do; readFlag already fails closed.
  }
}

/** Only same-origin paths may be returned to; anything else goes home. */
export function safeReturnTo(value: string | null | undefined): string {
  if (!value || !value.startsWith('/') || value.startsWith('//')) return '/'
  return value
}

/** Records a deliberate sign-out, which suppresses silent sign-in. */
export function markSignedOut() {
  writeFlag(SIGNED_OUT_KEY, true)
  writeFlag(ATTEMPTED_KEY, true)
}

/** Lifts the suppression once a session exists again. */
export function clearSilentSsoState() {
  writeFlag(SIGNED_OUT_KEY, false)
  writeFlag(ATTEMPTED_KEY, false)
}

export function isSilentSsoPath(pathname: string): boolean {
  if (EXCLUDED_PATHS.includes(pathname)) return false
  return !EXCLUDED_PREFIXES.some(prefix => pathname.startsWith(prefix))
}

/**
 * Decides whether to try signing in without showing a login screen.
 *
 * prompt=none either answers with a code at once or comes back with
 * login_required, so trying it on every page load would bounce the browser
 * between Keycloak and kanpic forever. Three guards stop that: one attempt per
 * tab session, no attempt after a deliberate sign-out, and the sso=none marker
 * the callback leaves in the address when it was refused.
 */
export function shouldAttemptSilentSso({ config, pathname, search }: { config: AuthConfig | undefined; pathname: string; search: string }): boolean {
  if (!config?.oidc_enabled || !config.auto_login) return false
  if (!isSilentSsoPath(pathname)) return false
  const marker = new URLSearchParams(search).get('sso')
  if (marker === 'none' || marker === 'error') return false
  if (readFlag(SIGNED_OUT_KEY)) return false
  if (readFlag(ATTEMPTED_KEY)) return false
  return true
}

/** Address the browser is sent to for a silent attempt that returns to `returnTo`. */
export function silentSsoUrl(returnTo: string): string {
  return `/auth/login?prompt=none&return_to=${encodeURIComponent(safeReturnTo(returnTo))}`
}

/**
 * Sends the browser to Keycloak as a top-level navigation. A hidden iframe
 * would break under third-party cookie blocking and depends on the provider
 * allowing frames; a plain redirect needs neither.
 */
export function beginSilentSso(returnTo: string) {
  // Marked before leaving so a re-render during the navigation cannot start a
  // second attempt.
  writeFlag(ATTEMPTED_KEY, true)
  window.location.assign(silentSsoUrl(returnTo))
}
