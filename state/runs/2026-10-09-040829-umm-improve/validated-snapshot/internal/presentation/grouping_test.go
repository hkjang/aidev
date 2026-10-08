package presentation

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// Grouping thoughts nobody connected.
//
// The rule it has to keep is not "fewer slides" — that is easy to get by losing
// things. It is that every thought still reaches exactly one slide, and that
// anything the person said about a thought still decides where it goes.

func at(x, y float64, text string) Thought {
	return Thought{ID: uuid.New(), Content: text, X: x, Y: y, Width: 240, Height: 160}
}

// reached counts how many slides each thought appears on, which is the only
// number that says whether the deck is still the whole space.
func reached(story Storyline) map[uuid.UUID]int {
	seen := map[uuid.UUID]int{}
	for _, slide := range story.Slides {
		for _, id := range slide.From {
			seen[id]++
		}
	}
	return seen
}

func everyThoughtOnExactlyOneSlide(t *testing.T, thoughts []Thought, story Storyline, context string) {
	t.Helper()
	seen := reached(story)
	for _, thought := range thoughts {
		switch seen[thought.ID] {
		case 1:
		case 0:
			t.Fatalf("%s: a thought reached no slide at all — grouping lost it", context)
		default:
			t.Fatalf("%s: a thought reached %d slides, so the deck says it twice", context, seen[thought.ID])
		}
	}
	if len(seen) != len(thoughts) {
		t.Fatalf("%s: %d thoughts reached slides, %d were given", context, len(seen), len(thoughts))
	}
}

func TestThoughtsPlacedTogetherShareASlide(t *testing.T) {
	// A huddle: four notes within half a note of each other, as a topic gets
	// laid out.
	thoughts := []Thought{
		at(0, 0, "회고 주기를 격주로"),
		at(300, 0, "논의가 얕아질 위험"),
		at(0, 200, "주제를 하나로 좁히기"),
		at(300, 200, "다음 스프린트에 시험"),
	}
	story := Compile(thoughts, nil, Options{Title: "회고"})
	if len(story.Slides) != 1 {
		t.Fatalf("four notes placed together made %d slides, want 1", len(story.Slides))
	}
	everyThoughtOnExactlyOneSlide(t, thoughts, story, "one huddle")
	if len(story.Slides[0].Points) != 3 {
		t.Fatalf("the slide carries %d points, want the other three thoughts", len(story.Slides[0].Points))
	}
}

func TestThoughtsPlacedApartStayApart(t *testing.T) {
	// Nothing is near anything, so there is no arrangement to read and one
	// slide each is the honest answer.
	thoughts := []Thought{
		at(0, 0, "첫째"),
		at(5000, 0, "둘째"),
		at(10000, 0, "셋째"),
	}
	story := Compile(thoughts, nil, Options{Title: "흩어짐"})
	if len(story.Slides) != 3 {
		t.Fatalf("three scattered notes made %d slides, want 3", len(story.Slides))
	}
	everyThoughtOnExactlyOneSlide(t, thoughts, story, "scattered")
}

// What the person drew always wins. A thought that argues for another belongs
// with it, wherever it happens to sit.
func TestAStatedRelationshipBeatsPosition(t *testing.T) {
	claim := at(0, 0, "격주 회고가 낫다")
	// Right next to the claim, and also next to a stranger. Without the rule it
	// could be grouped with either.
	evidence := at(300, 0, "지난 분기에 참석률이 올랐다")
	stranger := at(600, 0, "점심 메뉴 정하기")
	thoughts := []Thought{claim, evidence, stranger}
	links := []Link{{From: evidence.ID, To: claim.ID, Relation: store.RelationSupports}}

	story := Compile(thoughts, links, Options{Title: "회고"})
	everyThoughtOnExactlyOneSlide(t, thoughts, story, "stated relationship")

	var claimSlide *Slide
	for i := range story.Slides {
		if story.Slides[i].From[0] == claim.ID {
			claimSlide = &story.Slides[i]
		}
	}
	if claimSlide == nil {
		t.Fatal("the claim did not lead a slide")
	}
	found := false
	for _, id := range claimSlide.From {
		if id == evidence.ID {
			found = true
		}
		if id == stranger.ID {
			t.Fatal("an unrelated thought was grouped onto the claim's slide because it sat nearby")
		}
	}
	if !found {
		t.Fatal("the evidence was carried away from the claim it argues for")
	}
}

