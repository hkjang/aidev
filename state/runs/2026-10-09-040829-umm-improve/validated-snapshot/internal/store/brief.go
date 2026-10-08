package store

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

// What happened while you were not looking.
//
// The brief only counts things umm actually produced. It is tempting to show a
// row for contradictions or for thoughts whose importance rose overnight, but
// umm detects neither, and a zero next to a category it never examined reads as
// "there are none" rather than "nobody looked". So the brief also carries what
// it could not check and why: an empty duplicates list means something different
// depending on whether the backend was fit to find them.

// BriefGroup is a count of one kind of thing waiting.
type BriefGroup struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// DuplicatePair is two thoughts that look like the same thought written twice.
type DuplicatePair struct {
	SpaceID uuid.UUID `json:"spaceId"`
	Space   string    `json:"space"`
	First   Note      `json:"first"`
	Second  Note      `json:"second"`
	Score   float64   `json:"score"`
	// SetAside names the line one of the two belongs to, when that line was
	// decided against and the other thought is not in it.
	//
	// The pair then means something other than "written twice": it means work is
	// being redone that someone already chose not to do. Recording the decision
	// only helps if it comes back at the moment it is being repeated, and the
	// reason travels with it because "we rejected that" without "because" invites
	// rejecting it again for a different reason, or not at all.
	SetAside *BranchRef `json:"setAside,omitempty"`
	// SetAsideNoteID says which of the two is the one in that line, so the other
	// can be read as the thought being written now.
	SetAsideNoteID *uuid.UUID `json:"setAsideNoteId,omitempty"`
}

// BriefSkip records something umm did not look for, and why.
type BriefSkip struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

const (
	// SkipBackendNotSemantic: the active embedding scores shared vocabulary above
	// shared meaning, so any answer would be about wording rather than substance.
	SkipBackendNotSemantic = "backend-not-semantic"
	// SkipDisabled: an administrator turned the underlying feature off.
	SkipDisabled = "disabled"
	// SkipSpaceTooLarge: a space held more thoughts than one brief compares.
	// Finding duplicates means comparing every pair, so the work grows with the
	// square of the space: measured at 1024 dimensions, 1,000 thoughts take
	// 229ms, 2,000 take 916ms and 10,000 would take about 23 seconds — per space,
	// every morning. A brief that times out silently is worse than one that says
	// what it skipped.
	SkipSpaceTooLarge = "space-too-large"
)

// MorningBrief summarises what is waiting for a person.
type MorningBrief struct {
	Since time.Time `json:"since"`
	// Dreams waiting to be read, grouped by what kind of dream they are.
	Dreams []BriefGroup `json:"dreams"`
	// Suggestions is the number of connections umm proposed and nobody has
	// answered yet.
	Suggestions int `json:"suggestions"`
	// Unfiled is how many captured thoughts are still sitting in the inbox.
	Unfiled int `json:"unfiled"`
	// Duplicates are pairs that look like the same thought written twice.
	Duplicates []DuplicatePair `json:"duplicates"`
	// Contradictions are disagreements someone recorded. umm does not detect
	// them: these are connections a person or an agent drew. The interface shows
	// nothing at all when the list is empty, because a zero here would read as
	// "this workspace has none" when it means "nobody has marked any".
	Contradictions []Contradiction `json:"contradictions"`
	// Questions marked open with nothing recorded as answering them. Like
	// contradictions, both halves are marked rather than inferred, so an empty
	// list means nothing is marked open — not that everything is answered.
	Questions []OpenQuestion `json:"questions"`
	// Skipped says what umm did not examine, so an empty list above is not read
	// as an all-clear.
	Skipped []BriefSkip `json:"skipped"`
	// Quiet is true when there is genuinely nothing to report and nothing was
	// skipped — the only case where "nothing happened" is the whole story.
	Quiet bool `json:"quiet"`
}

