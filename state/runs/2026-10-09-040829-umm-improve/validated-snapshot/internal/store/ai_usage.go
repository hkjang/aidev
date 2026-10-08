package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

/*
What a person's own thoughts were sent out for.

umm goes to some trouble to let someone hold a note back from analysis, hold a
whole space back, and keep note bodies off an embedding gateway. All of that is
a promise, and until now there was no screen on which a person could check it.
ai_calls existed, but only an administrator could read it, and it recorded what
a call cost rather than what it was.

This is the reading side of that promise. It answers for one person, about
their own calls, and it is careful about the two ways such a screen lies:

  - An empty list looks like "nothing happened" when it can equally mean "the
    log was cleaned up". So the retention window travels with the answer.
  - A list of chat-model calls looks like the whole story when embeddings go to
    a gateway too. Those are batched across whoever's notes are in the batch,
    so attributing one to a person would be a guess; the current policy is
    reported instead, which is a fact.
*/

// AIUsageEntry is one call, in the terms of the person who caused it.
type AIUsageEntry struct {
	At time.Time `json:"at"`
	// Purpose is what the call was for. Empty on calls recorded before
	// v0.67.0 — which means "not recorded", not "unknown purpose", and the
	// screen has to say the difference.
	Purpose string `json:"purpose"`
	Model   string `json:"model"`
	// Status is "success" or "failed". A failed call still sent the prompt, so
	// it belongs on this list.
	Status       string `json:"status"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

// AIUsage is one person's record, with the things that stop it being read as
// more, or less, than it is.
type AIUsage struct {
	Entries []AIUsageEntry `json:"entries"`
	// Counts is how many calls of each purpose fall inside the window, over all
	// of them rather than only the ones listed.
	Counts map[string]int `json:"counts"`
	// Total is every call in the window, listed or not.
	Total int `json:"total"`
	// Truncated says the window held more than Entries shows.
	Truncated bool `json:"truncated"`
	// RetentionDays is how long umm keeps these at all. Without it, a list that
	// stops three months back reads as three quiet months.
	RetentionDays int `json:"retentionDays"`
}

// maxAIUsageEntries bounds one page of the answer. Someone reading their own
// record is looking for what happened lately, not paging through a year.
const maxAIUsageEntries = 200

// PersonalAIUsage returns the calls made on one person's behalf, newest first.
//
// Scoped by user_id in the query rather than filtered afterwards: this is
// somebody else's record the moment that is got wrong.
func (s *Store) PersonalAIUsage(ctx context.Context, userID uuid.UUID, since time.Time) (AIUsage, error) {
	usage := AIUsage{Entries: []AIUsageEntry{}, Counts: map[string]int{}}

	rows, err := s.Pool.Query(ctx, `
		SELECT created_at,purpose,model,status,input_tokens,output_tokens
		FROM ai_calls
		WHERE user_id=$1 AND created_at>=$2
		ORDER BY created_at DESC
		LIMIT $3`, userID, since, maxAIUsageEntries+1)
	if err != nil {
		return AIUsage{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var entry AIUsageEntry
		if err := rows.Scan(&entry.At, &entry.Purpose, &entry.Model, &entry.Status,
			&entry.InputTokens, &entry.OutputTokens); err != nil {
			return AIUsage{}, err
		}
		usage.Entries = append(usage.Entries, entry)
	}
	if err := rows.Err(); err != nil {
		return AIUsage{}, err
	}
	// One more than the page was asked for, so "there is more" is known rather
	// than guessed from a full page.
	if len(usage.Entries) > maxAIUsageEntries {
		usage.Entries = usage.Entries[:maxAIUsageEntries]
		usage.Truncated = true
	}

	// Counted in the database rather than from the page above: the totals are
	// about the window, and the page is not the window.
	counts, err := s.Pool.Query(ctx, `
		SELECT purpose,count(*) FROM ai_calls
		WHERE user_id=$1 AND created_at>=$2
		GROUP BY purpose`, userID, since)
	if err != nil {
		return AIUsage{}, err
	}
	defer counts.Close()
	for counts.Next() {
		var purpose string
		var n int
		if err := counts.Scan(&purpose, &n); err != nil {
			return AIUsage{}, err
		}
		usage.Counts[purpose] = n
		usage.Total += n
	}
	return usage, counts.Err()
}
