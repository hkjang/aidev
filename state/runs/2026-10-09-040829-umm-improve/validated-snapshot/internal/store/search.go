package store

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

// noteTextExpression must stay byte-identical to the indexed expression created
// by migration 007 (notes_text_trgm_idx). Changing it without a matching
// migration silently drops lexical search back to a sequential scan.
const noteTextExpression = `(n.title || ' ' || n.content)`

// maxSearchTerms bounds how many trigram index probes one query can trigger.
// Terms past the limit are dropped, which only ever widens the result set.
const maxSearchTerms = 8

// semanticCandidateLimit caps how many recent non-matching notes are scored by
// vector similarity, so a large workspace cannot turn one search into a full scan.
const semanticCandidateLimit = 2000

// The semantic cutoffs umm used before thresholds became relative to the active
// embedding backend. They remain the fallback when a result set is too small to
// describe a distribution.
const (
	legacySemanticDropCutoff   = .18
	legacySemanticReasonCutoff = .28
)

// scoredResult holds a candidate between scoring and the distribution-aware
// filtering pass that decides whether it survives and how it is explained.
type scoredResult struct {
	result   NoteSearchResult
	lexical  float64
	semantic float64
	reasons  []string
}

// queryBuilder renders positional placeholders for predicates whose shape
// depends on the number of search terms.
type queryBuilder struct{ args []any }

func (b *queryBuilder) bind(value any) string {
	b.args = append(b.args, value)
	return "$" + strconv.Itoa(len(b.args))
}

// allMatch renders "expr ILIKE $1 AND expr ILIKE $2 ...".
//
// Each `ILIKE '%term%'` can be answered from a gin_trgm_ops index and the
// planner intersects the bitmaps. The pre-v0.8 formulation — NOT EXISTS over
// unnest() with a negated ILIKE — is logically the same but is never indexable,
// so it read every note body on every keystroke.
func allMatch(expression string, patterns []string, b *queryBuilder) string {
	if len(patterns) == 0 {
		return "false"
	}
	parts := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		parts = append(parts, expression+" ILIKE "+b.bind(pattern)+` ESCAPE E'\\'`)
	}
	return "(" + strings.Join(parts, " AND ") + ")"
}

func noteSearchPatterns(query string) []string {
	escape := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	terms := strings.Fields(strings.TrimSpace(query))
	if len(terms) > maxSearchTerms {
		terms = terms[:maxSearchTerms]
	}
	patterns := make([]string, 0, len(terms))
	for _, term := range terms {
		patterns = append(patterns, "%"+escape.Replace(term)+"%")
	}
	return patterns
}

// noteFilters renders the shared note predicates. Every placeholder is bound
// once and reused across the CTEs.
func noteFilters(b *queryBuilder, spaceID *uuid.UUID, kind string, updatedFrom, updatedTo any) string {
	space := b.bind(spaceID)
	kindParam := b.bind(strings.TrimSpace(kind))
	from := b.bind(updatedFrom)
	to := b.bind(updatedTo)
	return fmt.Sprintf(`n.deleted_at IS NULL
		  AND (%[1]s::uuid IS NULL OR n.space_id=%[1]s::uuid)
		  AND (%[2]s='' OR n.kind=%[2]s)
		  AND (%[3]s::timestamptz IS NULL OR n.updated_at>=%[3]s::timestamptz)
		  AND (%[4]s::timestamptz IS NULL OR n.updated_at<=%[4]s::timestamptz)`, space, kindParam, from, to)
}

