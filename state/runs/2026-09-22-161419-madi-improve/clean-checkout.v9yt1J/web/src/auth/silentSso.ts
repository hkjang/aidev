// Silent SSO (OIDC prompt=none): sign a visitor in without a login screen when
// the identity provider still has a session. Everything here exists to make
// sure that attempt happens at most once per tab, because a refused prompt=none
// that is retried on every page load bounces the browser between the provider
// and the app forever.

// sessionStorage rather than localStorage: a fresh tab tries again, while a
// reload after a refusal does not.
const ATTEMPTED_KEY = "madi.sso.silentAttempted";
const SIGNED_OUT_KEY = "madi.sso.signedOut";

type FlagStorage = Pick<Storage, "getItem" | "setItem" | "removeItem">;

export type SilentSsoEnv = {
  storage: () => FlagStorage;
  pathname: string;
  search: string;
};

function browserEnv(): SilentSsoEnv {
  return {
    storage: () => window.sessionStorage,
    pathname: window.location.pathname,
    search: window.location.search,
  };
}

function readFlag(env: SilentSsoEnv, key: string): boolean {
  try {
    return env.storage().getItem(key) === "true";
  } catch {
    // Private modes and blocked site data throw. Treating that as "already
    // attempted" fails closed, which is the only safe answer for a loop guard.
    return true;
  }
}

function writeFlag(env: SilentSsoEnv, key: string, value: boolean) {
  try {
    if (value) env.storage().setItem(key, "true");
    else env.storage().removeItem(key);
  } catch {
    /* readFlag already fails closed when storage is unavailable */
  }
}

/** Paths where a silent attempt must never start: callback, login, and non-browser routes. */
export function silentSsoPathAllowed(pathname: string): boolean {
  if (pathname === "/login" || pathname.startsWith("/login/")) return false;
  for (const prefix of ["/api/", "/mcp", "/healthz", "/readyz", "/share/"]) {
    if (pathname.startsWith(prefix)) return false;
  }
  return true;
}

/** Records a deliberate sign-out so the next page load does not sign back in. */
export function markSignedOut(env: SilentSsoEnv = browserEnv()) {
  writeFlag(env, SIGNED_OUT_KEY, true);
  writeFlag(env, ATTEMPTED_KEY, true);
}

/** Lifts the suppression once a session exists again. */
export function clearSilentSsoState(env: SilentSsoEnv = browserEnv()) {
  writeFlag(env, SIGNED_OUT_KEY, false);
  writeFlag(env, ATTEMPTED_KEY, false);
}

/**
 * Decides whether to try signing in without showing the login screen. Any
 * "no" here is final for this tab session; the caller shows the login page.
 */
export function shouldAttemptSilentSso(
  info: { oidc_enabled?: unknown; oidc_auto_login?: unknown },
  env: SilentSsoEnv = browserEnv(),
): boolean {
  if (info.oidc_enabled !== true || info.oidc_auto_login !== true) return false;
  if (!silentSsoPathAllowed(env.pathname)) return false;
  // The callback appends this marker when the provider had no session, so a
  // refusal is remembered even if sessionStorage was cleared in between.
  const sso = new URLSearchParams(env.search).get("sso");
  if (sso === "none" || sso === "error") return false;
  if (readFlag(env, SIGNED_OUT_KEY)) return false;
  if (readFlag(env, ATTEMPTED_KEY)) return false;
  return true;
}

/** Only same-origin paths may be used as a return target. */
export function safeReturnTo(value: string): string {
  return value.startsWith("/") && !value.startsWith("//") ? value : "/app";
}

/** Builds the top-level navigation target for one silent attempt and marks it as used. */
export function silentSsoStartUrl(
  returnTo: string,
  env: SilentSsoEnv = browserEnv(),
): string {
  writeFlag(env, ATTEMPTED_KEY, true);
  return `/api/v1/auth/oidc/start?prompt=none&return_to=${encodeURIComponent(safeReturnTo(returnTo))}`;
}

/** Sends the browser to the provider. A top-level move, not a hidden iframe. */
export function beginSilentSso(returnTo: string) {
  window.location.assign(silentSsoStartUrl(returnTo));
}
