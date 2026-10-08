package intelligence

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"testing"
)

// The whole claim of neighbours.go is that it computes the same thing as the
// obvious loop, only skipping arithmetic whose answer is known. So the test is
// not "the counts look reasonable" — it is that they are the same numbers, on
// inputs shaped like the ones a real space produces.

// dense is the loop this replaced, kept here as the thing to be equal to.
func dense(vectors [][]float32, band Band, fallback float64) ([]int, float64) {
	counts := make([]int, len(vectors))
	scores := []float64{}
	for i := range vectors {
		for j := i + 1; j < len(vectors); j++ {
			scores = append(scores, Cosine(vectors[i], vectors[j]))
		}
	}
	cutoff := NewSimilarityScale(scores).ThresholdOr(band, fallback)
	at := 0
	for i := range vectors {
		for j := i + 1; j < len(vectors); j++ {
			if scores[at] >= cutoff {
				counts[i]++
				counts[j]++
			}
			at++
		}
	}
	return counts, cutoff
}

// sparseVectors mimics the local char-gram embedding: about a fifth of the
// dimensions carry a value, which is what was measured on a real space.
func sparseVectors(n int, rng *rand.Rand) [][]float32 {
	out := make([][]float32, n)
	for i := range out {
		v := make([]float32, Dimensions)
		for k := 0; k < Dimensions/5; k++ {
			v[rng.Intn(Dimensions)] = rng.Float32()
		}
		out[i] = v
	}
	return out
}

func sameCounts(t *testing.T, got, want []int, context string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d counts, want %d", context, len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s: note %d counted %d, dense loop counted %d", context, i, got[i], want[i])
		}
	}
}

func TestCountsMatchTheDenseLoop(t *testing.T) {
	for _, n := range []int{2, 3, 17, 64, 200} {
		rng := rand.New(rand.NewSource(int64(n)))
		vectors := sparseVectors(n, rng)
		gotCounts, gotCutoff := NeighbourCounts(vectors, Band(1.1), 0.42)
		wantCounts, wantCutoff := dense(vectors, Band(1.1), 0.42)
		sameCounts(t, gotCounts, wantCounts, "sparse vectors")
		if gotCutoff != wantCutoff {
			t.Fatalf("n=%d: cutoff %v, dense loop %v", n, gotCutoff, wantCutoff)
		}
	}
}

// Dense vectors are the case where the shortcut saves nothing, and it must
// still be exactly right rather than merely close.
func TestCountsMatchOnVectorsWithNoZeros(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	vectors := make([][]float32, 80)
	for i := range vectors {
		v := make([]float32, Dimensions)
		for k := range v {
			v[k] = rng.Float32() * 0.1
		}
		vectors[i] = v
	}
	got, gotCutoff := NeighbourCounts(vectors, Band(0.6), 0.42)
	want, wantCutoff := dense(vectors, Band(0.6), 0.42)
	sameCounts(t, got, want, "dense vectors")
	if gotCutoff != wantCutoff {
		t.Fatalf("cutoff %v, dense loop %v", gotCutoff, wantCutoff)
	}
}

// A note whose text produced nothing has an all-zero vector. It resembles
// nothing, including other empty notes, and must not be counted as resembling
// them just because their dot product is equally zero.
func TestEmptyVectorsAreNotEachOthersNeighbours(t *testing.T) {
	vectors := [][]float32{
		make([]float32, Dimensions),
		make([]float32, Dimensions),
		make([]float32, Dimensions),
	}
	got, _ := NeighbourCounts(vectors, Band(1.1), 0.42)
	want, _ := dense(vectors, Band(1.1), 0.42)
	sameCounts(t, got, want, "empty vectors")
}

