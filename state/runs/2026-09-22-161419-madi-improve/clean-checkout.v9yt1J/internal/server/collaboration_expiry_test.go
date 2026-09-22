package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestPostgresCollaborationExpiryAfterLockWait(t *testing.T) {
	for _, stage := range []string{"before_authorization", "before_commit"} {
		t.Run(stage, func(t *testing.T) {
			s, _, _, id, session, doc, state := collaborationJournalFixture(t)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			lock, err := s.DB.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Rollback(context.Background())
			if stage == "before_authorization" {
				_, err = lock.Exec(ctx, "SELECT id FROM documents WHERE id=$1 FOR UPDATE", id)
			} else {
				_, err = lock.Exec(ctx, "LOCK TABLE collaboration_updates IN ACCESS EXCLUSIVE MODE")
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.DB.Exec(ctx, "UPDATE sessions SET expires_at=clock_timestamp()+interval '1500 milliseconds' WHERE token_hash=$1", session); err != nil {
				t.Fatal(err)
			}
			collaborationTestInsert(doc, " MUST_NOT_COMMIT_AFTER_EXPIRY")
			type result struct {
				message collaborationMessage
				err     error
			}
			done := make(chan result, 1)
			go func() {
				message, err := s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: doc.EncodeStateAsUpdate()})
				done <- result{message, err}
			}()
			pid := int32(lock.Conn().PgConn().PID())
			collaborationWait(t, "actual document/journal SQL wait", func() bool {
				var n int
				e := s.DB.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid))", pid).Scan(&n)
				return e == nil && n > 0
			})
			collaborationWait(t, "wall clock session expiry", func() bool {
				var expired bool
				e := s.DB.QueryRow(ctx, "SELECT expires_at<=clock_timestamp() FROM sessions WHERE token_hash=$1", session).Scan(&expired)
				return e == nil && expired
			})
			if err = lock.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			actual := <-done
			var fault *collaborationFault
			if !errors.As(actual.err, &fault) || fault.code != "forbidden" {
				t.Fatalf("expired session after %s accepted: message=%s committed=%v error=%v", stage, actual.message.Type, actual.message.Committed, actual.err)
			}
			var markdown string
			var version int
			if err = s.DB.QueryRow(ctx, "SELECT markdown,version FROM documents WHERE id=$1", id).Scan(&markdown, &version); err != nil {
				t.Fatal(err)
			}
			if markdown != state.Markdown || version != state.Version {
				t.Fatal("expired mutation changed canonical state/version")
			}
		})
	}
}

func TestPostgresCollaborationCompactionExpiryAfterJournalWait(t *testing.T) {
	s, admin, _, id, session, _, state := collaborationJournalFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	lock, err := s.DB.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Rollback(context.Background())
	if _, err = lock.Exec(ctx, "LOCK TABLE collaboration_updates IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec(ctx, "UPDATE sessions SET expires_at=clock_timestamp()+interval '1500 milliseconds' WHERE token_hash=$1", session); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]any{"expected_version": state.Version, "expected_epoch": state.Epoch, "confirm": true})
	request, err := http.NewRequestWithContext(ctx, "POST", admin.base+"/api/v1/documents/"+id+"/collaboration/compact", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Madi-Request", "1")
	request.Header.Set("Content-Type", "application/json")
	type result struct {
		status int
		err    error
	}
	done := make(chan result, 1)
	go func() {
		response, err := admin.client.Do(request)
		status := 0
		if response != nil {
			status = response.StatusCode
			_ = response.Body.Close()
		}
		done <- result{status, err}
	}()
	pid := int32(lock.Conn().PgConn().PID())
	collaborationWait(t, "actual compaction journal wait", func() bool {
		var n int
		e := s.DB.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid))", pid).Scan(&n)
		return e == nil && n > 0
	})
	collaborationWait(t, "compaction session wall clock expiry", func() bool {
		var expired bool
		e := s.DB.QueryRow(ctx, "SELECT expires_at<=clock_timestamp() FROM sessions WHERE token_hash=$1", session).Scan(&expired)
		return e == nil && expired
	})
	if err = lock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	actual := <-done
	if actual.err != nil || actual.status != 403 {
		t.Fatal("expired compaction accepted", actual.status, actual.err)
	}
	var epoch, markdown string
	var version int
	if err = s.DB.QueryRow(ctx, "SELECT c.epoch::text,d.markdown,d.version FROM document_collaboration c JOIN documents d ON d.id=c.document_id WHERE d.id=$1", id).Scan(&epoch, &markdown, &version); err != nil {
		t.Fatal(err)
	}
	if epoch != state.Epoch || markdown != state.Markdown || version != state.Version {
		t.Fatal("expired compaction changed epoch/canonical document")
	}
}

func TestPostgresCollaborationRevocationWinsQueuedUpdate(t *testing.T) {
	s, _, editor, id, _, doc, state := collaborationJournalFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	session := collaborationTestSession(t, editor)
	lock, err := s.DB.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Rollback(context.Background())
	if _, err = lock.Exec(ctx, "UPDATE documents SET visibility='private' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	collaborationTestInsert(doc, " REVOKED_QUEUED_UPDATE")
	done := make(chan error, 1)
	go func() {
		_, err := s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: doc.EncodeStateAsUpdate()})
		done <- err
	}()
	pid := int32(lock.Conn().PgConn().PID())
	collaborationWait(t, "actual queued edit behind ACL mutation", func() bool {
		var n int
		e := s.DB.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid))", pid).Scan(&n)
		return e == nil && n > 0
	})
	if err = lock.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var fault *collaborationFault
	if err = <-done; !errors.As(err, &fault) || fault.code != "forbidden" {
		t.Fatal("queued edit bypassed committed ACL", err)
	}
	var markdown string
	var version int
	if err = s.DB.QueryRow(ctx, "SELECT markdown,version FROM documents WHERE id=$1", id).Scan(&markdown, &version); err != nil {
		t.Fatal(err)
	}
	if markdown != state.Markdown || version != state.Version {
		t.Fatal("revoked queued update changed content")
	}
}
