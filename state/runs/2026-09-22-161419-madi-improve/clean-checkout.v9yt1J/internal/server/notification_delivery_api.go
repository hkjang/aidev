package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
)

//go:embed notification_delivery.sql
var notificationDeliverySchema string

func (s *Server) migrateNotificationDelivery(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, notificationDeliverySchema)
	return e
}

type notificationChannel struct {
	ID, Name, Kind, WorkspaceID string
	Config, Secrets             map[string]any
	Enabled                     bool
	Revision                    int
}

func (s *Server) notificationChannel(ctx context.Context, id string) (notificationChannel, error) {
	c := notificationChannel{}
	var raw []byte
	var encrypted string
	e := s.DB.QueryRow(ctx, "SELECT id::text,name,kind,coalesce(workspace_id::text,''),config,secret_ciphertext,enabled,revision FROM notification_channels WHERE id=$1", id).Scan(&c.ID, &c.Name, &c.Kind, &c.WorkspaceID, &raw, &encrypted, &c.Enabled, &c.Revision)
	if e != nil {
		return c, e
	}
	if json.Unmarshal(raw, &c.Config) != nil {
		return c, errors.New("채널 설정을 읽지 못했습니다")
	}
	plain, e := s.decrypt(encrypted)
	if e != nil || json.Unmarshal([]byte(plain), &c.Secrets) != nil {
		return c, errors.New("채널 비밀을 해독하지 못했습니다")
	}
	return c, nil
}
func notificationChannelPublic(c notificationChannel, admin bool) map[string]any {
	out := map[string]any{"id": c.ID, "name": c.Name, "kind": c.Kind, "workspace_id": c.WorkspaceID, "enabled": c.Enabled, "revision": c.Revision}
	if admin {
		out["config"] = c.Config
		out["secrets"] = map[string]bool{"username_configured": str(c.Secrets, "username") != "", "password_configured": str(c.Secrets, "password") != "", "url_configured": str(c.Secrets, "url") != "", "signing_secret_configured": str(c.Secrets, "signing_secret") != ""}
		if u, e := url.Parse(str(c.Secrets, "url")); e == nil {
			out["endpoint_host"] = u.Hostname()
		}
	}
	return out
}
func validNotificationAddress(value string) bool {
	address, e := mail.ParseAddress(value)
	return e == nil && address.Address == value && !strings.ContainsAny(value, "\r\n") && len(value) <= 320
}
func notificationHostList(input any) ([]string, error) {
	hosts := listStrings(input)
	if len(hosts) > 100 {
		return nil, errors.New("최대 100개 호스트를 허용할 수 있습니다")
	}
	for i, h := range hosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" || len(h) > 253 || strings.ContainsAny(h, "/:@*\\ \t\r\n") && net.ParseIP(h) == nil {
			return nil, errors.New("와일드카드 없이 정확한 호스트를 입력하세요")
		}
		hosts[i] = h
	}
	return hosts, nil
}
func validateNotificationChannel(c notificationChannel) error {
	if c.Name == "" || len(c.Name) > 120 || !oneOf(c.Kind, "smtp", "slack", "teams", "mattermost", "webhook") {
		return errors.New("채널 이름과 종류를 확인하세요")
	}
	if c.WorkspaceID != "" && !validID(c.WorkspaceID) {
		return errors.New("워크스페이스 ID를 확인하세요")
	}
	for key, v := range c.Config {
		switch key {
		case "host", "from", "tls_mode", "ca_pem":
			if _, ok := v.(string); !ok || len(str(c.Config, key)) > 65536 {
				return errors.New("채널 문자 설정을 확인하세요")
			}
		case "port":
			if n, ok := v.(float64); !ok || n != float64(int(n)) || n < 1 || n > 65535 {
				return errors.New("포트를 확인하세요")
			}
		case "allow_http", "insecure_tls", "allow_plaintext":
			if _, ok := v.(bool); !ok {
				return errors.New("채널 보안 옵션은 참/거짓이어야 합니다")
			}
		default:
			return errors.New("지원하지 않는 채널 설정입니다")
		}
	}
	if c.Kind == "smtp" {
		if !validNotificationAddress(str(c.Config, "from")) {
			return errors.New("발신 이메일 주소를 확인하세요")
		}
		if _, e := notificationHostList([]any{str(c.Config, "host")}); e != nil {
			return e
		}
		mode := str(c.Config, "tls_mode")
		if !oneOf(mode, "tls", "starttls", "plaintext") || mode == "plaintext" && !boolean(c.Config, "allow_plaintext") {
			return errors.New("TLS 또는 STARTTLS를 선택하세요. 평문은 사내 테스트에서만 명시 허용해야 합니다")
		}
	} else {
		u, e := connectorURL(str(c.Secrets, "url"), "")
		if e != nil {
			return errors.New("채널 Webhook URL을 입력하세요")
		}
		if u.Scheme == "http" && !boolean(c.Config, "allow_http") {
			return errors.New("사내 HTTP를 명시적으로 허용해야 합니다")
		}
		if c.Kind == "webhook" && len(str(c.Secrets, "signing_secret")) < 32 {
			return errors.New("일반 Webhook 서명 비밀은 32자 이상이어야 합니다")
		}
	}
	return nil
}
func (s *Server) registerNotificationDelivery() {
	s.admin("GET /api/v1/admin/notification-settings", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.one(r.Context(), "SELECT to_jsonb(x) FROM notification_settings x WHERE id=1")
		respond(w, v, e)
	})
	s.admin("PUT /api/v1/admin/notification-settings", func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		if decode(r, &in) != nil {
			apiError(w, 400, "알림 정책을 확인하세요")
			return
		}
		hosts, e := notificationHostList(in["allowed_hosts"])
		if e != nil {
			apiError(w, 400, e.Error())
			return
		}
		_, e = s.DB.Exec(r.Context(), "UPDATE notification_settings SET enabled=$1,allowed_hosts=$2 WHERE id=1", boolean(in, "enabled"), jsonValue(hosts))
		if e == nil {
			s.audit(r, "NOTIFICATION_POLICY", "", map[string]any{"enabled": boolean(in, "enabled")})
		}
		respond(w, map[string]bool{"ok": e == nil}, e)
	})
	s.admin("GET /api/v1/admin/notification-channels", s.listNotificationChannels)
	s.admin("POST /api/v1/admin/notification-channels", s.saveNotificationChannel)
	s.admin("PUT /api/v1/admin/notification-channels/{id}", s.saveNotificationChannel)
	s.handle("GET /api/v1/notification-preferences", s.notificationPreferences)
	s.handle("PUT /api/v1/notification-preferences/{id}", s.saveNotificationPreference)
	s.handle("GET /api/v1/notification-deliveries", s.notificationDeliveries)
	s.RegisterJobHandler("notification.deliver", s.deliverNotification)
}
func (s *Server) listNotificationChannels(w http.ResponseWriter, r *http.Request) {
	rows, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',id) FROM notification_channels ORDER BY name")
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []map[string]any{}
	for _, v := range rows {
		c, e := s.notificationChannel(r.Context(), str(v, "id"))
		if e != nil {
			respond(w, nil, e)
			return
		}
		out = append(out, notificationChannelPublic(c, true))
	}
	respond(w, out, nil)
}
func (s *Server) saveNotificationChannel(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "채널 설정을 확인하세요")
		return
	}
	id := r.PathValue("id")
	c := notificationChannel{ID: id, Name: str(in, "name"), Kind: str(in, "kind"), WorkspaceID: str(in, "workspace_id"), Enabled: boolean(in, "enabled"), Secrets: map[string]any{}}
	c.Config, _ = in["config"].(map[string]any)
	if c.Config == nil {
		c.Config = map[string]any{}
	}
	if id != "" {
		old, e := s.notificationChannel(r.Context(), id)
		if e != nil {
			apiError(w, 404, "채널이 없습니다")
			return
		}
		if number(in, "revision", 0) != old.Revision {
			apiError(w, 409, "다른 관리자가 채널을 변경했습니다")
			return
		}
		if c.Kind != old.Kind || c.WorkspaceID != old.WorkspaceID {
			apiError(w, 400, "종류와 대상 범위를 변경하려면 새 채널을 만드세요")
			return
		}
		c.Secrets = old.Secrets
		c.Revision = old.Revision
	} else {
		c.ID = newID()
	}
	secrets, _ := in["secrets"].(map[string]any)
	for key, v := range secrets {
		value, ok := v.(string)
		if !ok || !oneOf(key, "username", "password", "url", "signing_secret") || len(value) > 16384 || strings.ContainsAny(value, "\r\n") {
			apiError(w, 400, "채널 비밀 형식을 확인하세요")
			return
		}
		if value != "" {
			c.Secrets[key] = value
		}
	}
	if e := validateNotificationChannel(c); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	cipher, e := s.encrypt(string(jsonValue(c.Secrets)))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if id == "" {
		_, e = s.DB.Exec(r.Context(), "INSERT INTO notification_channels(id,name,kind,workspace_id,config,secret_ciphertext,enabled) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7)", c.ID, c.Name, c.Kind, c.WorkspaceID, jsonValue(c.Config), cipher, c.Enabled)
	} else {
		tag, err := s.DB.Exec(r.Context(), "UPDATE notification_channels SET name=$2,config=$3,secret_ciphertext=$4,enabled=$5,revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$6", id, c.Name, jsonValue(c.Config), cipher, c.Enabled, c.Revision)
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			apiError(w, 409, "채널이 변경되었습니다")
			return
		}
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "NOTIFICATION_CHANNEL_CONFIGURE", c.ID, map[string]any{"kind": c.Kind, "enabled": c.Enabled})
	c, e = s.notificationChannel(r.Context(), c.ID)
	respond(w, notificationChannelPublic(c, true), e)
}
func notificationHuman(w http.ResponseWriter, r *http.Request) bool {
	p := current(r)
	if p.TokenID != "" || p.ScopeRestricted || p.Kind != "user" {
		apiError(w, 403, "알림 채널은 개인 설정 화면에서 선택하세요")
		return false
	}
	return true
}
func (s *Server) notificationPreferences(w http.ResponseWriter, r *http.Request) {
	if !notificationHuman(w, r) {
		return
	}
	rows, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',c.id,'name',c.name,'kind',c.kind,'workspace_id',c.workspace_id,'available',c.enabled,'enabled',coalesce(p.enabled,false)) FROM notification_channels c LEFT JOIN notification_preferences p ON p.channel_id=c.id AND p.user_id=$1 WHERE c.workspace_id IS NULL OR EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id=c.workspace_id AND m.user_id=$1) ORDER BY c.name`, current(r).ID)
	respond(w, rows, e)
}
func (s *Server) saveNotificationPreference(w http.ResponseWriter, r *http.Request) {
	if !notificationHuman(w, r) {
		return
	}
	c, e := s.notificationChannel(r.Context(), r.PathValue("id"))
	if e != nil || c.WorkspaceID != "" && !s.canWorkspace(r.Context(), current(r), c.WorkspaceID, false) {
		apiError(w, 404, "선택 가능한 채널이 없습니다")
		return
	}
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "채널 선택을 확인하세요")
		return
	}
	_, e = s.DB.Exec(r.Context(), "INSERT INTO notification_preferences(user_id,channel_id,enabled) VALUES($1,$2,$3) ON CONFLICT(user_id,channel_id) DO UPDATE SET enabled=excluded.enabled", current(r).ID, c.ID, boolean(in, "enabled"))
	respond(w, map[string]bool{"ok": e == nil}, e)
}
func (s *Server) notificationDeliveries(w http.ResponseWriter, r *http.Request) {
	if !notificationHuman(w, r) {
		return
	}
	v, e := s.rows(r.Context(), `SELECT to_jsonb(d)||jsonb_build_object('channel_name',c.name,'kind',c.kind,'job_status',j.status,'attempts',j.attempts) FROM notification_deliveries d JOIN notification_channels c ON c.id=d.channel_id LEFT JOIN automation_jobs j ON j.id=d.job_id WHERE d.user_id=$1 ORDER BY d.created_at DESC LIMIT 100`, current(r).ID)
	respond(w, v, e)
}
