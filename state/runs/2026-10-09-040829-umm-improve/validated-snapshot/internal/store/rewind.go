package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

/*
The space as it was.

Every note update snapshots the state it is replacing — content, title, colour,
kind, and where the note sat — so the material for looking backwards has been
accumulating since v0.2.0 without anything reading it that way. NoteHistory
shows one note's versions; this shows every note at once, which is the question
people actually ask: what did this look like in June.

The reconstruction rule follows from how revisions are written. A revision row
holds the state *before* the change that created it, and its timestamp is when
that change happened. So a note's state at an instant is the earliest revision
recorded after that instant, and if nothing was recorded after it, the note has
not changed since and its current state is its state then.

Pictures follow the same rule as words. An attachment is dated, so a canvas
showing June shows the photographs that were on those thoughts in June and not
the screenshot pasted in August — otherwise the one part of a thought whose age
nobody can tell by reading it is the part that quietly comes from today.

What this cannot do is replay connections that were removed. A deletion event
records that an edge went, never what it joined — it has only ever carried the
edge's id — and the row itself is gone. So removed connections are counted and
said rather than quietly missing from the picture, which would be a canvas
claiming those connections never existed. The pictures that can no longer be
drawn are counted for the same reason.
*/

// ErrSpaceNotVisible is returned when the caller may not read the space. A
// distinct error rather than an empty snapshot: "you cannot see this" and
// "this was empty then" are different answers and the screen says different
// things about them.
var ErrSpaceNotVisible = errors.New("space not visible")

// SpaceSnapshot is one space at one instant.
type SpaceSnapshot struct {
	At    time.Time `json:"at"`
	Notes []Note    `json:"notes"`
	Edges []Edge    `json:"edges"`
	// RemovedEdges is how many connections were deleted between that instant
	// and now. They existed then and cannot be drawn, so the number is shown
	// instead of a picture that silently omits them.
	RemovedEdges int `json:"removedEdges"`
	// Attachments are the pictures that were on those thoughts at that instant.
	// Carried with the snapshot rather than left to the canvas's own list,
	// which is today's: a photograph pasted last week would otherwise appear on
	// a thought as it stood in June, and a picture is the one part of a thought
	// whose age nobody can tell by reading it.
	Attachments []Attachment `json:"attachments"`
	// RemovedAttachments is how many pictures were on the canvas then and can
	// no longer be drawn, because the thought they hung on has been deleted
	// since. That thought does come back — it was there — so without this
	// number it would come back looking as though it never had a picture.
	//
	// A picture somebody deleted outright leaves no row and cannot be counted
	// here at all, the same limit removed connections have.
	RemovedAttachments int `json:"removedAttachments"`
	// Earliest is the first moment this space has anything to show, so a
	// control cannot offer a date before the space existed.
	Earliest time.Time `json:"earliest"`
}

