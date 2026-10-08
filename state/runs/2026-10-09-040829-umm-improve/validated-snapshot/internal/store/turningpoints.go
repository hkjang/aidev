package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// What changed in someone's thinking, in the order it changed.
//
// Six months later the question is rarely "what did I write" — it is "what did
// we decide about this, and why". The reason is the part that goes missing
// first, and by the time it is missed the people who knew it have forgotten.
//
// Everything here was marked by a person. umm does not read a burst of activity
// and call it a turning point, and it does not decide that a line went quiet
// because it was abandoned. A record that mixes what someone decided with what
// software guessed is worse than no record, because you cannot tell which is
// which when you need it.

// Turning point kinds. Each is something a person did deliberately.
const (
	// TurningAdopted: a line of thinking was taken.
	TurningAdopted = "adopted"
	// TurningAbandoned: a line was considered and set aside.
	TurningAbandoned = "abandoned"
	// TurningAnswered: a question was marked as answered by a thought.
	TurningAnswered = "answered"
	// TurningContradicted: two thoughts were marked as being in tension.
	TurningContradicted = "contradicted"
)

// TurningPoint is one thing that changed, and what it was about.
type TurningPoint struct {
	Kind    string    `json:"kind"`
	At      time.Time `json:"at"`
	SpaceID uuid.UUID `json:"spaceId"`
	Space   string    `json:"space"`
	// Subject is what changed: the line's name, the question, or the claim.
	Subject string `json:"subject"`
	// Detail is the other half: why a line was resolved, the thought that
	// answered a question, or the counter-claim. Empty when there is no second
	// half rather than filled with something invented.
	Detail string `json:"detail"`
	// NoteID is where to go to see it, when a thought is involved.
	NoteID *uuid.UUID `json:"noteId,omitempty"`
}

// maxTurningPoints bounds one read. A record nobody can scroll to the end of is
// a record nobody reads.
const maxTurningPoints = 120

// TurningPoints returns what a person marked, newest first.
//
// The four kinds are unioned in SQL rather than fetched separately and merged in
// Go, so the limit applies to the merged order. Taking the newest 120 of each
// kind and then sorting would drop older entries of a rare kind in favour of
// newer entries of a common one, which reads as "nothing was decided that year".
// TurningPoints also reports whether there were more than one read returns.
// Everywhere else umm says what it left out — the morning review names what it
// did not examine, retrieval counts what it held back — and a record that
// silently stops at the newest hundred and twenty reads as the whole story.
func (s *Store) TurningPoints(ctx context.Context, userID uuid.UUID, spaceID *uuid.UUID) ([]TurningPoint, bool, error) {
	rows, err := s.Pool.Query(ctx, `
		WITH visible AS (
			SELECT sp.id, sp.name
			FROM spaces sp
			LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$1
			WHERE (sp.owner_id=$1 OR m.user_id=$1) AND ($2::uuid IS NULL OR sp.id=$2)
		)
		SELECT b.status, b.resolved_at, v.id, v.name, b.name, b.resolution, b.root_note_id
		FROM branches b JOIN visible v ON v.id=b.space_id
		WHERE b.status<>'open' AND b.resolved_at IS NOT NULL

		UNION ALL

		SELECT 'answered', e.created_at, v.id, v.name, q.content, a.content, q.id
		FROM note_edges e
		JOIN visible v ON v.id=e.space_id
		JOIN notes q ON q.id=e.target_note_id AND q.deleted_at IS NULL
		JOIN notes a ON a.id=e.source_note_id AND a.deleted_at IS NULL
		WHERE e.relation='answers'

		UNION ALL

		SELECT 'contradicted', e.created_at, v.id, v.name, c.content, d.content, c.id
		FROM note_edges e
		JOIN visible v ON v.id=e.space_id
		JOIN notes c ON c.id=e.source_note_id AND c.deleted_at IS NULL
		JOIN notes d ON d.id=e.target_note_id AND d.deleted_at IS NULL
		WHERE e.relation='contradicts'

		ORDER BY 2 DESC
		-- One past the cap, so "there were more" comes from the read itself.
		LIMIT $3`, userID, spaceID, maxTurningPoints+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	out := []TurningPoint{}
	for rows.Next() {
		var point TurningPoint
		if err := rows.Scan(&point.Kind, &point.At, &point.SpaceID, &point.Space,
			&point.Subject, &point.Detail, &point.NoteID); err != nil {
			return nil, false, err
		}
		out = append(out, point)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(out) > maxTurningPoints
	if more {
		out = out[:maxTurningPoints]
	}
	return out, more, nil
}
