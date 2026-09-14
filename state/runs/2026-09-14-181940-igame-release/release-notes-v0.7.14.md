Ships as a patch release the one feature landed since v0.7.13. The games, the
published content, the API contract and the runtime environment are unchanged;
RealmGuard content 0.3.1 and Defense Series content 0.4.0 ride along untouched.
One migration rides with it, 010_silent_sso.sql, which adds the column that
records whether an OIDC flow was started silently; the Go binary applies it on
start as it does every other embedded migration.

Someone already signed in to Keycloak still met the login screen on every
visit. The new OIDC setting auto_login (off by default, a switch on the OIDC
card of the administrator settings screen) lets the portal ask the provider once
with prompt=none before it draws anything: an existing session answers with a
code and the ordinary flow finishes, no session answers with login_required and
the login screen appears as before. Three guards keep it from looping: one
attempt per tab session in sessionStorage, suppression after a deliberate
sign-out until a session exists again, and the sso=none marker the callback
puts in the address so a cleared store still cannot retry; a store that cannot
be read counts as already attempted. Nothing starts from /login or a non-screen
path, and the attempt is a top-level navigation rather than a hidden iframe.
The server decides, not the address: prompt=none is only forwarded when
auto_login is on and is otherwise silently an ordinary login, the callback lands
a silent refusal on /login?sso=none&return_to=... with only a path on this
service and consumes the state either way, and GET /api/v1/public/config
publishes oidc_auto_login only alongside an enabled provider. The default is
off, so a deployment that never saved the setting behaves as before.

Moves VERSION, the Compose image tag and the web and SDK package versions
together as the release contract requires, and rebuilds the two enterprise
manuals so their covers carry v0.7.14. The administrator guide's load, inspect
and `/readyz` examples move to v0.7.14 with them; the sentence recording which
service the screens were captured from keeps naming v0.7.11, which is the build
that was photographed, and the two guide PDFs are left as published because they
come from the guide tool rather than from `make docs-pdf`.