// A group bigger than a slide can hold continues onto the next one rather than
// growing one nobody can read.
func TestALargeHuddleBecomesSeveralReadableSlides(t *testing.T) {
	thoughts := make([]Thought, 20)
	for i := range thoughts {
		thoughts[i] = at(float64(i%5)*300, float64(i/5)*200, fmt.Sprintf("메모 %d", i))
	}
	story := Compile(thoughts, nil, Options{Title: "한 덩어리"})
	everyThoughtOnExactlyOneSlide(t, thoughts, story, "large huddle")
	if len(story.Slides) < 2 {
		t.Fatalf("twenty thoughts fitted into %d slide(s); a slide that long cannot be read", len(story.Slides))
	}
	for i, slide := range story.Slides {
		if len(slide.Points) > maxPointsPerSlide {
			t.Fatalf("slide %d carries %d points, more than the %d a slide may hold", i, len(slide.Points), maxPointsPerSlide)
		}
		// Every continuation is led by its own thought: a slide with points and
		// no heading is not a slide.
		if slide.Title == "" {
			t.Fatalf("slide %d has no heading", i)
		}
	}
}

// The escape hatch, for a space whose arrangement means nothing.
func TestOneSlidePerThoughtTurnsGroupingOff(t *testing.T) {
	thoughts := []Thought{
		at(0, 0, "가"), at(300, 0, "나"), at(0, 200, "다"), at(300, 200, "라"),
	}
	story := Compile(thoughts, nil, Options{Title: "끄기", OneSlidePerThought: true})
	if len(story.Slides) != len(thoughts) {
		t.Fatalf("with grouping off, %d thoughts made %d slides", len(thoughts), len(story.Slides))
	}
	everyThoughtOnExactlyOneSlide(t, thoughts, story, "grouping off")
}

// The same space must always produce the same deck. Grouping walks maps, and a
// map is the usual way that stops being true.
func TestGroupingIsTheSameEveryTime(t *testing.T) {
	thoughts := make([]Thought, 30)
	for i := range thoughts {
		thoughts[i] = at(float64(i%6)*300, float64(i/6)*200, fmt.Sprintf("메모 %d", i))
	}
	first := WriteSource(Compile(thoughts, nil, Options{Title: "반복"}), SourceOptions{Trace: true})
	for run := 0; run < 20; run++ {
		again := WriteSource(Compile(thoughts, nil, Options{Title: "반복"}), SourceOptions{Trace: true})
		if again != first {
			t.Fatalf("run %d produced a different deck from the same space", run)
		}
	}
}

// Grouping must not depend on the order the notes arrive in, only on where they
// are: the store is free to return them in any order it likes.
func TestGroupingDoesNotDependOnTheOrderNotesArriveIn(t *testing.T) {
	thoughts := make([]Thought, 18)
	for i := range thoughts {
		thoughts[i] = at(float64(i%6)*300, float64(i/6)*200, fmt.Sprintf("메모 %d", i))
	}
	forwards := WriteSource(Compile(thoughts, nil, Options{Title: "순서"}), SourceOptions{Trace: true})

	reversed := make([]Thought, len(thoughts))
	for i, t := range thoughts {
		reversed[len(thoughts)-1-i] = t
	}
	backwards := WriteSource(Compile(reversed, nil, Options{Title: "순서"}), SourceOptions{Trace: true})
	if forwards != backwards {
		t.Fatal("the same space produced a different deck when its notes arrived in a different order")
	}
}

// A thought whose size was never recorded still takes up space. Treating it as
// a point would make it look further from its neighbours than it is.
func TestAThoughtWithNoRecordedSizeStillOccupiesOne(t *testing.T) {
	// 300 apart: with the default width of 240 the gap between edges is 60,
	// inside reach. As points they would be 300 apart and outside it.
	a := Thought{ID: uuid.New(), Content: "가", X: 0, Y: 0}
	b := Thought{ID: uuid.New(), Content: "나", X: 300, Y: 0}
	story := Compile([]Thought{a, b}, nil, Options{Title: "크기 없음"})
	if len(story.Slides) != 1 {
		t.Fatalf("two adjacent notes with no recorded size made %d slides, want 1", len(story.Slides))
	}
}

func TestHuddlesIgnoresAGroupOfOne(t *testing.T) {
	lonely := at(0, 0, "혼자")
	if got := huddles([]Thought{lonely}); len(got) != 0 {
		t.Fatalf("a single thought was reported as a group: %v", got)
	}
	if got := huddles(nil); len(got) != 0 {
		t.Fatalf("no thoughts reported %d groups", len(got))
	}
}

// Single-link is the point: a row laid end to end is one train of thought, not
// a chain of pairs.
func TestARowLaidEndToEndIsOneGroup(t *testing.T) {
	thoughts := []Thought{
		at(0, 0, "하나"), at(300, 0, "둘"), at(600, 0, "셋"), at(900, 0, "넷"),
	}
	got := huddles(thoughts)
	if len(got) != 4 {
		t.Fatalf("%d of 4 thoughts were grouped", len(got))
	}
	for _, thought := range thoughts {
		if len(got[thought.ID]) != 4 {
			t.Fatalf("a thought in the row was put in a group of %d, want all four", len(got[thought.ID]))
		}
	}
}
