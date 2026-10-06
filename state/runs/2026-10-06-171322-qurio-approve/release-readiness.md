# Qurio v1.4.14 release handoff

Scope: Tier three improvement; release notes only. Existing Qurio users running
PostgreSQL or Oracle SQL safety validation benefit from reason deduplication.
No new audience, pricing, packaging, or support procedure is introduced.

Publication owner: the external runner pushes the local release commit and
annotated tag; `.github/workflows/release.yml` verifies, builds the linux/amd64
image, reloads and smoke-tests its exact archive, and creates the GitHub Release
with only that archive. GitHub Pages follows the main-branch docs update.
Local assets are therefore intentionally empty and github_release is false.

Abort criteria: any required local verification failure prevents creation of the
release commit/tag; the workflow must also pass all verification and archive
smoke checks before publication. Existing release scripts do not push packages.

Database change: migration 0048 updates release identity only and retains prior
history; no application tables or behavior are changed by this release commit.
The previous published image is qurio:v1.4.12. An operational rollback needs an
explicit review of database release identity; do not remove migration history.

Post-publication checks belong to the runner/operator: successful release
workflow, one archive, matching SHA-256, readiness and version after installation.
For an initial operator-controlled installation, require zero readiness,
login, or SQL-verdict regressions before expanding rollout; monitor SQL
validation latency and initial support reports. No adoption measurements or
external-user fresh-start review are claimed during this local-only session.

## Historical evidence

- Initial HEAD: af1304dc0e010147a8248f2a8a5974fb041702c8, detached and clean.
- Latest public release/tag: v1.4.12; source already contains the 1.4.13 release contract. Next patch: 1.4.14.
- Three inspected tags (v1.4.12, v1.4.10, v1.4.7) are annotated with Qurio vVERSION; their commits use chore: release Qurio vVERSION.
- Current-version files: VERSION, compose.yml, README.md, web/package.json, web/package-lock.json, docs/index.html, docs/gallery.html, docs/requirements.md, docs/guides/offline-install.md, web/src/App.test.tsx, web/e2e/application.spec.ts, generated internal/webui/dist, and the new identity migration.
- Go version defaults remain dev/unknown and are injected at build time. The web version comes from package.json at build time. Historical migrations are retained.
- No tracked CHANGELOG or standalone release notes: published notes are English Offline image plus generated What's Changed, with a Full Changelog link.
- Prior public asset: one qurio-vVERSION.tar.gz. scripts/release.sh and Makefile release-archive provide local generation, while the tag workflow builds/uploads the same archive automatically.
- No remote write, branch creation, author override, or commit trailer is authorized or performed.

## Completed verification

- make build: passed; embedded web output exactly matches web/dist; binary reports Qurio v1.4.14.
- make check-release-contract, make lint, and git diff --check: passed.
- npm test --prefix web -- --run: 27 files and 134 tests passed.
- npm audit --prefix web --omit=dev --audit-level=high: zero vulnerabilities.
- Isolated workflow-pinned PostgreSQL and Oracle services: fresh installation, migration bootstrap, live Oracle tests, go test -race -p=1 -tags=integration ./..., and go vet -tags=integration ./... all passed; containers removed.
- Workflow-pinned govulncheck v1.6.0 and gosec v2.28.0: passed.
- Initial Playwright attempt could not launch because the session HOME browser cache was empty. Retrying with the already installed matching Chromium via PLAYWRIGHT_BROWSERS_PATH; no source changes were needed.
- Docker archive generation/reload smoke and remote publication are performed by the tag workflow; no local artifact is handed to the runner, per the requested workflow exception.
- Playwright rerun: all 14 desktop/mobile tests passed (2.2 minutes), including every route and direct reload.
- Final local commit: cbbf578d533f7a27039609a63bc1bd88a75c52ca; annotated tag v1.4.14; author/committer/tagger hkjang; clean detached checkout.
