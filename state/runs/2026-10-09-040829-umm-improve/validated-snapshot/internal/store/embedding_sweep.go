package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

/*
Keeping vectors up to date, away from anybody who is waiting.

Until v0.72.0 the only thing that ever wrote an embedding was reading a space.
Writing a note did not, and nothing ran in the background, so the vectors for
everything anyone had changed were built the next time somebody opened the
canvas — on that person's request, in front of them. With an embedding gateway
configured that put somebody else's server on the path to your own notes.

So it moved here. The read nudges this sweep and returns; the sweep does the
work on its own clock. A thought with no vector is still invisible to search
and to Dream, which is why the original code sat where it did, so the interval
is short and a nudge jumps the queue.

Note-level and space-level AI exclusion are honoured exactly as before, and by
the same code: excluded thoughts are collected separately and handed to
ensureEmbeddings in their own batch, which resolves them to the local algorithm
and never sends their words anywhere.
*/

// embeddingSweepInterval is how often the sweep looks for work with nothing
// nudging it. Short, because a thought nobody has indexed is a thought nobody
// can find.
const embeddingSweepInterval = 5 * time.Second

// embeddingSweepBatch bounds one pass. A backlog is worked through over several
// passes rather than in one long transaction that holds a gateway open.
const embeddingSweepBatch = 200

// nudgeBuffer is how many spaces can be waiting to be looked at first. Small:
// this is a hint, and dropping one costs a few seconds rather than correctness,
// because the untargeted pass finds the same notes anyway.
const nudgeBuffer = 64

// NudgeEmbeddings asks the sweep to look at one space next. Never blocks: a
// full buffer means the sweep is already busy, and the ordinary pass will reach
// these notes regardless.
func (s *Store) NudgeEmbeddings(spaceID uuid.UUID) {
	if s.embeddingNudges == nil || spaceID == uuid.Nil {
		return
	}
	select {
	case s.embeddingNudges <- spaceID:
	default:
	}
}

// StartEmbeddingSweep runs the indexer until the context is cancelled.
//
// Started by the server rather than lazily by the first read, so an
// installation nobody has opened yet still indexes what was imported into it.
func (s *Store) StartEmbeddingSweep(ctx context.Context) {
	if s.embeddingNudges == nil {
		s.embeddingNudges = make(chan uuid.UUID, nudgeBuffer)
	}
	go func() {
		ticker := time.NewTicker(embeddingSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case spaceID := <-s.embeddingNudges:
				s.sweepEmbeddings(ctx, &spaceID)
			case <-ticker.C:
				s.sweepEmbeddings(ctx, nil)
			}
		}
	}()
}

// SweepEmbeddingsOnce does one bounded pass and reports how many thoughts it
// brought up to date. Exported for tests and for a caller that wants the work
// done now rather than on the next tick.
func (s *Store) SweepEmbeddingsOnce(ctx context.Context, spaceID *uuid.UUID) int {
	return s.sweepEmbeddings(ctx, spaceID)
}

func (s *Store) sweepEmbeddings(ctx context.Context, spaceID *uuid.UUID) int {
	// The algorithm a thought should be stored under depends on whether it, or
	// its space, is held back from analysis: those stay on the local algorithm
	// whatever is configured. Deciding that in the query means one statement
	// finds everything that is behind, rather than the whole table being read
	// into Go to find out.
	configured := s.EmbeddingProvider(ctx).Algorithm()
	rows, err := s.Pool.Query(ctx, `
		SELECT n.id,n.space_id,n.content,n.version,(n.ai_excluded OR sp.ai_excluded)
		FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		WHERE n.deleted_at IS NULL
		  AND ($1::uuid IS NULL OR n.space_id=$1)
		  AND NOT EXISTS(
		    SELECT 1 FROM note_embeddings e
		    WHERE e.note_id=n.id AND e.content_version>=n.version
		      AND e.algorithm = CASE WHEN (n.ai_excluded OR sp.ai_excluded) THEN $2 ELSE $3 END)
		ORDER BY n.updated_at DESC
		LIMIT $4`, spaceID, intelligence.LocalAlgorithm, configured, embeddingSweepBatch)
	if err != nil {
		slog.Warn("embedding sweep query failed", "error", err)
		return 0
	}
	defer rows.Close()

	var shared, held []Note
	for rows.Next() {
		var note Note
		var excluded bool
		if err := rows.Scan(&note.ID, &note.SpaceID, &note.Content, &note.Version, &excluded); err != nil {
			slog.Warn("embedding sweep scan failed", "error", err)
			return 0
		}
		if excluded {
			// Marked so that ensureEmbeddings resolves this batch to the local
			// algorithm, which is the same decision it made when this ran on
			// the read path.
			note.AIExcluded = true
			held = append(held, note)
			continue
		}
		shared = append(shared, note)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("embedding sweep read failed", "error", err)
		return 0
	}
	rows.Close()

	// Two batches, never one. ensureEmbeddings drops the whole batch to the
	// local algorithm if any thought in it is held back, so mixing them would
	// quietly stop the configured model being used for everything else.
	done := 0
	for _, batch := range [][]Note{shared, held} {
		if len(batch) == 0 {
			continue
		}
		s.ensureEmbeddings(ctx, batch)
		done += len(batch)
	}
	return done
}
