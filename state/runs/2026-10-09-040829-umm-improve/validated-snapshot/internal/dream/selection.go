package dream

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

func pairKey(left, right uuid.UUID) string {
	if left.String() > right.String() {
		left, right = right, left
	}
	return left.String() + ":" + right.String()
}

// selectSources chooses the least recently dreamed eligible space and then
// ranks its notes for useful surprise: an unconnected, moderately related pair
// comes first, followed by notes that add information rather than repeating the
// same central cluster.
func (s *Service) selectSources(ctx context.Context, cfg Config, userID uuid.UUID) ([]sourceNote, error) {
	if cfg.MaxContextNotes < 2 {
		cfg.MaxContextNotes = 20
	}
	if cfg.ContextDays < 1 {
		cfg.ContextDays = 7
	}
	if cfg.MinNotes < 2 {
		cfg.MinNotes = 2
	}
	var includeOld bool
	_ = s.Store.Pool.QueryRow(ctx, `SELECT include_old_notes FROM user_preferences WHERE user_id=$1`, userID).Scan(&includeOld)
	var spaceID uuid.UUID
	err := s.Store.Pool.QueryRow(ctx, `
		SELECT n.space_id
		FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		WHERE n.author_id=$1 AND n.deleted_at IS NULL AND n.source!='dream'
		  AND n.ai_excluded=false AND sp.ai_excluded=false
		  AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
		  AND n.updated_at>now()-make_interval(days=>$2)
		  AND length(trim(n.content))>0
		GROUP BY n.space_id
		HAVING count(*) >= $3
		ORDER BY COALESCE((SELECT max(d.generated_at) FROM dream_notes d WHERE d.user_id=$1 AND d.space_id=n.space_id),'-infinity'::timestamptz),
		         max(n.updated_at) DESC
		LIMIT 1`, userID, cfg.ContextDays, cfg.MinNotes).Scan(&spaceID)
	if err != nil {
		return nil, errors.New("not enough eligible notes in one space")
	}
	recentLimit := cfg.MaxContextNotes
	if includeOld && recentLimit > 3 {
		recentLimit -= 2
	}
	rows, err := s.Store.Pool.Query(ctx, `
		SELECT n.id,n.space_id,n.content,n.x,n.y,n.updated_at
		FROM notes n JOIN spaces sp ON sp.id=n.space_id
		WHERE n.author_id=$1 AND n.space_id=$2 AND n.deleted_at IS NULL AND n.source!='dream'
		  AND n.ai_excluded=false AND sp.ai_excluded=false
		  AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
		  AND n.updated_at>now()-make_interval(days=>$3)
		  AND length(trim(content))>0
		ORDER BY n.updated_at DESC LIMIT $4`, userID, spaceID, cfg.ContextDays, recentLimit)
	if err != nil {
		return nil, err
	}
	sources := []sourceNote{}
	for rows.Next() {
		var note sourceNote
		if err := rows.Scan(&note.ID, &note.SpaceID, &note.Content, &note.X, &note.Y, &note.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		sources = append(sources, note)
	}
	rows.Close()
	if includeOld && len(sources) < cfg.MaxContextNotes {
		oldRows, oldErr := s.Store.Pool.Query(ctx, `
			SELECT n.id,n.space_id,n.content,n.x,n.y,n.updated_at
			FROM notes n JOIN spaces sp ON sp.id=n.space_id
			WHERE n.author_id=$1 AND n.space_id=$2 AND n.deleted_at IS NULL AND n.source!='dream'
			  AND n.ai_excluded=false AND sp.ai_excluded=false
			  AND (sp.owner_id=$1 OR EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=sp.id AND sm.user_id=$1))
			  AND n.updated_at<=now()-make_interval(days=>$3)
			  AND length(trim(content))>0
			ORDER BY n.updated_at ASC LIMIT $4`, userID, spaceID, cfg.ContextDays, cfg.MaxContextNotes-len(sources))
		if oldErr == nil {
			for oldRows.Next() {
				var note sourceNote
				if oldRows.Scan(&note.ID, &note.SpaceID, &note.Content, &note.X, &note.Y, &note.UpdatedAt) == nil {
					sources = append(sources, note)
				}
			}
			oldRows.Close()
		}
	}
	if len(sources) < cfg.MinNotes {
		return nil, errors.New("not enough eligible notes in one space")
	}
	ids := make([]uuid.UUID, len(sources))
	for index := range sources {
		ids[index] = sources[index].ID
	}
	connected := map[string]bool{}
	edgeRows, edgeErr := s.Store.Pool.Query(ctx, `SELECT source_note_id,target_note_id FROM note_edges WHERE space_id=$1 AND source_note_id=ANY($2) AND target_note_id=ANY($2)`, spaceID, ids)
	if edgeErr == nil {
		for edgeRows.Next() {
			var left, right uuid.UUID
			if edgeRows.Scan(&left, &right) == nil {
				connected[pairKey(left, right)] = true
			}
		}
		edgeRows.Close()
	}
	recentPairs := map[string]bool{}
	pairRows, pairErr := s.Store.Pool.Query(ctx, `
		SELECT left_source.source_note_id,right_source.source_note_id
		FROM dream_sources left_source
		JOIN dream_sources right_source ON right_source.dream_id=left_source.dream_id AND right_source.rank>left_source.rank
		JOIN dream_notes d ON d.dream_id=left_source.dream_id
		WHERE d.user_id=$1 AND d.space_id=$2 AND d.generated_at>now()-interval '30 days'`, userID, spaceID)
	if pairErr == nil {
		for pairRows.Next() {
			var left, right uuid.UUID
			if pairRows.Scan(&left, &right) == nil {
				recentPairs[pairKey(left, right)] = true
			}
		}
		pairRows.Close()
	}
	return rankSources(sources, connected, recentPairs, time.Now()), nil
}

func rankSources(sources []sourceNote, connected, recentPairs map[string]bool, now time.Time) []sourceNote {
	if len(sources) < 2 {
		return append([]sourceNote(nil), sources...)
	}
	vectors := make([][]float32, len(sources))
	centrality := make([]float64, len(sources))
	for index, source := range sources {
		vectors[index] = intelligence.Embed(source.Content)
	}
	for left := range sources {
		for right := range sources {
			if left != right {
				centrality[left] += intelligence.Cosine(vectors[left], vectors[right])
			}
		}
		centrality[left] /= float64(max(1, len(sources)-1))
	}
	// The bridge band is placed against this pool's own similarities. A fixed
	// peak at cosine 0.35 suited the offline algorithm and returned exactly zero
	// for every pair above 0.70, which is most genuinely related pairs once an
	// embedding model is configured — the 55% term would silently drop out and
	// Dream would pick on position and age alone.
	pairScores := make([]float64, 0, len(sources)*len(sources)/2)
	highest := 0.0
	for left := 0; left < len(sources); left++ {
		for right := left + 1; right < len(sources); right++ {
			score := intelligence.Cosine(vectors[left], vectors[right])
			pairScores = append(pairScores, score)
			highest = math.Max(highest, score)
		}
	}
	typical := intelligence.Typical(pairScores)

	bestLeft, bestRight, bestScore := 0, 1, -1.0
	for left := 0; left < len(sources); left++ {
		for right := left + 1; right < len(sources); right++ {
			similarity := intelligence.Cosine(vectors[left], vectors[right])
			// Enough shared context to be meaningful, but far enough apart that
			// connecting them says something new.
			bridge := intelligence.BridgeScore(similarity, typical, highest)
			distance := math.Hypot(sources[left].X-sources[right].X, sources[left].Y-sources[right].Y)
			spatial := math.Min(1, distance/900)
			days := math.Abs(sources[left].UpdatedAt.Sub(sources[right].UpdatedAt).Hours()) / 24
			temporal := math.Min(1, days/60)
			score := .55*bridge + .14*spatial + .11*temporal + .20*(centrality[left]+centrality[right])/2
			key := pairKey(sources[left].ID, sources[right].ID)
			if !connected[key] {
				score += .18
			}
			if recentPairs[key] {
				score -= .35
			}
			if score > bestScore {
				bestLeft, bestRight, bestScore = left, right, score
			}
		}
	}
	selected := []int{bestLeft, bestRight}
	used := map[int]bool{bestLeft: true, bestRight: true}
	for len(selected) < len(sources) {
		bestIndex, bestValue := -1, -1.0
		for candidate := range sources {
			if used[candidate] {
				continue
			}
			maxSimilarity := 0.0
			for _, chosen := range selected {
				maxSimilarity = math.Max(maxSimilarity, intelligence.Cosine(vectors[candidate], vectors[chosen]))
			}
			dormancy := math.Min(1, now.Sub(sources[candidate].UpdatedAt).Hours()/(24*180))
			value := .58*(1-maxSimilarity) + .27*centrality[candidate] + .15*dormancy
			if value > bestValue {
				bestIndex, bestValue = candidate, value
			}
		}
		if bestIndex < 0 {
			break
		}
		selected = append(selected, bestIndex)
		used[bestIndex] = true
	}
	result := make([]sourceNote, 0, len(sources))
	for _, index := range selected {
		result = append(result, sources[index])
	}
	return result
}

func sourceSpace(sources []sourceNote) uuid.UUID {
	if len(sources) == 0 {
		return uuid.Nil
	}
	return sources[0].SpaceID
}
