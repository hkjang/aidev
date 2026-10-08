package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSpaceEventsRecheckAccessBeforeReturningPayloadIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()

	ownerID, memberID, spaceID := uuid.New(), uuid.New(), uuid.New()
	ownerName, memberName := "event_owner_"+ownerID.String(), "event_member_"+memberID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text),($3,$4::citext,$4::text)`, ownerID, ownerName, memberID, memberName); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1 OR id=$2`, ownerID, memberID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'event access boundary')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'view')`, spaceID, memberID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO space_events(space_id,actor_id,event_type,payload) VALUES($1,$2,'comment.created','{"body":"visible before removal"}')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}

	events, allowed, err := db.SpaceEvents(ctx, memberID, spaceID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed || len(events) != 1 || !strings.Contains(string(events[0].Payload), "visible before removal") {
		t.Fatalf("authorized event batch = allowed:%v events:%#v", allowed, events)
	}
	last := events[0].Sequence

	if _, err = db.Pool.Exec(ctx, `DELETE FROM space_members WHERE space_id=$1 AND user_id=$2`, spaceID, memberID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO space_events(space_id,actor_id,event_type,payload) VALUES($1,$2,'comment.created','{"body":"secret after removal"}')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	events, allowed, err = db.SpaceEvents(ctx, memberID, spaceID, last, 100)
	if err != nil {
		t.Fatal(err)
	}
	if allowed || len(events) != 0 {
		t.Fatalf("revoked member received event payloads: allowed:%v events:%#v", allowed, events)
	}
}
