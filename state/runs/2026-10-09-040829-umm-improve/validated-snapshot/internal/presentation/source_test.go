package presentation

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

func compileSource(t *testing.T, thoughts []Thought, links []Link, opts Options, sopts SourceOptions) string {
	t.Helper()
	return WriteSource(Compile(thoughts, links, opts), sopts)
}

func mustContain(t *testing.T, source, want string) {
	t.Helper()
	if !strings.Contains(source, want) {
		t.Fatalf("source does not contain %q:\n%s", want, source)
	}
}

func TestSourceOpensWithTheSpaceTitle(t *testing.T) {
	source := compileSource(t, []Thought{thought(1, "첫 생각", 0, 0)}, nil, Options{Title: "회고 개선"}, SourceOptions{})
	if !strings.HasPrefix(source, "# 회고 개선\n@cover\n") {
		t.Fatalf("expected a cover slide first:\n%s", source)
	}
}

func TestSourceWithoutATitleHasNoCover(t *testing.T) {
	source := compileSource(t, []Thought{thought(1, "첫 생각", 0, 0)}, nil, Options{}, SourceOptions{})
	if strings.Contains(source, "@cover") {
		t.Fatalf("a talk with no title should not invent a cover:\n%s", source)
	}
}

// The promise of the whole package: what the person wrote is what the slide
// says. If this ever fails, the deck has started paraphrasing someone.
func TestThoughtsReachTheSlideWordForWord(t *testing.T) {
	const written = "온보딩 문서를 다시 쓸지 아직 모르겠다"
	thoughts := []Thought{thought(1, "주장", 0, 0), thought(2, written, 0, 300)}
	source := compileSource(t, thoughts, []Link{link(2, 1, store.RelationSupports)}, Options{}, SourceOptions{})
	mustContain(t, source, "- "+written)
}

func TestRolesBecomeSlideKinds(t *testing.T) {
	q := Thought{ID: id(1), Content: "정말 더 나은가?", Kind: "question"}
	a := thought(2, "가", 400, 0)
	b := thought(3, "나", 800, 0)
	source := compileSource(t, []Thought{q, a, b}, []Link{link(2, 3, store.RelationContradicts)}, Options{}, SourceOptions{})
	mustContain(t, source, "@section")
	mustContain(t, source, "@comparison")
}

func TestNestingIsTwoSpacesPerLevel(t *testing.T) {
	thoughts := []Thought{thought(1, "주장", 0, 0), thought(2, "근거", 0, 300), thought(3, "근거의 근거", 0, 600)}
	links := []Link{link(2, 1, store.RelationSupports), link(3, 2, store.RelationSupports)}
	source := compileSource(t, thoughts, links, Options{}, SourceOptions{})
	mustContain(t, source, "\n- 근거\n")
	mustContain(t, source, "\n  - 근거의 근거\n")
}

// A thought written over several lines is several lines. Joining them into a
// paragraph would be rewriting it.
func TestAMultiLineThoughtKeepsItsLines(t *testing.T) {
	thoughts := []Thought{thought(1, "주장", 0, 0), thought(2, "첫 줄\n둘째 줄", 0, 300)}
	source := compileSource(t, thoughts, []Link{link(2, 1, store.RelationSupports)}, Options{}, SourceOptions{})
	mustContain(t, source, "- 첫 줄")
	mustContain(t, source, "  - 둘째 줄")
}

