package server

import (
	"strings"
	"testing"
)

func TestDocumentSplitWholeBlockBoundaries(t *testing.T) {
	for _, test := range []struct {
		name, source, selected string
		ok                     bool
	}{
		{"paragraph", "# 제목\n\n유일한 문단 😀\n\n끝 문단\n", "유일한 문단 😀\n", true},
		{"gfm-table", "머리\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n끝\n", "| A | B |\n|---|---|\n| 1 | 2 |\n", true},
		{"complete-fence", "머리\n\n```go\nfmt.Print(\"안녕\")\n```\n\n끝\n", "```go\nfmt.Print(\"안녕\")\n```\n", true},
		{"nested-list", "머리\n\n- 상위\n  - 하위\n- 다음\n\n끝\n", "- 상위\n  - 하위\n- 다음\n", true},
		{"multiline-paragraph-cut", "한 문단\n계속된 줄\n", "계속된 줄\n", false},
		{"half-table", "| A | B |\n|---|---|\n| 1 | 2 |\n", "| 1 | 2 |\n", false},
		{"inside-fence", "```text\n첫 줄\n다음 줄\n```\n", "다음 줄\n", false},
		{"inside-list", "- 첫 항목\n- 다음 항목\n", "- 다음 항목\n", false},
		{"frontmatter", "---\ntags: [팀]\n---\n\n내용\n", "tags: [팀]\n", false},
		{"after-frontmatter", "---\ntags: [팀]\n---\n\n내용\n\n끝\n", "내용\n", true},
		{"html-advanced", "<details>\n<summary>제목</summary>\n본문\n</details>\n", "<details>\n<summary>제목</summary>\n본문\n</details>\n", false},
		{"reference-dependency", "[문서][x]\n\n[x]: https://example.test\n", "[문서][x]\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			start := strings.Index(test.source, test.selected)
			e := validateDocumentSplitRange(test.source, start, start+len(test.selected), test.selected)
			if (e == nil) != test.ok {
				t.Fatal("boundary", e, test.ok)
			}
		})
	}
}