// skipWorthMentioning returns the skip only when the check it describes had
// something to examine.
//
// A skip exists so an empty list is not read as an all-clear. That is a real
// service to someone with thoughts umm could not compare — and nothing at all
// to someone who has not written two yet. It was measured on a fresh account:
// the very first thing the home screen said, above the guide and the empty
// review, was that overlapping thoughts could not be checked because the
// embedding measures word overlap rather than meaning. Nothing had been
// skipped. There was nothing to skip.
//
// A duplicate is a pair, so two is the number that makes the check meaningful.
func (s *Store) skipWorthMentioning(ctx context.Context, userID uuid.UUID) *BriefSkip {
	skip := &BriefSkip{Kind: "duplicates", Reason: SkipBackendNotSemantic}
	var comparable int
	err := s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT 1 FROM notes n
			JOIN spaces sp ON sp.id=n.space_id
			LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$1
			WHERE n.deleted_at IS NULL AND n.archived=false AND (sp.owner_id=$1 OR m.user_id=$1)
			LIMIT 2
		) t`, userID).Scan(&comparable)
	if err != nil {
		// Not knowing is not a reason to go quiet: staying silent would be the
		// all-clear this whole mechanism exists to avoid.
		slog.Warn("could not tell whether the duplicate check had anything to examine", "user_id", userID, "error", err)
		return skip
	}
	if comparable < 2 {
		return nil
	}
	return skip
}

// maxDuplicateScanNotes bounds how many thoughts in one space are compared
// against each other.
//
// Set from measurement rather than taste: the pairwise pass costs 229ms at 1,000
// thoughts and grows quadratically, so this keeps the worst case per space near a
// third of a second. Above it the newest thoughts are compared and the rest are
// left, because a duplicate almost always involves something written recently —
// and the brief says the space was too large rather than reporting a clean bill
// of health it did not earn.
const maxDuplicateScanNotes = 1000

// recentNotesForDuplicateScan reads the newest thoughts in a space, and says
// whether there were more.
//
// A dedicated read rather than ListNotes-then-trim. ListNotes returns every
// thought and every connection in the space, so trimming afterwards meant
// loading two thousand notes and their edges to compare one thousand and throw
// the rest away — the cap saved the comparison and not the reading.
func (s *Store) recentNotesForDuplicateScan(ctx context.Context, userID, spaceID uuid.UUID) ([]Note, bool, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT n.id,n.space_id,n.author_id,n.content,n.title,n.color,n.kind,n.source,n.ai_excluded,
		       n.x,n.y,n.width,n.height,n.rotation,n.version,n.created_at,n.updated_at
		FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$1
		WHERE n.space_id=$2 AND n.deleted_at IS NULL AND (sp.owner_id=$1 OR m.user_id=$1)
		ORDER BY n.created_at DESC
		-- One past the bound, so "there were more" is answered by the read itself
		-- rather than by a second count.
		LIMIT $3`, userID, spaceID, maxDuplicateScanNotes+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	notes := []Note{}
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.SpaceID, &note.AuthorID, &note.Content, &note.Title,
			&note.Color, &note.Kind, &note.Source, &note.AIExcluded, &note.X, &note.Y,
			&note.Width, &note.Height, &note.Rotation, &note.Version, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, false, err
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	trimmed := len(notes) > maxDuplicateScanNotes
	if trimmed {
		notes = notes[:maxDuplicateScanNotes]
	}
	return notes, trimmed, nil
}

// sortDuplicatesForBrief promotes pairs that repeat a decision above ordinary
// duplicates, leaving the rest in similarity order.
//
// Stable on purpose: the label decides who rises, and everything else keeps the
// order similarity gave it.
func sortDuplicatesForBrief(pairs []DuplicatePair) {
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].SetAside != nil && pairs[j].SetAside == nil
	})
}

// maxDuplicateLabelBand is how many of the highest-scoring pairs are looked up
// before the list is trimmed to what a brief shows.
//
// Wider than what is shown so a pair repeating a decision can rise past ordinary
// duplicates, and bounded because each label costs a lookup and a workspace full
// of near-identical thoughts produces thousands of pairs.
const maxDuplicateLabelBand = 100

// maxDuplicatePairs bounds what one brief will show. A workspace that has been
// duplicating for months should produce a readable list, not a wall.
const maxDuplicatePairs = 10

