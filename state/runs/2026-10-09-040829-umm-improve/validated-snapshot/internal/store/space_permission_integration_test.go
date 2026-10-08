package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// A screen cannot work out what someone may do with a shared space. The owner
// is obvious from the owner id, but a member who may edit and a member who may
// only read look identical from outside — so the canvas showed a read-only
// member the editor, and they learnt the truth by typing into it and watching
// the text revert.
func TestListSpacesSaysWhatThePersonMayDoIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedStore(t, dsn)

	ownerID, readerID, editorID := uuid.New(), uuid.New(), uuid.New()
	name := func(id uuid.UUID) string { return "perm_" + strings.ReplaceAll(id.String(), "-", "") }
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text),($3,$4::citext,$4::text),($5,$6::citext,$6::text)`,
		ownerID, name(ownerID), readerID, name(readerID), editorID, name(editorID)); err != nil {
		t.Fatal(err)
	}
	spaceID := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'공유 공간')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view'),($1,$3,'edit')`,
		spaceID, readerID, editorID); err != nil {
		t.Fatal(err)
	}

	permissionFor := func(userID uuid.UUID) string {
		t.Helper()
		spaces, err := db.ListSpaces(ctx, userID)
		if err != nil {
			t.Fatal(err)
		}
		for _, space := range spaces {
			if space.ID == spaceID {
				return space.Permission
			}
		}
		t.Fatalf("the space is missing from %s's listing", userID)
		return ""
	}

	for _, tc := range []struct {
		who  uuid.UUID
		want string
		note string
	}{
		{ownerID, "manage", "the owner"},
		{editorID, "edit", "a member who may write"},
		{readerID, "view", "a member who may only read"},
	} {
		if got := permissionFor(tc.who); got != tc.want {
			t.Errorf("%s: permission %q, want %q", tc.note, got, tc.want)
		}
	}
}

// The permission has to come from the same row the listing is built from, or
// the two can drift into describing different things.
func TestSpacePermissionFollowsAChangedMembershipIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedStore(t, dsn)

	ownerID, memberID, spaceID := uuid.New(), uuid.New(), uuid.New()
	owner := "perm_owner_" + strings.ReplaceAll(ownerID.String(), "-", "")
	member := "perm_member_" + strings.ReplaceAll(memberID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text),($3,$4::citext,$4::text)`,
		ownerID, owner, memberID, member); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'권한 변경')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view')`, spaceID, memberID); err != nil {
		t.Fatal(err)
	}

	read := func() string {
		spaces, err := db.ListSpaces(ctx, memberID)
		if err != nil {
			t.Fatal(err)
		}
		if len(spaces) != 1 {
			t.Fatalf("expected the one shared space, got %d", len(spaces))
		}
		return spaces[0].Permission
	}
	if got := read(); got != "view" {
		t.Fatalf("before the change: %q", got)
	}
	if _, err := db.Pool.Exec(ctx, `UPDATE space_members SET permission='edit' WHERE space_id=$1 AND user_id=$2`, spaceID, memberID); err != nil {
		t.Fatal(err)
	}
	if got := read(); got != "edit" {
		t.Fatalf("after being given write access: %q", got)
	}
}
