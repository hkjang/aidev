Offline image: `igame:v0.7.18`

## What's Changed
* Add PostgreSQL regression coverage for migration checksums, atomic rollback and retries in https://github.com/hkjang/igame/pull/23

Three isolated-schema tests execute the production migrator and embedded SQL, checking unchanged checksums, application timestamps and seeds on rerun; checksum tampering rejection and recovery; and rollback of migration 010 DDL and history when its history INSERT fails, preserving earlier commits and allowing retry. `make test-db` now includes the database package. README documents the dedicated pgcrypto extension schema and search path.

**Full Changelog**: https://github.com/hkjang/igame/compare/v0.7.17...v0.7.18
