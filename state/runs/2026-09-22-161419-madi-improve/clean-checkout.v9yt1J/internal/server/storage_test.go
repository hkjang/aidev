package server

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testS3Signature(r *http.Request, secret string) bool {
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ")
	parts := map[string]string{}
	for _, part := range strings.Split(auth, ",") {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) == 2 {
			parts[pair[0]] = pair[1]
		}
	}
	credential := strings.Split(parts["Credential"], "/")
	if len(credential) != 5 {
		return false
	}
	headers := strings.Split(parts["SignedHeaders"], ";")
	var canonical strings.Builder
	for _, key := range headers {
		value := r.Header.Get(key)
		if key == "host" {
			value = r.Host
		}
		canonical.WriteString(key + ":" + strings.Join(strings.Fields(value), " ") + "\n")
	}
	request := r.Method + "\n" + r.URL.EscapedPath() + "\n" + r.URL.Query().Encode() + "\n" + canonical.String() + "\n" + parts["SignedHeaders"] + "\n" + r.Header.Get("X-Amz-Content-Sha256")
	sum := sha256.Sum256([]byte(request))
	scope := strings.Join(credential[1:], "/")
	message := "AWS4-HMAC-SHA256\n" + r.Header.Get("X-Amz-Date") + "\n" + scope + "\n" + hex.EncodeToString(sum[:])
	key := []byte("AWS4" + secret)
	for _, part := range credential[1:] {
		m := hmac.New(sha256.New, key)
		m.Write([]byte(part))
		key = m.Sum(nil)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(parts["Signature"]))
}
func decodeS3Body(r *http.Request) ([]byte, error) {
	if !strings.Contains(r.Header.Get("Content-Encoding"), "aws-chunked") && !strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
		return io.ReadAll(r.Body)
	}
	reader := bufio.NewReader(r.Body)
	var out bytes.Buffer
	for {
		line, e := reader.ReadString('\n')
		if e != nil {
			return nil, e
		}
		size, e := strconv.ParseInt(strings.Split(strings.TrimSpace(line), ";")[0], 16, 64)
		if e != nil {
			return nil, e
		}
		if size == 0 {
			return out.Bytes(), nil
		}
		if _, e = io.CopyN(&out, reader, size); e != nil {
			return nil, e
		}
		if _, e = reader.ReadString('\n'); e != nil {
			return nil, e
		}
	}
}

type s3Fixture struct {
	server       *httptest.Server
	mu           sync.Mutex
	objects      map[string][]byte
	badSignature atomic.Bool
	requests     atomic.Int32
}

