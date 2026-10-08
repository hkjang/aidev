package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// What a space produced, and which thoughts each slide came from.
//
// Only the link. Ptium owns the presentation and its slides; a copy here would
// drift the moment someone edits the deck, and umm would go on confidently
// describing a slide that no longer says what it claims. What is kept is enough
// to answer two questions a person will actually ask: which talks came out of
// this space, and where did this slide's sentences come from.
//
// The second question is the one that matters most, because the slides carry
// the person's own words. Being able to get back to the note is being able to
// check that nothing was put in their mouth.

// Presentation link status vocabulary.
const (
	// PresentationPending: the deck exists in Ptium and umm has not compiled
	// into it yet.
	PresentationPending = "pending"
	// PresentationGenerating: Ptium is working on it.
	PresentationGenerating = "generating"
	// PresentationReady: the slides are there.
	PresentationReady = "ready"
	// PresentationFailed: it did not work, and Error says what Ptium said.
	PresentationFailed = "failed"
)

// ErrUnknownPresentationStatus is returned for a status outside the vocabulary,
// so the API answers 400 rather than 500 on a bad write.
var ErrUnknownPresentationStatus = errors.New("unknown presentation status")

func validPresentationStatus(status string) bool {
	switch status {
	case PresentationPending, PresentationGenerating, PresentationReady, PresentationFailed:
		return true
	}
	return false
}

// PresentationLink is one deck made from one space.
type PresentationLink struct {
	ID      uuid.UUID `json:"id"`
	SpaceID uuid.UUID `json:"spaceId"`
	// PtiumID is opaque here. umm never parses it, so Ptium is free to change
	// its shape.
	PtiumID string `json:"ptiumId"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	// FailureKind is what sort of failure it was. It decides the sentence a
	// person reads and whether retrying can help; Error holds the underlying
	// message, which is for whoever fixes it.
	FailureKind string `json:"failureKind,omitempty"`
	// CompiledSource is the deck source umm sent. Without it, "why does slide 4
	// say that" has no answer that does not involve guessing at a compiler run
	// that no longer exists.
	CompiledSource string `json:"compiledSource,omitempty"`
	ThoughtCount   int    `json:"thoughtCount"`
	ExcludedCount  int    `json:"excludedCount"`
	// TrimmedCount is how many thoughts a length cap left out. Apart from
	// ExcludedCount because the reason differs and so does the remedy: raising
	// the cap brings these back, and nothing brings back a thought its author
	// held out of analysis.
	TrimmedCount int        `json:"trimmedCount"`
	CreatedBy    *uuid.UUID `json:"createdBy,omitempty"`
	// Request is what the person asked for when they made this deck — the
	// selection, the length cap, the switches — as the compiler wrote it.
	// Opaque here, like the Ptium id: the store keeps it so that retrying into
	// a deck that failed can ask for the same talk again, and the compiler owns
	// what the shape means.
	//
	// Read back only where a retry needs it. A list of a space's talks does not
	// carry it, and neither does a response: it is the compiler's record of a
	// past request, not a field of the deck somebody is looking at.
	Request json.RawMessage `json:"-"`
}

// SlideSource is one thought that reached one slide.
type SlideSource struct {
	// SlidePosition rather than a Ptium slide id: a slide id changes when the
	// deck is recompiled and umm has no way to learn that it did. Position is
	// what umm actually knows, because umm wrote the source that produced it.
	SlidePosition int       `json:"slidePosition"`
	NoteID        uuid.UUID `json:"noteId"`
}

// CreatePresentationLink records that a space produced a deck, and what was
// asked for to produce it.
//
// Written before the deck is compiled, so a compile that fails leaves a link
// saying so rather than leaving a deck in Ptium that umm has no record of. The
// request is written here for the same reason: by the time somebody retries,
// the one that failed is gone, and a retry that has to guess at it makes a
// different talk.
func (s *Store) CreatePresentationLink(ctx context.Context, userID, spaceID uuid.UUID, ptiumID, title string, request json.RawMessage) (PresentationLink, error) {
	ptiumID = strings.TrimSpace(ptiumID)
	if ptiumID == "" {
		return PresentationLink{}, errors.New("ptium presentation id required")
	}
	if len(request) == 0 {
		// A caller with nothing to record leaves the same mark as every row
		// written before umm kept this: asked for nothing in particular.
		request = json.RawMessage("{}")
	}
	var link PresentationLink
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO presentation_links(space_id,ptium_presentation_id,title,created_by,request)
		SELECT $1,$3,$4,$2,$5
		WHERE EXISTS(
			SELECT 1 FROM spaces sp
			LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
			WHERE sp.id=$1 AND (sp.owner_id=$2 OR m.permission IN ('edit','manage')))
		RETURNING id,space_id,ptium_presentation_id,title,status,error,failure_kind,thought_count,excluded_count,trimmed_count,created_by,request`,
		spaceID, userID, ptiumID, strings.TrimSpace(title), request).
		Scan(&link.ID, &link.SpaceID, &link.PtiumID, &link.Title, &link.Status, &link.Error, &link.FailureKind,
			&link.ThoughtCount, &link.ExcludedCount, &link.TrimmedCount, &link.CreatedBy, &link.Request)
	if err != nil {
		return PresentationLink{}, err
	}
	return link, nil
}

