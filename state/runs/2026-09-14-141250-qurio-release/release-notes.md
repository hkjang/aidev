## Highlights

- Silent SSO (OIDC `prompt=none`) automatic sign-in behind the new `auth.oidc.auto_login` administrator setting (off by default). When enabled, an unauthenticated visit makes one top-level `prompt=none` request to the provider and returns to the original deep link; a `login_required` refusal lands on `/login?sso=none` instead of an error.
- Loop guards: the server only honours `prompt=none` while the setting is on and records silent attempts on the one-time authorization state; the SPA tries at most once per tab session, suppresses attempts after a deliberate sign-out, fails closed when `sessionStorage` is unreadable, and never starts from callback, login, API, MCP, or health paths. `return_to` accepts only same-origin paths.
- Migration `0033_oidc_silent_login.sql` (`qurio_oidc_states.silent`, setting definition) and `0034_release_1_4_0.sql` (release identity).
- Administrator UI switch and `docs/guides/admin-guide.md` documentation.
- Base image updates: golang 1.27.1-alpine, alpine 3.24, node digest bump.

**Full Changelog**: https://github.com/hkjang/qurio/compare/v1.3.1...v1.4.0