// SearchNotesHybrid combines indexed lexical matching with vector similarity.
// It stays fully offline: with no configured embedding model the vectors are
// the local character n-gram algorithm and no external service is contacted.
func (s *Store) SearchNotesHybrid(ctx context.Context, userID uuid.UUID, options SearchOptions) (HybridSearchPage, error) {
	patterns := noteSearchPatterns(options.Query)
	if len(patterns) == 0 {
		return HybridSearchPage{Notes: []NoteSearchResult{}}, nil
	}
	if options.Limit < 1 {
		options.Limit = 20
	}
	if options.Limit > 50 {
		options.Limit = 50
	}
	if options.Offset < 0 {
		options.Offset = 0
	}
	provider := s.embeddingProviderForSearch(ctx, userID, options.SpaceID)
	queryVector, algorithm, policyLease := s.embedQueryWithPolicy(ctx, userID, options.SpaceID, options.Query, provider)
	if policyLease != nil {
		defer policyLease.rollback()
	}

	b := &queryBuilder{}
	user := b.bind(userID)
	filters := noteFilters(b, options.SpaceID, options.Kind, options.UpdatedFrom, options.UpdatedTo)
	noteMatch := allMatch(noteTextExpression, patterns, b)
	spaceMatch := allMatch("a.name", patterns, b)
	// The ranking columns are computed only for rows that already passed the
	// indexed filter, so an unnest() here costs nothing on the scan.
	patternList := b.bind(patterns)
	phrase := b.bind(strings.TrimSpace(options.Query))
	lexicalLimit := b.bind(hybridLexicalCandidateLimit)
	candidateLimit := b.bind(semanticCandidateLimit)
	algorithmParam := b.bind(algorithm)
	query := fmt.Sprintf(`
		WITH accessible AS (
		  SELECT sp.id,sp.name FROM spaces sp
		  WHERE sp.owner_id=%[1]s OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=%[1]s)
		), lexical AS (
		  SELECT n.id
		  FROM notes n JOIN accessible a ON a.id=n.space_id
		  WHERE %[2]s AND (%[3]s OR %[4]s)
		  -- Rank before the cut so an exact match is never crowded out by
		  -- newer partial matches when a workspace has more hits than the limit.
		  ORDER BY
		    (lower(btrim(n.title))=lower(btrim(%[6]s))) DESC,
		    (lower(btrim(n.content))=lower(btrim(%[6]s))) DESC,
		    (strpos(lower(n.title),lower(btrim(%[6]s)))>0) DESC,
		    (strpos(lower(n.content),lower(btrim(%[6]s)))>0) DESC,
		    (SELECT count(*) FROM unnest(%[5]s::text[]) AS term(pattern) WHERE n.title ILIKE term.pattern ESCAPE E'\\') DESC,
		    (SELECT count(*) FROM unnest(%[5]s::text[]) AS term(pattern) WHERE n.content ILIKE term.pattern ESCAPE E'\\') DESC,
		    (SELECT count(*) FROM unnest(%[5]s::text[]) AS term(pattern) WHERE a.name ILIKE term.pattern ESCAPE E'\\') DESC,
		    n.updated_at DESC,n.id DESC
		  LIMIT %[7]s
		), recent AS (
		  SELECT n.id FROM notes n JOIN accessible a ON a.id=n.space_id
		  WHERE %[2]s AND NOT EXISTS(SELECT 1 FROM lexical l WHERE l.id=n.id)
		  ORDER BY n.updated_at DESC,n.id DESC LIMIT %[8]s
		), candidates AS (
		  SELECT id,true AS lexical FROM lexical
		  UNION ALL
		  SELECT id,false AS lexical FROM recent
		)
		SELECT n.id,n.space_id,a.name,n.title,left(n.content,2000),n.kind,n.updated_at,e.vector,c.lexical
		FROM candidates c
		JOIN notes n ON n.id=c.id
		JOIN accessible a ON a.id=n.space_id
		LEFT JOIN note_embeddings e ON e.note_id=n.id AND e.content_version=n.version AND e.algorithm=%[9]s`,
		user, filters, noteMatch, spaceMatch, patternList, phrase, lexicalLimit, candidateLimit, algorithmParam)

	rows, err := s.Pool.Query(ctx, query, b.args...)
	if err != nil {
		return HybridSearchPage{}, err
	}
	defer rows.Close()
	// Embedding a missing vector inline is only safe for the local algorithm; a
	// gateway backed one would turn a single search into hundreds of calls, so
	// those rows are scored lexically until the background refresh catches up.
	local := algorithm == intelligence.LocalAlgorithm
	results := []NoteSearchResult{}
	scored := []scoredResult{}
	semanticScores := []float64{}
	for rows.Next() {
		var result NoteSearchResult
		var vector []float32
		var fullLexicalMatch bool
		if err := rows.Scan(&result.ID, &result.SpaceID, &result.SpaceName, &result.Title, &result.Content, &result.Kind, &result.UpdatedAt, &vector, &fullLexicalMatch); err != nil {
			return HybridSearchPage{}, err
		}
		if len(vector) == 0 && local {
			vector = intelligence.Embed(result.Title + " " + result.Content)
		}
		lexical, reasons := lexicalScore(options.Query, result.Title, result.Content, result.SpaceName)
		if fullLexicalMatch && lexical < .35 {
			lexical = .35
			reasons = append(reasons, "전체 본문 키워드 일치")
		}
		semantic := 0.0
		if len(vector) > 0 && len(queryVector) > 0 {
			semantic = math.Max(0, intelligence.Cosine(queryVector, vector))
		}
		result.Score = math.Round((lexical*.68+semantic*.32)*10000) / 10000
		// Whether this row is semantically close enough to keep, or to label as
		// such, can only be judged once every candidate has been scored: the raw
		// cosine that means "related" differs by embedding backend.
		scored = append(scored, scoredResult{result: result, lexical: lexical, semantic: semantic, reasons: reasons})
		semanticScores = append(semanticScores, semantic)
	}
	if err := rows.Err(); err != nil {
		return HybridSearchPage{}, err
	}
	semanticScale := intelligence.NewSimilarityScale(semanticScores)
	bands := s.IntelligenceSettings(ctx)
	dropCutoff := semanticScale.ThresholdOr(intelligence.Band(bands.RelatedBand), legacySemanticDropCutoff)
	labelCutoff := semanticScale.ThresholdOr(intelligence.Band(bands.StrongBand), legacySemanticReasonCutoff)
	for _, candidate := range scored {
		// A row with no keyword match earns its place only by being unusually
		// close to the query for this result set.
		if candidate.lexical == 0 && candidate.semantic < dropCutoff {
			continue
		}
		reasons := candidate.reasons
		if candidate.semantic >= labelCutoff {
			reasons = append(reasons, "의미상 유사")
		}
		if len(reasons) == 0 {
			reasons = append(reasons, "연관 표현")
		}
		item := candidate.result
		item.Reason = strings.Join(reasons, " · ")
		results = append(results, item)
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			if results[i].UpdatedAt.Equal(results[j].UpdatedAt) {
				return results[i].ID.String() > results[j].ID.String()
			}
			return results[i].UpdatedAt.After(results[j].UpdatedAt)
		}
		return results[i].Score > results[j].Score
	})
	start := min(options.Offset, len(results))
	end := min(start+options.Limit, len(results))
	page := HybridSearchPage{Notes: results[start:end], HasMore: end < len(results)}
	if page.HasMore {
		page.NextOffset = end
	}
	return page, nil
}

