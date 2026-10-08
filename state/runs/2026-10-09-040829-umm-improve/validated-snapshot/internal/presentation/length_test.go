package presentation

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// contentSlide is a slide carrying `thoughts` of someone's own notes.
func contentSlide(title string, thoughts int) Slide {
	slide := Slide{Role: RoleContent, Title: title}
	for i := 0; i < thoughts; i++ {
		slide.From = append(slide.From, uuid.New())
	}
	return slide
}

// partHeading is a slide that is only a part heading: no words of the person's,
// no thought quoted.
func partHeading(title string) Slide {
	return Slide{Role: RoleSection, Title: title, Sectioned: true}
}

// dividedStory is a talk in three parts of five, five and two slides.
func dividedStory() Storyline {
	story := Storyline{Slides: []Slide{partHeading("문제")}}
	for i := 0; i < 5; i++ {
		story.Slides = append(story.Slides, contentSlide("문제 슬라이드", 1))
	}
	story.Slides = append(story.Slides, partHeading("대안"))
	for i := 0; i < 5; i++ {
		story.Slides = append(story.Slides, contentSlide("대안 슬라이드", 1))
	}
	story.Slides = append(story.Slides, partHeading("결정"))
	for i := 0; i < 2; i++ {
		story.Slides = append(story.Slides, contentSlide("결정 슬라이드", 1))
	}
	return story
}

// openHeadings reports the part headings that have a slide under them, and the
// ones that open nothing.
func openHeadings(story Storyline) (open, empty []string) {
	for index, slide := range story.Slides {
		if !slide.Sectioned {
			continue
		}
		if index+1 < len(story.Slides) && !story.Slides[index+1].Sectioned {
			open = append(open, slide.Title)
			continue
		}
		empty = append(empty, slide.Title)
	}
	return open, empty
}

// The whole point: what survives is what the person built most around.
func TestFitKeepsTheSlidesCarryingMost(t *testing.T) {
	story := Storyline{Slides: []Slide{
		contentSlide("혼자 적어 둔 것", 1),
		contentSlide("다섯이 떠받치는 주장", 5),
		contentSlide("역시 혼자", 1),
		contentSlide("셋이 떠받치는 주장", 3),
	}}

	if removed := fit(&story, 2); removed != 2 {
		t.Fatalf("removed %d slides, want 2", removed)
	}
	got := titles(story)
	want := []string{"다섯이 떠받치는 주장", "셋이 떠받치는 주장"}
	if len(got) != len(want) {
		t.Fatalf("kept %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("kept %v, want %v", got, want)
		}
	}
}

// Cutting decides what is in the talk, never what order it is in: that order is
// a follows chain or the layout the person made.
func TestFitKeepsTheOrderTheTalkWasIn(t *testing.T) {
	story := Storyline{Slides: []Slide{
		contentSlide("첫째", 3),
		contentSlide("둘째", 1),
		contentSlide("셋째", 9),
		contentSlide("넷째", 5),
	}}

	fit(&story, 3)

	// Ranked, "셋째" carries most and "첫째" least of the survivors — but the
	// talk still runs in the order it was written.
	got := titles(story)
	want := []string{"첫째", "셋째", "넷째"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("kept %v, want %v — cutting reordered the talk", got, want)
		}
	}
}

// umm records disagreements instead of resolving them by deleting a side. The
// slide holding both is the one a summary of the same notes would never
// produce, so it is the last thing a length cap may take.
func TestFitNeverDropsARecordedDisagreementFirst(t *testing.T) {
	disagreement := Slide{Role: RoleComparison, Title: "격주로 줄이자 / 얕아진다",
		From: []uuid.UUID{uuid.New(), uuid.New()}}
	story := Storyline{Slides: []Slide{
		contentSlide("떠받치는 주장이 많다", 9),
		disagreement,
		contentSlide("이것도 많다", 9),
		contentSlide("여기도", 9),
	}}

	fit(&story, 1)

	if len(story.Slides) != 1 || story.Slides[0].Role != RoleComparison {
		t.Fatalf("a length cap dropped the recorded disagreement and kept %v", titles(story))
	}
}

