package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type webhookOutboxEvent struct {
	ID         uuid.UUID       `json:"id"`
	Type       string          `json:"type"`
	SpaceID    uuid.UUID       `json:"spaceId"`
	ResourceID uuid.UUID       `json:"resourceId,omitempty"`
	ActorID    uuid.UUID       `json:"actorId"`
	Data       json.RawMessage `json:"data"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// SpaceEvent is the access-scoped collaboration record sent over SSE.
type SpaceEvent struct {
	Sequence   int64           `json:"sequence"`
	Type       string          `json:"type"`
	ResourceID *uuid.UUID      `json:"resourceId"`
	ActorID    *uuid.UUID      `json:"actorId"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// IdempotencyReservation identifies the pending HTTP response that must be
// completed in the same transaction as its domain mutation.
type IdempotencyReservation struct {
	UserID    uuid.UUID
	Key       string
	Method    string
	Path      string
	CreatedAt time.Time
	Status    int
}

type idempotencyReservationContextKey struct{}

func WithIdempotencyReservation(ctx context.Context, reservation IdempotencyReservation) context.Context {
	return context.WithValue(ctx, idempotencyReservationContextKey{}, reservation)
}

func idempotencyReservationFrom(ctx context.Context) (IdempotencyReservation, bool) {
	reservation, ok := ctx.Value(idempotencyReservationContextKey{}).(IdempotencyReservation)
	return reservation, ok
}

// SpaceEvents reads one bounded event batch and checks membership in the same
// PostgreSQL statement. A stream therefore cannot observe an event batch from
// a snapshot in which its caller no longer has access to the space.
func (s *Store) SpaceEvents(ctx context.Context, userID, spaceID uuid.UUID, after int64, limit int) ([]SpaceEvent, bool, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	var allowed bool
	var raw json.RawMessage
	err := s.Pool.QueryRow(ctx, `
		WITH access AS (
		  SELECT EXISTS(
		    SELECT 1 FROM spaces sp
		    LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=$1
		    WHERE sp.id=$2 AND (sp.owner_id=$1 OR sm.user_id=$1)
		  ) AS allowed
		), pending AS (
		  SELECT e.sequence,e.event_type,e.resource_id,e.payload,e.actor_id,e.created_at
		  FROM space_events e CROSS JOIN access a
		  WHERE a.allowed AND e.space_id=$2 AND e.sequence>$3
		  ORDER BY e.sequence
		  LIMIT $4
		)
		SELECT a.allowed,COALESCE(
		  jsonb_agg(jsonb_build_object(
		    'sequence',p.sequence,'type',p.event_type,'resourceId',p.resource_id,
		    'payload',p.payload,'actorId',p.actor_id,'createdAt',p.created_at
		  ) ORDER BY p.sequence) FILTER (WHERE p.sequence IS NOT NULL),
		  '[]'::jsonb
		)
		FROM access a LEFT JOIN pending p ON true
		GROUP BY a.allowed`, userID, spaceID, after, limit).Scan(&allowed, &raw)
	if err != nil {
		return nil, false, err
	}
	events := []SpaceEvent{}
	if err = json.Unmarshal(raw, &events); err != nil {
		return nil, false, err
	}
	return events, allowed, nil
}

// AppendSpaceEvent writes the collaboration log, every currently eligible
// webhook delivery and any HTTP idempotency result into the caller's mutation
// transaction. Callers must not commit their domain change when this returns an
// error. Supported idempotent routes return this same payload to the client.
func (s *Store) AppendSpaceEvent(ctx context.Context, tx pgx.Tx, actorID, spaceID uuid.UUID, eventType string, resourceID uuid.UUID, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var resource any = resourceID
	if resourceID == uuid.Nil {
		resource = nil
	}
	// The collaboration log gets the fact of the change, not the writing in it.
	//
	// The three readers of `raw` want different things. A webhook subscriber
	// asked for the note and gets it, and those deliveries are cleaned up. An
	// idempotent replay has to return the same HTTP response, and those records
	// expire in a day. space_events is read by exactly one consumer — the open
	// canvas, which looks at actorId and refetches — and it is the only one of
	// the three that is never cleaned up.
	//
	// So it was keeping a copy of every version of everybody's writing for
	// good, including notes their author had since deleted, in a table no
	// retention policy covered. Storing nothing costs that reader nothing.
	if _, err = tx.Exec(ctx, `INSERT INTO space_events(space_id,actor_id,event_type,resource_id,payload) VALUES($1,$2,$3,$4,'{}'::jsonb)`, spaceID, actorID, eventType, resource); err != nil {
		return err
	}
	event := webhookOutboxEvent{
		ID: uuid.New(), Type: eventType, SpaceID: spaceID, ResourceID: resourceID,
		ActorID: actorID, Data: raw, CreatedAt: time.Now().UTC(),
	}
	eventRaw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO webhook_deliveries(subscription_id,event_id,event_type,payload,status,next_attempt_at)
		SELECT ws.id,$2,$3,$4,'queued',now() FROM webhook_subscriptions ws
		JOIN users owner_user ON owner_user.id=ws.owner_id AND owner_user.active
		WHERE ws.active AND ($3=ANY(ws.events) OR '*'=ANY(ws.events))
		  AND EXISTS(
		    SELECT 1 FROM spaces sp LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=ws.owner_id
		    WHERE sp.id=$1 AND (sp.owner_id=ws.owner_id OR sm.user_id=ws.owner_id))
		ON CONFLICT(subscription_id,event_id) DO NOTHING`, spaceID, event.ID, eventType, json.RawMessage(eventRaw))
	if err != nil {
		return err
	}
	reservation, ok := idempotencyReservationFrom(ctx)
	if !ok {
		return nil
	}
	var responseBody any
	if reservation.Status != 204 {
		responseBody = json.RawMessage(raw)
	}
	command, err := tx.Exec(ctx, `
		UPDATE idempotency_records
		SET state='completed',response_status=$6,response_body=$7,expires_at=now()+interval '24 hours'
		WHERE user_id=$1 AND idempotency_key=$2 AND method=$3 AND path=$4
		  AND state='pending' AND created_at=$5`,
		reservation.UserID, reservation.Key, reservation.Method, reservation.Path,
		reservation.CreatedAt, reservation.Status, responseBody)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("idempotency reservation is no longer active")
	}
	return nil
}
