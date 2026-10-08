package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

// A thought arrives before you know where it belongs. Requiring a space first
// means the thought waits while you decide, and a thought that waits usually
// gets written somewhere else instead.
//
// Capture writes it down immediately, into a space that exists for exactly this
// purpose, and leaves the question of where it belongs for later — with a
// suggestion when umm is in a position to make one.

// ErrInboxSpace is returned when an operation would destroy the place a
// person's captures land.
var ErrInboxSpace = errors.New("the inbox space cannot be removed")

const inboxSpaceName = "생각 수집함"

// InboxSpace returns the user's capture space, creating it the first time.
//
// It is an ordinary space, so search, embeddings, Dream and connections all work
// on captured thoughts without knowing anything about capture.
func (s *Store) InboxSpace(ctx context.Context, userID uuid.UUID) (Space, error) {
	var space Space
	err := s.Pool.QueryRow(ctx,
		`SELECT id,owner_id,name,color,ai_excluded,is_inbox FROM spaces WHERE owner_id=$1 AND is_inbox`, userID).
		Scan(&space.ID, &space.OwnerID, &space.Name, &space.Color, &space.AIExcluded, &space.IsInbox)
	if err == nil {
		return space, nil
	}
	// The unique index makes this safe under a race: two concurrent captures
	// both try to insert, one wins, and the loser reads the winner's row.
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO spaces(owner_id,name,color,is_inbox) VALUES($1,$2,'#FFF0A8',true)
		ON CONFLICT (owner_id) WHERE is_inbox DO UPDATE SET name=spaces.name
		RETURNING id,owner_id,name,color,ai_excluded,is_inbox`, userID, inboxSpaceName).
		Scan(&space.ID, &space.OwnerID, &space.Name, &space.Color, &space.AIExcluded, &space.IsInbox)
	return space, err
}

// CaptureThought writes a thought down without asking where it goes.
func (s *Store) CaptureThought(ctx context.Context, userID uuid.UUID, content string) (Note, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Note{}, errors.New("empty thought")
	}
	space, err := s.InboxSpace(ctx, userID)
	if err != nil {
		return Note{}, fmt.Errorf("resolve inbox: %w", err)
	}
	// Stack captures down the canvas so a run of them stays readable if the
	// person opens the inbox as a space rather than as a list.
	var count int
	if err = s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM notes WHERE space_id=$1 AND deleted_at IS NULL`, space.ID).Scan(&count); err != nil {
		return Note{}, err
	}
	note := Note{SpaceID: space.ID, AuthorID: userID, Content: content}
	note.X, note.Y = 120, float64(120+(count%12)*70)
	return s.CreateNote(ctx, userID, note)
}