// Dropping a thought out of somebody's own space without saying so is the one
// thing this must never do.
func TestFitSaysWhatDidNotFit(t *testing.T) {
	kept := contentSlide("남는 슬라이드", 4)
	dropped := contentSlide("잘린 슬라이드", 2)
	alsoDropped := contentSlide("이것도 잘린다", 1)
	story := Storyline{Slides: []Slide{kept, dropped, alsoDropped}}

	fit(&story, 1)

	if story.TrimmedSlides != 2 {
		t.Fatalf("TrimmedSlides=%d, want 2", story.TrimmedSlides)
	}
	if len(story.Trimmed) != 3 {
		t.Fatalf("%d thoughts reported as not fitting, want 3 — a thought vanished silently", len(story.Trimmed))
	}
	reported := map[uuid.UUID]bool{}
	for _, id := range story.Trimmed {
		reported[id] = true
	}
	for _, slide := range []Slide{dropped, alsoDropped} {
		for _, id := range slide.From {
			if !reported[id] {
				t.Fatalf("thought %s was dropped without being reported", id)
			}
		}
	}
	// And nothing that stayed is also reported as missing.
	for _, id := range kept.From {
		if reported[id] {
			t.Fatalf("thought %s is on a slide and reported as not fitting", id)
		}
	}
}

// A cap is opt-in. Without one, a deck is the whole space.
func TestFitWithoutACapChangesNothing(t *testing.T) {
	for _, max := range []int{0, -1, 4, 99} {
		story := Storyline{Slides: []Slide{
			contentSlide("하나", 1), contentSlide("둘", 1),
			contentSlide("셋", 1), contentSlide("넷", 1),
		}}
		if removed := fit(&story, max); removed != 0 {
			t.Fatalf("max=%d removed %d slides", max, removed)
		}
		if len(story.Slides) != 4 || story.TrimmedSlides != 0 || len(story.Trimmed) != 0 {
			t.Fatalf("max=%d: %d slides, %d trimmed", max, len(story.Slides), story.TrimmedSlides)
		}
	}
}

// Two runs of the same space and the same cap give the same talk. That is what
// makes a preview worth reading.
func TestFitIsTheSameEveryTime(t *testing.T) {
	build := func() Storyline {
		return Storyline{Slides: []Slide{
			contentSlide("가", 2), contentSlide("나", 2), contentSlide("다", 2),
			contentSlide("라", 2), contentSlide("마", 2),
		}}
	}
	first, second := build(), build()
	fit(&first, 3)
	fit(&second, 3)
	a, b := titles(first), titles(second)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("two runs disagreed: %v vs %v", a, b)
		}
	}
	// Ties go to whichever comes first, so the opening survives.
	if a[0] != "가" {
		t.Fatalf("a tie did not go to the opening: %v", a)
	}
}

// Through the real compiler, from real thoughts, the way the API reaches it.
func TestCompileHonoursTheCap(t *testing.T) {
	var thoughts []Thought
	for i := 0; i < 12; i++ {
		thoughts = append(thoughts, Thought{
			ID: uuid.New(), Content: "생각", X: float64(i) * 2000, Y: 0,
		})
	}

	uncapped := Compile(thoughts, nil, Options{Title: "발표", OneSlidePerThought: true})
	if len(uncapped.Slides) != 12 {
		t.Fatalf("uncapped compile made %d slides, want 12", len(uncapped.Slides))
	}
	if uncapped.TrimmedSlides != 0 {
		t.Fatalf("an uncapped talk reported %d slides trimmed", uncapped.TrimmedSlides)
	}

	capped := Compile(thoughts, nil, Options{Title: "발표", OneSlidePerThought: true, MaxSlides: 5})
	if len(capped.Slides) != 5 {
		t.Fatalf("capped compile made %d slides, want 5", len(capped.Slides))
	}
	if capped.TrimmedSlides != 7 || len(capped.Trimmed) != 7 {
		t.Fatalf("capped compile reported %d slides / %d thoughts left out, want 7 / 7",
			capped.TrimmedSlides, len(capped.Trimmed))
	}
}

