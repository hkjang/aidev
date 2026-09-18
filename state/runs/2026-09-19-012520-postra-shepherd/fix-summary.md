# Fix summary — PR #14 gosec failure

- **Problem:** CI job `gosec (medium+)` failed on a single G101 (HIGH, LOW confidence) finding at `internal/platform/notifymail/notifymail.go:27`: the constant `EventSyncCredentialError = "sync_credential_error"` matches gosec's hardcoded-credential identifier heuristic. It is an event-name enum value (the `mail.notify_<event>` switch suffix), not a secret — a false positive, identical in kind to the already-annotated `AccountCredentialError` in `internal/domain/account.go`.
- **Reproduced locally** with the CI's exact command (`gosec@v2.28.0 -severity medium -exclude-dir=scripts ./...`): 1 issue, that line only.
- **Fix (commit f51281a):** added the repo-conventional inline `// #nosec G101 -- event name enum value, not a credential` annotation on that line. No behaviour change.
- **Verified:** gosec now exits 0 with 0 issues; `gofmt -l` clean; `go build ./...` and `go test ./internal/platform/notifymail/... ./internal/application/...` pass.
