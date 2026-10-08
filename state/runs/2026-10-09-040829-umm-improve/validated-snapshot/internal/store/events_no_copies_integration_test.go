package store

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// One payload feeds three readers with different needs, and getting that wrong
// is what made the collaboration log a second, permanent copy of everybody's
// writing. So each reader is checked for what it actually needs: the webhook
// subscriber asked for the note and must still get it, the idempotent replay
// has to return the same response, and the log must keep none of it.

func eventsSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	db := isolatedStore(t, dsn)
	ctx := context.Background()
	userID, spaceID := uuid.New(), uuid.New()
	name := "events_" + strings.ReplaceAll(userID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, name); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'이벤트 공간')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	return db, userID, spaceID
}

const secretSentence = "이 문장은 로그에 남으면 안 됩니다"

func TestCollaborationLogKeepsNoCopyOfTheWritingIntegration(t *testing.T) {
	db, userID, spaceID := eventsSpace(t)
	ctx := context.Background()

	note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: secretSentence})
	if err != nil {
		t.Fatal(err)
	}

	var payloads []string
	rows, err := db.Pool.Query(ctx, `SELECT payload::text FROM space_events WHERE space_id=$1`, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			t.Fatal(err)
		}
		payloads = append(payloads, raw)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(payloads) == 0 {
		t.Fatal("no event was written at all, so this proves nothing about what they contain")
	}
	for _, raw := range payloads {
		if strings.Contains(raw, secretSentence) {
			t.Fatalf("the collaboration log kept a copy of the note: %s", raw)
		}
	}

	// Editing writes another event, and the same has to hold for that one —
	// this is the row that used to accumulate on every drag.
	note.Content = secretSentence + " (고침)"
	if _, err := db.UpdateNote(ctx, userID, note, nil); err != nil {
		t.Fatal(err)
	}
	var leaked int
	if err := db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM space_events WHERE space_id=$1 AND payload::text LIKE '%'||$2||'%'`,
		spaceID, secretSentence).Scan(&leaked); err != nil {
		t.Fatal(err)
	}
	if leaked != 0 {
		t.Fatalf("%d events carry the note's text", leaked)
	}
}

// The subscriber asked for the note, and their deliveries are cleaned up. This
// is the half that must not change, or the fix above is a feature removal.
func TestWebhookDeliveryStillCarriesTheNoteIntegration(t *testing.T) {
	db, userID, spaceID := eventsSpace(t)
	ctx := context.Background()

	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO webhook_subscriptions(owner_id,name,url,secret_ciphertext,events,active)
		VALUES($1,'테스트 훅','https://example.com/hook','enc:x',ARRAY['note.created'],true)`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: secretSentence}); err != nil {
		t.Fatal(err)
	}

	var raw string
	if err := db.Pool.QueryRow(ctx,
		`SELECT payload::text FROM webhook_deliveries WHERE event_type='note.created' LIMIT 1`).Scan(&raw); err != nil {
		t.Fatalf("no delivery was queued: %v", err)
	}
	if !strings.Contains(raw, secretSentence) {
		t.Fatalf("the subscriber's delivery lost the note it subscribed to: %s", raw)
	}
	var body struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) == 0 || string(body.Data) == "{}" {
		t.Fatalf("the delivery body has no data: %s", raw)
	}
}

// And the log still records that something happened, which is the whole reason
// the table exists. Emptying the payload must not have emptied the table.
func TestCollaborationLogStillRecordsTheChangeIntegration(t *testing.T) {
	db, userID, spaceID := eventsSpace(t)
	ctx := context.Background()

	note, err := db.CreateNote(ctx, userID, Note{SpaceID: spaceID, AuthorID: userID, Content: secretSentence})
	if err != nil {
		t.Fatal(err)
	}
	events, allowed, err := db.SpaceEvents(ctx, userID, spaceID, 0, 100)
	if err != nil || !allowed {
		t.Fatalf("events: %v allowed=%v", err, allowed)
	}
	found := false
	for _, event := range events {
		if event.Type == "note.created" && event.ResourceID != nil && *event.ResourceID == note.ID {
			found = true
			if event.ActorID == nil || *event.ActorID != userID {
				t.Fatalf("the event does not say who made the change: %+v", event)
			}
		}
	}
	if !found {
		t.Fatal("the change was not recorded; the canvas would never learn about it")
	}
}