// A part heading outliving the part it opens is a title over nothing, and two
// of them in a row is a talk that says a part began twice and held nothing
// either time.
func TestALengthCapNeverLeavesAHeadingOverNothing(t *testing.T) {
	story := dividedStory()

	fit(&story, 8)

	open, empty := openHeadings(story)
	if len(empty) > 0 {
		t.Fatalf("part headings over nothing: %v (kept %v)", empty, titles(story))
	}
	if len(open) == 0 {
		t.Fatal("every part heading was cut, so this proves nothing")
	}
	// The parts that are left are the ones whose slides are left, and the one
	// whose slides all fell out of the length went with them.
	if len(open) != 2 || open[0] != "문제" || open[1] != "대안" {
		t.Fatalf("parts left: %v, want 문제 and 대안", open)
	}
}

// A heading spends a slot of the length on a title carrying none of the
// person's words. Spending it on a part that holds nothing spends it twice.
func TestALengthCapSpendsEverySlotItHas(t *testing.T) {
	story := dividedStory()

	fit(&story, 8)

	if len(story.Slides) != 8 {
		t.Fatalf("%d slides for a cap of 8: %v", len(story.Slides), titles(story))
	}
	// Six of the eight are the person's own slides; before, two were headings
	// over nothing and only five were.
	content := 0
	for _, slide := range story.Slides {
		if !slide.Sectioned {
			content++
		}
	}
	if content != 6 {
		t.Fatalf("%d of 8 slides carry the person's thoughts, want 6", content)
	}
	if story.TrimmedSlides != 7 || len(story.Trimmed) != 6 {
		t.Fatalf("reported %d slides / %d thoughts left out, want 7 / 6",
			story.TrimmedSlides, len(story.Trimmed))
	}
}

// A part that keeps one slide keeps its heading. It is thin, but dropping it
// silently extends the part before it, and that division is one nobody
// proposed.
func TestAPartWithOneSlideLeftKeepsItsHeading(t *testing.T) {
	story := dividedStory()

	// Fourteen leaves the last part exactly one of its two slides.
	fit(&story, 14)

	open, empty := openHeadings(story)
	if len(empty) > 0 {
		t.Fatalf("part headings over nothing: %v", empty)
	}
	if len(open) != 3 {
		t.Fatalf("%d parts left, want all 3: %v", len(open), titles(story))
	}
	last := story.Slides[len(story.Slides)-1]
	if last.Sectioned {
		t.Fatalf("the talk ends on a part heading: %v", titles(story))
	}
}

// Part headings are slides too. Someone who asked for fifteen and was handed
// eighteen did not get the length they asked for.
//
// A cap below the length at which a talk is divided at all turns parts off
// instead, which is why this asks for fifteen: short talks do not need parts,
// and the cap runs before sectioning is offered.
func TestPartHeadingsCountAgainstTheCap(t *testing.T) {
	spaces := &fakeSpaces{name: "긴 발표"}
	for i := 0; i < 30; i++ {
		spaces.notes = append(spaces.notes, note(byte(i+1), "생각", float64(i)*2000))
	}
	links := &fakeLinks{}
	sectioner := &fakeSectioner{sections: []Section{
		{Start: 0, Title: "문제"}, {Start: 5, Title: "대안"}, {Start: 10, Title: "결정"},
	}}
	svc := serviceWith(spaces, links, &fakePtium{}, Config{})
	svc.Sectioner = sectioner

	preview, err := svc.Preview(context.Background(), uuid.New(),
		Request{SpaceID: id(9), OneSlidePerThought: true, SectionDeck: true, MaxSlides: 15})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if sectioner.calls == 0 {
		t.Fatal("sectioning never ran, so this proves nothing about headings under a cap")
	}
	if got := len(preview.Storyline.Slides); got != 15 {
		t.Fatalf("%d slides for a cap of 15", got)
	}
	headings := 0
	for _, slide := range preview.Storyline.Slides {
		if slide.Sectioned {
			headings++
		}
	}
	if headings == 0 {
		t.Fatal("no part heading survived, so the cap was not tested against headings")
	}
	if preview.Storyline.Sections != headings {
		t.Fatalf("the deck says it is in %d parts and holds %d headings",
			preview.Storyline.Sections, headings)
	}
}

