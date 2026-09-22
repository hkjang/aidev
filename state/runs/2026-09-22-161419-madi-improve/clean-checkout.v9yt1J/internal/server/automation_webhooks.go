package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WebhookSignature signs timestamp.deliveryID.body. Receivers should additionally
// remember deliveryID after a successful transaction: retries reuse the ID.
func WebhookSignature(secret, timestamp, deliveryID string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + deliveryID + "."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
func VerifyWebhookSignature(secret, timestamp, deliveryID, signature string, body []byte, now time.Time) bool {
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || !validID(deliveryID) {
		return false
	}
	age := now.Sub(time.Unix(seconds, 0))
	if age > 5*time.Minute || age < -5*time.Minute {
		return false
	}
	expected := WebhookSignature(secret, timestamp, deliveryID, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}
func webhookDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("Webhook 주소 형식 오류")
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, errors.New("Webhook DNS 조회 실패")
	}
	// Private RFC1918 and loopback addresses are intentional for air-gapped LANs.
	// Link-local metadata endpoints, multicast and unspecified addresses are not.
	for _, address := range addresses {
		ip := address.IP
		if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return nil, errors.New("Webhook은 링크 로컬·메타데이터 주소를 사용할 수 없습니다")
		}
	}
	dialer := net.Dialer{Timeout: 10 * time.Second}
	for _, address := range addresses {
		conn, e := dialer.DialContext(ctx, network, net.JoinHostPort(address.IP.String(), port))
		if e == nil {
			return conn, nil
		}
	}
	return nil, errors.New("Webhook 연결 실패")
}
func (s *Server) deliverWebhook(ctx context.Context, j Job) (map[string]any, error) {
	var wid, ownerID, url, secretCipher string
	var enabled bool
	var timeout int
	var events []string
	err := s.DB.QueryRow(ctx, `SELECT workspace_id::text,owner_id::text,url,secret_ciphertext,enabled,timeout_seconds,ARRAY(SELECT jsonb_array_elements_text(events)) FROM automation_webhooks WHERE id=$1`, str(j.Payload, "webhook_id")).Scan(&wid, &ownerID, &url, &secretCipher, &enabled, &timeout, &events)
	if err != nil || !enabled || wid != j.WorkspaceID {
		return nil, jobPermanent("Webhook이 삭제 또는 비활성화되었거나 공간이 다릅니다")
	}
	owner, err := s.workerPrincipal(ctx, ownerID, "", wid)
	if err != nil {
		return nil, err
	}
	if !s.automationManager(ctx, owner, wid) {
		return nil, jobPermanent("Webhook 소유자의 현재 관리 권한이 없습니다")
	}
	actor, err := s.workerPrincipal(ctx, j.ActorID, j.TokenID, wid)
	if err != nil {
		return nil, err
	}
	event := Event{ID: newID(), Type: "webhook.test", WorkspaceID: wid, ActorID: j.ActorID, CreatedAt: time.Now().UTC()}
	if j.EventID != "" {
		event, err = s.eventByID(ctx, j.EventID)
		if err != nil {
			return nil, err
		}
		// An explicit automation action chooses the configured endpoint; ordinary
		// subscriptions must still match its current event filter at delivery time.
		if j.AutomationID == "" {
			matched := false
			for _, kind := range events {
				if kind == event.Type {
					matched = true
				}
			}
			if !matched {
				return nil, jobPermanent("Webhook 이벤트 구독이 변경되었습니다")
			}
		}
		if !s.automationSourceAccess(ctx, owner, event, false) || !s.automationSourceAccess(ctx, actor, event, false) {
			return nil, jobPermanent("Webhook 원본 문서 접근 권한이 회수되었습니다")
		}
		scope := "document:read"
		if event.ResourceType == "database" {
			scope = "database:read"
		}
		if !jobScope(actor, scope) {
			return nil, jobPermanent("API 키의 Webhook 원본 조회 범위가 없습니다")
		}
	} else if !s.automationManager(ctx, actor, wid) {
		return nil, jobPermanent("Webhook 테스트는 관리자만 실행합니다")
	}
	if !validWebhookURL(url) {
		return nil, jobPermanent("Webhook 주소를 확인하세요")
	}
	secret, err := s.decrypt(secretCipher)
	if err != nil {
		return nil, jobPermanent("Webhook 서명 비밀을 복호화할 수 없습니다")
	}
	body, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	request, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, jobPermanent("Webhook 요청 구성 실패")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "madi-webhook/"+s.Version)
	request.Header.Set("X-Madi-Delivery", j.ID)
	request.Header.Set("X-Madi-Timestamp", stamp)
	request.Header.Set("X-Madi-Signature", WebhookSignature(secret, stamp, j.ID, body))
	request.Header.Set("X-Madi-Event", event.Type)
	transport := &http.Transport{DialContext: webhookDial, ResponseHeaderTimeout: time.Duration(timeout) * time.Second, TLSHandshakeTimeout: 10 * time.Second, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Duration(timeout) * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("Webhook 연결 또는 응답 시간 초과")
	}
	defer response.Body.Close()
	// Do not persist response bodies/headers: receivers may accidentally echo secrets.
	n, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, (64<<10)+1))
	result := map[string]any{"http_status": response.StatusCode, "delivery_id": j.ID, "response_bytes": n}
	if n > 64<<10 {
		return result, jobPermanent("Webhook 응답 본문이 64KB 제한을 초과했습니다")
	}
	if readErr != nil {
		return result, errors.New("Webhook 응답 수신 실패")
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return result, nil
	}
	message := fmt.Sprintf("Webhook 수신 서버 HTTP %d", response.StatusCode)
	if response.StatusCode == 408 || response.StatusCode == 429 || response.StatusCode >= 500 {
		return result, errors.New(message)
	}
	return result, jobPermanent(message)
}
