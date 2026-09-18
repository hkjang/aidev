# Fix summary — PR #13 gosec failure

- **Problem:** the `gosec (medium+)` CI job failed with one G101 (hardcoded credentials, HIGH) on `internal/platform/notifymail/notifymail.go:27` — the constant `EventSyncCredentialError = "sync_credential_error"`. It is a notification event name (also the settings-key suffix `mail.notify_sync_credential_error`), not a secret; gosec matched on the word "credential" in the identifier. Reproduced locally with the exact CI command (`gosec -severity medium -exclude-dir=scripts -fmt sarif ./...` → exit 1).
- **Fix (commit c60def7):** added `// #nosec G101 -- event name, never a credential value` on that line, following the same convention already used in `internal/application/settings.go` for `SettingOIDCSecretRef` and friends. No code/behaviour change.
- **Verified:** the exact CI gosec command now exits 0 with 0 issues; `go build ./...`, `go vet`, and `go test ./internal/platform/notifymail/ ./internal/application/` pass.