// MorningBrief gathers everything waiting since the given time.
func (s *Store) MorningBrief(ctx context.Context, userID uuid.UUID, since time.Time) (MorningBrief, error) {
	// The brief's job is to say what is worth looking at, which means comparing
	// thoughts against each other. A thought waiting to be indexed scores
	// against nothing and simply would not appear — not as skipped, which the
	// brief reports honestly, but as absent. Indexing is a background job since
	// v0.72.0, so this catches up first: one bounded pass, finding nothing once
	// things have settled.
	s.SweepEmbeddingsOnce(ctx, nil)
	brief := MorningBrief{Since: since, Dreams: []BriefGroup{}, Duplicates: []DuplicatePair{},
		Contradictions: []Contradiction{}, Questions: []OpenQuestion{}, Skipped: []BriefSkip{}}

	rows, err := s.Pool.Query(ctx, `
		SELECT dream_type, count(*)
		FROM dream_notes
		WHERE user_id=$1 AND generated_at >= $2 AND status IN ('created','exposed')
		GROUP BY dream_type ORDER BY count(*) DESC, dream_type`, userID, since)
	if err != nil {
		return MorningBrief{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var group BriefGroup
		if err := rows.Scan(&group.Kind, &group.Count); err != nil {
			return MorningBrief{}, err
		}
		brief.Dreams = append(brief.Dreams, group)
	}
	if err = rows.Err(); err != nil {
		return MorningBrief{}, err
	}

	// Proposals nobody has answered. Scoped to spaces this person can reach, so
	// a shared workspace does not leak counts from one they cannot see.
	if err = s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM note_edges e
		JOIN spaces sp ON sp.id=e.space_id
		LEFT JOIN space_members m ON m.space_id=sp.id AND m.user_id=$1
		WHERE e.origin='auto' AND (sp.owner_id=$1 OR m.user_id=$1)`, userID).Scan(&brief.Suggestions); err != nil {
		return MorningBrief{}, err
	}

	if err = s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		WHERE sp.owner_id=$1 AND sp.is_inbox AND n.deleted_at IS NULL`, userID).Scan(&brief.Unfiled); err != nil {
		return MorningBrief{}, err
	}

	// Capped for the same reason as duplicates: a brief is a glance, not a report.
	recorded, err := s.Contradictions(ctx, userID, nil)
	if err != nil {
		return MorningBrief{}, err
	}
	if len(recorded) > maxDuplicatePairs {
		recorded = recorded[:maxDuplicatePairs]
	}
	brief.Contradictions = recorded

	questions, err := s.OpenQuestions(ctx, userID, nil)
	if err != nil {
		return MorningBrief{}, err
	}
	if len(questions) > maxDuplicatePairs {
		questions = questions[:maxDuplicatePairs]
	}
	brief.Questions = questions

	duplicates, skip, err := s.duplicateThoughts(ctx, userID)
	if err != nil {
		return MorningBrief{}, err
	}
	brief.Duplicates = duplicates
	if skip != nil {
		brief.Skipped = append(brief.Skipped, *skip)
	}

	brief.Quiet = len(brief.Dreams) == 0 && brief.Suggestions == 0 && brief.Unfiled == 0 &&
		len(brief.Duplicates) == 0 && len(brief.Contradictions) == 0 &&
		len(brief.Questions) == 0 && len(brief.Skipped) == 0
	return brief, nil
}

