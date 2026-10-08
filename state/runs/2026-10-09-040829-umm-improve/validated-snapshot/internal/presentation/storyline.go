// Package presentation turns a space full of thoughts into the outline of a talk.
//
// The rule the whole package is built around: a slide's words are the words the
// person wrote. This compiler decides order, grouping and depth — never wording.
// A thought that reaches a slide reaches it verbatim, so a deck made from a
// space is still the author's own sentences rather than a machine's paraphrase
// of them.
//
// Everything here is derived from the graph the person drew. That is what makes
// the result reviewable: the same space always compiles to the same talk, and
// when the order looks wrong the answer is in the connections, not in a model's
// mood. It is also why this file has no dependency on anything that talks to a
// network — the part that can be wrong is the reasoning, and reasoning needs no
// database to test.
package presentation

import (
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// Thought is the part of a note this compiler reads.
type Thought struct {
	ID      uuid.UUID
	Title   string
	Content string
	Kind    string
	X, Y    float64
	// Width and Height are the note's own size on the canvas. Grouping measures
	// the gap between edges rather than centres, so a wide note does not appear
	// to swallow its neighbours. Zero means the default a note is created at.
	Width, Height float64
	AIExcluded    bool
}

// Link is a connection between two thoughts, with what it asserts.
type Link struct {
	From, To uuid.UUID
	Relation store.Relation
	Origin   store.Origin
	// Reason is why the connection was drawn, in the author's own words. Empty
	// for most connections, and for every one drawn before umm could record it.
	Reason string
}

// Role is what a slide is doing in the talk.
type Role string

const (
	// RoleSection opens a part of the talk. A question a person recorded makes
	// the best one: it is the thing the following slides are answering.
	RoleSection Role = "section"
	// RoleContent is a claim with whatever supports it underneath.
	RoleContent Role = "content"
	// RoleComparison holds two thoughts that contradict each other. umm records
	// disagreements instead of resolving them by deleting one side, so a deck
	// built from a space can show both — which is the one slide a summary of the
	// same notes would never produce.
	RoleComparison Role = "comparison"
)

// Point is one line of a slide, and the thought it came from.
type Point struct {
	Text  string
	From  uuid.UUID
	Depth int
}

// Slide is one slide, still in umm's terms rather than Ptium's.
type Slide struct {
	Role Role
	// Title is the thought's own title, or its first line when it has none.
	Title string
	// Lead is the sentence under the title. It is only set when the thought has
	// more to say than its title already says.
	Lead   string
	Points []Point
	// From lists every thought that reached this slide, in the order they appear.
	// It is what lets a reader of the deck get back to what they wrote.
	From []uuid.UUID
	// Grouped marks a slide whose thoughts were put together by where they sit
	// rather than by anything the person said. Its heading is the first
	// thought's — umm's guess at what the huddle is about, and the one heading
	// worth improving. A slide led by a thought its author titled, or a claim
	// with the evidence they attached, already says what it is.
	Grouped bool
	// Named marks a heading proposed by a model rather than derived from the
	// person's own words. The screen says so, because a deck whose headings
	// quietly changed source is one nobody can check.
	Named bool
	// Sectioned marks a slide that is only a part heading. It carries none of
	// the person's words and quotes no thought; it was added to divide a long
	// talk, and nothing was moved to make room for it.
	Sectioned bool
}

// Storyline is a talk: an ordered run of slides, and what was left out.
type Storyline struct {
	Title  string
	Slides []Slide
	// Excluded names thoughts the compiler deliberately did not use, so the
	// count on screen is the real one. Silently dropping a thought from someone's
	// own space is the one thing this must not do.
	Excluded []uuid.UUID
	// Sections is how many part headings are in the deck. Reported for the
	// same reason NamedHeadings is: the shape is the model's, not the person's.
	Sections int
	// NamedHeadings is how many headings in the deck came from a model.
	// Reported so the screen can say so: a deck whose headings quietly changed
	// source is one nobody can check.
	//
	// Both counts are of the deck as delivered, not of what a model was asked
	// for or answered. A cap applied over the finished deck can drop a part
	// heading or a named slide, and a count of what was proposed would then be
	// describing slides the person is not looking at.
	NamedHeadings int
	// Trimmed names thoughts left out only because the talk was capped at a
	// number of slides. Kept apart from Excluded because the reason is
	// different and so is the remedy: raising the cap brings these back, and
	// nothing brings back a thought its author held out of analysis.
	Trimmed []uuid.UUID
	// TrimmedSlides is how many slides the cap removed. Counted separately
	// because one slide can carry several thoughts, and "3 slides did not fit"
	// is the sentence a person asked for a length can act on.
	TrimmedSlides int
}

// Options tunes what the compiler is allowed to use.
type Options struct {
	// Title of the talk. Usually the space's name.
	Title string
	// IncludeExcluded overrides the note-level "keep this out of analysis" mark.
	// Off by default: a thought held back from Dream is being held back from
	// having things done to it, and being put on a slide is one of those things.
	IncludeExcluded bool
	// Only restricts the talk to these thoughts, for building from a selection
	// or a cluster rather than a whole space. Empty means everything.
	Only []uuid.UUID
	// OneSlidePerThought turns off grouping thoughts by where they sit. What
	// the person connected still travels together — that is what they said, not
	// something inferred. This is for a space whose arrangement means nothing,
	// where being read the notes one at a time is the honest result.
	OneSlidePerThought bool
	// MaxSlides caps the talk. Zero means no cap, which is the default: a deck
	// is a whole space until somebody says how long they have. When it is set,
	// the slides carrying least are dropped and what they held is recorded on
	// the storyline, never silently.
	MaxSlides int
}

// Compile turns thoughts and their connections into a talk.
//
// The order comes from what the person drew, in this order of authority:
//
//  1. a `follows` chain, which is someone stating a sequence outright;
//  2. left-to-right, then top-to-bottom placement, because on a canvas that is
//     already how a line of argument gets laid out;
//  3. the note's id, so that two thoughts sharing a position still land in the
//     same order on every run.
func Compile(thoughts []Thought, links []Link, opts Options) Storyline {
	usable, excluded := selectThoughts(thoughts, opts)
	story := Storyline{Title: strings.TrimSpace(opts.Title), Excluded: excluded}
	if len(usable) == 0 {
		return story
	}

	byID := make(map[uuid.UUID]Thought, len(usable))
	for _, t := range usable {
		byID[t.ID] = t
	}
	rel := indexLinks(links, byID)

	// Thoughts that end up inside another slide must not also open one of their
	// own, or the deck says the same thing twice.
	consumed := make(map[uuid.UUID]bool)
	var slides []Slide

	// A recorded disagreement is the most informative thing a space says about
	// a thought, and it needs both sides on one slide — so both sides are
	// spoken for before anything else may absorb them. Without this, a question
	// swallowed its answer and the contradiction it was part of vanished from
	// the deck entirely.
	reserved := reservedForComparison(rel, byID)

	// Thoughts nobody connected are grouped by where they were put, the way the
	// canvas groups them when it is too far out to read. Without this a space
	// that was written quickly and never drawn on produced one slide per note.
	// Only the unconnected ones: a stated relationship always wins, and grouping
	// a supporter by its position would carry it away from the claim it argues
	// for.
	grouped := map[uuid.UUID][]uuid.UUID{}
	if !opts.OneSlidePerThought {
		grouped = huddles(unconnected(usable, rel))
	}
	emitted := map[uuid.UUID]bool{}

	for _, t := range order(usable, rel) {
		if consumed[t.ID] {
			continue
		}
		if group, ok := grouped[t.ID]; ok {
			if emitted[t.ID] {
				continue
			}
			for _, slide := range huddleSlides(group, byID, order(usable, rel)) {
				slides = append(slides, slide)
			}
			for _, id := range group {
				emitted[id], consumed[id] = true, true
			}
			continue
		}
		if slide, taken, ok := comparisonSlide(t, rel, byID, consumed); ok {
			slides = append(slides, slide)
			for _, id := range taken {
				consumed[id] = true
			}
			continue
		}
		slide, taken := claimSlide(t, rel, byID, consumed, reserved)
		slides = append(slides, slide)
		for _, id := range taken {
			consumed[id] = true
		}
	}

	story.Slides = slides
	// Last: cutting decides what is in the talk, and it can only judge that
	// once every slide exists. Whatever does not fit is recorded on the
	// storyline rather than dropped quietly.
	fit(&story, opts.MaxSlides)
	return story
}

// selectThoughts applies the caller's restriction and the note-level mark,
// returning what may be used and what was deliberately left out.
func selectThoughts(thoughts []Thought, opts Options) ([]Thought, []uuid.UUID) {
	var only map[uuid.UUID]bool
	if len(opts.Only) > 0 {
		only = make(map[uuid.UUID]bool, len(opts.Only))
		for _, id := range opts.Only {
			only[id] = true
		}
	}

	usable := make([]Thought, 0, len(thoughts))
	var excluded []uuid.UUID
	for _, t := range thoughts {
		if only != nil && !only[t.ID] {
			continue // not asked for; not "left out" either
		}
		if t.AIExcluded && !opts.IncludeExcluded {
			excluded = append(excluded, t.ID)
			continue
		}
		if strings.TrimSpace(t.Content) == "" && strings.TrimSpace(t.Title) == "" {
			excluded = append(excluded, t.ID)
			continue
		}
		usable = append(usable, t)
	}
	return usable, excluded
}

// links, grouped by what they assert, and only between thoughts in play.
type related struct {
	// supports[id] are the thoughts arguing for id.
	supports map[uuid.UUID][]uuid.UUID
	// refines[id] are the thoughts stating id more precisely.
	refines map[uuid.UUID][]uuid.UUID
	// answers[id] are the thoughts answering id, which is a question.
	answers map[uuid.UUID][]uuid.UUID
	// contradicts is symmetric: a disagreement has no direction.
	contradicts map[uuid.UUID][]uuid.UUID
	// next[id] is what the person said comes after id.
	next map[uuid.UUID][]uuid.UUID
	// hasPrev marks a thought something else leads into, so chains can be
	// started only from their beginning.
	hasPrev map[uuid.UUID]bool
	// why[{a,b}] is the sentence someone wrote beside the connection between
	// those two thoughts. Stored both ways round for a disagreement, which has
	// no direction. Only their own words end up on a slide, so this is the one
	// place a connection can speak on one.
	why map[[2]uuid.UUID]string
	// touched marks both ends of every link umm kept. The maps above are keyed
	// by one side each, so neither of them answers "did the person say anything
	// about this thought at all" — and that is the question that decides whether
	// a thought may be grouped by where it sits.
	touched map[uuid.UUID]bool
}

// connected reports whether the person stated any relationship involving this
// thought. A stated relationship always beats an inferred one.
func (r related) connected(id uuid.UUID) bool { return r.touched[id] }

func indexLinks(links []Link, byID map[uuid.UUID]Thought) related {
	r := related{
		supports:    map[uuid.UUID][]uuid.UUID{},
		refines:     map[uuid.UUID][]uuid.UUID{},
		answers:     map[uuid.UUID][]uuid.UUID{},
		contradicts: map[uuid.UUID][]uuid.UUID{},
		next:        map[uuid.UUID][]uuid.UUID{},
		hasPrev:     map[uuid.UUID]bool{},
		touched:     map[uuid.UUID]bool{},
		why:         map[[2]uuid.UUID]string{},
	}
	for _, l := range links {
		if _, ok := byID[l.From]; !ok {
			continue
		}
		if _, ok := byID[l.To]; !ok {
			continue
		}
		switch l.Relation {
		case store.RelationSupports, store.RelationRefines, store.RelationAnswers,
			store.RelationContradicts, store.RelationFollows:
			r.touched[l.From], r.touched[l.To] = true, true
		}
		if reason := strings.TrimSpace(l.Reason); reason != "" {
			// Both ways round: a disagreement has no direction, and the slide
			// that shows it is built from whichever side the compiler reached
			// first.
			r.why[[2]uuid.UUID{l.From, l.To}] = reason
			r.why[[2]uuid.UUID{l.To, l.From}] = reason
		}
		switch l.Relation {
		case store.RelationSupports:
			r.supports[l.To] = append(r.supports[l.To], l.From)
		case store.RelationRefines:
			r.refines[l.From] = append(r.refines[l.From], l.To)
		case store.RelationAnswers:
			r.answers[l.To] = append(r.answers[l.To], l.From)
		case store.RelationContradicts:
			r.contradicts[l.From] = append(r.contradicts[l.From], l.To)
			r.contradicts[l.To] = append(r.contradicts[l.To], l.From)
		case store.RelationFollows:
			r.next[l.From] = append(r.next[l.From], l.To)
			r.hasPrev[l.To] = true
		}
	}
	return r
}

// order walks the thoughts into the sequence the talk follows.
func order(thoughts []Thought, rel related) []Thought {
	byID := make(map[uuid.UUID]Thought, len(thoughts))
	for _, t := range thoughts {
		byID[t.ID] = t
	}

	placed := make(map[uuid.UUID]bool, len(thoughts))
	var out []Thought

	// A `follows` chain is the person stating a sequence outright, so it is
	// taken whole and in its own order — this is what turns 문제 → 원인 → 대안
	// drawn on a canvas into the running order of a talk.
	var walk func(id uuid.UUID)
	walk = func(id uuid.UUID) {
		if placed[id] {
			return // a cycle, or a chain rejoining one already laid down
		}
		placed[id] = true
		out = append(out, byID[id])
		nexts := append([]uuid.UUID(nil), rel.next[id]...)
		sortIDs(nexts, byID)
		for _, n := range nexts {
			walk(n)
		}
	}

	// Placement is the backbone and a chain is a run inside it. Taking every
	// chain first instead put any chain's head in front of everything else on
	// the canvas: a space whose first slide should have been the question
	// someone wrote opened on a thought from the middle of the argument,
	// because that thought happened to start a two-note sequence.
	for _, t := range spatial(thoughts) {
		if placed[t.ID] || rel.hasPrev[t.ID] {
			continue // it will arrive with the chain that leads into it
		}
		walk(t.ID)
	}

	// Whatever a chain never reached — every thought in a cycle has a
	// predecessor, so none of them is a head.
	for _, t := range spatial(thoughts) {
		if !placed[t.ID] {
			placed[t.ID] = true
			out = append(out, t)
		}
	}
	return out
}

// spatial sorts thoughts the way a canvas is read: across, then down, with the
// id as a final tiebreak so the result never depends on map iteration.
func spatial(thoughts []Thought) []Thought {
	out := append([]Thought(nil), thoughts...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		// Thoughts within a column of each other are read top-to-bottom first;
		// a canvas is not a grid, so an exact x match is too strict to rely on.
		if !nearly(a.X, b.X) {
			return a.X < b.X
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.ID.String() < b.ID.String()
	})
	return out
}

// columnWidth is how far apart two thoughts must be before one counts as being
// to the right of the other rather than beside it. A default note is 240 wide.
const columnWidth = 260

func nearly(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < columnWidth
}

func sortIDs(ids []uuid.UUID, byID map[uuid.UUID]Thought) {
	sort.SliceStable(ids, func(i, j int) bool {
		a, b := byID[ids[i]], byID[ids[j]]
		if !nearly(a.X, b.X) {
			return a.X < b.X
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.ID.String() < b.ID.String()
	})
}

// comparisonSlide builds the slide for a recorded disagreement, if this thought
// is on one side of one.
func comparisonSlide(t Thought, rel related, byID map[uuid.UUID]Thought, consumed map[uuid.UUID]bool) (Slide, []uuid.UUID, bool) {
	others := append([]uuid.UUID(nil), rel.contradicts[t.ID]...)
	sortIDs(others, byID)
	for _, id := range others {
		if consumed[id] {
			continue
		}
		other := byID[id]
		slide := Slide{Role: RoleComparison, Title: headline(t), From: []uuid.UUID{t.ID, other.ID}}
		// Why they said these two disagree, if they said. It is their sentence,
		// so it may go on a slide; and this is the slide where "why is this a
		// disagreement" is the question an audience actually has.
		slide.Lead = rel.why[[2]uuid.UUID{t.ID, other.ID}]
		// The title is already this side of the disagreement, word for word, so
		// repeating it as a point puts the same sentence on the slide twice.
		// Dropped here rather than only when writing the source: the preview
		// renders these points directly, and a preview that shows a line the
		// deck will not is exactly the kind of lie this feature exists to
		// avoid.
		if text := body(t); text != slide.Title {
			slide.Points = append(slide.Points, Point{Text: text, From: t.ID})
		}
		slide.Points = append(slide.Points, Point{Text: body(other), From: other.ID})
		return slide, []uuid.UUID{t.ID, other.ID}, true
	}
	return Slide{}, nil, false
}

// reservedForComparison names every thought that is one side of a recorded
// disagreement, so that nothing absorbs it into an ordinary slide first.
func reservedForComparison(rel related, byID map[uuid.UUID]Thought) map[uuid.UUID]bool {
	reserved := make(map[uuid.UUID]bool)
	for id, others := range rel.contradicts {
		if _, ok := byID[id]; !ok {
			continue
		}
		for _, other := range others {
			if _, ok := byID[other]; ok {
				reserved[id] = true
				reserved[other] = true
			}
		}
	}
	return reserved
}

// claimSlide builds an ordinary slide: the thought, and what the person
// connected underneath it.
func claimSlide(t Thought, rel related, byID map[uuid.UUID]Thought, consumed map[uuid.UUID]bool, reserved map[uuid.UUID]bool) (Slide, []uuid.UUID) {
	slide := Slide{Role: RoleContent, Title: headline(t), From: []uuid.UUID{t.ID}}
	if rest := body(t); rest != "" && rest != slide.Title {
		slide.Lead = rest
	}
	// A question the person marked is what a part of the talk is about, and the
	// thoughts answering it are that part's content.
	if t.Kind == "question" {
		slide.Role = RoleSection
	}

	// Evidence follows its claim all the way down, rather than only one step.
	// Stopping at one left the support of a support stranded on a slide of its
	// own, next to the claim it was evidence for and no longer visibly attached
	// to it. Depth is what is capped, not the walk: past one level of indent a
	// slide stops being readable, so deeper evidence still travels with its
	// claim and simply sits at the same indent as the rest.
	taken := []uuid.UUID{t.ID}
	seen := map[uuid.UUID]bool{t.ID: true}
	var absorb func(parent uuid.UUID, depth int)
	absorb = func(parent uuid.UUID, depth int) {
		for _, id := range supporting(parent, rel, byID) {
			if consumed[id] || seen[id] || reserved[id] {
				continue
			}
			seen[id] = true
			child := byID[id]
			slide.Points = append(slide.Points, Point{Text: body(child), From: child.ID, Depth: min(depth, 1)})
			slide.From = append(slide.From, child.ID)
			taken = append(taken, child.ID)
			absorb(id, depth+1)
		}
	}
	absorb(t.ID, 0)
	return slide, taken
}

// supporting lists what belongs under a thought: what argues for it, what
// states it more precisely, and — when it is a question — what answers it.
func supporting(id uuid.UUID, rel related, byID map[uuid.UUID]Thought) []uuid.UUID {
	seen := map[uuid.UUID]bool{}
	var out []uuid.UUID
	for _, group := range [][]uuid.UUID{rel.answers[id], rel.supports[id], rel.refines[id]} {
		for _, child := range group {
			if seen[child] {
				continue
			}
			seen[child] = true
			out = append(out, child)
		}
	}
	sortIDs(out, byID)
	return out
}

// maxHeadlineRunes is how long a slide heading may be before it stops being one.
//
// A thought the person titled keeps that title whatever its length — they chose
// it. This only bounds the heading umm falls back to when there is no title,
// which is the thought's first line, and a first line is often the whole
// thought: measured on ordinary working notes, 79 characters across the top of
// a slide.
//
// Nothing is lost by cutting it. The heading and the body are separate fields,
// and Lead is set whenever the two differ — so a thought whose first line is
// too long to head a slide appears shortened at the top and complete
// underneath.
const maxHeadlineRunes = 28

// headline is what the slide is called: the thought's title when it has one,
// and otherwise the opening of its first line.
func headline(t Thought) string {
	if title := strings.TrimSpace(t.Title); title != "" {
		return title
	}
	content := strings.TrimSpace(t.Content)
	if content == "" {
		return ""
	}
	first, _, _ := strings.Cut(content, "\n")
	return shorten(strings.TrimSpace(first))
}

// shorten cuts a line down to something that can head a slide.
//
// A sentence ending inside the limit is the best heading available, because the
// person wrote it as a unit. Failing that it cuts at the last space, so a
// heading never ends mid-word, and marks the cut so nobody reads a truncated
// line as the whole thought.
func shorten(line string) string {
	runes := []rune(line)
	if len(runes) <= maxHeadlineRunes {
		return line
	}
	if end := firstSentenceEnd(runes); end > 0 && end <= maxHeadlineRunes {
		return strings.TrimSpace(string(runes[:end]))
	}
	cut := maxHeadlineRunes
	for i := maxHeadlineRunes; i > maxHeadlineRunes/2; i-- {
		if runes[i] == ' ' {
			cut = i
			break
		}
	}
	return strings.TrimSpace(string(runes[:cut])) + "…"
}

// firstSentenceEnd is one past the first sentence-ending mark, or 0 when the
// line does not end a sentence.
func firstSentenceEnd(runes []rune) int {
	for i, r := range runes {
		if r == '.' || r == '?' || r == '!' {
			return i + 1
		}
	}
	return 0
}

// body is the thought as it will be read aloud — its own words, unchanged.
func body(t Thought) string {
	if content := strings.TrimSpace(t.Content); content != "" {
		return content
	}
	return strings.TrimSpace(t.Title)
}
