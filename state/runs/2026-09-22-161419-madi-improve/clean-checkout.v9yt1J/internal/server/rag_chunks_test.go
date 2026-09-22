package server

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRAGChunksPreserveKoreanSourceAndCitationOffsets(t *testing.T) {
	md := "---\ntags: [운영]\nowner: 작성자\n---\n# 첫 절\n\n" + strings.Repeat("한글 지식과 🧩 이모지, source\n", 80) + "\n## 코드 예제\n\n```sql\n" + strings.Repeat("SELECT '원문' FROM information_schema.tables;\n", 80) + "```\n"
	chunks, e := chunkMarkdown(md, 512, 64)
	if e != nil || len(chunks) < 10 {
		t.Fatal(len(chunks), e)
	}
	covered := make([]bool, len(md))
	for i, c := range chunks {
		if c.Index != i || c.End-c.Start > 512 || c.Content != md[c.Start:c.End] || !utf8.ValidString(c.Content) || c.Hash != digest(c.Content) || c.StartLine < 5 || c.EndLine < c.StartLine || c.Heading == "" {
			t.Fatal(c)
		}
		for n := c.Start; n < c.End; n++ {
			covered[n] = true
		}
	}
	for n := strings.Index(md, "# 첫 절"); n < len(md); n++ {
		if !covered[n] {
			t.Fatalf("source byte %d skipped", n)
		}
	}
	if strings.Contains(chunks[0].Content, "owner:") {
		t.Fatal("front matter not excluded")
	}
	if _, e = chunkMarkdown(md, 512, 256); e == nil {
		t.Fatal("invalid overlap accepted")
	}
	if _, e = chunkMarkdown(string([]byte{0xff}), 512, 64); e == nil {
		t.Fatal("invalid UTF8 accepted")
	}
	indented := "    SELECT '들여쓰기 코드';\n"
	chunks, e = chunkMarkdown(indented, 512, 64)
	if e != nil || len(chunks) != 1 || chunks[0].Content != indented {
		t.Fatal("first-line indentation lost", chunks, e)
	}
}
