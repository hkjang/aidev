package store

import (
	"github.com/google/uuid"
	"math"
	"sort"
	"strings"

	"github.com/hkjang/umm/internal/intelligence"
)

const (
	// ClusterByMeaning: the notes were grouped by what they say.
	ClusterByMeaning = "meaning"
	// ClusterByProximity: the notes were grouped by where they sit. umm could not
	// judge meaning, and inventing a topic from shared vocabulary would be worse
	// than reporting the arrangement a person actually made.
	ClusterByProximity = "proximity"
)

// proximityReach is how close two notes must be to count as placed together,
// measured between their edges rather than their centres so a wide note does not
// swallow its neighbours.
//
// This is an absolute distance, and unlike a similarity threshold it can be: the
// canvas has a fixed scale that the person works in, a default note is 260 by
// 180, and "within about half a note of each other" means the same thing in
// every workspace.
const proximityReach = 120

// proximityClusters groups notes by where they were put.
//
// Single-link: a note joins a group if it is close to any member, so a row of
// notes laid end to end reads as one group rather than as a chain of pairs. That
// matches how people arrange things — a train of thought is placed in a line.
func proximityClusters(notes []Note) []ThoughtCluster {
	parent := make([]int, len(notes))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for i := range notes {
		for j := i + 1; j < len(notes); j++ {
			if edgeGap(notes[i], notes[j]) <= proximityReach {
				if a, b := find(i), find(j); a != b {
					parent[a] = b
				}
			}
		}
	}

	groups := map[int][]int{}
	for i := range notes {
		root := find(i)
		groups[root] = append(groups[root], i)
	}
	clusters := []ThoughtCluster{}
	for root, members := range groups {
		if len(members) < 2 {
			continue
		}
		ids := make([]uuid.UUID, 0, len(members))
		var text strings.Builder
		var spread float64
		for _, index := range members {
			ids = append(ids, notes[index].ID)
			text.WriteString(notes[index].Content)
			text.WriteString(" ")
		}
		for _, a := range members {
			for _, b := range members {
				if a < b {
					spread = math.Max(spread, edgeGap(notes[a], notes[b]))
				}
			}
		}
		label := "가까이 둔 생각"
		if keywords := intelligence.Keywords(text.String(), 2); len(keywords) > 0 {
			label = strings.Join(keywords, " · ")
		}
		// Cohesion here is tightness of placement mapped onto 0..1, so a caller
		// can rank groups. It is not comparable with a meaning cluster's number,
		// which is why Basis travels with it.
		cohesion := 1 / (1 + spread/(proximityReach*4))
		clusters = append(clusters, ThoughtCluster{
			ID: "near-" + notes[root].ID.String(), Label: label,
			NoteIDs: ids, Cohesion: cohesion, Basis: ClusterByProximity,
		})
	}
	sort.Slice(clusters, func(i, j int) bool {
		if len(clusters[i].NoteIDs) != len(clusters[j].NoteIDs) {
			return len(clusters[i].NoteIDs) > len(clusters[j].NoteIDs)
		}
		return clusters[i].ID < clusters[j].ID
	})
	return clusters
}

// edgeGap is the distance between two notes' rectangles, zero when they overlap.
func edgeGap(a, b Note) float64 {
	dx := math.Max(0, math.Max(a.X-(b.X+b.Width), b.X-(a.X+a.Width)))
	dy := math.Max(0, math.Max(a.Y-(b.Y+b.Height), b.Y-(a.Y+a.Height)))
	return math.Hypot(dx, dy)
}
