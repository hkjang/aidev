package server

import (
	"slices"
	"testing"
)

func TestMarkdownIndexUsesCommonMarkStructure(t *testing.T) {
	md := "---\ntags: [서버, 운영]\nexample: '[[yaml 아님]]'\n---\n# 지식 연결\n\n[[실제 문서|이름]]과 [[다른 문서#제목]] #플랫폼\n\n`[[인라인 아님]] #숨김`\n\n```markdown\n[[펜스 아님]]\n- [ ] 가짜 할 일\n```\n\n    [[들여쓰기 코드 아님]]\n\n- [ ] 첫 할 일\n  - [x] 중첩 할 일 [[중첩 링크]]\n\n\\[[이스케이프 아님]]\n\n<div>[[HTML 아님]]</div>\n"
	index := indexMarkdown(md)
	if !slices.Equal(index.Links, []string{"실제 문서", "다른 문서", "중첩 링크"}) {
		t.Fatalf("links %#v", index.Links)
	}
	if !slices.Equal(index.Tags, []string{"서버", "운영", "플랫폼"}) {
		t.Fatalf("tags %#v", index.Tags)
	}
	if len(index.Headings) != 1 || index.Headings[0].Text != "지식 연결" || index.Headings[0].Level != 1 {
		t.Fatalf("outline %#v", index.Headings)
	}
	if len(index.Tasks) != 2 || index.Tasks[0].Done || !index.Tasks[1].Done || index.Tasks[0].Line != 17 || index.Tasks[1].Line != 18 {
		t.Fatalf("tasks %#v", index.Tasks)
	}
}

func TestMarkdownTaskLabelIgnoresCheckboxState(t *testing.T) {
	for _, prefix := range []string{"- ", "* ", "1. ", "  - "} {
		before, after := indexMarkdown(prefix+"[ ] 확인"), indexMarkdown(prefix+"[x] 확인")
		if len(before.Tasks) != 1 || len(after.Tasks) != 1 || before.Tasks[0].Text != "확인" || before.Tasks[0].Text != after.Tasks[0].Text || before.Tasks[0].Done || !after.Tasks[0].Done {
			t.Fatalf("%q: %#v -> %#v", prefix, before.Tasks, after.Tasks)
		}
	}
}
