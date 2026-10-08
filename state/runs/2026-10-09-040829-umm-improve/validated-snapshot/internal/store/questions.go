package store

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// A thought can now say that it is a question, and another can say that it
// answers one. Both are marked, not inferred: umm does not read a note and
// decide it is asking something. A sentence ending in a question mark is often
// not a question, and a question is often written without one.
//
// `kind` was reaching the database straight from the request body, which is the
// same hole `relation` had — anything at all could be stored in it, including a
// five-thousand-character string. Nothing read it back, so nothing noticed.

// Kind is what a thought is.
type Kind string

const (
	// KindThought is an ordinary note, and the default.
	KindThought Kind = "thought"
	// KindQuestion is something the person wants answered. Marked by hand,
	// because only they know whether they are asking or musing.
	KindQuestion Kind = "question"
	// KindIdea is written when a Dream is materialised into the space.
	KindIdea Kind = "idea"
)

// ErrUnknownKind is returned for a kind outside the vocabulary, so callers map
// it to a 400 rather than a 500.
var ErrUnknownKind = errors.New("unknown note kind")

var knownKinds = map[Kind]bool{KindThought: true, KindQuestion: true, KindIdea: true}

// Kinds lists the vocabulary in a stable order for the API to advertise.
func Kinds() []Kind { return []Kind{KindThought, KindQuestion, KindIdea} }

// ParseKind accepts what a client sent and returns the kind it names.
//
// Empty means the client did not choose, which is the ordinary case, and becomes
// a plain thought. Anything else must name a real kind: quietly rewriting an
// unrecognised one would file a thought as something the person did not say it
// was, and hide the mistake from whoever sent it.
func ParseKind(value string) (Kind, error) {
	trimmed := Kind(strings.ToLower(strings.TrimSpace(value)))
	if trimmed == "" {
		return KindThought, nil
	}
	if !knownKinds[trimmed] {
		return "", ErrUnknownKind
	}
	return trimmed, nil
}

// OpenQuestion is a question nothing has answered yet.
type OpenQuestion struct {
	Note    Note      `json:"note"`
	SpaceID uuid.UUID `json:"spaceId"`
	Space   string    `json:"space"`
	// Attempts counts connections pointing at the question that are not answers —
	// thoughts that circle it without settling it. It is the difference between a
	// question nobody has touched and one that has been argued over.
	Attempts int `json:"attempts"`
}

// OpenQuestions lists questions with no answer recorded against them.
//
// Open means nobody has drawn an answer, not that umm searched for one and found
// none. A caller must not present an empty result as "everything is answered".
func (s *Store) OpenQuestions(ctx context.Context, userID uuid.UUID, spaceID *uuid.UUID) ([]OpenQuestion, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT n.id,n.space_id,n.author_id,n.content,n.title,n.color,n.kind,n.source,n.ai_excluded,
		       n.x,n.y,n.width,n.height,n.rotation,n.version,n.created_at,n.updated_at,
		       sp.name,
		       (SELECT count(*) FROM note_edges e
		          WHERE e.target_note_id=n.id AND e.relation<>'answers') AS attempts
		FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$1
		WHERE n.kind='question' AND n.deleted_at IS NULL
		  AND (sp.owner_id=$1 OR m.user_id=$1)
		  AND ($2::uuid IS NULL OR n.space_id=$2)
		  AND NOT EXISTS(
		    SELECT 1 FROM note_edges e
		    JOIN notes a ON a.id=e.source_note_id AND a.deleted_at IS NULL
		    WHERE e.target_note_id=n.id AND e.relation='answers')
		ORDER BY n.created_at DESC
		LIMIT 200`, userID, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OpenQuestion{}
	for rows.Next() {
		var item OpenQuestion
		if err := rows.Scan(&item.Note.ID, &item.Note.SpaceID, &item.Note.AuthorID, &item.Note.Content,
			&item.Note.Title, &item.Note.Color, &item.Note.Kind, &item.Note.Source, &item.Note.AIExcluded,
			&item.Note.X, &item.Note.Y, &item.Note.Width, &item.Note.Height, &item.Note.Rotation,
			&item.Note.Version, &item.Note.CreatedAt, &item.Note.UpdatedAt,
			&item.Space, &item.Attempts); err != nil {
			return nil, err
		}
		item.SpaceID = item.Note.SpaceID
		out = append(out, item)
	}
	return out, rows.Err()
}
