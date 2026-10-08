package auth

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// Issuing, re-permissioning, rotating, revoking and finally being rid of a key.
//
// Revoking left the key on the screen for good: the listing returns every key a
// person has ever had, and a revoked one had no action left. Pressing 폐기 and
// watching the row stay reads as deletion not working, so a revoked key can now
// be removed — and only a revoked one, because stopping a key that still works
// must not be a single slip.
func TestAPIKeyLifecycleIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Close() })
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	userID := uuid.New()
	username := "keylife_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	service := &Service{Store: db}
	key, _, err := service.CreateKey(ctx, userID, "테스트 키", []string{"notes:read"}, 90)
	if err != nil {
		t.Fatal(err)
	}

	// Permissions can be changed after issue, without reissuing the key.
	if err = service.UpdateKeyScopes(ctx, userID, key.ID, []string{"notes:read", "notes:write"}); err != nil {
		t.Fatalf("changing the permissions: %v", err)
	}
	if scopes := scopesOf(t, db, key.ID); scopes != "notes:read,notes:write" {
		t.Fatalf("permissions did not change: %s", scopes)
	}

	// A revoked key cannot be re-permissioned: the scopes of a dead key are not
	// a thing anyone needs to edit, and allowing it invites the belief that it
	// still does something.
	second, _, err := service.CreateKey(ctx, userID, "두 번째 키", []string{"notes:read"}, 90)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.RevokeKey(ctx, userID, second.ID); err != nil {
		t.Fatal(err)
	}
	if err = service.UpdateKeyScopes(ctx, userID, second.ID, []string{"notes:write"}); err == nil {
		t.Error("a revoked key accepted a permission change")
	}

	// Forgetting only applies once revoked.
	if err = service.ForgetRevokedKey(ctx, userID, key.ID); err == nil {
		t.Error("a key that still works was removed from the list")
	}
	if count := keyCount(t, db, userID); count != 2 {
		t.Fatalf("expected both keys to remain, found %d", count)
	}

	if err = service.ForgetRevokedKey(ctx, userID, second.ID); err != nil {
		t.Fatalf("removing a revoked key: %v", err)
	}
	if count := keyCount(t, db, userID); count != 1 {
		t.Fatalf("the revoked key is still listed: %d keys remain", count)
	}

	// Revoking, then removing, leaves nothing behind.
	if err = service.RevokeKey(ctx, userID, key.ID); err != nil {
		t.Fatal(err)
	}
	if err = service.ForgetRevokedKey(ctx, userID, key.ID); err != nil {
		t.Fatal(err)
	}
	if count := keyCount(t, db, userID); count != 0 {
		t.Fatalf("%d keys remain after revoking and removing both", count)
	}
}

func scopesOf(t *testing.T, db *store.Store, keyID uuid.UUID) string {
	t.Helper()
	var scopes []string
	if err := db.Pool.QueryRow(context.Background(), `SELECT scopes FROM api_keys WHERE id=$1`, keyID).Scan(&scopes); err != nil {
		t.Fatal(err)
	}
	return strings.Join(scopes, ",")
}

func keyCount(t *testing.T, db *store.Store, userID uuid.UUID) int {
	t.Helper()
	var n int
	if err := db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM api_keys WHERE user_id=$1`, userID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
