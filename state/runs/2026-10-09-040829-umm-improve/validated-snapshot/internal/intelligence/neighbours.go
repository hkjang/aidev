package intelligence

import (
	"runtime"
	"sync"
)

// Comparing every thought in a space with every other one.
//
// This is the largest cost of opening a canvas, and several other things do it
// too. At two thousand notes it is two million comparisons; it was measured at
// 163ms of a 243ms request, and clustering paid it twice over because the
// function it calls to load the notes had already done the same work and thrown
// the result away.
//
// Nothing here changes what is computed. The scores, the cutoff and the counts
// are the same numbers the plain loop produces — tested for equality rather
// than closeness, because the cutoff is compared against the very scores it was
// derived from, and a difference in the last bits moves a borderline pair
// across the line and changes an integer someone reads.

// sparse is the part of a vector that can contribute to a dot product.
//
// The local embedding is a char-gram histogram, so most of a note's dimensions
// are simply absent from its text: measured across a real space, 42 of 192
// carry a value. Multiplying the other 150 by zero is arithmetic whose answer
// is known in advance.
type sparse struct {
	indices []int32
	values  []float32
}

func compress(vector []float32) sparse {
	out := sparse{}
	for i, value := range vector {
		if value != 0 {
			out.indices = append(out.indices, int32(i))
			out.values = append(out.values, value)
		}
	}
	return out
}

// dot multiplies a compressed vector by a dense one.
//
// One side stays dense so the other can be walked directly rather than the two
// being merged: a merge would have to advance both index lists and compare,
// which costs more than the indexing it saves at these sizes.
func (s sparse) dot(dense []float32) float64 {
	// Accumulated in float64 and multiplied in float32, exactly as Cosine does.
	// Summing in float32 instead agreed to about eight decimal places, which
	// sounds close enough and is not.
	var total float64
	for k, index := range s.indices {
		if int(index) >= len(dense) {
			break
		}
		total += float64(s.values[k] * dense[index])
	}
	return clampSimilarity(total)
}

// Prepared is a set of vectors arranged so that comparing pairs of them costs
// as little as it can.
//
// Worth preparing once and passing around: the preparation is linear in the
// number of vectors, and everything it is handed to is quadratic.
type Prepared struct {
	dense    [][]float32
	sparse   []sparse
	pairs    []float64
	prepared bool
}

// Prepare compresses a set of vectors for repeated comparison.
func Prepare(vectors [][]float32) *Prepared {
	p := &Prepared{dense: vectors, sparse: make([]sparse, len(vectors)), prepared: true}
	for i, vector := range vectors {
		p.sparse[i] = compress(vector)
	}
	return p
}

// Len is how many vectors are in the set.
func (p *Prepared) Len() int {
	if p == nil {
		return 0
	}
	return len(p.dense)
}

// Score is how much two of them resemble each other. Identical to Cosine on the
// same pair.
func (p *Prepared) Score(i, j int) float64 {
	if p == nil || i < 0 || j < 0 || i >= len(p.dense) || j >= len(p.dense) {
		return 0
	}
	return p.sparse[i].dot(p.dense[j])
}

// pairScores is every pair once, in i<j order, computed on first use and kept.
//
// Kept because Counts wants it twice: once to derive a cutoff from the
// distribution, and again to judge each pair against it. It costs memory as the
// square of the space — 260MB at eight thousand thoughts — which is why
// PerNoteCounts, the one of these on the path of opening a canvas, no longer
// asks for it. Cutoff still does, and clustering still pays that.
func (p *Prepared) pairScores() []float64 {
	if p.pairs != nil || len(p.dense) < 2 {
		return p.pairs
	}
	scores := make([]float64, 0, len(p.dense)*(len(p.dense)-1)/2)
	for i := range p.dense {
		row := p.sparse[i]
		for j := i + 1; j < len(p.dense); j++ {
			scores = append(scores, row.dot(p.dense[j]))
		}
	}
	p.pairs = scores
	return scores
}

// Cutoff is the line this set's own distribution draws, so that a score means
// the same thing whichever embedding produced the vectors.
func (p *Prepared) Cutoff(band Band, fallback float64) float64 {
	if p == nil || len(p.dense) < 2 {
		return fallback
	}
	return NewSimilarityScale(p.pairScores()).ThresholdOr(band, fallback)
}

// Counts is, for each vector, how many of the others resemble it closely enough
// to count, together with the cutoff that decided it.
func (p *Prepared) Counts(band Band, fallback float64) ([]int, float64) {
	if p == nil {
		return nil, fallback
	}
	counts := make([]int, len(p.dense))
	if len(p.dense) < 2 {
		return counts, fallback
	}
	scores := p.pairScores()
	cutoff := NewSimilarityScale(scores).ThresholdOr(band, fallback)

	// Similarity is symmetric, so one walk in the same order credits both sides
	// of each pair.
	at := 0
	for i := range p.dense {
		for j := i + 1; j < len(p.dense); j++ {
			if scores[at] >= cutoff {
				counts[i]++
				counts[j]++
			}
			at++
		}
	}
	return counts, cutoff
}

