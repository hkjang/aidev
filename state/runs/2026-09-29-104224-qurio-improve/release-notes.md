## Offline image

- Image: `qurio:v1.4.9`
- Archive: `qurio-v1.4.9.tar.gz`
- SHA-256: `7100e01691fa9c5ae88147c024c4674ef074bee74feade9b3a31f3777ab47cfd`

Only the compressed Docker image archive is attached to this release.

## What's Changed
* Fix PostgreSQL `WITH ORDINALITY` column aliases being rejected as disallowed function calls. Queries such as `FROM unnest(...) WITH ORDINALITY g(value,position)` now validate and execute with pagination. The exception requires both the ordinality suffix and a FROM-item context; the function allowlist remains unchanged.
* Includes the preceding FROM-item context fix for dotted PostgreSQL alias column lists since v1.4.7.

**Full Changelog**: https://github.com/hkjang/qurio/compare/v1.4.7...v1.4.9
