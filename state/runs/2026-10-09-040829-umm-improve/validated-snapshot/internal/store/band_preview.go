package store

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

// What a band change would do to the thoughts already here.
//
// The two bands decide how much of the graph a person sees: how many thoughts a
// card calls related, and how the canvas groups them once it is too far out to
// read. They are standard deviations above the mean of a space's own scores, so
// what a given number does depends entirely on the corpus — which meant the
// administrator screen offered two spin buttons whose effect could only be
// learned by saving them and going to look.
//
// Saving them and going to look is the wrong order. Everyone's canvas changes
// at once, and the person who changed it finds out last.
//
// So this measures both settings against the notes the installation actually
// holds, before anything is written: the current bands and the proposed ones,
// on the same sample, in one pass.

// BandOutcome is what one pair of bands does to the sampled notes.
type BandOutcome struct {
	RelatedBand float64 `json:"relatedBand"`
	ClusterBand float64 `json:"clusterBand"`

	// WithoutRelated is the number of notes whose card would show no related
	// thoughts at all. This is the number that matters most: a band raised too
	// far does not degrade gradually, it empties the panel.
	WithoutRelated int `json:"withoutRelated"`
	// MedianRelated and MostRelated describe the chip a person actually reads.
	MedianRelated int `json:"medianRelated"`
	MostRelated   int `json:"mostRelated"`

	Clusters       int `json:"clusters"`
	Grouped        int `json:"grouped"`
	LargestCluster int `json:"largestCluster"`
	// Ungrouped is what the summarised canvas would draw one by one.
	Ungrouped int `json:"ungrouped"`
}

// BandPreview is the comparison the administrator screen shows.
type BandPreview struct {
	Spaces int `json:"spaces"`
	Notes  int `json:"notes"`
	// Embedded is how many of those notes have a usable vector. A note without
	// one is in no band's answer, and saying so is the difference between "this
	// change groups nothing" and "nothing here has been embedded yet".
	Embedded int `json:"embedded"`
	// Semantic reports whether the active backend is judged fit to compare
	// meaning. When it is not, clustering falls back to position on the canvas
	// and the cluster band does nothing at all, so the cluster figures below are
	// left at zero and must not be read as counts — showing them anyway would be
	// describing arithmetic nobody will see. The related figures are unaffected:
	// related thoughts are scored whatever the backend.
	Semantic bool        `json:"semantic"`
	Current  BandOutcome `json:"current"`
	Proposed BandOutcome `json:"proposed"`
}

// bandPreviewSpaces and bandPreviewNotes bound the work. The comparison is
// quadratic in the notes of each space and runs while an administrator waits,
// so it samples the largest spaces rather than the whole installation: those
// are where a band change is felt, and a space of nine notes tells nobody
// anything.
const (
	bandPreviewSpaces = 5
	bandPreviewNotes  = 400
)

