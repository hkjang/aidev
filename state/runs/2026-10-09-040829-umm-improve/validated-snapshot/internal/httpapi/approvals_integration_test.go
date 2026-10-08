package httpapi

import (
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

// A reviewer is being asked to allow something to happen to a particular space.
// The listing used to name the requester, the action and the resource *type* —
// "export · space" — and nothing about which space, which is the one fact the
// decision turns on.

func TestApprovalsNameWhatIsBeingReviewedIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	adminID, spaceID, goneID := uuid.New(), uuid.New(), uuid.New()
	username := "approvals_" + strings.ReplaceAll(adminID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, adminID, username); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'제품 결정')`, spaceID, adminID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO approval_requests(requester_id,resource_type,resource_id,action,comment) VALUES
		($1,'space',$2,'export','분기 회고를 팀 외부와 공유하려 합니다.'),
		($1,'space',$3,'space_share','사라진 공간에 대한 요청')`, adminID, spaceID, goneID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, adminID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Get("/approvals", server.listApprovals)
	handler := authService.Middleware(auth.Require(router))

	request := httptest.NewRequest(http.MethodGet, "/approvals", nil)
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("listing approvals=%d body=%s", response.Code, response.Body.String())
	}

	var body struct {
		Requests []struct {
			Action       string `json:"action"`
			ResourceName string `json:"resourceName"`
		} `json:"requests"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	// A request whose space is gone must still be listed and reviewable. A
	// request outlives its subject, and dropping it would hide work someone is
	// waiting on.
	if len(body.Requests) != 2 {
		t.Fatalf("expected both requests, got %d: %s", len(body.Requests), response.Body.String())
	}

	byAction := map[string]string{}
	for _, r := range body.Requests {
		byAction[r.Action] = r.ResourceName
	}
	if byAction["export"] != "제품 결정" {
		t.Errorf("the export request does not say which space: %q", byAction["export"])
	}
	// Named as absent rather than guessed at: the page shows its own wording for
	// an empty name, and inventing one here would be worse than saying nothing.
	if byAction["space_share"] != "" {
		t.Errorf("a deleted space was given a name: %q", byAction["space_share"])
	}
}

// Someone who is not a reviewer sees their own requests and nobody else's. The
// listing builds one query with three different scopes, so the narrow one has
// to be exercised too.
func TestApprovalsStayWithinTheirScopeIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	memberID, strangerID, spaceID := uuid.New(), uuid.New(), uuid.New()
	member := "approvals_member_" + strings.ReplaceAll(memberID.String(), "-", "")
	stranger := "approvals_other_" + strings.ReplaceAll(strangerID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'user'),($3,$4::citext,$4::text,'user')`,
		memberID, member, strangerID, stranger); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'남의 공간')`, spaceID, strangerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO approval_requests(requester_id,resource_type,resource_id,action,comment) VALUES
		($1,'space',$3,'export','내 요청'),
		($2,'space',$3,'export','남의 요청')`, memberID, strangerID, spaceID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, memberID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Get("/approvals", server.listApprovals)
	handler := authService.Middleware(auth.Require(router))

	request := httptest.NewRequest(http.MethodGet, "/approvals", nil)
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	var body struct {
		Requests []struct {
			Comment      string `json:"comment"`
			ResourceName string `json:"resourceName"`
		} `json:"requests"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Requests) != 1 || body.Requests[0].Comment != "내 요청" {
		t.Fatalf("a plain user saw more than their own requests: %s", response.Body.String())
	}
	// The name still comes back for the one they may see.
	if body.Requests[0].ResourceName != "남의 공간" {
		t.Errorf("the request does not say which space: %q", body.Requests[0].ResourceName)
	}
}
