package store

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// umm can find two notes that say the same thing. Until now it could only point
// at them: there was no way to make them one, so the duplicate report was a list
// of problems with no button.
//
// Merging is the kind of operation that loses data quietly. Eight tables hang off
// a note — connections, comments, review schedules, revisions, embeddings, and
// three Dream tables — and anything not moved deliberately disappears when the
// losing note does. Each one is handled explicitly below rather than left to a
// cascade, and the result says what moved.
//
// Branch membership is deliberately not one of them. The surviving thought keeps
// whichever line it was already in, and does not inherit the other's. Merging
// says the caller chose which thought stands; moving a line onto it would file
// their surviving thought under a decision they did not make here — and if the
// line was one that was set aside, that decision reads as "this was rejected".
// Inventing that is worse than losing it, and the one place where the pairing
// matters most is already handled: the morning review refuses to offer a
// one-click merge when exactly one side sits in a line that was decided against.

// MergeResult reports what the merge did, so a caller can tell someone what
// happened to their thought rather than just that it worked.
type MergeResult struct {
	Note Note `json:"note"`
	// MovedEdges counts connections that now point at the surviving note.
	MovedEdges int `json:"movedEdges"`
	// DroppedEdges counts connections discarded because they would have become a
	// note connected to itself, or a duplicate of one the survivor already had.
	DroppedEdges  int `json:"droppedEdges"`
	MovedComments int `json:"movedComments"`
}

// ErrMergeSameNote is returned when both sides are the same thought.
var ErrMergeSameNote = errors.New("a note cannot be merged into itself")

