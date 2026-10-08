package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
)

func TestNotificationAccessRevocationIntegration(t *testing.T) {
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

	ownerID, memberID, spaceID, noteID, dreamID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ownerName := "notification_owner_" + ownerID.String()
	memberName := "notification_member_" + memberID.String()
	if _, err = db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name) VALUES
		($1,$2::citext,$2::text),($3,$4::citext,$4::text)`, ownerID, ownerName, memberID, memberName); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1 OR id=$2`, ownerID, memberID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'revoked notification access')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view')`, spaceID, memberID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'notification target')`, noteID, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = db.CreateComment(ctx, ownerID, noteID, nil, "revoked secret body", []string{memberName}); err != nil {
		t.Fatal(err)
	}
	// Simulate a notification created before resource_space_id was introduced.
	// Note-scoped access must be derived from the current note, not stale or
	// missing denormalized metadata.
	if _, err = db.Pool.Exec(ctx, `UPDATE notifications SET resource_space_id=NULL WHERE user_id=$1 AND resource_type='note' AND resource_id=$2`, memberID, noteID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO dream_notes(dream_id,user_id,space_id,dream_type,content) VALUES($1,$2,$3,'connection','revoked Dream body')`, dreamID, memberID, spaceID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `
		INSERT INTO notifications(user_id,kind,title,body,resource_type,resource_id) VALUES
		($1,'space_shared','legacy space notification','legacy revoked body','space',$2),
		($1,'dream','legacy Dream notification','revoked Dream notification body','dream',$3),
		($1,'dream','personal notification','safe personal body','dream',NULL)`, memberID, spaceID, dreamID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, memberID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	handler := authService.Middleware(auth.Require(http.HandlerFunc(server.listNotifications)))
	type notificationResponse struct {
		Notifications []struct {
			Kind string `json:"kind"`
			Body string `json:"body"`
		} `json:"notifications"`
		Unread int64 `json:"unread"`
	}
	fetch := func() notificationResponse {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("notification response=%d body=%s", response.Code, response.Body.String())
		}
		var payload notificationResponse
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}

	before := fetch()
	if len(before.Notifications) != 4 || before.Unread != 4 {
		t.Fatalf("accessible notifications=%d unread=%d, want 4", len(before.Notifications), before.Unread)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE notes SET deleted_at=now() WHERE id=$1`, noteID); err != nil {
		t.Fatal(err)
	}
	deleted := fetch()
	if len(deleted.Notifications) != 3 || deleted.Unread != 3 {
		t.Fatalf("deleted-note notifications=%d unread=%d, want 3", len(deleted.Notifications), deleted.Unread)
	}
	for _, item := range deleted.Notifications {
		if item.Body == "revoked secret body" {
			t.Fatalf("deleted note notification leaked: %#v", deleted.Notifications)
		}
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE notes SET deleted_at=NULL WHERE id=$1`, noteID); err != nil {
		t.Fatal(err)
	}
	restored := fetch()
	if len(restored.Notifications) != 4 || restored.Unread != 4 {
		t.Fatalf("restored-note notifications=%d unread=%d, want 4", len(restored.Notifications), restored.Unread)
	}
	if _, err = db.Pool.Exec(ctx, `DELETE FROM space_members WHERE space_id=$1 AND user_id=$2`, spaceID, memberID); err != nil {
		t.Fatal(err)
	}
	after := fetch()
	if len(after.Notifications) != 1 || after.Unread != 1 || after.Notifications[0].Kind != "dream" || after.Notifications[0].Body != "safe personal body" {
		t.Fatalf("revoked space notifications leaked: %#v unread=%d", after.Notifications, after.Unread)
	}
	var persisted int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id=$1`, memberID).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != 4 {
		t.Fatalf("test expected access filtering without deletion, persisted=%d", persisted)
	}
}

func TestNotificationsCompleteWithOneRequestConnectionIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("pool_max_conns", "2")
	parsed.RawQuery = query.Encode()

	ctx := context.Background()
	db, err := store.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if db.Pool.Config().MaxConns != 2 {
		t.Fatalf("test requires a two-connection pool, got %d", db.Pool.Config().MaxConns)
	}

	userID := uuid.New()
	username := "notification_pool_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO notifications(user_id,kind,title,body,resource_type) VALUES($1,'dream','pool-safe notification','one connection remains','dream')`, userID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	handler := authService.Middleware(auth.Require(http.HandlerFunc(server.listNotifications)))

	listenerConnection, err := db.Pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer listenerConnection.Release()
	requestContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil).WithContext(requestContext)
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("notification list exhausted the last request connection: status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Notifications []map[string]any `json:"notifications"`
		Unread        int64            `json:"unread"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Notifications) != 1 || payload.Unread != 1 {
		t.Fatalf("unexpected notification response: %#v", payload)
	}
}

func TestSpaceMemberListRechecksAccessIntegration(t *testing.T) {
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

	ownerID, memberID, otherID, spaceID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ownerName, memberName, otherName := "members_owner_"+ownerID.String(), "members_viewer_"+memberID.String(), "members_other_"+otherID.String()
	if _, err = db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name) VALUES
		($1,$2::citext,$2::text),($3,$4::citext,$4::text),($5,$6::citext,$6::text)`,
		ownerID, ownerName, memberID, memberName, otherID, otherName); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=ANY($1::uuid[])`, []uuid.UUID{ownerID, memberID, otherID})
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'member list access')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view'),($1,$3,'edit')`, spaceID, memberID, otherID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, memberID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Get("/api/v1/spaces/{spaceID}/members", server.listSpaceMembers)
	handler := authService.Middleware(auth.Require(router))
	fetch := func() *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/spaces/"+spaceID.String()+"/members", nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	before := fetch()
	if before.Code != http.StatusOK {
		t.Fatalf("member list before revocation=%d body=%s", before.Code, before.Body.String())
	}
	var payload struct {
		Members []map[string]any `json:"members"`
	}
	if err = json.Unmarshal(before.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Members) != 3 {
		t.Fatalf("member list before revocation=%#v", payload.Members)
	}
	if _, err = db.Pool.Exec(ctx, `DELETE FROM space_members WHERE space_id=$1 AND user_id=$2`, spaceID, memberID); err != nil {
		t.Fatal(err)
	}
	after := fetch()
	if after.Code != http.StatusNotFound {
		t.Fatalf("member list after revocation=%d body=%s", after.Code, after.Body.String())
	}
}
