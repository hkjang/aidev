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
)

// What happens to someone's spaces when they leave.
//
// Deactivating an account does not touch what it owns. The spaces stay theirs,
// reachable by whoever was shared in and nobody else, and there was no way to
// see that or hand them on — the metrics screen counted spaces and said
// everything was fine.
func TestAdminCanSeeAndHandOnASpaceIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	admin, leaver, stayer := uuid.New(), uuid.New(), uuid.New()
	name := func(id uuid.UUID) string { return "sp_" + strings.ReplaceAll(id.String(), "-", "") }
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name,role,active) VALUES
		($1,$2::citext,$2::text,'admin',true),
		($3,$4::citext,$4::text,'user',false),
		($5,$6::citext,$6::text,'user',true)`,
		admin, name(admin), leaver, name(leaver), stayer, name(stayer)); err != nil {
		t.Fatal(err)
	}
	left, inbox := uuid.New(), uuid.New()
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO spaces(id,owner_id,name,is_inbox) VALUES($1,$2,'떠난 사람의 공간',false),($3,$2,'수집함',true)`,
		left, leaver, inbox); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view')`, left, stayer); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService}
	router := chi.NewRouter()
	router.Get("/admin/spaces", server.adminSpaces)
	router.Get("/admin/spaces/{spaceID}/members", server.adminSpaceMembers)
	router.Put("/admin/spaces/{spaceID}/owner", server.transferSpaceOwner)
	handler := authService.Middleware(auth.Require(router))
	session, err := authService.CreateSession(ctx, admin, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	send := func(method, path string, payload any) (int, map[string]any) {
		t.Helper()
		var reader *bytes.Reader
		if payload != nil {
			raw, _ := json.Marshal(payload)
			reader = bytes.NewReader(raw)
		} else {
			reader = bytes.NewReader(nil)
		}
		request := httptest.NewRequest(method, path, reader)
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		out := map[string]any{}
		_ = json.Unmarshal(response.Body.Bytes(), &out)
		return response.Code, out
	}

	// The spaces a departed person still owns, asked for directly.
	code, body := send(http.MethodGet, "/admin/spaces?ownerInactive=true", nil)
	if code != http.StatusOK {
		t.Fatalf("listing spaces returned %d: %v", code, body)
	}
	spaces, _ := body["spaces"].([]any)
	found := map[string]bool{}
	for _, raw := range spaces {
		space, _ := raw.(map[string]any)
		found[space["name"].(string)] = true
		if space["ownerActive"] != false {
			t.Errorf("asking for inactive owners returned an active one: %v", space["owner"])
		}
	}
	if !found["떠난 사람의 공간"] {
		t.Fatalf("the departed person's space is not listed: %v", spaces)
	}

	// Who can reach it, and how.
	code, body = send(http.MethodGet, "/admin/spaces/"+left.String()+"/members", nil)
	if code != http.StatusOK {
		t.Fatalf("listing members returned %d: %v", code, body)
	}
	members, _ := body["members"].([]any)
	if len(members) != 2 {
		t.Fatalf("the space has an owner and one member, listed %d: %v", len(members), members)
	}

	// An inbox is nobody else's to take.
	if code, body = send(http.MethodPut, "/admin/spaces/"+inbox.String()+"/owner", map[string]any{"userId": stayer}); code != http.StatusBadRequest {
		t.Errorf("handing on an inbox returned %d, want 400: %v", code, body)
	}

	// Handing it to someone who has also left would only move the problem.
	if code, _ = send(http.MethodPut, "/admin/spaces/"+left.String()+"/owner", map[string]any{"userId": leaver}); code == http.StatusOK {
		t.Error("a space was handed to the person who already owns it")
	}

	// The transfer itself.
	code, body = send(http.MethodPut, "/admin/spaces/"+left.String()+"/owner", map[string]any{"userId": stayer})
	if code != http.StatusOK {
		t.Fatalf("transfer returned %d: %v", code, body)
	}
	var owner uuid.UUID
	if err = db.Pool.QueryRow(ctx, `SELECT owner_id FROM spaces WHERE id=$1`, left).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	if owner != stayer {
		t.Fatalf("the space is owned by %v, want %v", owner, stayer)
	}
	// The new owner's old membership row would say something weaker than the
	// truth, so it goes.
	var stale int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM space_members WHERE space_id=$1 AND user_id=$2`, left, stayer).Scan(&stale); err != nil {
		t.Fatal(err)
	}
	if stale != 0 {
		t.Error("the new owner still has a membership row alongside owning it")
	}
	// The departed owner is not given access back.
	var previous int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM space_members WHERE space_id=$1 AND user_id=$2`, left, leaver).Scan(&previous); err != nil {
		t.Fatal(err)
	}
	if previous != 0 {
		t.Error("a deactivated previous owner was given access to the space they left")
	}
	if body["previousKeptAccess"] != false {
		t.Errorf("the answer claims the previous owner kept access: %v", body["previousKeptAccess"])
	}
}

// Moving a space between two people who are both still here must not take it
// away from the one who had it.
func TestTransferKeepsAnActiveOwnersAccessIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	admin, from, to := uuid.New(), uuid.New(), uuid.New()
	name := func(id uuid.UUID) string { return "tr_" + strings.ReplaceAll(id.String(), "-", "") }
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name,role,active) VALUES
		($1,$2::citext,$2::text,'admin',true),($3,$4::citext,$4::text,'user',true),($5,$6::citext,$6::text,'user',true)`,
		admin, name(admin), from, name(from), to, name(to)); err != nil {
		t.Fatal(err)
	}
	space := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'옮기는 공간')`, space, from); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService}
	router := chi.NewRouter()
	router.Put("/admin/spaces/{spaceID}/owner", server.transferSpaceOwner)
	handler := authService.Middleware(auth.Require(router))
	session, err := authService.CreateSession(ctx, admin, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"userId": to})
	request := httptest.NewRequest(http.MethodPut, "/admin/spaces/"+space.String()+"/owner", bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("transfer returned %d: %s", response.Code, response.Body.String())
	}

	var permission string
	if err = db.Pool.QueryRow(ctx, `SELECT permission FROM space_members WHERE space_id=$1 AND user_id=$2`, space, from).Scan(&permission); err != nil {
		t.Fatalf("the previous owner lost the space entirely: %v", err)
	}
	if permission != "manage" {
		t.Errorf("the previous owner was left with %q", permission)
	}
}