// Vectors of different lengths turn up when the embedding backend changes and
// old rows have not been rewritten yet. Neither path may read past the end.
func TestRaggedVectorsDoNotReadPastTheEnd(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	vectors := [][]float32{
		sparseVectors(1, rng)[0],
		make([]float32, 8),
		make([]float32, Dimensions*2),
		{},
	}
	for i := range vectors[1] {
		vectors[1][i] = rng.Float32()
	}
	for i := range vectors[2] {
		vectors[2][i] = rng.Float32()
	}
	got, _ := NeighbourCounts(vectors, Band(1.1), 0.42)
	want, _ := dense(vectors, Band(1.1), 0.42)
	sameCounts(t, got, want, "ragged vectors")
}

func TestTooFewToCompare(t *testing.T) {
	counts, cutoff := NeighbourCounts(nil, Band(1.1), 0.42)
	if len(counts) != 0 || cutoff != 0.42 {
		t.Fatalf("no vectors: %v %v", counts, cutoff)
	}
	counts, cutoff = NeighbourCounts([][]float32{make([]float32, Dimensions)}, Band(1.1), 0.42)
	if len(counts) != 1 || counts[0] != 0 || cutoff != 0.42 {
		t.Fatalf("one vector: %v %v", counts, cutoff)
	}
}

// The compressed dot product must equal the dense one on the same pair, not
// approximately: the counts are integers derived from a threshold, so a score
// that differs in the last bit can move a note across it.
func TestCompressedDotEqualsCosine(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	vectors := sparseVectors(300, rng)
	for i := 0; i < len(vectors); i += 7 {
		for j := i + 1; j < len(vectors); j += 11 {
			got := compress(vectors[i]).dot(vectors[j])
			want := Cosine(vectors[i], vectors[j])
			if math.Abs(got-want) > 1e-12 {
				t.Fatalf("pair (%d,%d): compressed %v, Cosine %v", i, j, got, want)
			}
		}
	}
}

func BenchmarkNeighbourCounts500(b *testing.B)  { benchmarkNeighbours(b, 500) }
func BenchmarkNeighbourCounts1000(b *testing.B) { benchmarkNeighbours(b, 1000) }
func BenchmarkNeighbourCounts2000(b *testing.B) { benchmarkNeighbours(b, 2000) }

func benchmarkNeighbours(b *testing.B, n int) {
	rng := rand.New(rand.NewSource(1))
	vectors := sparseVectors(n, rng)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NeighbourCounts(vectors, Band(1.1), 0.42)
	}
}

// Score has to be the same number Cosine gives, because clustering compares it
// against a cutoff derived from those same scores.
func TestPreparedScoreEqualsCosine(t *testing.T) {
	rng := rand.New(rand.NewSource(23))
	vectors := sparseVectors(120, rng)
	p := Prepare(vectors)
	for i := range vectors {
		for j := i + 1; j < len(vectors); j += 3 {
			if got, want := p.Score(i, j), Cosine(vectors[i], vectors[j]); math.Abs(got-want) > 1e-12 {
				t.Fatalf("pair (%d,%d): prepared %v, Cosine %v", i, j, got, want)
			}
		}
	}
}

func TestCutoffMatchesTheDenseLoop(t *testing.T) {
	for _, n := range []int{2, 9, 150} {
		rng := rand.New(rand.NewSource(int64(n) + 100))
		vectors := sparseVectors(n, rng)
		_, want := dense(vectors, Band(1.1), 0.42)
		if got := Prepare(vectors).Cutoff(Band(1.1), 0.42); got != want {
			t.Fatalf("n=%d: cutoff %v, dense loop %v", n, got, want)
		}
	}
}

// The pair scores are computed once and reused. Asking twice must not give two
// different answers, and must not silently recompute either.
func TestPreparedIsStableAcrossCalls(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	p := Prepare(sparseVectors(50, rng))
	firstCounts, firstCutoff := p.Counts(Band(1.1), 0.42)
	secondCutoff := p.Cutoff(Band(1.1), 0.42)
	secondCounts, thirdCutoff := p.Counts(Band(1.1), 0.42)
	if firstCutoff != secondCutoff || firstCutoff != thirdCutoff {
		t.Fatalf("cutoff moved: %v %v %v", firstCutoff, secondCutoff, thirdCutoff)
	}
	sameCounts(t, secondCounts, firstCounts, "second call")
}

