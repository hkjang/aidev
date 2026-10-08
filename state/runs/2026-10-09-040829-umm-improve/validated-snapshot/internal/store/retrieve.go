package store

import (
	"context"

	"github.com/google/uuid"
)

// Answering a question from someone's memory means deciding what to read before
// deciding what to say. This is that first half, and it is separate on purpose:
// what gets retrieved determines whether an answer can be grounded at all, and
// it is testable without a language model in the loop.
//
// The rule that matters most here is exclusion. A note or a space marked as
// excluded from AI must not reach a gateway, and retrieval is exactly where that
// gets forgotten — umm shipped that defect once already, in a path that captured
// content before checking the flag.

// Retrieved is one thought pulled in to answer a question, and why.
type Retrieved struct {
	Note NoteSearchResult `json:"note"`
	// Via says how this thought was reached: matched the question directly, or
	// was connected to something that did.
	Via string `json:"via"`
	// Through names the thought it was reached from, when Via is a connection.
	Through *uuid.UUID `json:"through,omitempty"`
	// Branch says which line of thinking this thought belongs to, when it belongs
	// to one. An abandoned line reads exactly like a live one without it, which is
	// how someone ends up acting on the option they already rejected.
	Branch *BranchRef `json:"branch,omitempty"`
}

const (
	// ViaMatch: the thought matched the question itself.
	ViaMatch = "match"
	// ViaConnection: the thought is connected to one that matched. Following the
	// graph one step is what makes this different from search — the answer to a
	// question is often next to the thought that mentions it, not in it.
	ViaConnection = "connection"
)

// RetrievalResult is what was gathered, and what was deliberately left out.
type RetrievalResult struct {
	Thoughts []Retrieved `json:"thoughts"`
	// Excluded counts thoughts that matched but are marked as excluded from AI.
	// It is reported rather than silently dropped: an answer built from less than
	// the person expected should say so, and the number is the only honest way to
	// tell them without showing the content.
	Excluded int `json:"excluded"`
}

// maxRetrievalNeighbours bounds how far one match spreads. A well-connected
// thought can reach dozens, and filling the context with one note's neighbourhood
// crowds out the other matches.
const maxRetrievalNeighbours = 3

// RetrieveForQuestion gathers the thoughts an answer could be built from.
//
// It searches, then follows one step along the graph from each match, because
// the answer to a question is often recorded next to the thought that raises it
// rather than inside it. Anything marked as excluded from AI is dropped and
// counted, never returned.
func (s *Store) RetrieveForQuestion(ctx context.Context, userID uuid.UUID, question string, limit int) (RetrievalResult, error) {
	if limit < 1 || limit > 40 {
		limit = 12
	}
	page, err := s.SearchNotesHybrid(ctx, userID, SearchOptions{Query: question, Limit: limit})
	if err != nil {
		return RetrievalResult{}, err
	}

	result := RetrievalResult{Thoughts: []Retrieved{}}
	seen := map[uuid.UUID]bool{}
	matched := make([]uuid.UUID, 0, len(page.Notes))
	for _, note := range page.Notes {
		matched = append(matched, note.ID)
	}
	allowed, err := s.notesAllowedForAI(ctx, matched)
	if err != nil {
		return RetrievalResult{}, err
	}
	for _, note := range page.Notes {
		if !allowed[note.ID] {
			result.Excluded++
			continue
		}
		seen[note.ID] = true
		result.Thoughts = append(result.Thoughts, Retrieved{Note: note, Via: ViaMatch})
	}

	// One step along the graph from each match, in match order so the strongest
	// hits get their neighbourhood in first.
	for _, note := range page.Notes {
		if !allowed[note.ID] {
			continue
		}
		links, linkErr := s.Backlinks(ctx, userID, note.ID)
		if linkErr != nil {
			continue
		}
		added := 0
		for _, link := range links {
			if added >= maxRetrievalNeighbours {
				break
			}
			if seen[link.Note.ID] {
				continue
			}
			neighbourAllowed, allowErr := s.notesAllowedForAI(ctx, []uuid.UUID{link.Note.ID})
			if allowErr != nil || !neighbourAllowed[link.Note.ID] {
				if allowErr == nil {
					result.Excluded++
				}
				continue
			}
			seen[link.Note.ID] = true
			added++
			source := note.ID
			result.Thoughts = append(result.Thoughts, Retrieved{
				Note: NoteSearchResult{
					ID: link.Note.ID, SpaceID: link.Note.SpaceID, Title: link.Note.Title,
					Content: link.Note.Content, Kind: link.Note.Kind, UpdatedAt: link.Note.UpdatedAt,
					Reason: string(link.Edge.Relation),
				},
				Via: ViaConnection, Through: &source,
			})
		}
	}

	// Label every thought with the line it belongs to, once, after both passes.
	// Doing it here rather than in each query means a thought reached through a
	// connection cannot arrive unlabelled — the case that would matter most,
	// since nobody chose to look at it.
	labelled := make([]uuid.UUID, 0, len(result.Thoughts))
	for _, thought := range result.Thoughts {
		labelled = append(labelled, thought.Note.ID)
	}
	refs, err := s.branchRefsForNotes(ctx, labelled)
	if err != nil {
		return RetrievalResult{}, err
	}
	for index := range result.Thoughts {
		if ref, ok := refs[result.Thoughts[index].Note.ID]; ok {
			result.Thoughts[index].Branch = &ref
		}
	}
	return result, nil
}

// notesAllowedForAI reports which of the given notes may be sent to a gateway.
//
// A note is excluded if it says so or if its space does. Both are checked here
// rather than at the call site, because the call site is where this gets
// forgotten and the consequence is content leaving a deployment that asked it not
// to.
func (s *Store) notesAllowedForAI(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	allowed := map[uuid.UUID]bool{}
	if len(ids) == 0 {
		return allowed, nil
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT n.id, (n.ai_excluded OR sp.ai_excluded) AS excluded
		FROM notes n JOIN spaces sp ON sp.id=n.space_id
		WHERE n.id=ANY($1) AND n.deleted_at IS NULL`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var excluded bool
		if err := rows.Scan(&id, &excluded); err != nil {
			return nil, err
		}
		allowed[id] = !excluded
	}
	return allowed, rows.Err()
}
