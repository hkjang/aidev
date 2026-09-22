package server

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPostgresResumableMigrationCredentialExpiresDuringLock(t *testing.T) {
	for _, credential := range []string{"session", "key"} {
		t.Run(credential, func(t *testing.T) {
			s, c, baseCtx, original, wid := jobTestFixture(t)
			ctx, cancel := context.WithTimeout(baseCtx, 8*time.Second)
			defer cancel()
			id := resumeTestUpload(t, c, wid, "expiry-lock", map[string]string{"a.md": "만료 검증"})
			spaceID := newID()
			if _, e := s.DB.Exec(ctx, "INSERT INTO spaces(id,workspace_id,name,slug,owner_id) VALUES($1,$2,'검증 공간','expiry',$3)", spaceID, wid, original.ID); e != nil {
				t.Fatal(e)
			}
			p := *original
			var expires time.Time
			if credential == "session" {
				if e := s.DB.QueryRow(ctx, "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES('migration-expiring-session',$1,clock_timestamp()+interval '1500 milliseconds') RETURNING expires_at", p.ID).Scan(&expires); e != nil {
					t.Fatal(e)
				}
				if _, e := s.DB.Exec(ctx, "UPDATE migration_sessions SET space_id=$2,request_session_hash='migration-expiring-session' WHERE id=$1", id, spaceID); e != nil {
					t.Fatal(e)
				}
			} else {
				p.TokenID, p.WorkspaceID, p.Scopes = newID(), wid, []string{"document:write"}
				if e := s.DB.QueryRow(ctx, "INSERT INTO api_keys(id,user_id,name,prefix,token_hash,scopes,workspace_id,expires_at) VALUES($1,$2,'만료','fixture','migration-expiring-key',$3,$4,clock_timestamp()+interval '1500 milliseconds') RETURNING expires_at", p.TokenID, p.ID, p.Scopes, wid).Scan(&expires); e != nil {
					t.Fatal(e)
				}
				if _, e := s.DB.Exec(ctx, "UPDATE migration_sessions SET space_id=$2,request_token_hash='migration-expiring-key',request_ip='127.0.0.1' WHERE id=$1", id, spaceID); e != nil {
					t.Fatal(e)
				}
			}
			v, e := scanMigrationSession(s.DB.QueryRow(ctx, "SELECT "+migrationSessionSelect+" FROM migration_sessions WHERE id=$1", id))
			if e != nil {
				t.Fatal(e)
			}
			blocking, e := s.DB.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer blocking.Rollback(context.WithoutCancel(ctx))
			if _, e = blocking.Exec(ctx, "SELECT id FROM spaces WHERE id=$1 FOR UPDATE", spaceID); e != nil {
				t.Fatal(e)
			}
			tx, e := s.DB.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(context.WithoutCancel(ctx))
			pid := tx.Conn().PgConn().PID()
			result := make(chan error, 1)
			go func() { result <- s.migrationActorTx(ctx, tx, &p, v, false) }()
			waiting := false
			for time.Now().Before(expires) {
				if e = s.DB.QueryRow(ctx, "SELECT coalesce(wait_event_type='Lock',false) FROM pg_stat_activity WHERE pid=$1", pid).Scan(&waiting); e != nil {
					t.Fatal(e)
				}
				if waiting {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("fixture did not reach the later ACL lock before credential expiry")
			}
			if delay := time.Until(expires) + 30*time.Millisecond; delay > 0 {
				time.Sleep(delay)
			}
			if e = blocking.Rollback(ctx); e != nil {
				t.Fatal(e)
			}
			if e = <-result; !errors.Is(e, errMigrationChanged) {
				t.Fatalf("credential expiring after its read was accepted: %v", e)
			}
		})
	}
}
