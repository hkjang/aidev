package apikeys

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/google/uuid"
)

func TestRotationEnforcesOwnerAndGrantableScopesBeforeReturningSecret(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	ownerID := uuid.NewString()
	otherID := uuid.NewString()
	for _, user := range []struct{ id, name string }{
		{ownerID, "rotate.policy.owner"},
		{otherID, "rotate.policy.other"},
	} {
		if _, err := runtime.DB().Exec(
			`INSERT INTO users(
			 id,username,normalized_username,display_name,active,super_admin
			 ) VALUES($1,$2,$2,$2,TRUE,FALSE)`,
			user.id,
			user.name,
		); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(runtime.DB())
	created, err := service.Create(
		context.Background(),
		ownerID,
		"sensitive-rotation",
		[]string{"assets.read", "queries.execute"},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.Rotate(
		context.Background(),
		created.Key.ID,
		time.Minute,
		Access{
			UserID:          ownerID,
			GrantableScopes: []string{"assets.read"},
		},
	); !errors.Is(err, ErrForbidden) {
		t.Fatalf("scope-escalating rotation error = %v, want ErrForbidden", err)
	}
	if _, err := service.Rotate(
		context.Background(),
		created.Key.ID,
		time.Minute,
		Access{
			UserID:          otherID,
			GrantableScopes: []string{"assets.read", "queries.execute"},
		},
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-owner rotation error = %v, want ErrNotFound", err)
	}
	if _, err := service.Authenticate(context.Background(), created.Secret); err != nil {
		t.Fatalf("rejected rotation changed the original secret: %v", err)
	}

	rotated, err := service.Rotate(
		context.Background(),
		created.Key.ID,
		time.Minute,
		Access{
			UserID:          ownerID,
			GrantableScopes: []string{"queries.execute", "assets.read"},
		},
	)
	if err != nil || rotated.Secret == "" || rotated.Secret == created.Secret {
		t.Fatalf("authorized owner rotation = %#v, %v", rotated, err)
	}

	ownerKeys, err := service.ListFor(context.Background(), Access{UserID: ownerID})
	if err != nil || len(ownerKeys) != 1 {
		t.Fatalf("owner ListFor() = %#v, %v", ownerKeys, err)
	}
	otherKeys, err := service.ListFor(context.Background(), Access{UserID: otherID})
	if err != nil || len(otherKeys) != 0 {
		t.Fatalf("non-owner ListFor() = %#v, %v", otherKeys, err)
	}
	if _, err := service.GetFor(
		context.Background(), created.Key.ID, Access{UserID: otherID},
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-owner GetFor() error = %v, want ErrNotFound", err)
	}
}

func TestScopeReplacementRejectsStaleRevisionAtomically(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	userID := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO users(
		 id,username,normalized_username,display_name,active,super_admin
		 ) VALUES($1,'scope.owner','scope.owner','Scope Owner',TRUE,TRUE)`,
		userID,
	); err != nil {
		t.Fatal(err)
	}
	service := NewService(runtime.DB())
	created, err := service.Create(
		context.Background(),
		userID,
		"scope-revision",
		[]string{"mcp.access"},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	var staleScopes string
	if err := runtime.DB().QueryRow(
		"SELECT scopes_json FROM api_keys WHERE id=$1",
		created.Key.ID,
	).Scan(&staleScopes); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddScopes(
		context.Background(),
		created.Key.ID,
		[]string{"agents.read"},
	); err != nil {
		t.Fatal(err)
	}
	newName := "must-not-be-partially-applied"
	replacement := []string{"assets.write", "mcp.access"}
	if _, err := service.update(
		context.Background(),
		created.Key.ID,
		&newName,
		newName,
		&replacement,
		`["assets.write","mcp.access"]`,
		staleScopes,
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale scope replacement error = %v, want ErrConflict", err)
	}
	key, err := service.Get(context.Background(), created.Key.ID)
	if err != nil {
		t.Fatal(err)
	}
	if key.Name != "scope-revision" {
		t.Fatalf("stale scope replacement changed name to %q", key.Name)
	}
	if len(key.Scopes) != 2 || key.Scopes[0] != "agents.read" ||
		key.Scopes[1] != "mcp.access" {
		t.Fatalf("stale scope replacement overwrote scopes: %v", key.Scopes)
	}
}

func TestRotationRejectsStaleKeyRevision(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	userID := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO users(
		 id,username,normalized_username,display_name,active,super_admin
		 ) VALUES($1,'rotate.owner','rotate.owner','Rotate Owner',TRUE,TRUE)`,
		userID,
	); err != nil {
		t.Fatal(err)
	}
	service := NewService(runtime.DB())
	created, err := service.Create(
		context.Background(),
		userID,
		"rotation-revision",
		[]string{"mcp.access"},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	var staleHash string
	if err := runtime.DB().QueryRow(
		"SELECT key_hash FROM api_keys WHERE id=$1",
		created.Key.ID,
	).Scan(&staleHash); err != nil {
		t.Fatal(err)
	}
	rotated, err := service.Rotate(
		context.Background(),
		created.Key.ID,
		0,
		Access{UserID: userID, SuperAdmin: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.rotate(
		context.Background(),
		created.Key.ID,
		0,
		staleHash,
	); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale rotation error = %v, want ErrConflict", err)
	}
	if _, err := service.Authenticate(
		context.Background(),
		rotated.Secret,
	); err != nil {
		t.Fatalf("winning rotated secret was rejected: %v", err)
	}
}