// A different band has to give a different line, or the setting does nothing.
func TestABroaderBandCountsMoreNeighbours(t *testing.T) {
	rng := rand.New(rand.NewSource(31))
	p := Prepare(sparseVectors(120, rng))
	narrow, narrowCutoff := p.Counts(Band(2.0), 0.42)
	broad, broadCutoff := p.Counts(Band(0.2), 0.42)
	if !(broadCutoff < narrowCutoff) {
		t.Fatalf("a broader band should lower the line: %v vs %v", broadCutoff, narrowCutoff)
	}
	sum := func(v []int) int {
		t := 0
		for _, x := range v {
			t += x
		}
		return t
	}
	if sum(broad) <= sum(narrow) {
		t.Fatalf("a lower line counted no more: %d vs %d", sum(broad), sum(narrow))
	}
}

func TestNilPreparedIsHarmless(t *testing.T) {
	var p *Prepared
	if p.Len() != 0 || p.Score(0, 1) != 0 {
		t.Fatal("a nil set answered as though it held vectors")
	}
	if cutoff := p.Cutoff(Band(1.1), 0.42); cutoff != 0.42 {
		t.Fatalf("nil cutoff %v", cutoff)
	}
}

// densePerNote is the obvious loop PerNoteCounts replaced: score a vector
// against every other one, draw the line from those scores, count what clears
// it. The index arithmetic that reads a symmetric pair back from the i<j
// ordering is the only thing PerNoteCounts does differently, and getting it
// wrong would quietly count the wrong row.
func densePerNote(vectors [][]float32, band Band, fallback float64) []int {
	counts := make([]int, len(vectors))
	for i := range vectors {
		row := []float64{}
		for j := range vectors {
			if i == j {
				continue
			}
			row = append(row, Cosine(vectors[i], vectors[j]))
		}
		cutoff := NewSimilarityScale(row).ThresholdOr(band, fallback)
		for _, score := range row {
			if score >= cutoff {
				counts[i]++
			}
		}
	}
	return counts
}

func TestPerNoteCountsMatchTheDenseLoop(t *testing.T) {
	// 600 is past the point where the rows are split across goroutines, so the
	// dense loop is checking the parallel path and not only the serial one.
	// Asserted rather than assumed: raising perNoteRowsPerWorker would
	// otherwise quietly take this coverage away.
	if perNoteWorkers(600) < 2 {
		t.Fatalf("600 rows ran on %d worker(s); this test no longer covers the split", perNoteWorkers(600))
	}
	for _, n := range []int{2, 3, 17, 64, 200, 600} {
		rng := rand.New(rand.NewSource(int64(n) + 500))
		vectors := sparseVectors(n, rng)
		got := Prepare(vectors).PerNoteCounts(Band(0.6), 0.42)
		want := densePerNote(vectors, Band(0.6), 0.42)
		sameCounts(t, got, want, "per-note sparse vectors")
	}
}

// The line is drawn per vector, so a vector that resembles nothing is judged
// against its own flat distribution rather than the space's.
func TestPerNoteCountsHandleEmptyAndRaggedVectors(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	vectors := [][]float32{
		sparseVectors(1, rng)[0],
		make([]float32, Dimensions),
		make([]float32, 8),
		{},
	}
	got := Prepare(vectors).PerNoteCounts(Band(1.1), 0.42)
	want := densePerNote(vectors, Band(1.1), 0.42)
	sameCounts(t, got, want, "per-note ragged vectors")
}

func TestPerNoteCountsTooFewToCompare(t *testing.T) {
	var p *Prepared
	if got := p.PerNoteCounts(Band(0.6), 0.42); got != nil {
		t.Fatalf("nil set returned %v", got)
	}
	if got := Prepare([][]float32{make([]float32, Dimensions)}).PerNoteCounts(Band(0.6), 0.42); len(got) != 1 || got[0] != 0 {
		t.Fatalf("one vector returned %v", got)
	}
}

