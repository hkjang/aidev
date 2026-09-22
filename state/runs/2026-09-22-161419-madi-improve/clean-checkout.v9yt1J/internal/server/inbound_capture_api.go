package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type inboundRateLimit struct{}

func (inboundRateLimit) Error() string {
	return "채널당 분당 60건까지 수집할 수 있습니다"
}

func (s *Server) requestedInboundChannel(w http.ResponseWriter, r *http.Request) (inboundChannel, bool) {
	c, e := s.inboundChannel(r.Context(), r.PathValue("id"))
	if e != nil || c.UserID != current(r).ID || current(r).TokenID != "" || current(r).ScopeRestricted || !s.canWorkspace(r.Context(), current(r), c.WorkspaceID, false) {
		apiError(w, 404, "내 수집 채널을 찾을 수 없습니다")
		return c, false
	}
	return c, true
}
func (s *Server) listInboundChannels(w http.ResponseWriter, r *http.Request) {
	if !notificationHuman(w, r) {
		return
	}
	rows, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',id) FROM inbound_capture_channels WHERE user_id=$1 ORDER BY name", current(r).ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []map[string]any{}
	for _, row := range rows {
		c, e := s.inboundChannel(r.Context(), str(row, "id"))
		if e != nil {
			respond(w, nil, e)
			return
		}
		if s.canWorkspace(r.Context(), current(r), c.WorkspaceID, false) {
			out = append(out, inboundChannelPublic(c))
		}
	}
	respond(w, out, nil)
}
func validateInboundChannel(c inboundChannel) error {
	if c.Name == "" || len(c.Name) > 120 || !oneOf(c.Kind, "imap", "hmac") {
		return errors.New("수집 채널 이름과 종류를 확인하세요")
	}
	for key, v := range c.Config {
		switch key {
		case "host", "mailbox", "ca_pem":
			if _, ok := v.(string); !ok || len(str(c.Config, key)) > 65536 {
				return errors.New("메일 연결 문자열을 확인하세요")
			}
			if key != "ca_pem" && strings.ContainsAny(str(c.Config, key), "\r\n") {
				return errors.New("메일 연결 문자열을 확인하세요")
			}
		case "port", "interval_minutes":
			n, ok := v.(float64)
			if !ok || n != float64(int(n)) {
				return errors.New("수집 연결 숫자 설정을 확인하세요")
			}
		case "insecure_tls", "from_now":
			if _, ok := v.(bool); !ok {
				return errors.New("수집 옵션은 참/거짓이어야 합니다")
			}
		default:
			return errors.New("지원하지 않는 수집 설정입니다")
		}
	}
	if c.Kind == "imap" {
		if _, e := notificationHostList([]any{str(c.Config, "host")}); e != nil {
			return e
		}
		if n := number(c.Config, "port", 993); n < 1 || n > 65535 {
			return errors.New("IMAP 포트를 확인하세요")
		}
		if n := number(c.Config, "interval_minutes", 5); n < 1 || n > 1440 {
			return errors.New("메일 수집 주기는 1~1440분입니다")
		}
		if str(c.Config, "mailbox") == "" || len(str(c.Config, "mailbox")) > 500 {
			return errors.New("IMAP 폴더를 입력하세요")
		}
		if str(c.Secrets, "username") == "" || str(c.Secrets, "password") == "" {
			return errors.New("메일 계정과 비밀번호 또는 앱 암호가 필요합니다")
		}
	}
	return nil
}
func (s *Server) saveInboundChannel(w http.ResponseWriter, r *http.Request) {
	if !notificationHuman(w, r) {
		return
	}
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "수집 채널 설정을 확인하세요")
		return
	}
	c := inboundChannel{ID: r.PathValue("id"), UserID: current(r).ID, WorkspaceID: str(in, "workspace_id"), Name: str(in, "name"), Kind: str(in, "kind"), Enabled: boolean(in, "enabled"), Secrets: map[string]any{}}
	c.Config, _ = in["config"].(map[string]any)
	if c.Config == nil {
		c.Config = map[string]any{}
	}
	if !s.canWorkspace(r.Context(), current(r), c.WorkspaceID, true) {
		apiError(w, 403, "수집 대상 워크스페이스의 작성 권한이 없습니다")
		return
	}
	created := c.ID == ""
	if created && c.Kind == "imap" {
		if _, supplied := c.Config["from_now"]; !supplied {
			c.Config["from_now"] = true
		}
	}
	signingSecret := ""
	if !created {
		old, ok := s.requestedInboundChannel(w, r)
		if !ok {
			return
		}
		if old.Kind != c.Kind || old.WorkspaceID != c.WorkspaceID {
			apiError(w, 400, "종류나 워크스페이스를 바꾸려면 새 수집 채널을 만드세요")
			return
		}
		if old.Revision != number(in, "revision", 0) {
			apiError(w, 409, "수집 채널이 변경되었습니다")
			return
		}
		if c.Kind == "imap" {
			secrets, _ := in["secrets"].(map[string]any)
			if str(old.Config, "host") != str(c.Config, "host") || number(old.Config, "port", 993) != number(c.Config, "port", 993) || str(old.Config, "mailbox") != str(c.Config, "mailbox") || (str(secrets, "username") != "" && str(secrets, "username") != str(old.Secrets, "username")) {
				apiError(w, 400, "메일 서버·계정·폴더를 바꾸려면 새 수집 채널을 만드세요")
				return
			}
		}
		c.Secrets = old.Secrets
		c.Revision = old.Revision
	} else {
		c.ID = newID()
		if c.Kind == "hmac" {
			signingSecret = integrationSecret()
			c.Secrets["signing_secret"] = signingSecret
		}
	}
	secrets, _ := in["secrets"].(map[string]any)
	for key, v := range secrets {
		value, ok := v.(string)
		if !ok || !oneOf(key, "username", "password") || len(value) > 16384 || strings.ContainsAny(value, "\r\n") {
			apiError(w, 400, "수집 비밀 형식을 확인하세요")
			return
		}
		if value != "" {
			c.Secrets[key] = value
		}
	}
	if e := validateInboundChannel(c); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	cipher, e := s.encrypt(string(jsonValue(c.Secrets)))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if created {
		_, e = s.DB.Exec(r.Context(), "INSERT INTO inbound_capture_channels(id,user_id,workspace_id,name,kind,config,secret_ciphertext,enabled,next_run) VALUES($1,$2,$3,$4,$5,$6,$7,$8,CASE WHEN $5='imap' AND $8 THEN now() ELSE NULL END)", c.ID, c.UserID, c.WorkspaceID, c.Name, c.Kind, jsonValue(c.Config), cipher, c.Enabled)
	} else {
		tag, err := s.DB.Exec(r.Context(), "UPDATE inbound_capture_channels SET name=$2,config=$3,secret_ciphertext=$4,enabled=$5,revision=revision+1,next_run=CASE WHEN kind='imap' AND $5 THEN now() ELSE NULL END,updated_at=now() WHERE id=$1 AND revision=$6", c.ID, c.Name, jsonValue(c.Config), cipher, c.Enabled, c.Revision)
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			apiError(w, 409, "수집 채널이 변경되었습니다")
			return
		}
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	c, e = s.inboundChannel(r.Context(), c.ID)
	out := inboundChannelPublic(c)
	if signingSecret != "" {
		out["signing_secret"] = signingSecret
	}
	s.audit(r, "INBOUND_CHANNEL_CONFIGURE", c.ID, map[string]any{"kind": c.Kind, "enabled": c.Enabled})
	respond(w, out, e)
}
func (s *Server) rotateInboundSecret(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedInboundChannel(w, r)
	if !ok {
		return
	}
	if c.Kind != "hmac" {
		apiError(w, 400, "Webhook 수집 채널만 서명 비밀을 회전합니다")
		return
	}
	var in map[string]any
	if decode(r, &in) != nil || number(in, "revision", 0) != c.Revision {
		apiError(w, 409, "현재 채널 버전을 확인하세요")
		return
	}
	secret := integrationSecret()
	c.Secrets["signing_secret"] = secret
	cipher, e := s.encrypt(string(jsonValue(c.Secrets)))
	if e != nil {
		respond(w, nil, e)
		return
	}
	tag, e := s.DB.Exec(r.Context(), "UPDATE inbound_capture_channels SET secret_ciphertext=$2,revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$3", c.ID, cipher, c.Revision)
	if e == nil && tag.RowsAffected() != 1 {
		apiError(w, 409, "채널이 변경되었습니다")
		return
	}
	s.audit(r, "INBOUND_SECRET_ROTATE", c.ID, nil)
	respond(w, map[string]any{"signing_secret": secret, "revision": c.Revision + 1}, e)
}
func (s *Server) inboundCaptureHistory(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedInboundChannel(w, r)
	if !ok {
		return
	}
	v, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',m.id,'message_key',m.message_key,'message_id',m.message_id,'title',m.title,'status',m.status,'message',m.message,'document_id',CASE WHEN m.document_id IS NOT NULL AND madi_document_allowed($2,m.document_id,false) THEN m.document_id ELSE NULL END,'created_at',m.created_at,'completed_at',m.completed_at,'job_status',j.status,'attempts',j.attempts) FROM inbound_capture_messages m LEFT JOIN automation_jobs j ON j.id=m.job_id WHERE m.channel_id=$1 ORDER BY m.created_at DESC LIMIT 100`, c.ID, current(r).ID)
	respond(w, v, e)
}
func (s *Server) cancelInboundMessage(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedInboundChannel(w, r)
	if !ok {
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var jid string
	e = tx.QueryRow(r.Context(), "UPDATE inbound_capture_messages SET status='cancelled',message='사용자가 수집을 취소했습니다',completed_at=now() WHERE id=$1 AND channel_id=$2 AND status='pending' RETURNING coalesce(job_id::text,'')", r.PathValue("message"), c.ID).Scan(&jid)
	if e != nil {
		apiError(w, 409, "수집이 완료되었거나 취소되었습니다")
		return
	}
	if jid != "" {
		_, e = tx.Exec(r.Context(), "UPDATE automation_jobs SET cancel_requested=true,updated_at=now() WHERE id=$1 AND owner_id=$2 AND status IN ('pending','running')", jid, c.UserID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]bool{"ok": e == nil}, e)
}
func (s *Server) receiveCaptureHook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "수집 채널이 없습니다")
		return
	}
	c, e := s.inboundChannel(r.Context(), id)
	var enabled bool
	_ = s.DB.QueryRow(r.Context(), "SELECT hooks_enabled FROM inbound_capture_settings WHERE id=1").Scan(&enabled)
	if e != nil || c.Kind != "hmac" || !c.Enabled || !enabled {
		apiError(w, 404, "수집 채널이 없습니다")
		return
	}
	if r.ContentLength > inboundCaptureMax {
		apiError(w, 413, "Webhook 수집 요청은 10MB 이하여야 합니다")
		return
	}
	// A public signed endpoint must not keep a connection indefinitely while
	// waiting for the body needed to verify its signature.
	controller := http.NewResponseController(w)
	_ = controller.SetReadDeadline(time.Now().Add(30 * time.Second))
	defer controller.SetReadDeadline(time.Time{})
	if !validID(r.Header.Get("X-Madi-ID")) || len(r.Header.Get("X-Madi-Signature")) != 71 || len(r.Header.Get("X-Madi-Timestamp")) > 16 {
		apiError(w, 401, "수집 서명 또는 타임스탬프가 올바르지 않습니다")
		return
	}
	body, e := io.ReadAll(io.LimitReader(r.Body, inboundCaptureMax+1))
	if e != nil || len(body) > inboundCaptureMax {
		apiError(w, 413, "Webhook 본문 크기를 확인하세요")
		return
	}
	deliveryID := r.Header.Get("X-Madi-ID")
	if !VerifyWebhookSignature(str(c.Secrets, "signing_secret"), r.Header.Get("X-Madi-Timestamp"), deliveryID, r.Header.Get("X-Madi-Signature"), body, time.Now()) {
		apiError(w, 401, "수집 서명 또는 타임스탬프가 올바르지 않습니다")
		return
	}
	var in struct {
		Title       string              `json:"title"`
		Text        string              `json:"text"`
		URL         string              `json:"url"`
		Attachments []inboundAttachment `json:"attachments"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&in) != nil || decoder.Decode(new(any)) != io.EOF {
		apiError(w, 400, "수집 JSON 형식을 확인하세요")
		return
	}
	if in.URL != "" {
		if _, e := connectorURL(in.URL, ""); e != nil {
			apiError(w, 400, "수집 출처 URL을 확인하세요")
			return
		}
	}
	content := inboundContent{Title: in.Title, Markdown: in.Text, Attachments: in.Attachments, Source: map[string]any{"origin": "webhook", "source_url": in.URL}}
	if in.URL != "" {
		content.Markdown += "\n\n출처: " + in.URL
	}
	mid, duplicate, e := s.stageInboundContent(r.Context(), c, "hook:"+deliveryID, content)
	if e != nil {
		var limited inboundRateLimit
		if errors.As(e, &limited) {
			apiError(w, 429, e.Error())
			return
		}
		apiError(w, 409, "수집 내용을 확인하세요. 같은 ID의 내용 변경, 대기 한도 초과 또는 현재 채널·작성 권한 변경은 수집할 수 없습니다")
		return
	}
	status := "accepted"
	if duplicate {
		_ = s.DB.QueryRow(r.Context(), "SELECT status FROM inbound_capture_messages WHERE id=$1", mid).Scan(&status)
	}
	jsonResponse(w, 202, map[string]any{"id": mid, "duplicate": duplicate, "status": status})
}
