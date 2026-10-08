package store

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

// Auto-link is umm proposing connections nobody drew.
//
// It is the first thing in the product that writes into someone's memory on its
// own, so two constraints shape it. It only runs on a backend that has been
// measured to tell meaning from vocabulary, and everything it writes is marked
// as inferred, carries a score, and can be accepted or dismissed. A suggestion
// that cannot be told from a drawn line is not a suggestion.

// SuggestionOutcome explains what a run did, including when it did nothing. A
// caller that gets an empty list needs to know whether the workspace has no
// candidates or whether umm refused to guess.
type SuggestionOutcome string

const (
	// OutcomeSuggested: umm proposed at least one connection.
	OutcomeSuggested SuggestionOutcome = "suggested"
	// OutcomeNoCandidates: the backend is fit to judge and found nothing that
	// stands out from the rest of the workspace.
	OutcomeNoCandidates SuggestionOutcome = "no-candidates"
	// OutcomeBackendNotSemantic: the active embedding scores shared vocabulary
	// above shared meaning, so anything it proposed would be connections between
	// notes that happen to use the same words.
	OutcomeBackendNotSemantic SuggestionOutcome = "backend-not-semantic"
	// OutcomeTooFewNotes: not enough notes to say what "unusually similar" means
	// in this workspace.
	OutcomeTooFewNotes SuggestionOutcome = "too-few-notes"
	// OutcomeDisabled: an administrator turned proposals off, for a deployment
	// that wants the graph to hold only what people put in it.
	OutcomeDisabled SuggestionOutcome = "disabled"
	// OutcomeReadOnly is a space this person may read but not change.
	//
	// Suggesting connections writes them, so it is a change to someone's
	// workspace and not a read. Reading the notes is allowed for a member shared
	// in to view, and that was as far as the check went — the edges were then
	// inserted without asking whether they could be.
	OutcomeReadOnly SuggestionOutcome = "read-only"
)

// SuggestionResult is what a run produced and why.
type SuggestionResult struct {
	Outcome SuggestionOutcome `json:"outcome"`
	// Always a list, never null, including on the paths that decline to run: a
	// caller iterating the result should not have to special-case the refusals.
	Edges []Edge `json:"edges"`
	// Considered is the number of pairs scored, so a person can see that a quiet
	// result came from looking rather than from not looking.
	Considered int `json:"considered"`
}

// SuggestLinks proposes connections between notes that sit unusually close
// together in the active embedding space, and records them as inferred edges.
//
// It refuses to run on a lexical backend. umm measures its own embedding on
// labelled data, and the offline default ranks two sentences that share words
// above two that share meaning — proposing from it would connect "PostgreSQL
// backup schedule" to "I like the PostgreSQL logo" and call it a discovery.
func (s *Store) SuggestLinks(ctx context.Context, userID, spaceID uuid.UUID) (SuggestionResult, error) {
	settings := s.IntelligenceSettings(ctx)
	if !settings.AutoLinkEnabled {
		return SuggestionResult{Outcome: OutcomeDisabled, Edges: []Edge{}}, nil
	}
	// Asked before the backend is measured. Whether this person may change the
	// space at all decides more than whether the embedding could judge it, and
	// answering it first means no embedding work is done for someone whose
	// result could never be written.
	writable, err := s.canWriteToSpace(ctx, userID, spaceID)
	if err != nil {
		return SuggestionResult{}, err
	}
	// Indexing is a background job now, so a thought written moments ago may
	// have no vector yet. Proposing connections while ignoring the newest
	// thought would answer a question about the space as it was; this asks for
	// that space to be caught up first. Bounded, and only on a run somebody
	// deliberately started.
	s.SweepEmbeddingsOnce(ctx, &spaceID)
	if !writable {
		return SuggestionResult{Outcome: OutcomeReadOnly, Edges: []Edge{}}, nil
	}

	report, err := s.MeasureEmbeddingQuality(ctx, false)
	if err != nil {
		return SuggestionResult{}, fmt.Errorf("measure embedding backend: %w", err)
	}
	if !report.Semantic {
		return SuggestionResult{Outcome: OutcomeBackendNotSemantic, Edges: []Edge{}}, nil
	}

	notes, _, err := s.ListNotes(ctx, userID, spaceID, "")
	if err != nil {
		return SuggestionResult{}, err
	}
	if len(notes) < settings.AutoLinkMinNote {
		return SuggestionResult{Outcome: OutcomeTooFewNotes, Edges: []Edge{}}, nil
	}
	vectors := s.loadEmbeddings(ctx, notes)

	// Pairs to leave alone: already connected, or already turned down. Without
	// the second, deleting a suggestion removes the only record that kept umm
	// from proposing it again and the next run brings it straight back.
	existing, err := s.existingPairs(ctx, spaceID)
	if err != nil {
		return SuggestionResult{}, err
	}
	dismissed, err := s.dismissedPairs(ctx, spaceID)
	if err != nil {
		return SuggestionResult{}, err
	}

	type candidate struct {
		source, target uuid.UUID
		score          float64
	}
	candidates := make([]candidate, 0, len(notes))
	observed := make([]float64, 0, len(notes)*(len(notes)-1)/2)
	for i := 0; i < len(notes); i++ {
		for j := i + 1; j < len(notes); j++ {
			score := intelligence.Cosine(vectors[notes[i].ID], vectors[notes[j].ID])
			// Every pair feeds the distribution, including ones already connected:
			// the bar describes the workspace, not the unconnected part of it.
			observed = append(observed, score)
			key := pairKey(notes[i].ID, notes[j].ID)
			if existing[key] || dismissed[key] {
				continue
			}
			candidates = append(candidates, candidate{notes[i].ID, notes[j].ID, score})
		}
	}
	result := SuggestionResult{Considered: len(observed), Edges: []Edge{}}

	scale := intelligence.NewSimilarityScale(observed)
	if !scale.Usable() {
		// Every pair scoring alike carries no signal to threshold on, and an
		// absolute fallback is exactly what breaks across backends.
		result.Outcome = OutcomeNoCandidates
		return result, nil
	}
	cutoff := scale.Threshold(intelligence.Band(settings.AutoLinkBand))

	kept := candidates[:0]
	for _, item := range candidates {
		// A near-duplicate is not a connection. Two copies of the same thought sit
		// at the very top of any similarity ranking, so they were always the first
		// thing suggested — and "related" understates them: the honest reading is
		// that one thought was written twice, which the morning review already
		// handles. Proposing a link as well makes one situation cost two chores,
		// and accepting it records a weaker claim than the truth.
		//
		// The duplicate bar is the one absolute threshold umm keeps, and this path
		// is already gated on a semantic backend — the same condition that makes
		// that bar meaningful.
		if item.score >= settings.DuplicateSimilarity {
			continue
		}
		if item.score >= cutoff {
			kept = append(kept, item)
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].score > kept[j].score })
	if len(kept) > settings.AutoLinkMaxRun {
		kept = kept[:settings.AutoLinkMaxRun]
	}
	if len(kept) == 0 {
		result.Outcome = OutcomeNoCandidates
		return result, nil
	}

	for _, item := range kept {
		confidence := suggestionConfidence(item.score, scale, settings.AutoLinkBand)
		edge, err := s.createInferredEdge(ctx, userID, spaceID, item.source, item.target, confidence)
		if err != nil {
			return SuggestionResult{}, err
		}
		result.Edges = append(result.Edges, edge)
	}
	result.Outcome = OutcomeSuggested
	return result, nil
}

