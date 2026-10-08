package httpapi

import (
	"context"
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

// Taking someone out of a shared space has to actually take them out.
//
// This is the shape that has been wrong here before: a screen lists something,
// offers to remove it, and the list is the only evidence anything happened. The
// API keys were reported that way, and the sessions screen had no test at all.
// Removing a person is the one where being wrong matters most — the row leaves
// the owner's screen and the reader keeps reading.
//
// Plenty of tests delete from space_members with SQL to arrange "not a member".
// None went through the endpoint someone actually presses, and none checked the
// surfaces a removed person could still reach.
func TestRemovingAMemberActuallyRevokesTheirAccessIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	ownerID, memberID, strangerID := uuid.New(), uuid.New(), uuid.New()
	name := func(id uuid.UUID) string { return "rm_" + strings.ReplaceAll(id.String(), "-", "") }
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text),($3,$4::citext,$4::text),($5,$6::citext,$6::text)`,
		ownerID, name(ownerID), memberID, name(memberID), strangerID, name(strangerID)); err != nil {
		t.Fatal(err)
	}
	spaceID, noteID := uuid.New(), uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'나가는 공간')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'공유된 생각')`,
		noteID, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'edit')`,
		spaceID, memberID); err != nil {
		t.Fatal(err)
	}

	// While shared, the member reaches the space and its thoughts.
	spaces, err := db.ListSpaces(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSpace(spaces, spaceID) {
		t.Fatal("the member cannot see the space they were shared into; the rest of this test would prove nothing")
	}
	notes, _, err := db.ListNotes(ctx, memberID, spaceID, "")
	if err != nil || len(notes) != 1 {
		t.Fatalf("the member could not read the shared thought: %d notes, %v", len(notes), err)
	}

	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService}
	router := chi.NewRouter()
	router.Delete("/spaces/{spaceID}/members/{memberID}", server.removeSpaceMember)
	handler := authService.Middleware(auth.Require(router))

	remove := func(actor uuid.UUID, target uuid.UUID) int {
		session, err := authService.CreateSession(ctx, actor, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodDelete, "/spaces/"+spaceID.String()+"/members/"+target.String(), nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}

	// Someone with no standing in the space cannot remove anybody from it.
	if code := remove(strangerID, memberID); code != http.StatusForbidden {
		t.Errorf("a stranger removing a member returned %d, want 403", code)
	}
	if spaces, err = db.ListSpaces(ctx, memberID); err != nil || !hasSpace(spaces, spaceID) {
		t.Error("a refused removal took the space away anyway")
	}

	if code := remove(ownerID, memberID); code != http.StatusNoContent {
		t.Fatalf("the owner removing a member returned %d, want 204", code)
	}

	// The part a listing cannot prove: the space is gone from their side, and
	// the thoughts in it are no longer readable.
	spaces, err = db.ListSpaces(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if hasSpace(spaces, spaceID) {
		t.Error("the removed person still lists the space")
	}
	notes, _, err = db.ListNotes(ctx, memberID, spaceID, "")
	if err == nil && len(notes) > 0 {
		t.Errorf("the removed person can still read %d thoughts in the space", len(notes))
	}

	// And the owner still has their own space.
	spaces, err = db.ListSpaces(ctx, ownerID)
	if err != nil || !hasSpace(spaces, spaceID) {
		t.Error("removing a member took the space from its owner")
	}

	// Removing them twice says so rather than reporting success again.
	if code := remove(ownerID, memberID); code != http.StatusNotFound {
		t.Errorf("removing an already removed member returned %d, want 404", code)
	}
}

func hasSpace(spaces []store.Space, id uuid.UUID) bool {
	for _, space := range spaces {
		if space.ID == id {
			return true
		}
	}
	return false
}
