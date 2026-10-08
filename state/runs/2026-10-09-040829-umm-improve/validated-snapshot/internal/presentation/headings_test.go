package presentation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type fakeNamer struct {
	labels []string
	err    error
	sent   []NameRequest
	calls  int
}

func (f *fakeNamer) NameGroups(_ context.Context, groups []NameRequest, _ string) ([]string, error) {
	f.calls++
	f.sent = append(f.sent, groups...)
	return f.labels, f.err
}

func groupedStory(t *testing.T, count int) Storyline {
	t.Helper()
	thoughts := make([]Thought, count)
	for i := range thoughts {
		thoughts[i] = at(float64(i%4)*300, float64(i/4)*200, "회고 주기 관련 메모")
	}
	return Compile(thoughts, nil, Options{Title: "회고"})
}

func TestAProposedHeadingReplacesTheDerivedOne(t *testing.T) {
	story := groupedStory(t, 4)
	if len(story.Slides) != 1 || !story.Slides[0].Grouped {
		t.Fatalf("expected one grouped slide, got %d", len(story.Slides))
	}
	derived := story.Slides[0].Title

	namer := &fakeNamer{labels: []string{"회고 주기 단축"}}
	if applied := nameGroups(context.Background(), namer, &story, "ko"); applied != 1 {
		t.Fatalf("applied %d headings", applied)
	}
	if story.Slides[0].Title != "회고 주기 단축" {
		t.Fatalf("heading = %q, want the proposed one", story.Slides[0].Title)
	}
	if story.Slides[0].Title == derived {
		t.Fatal("the heading did not change, so this proves nothing")
	}
	// The screen has to be able to say the heading is not the person's.
	if !story.Slides[0].Named {
		t.Fatal("a proposed heading was not marked as one")
	}
}

// The words on the slides are the person's whether or not a model is involved.
// This is the line the whole feature sits behind.
func TestNamingChangesOnlyTheHeading(t *testing.T) {
	story := groupedStory(t, 5)
	before := story.Slides[0]
	namer := &fakeNamer{labels: []string{"완전히 다른 제목"}}
	nameGroups(context.Background(), namer, &story, "ko")
	after := story.Slides[0]

	if after.Lead != before.Lead {
		t.Fatalf("the sentence under the heading changed: %q -> %q", before.Lead, after.Lead)
	}
	if len(after.Points) != len(before.Points) {
		t.Fatalf("the points changed: %d -> %d", len(before.Points), len(after.Points))
	}
	for i := range after.Points {
		if after.Points[i].Text != before.Points[i].Text {
			t.Fatalf("point %d was rewritten: %q -> %q", i, before.Points[i].Text, after.Points[i].Text)
		}
	}
	if len(after.From) != len(before.From) {
		t.Fatal("the thoughts on the slide changed")
	}
}

// A slide the person's own words already head is not for a model to rename.
func TestOnlyGroupedSlidesAreNamed(t *testing.T) {
	lone := at(0, 0, "혼자 있는 생각")
	far := at(9000, 0, "멀리 있는 생각")
	story := Compile([]Thought{lone, far}, nil, Options{Title: "x"})
	for _, slide := range story.Slides {
		if slide.Grouped {
			t.Fatal("a slide with nothing near it was marked grouped")
		}
	}
	namer := &fakeNamer{labels: []string{"제안", "제안"}}
	if applied := nameGroups(context.Background(), namer, &story, "ko"); applied != 0 {
		t.Fatalf("%d headings were replaced on slides that already had the person's own", applied)
	}
	if namer.calls != 0 {
		t.Fatal("the model was called although there was nothing to name")
	}
}

// Polish that breaks must not break the thing it was polishing.
func TestADeckStillCompilesWhenNamingFails(t *testing.T) {
	story := groupedStory(t, 4)
	derived := story.Slides[0].Title
	namer := &fakeNamer{err: errors.New("gateway is down")}
	if applied := nameGroups(context.Background(), namer, &story, "ko"); applied != 0 {
		t.Fatalf("applied %d", applied)
	}
	if story.Slides[0].Title != derived {
		t.Fatalf("a failed call changed the heading to %q", story.Slides[0].Title)
	}
	if story.Slides[0].Named {
		t.Fatal("a heading umm derived was marked as proposed")
	}
}

func TestNoNamerLeavesTheDeckAlone(t *testing.T) {
	story := groupedStory(t, 4)
	derived := story.Slides[0].Title
	if applied := nameGroups(context.Background(), nil, &story, "ko"); applied != 0 {
		t.Fatalf("applied %d with no namer", applied)
	}
	if story.Slides[0].Title != derived {
		t.Fatal("the heading changed with no namer")
	}
}