// Ptium reads a leading marker as markup, so a thought that starts with one has
// to be escaped or it silently turns into a different slide.
func TestLinesThatWouldBeReadAsMarkupAreEscaped(t *testing.T) {
	for _, written := range []string{"# 진짜 제목이 아님", "- 목록처럼 보임", "> 인용처럼 보임", "@cover 처럼 보임", "// 주석처럼 보임", "::kpi 처럼 보임", "!notes 처럼 보임"} {
		thoughts := []Thought{thought(1, "주장", 0, 0), thought(2, written, 0, 300)}
		source := compileSource(t, thoughts, []Link{link(2, 1, store.RelationSupports)}, Options{}, SourceOptions{})
		mustContain(t, source, `- \`+written)
	}
}

// And only a leading marker. A dash mid-sentence is punctuation, and escaping
// it would put a backslash into someone's words.
func TestPunctuationInsideASentenceIsLeftAlone(t *testing.T) {
	const written = "비용 - 편익 분석은 # 태그와 무관하다"
	thoughts := []Thought{thought(1, "주장", 0, 0), thought(2, written, 0, 300)}
	source := compileSource(t, thoughts, []Link{link(2, 1, store.RelationSupports)}, Options{}, SourceOptions{})
	mustContain(t, source, "- "+written)
	if strings.Contains(source, `\`) {
		t.Fatalf("a backslash was put into the author's sentence:\n%s", source)
	}
}

func TestTraceIsOffUnlessAskedFor(t *testing.T) {
	source := compileSource(t, []Thought{thought(1, "생각", 0, 0)}, nil, Options{}, SourceOptions{})
	if strings.Contains(source, TracePrefix) {
		t.Fatalf("trace comments appeared without being asked for:\n%s", source)
	}
}

func TestTraceNamesTheThoughtsASlideCameFrom(t *testing.T) {
	thoughts := []Thought{thought(1, "주장", 0, 0), thought(2, "근거", 0, 300)}
	source := compileSource(t, thoughts, []Link{link(2, 1, store.RelationSupports)}, Options{}, SourceOptions{Trace: true})

	var found []uuid.UUID
	for _, line := range strings.Split(source, "\n") {
		if ids := ParseTrace(line); ids != nil {
			found = ids
		}
	}
	if len(found) != 2 || found[0] != id(1) || found[1] != id(2) {
		t.Fatalf("trace does not name both thoughts, got %v", found)
	}
}

func TestParseTraceIgnoresAComentSomeoneWrote(t *testing.T) {
	for _, line := range []string{"// 나중에 고칠 것", "// umm: 아님", "- 주석이 아님", ""} {
		if ids := ParseTrace(line); ids != nil {
			t.Fatalf("%q was read as a trace: %v", line, ids)
		}
	}
}

func TestTraceSurvivesTheRoundTrip(t *testing.T) {
	ids := []uuid.UUID{id(7), id(9)}
	got := ParseTrace("// " + TraceComment(ids))
	if len(got) != 2 || got[0] != id(7) || got[1] != id(9) {
		t.Fatalf("round trip lost ids: %v", got)
	}
}

func TestEmptyStorylineWritesNothing(t *testing.T) {
	if source := WriteSource(Storyline{}, SourceOptions{}); source != "" {
		t.Fatalf("expected empty source, got %q", source)
	}
}

// A slide with no title and no words cannot be rendered, and emitting a bare
// `#` would make Ptium compile a blank slide rather than skip it.
func TestASlideWithNothingToSayIsNotWritten(t *testing.T) {
	story := Storyline{Slides: []Slide{{Role: RoleContent}, {Role: RoleContent, Title: "진짜"}}}
	source := WriteSource(story, SourceOptions{})
	if strings.Count(source, "# ") != 1 {
		t.Fatalf("expected exactly one slide:\n%s", source)
	}
}

// The compiler's output has to be something Ptium can actually read back: every
// line is either a marker it documents or prose.
func TestEveryLineIsValidDeckSource(t *testing.T) {
	q := Thought{ID: id(1), Content: "정말 더 나은가?", Kind: "question"}
	thoughts := []Thought{
		q,
		thought(2, "격주가 낫다", 400, 0),
		thought(3, "주 1회를 지켜야 한다", 800, 0),
		thought(4, "지난 분기 3회 취소", 400, 300),
		thought(5, "# 마커로 시작하는 생각", 1200, 0),
	}
	links := []Link{
		link(4, 2, store.RelationSupports),
		link(2, 3, store.RelationContradicts),
		link(2, 1, store.RelationAnswers),
	}
	source := compileSource(t, thoughts, links, Options{Title: "회고"}, SourceOptions{Trace: true})

	for _, line := range strings.Split(source, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		trimmed := strings.TrimLeft(line, " ")
		if (len(line)-len(trimmed))%2 != 0 {
			t.Fatalf("indent is not a multiple of two: %q", line)
		}
		switch {
		case strings.HasPrefix(trimmed, "# "),
			strings.HasPrefix(trimmed, "@"),
			strings.HasPrefix(trimmed, "> "),
			strings.HasPrefix(trimmed, "- "),
			strings.HasPrefix(trimmed, "// "):
		default:
			t.Fatalf("line is neither a documented marker nor indented prose: %q\n%s", line, source)
		}
	}
}

// The invariant the preview screen depends on: what a person reads in the
// preview is what the deck will say.
//
// The preview renders Storyline.Points directly while Ptium renders the source,
// so the two can drift. They did: the emitter dropped a point identical to its
// slide's title and the storyline kept it, so a comparison slide showed the
// same sentence twice on screen and once in the deck. A preview that shows a
// line the deck will not is the one thing this feature must never do.
func TestThePreviewSaysWhatTheDeckWillSay(t *testing.T) {
	q := Thought{ID: id(1), Content: "정말 더 나은가?", Kind: "question"}
	thoughts := []Thought{
		q,
		thought(2, "격주가 낫다", 700, 0),
		thought(3, "맥락을 잊는다", 1400, 0),
		thought(4, "지난 분기 3회 취소", 700, 300),
		thought(5, "다음 스프린트에 시험한다", 2100, 0),
	}
	links := []Link{
		link(2, 1, store.RelationAnswers),
		link(4, 2, store.RelationSupports),
		link(2, 3, store.RelationContradicts),
		link(3, 5, store.RelationFollows),
	}
	story := Compile(thoughts, links, Options{Title: "회고"})
	source := WriteSource(story, SourceOptions{})

	// Every bullet the deck will show, in order.
	var inDeck []string
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, "- "); ok {
			inDeck = append(inDeck, strings.TrimPrefix(after, `\`))
		}
	}
	// Every bullet the preview will show, in the same order.
	var inPreview []string
	for _, slide := range story.Slides {
		for _, point := range slide.Points {
			for _, line := range splitLines(point.Text) {
				inPreview = append(inPreview, line)
			}
		}
	}

	if len(inDeck) != len(inPreview) {
		t.Fatalf("the preview shows %d lines and the deck %d:\npreview %q\ndeck    %q", len(inPreview), len(inDeck), inPreview, inDeck)
	}
	for i := range inDeck {
		if inDeck[i] != inPreview[i] {
			t.Fatalf("line %d differs: preview %q, deck %q", i, inPreview[i], inDeck[i])
		}
	}
}
