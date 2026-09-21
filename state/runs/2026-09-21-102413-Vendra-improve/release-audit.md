# Vendra v0.7.57 release audit

- HEAD: 28cd677; merged changes: #128 and #129 (a05402d).
- v0.7.54, v0.7.55, v0.7.56 are lightweight tags on merge commits, without release commits or tag annotations. Preserved this convention; no additional commit or source edits.
- Version defaults: Makefile, web/package.json and root entries of web/package-lock.json remain 0.6.21, unchanged across recent releases. Runtime defaults in internal/httpapi/app.go are dev/unknown; release workflow injects the tag version through Docker build arguments.
- CHANGELOG.md last entry is v0.6.45 (Korean). Recent releases use generated English GitHub notes, preserved in release-notes.md. README/compose examples reference 0.6.21/0.7.20/0.7.26; guides document 0.7.49 and an upgrade to 0.7.50; older operations/index/traceability examples reference 0.3.0. These are historical examples, not release version sources.
- .github/workflows/release.yml builds and publishes vendra-vVERSION.tar.gz and creates Vendra vVERSION on v* tag push. Therefore github_release=false and assets=[]; CI recreates the tested archive.
- Executed scripts/offline-release.sh 0.7.57 locally; image inspection, nonempty archive and gzip validation passed. No uploads, pushes or registry publication.
- Go tests ran against three isolated PostgreSQL 16 databases; gofmt, go vet and git diff --check passed. Web npm ci --ignore-scripts, TypeScript, ESLint, 21 test files/93 tests and production build passed.
- marketing:product-launch and technology:release-and-deployment were unavailable in exposed tools/resources and local skill searches. Followed the user's explicit release procedure; their skill-specific return formats could not be verified.
