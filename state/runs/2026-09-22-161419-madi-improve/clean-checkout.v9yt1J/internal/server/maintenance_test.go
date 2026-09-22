package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func maintenanceFixture(t *testing.T) (*Server, *integrationTestClient, string, string) {
	t.Helper()
	s, httpServer := integrationTestServer(t)
	admin := newIntegrationTestClient(t, httpServer.URL)
	u := testJSONObject(t, admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	w := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "Retention fixture"}, 200))
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"trash_retention_days": 10}, 200)
	return s, admin, str(w, "id"), str(u, "id")
}
func runMaintenanceTest(t *testing.T, s *Server) (maintenanceResult, error) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		result, e := s.runMaintenance(context.Background())
		if e != nil || !result.Skipped || time.Now().After(deadline) {
			return result, e
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestPostgresMaintenanceRetentionAndExactFileDeletion(t *testing.T) {
	s, admin, wid, uid := maintenanceFixture(t)
	ctx := context.Background()
	storage := t.TempDir()
	create := func(title, visibility, parent string) string {
		t.Helper()
		return str(testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "visibility": visibility, "parent_id": parent}, 200)), "id")
	}
	old := create("오래된 휴지통", "workspace", "")
	ancestor := create("하위가 남은 휴지통", "private", "")
	young := create("최근 휴지통", "workspace", "")
	private := create("개인 원문", "private", "")
	child := create("남아 있는 하위 문서", "workspace", ancestor)
	if _, err := s.DB.Exec(ctx, `UPDATE documents SET deleted_at=now()-interval '11 days' WHERE id=$1`, ancestor); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(ctx, `UPDATE documents SET deleted_at=CASE WHEN id=$1 THEN now()-interval '11 days' ELSE now()-interval '5 days' END WHERE id IN($1,$2)`, old, young); err != nil {
		t.Fatal(err)
	}
	attachment := func(doc string) string {
		t.Helper()
		fid := newID()
		filename := filepath.Join(storage, fid)
		if err := os.WriteFile(filename, []byte("retention fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB.Exec(ctx, `INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path) VALUES($1,$2,$3,'fixture.txt','text/plain',17,$4)`, fid, doc, uid, filename); err != nil {
			t.Fatal(err)
		}
		return filename
	}
	oldFile, youngFile, privateFile := attachment(old), attachment(young), attachment(private)
	orphan := filepath.Join(storage, newID())
	if err := os.WriteFile(orphan, []byte("restore retained file"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`INSERT INTO comments(id,document_id,user_id,body) VALUES($3,$1,$2,'comment')`,
		`INSERT INTO notifications(id,document_id,user_id,title) VALUES($3,$1,$2,'notice')`,
	} {
		if _, err := s.DB.Exec(ctx, query, old, uid, newID()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO favorites(user_id,document_id) VALUES($1,$2)`, uid, old); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO document_shares(user_id,document_id,permission) VALUES($1,$2,'read')`, uid, old); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES('expired-retention-fixture',$1,now()-interval '1 minute')`, uid); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO oidc_attempts(state_hash,browser_hash,nonce,verifier,issuer,client_id,redirect_uri,expires_at) VALUES('expired-retention-fixture','b','n','v','i','c','r',now()-interval '1 minute')`); err != nil {
		t.Fatal(err)
	}
	result, err := runMaintenanceTest(t, s)
	if err != nil {
		t.Fatal(err)
	}
	if result.Documents != 1 || result.Files != 1 || result.ParentsDetached != 0 || result.ParentsRetained != 1 || result.Sessions != 1 || result.OIDCAttempts != 1 || len(result.FileErrors) != 0 {
		t.Fatalf("unexpected retention result: %+v", result)
	}
	if _, err = os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("old attachment still exists: %v", err)
	}
	for _, filename := range []string{youngFile, privateFile, orphan} {
		if _, err = os.Stat(filename); err != nil {
			t.Fatalf("unrelated attachment removed: %s", filename)
		}
	}
	var parent *string
	if err = s.DB.QueryRow(ctx, `SELECT parent_id::text FROM documents WHERE id=$1`, child).Scan(&parent); err != nil || parent == nil || *parent != ancestor {
		t.Fatalf("surviving child ancestor was not retained: %v %v", parent, err)
	}
	for _, table := range []string{"comments", "notifications", "favorites", "document_shares", "document_versions", "attachments"} {
		var count int
		if err = s.DB.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE document_id=$1", old).Scan(&count); err != nil || count != 0 {
			t.Fatalf("dependent %s not removed: %d %v", table, count, err)
		}
	}
	admin.request("GET", "/api/v1/documents/"+private, nil, 200)
	admin.request("GET", "/api/v1/documents/"+young, nil, 200)
}