// duplicateThoughts finds pairs that look like the same thought recorded twice.
//
// This is the one place umm uses an absolute similarity rather than a
// distribution-relative one, and it is deliberate. Every other threshold is
// relative because backends disagree about what "close" means; near-identical
// text lands at the top of any sane embedding space, and the two models measured
// agree on where. A relative rule would also fail here for a specific reason: a
// workspace full of duplicates moves its own distribution, so the duplicates stop
// looking unusual precisely when there are the most of them.
func (s *Store) duplicateThoughts(ctx context.Context, userID uuid.UUID) ([]DuplicatePair, *BriefSkip, error) {
	report, err := s.MeasureEmbeddingQuality(ctx, false)
	// The offline algorithm scores genuinely different sentences as high as
	// 0.889 while some real duplicates fall to 0.505 — the ranges overlap, so
	// anything it reported here would be about wording, not substance.
	if err != nil || !report.Semantic {
		return []DuplicatePair{}, s.skipWorthMentioning(ctx, userID), nil
	}
	settings := s.IntelligenceSettings(ctx)
	spaces, err := s.ListSpaces(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	pairs := []DuplicatePair{}
	var skippedLarge bool
	for _, space := range spaces {
		notes, trimmed, listErr := s.recentNotesForDuplicateScan(ctx, userID, space.ID)
		if listErr != nil || len(notes) < 2 {
			continue
		}
		skippedLarge = skippedLarge || trimmed
		vectors := s.loadEmbeddings(ctx, notes)
		for i := 0; i < len(notes); i++ {
			for j := i + 1; j < len(notes); j++ {
				score := float64(intelligence.Cosine(vectors[notes[i].ID], vectors[notes[j].ID]))
				if score >= settings.DuplicateSimilarity {
					pairs = append(pairs, DuplicatePair{
						SpaceID: space.ID, Space: space.Name,
						First: notes[i], Second: notes[j], Score: score,
					})
				}
			}
		}

		// A repeated decision does not look like an ordinary duplicate. It pairs
		// something written recently with something rejected long ago, so the old
		// half sits outside the recency window above and the pair is never formed.
		// Trimming the space introduced exactly that hole, in the long-lived
		// workspaces this guard exists for. These thoughts are therefore carried in
		// and compared against the recent window — a cross-product, not another
		// full pass, because the question is only ever "is something new a repeat
		// of something set aside".
		if !trimmed {
			continue
		}
		setAside, asideErr := s.NotesInSetAsideLines(ctx, userID, space.ID)
		if asideErr != nil {
			slog.Warn("could not read set-aside thoughts for the duplicate pass",
				"space_id", space.ID, "error", asideErr)
			continue
		}
		inWindow := make(map[uuid.UUID]bool, len(notes))
		for _, note := range notes {
			inWindow[note.ID] = true
		}
		outside := setAside[:0]
		for _, note := range setAside {
			if !inWindow[note.ID] {
				outside = append(outside, note)
			}
		}
		if len(outside) == 0 {
			continue
		}
		asideVectors := s.loadEmbeddings(ctx, outside)
		for _, old := range outside {
			for _, recent := range notes {
				score := float64(intelligence.Cosine(asideVectors[old.ID], vectors[recent.ID]))
				if score >= settings.DuplicateSimilarity {
					pairs = append(pairs, DuplicatePair{
						SpaceID: space.ID, Space: space.Name,
						First: old, Second: recent, Score: score,
					})
				}
			}
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Score > pairs[j].Score })

	// Label a wider band than will be shown, then let the labels decide who stays.
	//
	// Sorting by score alone loses the pair that matters most. A workspace with
	// many near-identical thoughts fills all ten slots with ordinary duplicates
	// and drowns out the one where a decision is being repeated — observed on
	// real data, not imagined. "You wrote this twice" and "you are redoing what
	// you already rejected" are not the same news, and only the second has a
	// reason attached that the person needs.
	//
	// The band is bounded because labelling costs a lookup per pair and a
	// workspace can produce thousands.
	if len(pairs) > maxDuplicateLabelBand {
		pairs = pairs[:maxDuplicateLabelBand]
	}
	if err := s.markSetAsideDuplicates(ctx, pairs); err != nil {
		// The pairs are still worth showing without the label. Losing the whole
		// section because one lookup failed would be a worse trade.
		slog.Warn("could not label duplicate pairs with their line", "error", err)
	}
	sortDuplicatesForBrief(pairs)
	if len(pairs) > maxDuplicatePairs {
		pairs = pairs[:maxDuplicatePairs]
	}
	if skippedLarge {
		// Labelling still runs above: the pairs that were found are as good as any
		// others, and a large workspace is exactly where a repeated decision is
		// most likely. Only the coverage is partial, and the brief says so.
		return pairs, &BriefSkip{Kind: "duplicates", Reason: SkipSpaceTooLarge}, nil
	}
	return pairs, nil, nil
}

// markSetAsideDuplicates labels pairs where exactly one side sits in a line that
// was decided against.
//
// Exactly one, deliberately. Two duplicates inside the same set-aside line are
// just two duplicates — nothing is being repeated against a decision, because
// the decision covers both of them.
func (s *Store) markSetAsideDuplicates(ctx context.Context, pairs []DuplicatePair) error {
	if len(pairs) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(pairs)*2)
	for _, pair := range pairs {
		ids = append(ids, pair.First.ID, pair.Second.ID)
	}
	refs, err := s.branchRefsForNotes(ctx, ids)
	if err != nil {
		return err
	}
	for index := range pairs {
		first, firstIn := refs[pairs[index].First.ID]
		second, secondIn := refs[pairs[index].Second.ID]
		firstAside := firstIn && first.Status == BranchAbandoned
		secondAside := secondIn && second.Status == BranchAbandoned
		if firstAside == secondAside {
			continue
		}
		ref, noteID := first, pairs[index].First.ID
		if secondAside {
			ref, noteID = second, pairs[index].Second.ID
		}
		pairs[index].SetAside = &ref
		pairs[index].SetAsideNoteID = &noteID
	}
	return nil
}