// PreviewBands measures the current and proposed bands against the same notes.
func (s *Store) PreviewBands(ctx context.Context, relatedBand, clusterBand float64) (BandPreview, error) {
	settings := s.IntelligenceSettings(ctx)
	preview := BandPreview{
		Current:  BandOutcome{RelatedBand: settings.RelatedBand, ClusterBand: settings.ClusterBand},
		Proposed: BandOutcome{RelatedBand: relatedBand, ClusterBand: clusterBand},
	}
	if report, err := s.MeasureEmbeddingQuality(ctx, false); err == nil {
		preview.Semantic = report.Semantic
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT n.space_id, count(*) AS total
		FROM notes n
		JOIN spaces sp ON sp.id=n.space_id
		WHERE n.deleted_at IS NULL
		GROUP BY n.space_id
		HAVING count(*) >= 2
		ORDER BY total DESC, n.space_id
		LIMIT $1`, bandPreviewSpaces)
	if err != nil {
		return preview, err
	}
	spaceIDs := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		var total int
		if err := rows.Scan(&id, &total); err != nil {
			rows.Close()
			return preview, err
		}
		spaceIDs = append(spaceIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return preview, err
	}

	relatedCurrent, relatedProposed := []int{}, []int{}
	for _, spaceID := range spaceIDs {
		notes, err := s.bandPreviewNotes(ctx, spaceID)
		if err != nil {
			return preview, err
		}
		if len(notes) < 2 {
			continue
		}
		preview.Spaces++
		preview.Notes += len(notes)

		vectors := s.loadEmbeddings(ctx, notes)
		ordered := make([][]float32, 0, len(notes))
		for _, note := range notes {
			vector, ok := vectors[note.ID]
			if !ok {
				continue
			}
			preview.Embedded++
			ordered = append(ordered, vector)
		}
		if len(ordered) < 2 {
			continue
		}

		// One preparation, four questions. Preparing is linear and everything
		// below it is quadratic, so the compression is worth doing once. The
		// two related passes each score every pair again — this is an estimate
		// run from a settings screen, not a canvas being opened, and the
		// alternative was holding a table of every pair in every space at once
		// while the loop moved on to the next one.
		prepared := intelligence.Prepare(ordered)
		relatedCurrent = append(relatedCurrent,
			prepared.PerNoteCounts(intelligence.Band(settings.RelatedBand), legacyRelatedCutoff)...)
		relatedProposed = append(relatedProposed,
			prepared.PerNoteCounts(intelligence.Band(relatedBand), legacyRelatedCutoff)...)
		// Only when the cluster band is what the canvas will actually use. On a
		// backend judged unfit to compare meaning the canvas groups by position
		// instead, and these numbers would describe a grouping nobody is going
		// to see.
		if preview.Semantic {
			addClustering(&preview.Current, prepared, settings.ClusterBand)
			addClustering(&preview.Proposed, prepared, clusterBand)
		}
	}

	summariseRelated(&preview.Current, relatedCurrent)
	summariseRelated(&preview.Proposed, relatedProposed)
	return preview, nil
}

// bandPreviewNotes reads a space's notes without a permission check, which is
// why nothing outside an administrator route may call it. It is the same
// ordering listNotes uses, so the sample is the space's oldest notes rather
// than an arbitrary slice of it.
func (s *Store) bandPreviewNotes(ctx context.Context, spaceID uuid.UUID) ([]Note, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, version FROM notes
		WHERE space_id=$1 AND deleted_at IS NULL
		ORDER BY created_at, id LIMIT $2`, spaceID, bandPreviewNotes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notes := []Note{}
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.Version); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

// addClustering runs the same greedy seeding the canvas runs, and counts what
// comes out. Counting rather than labelling: the labels cost keyword extraction
// over the joined text of every group, and nobody reads them here.
func addClustering(outcome *BandOutcome, prepared *intelligence.Prepared, band float64) {
	n := prepared.Len()
	cutoff := prepared.Cutoff(intelligence.Band(band), legacyClusterCutoff)
	used := make([]bool, n)
	for seed := 0; seed < n; seed++ {
		if used[seed] {
			continue
		}
		used[seed] = true
		size := 1
		for candidate := 0; candidate < n; candidate++ {
			if used[candidate] {
				continue
			}
			if prepared.Score(seed, candidate) >= cutoff {
				used[candidate] = true
				size++
			}
		}
		if size < 2 {
			// A seed that gathered nobody is not a group — the canvas draws it
			// as itself, which is what Ungrouped counts.
			outcome.Ungrouped++
			continue
		}
		outcome.Clusters++
		outcome.Grouped += size
		if size > outcome.LargestCluster {
			outcome.LargestCluster = size
		}
	}
}

func summariseRelated(outcome *BandOutcome, counts []int) {
	if len(counts) == 0 {
		return
	}
	sorted := append([]int(nil), counts...)
	sort.Ints(sorted)
	for _, count := range sorted {
		if count > 0 {
			break
		}
		outcome.WithoutRelated++
	}
	outcome.MedianRelated = sorted[len(sorted)/2]
	outcome.MostRelated = sorted[len(sorted)-1]
}
