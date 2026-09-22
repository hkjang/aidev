package extract

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestPDFAndOCRSourceLocations(t *testing.T) {
	c := newCollector(DefaultLimits())
	xml := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "https://example.invalid/no-fetch"><html><doc><page width="612" height="792"><line xMin="-5" yMin="10" xMax="200" yMax="40"><word>PDF</word><word>원본</word></line></page></doc></html>`
	size, e := pdfWords([]byte(xml), 2, c)
	if e != nil {
		t.Fatal(e)
	}
	if len(c.result.Fragments) != 1 || c.result.Fragments[0].Text != "PDF 원본" || c.result.Fragments[0].Position.Page != 2 || c.result.Fragments[0].Position.Bounds[0] != 0 {
		t.Fatalf("position %+v", c.result)
	}
	tsv := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n5\t1\t1\t1\t1\t1\t10\t20\t30\t40\t98.2\t\"원문\n5\t1\t1\t1\t1\t2\t45\t20\t20\t40\t98.2\t검증\"\n"
	if e = ocrWords([]byte(tsv), 2, size, 612, 792, c); e != nil {
		t.Fatal(e)
	}
	f := c.result.Fragments[1]
	if f.Text != "\"원문 검증\"" || !f.Position.OCR || f.Position.Bounds[2] != 65 {
		t.Fatalf("OCR: %+v", f)
	}
	if _, e = pdfPages([]byte("Pages: 2\nEncrypted: yes (print:yes)\n")); e == nil {
		t.Fatal("encrypted PDF accepted")
	}
	if _, e = pdfWords([]byte(`<html/>`), 1, newCollector(DefaultLimits())); e == nil {
		t.Fatal("missing page accepted")
	}
}

// Run against a test-only offline container with its /tmp/madi-extract-native
// fixture already installed. See tests/attachment-fixtures.mjs. The source file
// is generated locally, not downloaded or borrowed from a user workspace.
func TestPDFNativeContainer(t *testing.T) {
	container := os.Getenv("MADI_EXTRACT_TEST_CONTAINER")
	if container == "" {
		t.Skip("explicit disposable native extractor container required")
	}
	if !regexp.MustCompile(`^madi-extraction-[a-z0-9-]+$`).MatchString(container) {
		t.Fatal("test container prefix required")
	}
	docker := os.Getenv("MADI_EXTRACT_TEST_DOCKER")
	if docker == "" {
		docker = "docker"
	}
	base := []string{}
	if host := os.Getenv("MADI_EXTRACT_TEST_DOCKER_HOST"); host != "" {
		base = append(base, "--host="+host)
	}
	dir, e := os.MkdirTemp(t.TempDir(), "madi-extract-")
	if e != nil {
		t.Fatal(e)
	}
	call := func(ctx context.Context, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, docker, append(append([]string{}, base...), args...)...)
		return cmd.Output()
	}
	run := func(ctx context.Context, _, op string, page int) ([]byte, error) {
		output, e := call(ctx, "exec", "--user", "10001:10001", container, "/usr/local/bin/madi-extract-worker", workerFlag, op, "/tmp/madi-extract-native", strconv.Itoa(page))
		if e != nil {
			return nil, e
		}
		if op == "pdf-render" {
			data, e := call(ctx, "exec", "--user", "10001:10001", container, "/bin/cat", "/tmp/madi-extract-native/page.png")
			if e != nil {
				return nil, e
			}
			if e = os.WriteFile(filepath.Join(dir, "page.png"), data, 0600); e != nil {
				return nil, e
			}
		}
		return output, nil
	}
	plain, e := readPDF(t.Context(), dir, nil, DefaultLimits(), run)
	if e != nil {
		t.Fatal(e)
	}
	if plain.Pages != 2 || len(plain.EmptyPages) != 1 || plain.EmptyPages[0] != 2 {
		t.Fatalf("unexpected source pages: %+v", plain)
	}
	for _, fragment := range plain.Fragments {
		if fragment.Position.OCR || fragment.Position.Page != 1 {
			t.Fatal("unselected scan was processed")
		}
	}
	result, e := readPDF(t.Context(), dir, []int{1, 2}, DefaultLimits(), run)
	if e != nil {
		t.Fatal(e)
	}
	text := ""
	ocr := 0
	for _, fragment := range result.Fragments {
		if fragment.Position.OCR {
			ocr++
			if fragment.Position.Page != 2 || len(fragment.Position.Bounds) != 4 {
				t.Fatalf("OCR provenance missing: %+v", fragment)
			}
		}
		text += fragment.Text + "\n"
	}
	if !strings.Contains(text, "Offline knowledge source alpha 2026") || !strings.Contains(text, "MADI OFFLINE OCR 2026") || !strings.Contains(text, "오프라인 문서 추출 검증") || ocr < 2 {
		t.Fatalf("native text/scan missing: %s", text)
	}
	if path := os.Getenv("MADI_EXTRACT_TEST_REPORT"); path != "" {
		data, _ := json.MarshalIndent(result, "", "  ")
		if e = os.WriteFile(path, data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	t.Logf("native PDF+selective English/Korean OCR: pages=%d fragments=%d OCR=%d; current source locations preserved", result.Pages, len(result.Fragments), ocr)
}
