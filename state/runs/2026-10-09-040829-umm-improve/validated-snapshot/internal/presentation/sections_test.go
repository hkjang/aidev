package presentation

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeSectioner struct {
	sections []Section
	err      error
	sent     []string
	calls    int
}

func (f *fakeSectioner) ProposeSections(_ context.Context, headings []string, _ string) ([]Section, error) {
	f.calls++
	f.sent = headings
	return f.sections, f.err
}

func flatStory(slides int) Storyline {
	story := Storyline{Title: "긴 발표"}
	for i := 0; i < slides; i++ {
		story.Slides = append(story.Slides, Slide{
			Role: RoleContent, Title: "제목", Lead: "본문", From: []uuid.UUID{uuid.New()},
		})
	}
	return story
}

// The whole design: sections add slides and move none.
func TestSectionsAddSlidesAndMoveNone(t *testing.T) {
	story := flatStory(20)
	before := append([]Slide(nil), story.Slides...)

	sectioner := &fakeSectioner{sections: []Section{{Start: 0, Title: "문제"}, {Start: 8, Title: "대안"}}}
	if added := sectionDeck(context.Background(), sectioner, &story, "ko"); added != 2 {
		t.Fatalf("added %d sections", added)
	}
	if len(story.Slides) != len(before)+2 {
		t.Fatalf("%d slides, want %d", len(story.Slides), len(before)+2)
	}

	// Every original slide is still there, in order, with its content and the
	// thoughts it came from.
	var kept []Slide
	for _, slide := range story.Slides {
		if slide.Sectioned {
			if len(slide.From) != 0 {
				t.Fatal("a part heading claims to quote a thought")
			}
			continue
		}
		kept = append(kept, slide)
	}
	if len(kept) != len(before) {
		t.Fatalf("%d original slides survived, want %d", len(kept), len(before))
	}
	for i := range before {
		if kept[i].Title != before[i].Title || kept[i].Lead != before[i].Lead {
			t.Fatalf("slide %d changed", i)
		}
		if len(kept[i].From) != len(before[i].From) || kept[i].From[0] != before[i].From[0] {
			t.Fatalf("slide %d lost the thought it came from", i)
		}
	}
}

// The boundaries have to land where they were asked to land, or the parts are
// dividing something other than what the model read.
func TestSectionsLandOnTheSlidesTheyNamed(t *testing.T) {
	story := flatStory(20)
	marked := story.Slides[8].From[0]
	sectioner := &fakeSectioner{sections: []Section{{Start: 0, Title: "문제"}, {Start: 8, Title: "대안"}}}
	sectionDeck(context.Background(), sectioner, &story, "ko")

	// The second part heading must sit immediately before the slide that was at
	// index 8, not one pushed along by the first insertion.
	for i, slide := range story.Slides {
		if slide.Sectioned && slide.Title == "대안" {
			next := story.Slides[i+1]
			if len(next.From) == 0 || next.From[0] != marked {
				t.Fatalf("the part heading landed before the wrong slide")
			}
			return
		}
	}
	t.Fatal("the second part heading is missing")
}

// Slide positions are what lets a slide say which thoughts it quotes, and
// inserting headings shifts every position after them.
func TestTracesStillPointAtTheRightSlidesAfterSectioning(t *testing.T) {
	story := flatStory(20)
	sectioner := &fakeSectioner{sections: []Section{{Start: 0, Title: "문제"}, {Start: 10, Title: "대안"}}}
	sectionDeck(context.Background(), sectioner, &story, "ko")

	sources := story.SlideSources()
	byPosition := map[int][]uuid.UUID{}
	for _, source := range sources {
		byPosition[source.SlidePosition] = append(byPosition[source.SlidePosition], source.NoteID)
	}
	position := 1 // the cover
	for _, slide := range story.Slides {
		position++
		if slide.Sectioned {
			if len(byPosition[position]) != 0 {
				t.Fatalf("position %d is a part heading and claims thoughts", position)
			}
			continue
		}
		got := byPosition[position]
		if len(got) != 1 || got[0] != slide.From[0] {
			t.Fatalf("position %d maps to %v, want the slide's own thought", position, got)
		}
	}
}

func TestAShortTalkIsNotDivided(t *testing.T) {
	story := flatStory(minSectionedSlides - 1)
	sectioner := &fakeSectioner{sections: []Section{{Start: 0, Title: "가"}, {Start: 4, Title: "나"}}}
	if added := sectionDeck(context.Background(), sectioner, &story, "ko"); added != 0 {
		t.Fatalf("a short talk was divided into %d parts", added)
	}
	if sectioner.calls != 0 {
		t.Fatal("the model was called for a talk too short to divide")
	}
}

