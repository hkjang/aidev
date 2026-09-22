package server

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Opt-in because pg_dump/pg_restore and CREATE DATABASE are additional operational
// prerequisites. Every database mutated or dropped below is freshly generated.
func TestNativeBackupCurrentFullSchema(t *testing.T) {
	if os.Getenv("MADI_NATIVE_BACKUP_FULL") != "1" {
		t.Skip("MADI_NATIVE_BACKUP_FULL=1 enables isolated native full-schema restore")
	}
	dsn := os.Getenv("MADI_TEST_POSTGRES_DSN")
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("An explicit disposable PostgreSQL URI ending in _test is required")
	}
	for _, command := range []string{"bash", "pg_dump", "pg_restore", "psql", "tar", "sha256sum"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Fatalf("required operational tool unavailable: %s", command)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	stamp := strings.ReplaceAll(newID(), "-", "")[:16]
	sourceName, targetName := "madi_native_"+stamp, "madi_native_"+stamp+"_restore_test"
	for _, name := range []string{sourceName, targetName} {
		if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
			defer stop()
			if _, err := admin.Exec(cleanup, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
				t.Errorf("cleanup of this test's database %s failed: %v", name, err)
			}
		})
	}
	databaseURL := func(name string, password bool) string {
		copyURL := *u
		copyURL.Path = "/" + name
		if !password {
			copyURL.User = url.User(u.User.Username())
		}
		return copyURL.String()
	}
	source, err := pgxpool.New(ctx, databaseURL(sourceName, true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(source.Close)
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	versionBytes, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	version := strings.TrimSpace(string(versionBytes))
	key := bytes.Repeat([]byte{42}, 32)
	s, err := New(ctx, source, key, version, "admin@example.test", "Native-Backup-Password-2026!", fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("madi")}})
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	attachments := filepath.Join(temp, "source")
	if err := os.Mkdir(attachments, 0700); err != nil {
		t.Fatal(err)
	}
	attachmentID := newID()
	attachmentBytes := []byte("실제 전체 스키마 운영 백업 첨부\n")
	if err := os.WriteFile(filepath.Join(attachments, attachmentID), attachmentBytes, 0600); err != nil {
		t.Fatal(err)
	}
	var documentID, actorID, originalMarkdown string
	if err := source.QueryRow(ctx, "SELECT id::text,owner_id::text,markdown FROM documents LIMIT 1").Scan(&documentID, &actorID, &originalMarkdown); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := s.encrypt("native-fixture-secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, "INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path) VALUES($1,$2,$3,'fixture.txt','text/plain',$4,$5)", attachmentID, documentID, actorID, len(attachmentBytes), filepath.Join(attachments, attachmentID)); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, "UPDATE settings SET data=data||jsonb_build_object('storage_path',$1::text,'native_test_secret',$2::text,'support_enabled',true,'otel_enabled',true) WHERE id=1", attachments, ciphertext); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES('native-test-cookie',$1,now()+interval '1 hour');`, actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `UPDATE job_settings SET paused=false;
