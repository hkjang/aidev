## Offline image

- Image: `qurio:v1.4.14`
- Archive: `qurio-v1.4.14.tar.gz`

Only the compressed Docker image archive is attached to this release.
The tag-triggered Offline release workflow builds and verifies the archive and
adds its SHA-256 to the published release notes.

## What's Changed
* auto-improve: test(web): wait for loaded state instead of racing the mocked response by @hkjang in https://github.com/hkjang/qurio/pull/34
* auto-improve: perf(sqlsafe): deduplicate violation reasons through a membership index by @hkjang in https://github.com/hkjang/qurio/pull/35

SQL safety analysis now uses a membership index to avoid repeatedly scanning
all previously collected violation reasons, while preserving their order and
existing safety decisions for PostgreSQL and Oracle queries. Web component
tests wait for loaded responses before interacting with controls or checking
response-dependent content.

**Full Changelog**: https://github.com/hkjang/qurio/compare/v1.4.12...v1.4.14
