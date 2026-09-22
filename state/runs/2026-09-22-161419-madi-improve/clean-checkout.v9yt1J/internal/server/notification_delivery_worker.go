package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) StartNotificationDelivery(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			if e := s.dispatchNotifications(ctx); e != nil && ctx.Err() == nil {
				slog.Error("notification dispatcher failed")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (s *Server) dispatchNotifications(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var paused bool
	if e = tx.QueryRow(ctx, "SELECT paused FROM job_settings WHERE id=1").Scan(&paused); e != nil {
		return e
	}
	if paused {
		return nil
	}
	var enabled bool
	if e = tx.QueryRow(ctx, "SELECT enabled FROM notification_settings WHERE id=1").Scan(&enabled); e != nil {
		return e
	}
	rows, e := tx.Query(ctx, "SELECT notification_id::text FROM notification_outbox WHERE processed_at IS NULL ORDER BY created_at LIMIT 100 FOR UPDATE SKIP LOCKED")
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		var uid, wid, documentWorkspace string
		var allowed bool
		e = tx.QueryRow(ctx, `SELECT n.user_id::text,coalesce(d.workspace_id::text,(SELECT m.workspace_id::text FROM workspace_members m WHERE m.user_id=n.user_id ORDER BY m.workspace_id LIMIT 1),''),coalesce(d.workspace_id::text,''),NOT u.disabled AND u.kind='user' AND n.read_at IS NULL AND (n.document_id IS NULL OR madi_document_allowed(n.user_id,n.document_id,false)) FROM notifications n JOIN users u ON u.id=n.user_id LEFT JOIN documents d ON d.id=n.document_id WHERE n.id=$1`, id).Scan(&uid, &wid, &documentWorkspace, &allowed)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if e == nil && enabled && allowed && wid != "" {
			channels, e := tx.Query(ctx, `SELECT c.id::text,c.revision FROM notification_channels c JOIN notification_preferences p ON p.channel_id=c.id AND p.user_id=$1 AND p.enabled WHERE c.enabled AND (c.workspace_id IS NULL OR c.workspace_id::text=$2)`, uid, documentWorkspace)
			if e != nil {
				return e
			}
			type target struct {
				id       string
				revision int
			}
			targets := []target{}
			for channels.Next() {
				var c target
				if e = channels.Scan(&c.id, &c.revision); e != nil {
					channels.Close()
					return e
				}
				targets = append(targets, c)
			}
			e = channels.Err()
			channels.Close()
			if e != nil {
				return e
			}
			for _, c := range targets {
				did := newID()
				tag, e := tx.Exec(ctx, "INSERT INTO notification_deliveries(id,notification_id,channel_id,user_id,channel_revision) VALUES($1,$2,$3,$4,$5) ON CONFLICT(notification_id,channel_id) DO NOTHING", did, id, c.id, uid, c.revision)
				if e != nil {
					return e
				}
				if tag.RowsAffected() == 0 {
					continue
				}
				jid, e := s.EnqueueJob(ctx, tx, "notification.deliver", uid, wid, map[string]any{"delivery_id": did})
				if e != nil {
					return e
				}
				if _, e = tx.Exec(ctx, "UPDATE notification_deliveries SET job_id=$2 WHERE id=$1", did, jid); e != nil {
					return e
				}
			}
		}
		if _, e = tx.Exec(ctx, "UPDATE notification_outbox SET processed_at=now() WHERE notification_id=$1", id); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
func (s *Server) notificationNetwork(ctx context.Context, host string, cfg map[string]any) (*tls.Config, error) {
	var enabled bool
	var raw []byte
	if s.DB.QueryRow(ctx, "SELECT enabled,allowed_hosts FROM notification_settings WHERE id=1").Scan(&enabled, &raw) != nil || !enabled {
		return nil, jobPermanent("외부 알림이 관리자 설정에서 중지되었습니다")
	}
	var hosts []string
	if json.Unmarshal(raw, &hosts) != nil {
		return nil, errors.New("알림 호스트 정책을 읽지 못했습니다")
	}
	found := false
	for _, h := range hosts {
		if strings.EqualFold(h, host) {
			found = true
		}
	}
	if !found {
		return nil, jobPermanent("관리자가 허용하지 않은 알림 호스트입니다")
	}
	config := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host, InsecureSkipVerify: boolean(cfg, "insecure_tls")}
	if pem := str(cfg, "ca_pem"); pem != "" {
		pool, _ := x509.SystemCertPool()
		if pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM([]byte(pem)) {
			return nil, jobPermanent("사내 CA 인증서 형식을 확인하세요")
		}
		config.RootCAs = pool
	}
	return config, nil
}
func (s *Server) deliverNotification(ctx context.Context, j Job) (map[string]any, error) {
	id := str(j.Payload, "delivery_id")
	var channelID, email, status string
	var revision int
	var eligible bool
	e := s.DB.QueryRow(ctx, `SELECT d.channel_id::text,u.email,d.status,d.channel_revision,NOT u.disabled AND u.kind='user' AND n.read_at IS NULL AND coalesce(p.enabled,false) AND (n.document_id IS NULL OR madi_document_allowed(d.user_id,n.document_id,false)) FROM notification_deliveries d JOIN notifications n ON n.id=d.notification_id JOIN users u ON u.id=d.user_id LEFT JOIN notification_preferences p ON p.channel_id=d.channel_id AND p.user_id=d.user_id WHERE d.id=$1 AND d.user_id=$2 AND d.job_id=$3`, id, j.OwnerID, j.ID).Scan(&channelID, &email, &status, &revision, &eligible)
	if e != nil {
		return nil, jobPermanent("현재 알림 전송 대상을 찾을 수 없습니다")
	}
	if status == "sent" {
		return map[string]any{"sent": true, "deduplicated": true}, nil
	}
	finish := func(state, message string) {
		_, _ = s.DB.Exec(ctx, "UPDATE notification_deliveries SET status=$2,message=$3,sent_at=CASE WHEN $2='sent' THEN now() ELSE sent_at END,updated_at=now() WHERE id=$1", id, state, message)
	}
	if !eligible {
		finish("skipped", "현재 사용자·문서·알림 선택 조건이 변경되었습니다")
		return map[string]any{"skipped": true}, nil
	}
	c, e := s.notificationChannel(ctx, channelID)
	if e != nil || !c.Enabled || c.Revision != revision {
		finish("skipped", "채널이 중지되거나 설정이 변경되었습니다")
		return map[string]any{"skipped": true}, nil
	}
	if c.WorkspaceID != "" && c.WorkspaceID != j.WorkspaceID {
		finish("skipped", "채널 대상 워크스페이스가 다릅니다")
		return map[string]any{"skipped": true}, nil
	}
	cfg, e := s.settings(ctx)
	if e != nil {
		return nil, e
	}
	site := strings.TrimRight(str(cfg, "site_url"), "/")
	link := ""
	if u, e := connectorURL(site, ""); e == nil {
		link = u.String() + "/app/notifications"
	}
	text := "madi에 새 알림이 있습니다. 로그인하여 개인 알림함에서 확인하세요."
	if link != "" {
		text += "\n" + link
	}
	finish("sending", "")
	if c.Kind == "smtp" {
		e = s.sendNotificationEmail(ctx, c, id, email, text)
	} else {
		e = s.sendNotificationWebhook(ctx, c, id, text, link)
	}
	if e != nil {
		finish("failed", "외부 알림 전송에 실패했습니다. 채널 정책과 작업 이력을 확인하세요")
		return nil, e
	}
	finish("sent", "")
	return map[string]any{"sent": true, "channel_id": c.ID}, nil
}
func (s *Server) sendNotificationWebhook(ctx context.Context, c notificationChannel, id, text, link string) error {
	endpoint, e := connectorURL(str(c.Secrets, "url"), "")
	if e != nil {
		return jobPermanent("Webhook 주소를 확인하세요")
	}
	if endpoint.Scheme == "http" && !boolean(c.Config, "allow_http") {
		return jobPermanent("HTTP 알림 연결이 허용되지 않았습니다")
	}
	tlsConfig, e := s.notificationNetwork(ctx, endpoint.Hostname(), c.Config)
	if e != nil {
		return e
	}
	payload := map[string]any{"text": text}
	if c.Kind == "teams" {
		payload = map[string]any{"type": "message", "attachments": []any{map[string]any{"contentType": "application/vnd.microsoft.card.adaptive", "content": map[string]any{"$schema": "http://adaptivecards.io/schemas/adaptive-card.json", "type": "AdaptiveCard", "version": "1.2", "body": []any{map[string]any{"type": "TextBlock", "text": text, "wrap": true}}}}}}
	}
	if c.Kind == "webhook" {
		payload = map[string]any{"id": id, "type": "notification.available", "text": text, "url": link}
	}
	body := jsonValue(payload)
	request, e := http.NewRequestWithContext(ctx, "POST", endpoint.String(), bytes.NewReader(body))
	if e != nil {
		return jobPermanent("알림 요청을 만들지 못했습니다")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "madi-notifications/"+s.Version)
	if c.Kind == "webhook" {
		stamp := strconv.FormatInt(time.Now().Unix(), 10)
		request.Header.Set("X-Madi-ID", id)
		request.Header.Set("X-Madi-Timestamp", stamp)
		request.Header.Set("X-Madi-Signature", WebhookSignature(str(c.Secrets, "signing_secret"), stamp, id, body))
	}
	transport := &http.Transport{Proxy: nil, TLSClientConfig: tlsConfig, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 20 * time.Second, DisableKeepAlives: true, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, e := net.SplitHostPort(address)
		if e != nil || !strings.EqualFold(host, endpoint.Hostname()) {
			return nil, errors.New("알림 연결 대상이 다릅니다")
		}
		return webhookDial(ctx, network, address)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, e := client.Do(request)
	if e != nil {
		return errors.New("Webhook 연결 실패: 네트워크와 인증서를 확인하세요")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 65536))
	if response.StatusCode == 429 {
		return connectorRetry{delay: time.Minute}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if response.StatusCode < 500 {
			return jobPermanent(fmt.Sprintf("Webhook HTTP %d: 주소와 채널 권한을 확인하세요", response.StatusCode))
		}
		return fmt.Errorf("Webhook HTTP %d", response.StatusCode)
	}
	return nil
}
func (s *Server) sendNotificationEmail(ctx context.Context, c notificationChannel, id, recipient, text string) error {
	if !validNotificationAddress(recipient) {
		return jobPermanent("현재 사용자 이메일 주소가 올바르지 않습니다")
	}
	host := str(c.Config, "host")
	tlsConfig, e := s.notificationNetwork(ctx, host, c.Config)
	if e != nil {
		return e
	}
	port := number(c.Config, "port", 465)
	conn, e := webhookDial(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if e != nil {
		return errors.New("SMTP 연결에 실패했습니다")
	}
	defer conn.Close()
	deadline := time.Now().Add(30 * time.Second)
	if when, ok := ctx.Deadline(); ok && when.Before(deadline) {
		deadline = when
	}
	_ = conn.SetDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	mode := str(c.Config, "tls_mode")
	if mode == "tls" {
		secure := tls.Client(conn, tlsConfig)
		if e = secure.HandshakeContext(ctx); e != nil {
			return errors.New("SMTP TLS 인증서 검증에 실패했습니다")
		}
		conn = secure
	} else if mode != "starttls" && (mode != "plaintext" || !boolean(c.Config, "allow_plaintext")) {
		return jobPermanent("SMTP 보안 모드를 확인하세요")
	}
	client, e := smtp.NewClient(conn, host)
	if e != nil {
		return errors.New("SMTP 서버 인사를 읽지 못했습니다")
	}
	defer client.Close()
	if mode == "starttls" {
		if e = client.StartTLS(tlsConfig); e != nil {
			return errors.New("SMTP STARTTLS에 실패했습니다")
		}
	}
	if username := str(c.Secrets, "username"); username != "" {
		if mode == "plaintext" {
			return jobPermanent("SMTP 인증 비밀은 TLS 연결에서만 전송합니다")
		}
		if e = client.Auth(smtp.PlainAuth("", username, str(c.Secrets, "password"), host)); e != nil {
			return jobPermanent("SMTP 인증에 실패했습니다")
		}
	}
	if e = client.Mail(str(c.Config, "from")); e != nil {
		return errors.New("SMTP 발신 주소가 거부되었습니다")
	}
	if e = client.Rcpt(recipient); e != nil {
		return errors.New("SMTP 수신 주소가 거부되었습니다")
	}
	writer, e := client.Data()
	if e != nil {
		return errors.New("SMTP DATA를 시작하지 못했습니다")
	}
	message := "From: " + str(c.Config, "from") + "\r\nTo: " + recipient + "\r\nSubject: " + mime.BEncoding.Encode("utf-8", "madi 새 알림") + "\r\nDate: " + time.Now().Format(time.RFC1123Z) + "\r\nMessage-ID: <" + id + "@madi>\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n" + strings.ReplaceAll(text, "\n", "\r\n") + "\r\n"
	if _, e = io.WriteString(writer, message); e != nil {
		return errors.New("SMTP 본문 전송이 완료되지 않았습니다")
	}
	if e = writer.Close(); e != nil {
		return errors.New("SMTP 전송 확인을 받지 못했습니다. 중복 전달 가능성이 있습니다")
	}
	_ = client.Quit()
	return nil
}