UPDATE connector_settings SET enabled=true;
UPDATE notification_settings SET enabled=true;
UPDATE inbound_capture_settings SET hooks_enabled=true,imap_enabled=true;
UPDATE runbook_settings SET enabled=true;
UPDATE export_settings SET enabled=true;
UPDATE protection_settings SET data=data||'{"public_shares_enabled":true}'::jsonb;
UPDATE git_sync_settings SET data=data||'{"enabled":true}'::jsonb;`); err != nil {
		t.Fatal(err)
	}
	oldEpoch, migrationID, migrationItemID := newID(), newID(), newID()
	if _, err := source.Exec(ctx, `INSERT INTO document_collaboration(document_id,epoch,sequence,state,projected_markdown,document_version,snapshot_sequence,head_hash) SELECT id,$2,3,'old-binary'::bytea,markdown,version,2,'old-hash' FROM documents WHERE id=$1`, documentID, oldEpoch); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `UPDATE knowledge_evidence_policy SET enabled=true; UPDATE knowledge_package_policy SET enabled=true`); err != nil {
		t.Fatal(err)
	}
	learningPathID, learningStepID, learningProgressID := newID(), newID(), newID()
	if _, err := source.Exec(ctx, `INSERT INTO knowledge_paths(id,workspace_id,owner_id,title) SELECT $1,workspace_id,owner_id,'개인 지식 경로 복원' FROM documents WHERE id=$2`, learningPathID, documentID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO knowledge_path_steps(id,path_id,document_id,ordinal,kind,title) VALUES($1,$2,$3,0,'practice','실습 기록')`, learningStepID, learningPathID, documentID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO knowledge_path_progress(id,step_id,user_id,path_revision,document_version,state,proof) VALUES($1,$2,$3,1,1,'confirmed',$4)`, learningProgressID, learningStepID, actorID, ciphertext); err != nil {
		t.Fatal(err)
	}
	structuredDB, structuredID := newID(), newID()
	if _, err := source.Exec(ctx, `INSERT INTO databases(id,workspace_id,name) SELECT $1,workspace_id,'구조화 초안 복원' FROM documents WHERE id=$2`, structuredDB, documentID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO knowledge_structured_drafts(id,owner_id,document_id,database_id,source_version,source_hash,start_byte,end_byte,schema_hash,destination_hash,provider_hash,ciphertext,expires_at) VALUES($1,$2,$3,$4,1,'source',0,1,'schema','destination','provider',$5,now()+interval '1 hour')`, structuredID, actorID, documentID, structuredDB, ciphertext); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO migration_sessions(id,workspace_id,user_id,source_key,label,format,status,declared_bytes,uploaded_bytes,item_count,prepared_count) SELECT $1,workspace_id,owner_id,'native-source','복원 검증','markdown','ready',4,4,1,1 FROM documents WHERE id=$2`, migrationID, documentID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO migration_session_items(id,session_id,source_id,source_hash,file_path,kind,source_bytes,chunk_count,target_id,status,prepared_data,staged_object) VALUES($1,$2,'native','hash','native.md','document',4,1,$3,'prepared',$4,'{"key":"attachments/not-replayed"}');`, migrationItemID, migrationID, newID(), []byte(ciphertext)); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO migration_session_chunks(item_id,ordinal,size,checksum,data) VALUES($1,0,4,'hash',$2);`, migrationItemID, []byte(ciphertext)); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO migration_session_objects(item_id,document_id,attachment_id,object,provider_fingerprint,status) VALUES($1,$2,$3,'{"key":"attachments/not-replayed"}','old-provider','ready')`, migrationItemID, documentID, newID()); err != nil {
		t.Fatal(err)
	}
	generationID, validationID := newID(), newID()
	if _, err := source.Exec(ctx, `INSERT INTO rag_generations(id,workspace_id,name,status,dimensions,provider_fingerprint,config_cipher,created_by) SELECT $1,workspace_id,'복원 세대','active',3,'fixture',$2,owner_id FROM documents WHERE id=$3`, generationID, ciphertext, documentID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO rag_generation_state(workspace_id,active_id,mode) SELECT workspace_id,$1,'ann' FROM documents WHERE id=$2 ON CONFLICT(workspace_id) DO UPDATE SET active_id=EXCLUDED.active_id,mode='ann'`, generationID, documentID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO rag_generation_validations(id,generation_id,workspace_id,actor_id,session_hash,generation_revision,state_revision,settings_fingerprint,query_hash,cohort,report,expires_at) SELECT $1,$2,workspace_id,owner_id,'old-session',1,1,'fixture','fixture','[{}]','{}',now()+interval '15 minutes' FROM documents WHERE id=$3`, validationID, generationID, documentID); err != nil {
		t.Fatal(err)
	}
	distributionKeyID, distributionID := newID(), newID()
	if _, err := source.Exec(ctx, `UPDATE knowledge_distribution_policy SET enabled=true`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO knowledge_distribution_keys(id,kind,label,source_instance,source_key_id,public_key,fingerprint,private_ciphertext,created_by) SELECT $1,'signing','복원 서명 키',instance_id,$1,'fixture-public','fixture-fingerprint',$2,$3 FROM knowledge_distribution_policy WHERE id=1`, distributionKeyID, ciphertext, actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO knowledge_distribution_exports(id,workspace_id,owner_id,signing_key_id,receiver_instance,policy_revision,status,expires_at) SELECT $1,workspace_id,owner_id,$2,$3,1,'ready',now()+interval '1 day' FROM documents WHERE id=$4`, distributionID, distributionKeyID, newID(), documentID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO knowledge_distribution_artifacts(export_id,ciphertext) VALUES($1,$2)`, distributionID, []byte(ciphertext)); err != nil {
		t.Fatal(err)
	}
	extractionID, fragmentID := newID(), newID()
	if _, err := source.Exec(ctx, `UPDATE attachment_extraction_settings SET data=data||'{"enabled":true,"ocr_enabled":true}'; UPDATE system_status_policy SET enabled=true`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO attachment_extractions(id,attachment_id,document_id,workspace_id,actor_id,status,policy_revision,document_version,checksum,object_snapshot,format,session_hash,token_hash,temp_path) SELECT $1,$2,id,workspace_id,owner_id,'ready',1,version,$3,'{}','text','old-session','old-token','/tmp/madi-extract-old' FROM documents WHERE id=$4`, extractionID, attachmentID, digest(string(attachmentBytes)), documentID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO attachment_extraction_fragments(id,extraction_id,ordinal,text,content_hash,position) VALUES($1,$2,0,'derived',$3,'{}')`, fragmentID, extractionID, digest("derived")); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO attachment_extraction_heads(attachment_id,extraction_id) VALUES($1,$2)`, attachmentID, extractionID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Exec(ctx, `INSERT INTO system_status_cards(document_id,workspace_id,document_version,owner_id,ttl_seconds,ciphertext,updated_by) SELECT id,workspace_id,version,owner_id,3600,$2,owner_id FROM documents WHERE id=$1`, documentID, ciphertext); err != nil {
		t.Fatal(err)
	}
	environment := os.Environ()
	if password, ok := u.User.Password(); ok {
		escape := strings.NewReplacer("\\", "\\\\", ":", "\\:").Replace
		port := u.Port()
		if port == "" {
			port = "5432"
		}
		passfile := filepath.Join(temp, "pgpass")
		line := escape(u.Hostname()) + ":" + port + ":*:" + escape(u.User.Username()) + ":" + escape(password) + "\n"
		if err := os.WriteFile(passfile, []byte(line), 0600); err != nil {
			t.Fatal(err)
		}
		environment = append(environment, "PGPASSFILE="+passfile)
	}
	run := func(script string, args ...string) {
		t.Helper()
		command := exec.CommandContext(ctx, "bash", append([]string{filepath.Join(root, "scripts", script)}, args...)...)
		command.Dir, command.Env = root, environment
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v\n%s", script, err, output)
		}
	}
	backup, restored := filepath.Join(temp, "backup"), filepath.Join(temp, "restored")
	run("backup-postgres.sh", "--dsn", databaseURL(sourceName, false), "--attachments", attachments, "--output", backup, "--acknowledge", "WRITES_STOPPED")
	run("restore-postgres.sh", "--backup", backup, "--target-dsn", databaseURL(targetName, false), "--attachments", restored, "--acknowledge", "RESTORE_DISPOSABLE_TARGET")
	target, err := pgxpool.New(ctx, databaseURL(targetName, true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(target.Close)
	var valid bool
	if err := target.QueryRow(ctx, `SELECT
(SELECT count(*)=0 FROM sessions) AND (SELECT paused FROM job_settings WHERE id=1)
AND (SELECT NOT enabled FROM connector_settings WHERE id=1)
AND (SELECT NOT enabled FROM notification_settings WHERE id=1)
AND (SELECT NOT hooks_enabled AND NOT imap_enabled FROM inbound_capture_settings WHERE id=1)
AND (SELECT NOT enabled FROM runbook_settings WHERE id=1)
AND (SELECT NOT enabled FROM export_settings WHERE id=true)
AND (SELECT NOT (data->>'public_shares_enabled')::boolean FROM protection_settings WHERE id=1)
AND (SELECT NOT (data->>'enabled')::boolean FROM git_sync_settings WHERE id=1)
AND (SELECT NOT (data->>'support_enabled')::boolean AND NOT (data->>'otel_enabled')::boolean AND data->>'storage_path'=$1 FROM settings WHERE id=1)
AND EXISTS(SELECT 1 FROM search_index_queue WHERE document_id=$2)
AND (SELECT markdown=$3 FROM documents WHERE id=$2)
AND (SELECT path=$4 FROM attachments WHERE id=$5)`, restored, documentID, originalMarkdown, filepath.Join(restored, attachmentID), attachmentID).Scan(&valid); err != nil || !valid {
		t.Fatalf("restored full-schema data/safety checks: valid=%v err=%v", valid, err)
	}
	var restoredCipher string
	if err := target.QueryRow(ctx, `SELECT state='needs_recheck' AND revision=2 AND approval_id IS NULL AND proof=$2 FROM knowledge_path_progress WHERE id=$1`, learningProgressID, ciphertext).Scan(&valid); err != nil || !valid {
		t.Fatal("learning restore did not require reconfirmation", valid, err)
	}
	if err := target.QueryRow(ctx, `SELECT state='expired' AND revision=2 AND expires_at<=clock_timestamp() AND ciphertext=$2 FROM knowledge_structured_drafts WHERE id=$1`, structuredID, ciphertext).Scan(&valid); err != nil || !valid {
		t.Fatal("structured draft was revived after restore", valid, err)
	}
	if err := target.QueryRow(ctx, "SELECT data->>'native_test_secret' FROM settings WHERE id=1").Scan(&restoredCipher); err != nil {
		t.Fatal(err)
	}
	if plain, err := s.decrypt(restoredCipher); err != nil || plain != "native-fixture-secret" || restoredCipher != ciphertext {
		t.Fatal("encrypted configuration did not survive native restore")
	}
	var invalidated bool
	if err := target.QueryRow(ctx, `SELECT c.epoch<>$2::uuid AND c.sequence=0 AND octet_length(c.state)=0 AND c.snapshot_sequence=0 AND c.head_hash='' AND c.reset_reason='backup_restored' AND c.projected_markdown=d.markdown AND c.document_version=d.version FROM document_collaboration c JOIN documents d ON d.id=c.document_id WHERE d.id=$1`, documentID, oldEpoch).Scan(&invalidated); err != nil || !invalidated {
		t.Fatal("native CRDT epoch not invalidated", err)
	}
	if err := target.QueryRow(ctx, `SELECT m.status='cancelled' AND m.purged_at IS NOT NULL AND i.prepared_data IS NULL AND i.staged_object='{}' AND NOT EXISTS(SELECT 1 FROM migration_session_chunks WHERE item_id=i.id) AND NOT EXISTS(SELECT 1 FROM migration_session_objects WHERE item_id=i.id AND status<>'discarded') FROM migration_sessions m JOIN migration_session_items i ON i.session_id=m.id WHERE m.id=$1`, migrationID).Scan(&invalidated); err != nil || !invalidated {
		t.Fatal("native migration staging not invalidated", err)
	}
	for _, table := range []string{"knowledge_evidence_policy", "knowledge_package_policy"} {
		if err := target.QueryRow(ctx, "SELECT NOT p.enabled AND EXISTS(SELECT 1 FROM "+table+"_history h WHERE h.version=p.version AND NOT h.enabled) FROM "+table+" p WHERE p.id=1").Scan(&invalidated); err != nil || !invalidated {
			t.Fatal("native policy history missing", table, err)
		}
	}
	if err := target.QueryRow(ctx, `SELECT g.status='disabled' AND g.revision=2 AND g.index_job_id IS NULL AND st.active_id IS NULL AND st.mode='exact' AND st.revision>=2 AND v.session_hash='' AND v.expires_at<=now() AND v.consumed_at IS NOT NULL FROM rag_generations g JOIN rag_generation_state st USING(workspace_id) JOIN rag_generation_validations v ON v.generation_id=g.id WHERE g.id=$1 AND v.id=$2`, generationID, validationID).Scan(&invalidated); err != nil || !invalidated {
		t.Fatal("native RAG generation or validation not invalidated", err)
	}
	if err := target.QueryRow(ctx, `SELECT NOT p.enabled AND p.revision=2 AND EXISTS(SELECT 1 FROM knowledge_distribution_policy_history h WHERE h.revision=p.revision AND NOT h.enabled) AND k.revoked_at IS NOT NULL AND k.revision=2 AND e.status='expired' AND NOT EXISTS(SELECT 1 FROM knowledge_distribution_artifacts a WHERE a.export_id=e.id) FROM knowledge_distribution_policy p CROSS JOIN knowledge_distribution_keys k CROSS JOIN knowledge_distribution_exports e WHERE p.id=1 AND k.id=$1 AND e.id=$2`, distributionKeyID, distributionID).Scan(&invalidated); err != nil || !invalidated {
		t.Fatal("native signed distribution not invalidated", err)
	}
	if err := target.QueryRow(ctx, `SELECT NOT (p.data->>'enabled')::boolean AND NOT (p.data->>'ocr_enabled')::boolean AND p.revision=2 AND EXISTS(SELECT 1 FROM attachment_extraction_policy_history h WHERE h.revision=p.revision AND h.data=p.data) AND e.status='obsolete' AND e.session_hash='' AND e.token_hash='' AND e.temp_path='' AND NOT EXISTS(SELECT 1 FROM attachment_extraction_heads) AND NOT EXISTS(SELECT 1 FROM attachment_extraction_fragments) FROM attachment_extraction_settings p CROSS JOIN attachment_extractions e WHERE p.id=1 AND e.id=$1`, extractionID).Scan(&invalidated); err != nil || !invalidated {
		t.Fatal("native extraction policy/capabilities/projection not invalidated", err)
	}
	if err := target.QueryRow(ctx, `SELECT NOT p.enabled AND p.revision=2 AND c.verification_epoch=2 AND c.revision=2 AND EXISTS(SELECT 1 FROM system_status_policy_history h WHERE h.revision=p.revision AND NOT (h.policy->>'enabled')::boolean) FROM system_status_policy p CROSS JOIN system_status_cards c WHERE p.id=1 AND c.document_id=$1`, documentID).Scan(&invalidated); err != nil || !invalidated {
		t.Fatal("native system status freshness/policy not invalidated", err)
	}
	if actual, err := os.ReadFile(filepath.Join(restored, attachmentID)); err != nil || !bytes.Equal(actual, attachmentBytes) {
		t.Fatal("attachment bytes did not survive native restore")
	}
	t.Log("Current complete schema, encrypted settings, Markdown and attachment bytes restored; external execution policies disabled; disposable databases removed on cleanup")
}
