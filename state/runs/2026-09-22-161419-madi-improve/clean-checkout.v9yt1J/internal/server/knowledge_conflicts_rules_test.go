package server

import (
	"context"
	"strings"
	"testing"
)

func TestKnowledgeConflictLiteralRules(t *testing.T) {
	sources := []conflictSource{{ID: "a", Version: 1, Title: "운영 정책", Markdown: "# 토큰 정책\n보존 기간은 30일입니다.\n외부 발급은 허용합니다.\n검토 절차는 필수입니다.\n", Tags: []string{"운영"}}, {ID: "b", Version: 2, Title: "다른 제목", Markdown: "# 토큰 정책\n보존 기간은 90일입니다.\n외부 발급은 금지합니다.\n검토 절차는 선택입니다.\n", Tags: []string{"운영"}}}
	out, stats, e := detectKnowledgeConflicts(t.Context(), sources)
	if e != nil || len(out) != 3 || stats.Documents != 2 {
		t.Fatal(len(out), stats, e)
	}
	for _, pair := range out {
		if pair.Left.Excerpt != sources[0].Markdown[pair.Left.Start:pair.Left.End] || pair.Right.Excerpt != sources[1].Markdown[pair.Right.Start:pair.Right.End] || pair.Left.Line < 2 || pair.Right.Version != 2 {
			t.Fatal("original offsets/version lost", pair)
		}
	}
	sources[1].Markdown = "# 토큰 정책\n보존 기간은 30 일입니다.\n외부 발급은 허용합니다.\n검토 절차는 필수입니다.\n"
	out, _, e = detectKnowledgeConflicts(t.Context(), sources)
	if e != nil || len(out) != 0 {
		t.Fatal("equivalent lexical values not normalized", out, e)
	}
	sources[1].Markdown = "# 다른 주제\n보존 기간은 90일입니다.\n"
	sources[1].Tags = []string{"인사"}
	out, _, e = detectKnowledgeConflicts(t.Context(), sources)
	if e != nil || len(out) != 0 {
		t.Fatal("unrelated topic matched", out, e)
	}
	sources[1].Tags = []string{"운영"}
	out, _, e = detectKnowledgeConflicts(t.Context(), sources)
	if e != nil || len(out) != 1 {
		t.Fatal("explicit common tag not used", out, e)
	}
}
func TestKnowledgeConflictExcludesExamplesAndBounds(t *testing.T) {
	md := "---\nnote: 90일\n---\n# 토큰 정책\n```text\n보존 기간은 90일입니다.\n```\n> 보존 기간은 90일입니다.\n\n    보존 기간은 90일입니다.\n\n예시: 보존 기간은 90일입니다.\n\n\"보존 기간은 90일입니다.\"\n\n보존 기간은 `90일`입니다.\n\n<div>보존 기간은 90일입니다.</div>\n"
	a := conflictSource{ID: "a", Title: "토큰 정책", Markdown: "보존 기간은 30일입니다.\n"}
	b := conflictSource{ID: "b", Title: "토큰 정책", Markdown: md}
	out, _, e := detectKnowledgeConflicts(t.Context(), []conflictSource{a, b})
	if e != nil || len(out) != 0 {
		t.Fatal("example leaked into rule matches", out, e)
	}
	b.Markdown = strings.Repeat("x", 1<<20+1)
	if _, _, e = detectKnowledgeConflicts(t.Context(), []conflictSource{a, b}); e == nil {
		t.Fatal("oversize source accepted")
	}
	b.Markdown = strings.Repeat("본문\n", 2100)
	_, stats, e := detectKnowledgeConflicts(t.Context(), []conflictSource{a, b})
	if e != nil || !stats.Truncated {
		t.Fatal("line scan bound not disclosed", stats, e)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, e = detectKnowledgeConflicts(ctx, []conflictSource{a, b}); e == nil {
		t.Fatal("cancellation ignored")
	}
}
