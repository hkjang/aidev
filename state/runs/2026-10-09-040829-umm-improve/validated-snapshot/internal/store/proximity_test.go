package store

import (
	"testing"

	"github.com/google/uuid"
)

func placed(x, y float64) Note {
	return Note{ID: uuid.New(), X: x, Y: y, Width: 260, Height: 180, Content: "생각"}
}

// umm is a spatial tool, so where a note sits is something a person decided.
// Grouping by that reports a real structure; grouping by shared vocabulary when
// the embedding cannot read meaning invents one.
func TestProximityGroupsWhatWasPlacedTogether(t *testing.T) {
	// Two obvious huddles, far apart.
	left := []Note{placed(0, 0), placed(300, 0), placed(150, 220)}
	right := []Note{placed(3000, 0), placed(3300, 0)}
	clusters := proximityClusters(append(append([]Note{}, left...), right...))

	if len(clusters) != 2 {
		t.Fatalf("expected the two huddles, got %d clusters", len(clusters))
	}
	if len(clusters[0].NoteIDs) != 3 || len(clusters[1].NoteIDs) != 2 {
		t.Errorf("groups came out as %d and %d", len(clusters[0].NoteIDs), len(clusters[1].NoteIDs))
	}
	for _, cluster := range clusters {
		if cluster.Basis != ClusterByProximity {
			t.Errorf("basis=%q; a caller cannot tell what grouped these", cluster.Basis)
		}
	}
}

// A row of notes laid end to end is one train of thought, not a chain of pairs.
func TestProximityFollowsARowOfNotes(t *testing.T) {
	row := []Note{}
	for i := 0; i < 6; i++ {
		row = append(row, placed(float64(i)*340, 0))
	}
	clusters := proximityClusters(row)
	if len(clusters) != 1 {
		t.Fatalf("a single row produced %d groups", len(clusters))
	}
	if len(clusters[0].NoteIDs) != 6 {
		t.Errorf("the row lost members: %d of 6", len(clusters[0].NoteIDs))
	}
}

// A note off on its own is not a group. Reporting it as one would fill a
// zoomed-out canvas with bubbles containing a single thought.
func TestProximityIgnoresLoneNotes(t *testing.T) {
	clusters := proximityClusters([]Note{placed(0, 0), placed(5000, 5000), placed(9000, 100)})
	if len(clusters) != 0 {
		t.Fatalf("scattered notes produced %d groups", len(clusters))
	}
}

// Overlapping notes are as together as notes get, and the arithmetic must not
// produce a negative distance or a cohesion outside its range.
func TestProximityHandlesOverlap(t *testing.T) {
	clusters := proximityClusters([]Note{placed(0, 0), placed(10, 10)})
	if len(clusters) != 1 {
		t.Fatalf("overlapping notes produced %d groups", len(clusters))
	}
	if gap := edgeGap(placed(0, 0), placed(10, 10)); gap != 0 {
		t.Errorf("overlapping notes measured %v apart", gap)
	}
	if c := clusters[0].Cohesion; c <= 0 || c > 1 {
		t.Errorf("cohesion %v is outside the range a caller can rank on", c)
	}
}

// The order has to be stable, or zooming out twice rearranges the same canvas.
func TestProximityOrderIsStable(t *testing.T) {
	notes := []Note{placed(0, 0), placed(300, 0), placed(3000, 0), placed(3300, 0), placed(3600, 0)}
	first := proximityClusters(notes)
	second := proximityClusters(notes)
	if len(first) != len(second) {
		t.Fatalf("two runs produced %d and %d groups", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Fatalf("group order changed between runs at %d", i)
		}
	}
	// Bigger groups first: at low zoom the largest shapes are what a person is
	// orienting by.
	if len(first[0].NoteIDs) < len(first[1].NoteIDs) {
		t.Error("a smaller group was ranked above a larger one")
	}
}