// CompletePresentationLink records what compiling produced, together with the
// source and the mapping from slide to thought.
//
// One transaction: a link that says "ready" while its sources are missing would
// show a deck whose slides cannot say where they came from, which is exactly
// the claim this table exists to support.
func (s *Store) CompletePresentationLink(ctx context.Context, userID, linkID uuid.UUID, status, source string, sources []SlideSource, thoughtCount, excludedCount, trimmedCount int, failure, failureKind string) error {
	if !validPresentationStatus(status) {
		return ErrUnknownPresentationStatus
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `
		UPDATE presentation_links l
		SET status=$3, compiled_source=$4, thought_count=$5, excluded_count=$6, trimmed_count=$9, error=$7, failure_kind=$8, updated_at=now()
		WHERE l.id=$1
		  AND EXISTS(
			SELECT 1 FROM spaces sp
			LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
			WHERE sp.id=l.space_id AND (sp.owner_id=$2 OR m.permission IN ('edit','manage')))`,
		linkID, userID, status, source, thoughtCount, excludedCount, strings.TrimSpace(failure), strings.TrimSpace(failureKind), trimmedCount)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	// Replaced rather than added to: recompiling a deck produces a new mapping,
	// and keeping the old rows would have a slide claiming thoughts that are no
	// longer on it.
	if _, err := tx.Exec(ctx, `DELETE FROM presentation_sources WHERE presentation_link_id=$1`, linkID); err != nil {
		return err
	}
	for _, src := range sources {
		// A thought from another space would credit a slide to something the
		// deck's readers cannot see.
		// The fingerprint is taken from the same row being credited, by the
		// database, so what "the same words" means is defined once and cannot
		// drift between the write and the later check.
		if _, err := tx.Exec(ctx, `
			INSERT INTO presentation_sources(presentation_link_id,slide_position,note_id,note_fingerprint)
			SELECT $1,$2,$3,md5(coalesce(n.title,'') || E'\n' || coalesce(n.content,''))
			FROM notes n
			JOIN presentation_links l ON l.id=$1
			WHERE n.id=$3 AND n.space_id=l.space_id AND n.deleted_at IS NULL
			ON CONFLICT DO NOTHING`,
			linkID, src.SlidePosition, src.NoteID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListPresentationLinks returns the decks made from a space, newest first.
//
// Without the compiled source: a list is read far more often than any one
// deck's source, and shipping every source in it would send megabytes to draw a
// few rows.
func (s *Store) ListPresentationLinks(ctx context.Context, userID, spaceID uuid.UUID) ([]PresentationLink, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT l.id,l.space_id,l.ptium_presentation_id,l.title,l.status,l.error,l.failure_kind,
		       l.thought_count,l.excluded_count,l.trimmed_count,l.created_by
		FROM presentation_links l
		JOIN spaces sp ON sp.id=l.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		WHERE l.space_id=$1 AND (sp.owner_id=$2 OR m.permission IS NOT NULL)
		ORDER BY l.created_at DESC`, spaceID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []PresentationLink{}
	for rows.Next() {
		var link PresentationLink
		if err := rows.Scan(&link.ID, &link.SpaceID, &link.PtiumID, &link.Title, &link.Status,
			&link.Error, &link.FailureKind, &link.ThoughtCount, &link.ExcludedCount, &link.TrimmedCount, &link.CreatedBy); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

// PresentationSources returns which thoughts reached which slide.
func (s *Store) PresentationSources(ctx context.Context, userID, linkID uuid.UUID) ([]SlideSource, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT ps.slide_position, ps.note_id
		FROM presentation_sources ps
		JOIN presentation_links l ON l.id=ps.presentation_link_id
		JOIN spaces sp ON sp.id=l.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		WHERE ps.presentation_link_id=$1 AND (sp.owner_id=$2 OR m.permission IS NOT NULL)
		ORDER BY ps.slide_position, ps.note_id`, linkID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := []SlideSource{}
	for rows.Next() {
		var src SlideSource
		if err := rows.Scan(&src.SlidePosition, &src.NoteID); err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	return sources, rows.Err()
}

// PresentationsUsingNote answers the other direction: which talks a thought
// ended up in.
//
// Worth having on its own. Someone editing a note that six decks quote is
// making a different decision from someone editing one nobody has used, and
// only umm is in a position to tell them which they are doing.
func (s *Store) PresentationsUsingNote(ctx context.Context, userID, noteID uuid.UUID) ([]PresentationLink, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT DISTINCT l.id,l.space_id,l.ptium_presentation_id,l.title,l.status,l.error,l.failure_kind,
		       l.thought_count,l.excluded_count,l.trimmed_count,l.created_by,l.created_at
		FROM presentation_sources ps
		JOIN presentation_links l ON l.id=ps.presentation_link_id
		JOIN spaces sp ON sp.id=l.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		WHERE ps.note_id=$1 AND (sp.owner_id=$2 OR m.permission IS NOT NULL)
		ORDER BY l.created_at DESC`, noteID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []PresentationLink{}
	for rows.Next() {
		var link PresentationLink
		var createdAt any
		if err := rows.Scan(&link.ID, &link.SpaceID, &link.PtiumID, &link.Title, &link.Status,
			&link.Error, &link.FailureKind, &link.ThoughtCount, &link.ExcludedCount, &link.TrimmedCount, &link.CreatedBy, &createdAt); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

// PresentationLinkSource returns one deck's compiled source, for the reader who
// wants to know what umm actually sent.
func (s *Store) PresentationLinkSource(ctx context.Context, userID, linkID uuid.UUID) (string, error) {
	var source string
	err := s.Pool.QueryRow(ctx, `
		SELECT l.compiled_source
		FROM presentation_links l
		JOIN spaces sp ON sp.id=l.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		WHERE l.id=$1 AND (sp.owner_id=$2 OR m.permission IS NOT NULL)`, linkID, userID).Scan(&source)
	return source, err
}

// StaleSlide is a slide whose thought no longer says what it said.
type StaleSlide struct {
	SlidePosition int       `json:"slidePosition"`
	NoteID        uuid.UUID `json:"noteId"`
	// Reason is "changed" when the words were rewritten, "deleted" when the
	// thought is gone. They call for different things — one slide can be
	// updated, the other has lost its source entirely — so they are not
	// collapsed into a single "out of date".
	Reason string `json:"reason"`
	// Content is what the thought says now, so a person can see what changed
	// without leaving the deck. Empty for a deleted one.
	Content string `json:"content,omitempty"`
}

// Reasons a slide is out of date.
const (
	StaleChanged = "changed"
	StaleDeleted = "deleted"
)

// StaleSlides reports which of a deck's slides no longer match their thoughts.
//
// A slide whose fingerprint is empty is never reported: it was compiled before
// umm recorded one, and claiming a slide is stale when nobody knows is worse
// than saying nothing. Moving a note does not make a slide stale — only
// rewriting its words does, which is what the fingerprint is of.
func (s *Store) StaleSlides(ctx context.Context, userID, linkID uuid.UUID) ([]StaleSlide, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT ps.slide_position,
		       ps.note_id,
		       CASE WHEN n.id IS NULL OR n.deleted_at IS NOT NULL THEN 'deleted' ELSE 'changed' END,
		       coalesce(n.content,'')
		FROM presentation_sources ps
		JOIN presentation_links l ON l.id=ps.presentation_link_id
		JOIN spaces sp ON sp.id=l.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		LEFT JOIN notes n ON n.id=ps.note_id
		WHERE ps.presentation_link_id=$1
		  AND (sp.owner_id=$2 OR m.permission IS NOT NULL)
		  AND ps.note_fingerprint <> ''
		  AND (
			n.id IS NULL
			OR n.deleted_at IS NOT NULL
			OR md5(coalesce(n.title,'') || E'\n' || coalesce(n.content,'')) <> ps.note_fingerprint
		  )
		ORDER BY ps.slide_position, ps.note_id`, linkID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stale := []StaleSlide{}
	for rows.Next() {
		var row StaleSlide
		if err := rows.Scan(&row.SlidePosition, &row.NoteID, &row.Reason, &row.Content); err != nil {
			return nil, err
		}
		if row.Reason == StaleDeleted {
			row.Content = ""
		}
		stale = append(stale, row)
	}
	return stale, rows.Err()
}

// StaleCounts reports, for every deck made from a space, how many of its slides
// have gone out of date.
//
// One query rather than one per deck, because this is read to draw a list.
func (s *Store) StaleCounts(ctx context.Context, userID, spaceID uuid.UUID) (map[uuid.UUID]int, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT l.id, count(DISTINCT ps.slide_position)
		FROM presentation_links l
		JOIN spaces sp ON sp.id=l.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		JOIN presentation_sources ps ON ps.presentation_link_id=l.id
		LEFT JOIN notes n ON n.id=ps.note_id
		WHERE l.space_id=$1
		  AND (sp.owner_id=$2 OR m.permission IS NOT NULL)
		  AND ps.note_fingerprint <> ''
		  AND (
			n.id IS NULL
			OR n.deleted_at IS NOT NULL
			OR md5(coalesce(n.title,'') || E'\n' || coalesce(n.content,'')) <> ps.note_fingerprint
		  )
		GROUP BY l.id`, spaceID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[uuid.UUID]int{}
	for rows.Next() {
		var id uuid.UUID
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		counts[id] = count
	}
	return counts, rows.Err()
}

// FailedPresentationLink loads one failed link, for retrying into the deck that
// already exists rather than making another.
//
// Only a failed one: a deck that compiled is not something to re-apply source
// to behind its owner's back, and a pending one is a compile still running.
// Permission is the same as writing, because this leads to changing a deck.
//
// The request comes with it. This is the one read that needs it: retrying is
// asking for the same talk again, and without it a retry asks for a different
// one.
func (s *Store) FailedPresentationLink(ctx context.Context, userID, linkID uuid.UUID) (PresentationLink, error) {
	var link PresentationLink
	err := s.Pool.QueryRow(ctx, `
		SELECT l.id,l.space_id,l.ptium_presentation_id,l.title,l.status,l.error,l.failure_kind,
		       l.thought_count,l.excluded_count,l.trimmed_count,l.created_by,l.request
		FROM presentation_links l
		JOIN spaces sp ON sp.id=l.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
		WHERE l.id=$1 AND l.status=$3
		  AND (sp.owner_id=$2 OR m.permission IN ('edit','manage'))`, linkID, userID, PresentationFailed).
		Scan(&link.ID, &link.SpaceID, &link.PtiumID, &link.Title, &link.Status, &link.Error, &link.FailureKind,
			&link.ThoughtCount, &link.ExcludedCount, &link.TrimmedCount, &link.CreatedBy, &link.Request)
	if err != nil {
		return PresentationLink{}, err
	}
	return link, nil
}
