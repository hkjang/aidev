package extract

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func officeFixture(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	entries["[Content_Types].xml"] = `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`
	for name, value := range entries {
		w, e := z.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = w.Write([]byte(value)); e != nil {
			t.Fatal(e)
		}
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}

func TestOfficeSourcePositionsAndNoExecution(t *testing.T) {
	t.Run("word paragraphs and table", func(t *testing.T) {
		b := officeFixture(t, map[string]string{"word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>안전한 </w:t></w:r><w:r><w:t>문서</w:t><w:tab/><w:t>본문</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>정확한 표 셀</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`, "word/vbaProject.bin": "never execute"})
		r, e := Office(context.Background(), bytes.NewReader(b), int64(len(b)), "docx", DefaultLimits())
		if e != nil {
			t.Fatal(e)
		}
		if len(r.Fragments) != 2 || r.Fragments[0].Text != "안전한 문서\t본문" || r.Fragments[1].Position.Paragraph != 2 || r.Fragments[1].Position.Table != 1 || r.Fragments[1].Position.Column != 1 {
			t.Fatalf("positions: %+v", r)
		}
		if len(r.Warnings) < 2 {
			t.Fatal("missing macro/layout warnings")
		}
	})
	t.Run("presentation uses manifest order not filename order", func(t *testing.T) {
		b := officeFixture(t, map[string]string{"ppt/presentation.xml": `<p:presentation xmlns:p="p" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst><p:sldId id="1" r:id="s2"/><p:sldId id="2" r:id="s1"/></p:sldIdLst></p:presentation>`, "ppt/_rels/presentation.xml.rels": `<Relationships><Relationship Id="s1" Target="slides/slide1.xml"/><Relationship Id="s2" Target="slides/slide2.xml"/><Relationship Id="web" Target="https://127.0.0.1/secret" TargetMode="External"/></Relationships>`, "ppt/slides/slide1.xml": `<slide><p><r><t>두 번째</t></r></p></slide>`, "ppt/slides/slide2.xml": `<slide><p><r><t>첫 번째</t></r></p></slide>`})
		r, e := Office(context.Background(), bytes.NewReader(b), int64(len(b)), "pptx", DefaultLimits())
		if e != nil {
			t.Fatal(e)
		}
		if len(r.Fragments) != 2 || r.Fragments[0].Text != "첫 번째" || r.Fragments[0].Position.Slide != 1 || r.Fragments[1].Position.Slide != 2 {
			t.Fatalf("slides: %+v", r)
		}
	})
	t.Run("spreadsheet shared inline cached formula", func(t *testing.T) {
		b := officeFixture(t, map[string]string{"xl/workbook.xml": `<workbook xmlns:r="r"><sheets><sheet name="계획" r:id="r1"/></sheets></workbook>`, "xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="r1" Target="worksheets/sheet1.xml"/></Relationships>`, "xl/sharedStrings.xml": `<sst><si><r><t>공유 </t></r><r><t>문자열</t></r></si></sst>`, "xl/worksheets/sheet1.xml": `<worksheet><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="inlineStr"><is><t>직접 값</t></is></c><c r="C1"><f>WEBSERVICE("https://127.0.0.1")</f><v>123</v></c><c r="D1" t="b"><v>1</v></c></row></sheetData></worksheet>`})
		r, e := Office(context.Background(), bytes.NewReader(b), int64(len(b)), "xlsx", DefaultLimits())
		if e != nil {
			t.Fatal(e)
		}
		if len(r.Fragments) != 4 || r.Fragments[0].Text != "공유 문자열" || r.Fragments[1].Text != "직접 값" || r.Fragments[2].Text != "123" || r.Fragments[2].Position.Cell != "C1" || r.Fragments[2].Position.Sheet != "계획" || r.Fragments[3].Text != "true" {
			t.Fatalf("cells: %+v", r)
		}
		if len(r.Warnings) < 2 {
			t.Fatal("cached formula disclosure missing")
		}
	})
}

func TestOfficeRejectsUnsafeAndBounded(t *testing.T) {
	for _, test := range []struct{ name, extra, xml string }{
		{"traversal", "../private", `<document/>`},
		{"entity", "word/extra", `<!DOCTYPE x [<!ENTITY raw SYSTEM "file:///etc/passwd">]><document><p><t>&raw;</t></p></document>`},
		{"depth", "word/extra", strings.Repeat("<x>", 130) + strings.Repeat("</x>", 130)},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := officeFixture(t, map[string]string{"word/document.xml": test.xml, test.extra: "x"})
			if _, e := Office(context.Background(), bytes.NewReader(b), int64(len(b)), "docx", DefaultLimits()); e == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
	b := officeFixture(t, map[string]string{"word/document.xml": `<document><p><t>원본 길이를 제한합니다</t></p></document>`})
	limits := DefaultLimits()
	limits.TextBytes = 5
	if _, e := Office(context.Background(), bytes.NewReader(b), int64(len(b)), "docx", limits); !errors.Is(e, ErrLimit) {
		t.Fatalf("text limit: %v", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := Office(ctx, bytes.NewReader(b), int64(len(b)), "docx", DefaultLimits()); !errors.Is(e, context.Canceled) {
		t.Fatalf("cancel: %v", e)
	}
	var symlink bytes.Buffer
	z := zip.NewWriter(&symlink)
	h := &zip.FileHeader{Name: "word/link"}
	h.SetMode(os.ModeSymlink | 0700)
	w, _ := z.CreateHeader(h)
	w.Write([]byte("/etc/passwd"))
	z.Close()
	if _, e := Office(context.Background(), bytes.NewReader(symlink.Bytes()), int64(symlink.Len()), "docx", DefaultLimits()); e == nil {
		t.Fatal("symlink accepted")
	}
}
