#!/usr/bin/env bash
set -euo pipefail
source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/postgres-common.sh"

usage() {
  printf '%s\n' \
    'Usage: bash scripts/restore-postgres.sh --backup BACKUP_DIRECTORY --target-dsn PASSWORD_FREE_URI --attachments NEW_DIRECTORY --acknowledge RESTORE_DISPOSABLE_TARGET [--target-storage-path CONTAINER_DIRECTORY]' \
    'The target database must already exist, be empty, and end in _restore_test or _restore_verify.' \
    'The attachment directory must not exist. No existing database or directory is deleted.' \
    'Run the restored service with the original ENCRYPTION_KEY and the recorded madi version.'
}
MADI_BACKUP=""; MADI_TARGET_DSN=""; MADI_TARGET_ATTACHMENTS=""; MADI_ACK=""; MADI_TARGET_STORAGE=""
while (($#)); do
  case "$1" in
    --help|-h) usage; exit 0 ;;
    --backup|--target-dsn|--attachments|--acknowledge|--target-storage-path)
      (($# >= 2)) || madi_fail "Missing value for $1"
      case "$1" in
        --backup) MADI_BACKUP="$2" ;;
        --target-dsn) MADI_TARGET_DSN="$2" ;;
        --attachments) MADI_TARGET_ATTACHMENTS="$2" ;;
        --acknowledge) MADI_ACK="$2" ;;
        --target-storage-path) MADI_TARGET_STORAGE="$2" ;;
      esac
      shift 2 ;;
    *) madi_fail 'Unknown option; run --help.' ;;
  esac
