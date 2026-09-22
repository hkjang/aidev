package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPostgresNotificationOutboxRetryMinimalContentAndRevocation(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	var calls atomic.Int32
	secret := "notification-test-hmac-secret-2026-strong"
	var deliveryID atomic.Value
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		id := r.Header.Get("X-Madi-ID")
		if !VerifyWebhookSignature(secret, r.Header.Get("X-Madi-Timestamp"), id, r.Header.Get("X-Madi-Signature"), body, time.Now()) {
			t.Error("invalid notification signature")
		}
		if old := deliveryID.Load(); old != nil && old.(string) != id {
			t.Error("retry delivery identity changed")
		}
		deliveryID.Store(id)
		if strings.Contains(string(body), "PRIVATE_SENTINEL") || strings.Contains(string(body), p.ID) {
			t.Error("private notification content escaped", string(body))
		}
		if calls.Add(1) == 1 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	}))
	defer receiver.Close()
	admin.request("PUT", "/api/v1/admin/notification-settings", map[string]any{"enabled": true, "allowed_hosts": []string{"127.0.0.1"}}, 200)
	c := testJSONObject(t, admin.request("POST", "/api/v1/admin/notification-channels", map[string]any{"name": "서명 검증 채널", "kind": "webhook", "enabled": true, "config": map[string]any{"allow_http": true}, "secrets": map[string]any{"url": receiver.URL + "/secret-path", "signing_secret": secret}}, 200))
	cid := str(c, "id")
	if strings.Contains(string(jsonValue(c)), secret) || strings.Contains(string(jsonValue(c)), "secret-path") {
		t.Fatal("channel secret exposed")
	}
	admin.request("PUT", "/api/v1/notification-preferences/"+cid, map[string]any{"enabled": true}, 200)
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "PRIVATE_SENTINEL", "markdown": "PRIVATE_SENTINEL_BODY", "visibility": "private"}, 200))
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	nid := newID()
	_, e = tx.Exec(ctx, "INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,'PRIVATE_SENTINEL',$3)", nid, p.ID, doc["id"])
	if e != nil {
		t.Fatal(e)
	}
	_ = tx.Rollback(ctx)
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM notification_outbox WHERE notification_id=$1", nid).Scan(&count)
	if count != 0 {
		t.Fatal("rolled-back notification left outbox")
	}
	_, e = s.DB.Exec(ctx, "INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,'PRIVATE_SENTINEL',$3)", nid, p.ID, doc["id"])
	if e != nil {
		t.Fatal(e)
	}
	if e = s.dispatchNotifications(context.Background()); e != nil {
		t.Fatal(e)
	}
	drainJobs(t, s)
	if calls.Load() != 2 {
		t.Fatal("webhook transient retry missing", calls.Load())
	}
	var status string
	_ = s.DB.QueryRow(ctx, "SELECT status FROM notification_deliveries WHERE notification_id=$1", nid).Scan(&status)
	if status != "sent" {
		t.Fatal("delivery not persisted", status)
	}
	if e = s.dispatchNotifications(context.Background()); e != nil {
		t.Fatal(e)
	}
	drainJobs(t, s)
	if calls.Load() != 2 {
		t.Fatal("duplicate notification dispatch")
	}
	nid = newID()
	_, e = s.DB.Exec(ctx, "INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,'PRIVATE_SENTINEL',$3)", nid, p.ID, doc["id"])
	if e != nil {
		t.Fatal(e)
	}
	if e = s.dispatchNotifications(context.Background()); e != nil {
		t.Fatal(e)
	}
	admin.request("PUT", "/api/v1/notification-preferences/"+cid, map[string]any{"enabled": false}, 200)
	drainJobs(t, s)
	if calls.Load() != 2 {
		t.Fatal("revoked personal preference still delivered")
	}
	_ = s.DB.QueryRow(ctx, "SELECT status FROM notification_deliveries WHERE notification_id=$1", nid).Scan(&status)
	if status != "skipped" {
		t.Fatal("revoked delivery not skipped", status)
	}
	issued := testJSONObject(t, admin.request("POST", "/api/v1/keys", map[string]any{"name": "외부 키", "workspace_id": wid, "scopes": []string{"document:read"}}, 201))
	key := newIntegrationTestClient(t, admin.base)
	key.token = str(issued, "token")
	key.request("PUT", "/api/v1/notification-preferences/"+cid, map[string]any{"enabled": true}, 403)
}

func TestPostgresNotificationSMTPTLS(t *testing.T) {
	s, _, ctx, _, _ := jobTestFixture(t)
	certServer := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	cert := certServer.TLS.Certificates[0]
	der := certServer.Certificate().Raw
	certServer.Close()
	listener, e := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	message := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, e := listener.Accept()
		if e != nil {
			errCh <- e
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		send := func(text string) { fmt.Fprint(rw, text+"\r\n"); _ = rw.Flush() }
		send("220 localhost SMTP")
		for {
			line, e := rw.ReadString('\n')
			if e != nil {
				errCh <- e
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				send("250-localhost\r\n250 8BITMIME")
			case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"):
				send("250 OK")
			case strings.HasPrefix(line, "DATA"):
				send("354 end with dot")
				var body strings.Builder
				for {
					v, e := rw.ReadString('\n')
					if e != nil {
						errCh <- e
						return
					}
					if v == ".\r\n" {
						break
					}
					body.WriteString(v)
				}
				message <- body.String()
				send("250 accepted")
			case strings.HasPrefix(line, "QUIT"):
				send("221 bye")
				return
			default:
				send("500 unsupported")
			}
		}
	}()
	_, e = s.DB.Exec(ctx, "UPDATE notification_settings SET enabled=true,allowed_hosts='[\"127.0.0.1\"]'")
	if e != nil {
		t.Fatal(e)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	c := notificationChannel{Kind: "smtp", Config: map[string]any{"host": "127.0.0.1", "port": float64(port), "from": "madi@example.test", "tls_mode": "tls", "ca_pem": string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))}, Secrets: map[string]any{}}
	if e = s.sendNotificationEmail(ctx, c, newID(), "recipient@example.test", "madi 새 알림\n개인 알림함에서 확인하세요."); e != nil {
		t.Fatal(e)
	}
	select {
	case body := <-message:
		if !strings.Contains(body, "To: recipient@example.test") || !strings.Contains(body, "Content-Type: text/plain; charset=UTF-8") || !strings.Contains(body, "Message-ID:") {
			t.Fatal("invalid MIME message", body)
		}
	case e := <-errCh:
		t.Fatal(e)
	case <-time.After(5 * time.Second):
		t.Fatal("no SMTP message")
	}
	if validNotificationAddress("victim@example.test\r\nBcc: other@example.test") {
		t.Fatal("email header injection accepted")
	}
}
