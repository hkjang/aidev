package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// The SQL predicate can be evaluated before a row-lock wait. Commit-time
// credentials must still be valid after the last lock, including key settings.
func TestPostgresKnowledgeActorCredentialExpiresDuringLock(t *testing.T) {
	for _, kind := range []string{"session", "key"} {
		t.Run(kind, func(t *testing.T) {
			s, _, base, original, wid := jobTestFixture(t)
			ctx, cancel := context.WithTimeout(base, 8*time.Second)
			defer cancel()
			p := *original
			secret := "knowledge-expiry-regression-only"
			var expires time.Time
			if kind == "key" {
				p.TokenID, p.WorkspaceID, p.Scopes = newID(), wid, []string{"document:read"}
				if e := s.DB.QueryRow(ctx, `INSERT INTO api_keys(id,user_id,name,prefix,token_hash,scopes,workspace_id,expires_at) VALUES($1,$2,'만료 회귀','fixture',$3,$4,$5,clock_timestamp()+interval '1500 milliseconds') RETURNING expires_at`, p.TokenID, p.ID, integrationHash(secret), p.Scopes, wid).Scan(&expires); e != nil {
					t.Fatal(e)
				}
			} else if e := s.DB.QueryRow(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,clock_timestamp()+interval '1500 milliseconds') RETURNING expires_at`, digest(secret), p.ID).Scan(&expires); e != nil {
				t.Fatal(e)
			}
			r := httptest.NewRequest("GET", "/", nil).WithContext(context.WithValue(ctx, principalKey, &p))
			r.RemoteAddr = "127.0.0.1:12345"
			if kind == "key" {
				r.Header.Set("Authorization", "Bearer "+secret)
			} else {
				r.AddCookie(&http.Cookie{Name: "madi_session", Value: secret})
			}
			blocking, e := s.DB.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer blocking.Rollback(context.WithoutCancel(ctx))
			if kind == "key" {
				_, e = blocking.Exec(ctx, `SELECT id FROM settings WHERE id=1 FOR UPDATE`)
			} else {
				_, e = blocking.Exec(ctx, `SELECT user_id FROM sessions WHERE token_hash=$1 FOR UPDATE`, digest(secret))
			}
			if e != nil {
				t.Fatal(e)
			}
			tx, e := s.DB.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(context.WithoutCancel(ctx))
			pid := tx.Conn().PgConn().PID()
			result := make(chan error, 1)
			go func() { result <- s.knowledgeActorTx(r, tx, wid, "document:read") }()
			waiting := false
			for time.Now().Before(expires) {
				if e = s.DB.QueryRow(ctx, `SELECT coalesce(wait_event_type='Lock',false) FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&waiting); e != nil {
					t.Fatal(e)
				}
				if waiting {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("fixture did not wait on the intended lock before expiry")
			}
			if delay := time.Until(expires) + 30*time.Millisecond; delay > 0 {
				time.Sleep(delay)
			}
			if e = blocking.Rollback(ctx); e != nil {
				t.Fatal(e)
			}
			if e = <-result; e == nil {
				t.Fatal("expired credential was accepted after the last row lock")
			}
		})
	}
}
