package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
)

func TestRemovedMemberCommentMutationsAreTerminalIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	ownerID, memberID, spaceID := uuid.New(), uuid.New(), uuid.New()
	ownerName := "comment_terminal_owner_" + strings.ReplaceAll(ownerID.String(), "-", "")
	memberName := "comment_terminal_member_" + strings.ReplaceAll(memberID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name) VALUES
		($1,$2::citext,$2::text),($3,$4::citext,$4::text)`, ownerID, ownerName, memberID, memberName); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'terminal comment replay')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'edit')`, spaceID, memberID); err != nil {
		t.Fatal(err)
	}
	note, err := db.CreateNote(ctx, ownerID, store.Note{SpaceID: spaceID, Content: "comment replay", Color: "yellow", Kind: "thought"})
	if err != nil {
		t.Fatal(err)
	}
	resolveTarget, _, err := db.CreateComment(ctx, memberID, note.ID, nil, "resolve offline", nil)
	if err != nil {
		t.Fatal(err)
	}
	deleteTarget, _, err := db.CreateComment(ctx, memberID, note.ID, nil, "delete offline", nil)
	if err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, memberID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Put("/api/v1/comments/{commentID}/resolve", server.resolveComment)
	router.Delete("/api/v1/comments/{commentID}", server.deleteComment)
	handler := authService.Middleware(auth.Require(server.idempotency(router)))

	if _, err = db.Pool.Exec(ctx, `DELETE FROM space_members WHERE space_id=$1 AND user_id=$2`, spaceID, memberID); err != nil {
		t.Fatal(err)
	}
	var eventsBefore int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM space_events WHERE space_id=$1`, spaceID).Scan(&eventsBefore); err != nil {
		t.Fatal(err)
	}
	send := func(method, path, key, body string) (*httptest.ResponseRecorder, map[string]any) {
		t.Helper()
		request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", key)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		var payload map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v body=%s", err, response.Body.String())
		}
		return response, payload
	}
	assertTerminal := func(name string, response *httptest.ResponseRecorder, payload map[string]any) {
		t.Helper()
		if response.Code != http.StatusForbidden || payload["type"] != "https://umm.local/problems/comment-mutation-forbidden" {
			t.Fatalf("%s response=%d payload=%#v", name, response.Code, payload)
		}
		if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/problem+json") {
			t.Fatalf("%s content type=%q", name, contentType)
		}
	}

	resolved, resolvedPayload := send(http.MethodPut, "/api/v1/comments/"+resolveTarget.ID.String()+"/resolve", "offline:resolve-comment", `{"resolved":true}`)
	assertTerminal("resolve", resolved, resolvedPayload)
	deleted, deletedPayload := send(http.MethodDelete, "/api/v1/comments/"+deleteTarget.ID.String(), "offline:delete-comment", "")
	assertTerminal("delete", deleted, deletedPayload)

	var resolvedState, deleteStillActive bool
	if err = db.Pool.QueryRow(ctx, `SELECT resolved_at IS NOT NULL FROM note_comments WHERE id=$1`, resolveTarget.ID).Scan(&resolvedState); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT deleted_at IS NULL FROM note_comments WHERE id=$1`, deleteTarget.ID).Scan(&deleteStillActive); err != nil {
		t.Fatal(err)
	}
	var eventsAfter, pendingRetries int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM space_events WHERE space_id=$1`, spaceID).Scan(&eventsAfter); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM idempotency_records WHERE user_id=$1`, memberID).Scan(&pendingRetries); err != nil {
		t.Fatal(err)
	}
	if resolvedState || !deleteStillActive || eventsAfter != eventsBefore || pendingRetries != 0 {
		t.Fatalf("forbidden replay changed state: resolved=%v delete_active=%v events=%d/%d pending=%d", resolvedState, deleteStillActive, eventsAfter, eventsBefore, pendingRetries)
	}
}
