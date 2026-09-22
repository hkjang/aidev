package server

import "testing"

func TestPostgresGitSyncRestoreRequiresNewConsent(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	if _, e := s.DB.Exec(ctx, "UPDATE git_sync_settings SET data=data||'{\"enabled\":true}'::jsonb WHERE id=1"); e != nil {
		t.Fatal(e)
	}
	connection := newID()
	secret, e := s.encrypt("restore-fixture-secret")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `INSERT INTO git_sync_connections(id,workspace_id,owner_id,name,enabled,config) VALUES($1,$2,$3,'복원 Git',true,$4)`, connection, wid, p.ID, jsonValue(map[string]any{"url": "https://git.example.invalid/restore.git", "branch": "main", "prefix": "madi", "password": secret})); e != nil {
		t.Fatal(e)
	}
	preview, running := newID(), newID()
	for id, status := range map[string]string{preview: "preview", running: "running"} {
		if _, e = s.DB.Exec(ctx, `INSERT INTO git_sync_runs(id,connection_id,workspace_id,owner_id,direction,status,config_fingerprint,snapshot_cipher,snapshot_size,candidate_commit) VALUES($1,$2,$3,$4,'push',$5,'fixture',$6,8,'abcd')`, id, connection, wid, p.ID, status, []byte(secret)); e != nil {
			t.Fatal(e)
		}
	}
	backup := admin.request("GET", "/api/v1/admin/backup", nil, 200)
	storageMultipart(t, admin, "/api/v1/admin/restore", "git-backup.zip", backup, "RESTORE", 200)
	var enabled bool
	if e = s.DB.QueryRow(ctx, "SELECT (data->>'enabled')::boolean FROM git_sync_settings WHERE id=1").Scan(&enabled); e != nil || enabled {
		t.Fatal("restored Git policy still enabled", e)
	}
	if e = s.DB.QueryRow(ctx, "SELECT enabled FROM git_sync_connections WHERE id=$1", connection).Scan(&enabled); e != nil || enabled {
		t.Fatal("restored Git connection still enabled", e)
	}
	for id, want := range map[string]string{preview: "cancelled", running: "unknown"} {
		var status string
		var cleared bool
		if e = s.DB.QueryRow(ctx, "SELECT status,snapshot_cipher IS NULL AND snapshot_size=0 FROM git_sync_runs WHERE id=$1", id).Scan(&status, &cleared); e != nil || status != want || !cleared {
			t.Fatal("restored Git consent replay remains", status, want, cleared, e)
		}
	}
	var encrypted string
	if e = s.DB.QueryRow(ctx, "SELECT config->>'password' FROM git_sync_connections WHERE id=$1", connection).Scan(&encrypted); e != nil {
		t.Fatal(e)
	}
	if value, e := s.decrypt(encrypted); e != nil || value != "restore-fixture-secret" {
		t.Fatal("encrypted Git credential backup did not roundtrip", e)
	}
}
