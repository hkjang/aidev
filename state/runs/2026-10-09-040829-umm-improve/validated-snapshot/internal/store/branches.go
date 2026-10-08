package store

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// Lines of thinking that were tried, and what became of them.
//
// The problem is not that abandoned ideas take up room. It is that they come
// back looking exactly like current ones. Search returns them, an answer cites
// them, the assistant reasons from them — and nothing in the record says this
// was already considered and set aside. The cost is re-deciding something you
// already decided, or worse, acting on the option you rejected.
//
// A branch is named by a person and resolved by a person. umm never decides that
// a line was abandoned because nothing was added to it: a month of silence is
// not a decision, and treating it as one would quietly bury paused work.

// Branch status vocabulary. Small on purpose — a status umm does nothing with
// would advertise a distinction it does not actually keep.
const (
	// BranchOpen: still being explored. The default, and the only status that
	// needs no reason.
	BranchOpen = "open"
	// BranchAdopted: this is the line that was taken.
	BranchAdopted = "adopted"
	// BranchAbandoned: considered and set aside. Thoughts in it stay readable,
	// but everything that surfaces them says so.
	BranchAbandoned = "abandoned"
)

// ErrUnknownBranchStatus is returned for a status outside the vocabulary, so the
// API answers 400 rather than 500.
var ErrUnknownBranchStatus = errors.New("unknown branch status")

// ErrResolutionRequired is returned when a branch is resolved without saying
// why. The decision without the reason is the half people actually lose.
var ErrResolutionRequired = errors.New("branch resolution requires a reason")

// Branch is one line of thinking within a space.
type Branch struct {
	ID         uuid.UUID  `json:"id"`
	SpaceID    uuid.UUID  `json:"spaceId"`
	RootNoteID *uuid.UUID `json:"rootNoteId,omitempty"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	Resolution string     `json:"resolution"`
	ResolvedAt *string    `json:"resolvedAt,omitempty"`
	// Notes counts thoughts currently filed under this line.
	Notes int `json:"notes"`
}

// BranchRef is the little that other views need: enough to label a thought
// without carrying the whole branch around.
type BranchRef struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Status string    `json:"status"`
	// Resolution travels with the label because the label alone is not enough.
	// "This line was set aside" invites the question it does not answer, and a
	// model handed only the status will invent one or hedge into saying the
	// decision is still open.
	Resolution string `json:"resolution,omitempty"`
}

// ParseBranchStatus accepts what a client sent for a resolution.
//
// Only the two resolved statuses are accepted here: reopening a resolved branch
// is a separate act with its own meaning, and folding it into "set the status"
// would let a reason be erased by accident.
func ParseBranchStatus(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed != BranchAdopted && trimmed != BranchAbandoned {
		return "", ErrUnknownBranchStatus
	}
	return trimmed, nil
}

// CreateBranch starts a named line of thinking in a space.
//
// The root thought is optional. A line often starts from something already
// written, but not always, and requiring one would push people to invent a
// parent note that says nothing.
func (s *Store) CreateBranch(ctx context.Context, userID, spaceID uuid.UUID, name string, rootNoteID *uuid.UUID) (Branch, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Branch{}, errors.New("branch name required")
	}
	if len([]rune(name)) > 80 {
		return Branch{}, errors.New("branch name is too long")
	}
	var branch Branch
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO branches(space_id,root_note_id,name,created_by)
		SELECT $1,$4,$3,$2
		WHERE EXISTS(
			SELECT 1 FROM spaces sp
			LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
			WHERE sp.id=$1 AND (sp.owner_id=$2 OR m.permission IN ('edit','manage')))
		  -- A root from another space would file the line under a thought its
		  -- readers cannot see.
		  AND ($4::uuid IS NULL OR EXISTS(
			SELECT 1 FROM notes n WHERE n.id=$4 AND n.space_id=$1 AND n.deleted_at IS NULL))
		RETURNING id,space_id,root_note_id,name,status,resolution`,
		spaceID, userID, name, rootNoteID).
		Scan(&branch.ID, &branch.SpaceID, &branch.RootNoteID, &branch.Name, &branch.Status, &branch.Resolution)
	if err != nil {
		return Branch{}, err
	}
	return branch, nil
}

