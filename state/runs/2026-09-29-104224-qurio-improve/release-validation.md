# Qurio v1.4.9 release validation

- Scope: Tier 3 maintenance release for PostgreSQL users of WITH ORDINALITY alias column lists; release notes only.
- Existing repository identity was 1.4.8 (commit eac2424); most recent supplied published release was v1.4.7. Next patch identity: 1.4.9.
- Release identity updated in VERSION, compose, web package/lock, version assertions, documentation, embedded SPA, and migration 0043. Historical migrations retained.
- Annotated tag and commit follow the existing Qurio release format. No remote publication performed.
- Build: make build; make check-release-contract.
- Static/unit/security: make lint; go test ./...; go vet -tags=integration ./...; govulncheck; gosec -quiet. All passed. govulncheck reported zero affected vulnerabilities; 25 module-level findings not called by this code.
- Image: scripts/release.sh passed build, version smoke check, deterministic gzip export and verify-offline-image.sh. Verified linux/amd64, empty database, 43 migrations, restart, bootstrap login, blocked egress.
- Local verification archive SHA-256: 7100e01691fa9c5ae88147c024c4674ef074bee74feade9b3a31f3777ab47cfd. This archive is retained under assets as verification evidence. The tag workflow builds and uploads its own archive; release.json assets is intentionally empty, and github_release is false.
- Live Oracle is unavailable locally; its mandatory live-service gate remains in release.yml. Production rollout and external first-use/adoption checks are not performed in this local-only task.
- Rollout gate: workflow verification must pass before publication; abort on any failed gate. Operator hkjang owns deployment decisions. If production readiness/login or valid ordinality queries fail, halt rollout and restore the previously deployed image; preserve release-history migrations.
- Post-release observation: within 7 days, operator should verify at least one affected PostgreSQL account can execute an ordinality query and monitor rejection reports. No outreach or external announcement sent.
- PostgreSQL: fresh-install and migration bootstrap passed; full race integration suite passed after restoring the CI test DSN password field required by runtimeapi input validation. No application code was changed. Current release row is 1.4.9; temporary PostgreSQL container removed.
- Web: 27 files / 134 tests passed; npm production audit found zero vulnerabilities.
- Browser E2E: all 14 tests passed.
- Final commit: 3f7cf657a9eb01de8d33460c326e6a0e4b4306b6; annotated tag v1.4.9 points to this commit; worktree clean.
