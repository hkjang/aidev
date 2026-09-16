## Offline image

- Image: `qurio:v1.4.1`
- Archive: `qurio-v1.4.1.tar.gz`
- SHA-256: written by the `Offline release` workflow after `docker save | gzip -n -9`

Only the compressed Docker image archive is attached to this release.

## Changes

- Overdue `PENDING_REVIEW` approval requests are now transitioned to `EXPIRED` by the hourly maintenance sweep (`store.ExpireApprovalRequests`), recording an `EXPIRED` approval event and an `approval.expired` audit entry (`details.source=scheduler`). Concurrent reviewer decisions win; each request expires at most once.
- Release identity migration `0035_release_1_4_1.sql`.

**Full Changelog**: https://github.com/hkjang/qurio/compare/v1.4.0...v1.4.1