// suggestionConfidence maps a score onto 0..1 by how far above the workspace's
// own average it sits.
//
// It is not a probability that the connection is meaningful — nothing here can
// estimate that. It says how strongly this pair stands out from everything else
// in the space, which is the only claim the measurement supports, and it is what
// the interface should show a person deciding whether to keep the suggestion.
func suggestionConfidence(score float64, scale intelligence.SimilarityScale, band float64) float64 {
	if scale.StdDev <= 0 {
		return 0.5
	}
	deviations := (score - scale.Mean) / scale.StdDev
	// The bar already sits at suggestionBand deviations, so a pair that barely
	// clears it starts at 0.5 and rises from there; three deviations above the
	// bar reaches the ceiling.
	confidence := 0.5 + (deviations-band)/6
	return math.Min(0.99, math.Max(0.5, confidence))
}

func pairKey(a, b uuid.UUID) string {
	if a.String() < b.String() {
		return a.String() + "\x00" + b.String()
	}
	return b.String() + "\x00" + a.String()
}

// existingPairs collects every pair that is already connected in either
// direction, whoever made the connection. umm must not propose a link a person
// already drew, nor propose the same one twice across runs.
func (s *Store) existingPairs(ctx context.Context, spaceID uuid.UUID) (map[string]bool, error) {
	rows, err := s.Pool.Query(ctx, `SELECT source_note_id,target_note_id FROM note_edges WHERE space_id=$1`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pairs := map[string]bool{}
	for rows.Next() {
		var source, target uuid.UUID
		if err := rows.Scan(&source, &target); err != nil {
			return nil, err
		}
		pairs[pairKey(source, target)] = true
	}
	return pairs, rows.Err()
}

// dismissedPairs collects the connections someone has already turned down. They
// stay turned down: re-proposing a suggestion a person rejected teaches them to
// ignore the feature.
func (s *Store) dismissedPairs(ctx context.Context, spaceID uuid.UUID) (map[string]bool, error) {
	rows, err := s.Pool.Query(ctx, `SELECT low_note_id,high_note_id FROM link_dismissals WHERE space_id=$1`, spaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pairs := map[string]bool{}
	for rows.Next() {
		var low, high uuid.UUID
		if err := rows.Scan(&low, &high); err != nil {
			return nil, err
		}
		pairs[pairKey(low, high)] = true
	}
	return pairs, rows.Err()
}

// createInferredEdge is the only path that writes origin='auto'. It is separate
// from createEdge because an inferred edge carries a confidence, which the
// database refuses on every other origin.
// canWriteToSpace answers whether this person may change the space at all.
//
// The same condition the note and edge writes carry, in one place so a new
// caller can ask instead of inventing its own version of it.
func (s *Store) canWriteToSpace(ctx context.Context, userID, spaceID uuid.UUID) (bool, error) {
	var allowed bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM spaces s LEFT JOIN space_members m ON m.space_id=s.id AND m.user_id=$1
			WHERE s.id=$2 AND (s.owner_id=$1 OR m.permission IN ('edit','manage'))
		)`, userID, spaceID).Scan(&allowed)
	return allowed, err
}

func (s *Store) createInferredEdge(ctx context.Context, userID, spaceID, source, target uuid.UUID, confidence float64) (Edge, error) {
	edge := Edge{
		SpaceID: spaceID, SourceID: source, TargetID: target,
		Relation: RelationRelated, Origin: OriginAuto, Confidence: &confidence,
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Edge{}, err
	}
	defer tx.Rollback(ctx)
	// The same condition the ordinary edge path carries. Checked again here
	// rather than trusted from the caller: this insert is what actually changes
	// someone's workspace, and a guard that lives only in the caller is one
	// refactor away from being gone.
	if err = tx.QueryRow(ctx, `
		INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,confidence,created_by)
		SELECT $1,$2,$3,'related','auto',$4,$5
		WHERE EXISTS(
			SELECT 1 FROM spaces s LEFT JOIN space_members m ON m.space_id=s.id AND m.user_id=$5
			WHERE s.id=$1 AND (s.owner_id=$5 OR m.permission IN ('edit','manage'))
		)
		RETURNING id`, spaceID, source, target, confidence, userID).Scan(&edge.ID); err != nil {
		return Edge{}, err
	}
	if err = s.AppendSpaceEvent(ctx, tx, userID, spaceID, "edge.created", edge.ID, edge); err != nil {
		return Edge{}, err
	}
	return edge, tx.Commit(ctx)
}

// AcceptSuggestion turns an inferred edge into one the person stands behind.
//
// The confidence is cleared along with the origin: once someone has decided a
// connection is real, a score describing how much it stood out from the rest of
// the workspace no longer says anything about it.
func (s *Store) AcceptSuggestion(ctx context.Context, userID, edgeID uuid.UUID) (Edge, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Edge{}, err
	}
	defer tx.Rollback(ctx)
	var edge Edge
	err = tx.QueryRow(ctx, `
		UPDATE note_edges e
		SET origin='manual', confidence=NULL
		WHERE e.id=$1 AND e.origin='auto'
		  AND EXISTS(
		    SELECT 1 FROM spaces s LEFT JOIN space_members m ON m.space_id=s.id AND m.user_id=$2
		    WHERE s.id=e.space_id AND (s.owner_id=$2 OR m.permission IN ('edit','manage')))
		RETURNING id,space_id,source_note_id,target_note_id,relation,origin,confidence,reason`, edgeID, userID).
		Scan(&edge.ID, &edge.SpaceID, &edge.SourceID, &edge.TargetID, &edge.Relation, &edge.Origin, &edge.Confidence, &edge.Reason)
	if err != nil {
		return Edge{}, err
	}
	if err = s.AppendSpaceEvent(ctx, tx, userID, edge.SpaceID, "edge.created", edge.ID, edge); err != nil {
		return Edge{}, err
	}
	return edge, tx.Commit(ctx)
}

// DeleteEdge removes a connection. It is how a suggestion is dismissed, and also
// the first way anything could remove an edge at all — until now a line drawn by
// accident stayed forever.
func (s *Store) DeleteEdge(ctx context.Context, userID, edgeID uuid.UUID) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var spaceID, source, target uuid.UUID
	var origin Origin
	err = tx.QueryRow(ctx, `
		DELETE FROM note_edges e
		WHERE e.id=$1
		  AND EXISTS(
		    SELECT 1 FROM spaces s LEFT JOIN space_members m ON m.space_id=s.id AND m.user_id=$2
		    WHERE s.id=e.space_id AND (s.owner_id=$2 OR m.permission IN ('edit','manage')))
		RETURNING e.space_id,e.source_note_id,e.target_note_id,e.origin`, edgeID, userID).
		Scan(&spaceID, &source, &target, &origin)
	if err != nil {
		return err
	}
	// Deleting a suggestion is an answer, and it has to stick. Deleting a line a
	// person drew is not: they may well want it proposed again later, so only
	// inferred edges leave a dismissal behind.
	if origin == OriginAuto {
		low, high := source, target
		if high.String() < low.String() {
			low, high = high, low
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO link_dismissals(space_id,low_note_id,high_note_id,dismissed_by) VALUES($1,$2,$3,$4)
			 ON CONFLICT (low_note_id,high_note_id) DO UPDATE SET dismissed_by=EXCLUDED.dismissed_by,dismissed_at=now()`,
			spaceID, low, high, userID); err != nil {
			return err
		}
	}
	if err = s.AppendSpaceEvent(ctx, tx, userID, spaceID, "edge.deleted", edgeID, map[string]any{"id": edgeID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
