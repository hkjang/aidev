package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostgresReadModelJSONBudget(t *testing.T) {
	s, _ := integrationTestServer(t)
	ctx := t.Context()
	for _, test := range []struct {
		name        string
		count, size int
		limited     bool
	}{
		{"canonical-document-sized", 1, 6 << 20, false},
		{"escaped-canonical-document-sized", 1, 25 << 20, false},
		{"single-row-too-large", 1, (32 << 20) + 1, true},
		{"cumulative-too-large", 5, 7 << 20, true},
		{"bounded-list", 4, 7 << 20, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows, err := s.rows(ctx, `SELECT jsonb_build_object('payload',repeat('x',$1::integer)) FROM generate_series(1,$2::integer)`, test.size, test.count)
			if test.limited {
				if !errors.Is(err, errQueryResultTooLarge) || rows != nil {
					t.Fatalf("oversized result must fail without partial output: rows=%d error=%v", len(rows), err)
				}
			} else if err != nil || len(rows) != test.count || len(str(rows[0], "payload")) != test.size {
				t.Fatalf("valid bounded result rejected: rows=%d error=%v", len(rows), err)
			}
			if _, err = s.one(ctx, `SELECT jsonb_build_object('reusable',true)`); err != nil {
				t.Fatal("query connection not reusable after cancellation", err)
			}
		})
	}
}

func TestReadModelErrorsDoNotExposeDriverContext(t *testing.T) {
	for _, test := range []struct {
		err   error
		code  int
		retry string
	}{
		{fmt.Errorf("PRIVATE_SQL_ARGUMENT: %w", errQueryResultTooLarge), http.StatusRequestEntityTooLarge, ""},
		{fmt.Errorf("PRIVATE_SQL_ARGUMENT: %w", context.DeadlineExceeded), http.StatusServiceUnavailable, "1"},
	} {
		w := httptest.NewRecorder()
		respond(w, map[string]any{"private_partial_result": true}, test.err)
		if w.Code != test.code || w.Header().Get("Retry-After") != test.retry || strings.Contains(w.Body.String(), "PRIVATE_SQL_ARGUMENT") || strings.Contains(w.Body.String(), "private_partial_result") {
			t.Fatalf("unsafe result/error response: %d %s", w.Code, w.Body.String())
		}
	}
}

func TestPostgresReadModelEscapedMarkdownIsReadable(t *testing.T) {
	s, c, ctx, p, wid := jobTestFixture(t)
	id := newID()
	// Import accepts raw UTF-8 up to 4 MiB, independently of JSON wire size.
	// PostgreSQL escapes this valid non-NUL control character as six bytes.
	markdown := strings.Repeat("\x1f", 4<<20)
	if _, err := s.DB.Exec(ctx, `INSERT INTO documents(id,workspace_id,owner_id,title,markdown) VALUES($1,$2,$3,'원문 인코딩 상한',$4)`, id, wid, p.ID, markdown); err != nil {
		t.Fatal(err)
	}
	v := testJSONObject(t, c.request("GET", "/api/v1/documents/"+id, nil, 200))
	if str(v, "markdown") != markdown {
		t.Fatal("canonical Markdown was clipped or changed by the read-model budget")
	}
}

func TestPostgresReadModelBudgetJobDoesNotRetry(t *testing.T) {
	s, _, ctx, p, wid := jobTestFixture(t)
	s.RegisterJobHandler("test.read-budget", func(context.Context, Job) (map[string]any, error) {
		return map[string]any{"private_partial_result": true}, fmt.Errorf("PRIVATE_SQL_ARGUMENT: %w", errQueryResultTooLarge)
	})
	id, err := s.EnqueueJob(ctx, nil, "test.read-budget", p.ID, wid, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	j, err := s.claimJob(ctx)
	if err != nil || j.ID != id {
		t.Fatal("expected budget job", err)
	}
	s.runJob(ctx, j)
	var status, message string
	var attempts int
	var partial bool
	if err := s.DB.QueryRow(ctx, `SELECT status,last_error,attempts,coalesce(result ? 'private_partial_result',false) FROM automation_jobs WHERE id=$1`, id).Scan(&status, &message, &attempts, &partial); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || attempts != 1 || message != errQueryResultTooLarge.Error() || partial {
		t.Fatalf("permanent budget exhaustion retried or leaked context: %s %d %s", status, attempts, message)
	}
}