// SearchNotes powers the quick navigator. It is lexical only and ordered in the
// database, so it never loads more rows than it returns.
func (s *Store) SearchNotes(ctx context.Context, userID uuid.UUID, query string, limit int) ([]NoteSearchResult, error) {
	patterns := noteSearchPatterns(query)
	if len(patterns) == 0 {
		return []NoteSearchResult{}, nil
	}
	if limit < 1 {
		limit = 12
	}
	if limit > 30 {
		limit = 30
	}
	b := &queryBuilder{}
	user := b.bind(userID)
	noteMatch := allMatch(noteTextExpression, patterns, b)
	spaceMatch := allMatch("sp.name", patterns, b)
	limitParam := b.bind(limit)
	statement := fmt.Sprintf(`
		SELECT n.id,n.space_id,sp.name,n.title,left(n.content,500),n.kind,n.updated_at
		FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		WHERE n.deleted_at IS NULL
		  AND (sp.owner_id=%[1]s OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=%[1]s))
		  AND (%[2]s OR %[3]s)
		ORDER BY (NULLIF(trim(n.title),'') IS NOT NULL) DESC,n.updated_at DESC
		LIMIT %[4]s`, user, noteMatch, spaceMatch, limitParam)
	rows, err := s.Pool.Query(ctx, statement, b.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []NoteSearchResult{}
	for rows.Next() {
		var result NoteSearchResult
		if err := rows.Scan(&result.ID, &result.SpaceID, &result.SpaceName, &result.Title, &result.Content, &result.Kind, &result.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}
