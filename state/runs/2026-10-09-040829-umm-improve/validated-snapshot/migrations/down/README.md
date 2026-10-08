# Rollback scripts

Each `NNN_*.down.sql` reverses the migration of the same number. They exist so a
failed deployment has a path back that is not "restore last night's backup".

They are **not** embedded in the binary and are never applied automatically: a
rollback is a deliberate operator action, taken with a backup in hand, because
dropping a column drops the data in it.

```bash
# Roll back one migration, newest first.
psql "$POSTGRES_DSN" -v ON_ERROR_STOP=1 -f migrations/down/007_scale_and_security.down.sql
```

`scripts/migrate-dry-run.sh` applies every up migration to a scratch database and
then every down script in reverse, so both directions are exercised in CI before
a release ships.

Migrations 001–006 predate this directory. They created the schema from nothing,
so their rollback is dropping the database; a down script that silently discarded
every note would be more dangerous than useful, and none is provided.