// NeighbourCounts prepares a set and counts it in one call, for callers that
// have nothing else to do with the prepared form.
func NeighbourCounts(vectors [][]float32, band Band, fallback float64) ([]int, float64) {
	return Prepare(vectors).Counts(band, fallback)
}

// PerNoteCounts is, for each vector, how many others clear a line drawn from
// that vector's own scores rather than from every pair in the space.
//
// Counts and PerNoteCounts answer different questions and both are wanted. A
// line drawn across the whole space says how connected a thought is compared
// with the rest; a line drawn from one thought's own distribution says what
// that thought would show you if you asked it. The card shows a number and
// then opens the second one, so the card has to use the second one too —
// otherwise it offers a count nobody can reach.
//
// One row at a time, scored as it is read and never kept. This used to assemble
// each row out of a stored table of every pair, which saved arithmetic and cost
// memory that grows as the square: eight thousand thoughts is thirty-two
// million pairs and 260MB, measured, for a slice of numbers thrown away at the
// end of the request. A row is n floats. The rows are also independent of each
// other, which the stored table was not, so they are split across cores.
//
// The numbers are unchanged, and that is not an accident of rounding. A pair is
// scored here from the row's own thought — sparse[i] against dense[j] — where
// before, half of every row was read back from the other side of the diagonal,
// sparse[j] against dense[i]. Those two agree exactly: both walk the nonzero
// dimensions in ascending order and the terms the other side does not share
// contribute a true zero, which a float64 accumulator carries without drift.
// Within a row the order of summation is the order it always was, and no total
// is ever combined across goroutines, so nothing here depends on float addition
// being associative — which it is not.
func (p *Prepared) PerNoteCounts(band Band, fallback float64) []int {
	if p == nil {
		return nil
	}
	return p.perNoteCounts(perNoteWorkers(len(p.dense)), band, fallback)
}

// perNoteCounts is PerNoteCounts with the fan-out chosen by the caller, so a
// test can ask for any number of workers and check that the answer does not
// depend on it.
func (p *Prepared) perNoteCounts(workers int, band Band, fallback float64) []int {
	n := len(p.dense)
	counts := make([]int, n)
	if n < 2 {
		return counts
	}
	if workers < 1 {
		workers = 1
	}
	// Strided rather than blocked so that the stretch of the space holding the
	// densest vectors does not become the one worker everyone waits for. Each
	// goroutine writes its own indices of counts and shares nothing else.
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(first int) {
			defer wg.Done()
			row := make([]float64, 0, n-1)
			for i := first; i < n; i += workers {
				counts[i], row = p.countRow(i, row, band, fallback)
			}
		}(w)
	}
	wg.Wait()
	return counts
}

// countRow scores one thought against every other, draws the line from those
// scores alone, and counts what clears it. The buffer comes back so the next
// row reuses it.
func (p *Prepared) countRow(i int, buf []float64, band Band, fallback float64) (int, []float64) {
	row := buf[:0]
	from := p.sparse[i]
	for j := range p.dense {
		if j == i {
			continue
		}
		row = append(row, from.dot(p.dense[j]))
	}
	cutoff := NewSimilarityScale(row).ThresholdOr(band, fallback)
	count := 0
	for _, score := range row {
		if score >= cutoff {
			count++
		}
	}
	return count, row
}

// perNoteRowsPerWorker is how many rows have to be waiting before another
// goroutine is worth starting.
//
// Low, because reading a row costs more than it used to. A row is scored from
// the thought it belongs to, so a pair is multiplied once for each of its two
// sides — where the stored table multiplied it once and read it twice. Split
// across cores that is comfortably ahead; on one core it is behind, measured at
// 3.7ms against 6.0ms for five hundred thoughts. So the split starts early
// enough that the one-core case is reached only by spaces small enough for
// neither number to matter.
const perNoteRowsPerWorker = 64

// perNoteMaxWorkers bounds the fan-out. This runs inside somebody's request,
// and a machine serving several canvases at once should not hand one of them
// every core it has.
const perNoteMaxWorkers = 8

func perNoteWorkers(n int) int {
	workers := n / perNoteRowsPerWorker
	if limit := runtime.GOMAXPROCS(0); workers > limit {
		workers = limit
	}
	if workers > perNoteMaxWorkers {
		workers = perNoteMaxWorkers
	}
	if workers < 1 {
		return 1
	}
	return workers
}
