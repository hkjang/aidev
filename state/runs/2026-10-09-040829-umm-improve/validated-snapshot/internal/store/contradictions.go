package store

import (
	"context"

	"github.com/google/uuid"
)

// umm has been able to record that two thoughts contradict each other since the
// connection vocabulary arrived, and nothing has read those back. They show on
// the canvas as a labelled line like any other connection, which means the
// question a person actually wants to ask — what disagrees in here — has no
// answer.
//
// This reports disagreements that were *recorded*. umm does not detect them:
// nothing reads two notes and concludes they conflict. That difference decides
// how the result may be presented — an empty list means nobody has marked any,
// not that a workspace is free of contradictions, so a caller must not render a
// zero beside the word.

// Contradiction is a recorded disagreement between two thoughts.
type Contradiction struct {
	EdgeID  uuid.UUID `json:"edgeId"`
	SpaceID uuid.UUID `json:"spaceId"`
	Space   string    `json:"space"`
	// Claim and Counter are the two sides, in the direction the connection was
	// drawn: Claim contradicts Counter.
	Claim   Note `json:"claim"`
	Counter Note `json:"counter"`
	// Origin says who recorded the disagreement — a person, an agent, or umm.
	Origin Origin `json:"origin"`
}

// Contradictions lists the disagreements recorded in the reachable spaces.
//
// A nil spaceID covers everything the person can see, which is what a review
// screen wants; a specific one covers a single canvas.
func (s *Store) Contradictions(ctx context.Context, userID uuid.UUID, spaceID *uuid.UUID) ([]Contradiction, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT e.id, sp.id, sp.name, e.origin,
		       a.id,a.space_id,a.author_id,a.content,a.title,a.color,a.kind,a.source,a.ai_excluded,
		       a.x,a.y,a.width,a.height,a.rotation,a.version,a.created_at,a.updated_at,
		       b.id,b.space_id,b.author_id,b.content,b.title,b.color,b.kind,b.source,b.ai_excluded,
		       b.x,b.y,b.width,b.height,b.rotation,b.version,b.created_at,b.updated_at
		FROM note_edges e
		JOIN spaces sp ON sp.id=e.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$1
		JOIN notes a ON a.id=e.source_note_id AND a.deleted_at IS NULL
		JOIN notes b ON b.id=e.target_note_id AND b.deleted_at IS NULL
		WHERE e.relation='contradicts'
		  AND (sp.owner_id=$1 OR m.user_id=$1)
		  AND ($2::uuid IS NULL OR e.space_id=$2)
		ORDER BY e.created_at DESC
		LIMIT 200`, userID, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Contradiction{}
	for rows.Next() {
		var item Contradiction
		if err := rows.Scan(&item.EdgeID, &item.SpaceID, &item.Space, &item.Origin,
			&item.Claim.ID, &item.Claim.SpaceID, &item.Claim.AuthorID, &item.Claim.Content, &item.Claim.Title,
			&item.Claim.Color, &item.Claim.Kind, &item.Claim.Source, &item.Claim.AIExcluded,
			&item.Claim.X, &item.Claim.Y, &item.Claim.Width, &item.Claim.Height, &item.Claim.Rotation,
			&item.Claim.Version, &item.Claim.CreatedAt, &item.Claim.UpdatedAt,
			&item.Counter.ID, &item.Counter.SpaceID, &item.Counter.AuthorID, &item.Counter.Content, &item.Counter.Title,
			&item.Counter.Color, &item.Counter.Kind, &item.Counter.Source, &item.Counter.AIExcluded,
			&item.Counter.X, &item.Counter.Y, &item.Counter.Width, &item.Counter.Height, &item.Counter.Rotation,
			&item.Counter.Version, &item.Counter.CreatedAt, &item.Counter.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
