package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

func TestInboundMIMEPlainHTMLAttachmentAndLimits(t *testing.T) {
	raw := []byte("From: sender@example.test\r\nSubject: =?UTF-8?B?7ZqM7J2Y66Gd?=\r\nMessage-ID: <message@example.test>\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=BOUND\r\n\r\n--BOUND\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<h1>회의</h1><script>BAD_SCRIPT</script><p>운영 검토</p>\r\n--BOUND\r\nContent-Type: text/plain; name=\"../../notes.txt\"\r\nContent-Disposition: attachment; filename=\"../../notes.txt\"\r\nContent-Transfer-Encoding: base64\r\n\r\naGVsbG8=\r\n--BOUND--\r\n")
	content, e := parseInboundMIME(raw)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(content.Markdown, "BAD_SCRIPT") || !strings.Contains(content.Markdown, "운영 검토") || len(content.Attachments) != 1 || content.Attachments[0].Name != "notes.txt" || string(content.Attachments[0].Data) != "hello" {
		t.Fatalf("MIME conversion failure %#v", content)
	}
	if _, e = parseInboundMIME(bytes.Repeat([]byte("x"), inboundCaptureMax+1)); e == nil {
		t.Fatal("oversized MIME accepted")
	}
	if _, e = parseInboundMIME([]byte("Subject: x\r\nContent-Type: multipart/mixed\r\n\r\nbody")); e == nil {
		t.Fatal("boundary-less multipart accepted")
	}
}

func TestPostgresCaptureHookHMACAtomicPrivateAndRotation(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	path := t.TempDir()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": path}, 200)
	admin.request("PUT", "/api/v1/admin/inbound-capture-settings", map[string]any{"hooks_enabled": true, "imap_enabled": false, "allowed_hosts": []string{}}, 200)
	channel := testJSONObject(t, admin.request("POST", "/api/v1/capture-channels", map[string]any{"workspace_id": wid, "name": "개인 Webhook 수집", "kind": "hmac", "enabled": true, "config": map[string]any{}}, 200))
	cid, secret := str(channel, "id"), str(channel, "signing_secret")
	if len(secret) < 32 {
		t.Fatal("HMAC one-time secret absent")
	}
	listed := admin.request("GET", "/api/v1/capture-channels", nil, 200)
	if strings.Contains(string(listed), secret) {
		t.Fatal("capture secret returned again")
	}
	request := func(id, key string, body []byte, want int) map[string]any {
		stamp := strconv.FormatInt(time.Now().Unix(), 10)
		r, e := http.NewRequest("POST", admin.base+"/api/v1/capture-hooks/"+cid, bytes.NewReader(body))
		if e != nil {
			t.Fatal(e)
		}
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Madi-ID", id)
		r.Header.Set("X-Madi-Timestamp", stamp)
		r.Header.Set("X-Madi-Signature", WebhookSignature(key, stamp, id, body))
		response, e := http.DefaultClient.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		data, _ := io.ReadAll(response.Body)
		if response.StatusCode != want {
			t.Fatalf("capture status %d want %d: %s", response.StatusCode, want, data)
		}
		return testJSONObject(t, data)
	}
	body := jsonValue(map[string]any{"title": "Webhook 개인 수집", "text": "자동 공유하지 않는 개인 메모입니다.", "attachments": []inboundAttachment{{Name: "../../notes.txt", Type: "text/plain", Data: []byte("private attachment")}}})
	key := newID()
	request(key, "wrong-secret", body, 401)
	receipt := request(key, secret, body, 202)
	duplicate := request(key, secret, body, 202)
	if !boolean(duplicate, "duplicate") || duplicate["id"] != receipt["id"] {
		t.Fatal("capture receipt duplicate changed")
	}
	request(key, secret, jsonValue(map[string]any{"title": "changed"}), 409)
	drainJobs(t, s)
	var did, visibility, owner, status string
	var count int
	e := s.DB.QueryRow(ctx, "SELECT m.document_id::text,m.status,d.visibility,d.owner_id::text FROM inbound_capture_messages m JOIN documents d ON d.id=m.document_id WHERE m.id=$1", receipt["id"]).Scan(&did, &status, &visibility, &owner)
	if e != nil || visibility != "private" || owner != p.ID || status != "completed" {
		t.Fatal("capture ownership failed", e, visibility, status)
	}
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM attachments WHERE document_id=$1", did).Scan(&count)
	if count != 1 {
		t.Fatal("capture attachment absent")
	}
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM capture_receipts WHERE request_id=$1 AND document_id=$2", receipt["id"], did).Scan(&count)
	if count != 1 {
		t.Fatal("common capture receipt absent")
	}
	rotated := testJSONObject(t, admin.request("POST", "/api/v1/capture-channels/"+cid+"/rotate", map[string]any{"revision": channel["revision"]}, 200))
	request(newID(), secret, body, 401)
	secret = str(rotated, "signing_secret")
	// A DB failure after files are prepared must not leave a half-created Inbox item.
	_, e = s.DB.Exec(ctx, "ALTER TABLE attachments ADD CONSTRAINT inbound_test_reject CHECK(name<>'reject.txt')")
	if e != nil {
		t.Fatal(e)
	}
	bad := request(newID(), secret, jsonValue(map[string]any{"title": "원자적 실패", "text": "저장되면 안 됨", "attachments": []inboundAttachment{{Name: "ok.txt", Type: "text/plain", Data: []byte("first")}, {Name: "reject.txt", Type: "text/plain", Data: []byte("second")}}}), 202)
	files := func() int {
		count := 0
		if err := filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				count++
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return count
	}
	before := files()
	drainJobs(t, s)
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE title='원자적 실패'").Scan(&count)
	if count != 0 {
		t.Fatal("partially created document survived")
	}
	var linked bool
	_ = s.DB.QueryRow(ctx, "SELECT document_id IS NOT NULL FROM inbound_capture_messages WHERE id=$1", bad["id"]).Scan(&linked)
	if linked {
		t.Fatal("failed message linked a partial document")
	}
	if files() != before {
		t.Fatal("failed capture left attachment files")
	}
}