// MoveNote files a thought into another space.
//
// Connections are scoped to a space and both endpoints must live in it, so a
// note that leaves takes its connections with it in the only way the model
// allows: they are removed. The count comes back so a caller can say so before
// the person agrees, rather than discovering it afterwards.
func (s *Store) MoveNote(ctx context.Context, userID, noteID, targetSpaceID uuid.UUID) (Note, int, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Note{}, 0, err
	}
	defer tx.Rollback(ctx)

	var sourceSpaceID uuid.UUID
	if err = tx.QueryRow(ctx, `
		SELECT n.space_id FROM notes n
		JOIN spaces s ON s.id=n.space_id
		LEFT JOIN space_members m ON m.space_id=s.id AND m.user_id=$2
		WHERE n.id=$1 AND n.deleted_at IS NULL
		  AND (s.owner_id=$2 OR m.permission IN ('edit','manage'))`, noteID, userID).Scan(&sourceSpaceID); err != nil {
		return Note{}, 0, err
	}
	if sourceSpaceID == targetSpaceID {
		return Note{}, 0, errors.New("the note is already in that space")
	}
	// Write access to the destination is a separate question from write access
	// to the origin, and both have to hold.
	var allowed bool
	if err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM spaces s LEFT JOIN space_members m ON m.space_id=s.id AND m.user_id=$2
			WHERE s.id=$1 AND (s.owner_id=$2 OR m.permission IN ('edit','manage')))`,
		targetSpaceID, userID).Scan(&allowed); err != nil {
		return Note{}, 0, err
	}
	if !allowed {
		return Note{}, 0, errors.New("no write access to the destination space")
	}

	removed := 0
	if err = tx.QueryRow(ctx, `
		WITH gone AS (
			DELETE FROM note_edges WHERE source_note_id=$1 OR target_note_id=$1 RETURNING 1
		) SELECT count(*) FROM gone`, noteID).Scan(&removed); err != nil {
		return Note{}, 0, err
	}

	var note Note
	if err = tx.QueryRow(ctx, `
		UPDATE notes SET space_id=$2,version=version+1,updated_at=now()
		WHERE id=$1 AND deleted_at IS NULL
		RETURNING id,space_id,author_id,content,title,color,kind,source,ai_excluded,
		          x,y,width,height,rotation,version,created_at,updated_at`, noteID, targetSpaceID).
		Scan(&note.ID, &note.SpaceID, &note.AuthorID, &note.Content, &note.Title, &note.Color, &note.Kind,
			&note.Source, &note.AIExcluded, &note.X, &note.Y, &note.Width, &note.Height, &note.Rotation,
			&note.Version, &note.CreatedAt, &note.UpdatedAt); err != nil {
		return Note{}, 0, err
	}
	// Both canvases change, so both have to hear about it: one loses a note and
	// the other gains one.
	if err = s.AppendSpaceEvent(ctx, tx, userID, sourceSpaceID, "note.deleted", noteID, map[string]any{"id": noteID}); err != nil {
		return Note{}, 0, err
	}
	if err = s.AppendSpaceEvent(ctx, tx, userID, targetSpaceID, "note.created", note.ID, note); err != nil {
		return Note{}, 0, err
	}
	return note, removed, tx.Commit(ctx)
}

// SpaceSuggestion is one candidate home for a captured thought.
type SpaceSuggestion struct {
	Space Space   `json:"space"`
	Score float64 `json:"score"`
	// Basis says what the ranking is built on, because the honest answer changes
	// with the embedding backend and a reader should not have to guess.
	Basis string `json:"basis"`
}

const (
	// BasisMeaning: ranked by how close the thought sits to what the space holds.
	BasisMeaning = "meaning"
	// BasisRecent: umm is not in a position to judge meaning, so these are simply
	// the spaces most recently worked in. Presenting word overlap as if it were
	// understanding would be worse than admitting the limit.
	BasisRecent = "recent"
)

// SuggestSpaces ranks where a captured thought might belong.
//
// When the embedding backend has been measured as semantic, the ranking is by
// closeness to each space's existing thoughts. When it has not, umm says so by
// falling back to recency rather than dressing up vocabulary overlap as
// understanding — the same line auto-link draws.
func (s *Store) SuggestSpaces(ctx context.Context, userID, noteID uuid.UUID, limit int) ([]SpaceSuggestion, error) {
	if limit < 1 || limit > 10 {
		limit = 3
	}
	// This ranks one thought against every space, so it needs all of them
	// current — a space still waiting to be indexed would score low and look
	// like a poor home rather than an unindexed one. Indexing is a background
	// job since v0.72.0, so this catches up first: one bounded pass, which in
	// the settled case is a query that finds nothing.
	s.SweepEmbeddingsOnce(ctx, nil)
	spaces, err := s.ListSpaces(ctx, userID)
	if err != nil {
		return nil, err
	}
	candidates := make([]Space, 0, len(spaces))
	for _, space := range spaces {
		if space.ID == noteID {
			continue
		}
		candidates = append(candidates, space)
	}

	report, err := s.MeasureEmbeddingQuality(ctx, false)
	if err != nil || !report.Semantic {
		return s.recentSpaces(ctx, userID, noteID, limit)
	}

	// Listing a space is what causes its notes to be embedded, so the thought's
	// own space has to be listed too. Reading it straight from the notes table
	// leaves it without a vector, and every comparison silently scores zero — a
	// ranking that looks ordered but is not.
	own, _, err := s.ListNotes(ctx, userID, noteSpaceID(ctx, s, noteID), "")
	if err != nil {
		return nil, err
	}
	var note Note
	found := false
	for _, candidate := range own {
		if candidate.ID == noteID {
			note, found = candidate, true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("thought %s is not readable", noteID)
	}

	// Gather every candidate note first, then resolve vectors once. Per-space
	// calls could each settle on a different stored algorithm, which would make
	// the scores incomparable across the very spaces being ranked.
	type spaceNotes struct {
		space Space
		notes []Note
	}
	groups := make([]spaceNotes, 0, len(candidates))
	all := append([]Note(nil), note)
	for _, space := range candidates {
		if space.ID == note.SpaceID {
			continue
		}
		notes, _, listErr := s.ListNotes(ctx, userID, space.ID, "")
		if listErr != nil || len(notes) == 0 {
			continue
		}
		groups = append(groups, spaceNotes{space, notes})
		all = append(all, notes...)
	}
	vectors := s.loadEmbeddings(ctx, all)
	if len(vectors[note.ID]) == 0 {
		// No vector for the thought means nothing can be compared to it. Say so by
		// falling back rather than returning a list of zeroes ordered by nothing.
		return s.recentSpaces(ctx, userID, noteID, limit)
	}

	suggestions := []SpaceSuggestion{}
	for _, group := range groups {
		best := 0.0
		for _, other := range group.notes {
			if score := intelligence.Cosine(vectors[note.ID], vectors[other.ID]); score > best {
				best = score
			}
		}
		suggestions = append(suggestions, SpaceSuggestion{Space: group.space, Score: best, Basis: BasisMeaning})
	}
	sort.Slice(suggestions, func(i, j int) bool { return suggestions[i].Score > suggestions[j].Score })
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}
	return suggestions, nil
}

// noteSpaceID resolves which space a thought lives in.
func noteSpaceID(ctx context.Context, s *Store, noteID uuid.UUID) uuid.UUID {
	var spaceID uuid.UUID
	_ = s.Pool.QueryRow(ctx, `SELECT space_id FROM notes WHERE id=$1 AND deleted_at IS NULL`, noteID).Scan(&spaceID)
	return spaceID
}

// recentSpaces is the honest fallback: the places this person has been working.
func (s *Store) recentSpaces(ctx context.Context, userID, noteID uuid.UUID, limit int) ([]SpaceSuggestion, error) {
	var currentSpace uuid.UUID
	_ = s.Pool.QueryRow(ctx, `SELECT space_id FROM notes WHERE id=$1`, noteID).Scan(&currentSpace)
	rows, err := s.Pool.Query(ctx, `
		SELECT s.id,s.owner_id,s.name,s.color,s.ai_excluded,s.is_inbox
		FROM spaces s
		LEFT JOIN space_members m ON m.space_id=s.id AND m.user_id=$1
		WHERE (s.owner_id=$1 OR m.user_id=$1) AND NOT s.is_inbox AND s.id<>$2
		ORDER BY COALESCE((SELECT max(updated_at) FROM notes n WHERE n.space_id=s.id AND n.deleted_at IS NULL), s.created_at) DESC
		LIMIT $3`, userID, currentSpace, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	suggestions := []SpaceSuggestion{}
	for rows.Next() {
		var suggestion SpaceSuggestion
		if err := rows.Scan(&suggestion.Space.ID, &suggestion.Space.OwnerID, &suggestion.Space.Name,
			&suggestion.Space.Color, &suggestion.Space.AIExcluded, &suggestion.Space.IsInbox); err != nil {
			return nil, err
		}
		suggestion.Basis = BasisRecent
		suggestions = append(suggestions, suggestion)
	}
	return suggestions, rows.Err()
}
