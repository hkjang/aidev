package server

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestPostgresCollaborationPresenceNotificationBoundaries(t *testing.T) {
	s, _, _, id, session, _, state := collaborationJournalFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	listener, err := pgx.ConnectConfig(ctx, s.DB.Config().ConnConfig.Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close(context.Background())
	var channel string
	if err = listener.QueryRow(ctx, "SELECT 'madi_collab_'||md5(current_schema())").Scan(&channel); err != nil {
		t.Fatal(err)
	}
	if _, err = listener.Exec(ctx, "LISTEN "+pgx.Identifier{channel}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	expect := func(payload string) {
		t.Helper()
		wait, done := context.WithTimeout(ctx, 3*time.Second)
		defer done()
		event, err := listener.WaitForNotification(wait)
		if err != nil {
			t.Fatal(err)
		}
		if event.Payload != payload {
			t.Fatalf("notification %q, expected %q", event.Payload, payload)
		}
	}
	connection := newID()
	if _, err = s.DB.Exec(ctx, `INSERT INTO collaboration_presence(connection_id,document_id,user_id,epoch) SELECT $1,$2,user_id,$3 FROM sessions WHERE token_hash=$4`, connection, id, state.Epoch, session); err != nil {
		t.Fatal(err)
	}
	expect(id)
	for range 30 {
		if _, err = s.DB.Exec(ctx, `UPDATE collaboration_presence SET touched_at=clock_timestamp(),operations=operations+1,operation_window=clock_timestamp() WHERE connection_id=$1`, connection); err != nil {
			t.Fatal(err)
		}
	}
	// An ordered barrier proves all 30 committed heartbeat/quota updates were
	// silent without sleeping or mistaking a slow notification for its absence.
	if _, err = s.DB.Exec(ctx, "SELECT pg_notify($1,'test-barrier')", channel); err != nil {
		t.Fatal(err)
	}
	expect("test-barrier")
	for _, sql := range []string{
		"UPDATE collaboration_presence SET client_id=7 WHERE connection_id=$1",
		"UPDATE collaboration_presence SET clock=clock+1 WHERE connection_id=$1",
		"UPDATE collaboration_presence SET state='{\"cursor\":null}'::jsonb WHERE connection_id=$1",
		"UPDATE collaboration_presence SET epoch='" + newID() + "' WHERE connection_id=$1",
	} {
		if _, err = s.DB.Exec(ctx, sql, connection); err != nil {
			t.Fatal(err)
		}
		expect(id)
	}
	if _, err = s.DB.Exec(ctx, "UPDATE users SET name=name WHERE id=(SELECT user_id FROM sessions WHERE token_hash=$1)", session); err != nil {
		t.Fatal(err)
	}
	expect("*")
	if _, err = s.DB.Exec(ctx, "DELETE FROM collaboration_presence WHERE connection_id=$1", connection); err != nil {
		t.Fatal(err)
	}
	expect(id)
}