func TestPostgresCaptureCancellationRevocationAndCheckpoint(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	admin.request("PUT", "/api/v1/admin/inbound-capture-settings", map[string]any{"hooks_enabled": true, "imap_enabled": false, "allowed_hosts": []string{}}, 200)
	channel := testJSONObject(t, admin.request("POST", "/api/v1/capture-channels", map[string]any{"workspace_id": wid, "name": "취소·권한 검증", "kind": "hmac", "enabled": true, "config": map[string]any{}}, 200))
	cid := str(channel, "id")
	c, e := s.inboundChannel(ctx, cid)
	if e != nil {
		t.Fatal(e)
	}
	stage := func(title string) string {
		id, _, err := s.stageInboundContent(ctx, c, "hook:"+newID(), inboundContent{Title: title, Markdown: "private data"})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	assertNoDocument := func(id, status string) {
		t.Helper()
		var actual string
		var linked bool
		if err := s.DB.QueryRow(ctx, "SELECT status,document_id IS NOT NULL FROM inbound_capture_messages WHERE id=$1", id).Scan(&actual, &linked); err != nil {
			t.Fatal(err)
		}
		if linked || actual != status {
			t.Fatalf("unexpected message result status=%s linked=%v", actual, linked)
		}
	}
	cancelled := stage("사용자가 취소한 수집")
	admin.request("POST", "/api/v1/capture-channels/"+cid+"/messages/"+cancelled+"/cancel", map[string]any{}, 200)
	drainJobs(t, s)
	assertNoDocument(cancelled, "cancelled")
	revoked := stage("작성권한 회수한 수집")
	if _, e = s.DB.Exec(ctx, "UPDATE workspace_members SET role='viewer' WHERE user_id=$1 AND workspace_id=$2", p.ID, wid); e != nil {
		t.Fatal(e)
	}
	drainJobs(t, s)
	assertNoDocument(revoked, "pending")
	var jobStatus string
	if e = s.DB.QueryRow(ctx, "SELECT j.status FROM automation_jobs j JOIN inbound_capture_messages m ON m.job_id=j.id WHERE m.id=$1", revoked).Scan(&jobStatus); e != nil || jobStatus != "failed" {
		t.Fatal("revoked job should fail closed", jobStatus, e)
	}
	if _, e = s.DB.Exec(ctx, "UPDATE workspace_members SET role='owner' WHERE user_id=$1 AND workspace_id=$2", p.ID, wid); e != nil {
		t.Fatal(e)
	}
	admin.request("POST", "/api/v1/capture-channels/"+cid+"/messages/"+revoked+"/cancel", map[string]any{}, 200)
	rotating := stage("회전 이후 대기 수집")
	admin.request("POST", "/api/v1/capture-channels/"+cid+"/rotate", map[string]any{"revision": c.Revision}, 200)
	drainJobs(t, s)
	assertNoDocument(rotating, "cancelled")
	c, e = s.inboundChannel(ctx, cid)
	if e != nil {
		t.Fatal(e)
	}
	for _, point := range [][3]int{{123, 20, 0}, {123, 10, 0}, {124, 5, 123}} {
		if e = s.advanceInboundCheckpoint(ctx, c, point[0], point[1], point[2]); e != nil {
			t.Fatal(e)
		}
		var uid int
		if e = s.DB.QueryRow(ctx, "SELECT (checkpoint->>'last_uid')::int FROM inbound_capture_channels WHERE id=$1", cid).Scan(&uid); e != nil {
			t.Fatal(e)
		}
		want := 20
		if point[0] == 124 {
			want = 5
		}
		if uid != want {
			t.Fatalf("checkpoint rewound: %d want %d", uid, want)
		}
	}
	if e = s.advanceInboundCheckpoint(ctx, c, 123, 99, 0); e == nil {
		t.Fatal("stale IMAP epoch replaced current generation")
	}
}

type captureIMAPSession struct {
	imapserver.Session
	examined, peek *atomic.Bool
}

func (s *captureIMAPSession) Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error) {
	if options != nil && options.ReadOnly {
		s.examined.Store(true)
	}
	return s.Session.Select(mailbox, options)
}
func (s *captureIMAPSession) Fetch(w *imapserver.FetchWriter, set imap.NumSet, options *imap.FetchOptions) error {
	for _, section := range options.BodySection {
		if section.Peek {
			s.peek.Store(true)
		}
	}
	return s.Session.Fetch(w, set, options)
}
func TestPostgresIMAPTLSReadonlyAndMessageDedup(t *testing.T) {
	s, admin, ctx, _, wid := jobTestFixture(t)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	admin.request("PUT", "/api/v1/admin/inbound-capture-settings", map[string]any{"hooks_enabled": false, "imap_enabled": true, "allowed_hosts": []string{"127.0.0.1"}}, 200)
	template := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	cert := template.TLS.Certificates[0]
	ca := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: template.Certificate().Raw}))
	template.Close()
	listener, e := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
	if e != nil {
		t.Fatal(e)
	}
	memory := imapmemserver.New()
	user := imapmemserver.NewUser("mail-reader", "private-mail-password")
	if e = user.Create("INBOX", nil); e != nil {
		t.Fatal(e)
	}
	raw := []byte("From: sender@example.test\r\nTo: receiver@example.test\r\nSubject: IMAP private note\r\nMessage-ID: <immutable-mail@example.test>\r\nDate: Tue, 08 Sep 2026 09:00:00 +0900\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n메일에서 수집한 개인 운영 메모입니다.\r\n")
	if _, e = user.Append("INBOX", bytes.NewReader(raw), &imap.AppendOptions{}); e != nil {
		t.Fatal(e)
	}
	memory.AddUser(user)
	var examined, peek atomic.Bool
	remote := imapserver.New(&imapserver.Options{NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
		return &captureIMAPSession{memory.NewSession(), &examined, &peek}, nil, nil
	}, Caps: imap.CapSet{imap.CapIMAP4rev1: {}}})
	defer remote.Close()
	go remote.Serve(listener)
	channel := testJSONObject(t, admin.request("POST", "/api/v1/capture-channels", map[string]any{"workspace_id": wid, "name": "TLS 메일 수집", "kind": "imap", "enabled": true, "config": map[string]any{"host": "127.0.0.1", "port": listener.Addr().(*net.TCPAddr).Port, "mailbox": "INBOX", "ca_pem": ca, "from_now": false, "interval_minutes": 5}, "secrets": map[string]any{"username": "mail-reader", "password": "private-mail-password"}}, 200))
	cid := str(channel, "id")
	admin.request("POST", "/api/v1/capture-channels/"+cid+"/poll", map[string]any{}, 200)
	drainJobs(t, s)
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM inbound_capture_messages WHERE channel_id=$1 AND status='completed'", cid).Scan(&count)
	if count != 1 || !examined.Load() || !peek.Load() {
		var jobs []byte
		_ = s.DB.QueryRow(ctx, "SELECT jsonb_agg(jsonb_build_object('kind',kind,'status',status,'error',last_error)) FROM automation_jobs WHERE kind LIKE 'capture.%'").Scan(&jobs)
		t.Fatal("IMAP readonly ingestion failed", count, examined.Load(), peek.Load(), string(jobs))
	}
	if _, e = user.Append("INBOX", bytes.NewReader(raw), &imap.AppendOptions{}); e != nil {
		t.Fatal(e)
	}
	admin.request("POST", "/api/v1/capture-channels/"+cid+"/poll", map[string]any{}, 200)
	drainJobs(t, s)
	_ = s.DB.QueryRow(context.Background(), "SELECT count(*) FROM inbound_capture_messages WHERE channel_id=$1", cid).Scan(&count)
	if count != 1 {
		t.Fatal("repeated Message-ID created duplicate captures", count)
	}
}