// The screen says "AI가 3개 부로 나눴습니다". Over a deck the length cut down to
// two, that is the same deck saying two different things — and the count is
// what tells the person how much of the shape is the model's.
func TestTheReportedPartCountIsThePartsInTheDeck(t *testing.T) {
	spaces := &fakeSpaces{name: "긴 발표"}
	for i := 0; i < 30; i++ {
		spaces.notes = append(spaces.notes, note(byte(i+1), "생각", float64(i)*2000))
	}
	// The last part opens two slides from the end of a fifteen-slide deck, so
	// the second pass over the finished deck cannot afford to open it.
	sectioner := &fakeSectioner{sections: []Section{
		{Start: 0, Title: "문제"}, {Start: 5, Title: "대안"}, {Start: 13, Title: "결정"},
	}}
	svc := serviceWith(spaces, &fakeLinks{}, &fakePtium{}, Config{})
	svc.Sectioner = sectioner

	preview, err := svc.Preview(context.Background(), uuid.New(),
		Request{SpaceID: id(9), OneSlidePerThought: true, SectionDeck: true, MaxSlides: 15})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	story := preview.Storyline
	open, empty := openHeadings(story)
	if len(empty) > 0 {
		t.Fatalf("part headings over nothing: %v", empty)
	}
	if len(open) != 2 {
		t.Fatalf("%d parts in the deck, want 2 — the third had no room to open", len(open))
	}
	if story.Sections != 2 {
		t.Fatalf("the deck says it is in %d parts and holds %d", story.Sections, len(open))
	}
	if len(story.Slides) != 15 {
		t.Fatalf("%d slides for a cap of 15", len(story.Slides))
	}
}

// The screen also says "묶음 제목 15개를 AI가 지었습니다". The same second pass
// that can cut a part heading out of the deck can cut a slide a model named,
// and that count is the other half of what tells a person how much of the deck
// is not theirs.
func TestTheReportedHeadingCountIsTheNamedSlidesInTheDeck(t *testing.T) {
	// Pairs: two notes close enough to be one huddle, and far enough from the
	// next pair to be a huddle of their own. Sixteen pairs is sixteen grouped
	// slides, which is more than a cap of fifteen and more than the length at
	// which a talk is divided at all.
	spaces := &fakeSpaces{name: "묶음 많은 공간"}
	for i := 0; i < 16; i++ {
		spaces.notes = append(spaces.notes,
			note(byte(i*2+1), "묶음 메모", float64(i)*1000),
			note(byte(i*2+2), "옆에 붙여 둔 메모", float64(i)*1000+300))
	}
	labels := make([]string, 16)
	for i := range labels {
		labels[i] = "AI가 지은 제목"
	}
	svc := serviceWith(spaces, &fakeLinks{}, &fakePtium{}, Config{})
	svc.Namer = &fakeNamer{labels: labels}
	svc.Sectioner = &fakeSectioner{sections: []Section{
		{Start: 0, Title: "문제"}, {Start: 5, Title: "대안"}, {Start: 10, Title: "결정"},
	}}

	preview, err := svc.Preview(context.Background(), uuid.New(),
		Request{SpaceID: id(9), NameGroups: true, SectionDeck: true, MaxSlides: 15})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	story := preview.Storyline
	if len(story.Slides) != 15 {
		t.Fatalf("%d slides for a cap of 15: %v", len(story.Slides), titles(story))
	}
	named := 0
	for _, slide := range story.Slides {
		if slide.Named {
			named++
		}
	}
	if named == 0 {
		t.Fatal("no named heading survived, so this proves nothing about the count")
	}
	if named == 16 {
		t.Fatal("nothing a model named was cut, so this proves nothing about the count")
	}
	if story.NamedHeadings != named {
		t.Fatalf("the deck says a model named %d headings and holds %d",
			story.NamedHeadings, named)
	}
	// The part count is recounted in the same place, so it must survive the
	// change that fixed the heading count.
	open, empty := openHeadings(story)
	if len(empty) > 0 {
		t.Fatalf("part headings over nothing: %v", empty)
	}
	if story.Sections != len(open) {
		t.Fatalf("the deck says it is in %d parts and holds %d", story.Sections, len(open))
	}
}
