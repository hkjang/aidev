package presentation

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// The point of this format is that it needs no model: the order and the nesting
// are things the person expressed by drawing, and the words are theirs. So what
// is checked is that nothing was invented and nothing was rearranged.

func TestOutlineUsesTheirSentencesAndNothingElse(t *testing.T) {
	thoughts := []Thought{
		{ID: id(1), Content: "회고를 격주로 줄이자", X: 0},
		{ID: id(2), Content: "지난 두 분기 회고가 같은 안건을 다시 올렸다", X: 400},
	}
	links := []Link{link(2, 1, store.RelationSupports)}

	outline := WriteOutline(Compile(thoughts, links, Options{Title: "회고 주기"}))

	for _, sentence := range []string{
		"# 회고 주기",
		"회고를 격주로 줄이자",
		"지난 두 분기 회고가 같은 안건을 다시 올렸다",
	} {
		if !strings.Contains(outline, sentence) {
			t.Fatalf("the outline lost %q:\n%s", sentence, outline)
		}
	}
	// Deck source markers have no business in a document.
	for _, marker := range []string{"@cover", "@kind", TracePrefix} {
		if strings.Contains(outline, marker) {
			t.Fatalf("the outline carries deck source %q:\n%s", marker, outline)
		}
	}
}

// The order is a follows chain — somebody stating a sequence outright — and a
// document that reorders it is disagreeing with what they said.
func TestOutlineKeepsTheOrderTheyStated(t *testing.T) {
	thoughts := []Thought{
		{ID: id(1), Content: "셋째로 정리한다", X: 0},
		{ID: id(2), Content: "먼저 문제를 말한다", X: 900},
		{ID: id(3), Content: "다음으로 대안을 놓는다", X: 1800},
	}
	// Placement says one order; the chain says another, and the chain wins.
	links := []Link{link(2, 3, store.RelationFollows), link(3, 1, store.RelationFollows)}

	outline := WriteOutline(Compile(thoughts, links, Options{Title: "순서"}))
	first := strings.Index(outline, "먼저 문제를 말한다")
	second := strings.Index(outline, "다음으로 대안을 놓는다")
	third := strings.Index(outline, "셋째로 정리한다")
	if first < 0 || second < 0 || third < 0 {
		t.Fatalf("a thought is missing:\n%s", outline)
	}
	if !(first < second && second < third) {
		t.Fatalf("the document does not follow the chain they drew:\n%s", outline)
	}
}

// A part heading is what somebody divided the talk with, so it outranks the
// sections inside it. Flattening them would throw away the division.
func TestOutlineNestsPartsAboveSections(t *testing.T) {
	story := Storyline{Title: "긴 글", Slides: []Slide{
		{Role: RoleSection, Sectioned: true, Title: "문제"},
		{Role: RoleContent, Title: "첫 번째 관찰", From: []uuid.UUID{uuid.New()}},
		{Role: RoleSection, Sectioned: true, Title: "대안"},
		{Role: RoleContent, Title: "두 번째 관찰", From: []uuid.UUID{uuid.New()}},
	}}

	// Matched as whole lines. "### 문제" contains "## 문제", so a substring
	// check here would pass with the two levels collapsed into one — which is
	// exactly the mistake this test exists to catch.
	lines := map[string]bool{}
	for _, line := range strings.Split(WriteOutline(story), "\n") {
		lines[line] = true
	}
	outline := WriteOutline(story)
	for _, want := range []string{"## 문제", "### 첫 번째 관찰", "## 대안", "### 두 번째 관찰"} {
		if !lines[want] {
			t.Fatalf("no line is exactly %q:\n%s", want, outline)
		}
	}
	// And the part headings are not sitting at the section level.
	for _, wrong := range []string{"### 문제", "### 대안"} {
		if lines[wrong] {
			t.Fatalf("a part heading dropped to the section level (%q):\n%s", wrong, outline)
		}
	}
	// And a part heading quotes nothing, so it must not sprout bullets.
	if strings.Contains(outline, "## 문제\n\n-") {
		t.Fatalf("a part heading was given content it does not have:\n%s", outline)
	}
}

// The heading is already the thought's own sentence. Repeating it underneath
// puts the same words on the page twice.
func TestOutlineDoesNotRepeatTheHeading(t *testing.T) {
	story := Storyline{Title: "반복", Slides: []Slide{{
		Role: RoleContent, Title: "한 문장", Lead: "한 문장",
		Points: []Point{{Text: "한 문장"}, {Text: "다른 문장"}},
		From:   []uuid.UUID{uuid.New()},
	}}}

	outline := WriteOutline(story)
	if strings.Count(outline, "한 문장") != 1 {
		t.Fatalf("the heading was repeated:\n%s", outline)
	}
	if !strings.Contains(outline, "다른 문장") {
		t.Fatalf("a real point was dropped:\n%s", outline)
	}
}

// Nesting is how far a thought sits from the claim it was attached to, which
// is something the person expressed by connecting them.
func TestOutlineKeepsHowFarAThoughtSits(t *testing.T) {
	story := Storyline{Title: "깊이", Slides: []Slide{{
		Role: RoleContent, Title: "주장",
		Points: []Point{{Text: "가까운 근거", Depth: 0}, {Text: "그 근거의 근거", Depth: 1}},
		From:   []uuid.UUID{uuid.New()},
	}}}

	outline := WriteOutline(story)
	if !strings.Contains(outline, "- 가까운 근거") {
		t.Fatalf("the near point lost its bullet:\n%s", outline)
	}
	if !strings.Contains(outline, "  - 그 근거의 근거") {
		t.Fatalf("the deeper point was flattened:\n%s", outline)
	}
}

// An empty space produces an empty document rather than a title over nothing.
func TestOutlineOfNothingIsNothing(t *testing.T) {
	if got := WriteOutline(Storyline{}); got != "" {
		t.Fatalf("got %q", got)
	}
}