// Splitting the rows must not change a single count, whatever the split is.
// Every worker count from one upwards is compared against the same set done by
// a single worker, so a mistake in the striding — a row done twice, a row done
// by nobody, a worker writing over its neighbour — shows up as a difference
// rather than as a race the detector only sometimes reports.
func TestPerNoteCountsDoNotDependOnTheSplit(t *testing.T) {
	const n = 1500
	if perNoteWorkers(n) < 3 {
		t.Fatalf("%d rows ran on %d worker(s); this test no longer covers the split", n, perNoteWorkers(n))
	}
	rng := rand.New(rand.NewSource(77))
	vectors := sparseVectors(n, rng)
	p := Prepare(vectors)
	one := p.perNoteCounts(1, Band(0.6), 0.42)
	// Zero means one — the caller of last resort gets an answer rather than a
	// slice of zeros. More workers than rows is harmless: the extra ones find
	// nothing to do.
	sameCounts(t, p.perNoteCounts(0, Band(0.6), 0.42), one, "no workers asked for")
	for _, workers := range []int{2, 3, 5, 8, 16, n, n + 7} {
		sameCounts(t, p.perNoteCounts(workers, Band(0.6), 0.42), one, fmt.Sprintf("%d workers", workers))
	}
	sameCounts(t, p.PerNoteCounts(Band(0.6), 0.42), one, "the split it chooses itself")
}

// A score that lands exactly on the line is on the inside of it.
//
// Nothing random ever tests this: a cutoff is a mean plus a multiple of a
// deviation and no ordinary score falls on it. So the line is placed by hand.
// Three thoughts give each row two scores, below the four the relative scale
// needs, so the cutoff is the fallback constant — and the first pair is built
// to score exactly that. Whether the boundary is counted decides a number
// someone reads off a card, so it is fixed here rather than left to whichever
// comparison was typed.
func TestPerNoteCountsIncludeAScoreOnTheLine(t *testing.T) {
	on := make([]float32, Dimensions)
	on[0] = 1
	half := make([]float32, Dimensions)
	half[0] = 0.5
	vectors := [][]float32{on, half, make([]float32, Dimensions)}

	const fallback = 0.5
	if score := Prepare(vectors).Score(0, 1); score != fallback {
		t.Fatalf("the pair built to sit on the line scored %v, not %v", score, fallback)
	}
	got := Prepare(vectors).PerNoteCounts(Band(0.6), fallback)
	if got[0] != 1 {
		t.Fatalf("a score exactly on the line was counted %d times, want 1", got[0])
	}
	if got[2] != 0 {
		t.Fatalf("a thought resembling nothing counted %d, want 0", got[2])
	}
}

// A space small enough that the fan-out would cost more than the work is left
// alone, one core is never asked to start zero or negative workers, and no
// space however large takes more than the cap.
func TestPerNoteWorkersStaysWithinBounds(t *testing.T) {
	for _, n := range []int{0, 1, 2, perNoteRowsPerWorker - 1} {
		if got := perNoteWorkers(n); got != 1 {
			t.Fatalf("%d rows asked for %d workers, want 1", n, got)
		}
	}
	if got := perNoteWorkers(2 * perNoteRowsPerWorker); got < 2 && runtime.GOMAXPROCS(0) > 1 {
		t.Fatalf("%d rows asked for %d workers on %d cores, want at least 2",
			2*perNoteRowsPerWorker, got, runtime.GOMAXPROCS(0))
	}
	if got := perNoteWorkers(1 << 20); got > perNoteMaxWorkers {
		t.Fatalf("a million rows asked for %d workers, above the %d cap", got, perNoteMaxWorkers)
	}
}

func BenchmarkPerNoteCounts2000(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	vectors := sparseVectors(2000, rng)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Prepare(vectors).PerNoteCounts(Band(0.6), 0.42)
	}
}
