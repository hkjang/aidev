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

func TestUpdateNoteReadOnlyIsTerminalIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	ownerID, editorID, spaceID := uuid.New(), uuid.New(), uuid.New()
	ownerName := "readonly_owner_" + ownerID.String()
	editorName := "readonly_editor_" + editorID.String()
	if _, err = db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name) VALUES
		($1,$2::citext,$2::text),($3,$4::citext,$4::text)`, ownerID, ownerName, editorID, editorName); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1 OR id=$2`, ownerID, editorID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'read-only replay')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'edit')`, spaceID, editorID); err != nil {
		t.Fatal(err)
	}
	note, err := db.CreateNote(ctx, ownerID, store.Note{SpaceID: spaceID, Content: "server content", Color: "yellow", Kind: "thought"})
	if err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, editorID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Put("/api/v1/notes/{noteID}", server.updateNote)
	handler := authService.Middleware(auth.Require(router))
	send := func(version int) (*httptest.ResponseRecorder, map[string]any) {
		t.Helper()
		body, marshalErr := json.Marshal(map[string]any{
			"content": "offline edit", "title": "", "color": "yellow", "kind": "thought",
			"x": 0, "y": 0, "width": 240, "height": 160, "rotation": 0, "version": version,
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		request := httptest.NewRequest(http.MethodPut, "/api/v1/notes/"+note.ID.String(), bytes.NewReader(body))
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		var payload map[string]any
		if response.Body.Len() > 0 {
			if unmarshalErr := json.Unmarshal(response.Body.Bytes(), &payload); unmarshalErr != nil {
				t.Fatalf("decode response: %v body=%s", unmarshalErr, response.Body.String())
			}
		}
		return response, payload
	}

	conflict, conflictPayload := send(note.Version - 1)
	if conflict.Code != http.StatusConflict || conflictPayload["type"] != "https://umm.local/problems/note-version-conflict" {
		t.Fatalf("editable stale update response=%d payload=%#v", conflict.Code, conflictPayload)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE space_members SET permission='view' WHERE space_id=$1 AND user_id=$2`, spaceID, editorID); err != nil {
		t.Fatal(err)
	}
	readOnly, readOnlyPayload := send(note.Version)
	if readOnly.Code != http.StatusForbidden || readOnlyPayload["type"] != "https://umm.local/problems/note-read-only" {
		t.Fatalf("read-only update response=%d payload=%#v", readOnly.Code, readOnlyPayload)
	}
	// This path answers a live edit and a replayed offline change alike, so the
	// message may not assume which. It used to say the "offline change" could
	// not be applied, and reached someone online who had just typed into a
	// space they may only read.
	if detail, _ := readOnlyPayload["detail"].(string); strings.Contains(detail, "오프라인") {
		t.Errorf("the read-only message assumes an offline change: %q", detail)
	}
	latest, err := db.NoteByID(ctx, editorID, note.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Content != "server content" || latest.Version != note.Version {
		t.Fatalf("read-only mutation changed note: content=%q version=%d", latest.Content, latest.Version)
	}
}