done
[[ -n "$MADI_BACKUP" && -n "$MADI_TARGET_DSN" && -n "$MADI_TARGET_ATTACHMENTS" ]] || { usage; exit 1; }
[[ "$MADI_ACK" == RESTORE_DISPOSABLE_TARGET ]] || madi_fail 'Pass --acknowledge RESTORE_DISPOSABLE_TARGET only for an isolated restoration target.'
madi_passwordless_dsn "$MADI_TARGET_DSN"
madi_plain_path "$MADI_BACKUP"; madi_plain_path "$MADI_TARGET_ATTACHMENTS"
for MADI_COMMAND in pg_restore psql tar sha256sum realpath awk sed; do madi_require "$MADI_COMMAND"; done
[[ -d "$MADI_BACKUP" ]] || madi_fail 'The backup directory does not exist.'
MADI_BACKUP="$(realpath -- "$MADI_BACKUP")"
MADI_TARGET_ATTACHMENTS="$(realpath -m -- "$MADI_TARGET_ATTACHMENTS")"
MADI_TARGET_STORAGE="${MADI_TARGET_STORAGE:-$MADI_TARGET_ATTACHMENTS}"
madi_plain_path "$MADI_TARGET_STORAGE"
[[ "$MADI_TARGET_STORAGE" == /* && "$MADI_TARGET_STORAGE" != / ]] || madi_fail 'The target storage path must be an absolute attachment directory.'
MADI_TARGET_STORAGE="${MADI_TARGET_STORAGE%/}"
[[ ! -e "$MADI_TARGET_ATTACHMENTS" ]] || madi_fail 'The attachment target must be a new directory.'
[[ -d "$(dirname -- "$MADI_TARGET_ATTACHMENTS")" ]] || madi_fail 'Create the attachment parent directory first.'
[[ ! -f "$MADI_BACKUP/INCOMPLETE" ]] || madi_fail 'An incomplete backup cannot be restored.'
for MADI_FILE in postgres.dump attachments.tar.gz backup-info.txt SHA256SUMS; do
  [[ -f "$MADI_BACKUP/$MADI_FILE" && ! -L "$MADI_BACKUP/$MADI_FILE" ]] || madi_fail 'A required backup file is missing or is a symbolic link.'
done
MADI_CHECKSUM_COUNT=0
while read -r MADI_HASH MADI_FILE MADI_EXTRA; do
  [[ "$MADI_HASH" =~ ^[a-f0-9]{64}$ && -z "$MADI_EXTRA" ]] || madi_fail 'Invalid checksum manifest.'
  [[ "$MADI_FILE" == postgres.dump || "$MADI_FILE" == attachments.tar.gz || "$MADI_FILE" == backup-info.txt ]] || madi_fail 'Unexpected path in checksum manifest.'
  MADI_CHECKSUM_COUNT=$((MADI_CHECKSUM_COUNT + 1))
done < "$MADI_BACKUP/SHA256SUMS"
[[ "$MADI_CHECKSUM_COUNT" == 3 ]] || madi_fail 'The checksum manifest must contain exactly three files.'
for MADI_FILE in postgres.dump attachments.tar.gz backup-info.txt; do
  [[ "$(awk -v expected="$MADI_FILE" '$2 == expected {n++} END {print n+0}' "$MADI_BACKUP/SHA256SUMS")" == 1 ]] || madi_fail 'Duplicate or missing checksum entries.'
done
(cd -- "$MADI_BACKUP" && sha256sum --check --status SHA256SUMS) || madi_fail 'Backup checksum verification failed.'
[[ "$(sed -n 's/^format=//p' "$MADI_BACKUP/backup-info.txt")" == madi-postgres-backup-v1 ]] || madi_fail 'Unsupported backup format.'
MADI_BACKUP_VERSION="$(sed -n 's/^service_version=//p' "$MADI_BACKUP/backup-info.txt")"
[[ "$MADI_BACKUP_VERSION" == "$(madi_version)" ]] || madi_fail 'Use restoration scripts and the service image from the recorded madi version.'
MADI_SOURCE_DATABASE="$(sed -n 's/^source_database=//p' "$MADI_BACKUP/backup-info.txt")"
MADI_SOURCE_ATTACHMENTS="$(sed -n 's/^source_attachments=//p' "$MADI_BACKUP/backup-info.txt")"
madi_plain_path "$MADI_SOURCE_ATTACHMENTS"
[[ "$MADI_SOURCE_ATTACHMENTS" == /* && "$MADI_SOURCE_ATTACHMENTS" != / ]] || madi_fail 'Invalid source attachment directory.'
tar --list --gzip --file="$MADI_BACKUP/attachments.tar.gz" | while IFS= read -r MADI_ENTRY; do
  [[ "$MADI_ENTRY" != /* && "/$MADI_ENTRY/" != *'/../'* ]] || madi_fail 'Unsafe archive path.'
done
tar --list --verbose --gzip --file="$MADI_BACKUP/attachments.tar.gz" | \
  awk 'substr($0,1,1) != "-" && substr($0,1,1) != "d" {exit 1}' || madi_fail 'The archive contains links or unsupported file types.'
pg_restore --list "$MADI_BACKUP/postgres.dump" >/dev/null
MADI_TARGET_DATABASE="$(psql -X -w --dbname="$MADI_TARGET_DSN" --tuples-only --no-align --set=ON_ERROR_STOP=1 --command='SELECT current_database()')"
[[ "$MADI_TARGET_DATABASE" =~ ^[A-Za-z0-9_]+_restore_(test|verify)$ ]] || madi_fail 'The disposable target database name must end in _restore_test or _restore_verify.'
[[ "$MADI_TARGET_DATABASE" != "$MADI_SOURCE_DATABASE" ]] || madi_fail 'The restoration target must differ from the recorded source database.'
MADI_RELATIONS="$(psql -X -w --dbname="$MADI_TARGET_DSN" --tuples-only --no-align --set=ON_ERROR_STOP=1 --command="SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%' AND c.relkind IN ('r','p','v','m','f','S')")"
[[ "$MADI_RELATIONS" == 0 ]] || madi_fail 'The target database is not empty. Nothing was changed.'
mkdir -m 700 -- "$MADI_TARGET_ATTACHMENTS"
tar --extract --gzip --file="$MADI_BACKUP/attachments.tar.gz" --directory="$MADI_TARGET_ATTACHMENTS" --no-same-owner --no-same-permissions --keep-old-files
pg_restore -w --dbname="$MADI_TARGET_DSN" --single-transaction --exit-on-error --no-owner --no-privileges "$MADI_BACKUP/postgres.dump"
psql -X -w --dbname="$MADI_TARGET_DSN" --quiet --set=ON_ERROR_STOP=1 \
  --set="source_storage=$MADI_SOURCE_ATTACHMENTS" --set="target_storage=$MADI_TARGET_STORAGE" <<'SQL'
BEGIN;
UPDATE attachments SET path = :'target_storage' || substring(path FROM length(:'source_storage') + 1)
 WHERE left(path, length(:'source_storage') + 1) = :'source_storage' || '/';
UPDATE settings SET data = jsonb_set(data, '{storage_path}', to_jsonb(:'target_storage'::text), true) WHERE id=1;
DELETE FROM sessions;
DO $$ BEGIN
 IF to_regclass('job_settings') IS NOT NULL THEN UPDATE job_settings SET paused=true; END IF;
 IF to_regclass('automation_rules') IS NOT NULL THEN UPDATE automation_rules SET enabled=false; END IF;
 IF to_regclass('automation_webhooks') IS NOT NULL THEN UPDATE automation_webhooks SET enabled=false; END IF;
 IF to_regclass('automation_jobs') IS NOT NULL THEN
  UPDATE automation_jobs SET status='cancelled',cancel_requested=true,lease_id=NULL,lease_until=NULL,last_error='운영 백업 복원으로 중단되었습니다. 관리자가 작업을 확인하세요',finished_at=now() WHERE status IN ('pending','running');
 END IF;
 IF to_regclass('backup_policy') IS NOT NULL THEN UPDATE backup_policy SET enabled=false,next_run=NULL; END IF;
 IF to_regclass('storage_providers') IS NOT NULL THEN UPDATE storage_providers SET enabled=false; END IF;
 IF to_regclass('storage_settings') IS NOT NULL THEN UPDATE storage_settings SET provider_id=NULL; END IF;
 IF to_regclass('storage_assignments') IS NOT NULL THEN UPDATE storage_assignments SET provider_id=NULL; END IF;
 IF to_regclass('backup_artifacts') IS NOT NULL THEN UPDATE backup_artifacts SET managed=false,last_error='복원 전 보관소의 이력입니다. 자동 보존 정리에서 제외됩니다'; END IF;
 IF to_regclass('collaboration_presence') IS NOT NULL THEN DELETE FROM collaboration_presence; END IF;
 IF to_regclass('document_collaboration') IS NOT NULL THEN
  IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='document_collaboration' AND column_name='snapshot_sequence') THEN
   UPDATE document_collaboration c SET epoch=gen_random_uuid(),sequence=0,state=''::bytea,projected_markdown=d.markdown,document_version=d.version,snapshot_sequence=0,snapshot_at=now(),head_hash='',reset_reason='backup_restored',updated_at=now() FROM documents d WHERE d.id=c.document_id;
  ELSE
   UPDATE document_collaboration c SET epoch=gen_random_uuid(),sequence=0,state=''::bytea,projected_markdown=d.markdown,document_version=d.version,updated_at=now() FROM documents d WHERE d.id=c.document_id;
  END IF;
 END IF;
 IF to_regclass('collaboration_updates') IS NOT NULL THEN DELETE FROM collaboration_updates; END IF;
 IF to_regclass('workspace_settings') IS NOT NULL THEN UPDATE workspace_settings SET data=jsonb_set(data,'{lifecycle_enabled}','false'::jsonb,true); END IF;
 IF to_regclass('migration_imports') IS NOT NULL THEN UPDATE migration_imports SET status='cancelled',source_data=NULL WHERE status<>'completed'; END IF;
 IF to_regclass('migration_sessions') IS NOT NULL THEN UPDATE migration_sessions SET status='cancelled',revision=revision+1,error='백업 복원으로 이관을 중단했습니다' WHERE status<>'completed'; UPDATE migration_sessions SET request_session_hash='',request_token_hash='',purged_at=now(); UPDATE migration_session_items SET prepared_data=NULL; DELETE FROM migration_session_chunks; END IF;
 IF to_regclass('migration_session_objects') IS NOT NULL THEN UPDATE migration_session_objects SET status='discarded'; UPDATE migration_session_items SET staged_object='{}',storage_fingerprint=''; END IF;
 IF to_regclass('knowledge_evidence_policy') IS NOT NULL THEN UPDATE knowledge_evidence_policy SET enabled=false,version=version+1; END IF;
 IF to_regclass('knowledge_package_policy') IS NOT NULL THEN UPDATE knowledge_package_policy SET enabled=false,version=version+1; END IF;
 IF to_regclass('knowledge_evidence_policy_history') IS NOT NULL THEN INSERT INTO knowledge_evidence_policy_history(version,enabled,retention_days) SELECT version,enabled,retention_days FROM knowledge_evidence_policy WHERE id=1 ON CONFLICT DO NOTHING; END IF;
 IF to_regclass('knowledge_package_policy_history') IS NOT NULL THEN INSERT INTO knowledge_package_policy_history(version,enabled,retention_hours,token_counter,allow_http) SELECT version,enabled,retention_hours,token_counter,allow_http FROM knowledge_package_policy WHERE id=1 ON CONFLICT DO NOTHING; END IF;
 IF to_regclass('knowledge_distribution_policy') IS NOT NULL THEN UPDATE knowledge_distribution_policy SET enabled=false,revision=revision+1; END IF;
 IF to_regclass('knowledge_distribution_policy_history') IS NOT NULL THEN INSERT INTO knowledge_distribution_policy_history(revision,enabled,max_valid_days) SELECT revision,enabled,max_valid_days FROM knowledge_distribution_policy WHERE id=1 ON CONFLICT DO NOTHING; END IF;
 IF to_regclass('knowledge_distribution_keys') IS NOT NULL THEN UPDATE knowledge_distribution_keys SET revoked_at=coalesce(revoked_at,now()),revision=revision+1; END IF;
 IF to_regclass('knowledge_distribution_exports') IS NOT NULL THEN UPDATE knowledge_distribution_exports SET status='expired' WHERE status IN ('queued','awaiting_review','ready'); END IF;
 IF to_regclass('knowledge_distribution_artifacts') IS NOT NULL THEN DELETE FROM knowledge_distribution_artifacts; END IF;
 IF to_regclass('attachment_extraction_settings') IS NOT NULL THEN UPDATE attachment_extraction_settings SET data=data||'{"enabled":false,"ocr_enabled":false}'::jsonb,revision=revision+1; END IF;
 IF to_regclass('attachment_extraction_policy_history') IS NOT NULL THEN INSERT INTO attachment_extraction_policy_history(revision,data) SELECT revision,data FROM attachment_extraction_settings WHERE id=1 ON CONFLICT DO NOTHING; END IF;
 IF to_regclass('attachment_extractions') IS NOT NULL THEN UPDATE attachment_extractions SET status=CASE WHEN status IN ('queued','running','ready') THEN 'obsolete' ELSE status END,revision=revision+1,session_hash='',token_hash='',temp_path=''; END IF;
 IF to_regclass('attachment_extraction_heads') IS NOT NULL THEN DELETE FROM attachment_extraction_heads; END IF;
 IF to_regclass('attachment_extraction_fragments') IS NOT NULL THEN DELETE FROM attachment_extraction_fragments; END IF;
 IF to_regclass('system_status_policy') IS NOT NULL THEN UPDATE system_status_policy SET enabled=false,revision=revision+1,updated_at=now(); END IF;
 IF to_regclass('system_status_policy_history') IS NOT NULL THEN INSERT INTO system_status_policy_history(revision,policy) SELECT revision,to_jsonb(p)-'id'-'updated_at' FROM system_status_policy p WHERE id=1 ON CONFLICT DO NOTHING; END IF;
 IF to_regclass('system_status_cards') IS NOT NULL THEN UPDATE system_status_cards SET verification_epoch=verification_epoch+1,revision=revision+1,updated_at=now(); END IF;
 IF to_regclass('knowledge_impact_exceptions') IS NOT NULL THEN UPDATE knowledge_impact_exceptions SET status='cancelled',updated_at=clock_timestamp() WHERE status IN ('pending','approved'); END IF;
 IF to_regclass('knowledge_questions') IS NOT NULL THEN UPDATE knowledge_questions SET state='proposed',revision=revision+1,confirmed_by=NULL,confirmed_at=NULL WHERE state='confirmed'; END IF;
 IF to_regclass('knowledge_path_progress') IS NOT NULL THEN UPDATE knowledge_path_progress SET state='needs_recheck',revision=revision+1,approval_id=NULL,updated_at=clock_timestamp() WHERE state IN ('confirmed','pending_review','approved'); END IF;
 IF to_regclass('knowledge_structured_drafts') IS NOT NULL THEN UPDATE knowledge_structured_drafts SET state='expired',revision=revision+1,expires_at=now() WHERE state='draft'; END IF;
 IF to_regclass('connector_settings') IS NOT NULL THEN UPDATE connector_settings SET enabled=false; UPDATE connector_configs SET enabled=false,next_run=NULL; END IF;
 IF to_regclass('sql_sources') IS NOT NULL THEN UPDATE sql_sources SET enabled=false; END IF;
 IF to_regclass('approval_policy_clock') IS NOT NULL THEN UPDATE approval_policy_clock SET revision=revision+1 WHERE id=1; UPDATE approval_requests SET status='superseded',version=version+1,completed_at=now(),updated_at=now() WHERE status='pending' OR (resource_kind<>'document' AND status='approved'); END IF;
 IF to_regclass('sql_source_proposals') IS NOT NULL THEN UPDATE sql_source_queries SET enabled=false WHERE origin_proposal_id IS NOT NULL; UPDATE sql_source_proposals SET status='cancelled',updated_at=now() WHERE status IN ('proposed','review') OR (approval_id IS NOT NULL AND status='approved'); END IF;
 IF to_regclass('notification_settings') IS NOT NULL THEN UPDATE notification_settings SET enabled=false; UPDATE notification_channels SET enabled=false; UPDATE notification_outbox SET processed_at=now() WHERE processed_at IS NULL; UPDATE notification_deliveries SET status='skipped',message='백업 복원으로 전송을 중단했습니다',updated_at=now() WHERE status IN ('pending','sending','failed'); END IF;
 IF to_regclass('search_fragments') IS NOT NULL THEN DELETE FROM search_fragments; END IF;
 IF to_regclass('search_folded_fragments') IS NOT NULL THEN DELETE FROM search_folded_fragments; END IF;
 IF to_regclass('search_folded_documents') IS NOT NULL THEN DELETE FROM search_folded_documents; END IF;
 IF to_regclass('search_index_queue') IS NOT NULL THEN INSERT INTO search_index_queue(document_id) SELECT id FROM documents WHERE deleted_at IS NULL ON CONFLICT(document_id) DO UPDATE SET enqueued_at=now(); END IF;
 IF to_regclass('inbound_capture_settings') IS NOT NULL THEN UPDATE inbound_capture_settings SET hooks_enabled=false,imap_enabled=false; UPDATE inbound_capture_channels SET enabled=false,next_run=NULL; UPDATE inbound_capture_messages SET status='cancelled',message='백업 복원으로 수집을 중단했습니다',completed_at=now() WHERE status='pending'; END IF;
 IF to_regclass('runbook_settings') IS NOT NULL THEN UPDATE runbook_settings SET enabled=false,revision=revision+1; UPDATE runbook_runners SET enabled=false; UPDATE runbook_executions SET status='cancelled',cancel_requested=false,completed_at=now(),updated_at=now(),last_error='백업 복원으로 실행을 중단했습니다' WHERE status IN ('prepared','review','approved','rejected'); UPDATE runbook_executions SET status='unknown',cancel_requested=false,reconcile_after=now(),updated_at=now(),last_error='복원 이전 외부 실행을 운영자가 확인해야 합니다' WHERE status IN ('queued','running'); UPDATE runbook_executions SET cancel_requested=false WHERE status='unknown'; UPDATE runbook_execution_steps SET state='unknown' WHERE state IN ('launching','running'); UPDATE runbook_execution_steps SET state='cancelled',completed_at=now() WHERE state='queued'; END IF;
 IF to_regclass('protection_settings') IS NOT NULL THEN UPDATE protection_settings SET data=jsonb_set(data,'{public_shares_enabled}','false'::jsonb,true),revision=revision+1; END IF;
 IF to_regclass('public_shares') IS NOT NULL THEN UPDATE public_shares SET revoked_at=coalesce(revoked_at,now()),revision=revision+1; DELETE FROM public_share_access; DELETE FROM public_share_rate_limits; END IF;
 IF to_regclass('rag_index_grants') IS NOT NULL THEN UPDATE rag_index_grants SET active=false,auto_reindex=false,revision=revision+1,last_job_id=NULL; DELETE FROM rag_vector_chunks; DELETE FROM rag_vector_indexes; DELETE FROM rag_reindex_queue; END IF;
 IF to_regclass('rag_generations') IS NOT NULL THEN UPDATE rag_generations SET status='disabled',revision=revision+1,index_job_id=NULL; END IF;
 IF to_regclass('rag_generation_state') IS NOT NULL THEN UPDATE rag_generation_state SET active_id=NULL,revision=revision+1,mode='exact'; END IF;
 IF to_regclass('rag_generation_validations') IS NOT NULL THEN UPDATE rag_generation_validations SET session_hash='',expires_at=now(),consumed_at=coalesce(consumed_at,now()); END IF;
 IF to_regclass('git_sync_settings') IS NOT NULL THEN UPDATE git_sync_settings SET data=jsonb_set(data,'{enabled}','false'::jsonb,true),revision=revision+1; UPDATE git_sync_connections SET enabled=false,revision=revision+1; UPDATE git_sync_runs SET status=CASE WHEN status='running' THEN 'unknown' ELSE 'cancelled' END,snapshot_cipher=NULL,snapshot_size=0,revision=revision+1 WHERE status IN ('preparing','preview','queued','running'); END IF;
 IF to_regclass('agent_runs') IS NOT NULL THEN UPDATE workspace_agents SET enabled=false,revision=revision+1; UPDATE agent_runs SET status='cancelled',session_hash='',error='백업 복원으로 실행을 취소했습니다',updated_at=now() WHERE status IN ('pending','running','awaiting_confirmation'); UPDATE agent_actions SET status='cancelled' WHERE status IN ('planned','confirmed'); END IF;
 IF to_regclass('support_sessions') IS NOT NULL THEN UPDATE support_sessions SET status='revoked',ended_at=now(),session_hash='' WHERE status='active'; END IF;
 IF to_regclass('search_history_preferences') IS NOT NULL THEN UPDATE search_history_preferences SET enabled=false,revision=revision+1; END IF;
 IF to_regclass('export_settings') IS NOT NULL THEN UPDATE export_settings SET enabled=false,revision=revision+1; UPDATE export_runs SET status=CASE WHEN status='ready' THEN 'expired' ELSE 'cancelled' END,session_hash='',updated_at=now() WHERE status IN ('queued','running','ready'); DELETE FROM export_artifact_blobs; END IF;
 IF to_regclass('graph_ai_runs') IS NOT NULL THEN UPDATE graph_ai_runs SET status='cancelled',finished_at=now(),error='백업 복원으로 분석이 중단되었습니다' WHERE status='running'; UPDATE graph_ai_runs SET session_hash=''; UPDATE graph_ai_actions SET status='cancelled',decided_at=now() WHERE status='proposed'; END IF;
 UPDATE settings SET data=jsonb_set(jsonb_set(data,'{support_enabled}','false'::jsonb,true),'{otel_enabled}','false'::jsonb,true) WHERE id=1;
END $$;
COMMIT;
SQL
printf 'Restoration completed in the isolated database: %s\nAttachments: %s\n' "$MADI_TARGET_DATABASE" "$MADI_TARGET_ATTACHMENTS"
printf 'Use madi %s and the original ENCRYPTION_KEY. Check file ownership, login, documents, attachments and integrations before promotion.\n' "$MADI_BACKUP_VERSION"
printf 'Jobs, automation, webhooks, scheduled backups and configured storage writes are disabled for recovery verification. Re-enable only after reviewing external targets.\n'