func TestPostgresMaintenanceRollbackPreservesAttachmentsAndParent(t *testing.T) {
	s, admin, wid, uid := maintenanceFixture(t)
	ctx := context.Background()
	old := str(testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "rollback old"}, 200)), "id")
	child := str(testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "rollback child", "parent_id": old}, 200)), "id")
	leaf := str(testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "rollback leaf"}, 200)), "id")
	if _, err := s.DB.Exec(ctx, `UPDATE documents SET deleted_at=now()-interval '11 days' WHERE id=$1`, leaf); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(ctx, `UPDATE documents SET deleted_at=now()-interval '11 days' WHERE id=$1`, old); err != nil {
		t.Fatal(err)
	}
	fid := newID()
	filename := filepath.Join(t.TempDir(), fid)
	if err := os.WriteFile(filename, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path) VALUES($1,$2,$3,'keep.txt','text/plain',4,$4)`, fid, old, uid, filename); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(ctx, `CREATE FUNCTION retention_refuse_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'maintenance rollback fixture'; END; $$; CREATE TRIGGER retention_refuse_delete BEFORE DELETE ON documents FOR EACH ROW EXECUTE FUNCTION retention_refuse_delete()`); err != nil {
		t.Fatal(err)
	}
	if _, err := runMaintenanceTest(t, s); err == nil {
		t.Fatal("expected failed transaction")
	}
	if _, err := os.Stat(filename); err != nil {
		t.Fatal("file was removed before commit")
	}
	var parent string
	if err := s.DB.QueryRow(ctx, `SELECT parent_id::text FROM documents WHERE id=$1`, child).Scan(&parent); err != nil || parent != old {
		t.Fatalf("parent change escaped rollback: %s %v", parent, err)
	}
	var exists bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM attachments WHERE id=$1)`, fid).Scan(&exists); err != nil || !exists {
		t.Fatal("attachment metadata was not rolled back")
	}
}

func TestPostgresMaintenancePreservesRevokedAncestorACL(t *testing.T) {
	s, admin, wid, owner := maintenanceFixture(t)
	ctx := context.Background()
	parent := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "기밀 상위", "visibility": "private"}, 200))
	childOwner := newID()
	_, e := s.DB.Exec(ctx, "INSERT INTO users(id,email,name,role) VALUES($1,'revoked-child@test.local','하위 소유자','editor')", childOwner)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = s.DB.Exec(ctx, "INSERT INTO workspace_members VALUES($1,$2,'editor')", wid, childOwner)
	child := newID()
	_, e = s.DB.Exec(ctx, "INSERT INTO documents(id,workspace_id,parent_id,title,owner_id,visibility) VALUES($1,$2,$3,'하위 원문',$4,'workspace')", child, wid, parent["id"], childOwner)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = s.DB.Exec(ctx, "UPDATE documents SET deleted_at=now()-interval '11 days' WHERE id=$1", parent["id"])
	p := &Principal{ID: childOwner, Role: "editor"}
	if s.canDocument(ctx, p, child, false) {
		t.Fatal("private ancestor was not applied")
	}
	result, e := runMaintenanceTest(t, s)
	if e != nil || result.ParentsRetained != 1 || result.Documents != 0 {
		t.Fatalf("ancestor retention failed: %+v %v", result, e)
	}
	if s.canDocument(ctx, p, child, false) {
		t.Fatal("maintenance granted revoked child owner access")
	}
	_ = owner
}
func TestPostgresMaintenanceLegalHoldAndRetainUntil(t *testing.T) {
	s, admin, wid, _ := maintenanceFixture(t)
	ctx := context.Background()
	ids := []string{}
	for _, name := range []string{"법적 보존", "기간 보존", "보존 만료"} {
		d := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": name}, 200))
		ids = append(ids, str(d, "id"))
	}
	_, e := s.DB.Exec(ctx, "UPDATE documents SET deleted_at=now()-interval '11 days' WHERE id=ANY($1::uuid[])", ids)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DB.Exec(ctx, `INSERT INTO knowledge_document_meta(document_id,legal_hold,retain_until) VALUES($1,true,NULL),($2,false,now()+interval '1 day'),($3,false,now()-interval '1 day')`, ids[0], ids[1], ids[2])
	if e != nil {
		t.Fatal(e)
	}
	result, e := runMaintenanceTest(t, s)
	if e != nil || result.Documents != 1 {
		t.Fatalf("legal hold retention: %+v %v", result, e)
	}
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE id=ANY($1::uuid[])", ids[:2]).Scan(&count)
	if count != 2 {
		t.Fatal("legal hold or retain_until document deleted")
	}
}

func TestPostgresMaintenanceReplicaLockAndBatchLimit(t *testing.T) {
	s, _, wid, uid := maintenanceFixture(t)
	ctx := context.Background()
	conn, err := s.DB.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock(726234801)`); err != nil {
		conn.Release()
		t.Fatal(err)
	}
	result, runErr := s.runMaintenance(ctx)
	_, _ = conn.Exec(ctx, `SELECT pg_advisory_unlock(726234801)`)
	conn.Release()
	if runErr != nil || !result.Skipped {
		t.Fatalf("replica lock not respected: %+v %v", result, runErr)
	}
	_, err = s.DB.Exec(ctx, `INSERT INTO documents(id,workspace_id,title,owner_id,deleted_at) SELECT md5($1::text || '-' || g::text)::uuid,$1::uuid,'old '||g::text,$2,now()-interval '11 days' FROM generate_series(1,501) g`, wid, uid)
	if err != nil {
		t.Fatal(err)
	}
	result, err = runMaintenanceTest(t, s)
	if err != nil || result.Documents != 500 {
		t.Fatalf("batch limit failed: %+v %v", result, err)
	}
	result, err = runMaintenanceTest(t, s)
	if err != nil || result.Documents != 1 {
		t.Fatalf("remaining batch failed: %+v %v", result, err)
	}
}
