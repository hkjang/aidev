Ships as a patch release the one feature landed since v0.7.12. The games, the
published content, the schema and the runtime environment are unchanged; no
migration rides with it, and RealmGuard content 0.3.1 and Defense Series
content 0.4.0 ride along untouched. The API contract grows by the tracking
endpoints listed in docs/api.md; nothing that existed changes shape.

Administrators can now attach a visitor tracking snippet from /admin/tracking
instead of rebuilding the image. The portal's content security policy is locked
to script-src 'self', so a pasted snippet was silently blocked with nothing to
tell the administrator why, and there was no place in the shell to put one. The
new internal/tracking package renders the snippet for momento, ga4, gtm, matomo
and a pasted custom snippet, applies a per-request nonce to every <script> tag,
reads the http(s) origins a snippet names and keeps a bounded list of the
origins the browser reports as blocked. securityHeaders adds the nonce, the
snippet's origins and a report-uri only while tracking is active on that page;
'unsafe-inline' never enters script-src and turning tracking off restores the
old policy byte for byte, which a test proves against the previous header
string. Non-page paths get default-src 'none'. Momento goes through a
same-origin proxy by default, so no external origin appears in the policy; the
proxy strips cookies and bearer keys and answers 404 unless that setup is
active. Blocked origins reported to POST /api/v1/tracking/csp-report show up on
the admin screen and are allowed with one click, audited like any other setting
change. The default is off, so a deployment that never saved the setting sends
the same policy and the same shell as before.

Moves VERSION, the Compose image tag and the web and SDK package versions
together as the release contract requires, and rebuilds the two enterprise
manuals so their covers carry v0.7.13. The administrator guide's load, inspect
and `/readyz` examples move to v0.7.13 with them; the sentence recording which
service the screens were captured from keeps naming v0.7.11, which is the build
that was photographed, and the two guide PDFs are left as published because they
come from the guide tool rather than from `make docs-pdf`.

