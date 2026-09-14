release: moina v0.1.30

v0.1.30 releases the two features merged after v0.1.29: silent SSO login
against a live Keycloak session and the administrator-managed visit tracking
snippet.

The OIDC settings gain autoLogin, off by default. When it is on, the web app
sends a visitor without a session to GET /auth/oidc/login?prompt=none as a
top-level navigation, so the provider answers from its own session only: with
a session the callback creates the MOINA session and returns to the deep link
the visitor opened, and without one the callback sends the visitor to
/login?sso=none. Provider errors on a state-verified flow now redirect to
/login?sso=error instead of answering JSON 400 missing_code, and only a
returnTo that starts with / and not // is kept. The web app keeps three loop
guards - one attempt per tab session in sessionStorage, suppression after a
manual logout, and the sso marker in the address - and never attempts on
/login, /api, /auth, /mcp or the health and metrics paths. When the setting
is off the server downgrades prompt=none to an ordinary login, so the switch
is the only place that changes the flow. A successful silent login is audited
as auth.oidc.login with silent: true.

The general settings screen gains a visit tracking card. The administrator
picks a provider (Momento, GA4, GTM, Matomo or a pasted snippet) and saves
the snippet; it is injected only into screen documents while tracking is on,
with a per-request nonce, and the CSP grows by that nonce and the origins the
snippet points at. 'unsafe-inline' is never used and the default is off.
Momento is served through a same-origin proxy (/momento/*) with an
exact-authority outbound policy that forwards no cookie or Authorization
header, so no external origin appears in the policy. While tracking is on a
report-uri collects blocked origins in memory and the card adds them to the
allow list in one click. Non-screen paths receive the narrower default-src
'none' policy and no snippet. The admin-settings visual baselines were
refreshed for the new card.

This commit itself carries only the version bump: VERSION, the OpenAPI info
block, the package manifests and their lockfiles, the SPA fallback version,
the Dockerfile build args, the OG card and the screenshot manifest, plus the
version strings the published docs, the Markdown guides and READMEs quote.
The PDF prints of the two guides are left as they are because the repository
carries no tool to regenerate them. The README release paragraph now names
the silent SSO login and the visit tracking snippet.

Verified with make fmt, make check, go build, go test -race and go vet across
the backend, staticcheck clean, and the frontend at 247 passing tests, ESLint
within its 40 warning budget, and a v0.1.30 production build.