// A model asked for a short phrase answers with all of these sooner or later,
// and none of them is a heading.
func TestUnusableAnswersAreRefused(t *testing.T) {
	for _, answer := range []string{
		"",
		"   ",
		"이 묶음은 회고 주기를 격주로 줄이는 것에 관한 것으로 보이며 여러 관점이 담겨 있습니다",
		"첫째 줄\n둘째 줄",
		"- 회고 주기",
	} {
		if got := usableHeading(answer); got != "" && strings.ContainsAny(got, "\n") {
			t.Fatalf("usableHeading(%q) = %q", answer, got)
		}
	}
	if got := usableHeading("이 묶음은 회고 주기를 격주로 줄이는 것에 관한 것으로 보이며 여러 관점이 담겨 있습니다"); got != "" {
		t.Fatalf("a sentence was accepted as a heading: %q", got)
	}
	if got := usableHeading("첫째 줄\n둘째 줄"); got != "" {
		t.Fatalf("a multi-line answer was accepted: %q", got)
	}
	// Quoting and list markers are formatting, not the label.
	if got := usableHeading(`"회고 주기 단축"`); got != "회고 주기 단축" {
		t.Fatalf("a quoted label came back as %q", got)
	}
	if got := usableHeading("- 회고 주기 단축"); got != "회고 주기 단축" {
		t.Fatalf("a list marker was kept: %q", got)
	}
}

// An answer that cannot be read is nothing, never a guess. A heading assembled
// from a reply nobody understood is the kind of thing that is only noticed on a
// screen in front of an audience.
func TestAnAnswerThatIsNotTheAskedForShapeIsIgnored(t *testing.T) {
	for _, reply := range []string{"", "회고 주기 단축", "{\"labels\":[\"가\"]}", "[not json"} {
		if got := decodeLabels(reply); len(got) != 0 {
			t.Fatalf("decodeLabels(%q) = %v", reply, got)
		}
	}
	if got := decodeLabels("여기 있습니다: [\"가\", \"나\"]"); len(got) != 2 || got[0] != "가" {
		t.Fatalf("a JSON array with chatter around it was not read: %v", got)
	}
}

// Fewer labels than groups is the common failure, and the groups that got none
// must keep the heading they had.
func TestGroupsWithNoLabelKeepTheirDerivedHeading(t *testing.T) {
	thoughts := []Thought{
		at(0, 0, "가"), at(300, 0, "나"),
		at(9000, 0, "다"), at(9300, 0, "라"),
	}
	story := Compile(thoughts, nil, Options{Title: "x"})
	if len(story.Slides) != 2 {
		t.Fatalf("%d slides", len(story.Slides))
	}
	second := story.Slides[1].Title
	namer := &fakeNamer{labels: []string{"첫 묶음"}}
	if applied := nameGroups(context.Background(), namer, &story, "ko"); applied != 1 {
		t.Fatalf("applied %d", applied)
	}
	if story.Slides[1].Title != second {
		t.Fatalf("a group with no label had its heading changed to %q", story.Slides[1].Title)
	}
	if story.Slides[1].Named {
		t.Fatal("a group with no label was marked as named")
	}
}

// One call for the whole deck, and a bounded one: a deck of hundreds of groups
// must not put the entire space in a single prompt.
func TestNamingIsOneBoundedCall(t *testing.T) {
	story := Storyline{}
	for i := 0; i < maxNamedGroups*3; i++ {
		story.Slides = append(story.Slides, Slide{Grouped: true, Title: "제목", Lead: "본문", From: []uuid.UUID{uuid.New()}})
	}
	namer := &fakeNamer{}
	nameGroups(context.Background(), namer, &story, "ko")
	if namer.calls != 1 {
		t.Fatalf("%d calls, want one", namer.calls)
	}
	if len(namer.sent) > maxNamedGroups {
		t.Fatalf("%d groups were sent, more than the %d bound", len(namer.sent), maxNamedGroups)
	}
}

// Each group sends the gist, not the huddle.
func TestAGroupSendsABoundedExtract(t *testing.T) {
	slide := Slide{Grouped: true, Title: "머리", Lead: strings.Repeat("가", 400)}
	for i := 0; i < 30; i++ {
		slide.Points = append(slide.Points, Point{Text: strings.Repeat("나", 400)})
	}
	extract := groupExtract(slide)
	if len(extract) > maxThoughtsPerNameRequest {
		t.Fatalf("%d thoughts sent, more than the %d bound", len(extract), maxThoughtsPerNameRequest)
	}
	for _, text := range extract {
		if len([]rune(text)) > maxNameRequestRunes {
			t.Fatalf("a thought of %d runes was sent", len([]rune(text)))
		}
	}
}