func newS3Fixture(t *testing.T) *s3Fixture {
	t.Helper()
	fixture := &s3Fixture{objects: map[string][]byte{}}
	fixture.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture.requests.Add(1)
		if !testS3Signature(r, "s3-test-secret-long-enough") {
			fixture.badSignature.Store(true)
		}
		fixture.mu.Lock()
		defer fixture.mu.Unlock()
		key := r.URL.Path
		switch r.Method {
		case "PUT":
			data, e := decodeS3Body(r)
			if e != nil {
				t.Error(e)
				w.WriteHeader(400)
				return
			}
			fixture.objects[key] = data
			w.Header().Set("ETag", `"test-etag"`)
			w.WriteHeader(200)
		case "HEAD", "GET":
			data, ok := fixture.objects[key]
			if !ok {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(404)
				fmt.Fprint(w, "<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>")
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.Header().Set("ETag", `"test-etag"`)
			if r.Method == "GET" {
				w.Write(data)
			}
		case "DELETE":
			delete(fixture.objects, key)
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	t.Cleanup(fixture.server.Close)
	return fixture
}
func s3Config(endpoint string) map[string]any {
	return map[string]any{"endpoint": endpoint, "bucket": "madi-bucket", "prefix": "isolated-madi", "region": "us-east-1", "access_key": "madi-test-access", "secret_key": "s3-test-secret-long-enough", "allow_http": true}
}
func storageMultipart(t *testing.T, client *integrationTestClient, endpoint, filename string, data []byte, confirmation string, status int) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, e := writer.CreateFormFile("file", filename)
	if e != nil {
		t.Fatal(e)
	}
	part.Write(data)
	if confirmation != "" {
		writer.WriteField("confirmation", confirmation)
	}
	writer.Close()
	request, _ := http.NewRequest("POST", client.base+endpoint, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-Madi-Request", "1")
	response, e := client.client.Do(request)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	result, e := io.ReadAll(response.Body)
	if e != nil {
		t.Fatal(e)
	}
	if response.StatusCode != status {
		t.Fatalf("multipart %s status=%d want=%d body=%s", endpoint, response.StatusCode, status, result)
	}
	return result
}

func TestStorageS3ProtocolACLChecksumAndRestore(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	fixture := newS3Fixture(t)
	provider := testJSONObject(t, client.request("POST", "/api/v1/storage/providers", map[string]any{"name": "사내 MinIO", "kind": "s3", "workspace_id": wid, "config": s3Config(fixture.server.URL), "enabled": true}, 200))
	pid := str(provider, "id")
	if strings.Contains(string(jsonValue(provider)), "s3-test-secret") || strings.Contains(string(jsonValue(provider)), "madi-test-access") {
		t.Fatal("S3 credentials exposed")
	}
	client.request("POST", "/api/v1/storage/providers/"+pid+"/test", map[string]any{}, 200)
	fixture.mu.Lock()
	remaining := len(fixture.objects)
	fixture.mu.Unlock()
	if remaining != 0 {
		t.Fatal("diagnostic left an object")
	}
	client.request("PUT", "/api/v1/storage/assignment", map[string]any{"workspace_id": wid, "provider_id": pid}, 200)
	doc := testJSONObject(t, client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "S3 문서"}, 200))
	did := str(doc, "id")
	uploaded := testJSONObject(t, storageMultipart(t, client, "/api/v1/attachments?document_id="+did, "test.txt", []byte("S3 original bytes"), "", 200))
	fid := str(uploaded, "id")
	got := client.request("GET", "/api/v1/attachments/"+fid, nil, 200)
	if string(got) != "S3 original bytes" {
		t.Fatal("S3 download differs")
	}
	uploadTestVault(t, client, wid, makeTestVault(t, map[string][]byte{"Imported.md": []byte("[image](picture.png)"), "picture.png": []byte("fake-image-import-bytes")}), 200)
	fixture.mu.Lock()
	beforeObjects := len(fixture.objects)
	fixture.mu.Unlock()
	_, _ = s.DB.Exec(ctx, `ALTER TABLE attachments ADD CONSTRAINT storage_reject_test CHECK(name<>'rollback.pdf')`)
	uploadTestVault(t, client, wid, makeTestVault(t, map[string][]byte{"Rollback.md": []byte("[file](rollback.pdf)"), "rollback.pdf": []byte("rollback bytes")}), 500)
	fixture.mu.Lock()
	afterObjects := len(fixture.objects)
	fixture.mu.Unlock()
	if beforeObjects != afterObjects {
		t.Fatal("failed S3 import left remote object")
	}
	var storedCipher string
	_ = s.DB.QueryRow(ctx, "SELECT config->>'secret_key' FROM storage_providers WHERE id=$1", pid).Scan(&storedCipher)
	if !strings.HasPrefix(storedCipher, "enc:") {
		t.Fatal("S3 secret not encrypted")
	}
	changed := s3Config(fixture.server.URL)
	changed["bucket"] = "other"
	client.request("PUT", "/api/v1/storage/providers/"+pid, map[string]any{"name": "변경", "kind": "s3", "config": changed, "enabled": true}, 409)
	// Provider belonging to a different workspace cannot be selected, even by a
	// service administrator who happens to own both workspaces.
	other := testJSONObject(t, client.request("POST", "/api/v1/workspaces", map[string]any{"name": "다른 저장 공간"}, 200))
	client.request("PUT", "/api/v1/storage/assignment", map[string]any{"workspace_id": other["id"], "provider_id": pid}, 403)
	var objectKey string
	_ = s.DB.QueryRow(ctx, "SELECT object_key FROM attachments WHERE id=$1", fid).Scan(&objectKey)
	wireKey := "/madi-bucket/isolated-madi/" + objectKey
	fixture.mu.Lock()
	fixture.objects[wireKey] = []byte("tampered bytes!!!!!")
	fixture.mu.Unlock()
	client.request("GET", "/api/v1/attachments/"+fid, nil, 404)
	fixture.mu.Lock()
	fixture.objects[wireKey] = []byte("S3 original bytes")
	fixture.mu.Unlock()
	export := client.request("GET", "/api/v1/export?workspace_id="+wid, nil, 200)
	if len(export) < 100 {
		t.Fatal("S3 workspace export missing")
	}
	local := t.TempDir()
	client.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": local}, 200)
	backup := client.request("GET", "/api/v1/admin/backup", nil, 200)
	storageMultipart(t, client, "/api/v1/admin/restore", "backup.zip", backup, "RESTORE", 200)
	if string(client.request("GET", "/api/v1/attachments/"+fid, nil, 200)) != "S3 original bytes" {
		t.Fatal("S3 to local restore failed")
	}
	var providerID *string
	var restoredPath string
	_ = s.DB.QueryRow(ctx, "SELECT storage_provider_id::text,path FROM attachments WHERE id=$1", fid).Scan(&providerID, &restoredPath)
	if providerID != nil || !strings.HasPrefix(restoredPath, local+"/") {
		t.Fatal("restored attachment did not rebase locally")
	}
	var enabled bool
	_ = s.DB.QueryRow(ctx, "SELECT enabled FROM storage_providers WHERE id=$1", pid).Scan(&enabled)
	if enabled {
		t.Fatal("restore left remote writes enabled")
	}
	if fixture.badSignature.Load() {
		t.Fatal("SDK SigV4 request verification failed")
	}
}

