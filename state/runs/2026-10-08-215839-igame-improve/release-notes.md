Offline image: `igame:v0.7.32`

## What's Changed

* Stabilize public game catalog pagination for games with tied names by ordering by name and then unique ID. PostgreSQL/Router regressions cover filters, page responses and paging after a description-only update in https://github.com/hkjang/igame/pull/39.

This patch does not add a database migration or configuration requirement. RealmGuard content `0.3.1` and Defense Series content `0.4.0` are retained. Offset pagination does not provide a snapshot across inserts, deletes or changes to sorting/filter fields.

**Full Changelog**: https://github.com/hkjang/igame/compare/v0.7.31...v0.7.32