func TestADeckStillCompilesWhenSectioningFails(t *testing.T) {
	story := flatStory(20)
	before := len(story.Slides)
	sectioner := &fakeSectioner{err: errors.New("gateway is down")}
	if added := sectionDeck(context.Background(), sectioner, &story, "ko"); added != 0 {
		t.Fatalf("added %d", added)
	}
	if len(story.Slides) != before {
		t.Fatal("a failed call changed the deck")
	}
}

func TestNoSectionerLeavesTheDeckAlone(t *testing.T) {
	story := flatStory(20)
	if added := sectionDeck(context.Background(), nil, &story, "ko"); added != 0 || len(story.Slides) != 20 {
		t.Fatal("the deck changed with no sectioner")
	}
}

// Everything a model gets wrong about boundaries, and what each would do to a
// talk if it were accepted.
func TestUnusableSectionsAreRefused(t *testing.T) {
	for _, c := range []struct {
		name     string
		proposed []Section
	}{
		{"out of order", []Section{{Start: 0, Title: "가"}, {Start: 9, Title: "나"}, {Start: 4, Title: "다"}}},
		{"past the end", []Section{{Start: 0, Title: "가"}, {Start: 99, Title: "나"}}},
		{"negative", []Section{{Start: -3, Title: "가"}, {Start: 5, Title: "나"}}},
		{"the same slide twice", []Section{{Start: 0, Title: "가"}, {Start: 0, Title: "나"}}},
		{"a part of one slide", []Section{{Start: 0, Title: "가"}, {Start: 1, Title: "나"}}},
		{"not starting at the beginning", []Section{{Start: 4, Title: "가"}, {Start: 9, Title: "나"}}},
		{"a sentence for a name", []Section{{Start: 0, Title: "이 부분은 회고 주기를 어떻게 바꿀지에 대해 여러 관점을 다룹니다"}, {Start: 9, Title: "나"}}},
		{"only one part", []Section{{Start: 0, Title: "전부"}}},
		{"nothing at all", nil},
	} {
		if got := usableSections(c.proposed, 20); len(got) != 0 {
			t.Errorf("%s: accepted %v", c.name, got)
		}
	}
}

// And a good answer is accepted, or the test above passes by refusing
// everything.
func TestAWellFormedDivisionIsAccepted(t *testing.T) {
	got := usableSections([]Section{{Start: 0, Title: "문제"}, {Start: 7, Title: "대안"}, {Start: 14, Title: "결론"}}, 20)
	if len(got) != 3 {
		t.Fatalf("accepted %d of 3 parts: %v", len(got), got)
	}
	if got[0].Start != 0 || got[2].Title != "결론" {
		t.Fatalf("parts came back as %v", got)
	}
}

// The last part must have slides in it too.
func TestAPartAtTheVeryEndIsRefused(t *testing.T) {
	got := usableSections([]Section{{Start: 0, Title: "가"}, {Start: 19, Title: "나"}}, 20)
	if len(got) != 0 {
		t.Fatalf("a part with one slide at the end was accepted: %v", got)
	}
}

// A model that proposes forty parts for a talk misunderstood the question, and
// keeping the first eight of them would divide the deck somewhere nobody chose.
func TestTooManyPartsAreRefusedRatherThanTrimmed(t *testing.T) {
	var proposed []Section
	for i := 0; i < 40; i++ {
		proposed = append(proposed, Section{Start: i * 2, Title: "부"})
	}
	if got := usableSections(proposed, 200); len(got) != 0 {
		t.Fatalf("%d parts accepted from a proposal that was not a division", len(got))
	}
}

func TestAnAnswerThatIsNotTheAskedForShapeIsIgnoredForSections(t *testing.T) {
	for _, reply := range []string{"", "문제와 대안", `{"sections":[{"start":0,"title":"가"}]}`, "[not json"} {
		if got := decodeSections(reply); len(got) != 0 {
			t.Fatalf("decodeSections(%q) = %v", reply, got)
		}
	}
	if got := decodeSections(`여기 있습니다: [{"start":0,"title":"문제"}]`); len(got) != 1 || got[0].Title != "문제" {
		t.Fatalf("a JSON array with chatter around it was not read: %v", got)
	}
}