// ListBranches returns the lines in a space, newest first.
func (s *Store) ListBranches(ctx context.Context, userID, spaceID uuid.UUID) ([]Branch, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT b.id,b.space_id,b.root_note_id,b.name,b.status,b.resolution,
		       to_char(b.resolved_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       (SELECT count(*) FROM notes n WHERE n.branch_id=b.id AND n.deleted_at IS NULL)
		FROM branches b
		JOIN spaces sp ON sp.id=b.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$1
		WHERE b.space_id=$2 AND (sp.owner_id=$1 OR m.user_id=$1)
		ORDER BY b.created_at DESC
		LIMIT 200`, userID, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Branch{}
	for rows.Next() {
		var branch Branch
		if err := rows.Scan(&branch.ID, &branch.SpaceID, &branch.RootNoteID, &branch.Name,
			&branch.Status, &branch.Resolution, &branch.ResolvedAt, &branch.Notes); err != nil {
			return nil, err
		}
		out = append(out, branch)
	}
	return out, rows.Err()
}

// BranchAssignments maps each thought in a space to the line it belongs to.
//
// Returned beside the branch list rather than as a column on Note: every path
// that reads a note would otherwise have to remember to select it, and umm has
// already shipped that bug once — Backlinks returned edges missing two fields
// for a whole release because a single read path was forgotten.
func (s *Store) BranchAssignments(ctx context.Context, userID, spaceID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT n.id,n.branch_id
		FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$1
		WHERE n.space_id=$2 AND n.branch_id IS NOT NULL AND n.deleted_at IS NULL
		  AND (sp.owner_id=$1 OR m.user_id=$1)`, userID, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]uuid.UUID{}
	for rows.Next() {
		var noteID, branchID uuid.UUID
		if err := rows.Scan(&noteID, &branchID); err != nil {
			return nil, err
		}
		out[noteID] = branchID
	}
	return out, rows.Err()
}

// SetNoteBranch files a thought under a line, or clears it when branchID is nil.
//
// The branch must be in the same space as the thought. Filing a thought under a
// line in a space its readers cannot see would make the label unexplainable.
func (s *Store) SetNoteBranch(ctx context.Context, userID, noteID uuid.UUID, branchID *uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE notes SET branch_id=$3,updated_at=now()
		WHERE id=$1 AND deleted_at IS NULL
		  AND EXISTS(
			SELECT 1 FROM spaces sp
			LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
			WHERE sp.id=notes.space_id AND (sp.owner_id=$2 OR m.permission IN ('edit','manage')))
		  AND ($3::uuid IS NULL OR EXISTS(
			SELECT 1 FROM branches b WHERE b.id=$3 AND b.space_id=notes.space_id))`,
		noteID, userID, branchID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("note or branch not available")
	}
	return nil
}

// ResolveBranch records what became of a line, and why.
//
// The reason is required. A branch marked abandoned with no reason is the same
// forgetting this feature exists to prevent, one step later: you know you
// rejected it and no longer know what for.
func (s *Store) ResolveBranch(ctx context.Context, userID, branchID uuid.UUID, status, resolution string) (Branch, error) {
	status, err := ParseBranchStatus(status)
	if err != nil {
		return Branch{}, err
	}
	resolution = strings.TrimSpace(resolution)
	if resolution == "" {
		return Branch{}, ErrResolutionRequired
	}
	if len([]rune(resolution)) > 1000 {
		return Branch{}, errors.New("resolution is too long")
	}
	var branch Branch
	err = s.Pool.QueryRow(ctx, `
		UPDATE branches b SET status=$3,resolution=$4,resolved_at=now()
		WHERE b.id=$1
		  AND EXISTS(
			SELECT 1 FROM spaces sp
			LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
			WHERE sp.id=b.space_id AND (sp.owner_id=$2 OR m.permission IN ('edit','manage')))
		RETURNING b.id,b.space_id,b.root_note_id,b.name,b.status,b.resolution,
		          to_char(b.resolved_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		branchID, userID, status, resolution).
		Scan(&branch.ID, &branch.SpaceID, &branch.RootNoteID, &branch.Name,
			&branch.Status, &branch.Resolution, &branch.ResolvedAt)
	if err != nil {
		return Branch{}, err
	}
	return branch, nil
}

