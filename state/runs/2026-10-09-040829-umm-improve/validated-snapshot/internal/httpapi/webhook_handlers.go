package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/webhook"
)

func newWebhookSecret() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (s *Server) listWebhooks(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "webhooks:write") {
		return
	}
	p := principal(r)
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT id,name,url,events,active,failure_count,last_delivered_at,last_error,created_at,updated_at FROM webhook_subscriptions WHERE owner_id=$1 ORDER BY created_at DESC`, p.User.ID)
	if err != nil {
		writeError(w, 500, "웹훅 목록을 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var name, target, lastError string
		var events []string
		var active bool
		var failures int
		var delivered *time.Time
		var created, updated time.Time
		if err := rows.Scan(&id, &name, &target, &events, &active, &failures, &delivered, &lastError, &created, &updated); err != nil {
			writeError(w, 500, "웹훅 목록을 읽지 못했습니다.")
			return
		}
		items = append(items, map[string]any{"id": id, "name": name, "url": target, "events": events, "active": active, "failureCount": failures, "lastDeliveredAt": delivered, "lastError": lastError, "createdAt": created, "updatedAt": updated})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "웹훅 목록을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"webhooks": items, "supportedEvents": webhook.SupportedEvents})
}

func (s *Server) createWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "webhooks:write") {
		return
	}
	var body struct {
		Name   string   `json:"name"`
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, 400, "웹훅 형식이 올바르지 않습니다.")
		return
	}
	body.Name, body.URL = strings.TrimSpace(body.Name), strings.TrimSpace(body.URL)
	if utf8.RuneCountInString(body.Name) < 1 || utf8.RuneCountInString(body.Name) > 200 || !webhook.ValidateEvents(body.Events) {
		writeError(w, 400, "웹훅 이름 또는 이벤트 목록이 올바르지 않습니다.")
		return
	}
	if err := webhook.ValidateEndpoint(r.Context(), body.URL); err != nil {
		writeError(w, 400, "안전한 웹훅 주소가 아닙니다: "+err.Error())
		return
	}
	secret, err := newWebhookSecret()
	if err != nil {
		writeError(w, 500, "웹훅 서명 키를 만들지 못했습니다.")
		return
	}
	ciphertext, err := s.Cipher.Encrypt(secret)
	if err != nil {
		writeError(w, 500, "웹훅 서명 키를 암호화하지 못했습니다.")
		return
	}
	p := principal(r)
	var id uuid.UUID
	err = s.Store.Pool.QueryRow(r.Context(), `INSERT INTO webhook_subscriptions(owner_id,name,url,secret_ciphertext,events) VALUES($1,$2,$3,$4,$5) RETURNING id`, p.User.ID, body.Name, body.URL, ciphertext, body.Events).Scan(&id)
	if err != nil {
		writeError(w, 500, "웹훅을 저장하지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "webhook.create", "webhook", id.String(), map[string]any{"events": body.Events})
	writeJSON(w, 201, map[string]any{"id": id, "secret": secret, "warning": "서명 키는 다시 표시되지 않습니다."})
}

func (s *Server) updateWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "webhooks:write") {
		return
	}
	id, ok := parseID(w, r, "webhookID")
	if !ok {
		return
	}
	var body struct {
		Name   string   `json:"name"`
		URL    string   `json:"url"`
		Events []string `json:"events"`
		Active bool     `json:"active"`
	}
	if decodeJSON(w, r, &body) != nil || !webhook.ValidateEvents(body.Events) {
		writeError(w, 400, "웹훅 설정이 올바르지 않습니다.")
		return
	}
	body.Name, body.URL = strings.TrimSpace(body.Name), strings.TrimSpace(body.URL)
	if utf8.RuneCountInString(body.Name) < 1 || utf8.RuneCountInString(body.Name) > 200 {
		writeError(w, 400, "웹훅 이름은 1~200자로 입력해 주세요.")
		return
	}
	if err := webhook.ValidateEndpoint(r.Context(), body.URL); err != nil {
		writeError(w, 400, "안전한 웹훅 주소가 아닙니다: "+err.Error())
		return
	}
	p := principal(r)
	command, err := s.Store.Pool.Exec(r.Context(), `UPDATE webhook_subscriptions SET name=$3,url=$4,events=$5,active=$6,failure_count=CASE WHEN $6 AND NOT active THEN 0 ELSE failure_count END,last_error=CASE WHEN $6 AND NOT active THEN '' ELSE last_error END,updated_at=now() WHERE id=$1 AND owner_id=$2`, id, p.User.ID, body.Name, body.URL, body.Events, body.Active)
	if err != nil {
		writeError(w, 500, "웹훅을 수정하지 못했습니다.")
		return
	}
	if command.RowsAffected() == 0 {
		writeError(w, 404, "웹훅을 찾을 수 없습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "webhook.update", "webhook", id.String(), map[string]any{"events": body.Events, "active": body.Active})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "webhooks:write") {
		return
	}
	id, ok := parseID(w, r, "webhookID")
	if !ok {
		return
	}
	p := principal(r)
	command, err := s.Store.Pool.Exec(r.Context(), `DELETE FROM webhook_subscriptions WHERE id=$1 AND owner_id=$2`, id, p.User.ID)
	if err != nil {
		// Saying a webhook cannot be found when the truth is that deleting it
		// failed tells someone their notes have stopped going somewhere they
		// are still going.
		writeError(w, 500, "웹훅을 삭제하지 못했습니다.")
		return
	}
	if command.RowsAffected() == 0 {
		writeError(w, 404, "웹훅을 찾을 수 없습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "webhook.delete", "webhook", id.String(), map[string]any{})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "webhooks:write") {
		return
	}
	id, ok := parseID(w, r, "webhookID")
	if !ok {
		return
	}
	p := principal(r)
	var exists bool
	if err := s.Store.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM webhook_subscriptions WHERE id=$1 AND owner_id=$2)`, id, p.User.ID).Scan(&exists); err != nil {
		writeError(w, 500, "웹훅 상태를 확인하지 못했습니다.")
		return
	}
	if !exists || s.Webhooks == nil {
		writeError(w, 404, "시험할 웹훅을 찾을 수 없습니다.")
		return
	}
	if err := s.Webhooks.Test(r.Context(), id, p.User.ID); err != nil {
		writeError(w, 502, "웹훅 전송 시험에 실패했습니다: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) rotateWebhookSecret(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "webhooks:write") {
		return
	}
	id, ok := parseID(w, r, "webhookID")
	if !ok {
		return
	}
	secret, err := newWebhookSecret()
	if err != nil {
		writeError(w, 500, "새 웹훅 서명 키를 만들지 못했습니다.")
		return
	}
	ciphertext, err := s.Cipher.Encrypt(secret)
	if err != nil {
		writeError(w, 500, "새 웹훅 서명 키를 암호화하지 못했습니다.")
		return
	}
	p := principal(r)
	command, err := s.Store.Pool.Exec(r.Context(), `UPDATE webhook_subscriptions SET secret_ciphertext=$3,updated_at=now() WHERE id=$1 AND owner_id=$2`, id, p.User.ID, ciphertext)
	if err != nil {
		writeError(w, 500, "웹훅 비밀키를 교체하지 못했습니다.")
		return
	}
	if command.RowsAffected() == 0 {
		writeError(w, 404, "웹훅을 찾을 수 없습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "webhook.secret.rotate", "webhook", id.String(), map[string]any{})
	writeJSON(w, 200, map[string]any{"secret": secret, "warning": "새 서명 키는 다시 표시되지 않습니다."})
}
