# v1.2.24 release validation

- Local release commit: a7b64fe; annotated tag: v1.2.24. No remote writes.
- Includes changes from PR #28 and PR #29 since v1.2.23.
- make lint and version metadata validation: PASS.
- Dedicated disposable PostgreSQL 16: go test -race -count=1 ./... PASS (server 83.204s).
- Pinned Keycloak 26.7.2: TestKeycloakOIDCAuthorizationCodeLogin -race -count=1 -v PASS (not skipped).
- Web tests: 18 files, 79 tests PASS. make build PASS.
- package-offline.sh and verify-offline.sh PASS: linux/amd64, OCI version v1.2.24. This local validation build used the pre-release commit; CI rebuilds from the release tag for publication.
- GitHub Pages exact 44 screenshot reference validation: PASS.
- Build-generated tracked UI bundle changes were restored to match previous release conventions; Docker builds the UI from source.
- release.yml creates GitHub Release and both assets on tag push, so github_release=false and assets=[].
- Requested marketing:product-launch and technology:release-and-deployment skills and Skill tool were not available in the provided tool catalog, local skill search, or resource catalog. Their specific procedures could not be applied.
