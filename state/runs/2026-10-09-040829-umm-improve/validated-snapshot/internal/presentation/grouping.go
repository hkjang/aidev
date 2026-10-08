package presentation

import (
	"math"
	"sort"

	"github.com/google/uuid"
)

/*
Thoughts nobody connected.

A thought that argues for another one travels with it: the deck already puts
evidence under its claim, both sides of a disagreement on one slide, and a
`follows` chain in order. That is the person telling umm what belongs together,
and it is the best signal there is.

What was missing is everything they did not say. A thought with no connection at
all became a slide of its own, so a space where somebody wrote quickly and never
drew a line produced one slide per note — two thousand notes, two thousand
slides. That is not a talk, and it is not what the space looks like either.

The canvas already answers this question. When it is too far out to read, it
draws one shape per group, and when it cannot judge meaning it groups by where
the notes sit, because a train of thought gets laid out in a line and a topic
gets laid out in a huddle. The deck now asks the same question the same way, so
a person looking at their summarised canvas and their deck sees the same
grouping rather than two different opinions about their space.

Position rather than meaning, deliberately. Compiling a deck is a pure function
of thoughts and links: the same space produces the same deck, offline, with no
embedding backend involved. Grouping by meaning would make the deck depend on
which model an administrator configured, and a deck that reshuffles itself when
the gateway changes is worse than one that follows the arrangement its author
actually made.
*/

// groupReach is how close two thoughts must be to count as placed together,
// measured between their edges. Same distance the canvas uses: a default note is
// 240 by 160, so this is about half a note.
const groupReach = 120

// maxPointsPerSlide is how many thoughts may ride along under the one that
// leads a slide. Past this a slide stops being something an audience can read,
// so a larger huddle becomes several slides in reading order rather than one
// unreadable one.
const maxPointsPerSlide = 6

// defaultThoughtWidth and defaultThoughtHeight are what a note is created at.
// A thought whose size was never recorded still occupies space, and treating it
// as a point would make it look further from its neighbours than it is.
const (
	defaultThoughtWidth  = 240
	defaultThoughtHeight = 160
)

func (t Thought) width() float64 {
	if t.Width > 0 {
		return t.Width
	}
	return defaultThoughtWidth
}

func (t Thought) height() float64 {
	if t.Height > 0 {
		return t.Height
	}
	return defaultThoughtHeight
}

// gap is the distance between two thoughts' nearest edges, and zero when they
// overlap.
func gap(a, b Thought) float64 {
	dx := math.Max(0, math.Max(a.X-(b.X+b.width()), b.X-(a.X+a.width())))
	dy := math.Max(0, math.Max(a.Y-(b.Y+b.height()), b.Y-(a.Y+a.height())))
	return math.Hypot(dx, dy)
}

// huddles groups thoughts by where they were put.
//
// Single-link, like the canvas: a thought joins a group if it is close to any
// member, so a row laid end to end reads as one group rather than a chain of
// pairs. Returns the group each thought belongs to, keyed by id; a thought with
// no neighbour is absent, because a group of one is just a thought.
func huddles(thoughts []Thought) map[uuid.UUID][]uuid.UUID {
	parent := make([]int, len(thoughts))
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
	for i := range thoughts {
		for j := i + 1; j < len(thoughts); j++ {
			if gap(thoughts[i], thoughts[j]) <= groupReach {
				if a, b := find(i), find(j); a != b {
					parent[a] = b
				}
			}
		}
	}

	members := map[int][]uuid.UUID{}
	for i := range thoughts {
		root := find(i)
		members[root] = append(members[root], thoughts[i].ID)
	}
	out := map[uuid.UUID][]uuid.UUID{}
	for _, group := range members {
		if len(group) < 2 {
			continue
		}
		for _, id := range group {
			out[id] = group
		}
	}
	return out
}

// unconnected lists the thoughts the person said nothing about.
//
// A thought at either end of any link has a stated relationship, and a stated
// relationship always wins: grouping it by where it happens to sit would put it
// on a slide away from the claim it argues for.
func unconnected(thoughts []Thought, rel related) []Thought {
	out := make([]Thought, 0, len(thoughts))
	for _, t := range thoughts {
		if rel.connected(t.ID) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// huddleSlides turns one group into the slides it needs.
//
// The thought that leads is the first one in reading order, and the rest become
// its points — the same shape a claim with its evidence already takes, so a
// group does not introduce a third kind of slide. The words are the person's
// throughout: nothing here writes a heading, because a deck compiled from
// somebody's thinking must not put sentences in their mouth.
//
// Past maxPointsPerSlide the group continues onto another slide rather than
// growing one nobody can read. Each continuation is led by its own first
// thought, so no slide is left without a heading.
func huddleSlides(group []uuid.UUID, byID map[uuid.UUID]Thought, reading []Thought) []Slide {
	// Reading order, taken from the order the whole talk is in, so a group sits
	// the way the canvas is read rather than the way union-find happened to
	// collect it.
	rank := make(map[uuid.UUID]int, len(reading))
	for i, t := range reading {
		rank[t.ID] = i
	}
	members := make([]Thought, 0, len(group))
	for _, id := range group {
		if t, ok := byID[id]; ok {
			members = append(members, t)
		}
	}
	sortByRank(members, rank)

	var slides []Slide
	for len(members) > 0 {
		take := len(members)
		if take > maxPointsPerSlide+1 {
			take = maxPointsPerSlide + 1
		}
		chunk := members[:take]
		members = members[take:]

		lead := chunk[0]
		slide := Slide{Role: RoleContent, Title: headline(lead), From: []uuid.UUID{lead.ID}, Grouped: true}
		if rest := body(lead); rest != "" && rest != slide.Title {
			slide.Lead = rest
		}
		if lead.Kind == "question" {
			slide.Role = RoleSection
		}
		for _, t := range chunk[1:] {
			slide.Points = append(slide.Points, Point{Text: body(t), From: t.ID})
			slide.From = append(slide.From, t.ID)
		}
		slides = append(slides, slide)
	}
	return slides
}

func sortByRank(thoughts []Thought, rank map[uuid.UUID]int) {
	sort.SliceStable(thoughts, func(i, j int) bool {
		return rank[thoughts[i].ID] < rank[thoughts[j].ID]
	})
}