// MergeNotes folds one thought into another and returns what moved.
//
// The caller supplies the surviving content. umm can tell that two notes are
// nearly the same; it cannot tell which words the person wants to keep, and
// concatenating them produces a note that says everything twice.
//
// Both notes must live in the same space. A connection is scoped to a space and
// needs both endpoints inside it, so merging across spaces would either strand
// the connections or silently move a thought somewhere its author did not put it.
func (s *Store) MergeNotes(ctx context.Context, userID, keepID, mergeID uuid.UUID, content string) (MergeResult, error) {
	if keepID == mergeID {
		return MergeResult{}, ErrMergeSameNote
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return MergeResult{}, errors.New("the surviving note needs content")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return MergeResult{}, err
	}
	defer tx.Rollback(ctx)

	// Lock both rows in a fixed order so two merges touching the same pair from
	// opposite directions cannot deadlock.
	first, second := keepID, mergeID
	if second.String() < first.String() {
		first, second = second, first
	}
	var spaces []uuid.UUID
	rows, err := tx.Query(ctx, `
		SELECT n.space_id FROM notes n
		JOIN spaces s ON s.id=n.space_id
		LEFT JOIN space_members m ON m.space_id=s.id AND m.user_id=$3
		WHERE n.id IN ($1,$2) AND n.deleted_at IS NULL
		  AND (s.owner_id=$3 OR m.permission IN ('edit','manage'))
		ORDER BY n.id
		FOR UPDATE OF n`, first, second, userID)
	if err != nil {
		return MergeResult{}, err
	}
	for rows.Next() {
		var spaceID uuid.UUID
		if err = rows.Scan(&spaceID); err != nil {
			rows.Close()
			return MergeResult{}, err
		}
		spaces = append(spaces, spaceID)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return MergeResult{}, err
	}
	if len(spaces) != 2 {
		return MergeResult{}, errors.New("both notes must exist and be writable")
	}
	if spaces[0] != spaces[1] {
		return MergeResult{}, errors.New("notes in different spaces cannot be merged")
	}
	spaceID := spaces[0]

	result := MergeResult{}

	// Connections. A merged pair that was connected to each other would become a
	// note connected to itself, and a connection the survivor already has would
	// become a duplicate — both are dropped rather than moved.
	if err = tx.QueryRow(ctx, `
		WITH gone AS (
			DELETE FROM note_edges e
			WHERE (e.source_note_id=$1 AND e.target_note_id=$2)
			   OR (e.source_note_id=$2 AND e.target_note_id=$1)
			   OR (e.source_note_id=$1 AND EXISTS(
			         SELECT 1 FROM note_edges k WHERE k.source_note_id=$2 AND k.target_note_id=e.target_note_id AND k.relation=e.relation))
			   OR (e.target_note_id=$1 AND EXISTS(
			         SELECT 1 FROM note_edges k WHERE k.target_note_id=$2 AND k.source_note_id=e.source_note_id AND k.relation=e.relation))
			RETURNING 1
		) SELECT count(*) FROM gone`, mergeID, keepID).Scan(&result.DroppedEdges); err != nil {
		return MergeResult{}, err
	}
	var movedOut, movedIn int
	if err = tx.QueryRow(ctx,
		`WITH m AS (UPDATE note_edges SET source_note_id=$2 WHERE source_note_id=$1 RETURNING 1) SELECT count(*) FROM m`,
		mergeID, keepID).Scan(&movedOut); err != nil {
		return MergeResult{}, err
	}
	if err = tx.QueryRow(ctx,
		`WITH m AS (UPDATE note_edges SET target_note_id=$2 WHERE target_note_id=$1 RETURNING 1) SELECT count(*) FROM m`,
		mergeID, keepID).Scan(&movedIn); err != nil {
		return MergeResult{}, err
	}
	result.MovedEdges = movedOut + movedIn

	// The discussion belongs to the thought, not to the row that held it.
	if err = tx.QueryRow(ctx,
		`WITH m AS (UPDATE note_comments SET note_id=$2 WHERE note_id=$1 RETURNING 1) SELECT count(*) FROM m`,
		mergeID, keepID).Scan(&result.MovedComments); err != nil {
		return MergeResult{}, err
	}

	// Dream provenance: a Dream that cited the losing note cited this thought, so
	// the citation follows. Rows that would collide with one the survivor already
	// has are dropped instead, because a Dream citing the same note twice says
	// nothing extra.
	if _, err = tx.Exec(ctx, `
		DELETE FROM dream_sources d
		WHERE d.source_note_id=$1
		  AND EXISTS(SELECT 1 FROM dream_sources k WHERE k.dream_id=d.dream_id AND k.source_note_id=$2)`,
		mergeID, keepID); err != nil {
		return MergeResult{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE dream_sources SET source_note_id=$2 WHERE source_note_id=$1`, mergeID, keepID); err != nil {
		return MergeResult{}, err
	}
	// dream_notes.note_id is unique, so the link only moves when the survivor has
	// none of its own; otherwise it goes with the losing note.
	if _, err = tx.Exec(ctx, `
		UPDATE dream_notes SET note_id=$2
		WHERE note_id=$1 AND NOT EXISTS(SELECT 1 FROM dream_notes k WHERE k.note_id=$2)`,
		mergeID, keepID); err != nil {
		return MergeResult{}, err
	}

	// A rejected suggestion stays rejected: the pair it named may now involve the
	// survivor, and re-proposing it would undo the person's answer.
	if _, err = tx.Exec(ctx, `DELETE FROM link_dismissals WHERE low_note_id=$1 OR high_note_id=$1`, mergeID); err != nil {
		return MergeResult{}, err
	}

	// Review schedules and revisions stay with their own note. A schedule is one
	// person's recall timing for a card they saw, and a revision is the history
	// of a row that is going away; carrying either across would describe the
	// surviving thought with events that did not happen to it.
	var note Note
	if err = tx.QueryRow(ctx, `
		UPDATE notes SET content=$2,version=version+1,updated_at=now()
		WHERE id=$1 AND deleted_at IS NULL
		RETURNING id,space_id,author_id,content,title,color,kind,source,ai_excluded,
		          x,y,width,height,rotation,version,created_at,updated_at`, keepID, content).
		Scan(&note.ID, &note.SpaceID, &note.AuthorID, &note.Content, &note.Title, &note.Color, &note.Kind,
			&note.Source, &note.AIExcluded, &note.X, &note.Y, &note.Width, &note.Height, &note.Rotation,
			&note.Version, &note.CreatedAt, &note.UpdatedAt); err != nil {
		return MergeResult{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE notes SET deleted_at=now(),updated_at=now() WHERE id=$1`, mergeID); err != nil {
		return MergeResult{}, err
	}
	// The content changed, so the stored vector describes text that no longer
	// exists. Dropping it makes the next read re-embed rather than compare
	// against the old wording.
	if _, err = tx.Exec(ctx, `DELETE FROM note_embeddings WHERE note_id=$1`, keepID); err != nil {
		return MergeResult{}, err
	}

	if err = s.AppendSpaceEvent(ctx, tx, userID, spaceID, "note.deleted", mergeID, map[string]any{"id": mergeID}); err != nil {
		return MergeResult{}, err
	}
	if err = s.AppendSpaceEvent(ctx, tx, userID, spaceID, "note.updated", note.ID, note); err != nil {
		return MergeResult{}, err
	}
	result.Note = note
	return result, tx.Commit(ctx)
}