// ReopenBranch puts a resolved line back in play.
//
// Kept separate from resolving because it is a different thing to say. The
// reason is cleared, since it described a decision that no longer stands, and
// leaving stale text under an open branch would misread as current.
func (s *Store) ReopenBranch(ctx context.Context, userID, branchID uuid.UUID) (Branch, error) {
	var branch Branch
	err := s.Pool.QueryRow(ctx, `
		UPDATE branches b SET status='open',resolution='',resolved_at=NULL
		WHERE b.id=$1
		  AND EXISTS(
			SELECT 1 FROM spaces sp
			LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
			WHERE sp.id=b.space_id AND (sp.owner_id=$2 OR m.permission IN ('edit','manage')))
		RETURNING b.id,b.space_id,b.root_note_id,b.name,b.status,b.resolution`,
		branchID, userID).
		Scan(&branch.ID, &branch.SpaceID, &branch.RootNoteID, &branch.Name, &branch.Status, &branch.Resolution)
	if err != nil {
		return Branch{}, err
	}
	return branch, nil
}

// DeleteBranch stops tracking a line. The thoughts in it stay; they simply are
// no longer filed under anything.
func (s *Store) DeleteBranch(ctx context.Context, userID, branchID uuid.UUID) error {
	tag, err := s.Pool.Exec(ctx, `
		DELETE FROM branches b
		WHERE b.id=$1
		  AND EXISTS(
			SELECT 1 FROM spaces sp
			LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$2
			WHERE sp.id=b.space_id AND (sp.owner_id=$2 OR m.permission IN ('edit','manage')))`,
		branchID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("branch not available")
	}
	return nil
}

// maxSetAsideScanNotes bounds how many thoughts from lines that were decided
// against are carried into the duplicate pass. Each is compared against the
// recent window, so the cost is this times that — bounded, and far smaller than
// comparing the whole space with itself.
const maxSetAsideScanNotes = 500

// NotesInSetAsideLines returns thoughts belonging to lines that were decided
// against, newest first.
//
// The duplicate pass compares the most recent thoughts in a space with each
// other, which is where a duplicate almost always is. It is not where a repeated
// decision is: that pairs something written today with something rejected long
// ago, and the old half falls outside any recency window. Without these the
// guard quietly stops working in exactly the long-lived workspaces it exists for.
func (s *Store) NotesInSetAsideLines(ctx context.Context, userID, spaceID uuid.UUID) ([]Note, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT n.id,n.space_id,n.author_id,n.content,n.title,n.color,n.kind,n.source,n.ai_excluded,
		       n.x,n.y,n.width,n.height,n.rotation,n.version,n.created_at,n.updated_at
		FROM notes n
		JOIN branches b ON b.id=n.branch_id AND b.status=$3
		JOIN spaces sp ON sp.id=n.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$1
		WHERE n.space_id=$2 AND n.deleted_at IS NULL AND (sp.owner_id=$1 OR m.user_id=$1)
		ORDER BY b.resolved_at DESC NULLS LAST, n.created_at DESC
		LIMIT $4`, userID, spaceID, BranchAbandoned, maxSetAsideScanNotes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.SpaceID, &note.AuthorID, &note.Content, &note.Title,
			&note.Color, &note.Kind, &note.Source, &note.AIExcluded, &note.X, &note.Y,
			&note.Width, &note.Height, &note.Rotation, &note.Version, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, note)
	}
	return out, rows.Err()
}

// BranchRefsForNotes labels a set of thoughts with the line each belongs to.
//
// Exported for the paths outside this package that surface a thought — search is
// one — because everywhere a thought appears has to be able to say that its line
// was decided against. umm has now found three such doors after the fact
// (retrieval, the export, MCP); the fourth should not need a fourth lookup of
// its own.
func (s *Store) BranchRefsForNotes(ctx context.Context, noteIDs []uuid.UUID) (map[uuid.UUID]BranchRef, error) {
	return s.branchRefsForNotes(ctx, noteIDs)
}

// branchRefsForNotes labels a set of thoughts with the line each belongs to.
//
// Separate from reading the notes themselves rather than a column on Note: every
// path that reads a note would otherwise have to remember to select it, and umm
// has already shipped that bug once — Backlinks returned edges missing two
// fields for a whole release because one read path was forgotten.
func (s *Store) branchRefsForNotes(ctx context.Context, noteIDs []uuid.UUID) (map[uuid.UUID]BranchRef, error) {
	refs := map[uuid.UUID]BranchRef{}
	if len(noteIDs) == 0 {
		return refs, nil
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT n.id,b.id,b.name,b.status,b.resolution
		FROM notes n JOIN branches b ON b.id=n.branch_id
		WHERE n.id = ANY($1)`, noteIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var noteID uuid.UUID
		var ref BranchRef
		if err := rows.Scan(&noteID, &ref.ID, &ref.Name, &ref.Status, &ref.Resolution); err != nil {
			return nil, err
		}
		refs[noteID] = ref
	}
	return refs, rows.Err()
}
