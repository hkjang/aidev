v0.1.31 releases the account security notifications merged after v0.1.30.
The "security" notification type was already wired as mandatory -
notificationInAppEnabled cannot switch it off, notificationBatched keeps it
out of digests, and an enabled email channel sends it at once - and the user
guide promised that approval and security notices cannot be disabled, but no
code path ever produced one.

The new security_notice.go adds notifySecurity, which records the notice for
the account owner with no actor so that enqueueNotification's
self-notification guard does not drop it: the point of a security notice is
reaching the owner even when the request looked like their own. Four events
raise it - the owner's own password change (password_changed), an
administrator reset (password_reset, delivered to the target user only and
without the administrator's address), API/MCP key creation (api_key_created)
and rotation (api_key_rotated). The body names the key and the trusted-proxy
resolved request IP, and decorateNotification titles it "계정 보안" and links
payload.event to /settings/keys or /settings/security so the notification
centre and the "[service] 계정 보안" email point at the same screen. Like an
audit row, an outbox enqueue failure after the committed change is logged as
a warning rather than undone. The notification centre shows a ShieldAlert
icon for the type and keeps Bell for unknown types. The user guide gains a
paragraph on the notice in section 3.5, and api/openapi.yaml documents the
type and its event values on Notification.type.

This commit itself carries only the version bump: VERSION, the OpenAPI info
block, the package manifests and their lockfiles, the SPA fallback version,
the Dockerfile build args, the OG card and the screenshot manifest, plus the
version strings the published docs, the Markdown guides and READMEs quote.
The PDF prints of the two guides are left as they are because the repository
carries no tool to regenerate them. The README release paragraph now names
the account security notifications.

Verified with make fmt, make check, go build, go test -race against a
throwaway PostgreSQL and go vet across the backend, staticcheck clean, and
the frontend at 249 passing tests, ESLint within its 40 warning budget, and
a v0.1.31 production build.