func TestBackupSizeWriterEnforcesExactBound(t *testing.T) {
	var target bytes.Buffer
	w := backupLimitWriter{w: &target, remaining: 4}
	if n, e := w.Write([]byte("1234")); n != 4 || e != nil {
		t.Fatalf("exact bound n=%d err=%v", n, e)
	}
	if n, e := w.Write([]byte("5")); n != 0 || e != errBackupTooLarge {
		t.Fatalf("overflow n=%d err=%v", n, e)
	}
	if target.String() != "1234" {
		t.Fatal("overflow reached file")
	}
}

func TestStorageTLSDefaultsAndCustomCA(t *testing.T) {
	s, client, _, _, wid := jobTestFixture(t)
	fixture := newS3Fixture(t)
	tlsServer := httptest.NewTLSServer(fixture.server.Config.Handler)
	defer tlsServer.Close()
	config := s3Config(fixture.server.URL)
	config["allow_http"] = false
	client.request("POST", "/api/v1/storage/providers", map[string]any{"name": "HTTP 금지", "kind": "s3", "workspace_id": wid, "config": config, "enabled": true}, 400)
	config = s3Config(tlsServer.URL)
	provider := testJSONObject(t, client.request("POST", "/api/v1/storage/providers", map[string]any{"name": "TLS 검증", "kind": "s3", "workspace_id": wid, "config": config, "enabled": true}, 200))
	client.request("POST", "/api/v1/storage/providers/"+str(provider, "id")+"/test", map[string]any{}, 502)
	config["ca_pem"] = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tlsServer.TLS.Certificates[0].Certificate[0]}))
	client.request("PUT", "/api/v1/storage/providers/"+str(provider, "id"), map[string]any{"name": "TLS 검증", "kind": "s3", "config": config, "enabled": true}, 200)
	client.request("POST", "/api/v1/storage/providers/"+str(provider, "id")+"/test", map[string]any{}, 200)
	_ = s
}
func TestStorageLocalSandboxAndBackupRetention(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	root := t.TempDir()
	provider := storageProvider{Kind: "local", Enabled: true, Config: map[string]any{"root": root}}
	outside := t.TempDir()
	if e := os.Symlink(outside, filepath.Join(root, "escape")); e != nil {
		t.Fatal(e)
	}
	if _, e := s.putStoredObject(ctx, provider, "escape/"+newID(), strings.NewReader("blocked"), 20, "text/plain"); e == nil {
		t.Fatal("local symlink escaped root")
	}
	object, e := s.putStoredObject(ctx, provider, "attachments/"+newID(), strings.NewReader("owned"), 20, "text/plain")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.putStoredObject(ctx, provider, object.Key, strings.NewReader("overwrite"), 20, "text/plain"); e == nil {
		t.Fatal("local object overwritten")
	}
	if e = s.deleteStoredObject(ctx, storedObject{Path: root}); e == nil {
		t.Fatal("broad directory deletion accepted")
	}
	backupRoot := t.TempDir()
	client.request("PUT", "/api/v1/admin/backups/policy", map[string]any{"enabled": true, "local_path": backupRoot, "interval_minutes": 5, "retention_count": 1}, 200)
	client.request("POST", "/api/v1/admin/backups/run", map[string]any{}, 200)
	drainJobs(t, s)
	first, e := s.one(ctx, "SELECT to_jsonb(a) FROM backup_artifacts a ORDER BY created_at LIMIT 1")
	if e != nil {
		t.Fatal(e)
	}
	foreign := filepath.Join(backupRoot, "do-not-delete.txt")
	if e = os.WriteFile(foreign, []byte("foreign operator file"), 0600); e != nil {
		t.Fatal(e)
	}
	client.request("POST", "/api/v1/admin/backups/run", map[string]any{}, 200)
	drainJobs(t, s)
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM backup_artifacts").Scan(&count)
	if count != 1 {
		t.Fatal("backup retention did not keep exactly one")
	}
	if _, e = os.Stat(str(first, "path")); !os.IsNotExist(e) {
		t.Fatal("old managed backup not removed")
	}
	if _, e = os.Stat(foreign); e != nil {
		t.Fatal("foreign file removed")
	}
	_, _ = s.DB.Exec(ctx, "UPDATE backup_policy SET next_run=now()-interval '1 minute'")
	_, _ = s.DB.Exec(ctx, "UPDATE job_settings SET paused=true")
	s.scheduleBackups(ctx)
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM automation_jobs WHERE status='pending' AND kind='backup.snapshot'").Scan(&count)
	if count != 0 {
		t.Fatal("paused backup schedule executed")
	}
	_ = wid
}
