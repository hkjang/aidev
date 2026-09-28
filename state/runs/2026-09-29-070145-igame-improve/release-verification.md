# v0.7.24 release verification

- Launch tier: three; release notes only. Audience: API PostgreSQL test contributors. No customer behavior or pricing change.
- Release owner: hkjang. Any required check failure aborts local tagging; the tag workflow must pass before publication.
- Local gates passed: make deps, check-contract, lint, test (SDK 9; web 246), test-race, build, both full npm audits, govulncheck v1.6.0, docs-pdf, git diff --check.
- PostgreSQL 17: make test-db passed (API 25.471s; database 2.076s); TestMigratedPoolIsolation -race -count=3 passed (3.874s).
- Cleanup: zero api_test_/migrate_ schemas; pgcrypto remains in extensions; disposable container removed.
- Two enterprise PDFs rebuilt; separate user/admin guide PDFs and screenshot provenance retained.
- No external announcement, deployment, remote push or publication performed. Independent outside-team first-run review was not available in this unattended session; this change affects test fixtures only.
- Tag workflow owns Docker archive, SBOM/checksum evidence, image/browser gates and GitHub Release creation. These remote gates await the runner's tag push; no local upload assets are required.
- Follow-through: runner/maintainer verifies tag workflow success and its single archive asset. Success signal: independent fixtures and clean schema cleanup remain passing in subsequent database runs.
- Rollback before publication: withhold the candidate tag push. Runtime fallback remains v0.7.23; this release introduces no database migration.