// SpaceAt reconstructs a space at an instant, for someone who may read it.
//
// Access is checked in the same statement that reads the rows, so a snapshot
// cannot be assembled from a moment when the caller could still see the space
// and returned after they no longer can.
func (s *Store) SpaceAt(ctx context.Context, userID, spaceID uuid.UUID, at time.Time) (SpaceSnapshot, error) {
	snapshot := SpaceSnapshot{At: at, Notes: []Note{}, Edges: []Edge{}, Attachments: []Attachment{}}

	var allowed bool
	if err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1 FROM spaces sp
		  LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=$1
		  WHERE sp.id=$2 AND (sp.owner_id=$1 OR sm.user_id=$1))`, userID, spaceID).Scan(&allowed); err != nil {
		return SpaceSnapshot{}, err
	}
	if !allowed {
		return SpaceSnapshot{}, ErrSpaceNotVisible
	}

	if err := s.Pool.QueryRow(ctx,
		`SELECT COALESCE(min(created_at), now()) FROM notes WHERE space_id=$1`, spaceID).
		Scan(&snapshot.Earliest); err != nil {
		return SpaceSnapshot{}, err
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT n.id,n.space_id,n.author_id,
		       COALESCE(r.content,n.content),COALESCE(r.title,n.title),COALESCE(r.color,n.color),
		       COALESCE(r.kind,n.kind),n.source,n.ai_excluded,
		       COALESCE(r.x,n.x),COALESCE(r.y,n.y),COALESCE(r.width,n.width),
		       COALESCE(r.height,n.height),COALESCE(r.rotation,n.rotation),
		       n.version,n.created_at,n.updated_at
		FROM notes n
		LEFT JOIN LATERAL (
		  SELECT rv.content,rv.title,rv.color,rv.kind,rv.x,rv.y,rv.width,rv.height,rv.rotation
		  FROM note_revisions rv
		  WHERE rv.note_id=n.id AND rv.created_at>$2
		  ORDER BY rv.created_at ASC
		  LIMIT 1
		) r ON true
		WHERE n.space_id=$1
		  AND n.created_at<=$2
		  AND (n.deleted_at IS NULL OR n.deleted_at>$2)
		ORDER BY n.created_at`, spaceID, at)
	if err != nil {
		return SpaceSnapshot{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.SpaceID, &n.AuthorID, &n.Content, &n.Title, &n.Color, &n.Kind,
			&n.Source, &n.AIExcluded, &n.X, &n.Y, &n.Width, &n.Height, &n.Rotation,
			&n.Version, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return SpaceSnapshot{}, err
		}
		snapshot.Notes = append(snapshot.Notes, n)
	}
	if err := rows.Err(); err != nil {
		return SpaceSnapshot{}, err
	}

	// Only connections that still exist, and only from when they were drawn.
	// Both ends have to have been on the canvas then, or the line would hang
	// off a thought that was not there yet.
	edges, err := s.Pool.Query(ctx, `
		SELECT e.id,e.space_id,e.source_note_id,e.target_note_id,e.relation,e.origin,e.confidence,e.reason
		FROM note_edges e
		JOIN notes src ON src.id=e.source_note_id
		JOIN notes dst ON dst.id=e.target_note_id
		WHERE e.space_id=$1 AND e.created_at<=$2
		  AND src.created_at<=$2 AND dst.created_at<=$2
		  AND (src.deleted_at IS NULL OR src.deleted_at>$2)
		  AND (dst.deleted_at IS NULL OR dst.deleted_at>$2)`, spaceID, at)
	if err != nil {
		return SpaceSnapshot{}, err
	}
	defer edges.Close()
	for edges.Next() {
		var e Edge
		if err := edges.Scan(&e.ID, &e.SpaceID, &e.SourceID, &e.TargetID, &e.Relation,
			&e.Origin, &e.Confidence, &e.Reason); err != nil {
			return SpaceSnapshot{}, err
		}
		snapshot.Edges = append(snapshot.Edges, e)
	}
	if err := edges.Err(); err != nil {
		return SpaceSnapshot{}, err
	}

	// Only the pictures that had been put on by then, and only the ones the
	// canvas can still fetch. Bytes are served against a thought that is not
	// deleted, so a picture on a thought deleted since would come back as a
	// broken image rather than a photograph; those are counted below instead.
	pictures, err := s.Pool.Query(ctx, `
		SELECT a.id,a.note_id,a.content_type,a.byte_size,a.filename,a.uploaded_by,a.created_at
		FROM note_attachments a
		JOIN notes n ON n.id=a.note_id
		WHERE a.space_id=$1 AND a.created_at<=$2 AND n.deleted_at IS NULL
		ORDER BY a.created_at`, spaceID, at)
	if err != nil {
		return SpaceSnapshot{}, err
	}
	defer pictures.Close()
	for pictures.Next() {
		var a Attachment
		if err := pictures.Scan(&a.ID, &a.NoteID, &a.ContentType, &a.ByteSize, &a.Filename,
			&a.UploadedBy, &a.CreatedAt); err != nil {
			return SpaceSnapshot{}, err
		}
		snapshot.Attachments = append(snapshot.Attachments, a)
	}
	if err := pictures.Err(); err != nil {
		return SpaceSnapshot{}, err
	}

	if err := s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM note_attachments a
		JOIN notes n ON n.id=a.note_id
		WHERE a.space_id=$1 AND a.created_at<=$2
		  AND n.deleted_at IS NOT NULL AND n.deleted_at>$2`, spaceID, at).
		Scan(&snapshot.RemovedAttachments); err != nil {
		return SpaceSnapshot{}, err
	}

	// The log knows a connection went even though it cannot say what it joined.
	// Counting is the honest half of that.
	if err := s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM space_events
		WHERE space_id=$1 AND event_type='edge.deleted' AND created_at>$2`, spaceID, at).
		Scan(&snapshot.RemovedEdges); err != nil {
		return SpaceSnapshot{}, err
	}
	return snapshot, nil
}
