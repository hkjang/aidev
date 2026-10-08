package docs

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// A spreadsheet is a deck's numbers, already gathered. One label column and one
// column of figures is a chart, which is what a person would draw from it.
func TestASheetOfFiguresBecomesAChart(t *testing.T) {
	document, err := Read("매출.csv", []byte("분기,매출\n1분기,1180\n2분기,1240\n3분기,1390\n"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, line := range []string{"# 매출", "::columns 매출", "- 1분기 | 1180", "!source 매출.csv | A1:B4"} {
		if !strings.Contains(document.Source, line) {
			t.Errorf("the deck is missing %q:\n%s", line, document.Source)
		}
	}
}

// Anything wider is a table, and it still says where it came from.
func TestAWiderSheetBecomesATable(t *testing.T) {
	document, err := Read("채널.csv", []byte("채널,3분기,4분기\n직영,420억,460억\n대리점,380억,410억\n"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, line := range []string{"::table 채널", "- 채널 | 3분기 | 4분기", "- 직영 | 420억 | 460억", "!source 채널.csv | A1:C3"} {
		if !strings.Contains(document.Source, line) {
			t.Errorf("the deck is missing %q:\n%s", line, document.Source)
		}
	}
}

// A report's headings are its slides, its sentences are its points, and every
// slide says which file and which heading it came from.
func TestAReportBecomesSlides(t *testing.T) {
	document, err := Read("보고서.md", []byte(
		"# 2026 상반기 보고\n\n## 실적\n- 매출 1,240억\n- 이익률 9.8%\n\n## 계획\n자동화 2단계를 착수합니다.\n"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, line := range []string{"# 실적", "- 매출 1,240억", "# 계획", "- 자동화 2단계를 착수합니다.", "!source 보고서.md | 계획"} {
		if !strings.Contains(document.Source, line) {
			t.Errorf("the deck is missing %q:\n%s", line, document.Source)
		}
	}
}

// A workbook is a zip of XML, and the sheet's own name is where the slide came
// from.
func TestAWorkbookIsReadWithoutALibrary(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	add := func(name, body string) {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	add("xl/workbook.xml", `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`+
		`<sheets><sheet name="분기 실적" sheetId="1" r:id="rId1"/></sheets></workbook>`)
	add("xl/_rels/workbook.xml.rels", `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`)
	add("xl/sharedStrings.xml", `<sst><si><t>분기</t></si><si><t>매출</t></si><si><t>1분기</t></si><si><t>2분기</t></si></sst>`)
	add("xl/worksheets/sheet1.xml", `<worksheet><sheetData>`+
		`<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>`+
		`<row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2"><v>1180</v></c></row>`+
		`<row r="3"><c r="A3" t="s"><v>3</v></c><c r="B3"><v>1240</v></c></row>`+
		`</sheetData></worksheet>`)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	document, err := Read("실적.xlsx", buffer.Bytes())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, line := range []string{"# 분기 실적", "::columns 매출", "- 1분기 | 1180", "!source 실적.xlsx | 분기 실적!A1:B3"} {
		if !strings.Contains(document.Source, line) {
			t.Errorf("the deck is missing %q:\n%s", line, document.Source)
		}
	}
}

// A file this package cannot read says so rather than producing an empty deck.
func TestAnUnreadableFileSaysSo(t *testing.T) {
	if _, err := Read("보고서.hwp", []byte("anything")); err == nil {
		t.Error("a .hwp was accepted, and nothing here can read one")
	}
	if Reads("보고서.hwp") || !Reads("매출.xlsx") || !Reads("보고서.pdf") {
		t.Error("Reads disagrees with Read")
	}
	// A PDF this cannot open is refused rather than turned into an empty deck.
	if _, err := Read("보고서.pdf", []byte("%PDF-1.7")); err == nil {
		t.Error("a truncated PDF was accepted")
	}
}

// A report's headings are its slides. Word writes the style as an attribute of
// an element inside the paragraph's properties, which is not a path a struct
// tag can reach through.
func TestAWordReportBecomesSlides(t *testing.T) {
	document := `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>2026 상반기 실적</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>매출은 1,240억으로 전년 대비 12% 늘었습니다.</w:t></w:r></w:p>` +
		`<w:p><w:pPr><w:pStyle w:val="제목 1"/></w:pPr><w:r><w:t>하반기 계획</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>자동화 2단계를 착수합니다.</w:t></w:r></w:p>` +
		`<w:tbl><w:tr><w:tc><w:p><w:r><w:t>채널</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>매출</w:t></w:r></w:p></w:tc></w:tr>` +
		`<w:tr><w:tc><w:p><w:r><w:t>직영</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>420억</w:t></w:r></w:p></w:tc></w:tr></w:tbl>` +
		`</w:body></w:document>`
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(document)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	read, err := Read("상반기 보고서.docx", buffer.Bytes())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, line := range []string{
		"# 2026 상반기 실적", "- 매출은 1,240억으로 전년 대비 12% 늘었습니다.",
		"# 하반기 계획", "::table", "- 직영 | 420억",
		"!source 상반기 보고서.docx | 2026 상반기 실적",
	} {
		if !strings.Contains(read.Source, line) {
			t.Errorf("the deck is missing %q:\n%s", line, read.Source)
		}
	}
}

// A report names itself on its first line. Calling the deck after the file, and
// then giving the document's own title a slide with nothing under it, is how an
// import announces that nobody read the document.
func TestTheDocumentNamesTheDeckRatherThanTheFile(t *testing.T) {
	read, err := Read("report.md", []byte("# 결제 시스템 이중화 이행 보고\n\n## 지금의 문제\n\n단일 리전에 의존하고 있습니다.\n\n## 요청\n\n예산 승인을 요청드립니다.\n"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.Title != "결제 시스템 이중화 이행 보고" {
		t.Fatalf("the deck is called %q", read.Title)
	}
	if !strings.HasPrefix(read.Source, "# 결제 시스템 이중화 이행 보고\n@cover\n> report.md\n") {
		t.Fatalf("the cover is not the document's own title:\n%s", read.Source)
	}
	// And the title does not also appear as a slide with nothing on it.
	if strings.Count(read.Source, "# 결제 시스템 이중화 이행 보고") != 1 {
		t.Fatalf("the document's title became a slide as well as the deck's name:\n%s", read.Source)
	}
	for _, line := range []string{"# 지금의 문제", "- 단일 리전에 의존하고 있습니다.", "# 요청", "- 예산 승인을 요청드립니다."} {
		if !strings.Contains(read.Source, line) {
			t.Fatalf("the import lost %q:\n%s", line, read.Source)
		}
	}
}

// A first section that carries something keeps its slide, and its citation with
// it: moving its sentences onto the cover would leave them with nothing saying
// where they came from.
func TestAFirstSectionWithContentKeepsItsSlide(t *testing.T) {
	read, err := Read("전략.md", []byte("# 2026 채널 전략\n\n직영 채널이 성장을 이끌고 있습니다.\n\n## 현황\n\n대리점은 3분기 연속 감소했습니다.\n"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.Title != "2026 채널 전략" {
		t.Fatalf("the deck is called %q", read.Title)
	}
	for _, line := range []string{
		"- 직영 채널이 성장을 이끌고 있습니다.",
		"!source 전략.md | 2026 채널 전략",
		"# 현황", "- 대리점은 3분기 연속 감소했습니다.",
	} {
		if !strings.Contains(read.Source, line) {
			t.Fatalf("the import lost %q:\n%s", line, read.Source)
		}
	}
}

// A spreadsheet has no title of its own, so the file names the deck.
func TestASpreadsheetIsNamedAfterItsFile(t *testing.T) {
	read, err := Read("quarterly.csv", []byte("채널,4분기,1분기\n직영,420,468\n대리점,310,287\n"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.Title != "quarterly" {
		t.Fatalf("the deck is called %q", read.Title)
	}
	if !strings.Contains(read.Source, "- 직영 | 420 | 468") {
		t.Fatalf("the table did not come across:\n%s", read.Source)
	}
}

func TestScoutSupportedFormats(t *testing.T) {
 for _, item := range []struct{name, body, want string}{
  {"매출.tsv", "분기\t매출\n1분기\t1180\n2분기\t1240\n", "::columns"},
  {"보고서.txt", "# 분기 요약\n- 매출이 늘었습니다.\n", "- 매출이 늘었습니다."},
  {"보고서.md", "\ufeff# 분기 요약\n- 매출이 늘었습니다.\n", "# 분기 요약"},
 } {
  d, err := Read(item.name, []byte(item.body))
  if err != nil { t.Fatal(err) }
  if !strings.Contains(d.Source, item.want) { t.Fatalf("missing %q: %s", item.want, d.Source) }
  t.Logf("%s: %s", item.name, d.Source)
 }
}
