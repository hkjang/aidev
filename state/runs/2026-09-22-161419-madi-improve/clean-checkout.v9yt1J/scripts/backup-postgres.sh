#!/usr/bin/env bash
set -euo pipefail
source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/postgres-common.sh"

usage() {
  printf '%s\n' \
    'Usage: bash scripts/backup-postgres.sh --dsn PASSWORD_FREE_URI --attachments ABSOLUTE_DIRECTORY --output NEW_BACKUP_DIRECTORY --acknowledge WRITES_STOPPED [--source-storage-path CONTAINER_DIRECTORY]' \
    'Stop all madi writers before running. Use .pgpass / PGPASSFILE for credentials.' \
    'This helper supports legacy local attachments only; configured storage providers require separate coordinated object-store backup.' \
    'The output contains PostgreSQL data, encrypted settings, legacy local attachments and checksums.' \
    'ENCRYPTION_KEY is never included. Preserve it in a separate secret store.'
}
MADI_SOURCE_DSN=""; MADI_ATTACHMENTS=""; MADI_OUTPUT=""; MADI_ACK=""; MADI_SOURCE_STORAGE=""
while (($#)); do
  case "$1" in
    --help|-h) usage; exit 0 ;;
    --dsn|--attachments|--output|--acknowledge|--source-storage-path)
      (($# >= 2)) || madi_fail "Missing value for $1"
      case "$1" in
        --dsn) MADI_SOURCE_DSN="$2" ;;
        --attachments) MADI_ATTACHMENTS="$2" ;;
        --output) MADI_OUTPUT="$2" ;;
        --acknowledge) MADI_ACK="$2" ;;
        --source-storage-path) MADI_SOURCE_STORAGE="$2" ;;
      esac
      shift 2 ;;
    *) madi_fail 'Unknown option; run --help.' ;;
  esac
done
[[ -n "$MADI_SOURCE_DSN" && -n "$MADI_ATTACHMENTS" && -n "$MADI_OUTPUT" ]] || { usage; exit 1; }
[[ "$MADI_ACK" == WRITES_STOPPED ]] || madi_fail 'Stop all madi writers, then pass --acknowledge WRITES_STOPPED.'
madi_passwordless_dsn "$MADI_SOURCE_DSN"
madi_plain_path "$MADI_ATTACHMENTS"; madi_plain_path "$MADI_OUTPUT"
for MADI_COMMAND in pg_dump pg_restore psql tar sha256sum realpath find; do madi_require "$MADI_COMMAND"; done
[[ -d "$MADI_ATTACHMENTS" ]] || madi_fail 'The attachment directory does not exist.'
MADI_ATTACHMENTS="$(realpath -- "$MADI_ATTACHMENTS")"
[[ "$MADI_ATTACHMENTS" != / ]] || madi_fail 'Choose the exact attachment directory, not the filesystem root.'
MADI_SOURCE_STORAGE="${MADI_SOURCE_STORAGE:-$MADI_ATTACHMENTS}"
madi_plain_path "$MADI_SOURCE_STORAGE"
[[ "$MADI_SOURCE_STORAGE" == /* && "$MADI_SOURCE_STORAGE" != / ]] || madi_fail 'The source storage path must be an absolute attachment directory.'
MADI_SOURCE_STORAGE="${MADI_SOURCE_STORAGE%/}"
MADI_OUTPUT="$(realpath -m -- "$MADI_OUTPUT")"
[[ ! -e "$MADI_OUTPUT" ]] || madi_fail 'The output directory must be new; existing backups are never overwritten.'
[[ -d "$(dirname -- "$MADI_OUTPUT")" ]] || madi_fail 'Create the backup parent directory first.'
[[ "$MADI_OUTPUT" != "$MADI_ATTACHMENTS/"* ]] || madi_fail 'The backup directory must be outside the attachment directory.'
[[ -z "$(find "$MADI_ATTACHMENTS" -type l -print -quit)" ]] || madi_fail 'Attachment backups do not accept symbolic links.'
MADI_SOURCE_DATABASE="$(psql -X -w --dbname="$MADI_SOURCE_DSN" --tuples-only --no-align --set=ON_ERROR_STOP=1 --command='SELECT current_database()')"
MADI_PROVIDER_FILES="$(psql -X -w --dbname="$MADI_SOURCE_DSN" --tuples-only --no-align --set=ON_ERROR_STOP=1 <<'SQL'
SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='attachments' AND column_name='storage_provider_id') AS has_storage_provider \gset
\if :has_storage_provider
SELECT count(*) FROM attachments WHERE storage_provider_id IS NOT NULL;
\else
SELECT 0;
\endif
SQL
)"
[[ "$MADI_PROVIDER_FILES" == 0 ]] || madi_fail 'Configured local/S3/MinIO attachment objects exist. This helper would be incomplete. Use coordinated pg_dump plus ALL provider roots/buckets/prefixes, or the bounded madi logical backup.'
MADI_OUTSIDE_FILES="$(psql -X -w --dbname="$MADI_SOURCE_DSN" --tuples-only --no-align --set=ON_ERROR_STOP=1 --set="attachment_root=$MADI_SOURCE_STORAGE" <<'SQL'
SELECT count(*) FROM attachments WHERE left(path, length(:'attachment_root') + 1) <> :'attachment_root' || '/';
SQL
)"
[[ "$MADI_OUTSIDE_FILES" == 0 ]] || madi_fail 'Some attachment paths are outside the selected directory; consolidate or back up those paths first.'
mkdir -m 700 -- "$MADI_OUTPUT"
printf 'Backup is incomplete until all checks pass.\n' > "$MADI_OUTPUT/INCOMPLETE"
pg_dump -w --dbname="$MADI_SOURCE_DSN" --format=custom --no-owner --no-acl --exclude-table-data='*.export_artifact_blobs' --file="$MADI_OUTPUT/postgres.dump"
pg_restore --list "$MADI_OUTPUT/postgres.dump" >/dev/null
tar --create --gzip --file="$MADI_OUTPUT/attachments.tar.gz" --directory="$MADI_ATTACHMENTS" .
tar --list --gzip --file="$MADI_OUTPUT/attachments.tar.gz" >/dev/null
printf 'format=madi-postgres-backup-v1\nservice_version=%s\ncreated_at=%s\nsource_database=%s\nsource_attachments=%s\nattachment_mode=legacy-local\nencryption_key_included=false\nwrites_stopped_acknowledged=true\n' \
  "$(madi_version)" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$MADI_SOURCE_DATABASE" "$MADI_SOURCE_STORAGE" > "$MADI_OUTPUT/backup-info.txt"
(
  cd -- "$MADI_OUTPUT"
  sha256sum postgres.dump attachments.tar.gz backup-info.txt > SHA256SUMS
  sha256sum --check --status SHA256SUMS
)
# Only remove the marker this invocation created after verification succeeds.
rm -- "$MADI_OUTPUT/INCOMPLETE"
printf 'Backup verified: %s\nENCRYPTION_KEY is not included; preserve the original key separately.\n' "$MADI_OUTPUT"
