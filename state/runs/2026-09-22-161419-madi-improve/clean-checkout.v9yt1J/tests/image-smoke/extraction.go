package main

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Real generated PDF: first page has embedded English/Korean text, second page
// is a bitmap containing English/Korean text. Generator: tests/attachment-fixtures.mjs.
//
//go:embed testdata/extraction.pdf
var extractionPDF []byte

func verifyRuntimeSources() {
	const root = "/usr/share/madi/sources"
	var manifest struct {
		Packages  []struct{ Name, Origin, Commit string }
		Origins   []struct{ Origin, Commit, Recipe string }
		Tesseract struct {
			Commit string
			Models []struct{ Language, Path, SHA256 string }
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil || json.Unmarshal(raw, &manifest) != nil {
		fail("runtime source manifest unavailable")
	}
	origins := map[string]bool{}
	for _, origin := range manifest.Origins {
		origins[origin.Origin+"/"+origin.Commit] = true
		if _, err = os.Stat(filepath.Join(root, origin.Recipe)); err != nil {
			fail("runtime source recipe missing")
		}
	}
	for _, pkg := range manifest.Packages {
		if pkg.Origin == "tesseract-ocr" && pkg.Commit == manifest.Tesseract.Commit {
			continue
		}
		if !origins[pkg.Origin+"/"+pkg.Commit] {
			fail("runtime package missing full original source: %s", pkg.Name)
		}
	}
	if len(manifest.Packages) != 69 || len(manifest.Origins) != 52 || len(manifest.Tesseract.Models) != 2 {
		fail("unexpected runtime source inventory size")
	}
	for _, model := range manifest.Tesseract.Models {
		if model.Path != "/usr/share/tessdata/"+model.Language+".traineddata" || (model.Language != "eng" && model.Language != "kor") {
			fail("unrecognized model path")
		}
		file, err := os.Open(model.Path)
		if err != nil {
			fail("model source missing")
		}
		hash := sha256.New()
		_, err = io.Copy(hash, file)
		file.Close()
		if err != nil || hex.EncodeToString(hash.Sum(nil)) != model.SHA256 {
			fail("model source checksum mismatch")
		}
	}
	fmt.Println("PASS exact 69 APKs / 52 full source origins plus Tesseract / English-Korean model hashes")
}

func extractionRequest(path string, value any) map[string]any {
	raw, _ := json.Marshal(value)
	var result map[string]any
	if e := json.Unmarshal(request("POST", "/api/v1"+path, "application/json", raw, 202), &result); e != nil {
		fail("extraction queue JSON: %v", e)
	}
	return result
}
func extractionReady(id string) map[string]any {
	for i := 0; i < 180; i++ {
		value := object("GET", "/attachment-extractions/"+id, nil)
		run := value["extraction"].(map[string]any)
		switch text(run, "status") {
		case "ready":
			if value["active"] != true {
				fail("extraction ready but not active")
			}
			return run
		case "failed", "cancelled", "obsolete":
			fail("native extraction failed: %s", run["error"])
		}
		time.Sleep(time.Second)
	}
	fail("native extraction timeout")
	return nil
}
func verifyExtractionImage(wid string) string {
	policy := object("GET", "/admin/attachment-extraction/settings", nil)
	data := policy["data"].(map[string]any)
	if data["enabled"] != false {
		fail("extraction enabled by default")
	}
	data["enabled"], data["ocr_enabled"] = true, true
	object("PUT", "/admin/attachment-extraction/settings", policy)
	doc := object("POST", "/documents", map[string]any{"workspace_id": wid, "title": "오프라인 PDF와 선택 OCR", "markdown": "원본 파일과 쪽·영역 인용", "visibility": "private"})
	did := text(doc, "id")
	var file map[string]any
	if e := json.Unmarshal(multipartRequest("/attachments?document_id="+did, "native-and-scan.pdf", extractionPDF, nil), &file); e != nil {
		fail("PDF upload")
	}
	aid := text(file, "id")
	args := map[string]any{"document_version": doc["version"], "checksum": file["checksum_sha256"]}
	id := text(extractionRequest("/attachments/"+aid+"/extractions", args), "id")
	run := extractionReady(id)
	result := run["result"].(map[string]any)
	if result["pages"] != float64(2) {
		fail("PDF page count %v", result["pages"])
	}
	empty, ok := result["empty_pages"].([]any)
	if !ok || len(empty) != 1 || empty[0] != float64(2) {
		fail("native empty pages %v", empty)
	}
	native := object("GET", "/attachment-extractions/"+id+"/fragments", nil)
	encoded, _ := json.Marshal(native)
	if !bytes.Contains(encoded, []byte("첫 페이지 원본 텍스트")) || bytes.Contains(encoded, []byte("MADI OFFLINE OCR")) {
		fail("native PDF extraction incorrectly OCR'd unselected page")
	}
	args["ocr_pages"], args["confirmation"] = []int{2}, "OCR"
	ocrID := text(extractionRequest("/attachments/"+aid+"/extractions", args), "id")
	extractionReady(ocrID)
	ocr := object("GET", "/attachment-extractions/"+ocrID+"/fragments", nil)
	fragments := ocr["fragments"].([]any)
	foundEnglish, foundKorean := false, false
	for _, entry := range fragments {
		f := entry.(map[string]any)
		position := f["position"].(map[string]any)
		body := text(f, "text")
		if position["ocr"] == true {
			if position["page"] != float64(2) || len(position["bounds"].([]any)) != 4 {
				fail("OCR missing exact page region")
			}
			foundEnglish = foundEnglish || strings.Contains(body, "MADI OFFLINE OCR 2026")
			foundKorean = foundKorean || strings.Contains(body, "오프라인 문서 추출 검증")
		}
	}
	if !foundEnglish || !foundKorean {
		fail("English/Korean selected OCR missing")
	}
	search := object("GET", "/search?workspace_id="+wid+"&type=file&q=OFFLINE", nil)
	if len(search["results"].([]any)) == 0 {
		fail("OCR body not searchable")
	}
	if !bytes.Equal(request("GET", "/api/v1/attachments/"+aid, "", nil, 200), extractionPDF) {
		fail("extraction changed original PDF")
	}
	fmt.Println("PASS actual offline PDF text, selected page OCR eng+kor, positioned fragments, body search, unchanged source")
	return ocrID
}

func verifyExtractionRestored(id string) {
	policy := object("GET", "/admin/attachment-extraction/settings", nil)["data"].(map[string]any)
	if policy["enabled"] != false || policy["ocr_enabled"] != false {
		fail("restore replayed extraction policy")
	}
	value := object("GET", "/attachment-extractions/"+id, nil)
	if value["active"] != false || text(value["extraction"].(map[string]any), "status") != "obsolete" {
		fail("restore preserved active projection")
	}
	request("GET", "/api/v1/attachment-extractions/"+id+"/fragments", "", nil, 410)
	fmt.Println("PASS restored extraction receipts retained, local derived text removed, fresh explicit extraction required")
}
