package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type encryptedValue struct {
	kind    string
	id      string
	section string
	field   string
	value   string
}

func encryptedPayload(value string) (string, bool) {
	if strings.HasPrefix(value, "enc:") {
		return strings.TrimPrefix(value, "enc:"), true
	}
	return value, false
}

func (s *Server) encryptedValues(r *http.Request) ([]encryptedValue, error) {
	values := []encryptedValue{}
	settings, err := s.Store.AllSettings(r.Context())
	if err != nil {
		return nil, err
	}
	for section, raw := range settings {
		var setting map[string]any
		if err := json.Unmarshal(raw, &setting); err != nil {
			return nil, fmt.Errorf("decode %s setting: %w", section, err)
		}
		for _, field := range []string{"client_secret", "api_key"} {
			value, _ := setting[field].(string)
			if strings.HasPrefix(value, "enc:") {
				values = append(values, encryptedValue{kind: "setting", section: section, field: field, value: value})
			}
		}
	}
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT id,secret_ciphertext FROM webhook_subscriptions`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id uuid.UUID
		var value string
		if err := rows.Scan(&id, &value); err != nil {
			rows.Close()
			return nil, err
		}
		if value != "" {
			values = append(values, encryptedValue{kind: "webhook", id: id.String(), value: value})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	rows, err = s.Store.Pool.Query(r.Context(), `SELECT id,prompt_ciphertext FROM ai_calls WHERE prompt_ciphertext<>''`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var value string
		if err := rows.Scan(&id, &value); err != nil {
			rows.Close()
			return nil, err
		}
		values = append(values, encryptedValue{kind: "ai_prompt", id: strconv.FormatInt(id, 10), value: value})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	return values, nil
}

func (s *Server) encryptionKeyStatus(w http.ResponseWriter, r *http.Request) {
	values, err := s.encryptedValues(r)
	if err != nil {
		writeError(w, 500, "암호화 키 상태를 확인하지 못했습니다.")
		return
	}
	pending, unreadable := 0, 0
	for _, item := range values {
		payload, prefixed := encryptedPayload(item.value)
		if _, decryptErr := s.Cipher.Decrypt(payload); decryptErr != nil {
			unreadable++
			continue
		}
		if (item.kind == "ai_prompt" && !prefixed) || s.Cipher.NeedsRotation(payload) {
			pending++
		}
	}
	writeJSON(w, 200, map[string]any{"keyId": s.Cipher.KeyID(), "fallbackKeys": s.Cipher.FallbackKeyCount(), "encryptedValues": len(values), "pendingRotation": pending, "unreadable": unreadable})
}

func (s *Server) rotateEncryptionKey(w http.ResponseWriter, r *http.Request) {
	if s.Cipher.FallbackKeyCount() == 0 {
		writeError(w, 400, "먼저 새 ENCRYPTION_KEY와 기존 키를 ENCRYPTION_KEY_PREVIOUS에 설정해 재시작해 주세요.")
		return
	}
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "키 회전 트랜잭션을 시작하지 못했습니다.")
		return
	}
	defer tx.Rollback(r.Context())
	rotated := 0
	for _, section := range []string{"oidc", "ai_gateway"} {
		if err = s.Store.LockSettingTx(r.Context(), tx, section); err != nil {
			writeError(w, 500, "암호화된 "+section+" 설정을 잠그지 못했습니다.")
			return
		}
		var raw []byte
		if err = tx.QueryRow(r.Context(), `SELECT value FROM app_settings WHERE key=$1 FOR UPDATE`, section).Scan(&raw); err != nil {
			writeError(w, 500, "암호화된 "+section+" 설정을 읽지 못했습니다.")
			return
		}
		var value map[string]any
		if err = json.Unmarshal(raw, &value); err != nil {
			writeError(w, 500, section+" 설정 형식이 올바르지 않아 키 회전을 중단했습니다.")
			return
		}
		changed := false
		for _, field := range []string{"client_secret", "api_key"} {
			stored, _ := value[field].(string)
			payload, prefixed := encryptedPayload(stored)
			if !prefixed || !s.Cipher.NeedsRotation(payload) {
				continue
			}
			plain, decryptErr := s.Cipher.Decrypt(payload)
			if decryptErr != nil {
				writeError(w, 409, "기존 "+section+" 비밀 값을 복호화하지 못해 회전을 중단했습니다.")
				return
			}
			next, encryptErr := s.Cipher.Encrypt(plain)
			if encryptErr != nil {
				writeError(w, 500, "비밀 값을 새 키로 암호화하지 못했습니다.")
				return
			}
			value[field] = "enc:" + next
			changed, rotated = true, rotated+1
		}
		if changed {
			encoded, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				writeError(w, 500, "회전한 설정 값을 직렬화하지 못했습니다.")
				return
			}
			if _, err = tx.Exec(r.Context(), `UPDATE app_settings SET value=$2,updated_at=now() WHERE key=$1`, section, encoded); err != nil {
				writeError(w, 500, "설정 비밀 값을 회전하지 못했습니다.")
				return
			}
		}
	}
	type rowValue struct{ id, value string }
	webhooks := []rowValue{}
	rows, err := tx.Query(r.Context(), `SELECT id::text,secret_ciphertext FROM webhook_subscriptions FOR UPDATE`)
	if err != nil {
		writeError(w, 500, "웹훅 서명 키를 읽지 못했습니다.")
		return
	}
	for rows.Next() {
		var item rowValue
		if err = rows.Scan(&item.id, &item.value); err != nil {
			rows.Close()
			writeError(w, 500, "웹훅 서명 키를 읽지 못했습니다.")
			return
		}
		webhooks = append(webhooks, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		writeError(w, 500, "웹훅 서명 키를 읽지 못했습니다.")
		return
	}
	rows.Close()
	for _, item := range webhooks {
		if !s.Cipher.NeedsRotation(item.value) {
			continue
		}
		plain, decryptErr := s.Cipher.Decrypt(item.value)
		if decryptErr != nil {
			writeError(w, 409, "기존 웹훅 서명 키를 복호화하지 못해 회전을 중단했습니다.")
			return
		}
		next, encryptErr := s.Cipher.Encrypt(plain)
		if encryptErr != nil {
			writeError(w, 500, "웹훅 서명 키를 새 키로 암호화하지 못했습니다.")
			return
		}
		if _, err = tx.Exec(r.Context(), `UPDATE webhook_subscriptions SET secret_ciphertext=$2,updated_at=now() WHERE id=$1::uuid`, item.id, next); err != nil {
			writeError(w, 500, "웹훅 서명 키를 회전하지 못했습니다.")
			return
		}
		rotated++
	}
	prompts := []rowValue{}
	rows, err = tx.Query(r.Context(), `SELECT id::text,prompt_ciphertext FROM ai_calls WHERE prompt_ciphertext<>'' FOR UPDATE`)
	if err != nil {
		writeError(w, 500, "AI 로그 암호문을 읽지 못했습니다.")
		return
	}
	for rows.Next() {
		var item rowValue
		if err = rows.Scan(&item.id, &item.value); err != nil {
			rows.Close()
			writeError(w, 500, "AI 로그 암호문을 읽지 못했습니다.")
			return
		}
		prompts = append(prompts, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		writeError(w, 500, "AI 로그 암호문을 읽지 못했습니다.")
		return
	}
	rows.Close()
	for _, item := range prompts {
		payload, prefixed := encryptedPayload(item.value)
		if prefixed && !s.Cipher.NeedsRotation(payload) {
			continue
		}
		plain, decryptErr := s.Cipher.Decrypt(payload)
		if decryptErr != nil {
			writeError(w, 409, "기존 AI 로그를 복호화하지 못해 회전을 중단했습니다.")
			return
		}
		next, encryptErr := s.Cipher.Encrypt(plain)
		if encryptErr != nil {
			writeError(w, 500, "AI 로그를 새 키로 암호화하지 못했습니다.")
			return
		}
		if _, err = tx.Exec(r.Context(), `UPDATE ai_calls SET prompt_ciphertext=$2 WHERE id=$1::bigint`, item.id, "enc:"+next); err != nil {
			writeError(w, 500, "AI 로그 암호문을 회전하지 못했습니다.")
			return
		}
		rotated++
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "키 회전 결과를 확정하지 못했습니다.")
		return
	}
	p := principal(r)
	s.Store.Audit(r.Context(), &p.User.ID, "encryption.rotate", "security", s.Cipher.KeyID(), map[string]any{"rotated": rotated})
	writeJSON(w, 200, map[string]any{"ok": true, "rotated": rotated, "keyId": s.Cipher.KeyID()})
}
