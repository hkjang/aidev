package presentation

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// Fixed ids, so a failure names the thought rather than a random uuid.
func id(n byte) uuid.UUID {
	var u uuid.UUID
	u[0] = n
	u[15] = n
	return u
}

func thought(n byte, content string, x, y float64) Thought {
	return Thought{ID: id(n), Content: content, X: x, Y: y}
}

func link(from, to byte, relation store.Relation) Link {
	return Link{From: id(from), To: id(to), Relation: relation, Origin: store.OriginManual}
}

// titles is what the deck says, in order — the thing every test is really about.
func titles(s Storyline) []string {
	out := make([]string, 0, len(s.Slides))
	for _, slide := range s.Slides {
		out = append(out, slide.Title)
	}
	return out
}

func equal(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d slides %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("slide %d: got %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

func TestCompileEmpty(t *testing.T) {
	story := Compile(nil, nil, Options{Title: "빈 공간"})
	if len(story.Slides) != 0 {
		t.Fatalf("expected no slides, got %d", len(story.Slides))
	}
	if story.Title != "빈 공간" {
		t.Fatalf("title lost: %q", story.Title)
	}
}

// The differentiator: a sequence someone drew on the canvas is the running
// order of the talk, whatever order the notes were created or placed in.
func TestFollowsChainSetsTheOrder(t *testing.T) {
	thoughts := []Thought{
		// Placed deliberately against the intended order, so only the chain can
		// produce the right answer.
		thought(4, "추천안", 0, 0),
		thought(1, "문제", 3000, 0),
		thought(3, "대안", 1000, 0),
		thought(2, "원인", 2000, 0),
	}
	links := []Link{
		link(1, 2, store.RelationFollows),
		link(2, 3, store.RelationFollows),
		link(3, 4, store.RelationFollows),
	}
	story := Compile(thoughts, links, Options{})
	equal(t, titles(story), []string{"문제", "원인", "대안", "추천안"})
}

func TestFollowsChainSurvivesACycle(t *testing.T) {
	thoughts := []Thought{thought(1, "하나", 0, 0), thought(2, "둘", 400, 0), thought(3, "셋", 800, 0)}
	// Every thought has a predecessor, so none of them is the head of a chain.
	links := []Link{
		link(1, 2, store.RelationFollows),
		link(2, 3, store.RelationFollows),
		link(3, 1, store.RelationFollows),
	}
	story := Compile(thoughts, links, Options{})
	// A talk still comes out, and every thought is in it exactly once. Which
	// one leads is arbitrary in a cycle; losing one would not be.
	if len(story.Slides) != 3 {
		t.Fatalf("a cycle dropped thoughts: %v", titles(story))
	}
}

// Without a stated sequence, a canvas is read the way it is laid out.
func TestPlacementOrdersWhatHasNoSequence(t *testing.T) {
	thoughts := []Thought{
		thought(3, "오른쪽", 2000, 0),
		thought(1, "왼쪽 위", 0, 0),
		thought(2, "왼쪽 아래", 40, 500),
	}
	story := Compile(thoughts, nil, Options{})
	// Within a column, top to bottom; then across. 40px apart is the same column.
	equal(t, titles(story), []string{"왼쪽 위", "왼쪽 아래", "오른쪽"})
}

func TestOrderIsTheSameEveryRun(t *testing.T) {
	thoughts := []Thought{
		thought(1, "가", 0, 0), thought(2, "나", 0, 0),
		thought(3, "다", 0, 0), thought(4, "라", 0, 0),
	}
	first := titles(Compile(thoughts, nil, Options{}))
	for i := 0; i < 20; i++ {
		equal(t, titles(Compile(thoughts, nil, Options{})), first)
	}
}

func TestSupportBecomesBulletsNotSlides(t *testing.T) {
	thoughts := []Thought{
		thought(1, "격주 회고로 바꾸자", 0, 0),
		thought(2, "주기가 짧으면 논의가 얕아진다", 0, 300),
		thought(3, "지난 분기 회고 3회가 취소됐다", 0, 600),
	}
	links := []Link{link(2, 1, store.RelationSupports), link(3, 1, store.RelationSupports)}
	story := Compile(thoughts, links, Options{})

	equal(t, titles(story), []string{"격주 회고로 바꾸자"})
	if len(story.Slides[0].Points) != 2 {
		t.Fatalf("expected the two supports as points, got %d", len(story.Slides[0].Points))
	}
	// And the words are the person's, not a rewrite.
	if story.Slides[0].Points[0].Text != "주기가 짧으면 논의가 얕아진다" {
		t.Fatalf("a support was reworded: %q", story.Slides[0].Points[0].Text)
	}
}

func TestSupportOfASupportNestsOneLevel(t *testing.T) {
	thoughts := []Thought{
		thought(1, "주장", 0, 0), thought(2, "근거", 0, 300), thought(3, "근거의 근거", 0, 600),
	}
	links := []Link{link(2, 1, store.RelationSupports), link(3, 2, store.RelationSupports)}
	story := Compile(thoughts, links, Options{})

	equal(t, titles(story), []string{"주장"})
	points := story.Slides[0].Points
	if len(points) != 2 || points[0].Depth != 0 || points[1].Depth != 1 {
		t.Fatalf("expected one nested level, got %+v", points)
	}
}

// The slide only umm can produce: a disagreement the person recorded instead of
// resolving it by deleting one side.
func TestContradictionBecomesAComparison(t *testing.T) {
	thoughts := []Thought{
		thought(1, "격주가 낫다", 0, 0),
		thought(2, "주 1회를 지켜야 한다", 400, 0),
	}
	links := []Link{link(1, 2, store.RelationContradicts)}
	story := Compile(thoughts, links, Options{})

	if len(story.Slides) != 1 {
		t.Fatalf("both sides should share one slide, got %v", titles(story))
	}
	if story.Slides[0].Role != RoleComparison {
		t.Fatalf("got role %q, want comparison", story.Slides[0].Role)
	}
	if len(story.Slides[0].From) != 2 {
		t.Fatalf("a comparison must record both sides, got %v", story.Slides[0].From)
	}
}

func TestAQuestionOpensASection(t *testing.T) {
	q := Thought{ID: id(1), Content: "격주 회고가 정말 더 나은가?", Kind: "question"}
	a := thought(2, "지난 분기에 3회가 취소됐다", 0, 300)
	story := Compile([]Thought{q, a}, []Link{link(2, 1, store.RelationAnswers)}, Options{})

	if len(story.Slides) != 1 {
		t.Fatalf("the answer belongs to the question's slide, got %v", titles(story))
	}
	if story.Slides[0].Role != RoleSection {
		t.Fatalf("a marked question should open a section, got %q", story.Slides[0].Role)
	}
	if len(story.Slides[0].Points) != 1 {
		t.Fatalf("the answer should be on the slide, got %+v", story.Slides[0].Points)
	}
}

// A thought held back from Dream is being held back from having things done to
// it, and being put on a slide is one of those things.
func TestExcludedThoughtsAreLeftOutAndSaidSo(t *testing.T) {
	kept := thought(1, "쓸 생각", 0, 0)
	held := Thought{ID: id(2), Content: "개인적인 메모", AIExcluded: true}
	story := Compile([]Thought{kept, held}, nil, Options{})

	equal(t, titles(story), []string{"쓸 생각"})
	if len(story.Excluded) != 1 || story.Excluded[0] != id(2) {
		t.Fatalf("the held-back thought must be reported, got %v", story.Excluded)
	}
}

func TestExcludedCanBeOverridden(t *testing.T) {
	held := Thought{ID: id(1), Content: "개인적인 메모", AIExcluded: true}
	story := Compile([]Thought{held}, nil, Options{IncludeExcluded: true})
	equal(t, titles(story), []string{"개인적인 메모"})
	if len(story.Excluded) != 0 {
		t.Fatalf("nothing was left out, but %v was reported", story.Excluded)
	}
}

func TestOnlyRestrictsToASelection(t *testing.T) {
	thoughts := []Thought{thought(1, "하나", 0, 0), thought(2, "둘", 400, 0), thought(3, "셋", 800, 0)}
	story := Compile(thoughts, nil, Options{Only: []uuid.UUID{id(1), id(3)}})
	equal(t, titles(story), []string{"하나", "셋"})
	// Not asked for is not the same as left out, so the count stays honest.
	if len(story.Excluded) != 0 {
		t.Fatalf("a thought outside the selection was reported as excluded: %v", story.Excluded)
	}
}

func TestEmptyThoughtsNeverReachASlide(t *testing.T) {
	thoughts := []Thought{thought(1, "", 0, 0), thought(2, "   \n  ", 400, 0), thought(3, "진짜 생각", 800, 0)}
	story := Compile(thoughts, nil, Options{})
	equal(t, titles(story), []string{"진짜 생각"})
	if len(story.Excluded) != 2 {
		t.Fatalf("empty thoughts should be reported, got %v", story.Excluded)
	}
}

// Every thought that is used has to be reachable from the deck, or the trace
// that sends a reader back to their own notes is broken.
func TestEveryUsedThoughtIsRecordedOnItsSlide(t *testing.T) {
	thoughts := []Thought{
		thought(1, "주장", 0, 0), thought(2, "근거", 0, 300),
		thought(3, "반대", 400, 0), thought(4, "따로", 900, 0),
	}
	links := []Link{link(2, 1, store.RelationSupports), link(1, 3, store.RelationContradicts)}
	story := Compile(thoughts, links, Options{})

	seen := map[uuid.UUID]int{}
	for _, slide := range story.Slides {
		for _, from := range slide.From {
			seen[from]++
		}
	}
	for _, th := range thoughts {
		if seen[th.ID] == 0 {
			t.Fatalf("thought %v reached no slide", th.ID)
		}
		if seen[th.ID] > 1 {
			t.Fatalf("thought %v is on %d slides; the deck repeats itself", th.ID, seen[th.ID])
		}
	}
}

func TestTitleComesFromTheTitleWhenThereIsOne(t *testing.T) {
	th := Thought{ID: id(1), Title: "회고 주기", Content: "격주로 줄이면 논의가 깊어질지 확인이 필요하다"}
	story := Compile([]Thought{th}, nil, Options{})
	equal(t, titles(story), []string{"회고 주기"})
	if story.Slides[0].Lead != "격주로 줄이면 논의가 깊어질지 확인이 필요하다" {
		t.Fatalf("the body should become the lead, got %q", story.Slides[0].Lead)
	}
}

func TestLeadIsNotARepeatOfTheTitle(t *testing.T) {
	story := Compile([]Thought{thought(1, "한 줄짜리 생각", 0, 0)}, nil, Options{})
	if story.Slides[0].Lead != "" {
		t.Fatalf("a one-line thought should not say itself twice, got lead %q", story.Slides[0].Lead)
	}
}

func TestLinksToThoughtsOutsideTheSelectionAreIgnored(t *testing.T) {
	thoughts := []Thought{thought(1, "안에 있는 생각", 0, 0)}
	// The other end is not in play; this must not panic or invent a slide.
	links := []Link{link(1, 9, store.RelationSupports), link(9, 1, store.RelationContradicts)}
	story := Compile(thoughts, links, Options{})
	equal(t, titles(story), []string{"안에 있는 생각"})
	if story.Slides[0].Role != RoleContent {
		t.Fatalf("a dangling contradiction made a comparison out of nothing")
	}
}

func TestSummaryCountsWhatIsThere(t *testing.T) {
	thoughts := []Thought{thought(1, "하나", 0, 0), Thought{ID: id(2), Content: "빠짐", AIExcluded: true}}
	got := Compile(thoughts, nil, Options{}).Summary()
	if !strings.Contains(got, "1 slides") || !strings.Contains(got, "1 left out") {
		t.Fatalf("summary does not describe the result: %q", got)
	}
}

// The ordering defect the realistic example exposed and no unit test had: any
// chain head used to be hoisted in front of everything else on the canvas, so a
// space whose first slide should have been the question someone wrote opened on
// a thought from the middle of the argument instead.
func TestAChainDoesNotJumpAheadOfWhatIsPlacedBeforeIt(t *testing.T) {
	thoughts := []Thought{
		thought(1, "첫 생각", 0, 0),
		thought(2, "체인 머리", 900, 0),
		thought(3, "체인 꼬리", 1800, 0),
	}
	story := Compile(thoughts, []Link{link(2, 3, store.RelationFollows)}, Options{})
	equal(t, titles(story), []string{"첫 생각", "체인 머리", "체인 꼬리"})
}

// A disagreement is the most informative thing a space says about a thought, so
// neither side may be absorbed into someone else's slide first. Before this, a
// question swallowed its answer and the contradiction disappeared from the deck.
func TestAContradictionSurvivesAQuestionThatWouldAbsorbIt(t *testing.T) {
	question := Thought{ID: id(1), Content: "정말 더 나은가?", Kind: "question"}
	answer := thought(2, "격주가 낫다", 700, 0)
	against := thought(3, "맥락을 잊는다", 1400, 0)

	story := Compile([]Thought{question, answer, against}, []Link{
		link(2, 1, store.RelationAnswers),
		link(2, 3, store.RelationContradicts),
	}, Options{})

	var comparisons int
	for _, slide := range story.Slides {
		if slide.Role == RoleComparison {
			comparisons++
		}
	}
	if comparisons != 1 {
		t.Fatalf("the disagreement was lost: %v", titles(story))
	}
	// And the question still opens the talk rather than being buried.
	if story.Slides[0].Role != RoleSection {
		t.Fatalf("the question should lead, got %v", titles(story))
	}
}

// Evidence used to stop travelling after one step, stranding the support of a
// support on a slide of its own beside the claim it was evidence for.
func TestEvidenceTravelsAllTheWayToItsClaim(t *testing.T) {
	thoughts := []Thought{
		thought(1, "주장", 0, 0),
		thought(2, "근거", 0, 300),
		thought(3, "근거의 근거", 0, 600),
		thought(4, "그 근거의 근거", 0, 900),
	}
	links := []Link{
		link(2, 1, store.RelationSupports),
		link(3, 2, store.RelationSupports),
		link(4, 3, store.RelationSupports),
	}
	story := Compile(thoughts, links, Options{})

	equal(t, titles(story), []string{"주장"})
	if len(story.Slides[0].Points) != 3 {
		t.Fatalf("evidence was stranded: %+v", story.Slides[0].Points)
	}
	// Indent is capped so the slide stays readable, but nothing is left behind.
	for _, p := range story.Slides[0].Points {
		if p.Depth > 1 {
			t.Fatalf("indent past one level: %+v", p)
		}
	}
}

func TestSupportCyclesDoNotHang(t *testing.T) {
	thoughts := []Thought{thought(1, "가", 0, 0), thought(2, "나", 0, 300), thought(3, "다", 0, 600)}
	links := []Link{
		link(2, 1, store.RelationSupports),
		link(3, 2, store.RelationSupports),
		link(1, 3, store.RelationSupports),
	}
	story := Compile(thoughts, links, Options{})
	if len(story.Slides) == 0 {
		t.Fatal("a support cycle produced no talk at all")
	}
}

// A heading is a heading, not the whole thought.
//
// A thought with no title of its own gets its first line as one, and on
// ordinary working notes a first line is often the entire thought — measured at
// 79 characters running across the top of a slide.
func TestAHeadingIsShortEnoughToBeOne(t *testing.T) {
	long := "회고 주기를 격주로 줄이면 논의가 얕아질 수 있다. 대신 주제를 하나로 좁혀서 깊게 들어가는 편이 나을지 다음 스프린트에 시험해 보기로 했다."
	got := headline(Thought{Content: long})
	if len([]rune(got)) > maxHeadlineRunes+1 {
		t.Fatalf("heading is %d runes: %q", len([]rune(got)), got)
	}
	// The first sentence is the best heading there is, because the person wrote
	// it as one unit.
	if want := "회고 주기를 격주로 줄이면 논의가 얕아질 수 있다."; got != want {
		t.Fatalf("heading = %q, want the first sentence %q", got, want)
	}
}

// Nothing is lost by shortening: the heading and the body are separate, so the
// thought still appears in full underneath.
func TestShorteningAHeadingKeepsTheWholeThoughtOnTheSlide(t *testing.T) {
	long := "고객사 A는 온프레미스만 쓰기로 계약서 5조에 명시했고 클라우드 제안은 처음부터 불가능하다"
	thought := Thought{ID: uuid.New(), Content: long}
	story := Compile([]Thought{thought}, nil, Options{Title: "x"})
	if len(story.Slides) != 1 {
		t.Fatalf("%d slides", len(story.Slides))
	}
	slide := story.Slides[0]
	if len([]rune(slide.Title)) > maxHeadlineRunes+1 {
		t.Fatalf("heading is %d runes", len([]rune(slide.Title)))
	}
	if slide.Lead != long {
		t.Fatalf("the thought was shortened away instead of moving under the heading: lead = %q", slide.Lead)
	}
}

// A title the person chose is theirs, however long.
func TestATitleThePersonWroteIsNotShortened(t *testing.T) {
	title := "이번 분기 회고 주기를 어떻게 바꿀지에 대한 팀 전체의 긴 논의 정리본"
	if got := headline(Thought{Title: title, Content: "본문"}); got != title {
		t.Fatalf("the author's own title was cut: %q", got)
	}
}

// A short line is left exactly as it is.
func TestAShortLineIsLeftAlone(t *testing.T) {
	for _, line := range []string{"배포 롤백이 3분 걸린다", "짧음", ""} {
		if got := headline(Thought{Content: line}); got != line {
			t.Fatalf("headline(%q) = %q", line, got)
		}
	}
}

// A long line with no sentence end is cut at a word boundary and marked, so
// nobody reads it as the whole thought.
func TestALongLineWithNoSentenceEndIsCutAtAWord(t *testing.T) {
	got := headline(Thought{Content: "고객사 A는 온프레미스만 쓰기로 계약서 오조에 명시했고 클라우드 제안은 처음부터 불가능하다"})
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("a cut heading does not say it was cut: %q", got)
	}
	if strings.HasSuffix(strings.TrimSuffix(got, "…"), " ") {
		t.Fatalf("heading ends on a space before the mark: %q", got)
	}
	if len([]rune(got)) > maxHeadlineRunes+1 {
		t.Fatalf("heading is %d runes: %q", len([]rune(got)), got)
	}
}

// Why they said two thoughts disagree belongs on the slide that shows the
// disagreement — it is the question an audience actually has there, and it is
// the person's own sentence, so putting it on a slide breaks no rule.
func TestComparisonSlideCarriesWhyTheyDisagree(t *testing.T) {
	const why = "같은 분기 수치를 서로 다르게 읽고 있다"
	thoughts := []Thought{
		{ID: id(1), Content: "회고를 격주로 줄이자", X: 0},
		{ID: id(2), Content: "주기가 짧으면 논의가 얕아진다", X: 400},
	}
	links := []Link{{From: id(2), To: id(1), Relation: store.RelationContradicts,
		Origin: store.OriginManual, Reason: why}}

	story := Compile(thoughts, links, Options{Title: "회고"})
	var comparison *Slide
	for i := range story.Slides {
		if story.Slides[i].Role == RoleComparison {
			comparison = &story.Slides[i]
		}
	}
	if comparison == nil {
		t.Fatalf("no comparison slide: %v", titles(story))
	}
	if comparison.Lead != why {
		t.Fatalf("the comparison slide reads %q, want the author's own %q", comparison.Lead, why)
	}
	// Both sides are still on it, word for word. The first side is the slide's
	// title — repeating it as a point would put the same sentence on twice —
	// so the other side is what has to be among the points.
	if len(comparison.From) != 2 {
		t.Fatalf("the slide no longer quotes both sides: %+v", comparison.From)
	}
	other := false
	for _, point := range comparison.Points {
		if point.Text == "주기가 짧으면 논의가 얕아진다" {
			other = true
		}
	}
	if !other {
		t.Fatalf("the reason displaced a side of the disagreement: %+v", comparison.Points)
	}

	// A disagreement has no direction, so it must not matter which side the
	// compiler reached first.
	reversed := Compile(thoughts, []Link{{From: id(1), To: id(2),
		Relation: store.RelationContradicts, Origin: store.OriginManual, Reason: why}}, Options{Title: "회고"})
	for _, slide := range reversed.Slides {
		if slide.Role == RoleComparison && slide.Lead != why {
			t.Fatalf("drawn the other way round, the reason was lost: %q", slide.Lead)
		}
	}
}

// A connection nobody explained must not make the slide claim one was given.
func TestComparisonSlideSaysNothingWhenNobodyExplained(t *testing.T) {
	story := Compile([]Thought{
		{ID: id(1), Content: "회고를 격주로 줄이자", X: 0},
		{ID: id(2), Content: "주기가 짧으면 논의가 얕아진다", X: 400},
	}, []Link{{From: id(2), To: id(1), Relation: store.RelationContradicts, Origin: store.OriginManual}},
		Options{Title: "회고"})

	for _, slide := range story.Slides {
		if slide.Role == RoleComparison && slide.Lead != "" {
			t.Fatalf("an unexplained disagreement came with the lead %q", slide.Lead)
		}
	}
}
