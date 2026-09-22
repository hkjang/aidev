#!/usr/bin/env bash
set -euo pipefail
# Destructive operations are deliberately absent: each invocation creates new
# disposable databases/directories and reports them for operator inspection.
MADI_TEST_DATABASE_BASE="${MADI_NATIVE_TEST_DATABASE_BASE:-postgres://gaga@127.0.0.1:15432}"
[[ "$MADI_TEST_DATABASE_BASE" =~ ^postgres(ql)?://[A-Za-z0-9_]+@127\.0\.0\.1:[0-9]+$ ]] || { printf '%s\n' 'Use a password-free local test server base URL.'; exit 1; }
MADI_NATIVE_STAMP="$(date +%s)_$RANDOM"
MADI_NATIVE_SOURCE="madi_native_${MADI_NATIVE_STAMP}"
MADI_NATIVE_TARGET="madi_native_${MADI_NATIVE_STAMP}_restore_test"
MADI_NATIVE_DIR="$(mktemp -d /tmp/madi-native-backup.XXXXXX)"
mkdir -- "$MADI_NATIVE_DIR/source"
cp -- tests/fixtures/native-backup-attachment.txt "$MADI_NATIVE_DIR/source/10101010-1010-4010-8010-101010101010"
createdb --maintenance-db="$MADI_TEST_DATABASE_BASE/postgres" "$MADI_NATIVE_SOURCE"
createdb --maintenance-db="$MADI_TEST_DATABASE_BASE/postgres" "$MADI_NATIVE_TARGET"
psql -X -w --dbname="$MADI_TEST_DATABASE_BASE/$MADI_NATIVE_SOURCE" --set=ON_ERROR_STOP=1 --set="root=$MADI_NATIVE_DIR/source" --quiet <<'SQL'
CREATE TABLE attachments(id uuid PRIMARY KEY,path text NOT NULL,storage_provider_id uuid);
INSERT INTO attachments VALUES('10101010-1010-4010-8010-101010101010',:'root'||'/10101010-1010-4010-8010-101010101010',NULL);
CREATE TABLE settings(id int PRIMARY KEY,data jsonb NOT NULL);
INSERT INTO settings VALUES(1,jsonb_build_object('storage_path',:'root','ai_api_key','enc:test-ciphertext-not-a-secret'));
CREATE TABLE sessions(id text); INSERT INTO sessions VALUES('test-session');
CREATE TABLE job_settings(paused boolean); INSERT INTO job_settings VALUES(false);
CREATE TABLE automation_rules(enabled boolean); INSERT INTO automation_rules VALUES(true);
CREATE TABLE automation_webhooks(enabled boolean); INSERT INTO automation_webhooks VALUES(true);
CREATE TABLE automation_jobs(status text,cancel_requested boolean,lease_id uuid,lease_until timestamptz,last_error text,finished_at timestamptz);
INSERT INTO automation_jobs(status,cancel_requested) VALUES('pending',false);
CREATE TABLE backup_policy(enabled boolean,next_run timestamptz); INSERT INTO backup_policy VALUES(true,now());
CREATE TABLE storage_providers(enabled boolean); INSERT INTO storage_providers VALUES(true);
CREATE TABLE storage_settings(provider_id uuid); INSERT INTO storage_settings VALUES('20202020-2020-4020-8020-202020202020');
CREATE TABLE storage_assignments(provider_id uuid); INSERT INTO storage_assignments SELECT * FROM storage_settings;
CREATE TABLE backup_artifacts(managed boolean,last_error text); INSERT INTO backup_artifacts VALUES(true,'');
CREATE TABLE collaboration_presence(id text); INSERT INTO collaboration_presence VALUES('ephemeral');
SQL
bash scripts/backup-postgres.sh --dsn "$MADI_TEST_DATABASE_BASE/$MADI_NATIVE_SOURCE" --attachments "$MADI_NATIVE_DIR/source" --output "$MADI_NATIVE_DIR/backup" --acknowledge WRITES_STOPPED
bash scripts/restore-postgres.sh --backup "$MADI_NATIVE_DIR/backup" --target-dsn "$MADI_TEST_DATABASE_BASE/$MADI_NATIVE_TARGET" --attachments "$MADI_NATIVE_DIR/restored" --acknowledge RESTORE_DISPOSABLE_TARGET
cmp -- tests/fixtures/native-backup-attachment.txt "$MADI_NATIVE_DIR/restored/10101010-1010-4010-8010-101010101010"
MADI_NATIVE_CHECK="$(psql -X -w --dbname="$MADI_TEST_DATABASE_BASE/$MADI_NATIVE_TARGET" --set=ON_ERROR_STOP=1 --set="root=$MADI_NATIVE_DIR/restored" --tuples-only --no-align <<'SQL'
SELECT (SELECT count(*)=0 FROM sessions)
 AND (SELECT bool_and(paused) FROM job_settings)
 AND (SELECT bool_and(NOT enabled) FROM automation_rules)
 AND (SELECT bool_and(NOT enabled) FROM automation_webhooks)
 AND (SELECT bool_and(status='cancelled' AND cancel_requested) FROM automation_jobs)
 AND (SELECT bool_and(NOT enabled AND next_run IS NULL) FROM backup_policy)
 AND (SELECT bool_and(NOT enabled) FROM storage_providers)
 AND (SELECT bool_and(provider_id IS NULL) FROM storage_settings)
 AND (SELECT bool_and(provider_id IS NULL) FROM storage_assignments)
 AND (SELECT bool_and(NOT managed) FROM backup_artifacts)
 AND (SELECT count(*)=0 FROM collaboration_presence)
 AND (SELECT bool_and(path=:'root'||'/10101010-1010-4010-8010-101010101010') FROM attachments)
 AND (SELECT data->>'storage_path'=:'root' FROM settings WHERE id=1);
SQL
)"
[[ "$MADI_NATIVE_CHECK" == t ]] || { printf '%s\n' 'Restored data or safety state differs.'; exit 1; }
psql -X -w --dbname="$MADI_TEST_DATABASE_BASE/$MADI_NATIVE_SOURCE" --set=ON_ERROR_STOP=1 --quiet --command="UPDATE attachments SET storage_provider_id='20202020-2020-4020-8020-202020202020'"
if bash scripts/backup-postgres.sh --dsn "$MADI_TEST_DATABASE_BASE/$MADI_NATIVE_SOURCE" --attachments "$MADI_NATIVE_DIR/source" --output "$MADI_NATIVE_DIR/incomplete-provider-backup" --acknowledge WRITES_STOPPED; then
 printf '%s\n' 'Provider-based attachment backup was incorrectly accepted.'; exit 1
fi
[[ ! -e "$MADI_NATIVE_DIR/incomplete-provider-backup" ]] || { printf '%s\n' 'Rejected backup wrote output.'; exit 1; }
printf 'Native backup/restore and safety assertions passed.\nSource DB: %s\nRestore DB: %s\nArtifacts retained: %s\n' "$MADI_NATIVE_SOURCE" "$MADI_NATIVE_TARGET" "$MADI_NATIVE_DIR"
