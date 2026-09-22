// image-smoke runs only inside the disposable internal Docker test network.
// It is mounted into a separate container and is not shipped in the service image.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var client *http.Client
var base string

func fail(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }
func request(method, path, contentType string, body []byte, status int) []byte {
	r, e := http.NewRequest(method, base+path, bytes.NewReader(body))
	if e != nil {
		fail("request: %v", e)
	}
	r.Header.Set("X-Madi-Request", "1")
	r.Header.Set("Origin", base)
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	res, e := client.Do(r)
	if e != nil {
		fail("%s %s: %v", method, path, e)
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if e != nil || res.StatusCode != status {
		fail("%s %s: status %d, expected %d; read error %v", method, path, res.StatusCode, status, e)
	}
	return raw
}
func object(method, path string, value any) map[string]any {
	var raw []byte
	if value != nil {
		raw, _ = json.Marshal(value)
	}
	var result map[string]any
	if e := json.Unmarshal(request(method, "/api/v1"+path, "application/json", raw, 200), &result); e != nil {
		fail("JSON %s: %v", path, e)
	}
	return result
}
func text(value map[string]any, key string) string {
	s, _ := value[key].(string)
	if s == "" {
		fail("missing %s", key)
	}
	return s
}
func login() {
	object("POST", "/auth/login", map[string]any{"email": "admin@example.internal", "password": os.Getenv("MADI_IMAGE_PASSWORD")})
	if text(object("GET", "/auth/me", nil), "role") != "admin" {
		fail("bootstrap administrator missing")
	}
}
func multipartRequest(path, name string, data []byte, fields map[string]string) []byte {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for key, value := range fields {
		if e := w.WriteField(key, value); e != nil {
			fail("multipart field: %v", e)
		}
	}
	f, e := w.CreateFormFile("file", name)
	if e != nil {
		fail("multipart file: %v", e)
	}
	if _, e = f.Write(data); e != nil {
		fail("multipart body: %v", e)
	}
	if e = w.Close(); e != nil {
		fail("multipart close: %v", e)
	}
	return request("POST", "/api/v1"+path, w.FormDataContentType(), body.Bytes(), 200)
}
func assets() {
	index := request("GET", "/login", "", nil, 200)
	refs := regexp.MustCompile(`(?:src|href)="([^"]+)"`).FindAllSubmatch(index, -1)
	fonts, scripts := 0, 0
	for _, ref := range refs {
		path := string(ref[1])
		if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
			fail("nonlocal initial runtime asset %q", path)
		}
		raw := request("GET", path, "", nil, 200)
		if strings.HasSuffix(path, ".js") {
			scripts++
		}
		if !strings.HasSuffix(path, ".css") {
			continue
		}
		for _, match := range regexp.MustCompile(`url\(["']?([^"')]+)["']?\)`).FindAllSubmatch(raw, -1) {
			font := string(match[1])
			if strings.HasPrefix(font, "data:") {
				continue
			}
			u, _ := url.Parse(base + path)
			relative, e := url.Parse(font)
			if e != nil {
				fail("asset URL")
			}
			target := u.ResolveReference(relative)
			if target.Scheme+"://"+target.Host != base {
				fail("external CSS runtime asset")
			}
			request("GET", target.RequestURI(), "", nil, 200)
			if strings.Contains(font, ".woff") {
				fonts++
			}
		}
	}
	if scripts == 0 || fonts == 0 {
		fail("bundled scripts/fonts were not exercised: %d/%d", scripts, fonts)
	}
	fmt.Printf("PASS bundled offline assets (%d scripts, %d fonts)\n", scripts, fonts)
}
func main() {
	if os.Getenv("MADI_IMAGE_SMOKE") != "disposable-internal-network" {
		fail("explicit disposable image test acknowledgement required")
	}
	verifyRuntimeSources()
	base = "http://madi:8080"
	if os.Getenv("MADI_IMAGE_PASSWORD") == "" {
		fail("test password missing")
	}
	jar, _ := cookiejar.New(nil)
	client = &http.Client{Jar: jar, Timeout: 90 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	for i := 0; i < 90; i++ {
		res, e := client.Get(base + "/readyz")
		if e == nil {
			res.Body.Close()
			if res.StatusCode == 200 {
				break
			}
		}
		time.Sleep(time.Second)
	}
	request("GET", "/readyz", "", nil, 200)
	version := object("GET", "/public", nil)
	if expected := os.Getenv("MADI_IMAGE_VERSION"); expected != "" && text(version, "version") != expected {
		fail("embedded version mismatch")
	}
	assets()
	login()
	wid := text(object("POST", "/workspaces", map[string]any{"name": "폐쇄망 이미지 검증"}), "id")
	original := "# 폐쇄망 기록\n\n네 개 환경변수와 로컬 첨부, 비동기 내보내기를 확인합니다.\n"
	doc := object("POST", "/documents", map[string]any{"workspace_id": wid, "title": "이미지 왕복", "markdown": original, "visibility": "private"})
	id := text(doc, "id")
	var attachment map[string]any
	payload := []byte("madi offline attachment exact bytes\n")
	if e := json.Unmarshal(multipartRequest("/attachments?document_id="+id, "offline.txt", payload, nil), &attachment); e != nil {
		fail("attachment JSON")
	}
	aid := text(attachment, "id")
	if !bytes.Equal(request("GET", "/api/v1/attachments/"+aid, "", nil, 200), payload) {
		fail("attachment bytes differ")
	}
	doc = object("GET", "/documents/"+id, nil)
	object("PUT", "/documents/"+id, map[string]any{"title": "이미지 왕복", "markdown": original + "\n[첨부](/api/v1/attachments/" + aid + ")\n", "version": doc["version"]})
	doc = object("GET", "/documents/"+id, nil)
	canonical := text(doc, "markdown")
	run := text(object("POST", "/exports", map[string]any{"workspace_id": wid, "format": "markdown", "document_ids": []string{id}}), "id")
	for i := 0; i < 90; i++ {
		result := object("GET", "/exports/"+run, nil)
		status := text(result, "status")
		if status == "ready" {
			break
		}
		if status == "failed" || status == "cancelled" {
			fail("export %s", status)
		}
		time.Sleep(time.Second)
	}
	raw := request("GET", "/api/v1/exports/"+run+"/download", "", nil, 200)
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		fail("export archive: %v", e)
	}
	found := false
	for _, f := range z.File {
		if !strings.HasSuffix(f.Name, ".md") {
			continue
		}
		r, e := f.Open()
		if e != nil {
			fail("export file")
		}
		data, e := io.ReadAll(r)
		r.Close()
		if e == nil && string(data) == canonical {
			found = true
		}
	}
	if !found {
		fail("raw Markdown bytes not preserved")
	}
	fmt.Println("PASS bootstrap, document CRUD, attachment bytes and durable raw export")
	extractionID := verifyExtractionImage(wid)
	backup := request("GET", "/api/v1/admin/backup", "", nil, 200)
	sessionURL, _ := url.Parse(base)
	oldCookies := client.Jar.Cookies(sessionURL)
	object("PUT", "/documents/"+id, map[string]any{"title": "백업 뒤 변경", "markdown": "복원 대상 변경", "version": doc["version"]})
	multipartRequest("/admin/restore", "madi-backup.zip", backup, map[string]string{"confirmation": "RESTORE"})
	// A matching restored administrator receives a new session; the old cookie
	// must nevertheless stop working in any other browser/device.
	staleJar, _ := cookiejar.New(nil)
	staleJar.SetCookies(sessionURL, oldCookies)
	freshJar := client.Jar
	client.Jar = staleJar
	request("GET", "/api/v1/auth/me", "", nil, 401)
	client.Jar = freshJar
	login()
	if text(object("GET", "/documents/"+id, nil), "markdown") != canonical {
		fail("restored document bytes differ")
	}
	if !bytes.Equal(request("GET", "/api/v1/attachments/"+aid, "", nil, 200), payload) {
		fail("restored attachment bytes differ")
	}
	if object("GET", "/admin/exports/settings", nil)["enabled"] != false {
		fail("restore did not disable asynchronous export policy")
	}
	verifyExtractionRestored(extractionID)
	fmt.Println("PASS logical backup/restore, attachment recovery, session invalidation and paused external workflows")
}
