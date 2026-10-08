package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hkjang/invenqor/server/internal/mail"
)

type settingUpdate struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	Secret    bool            `json:"secret"`
	ApplyMode string          `json:"apply_mode"`
	Reason    string          `json:"reason"`
}

const (
	keycloakDedicatedSetting       = "auth.keycloak"
	keycloakClientSecretSetting    = "auth.keycloak.client_secret"
	dedicatedSettingEndpointDetail = "This setting is managed by its dedicated administrative endpoint."
)

func (s *Server) listSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.database.DB().QueryContext(r.Context(),
		`SELECT key,value_json,secret,apply_mode,pending_value_json,
		 version,updated_by,updated_at FROM settings
		 WHERE key NOT IN ($1,$2) ORDER BY key`,
		keycloakDedicatedSetting,
		keycloakClientSecretSetting,
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var key, value, mode string
		var secret bool
		var pending, user, updated any
		var version int
		if err := rows.Scan(
			&key, &value, &secret, &mode, &pending, &version, &user, &updated,
		); err != nil {
			s.internalError(w, r, err)
			return
		}
		display := any(json.RawMessage(value))
		if secret {
			display = map[string]any{"configured": value != "", "masked": true}
		}
		items = append(items, map[string]any{
			"key": key, "value": display, "secret": secret,
			"apply_mode": mode, "pending": pending != nil, "version": version,
			"updated_by": user, "updated_at": apiTime(updated),
		})
	}
	writeJSON(w, 200, map[string]any{
		"items": items,
		"categories": []string{
			"general", "server", "postgresql", "sqlite", "authentication",
			"keycloak", "agents", "assets", "collection", "query",
			"retention", "security", "logging", "notifications", "backup",
		},
	})
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Settings []settingUpdate `json:"settings"`
	}
	if decodeJSON(r, &input) != nil || len(input.Settings) == 0 {
		writeAPIError(w, r, 400, "INVALID_SETTINGS", "At least one setting is required.")
		return
	}
	tx, err := s.database.DB().BeginTx(r.Context(), nil)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer tx.Rollback()
	userID := principalFromContext(r.Context()).User.ID
	for _, item := range input.Settings {
		if dedicatedSetting(item.Key) {
			writeAPIError(
				w, r, http.StatusConflict, "DEDICATED_SETTING_ENDPOINT",
				dedicatedSettingEndpointDetail,
			)
			return
		}
		if !validSettingKey(item.Key) || !json.Valid(item.Value) {
			writeAPIError(w, r, 400, "INVALID_SETTING", "A setting key or value is invalid.")
			return
		}
		if item.ApplyMode == "" {
			item.ApplyMode = "immediate"
		}
		if item.ApplyMode != "immediate" && item.ApplyMode != "restart" &&
			item.ApplyMode != "migration" {
			writeAPIError(w, r, 400, "INVALID_APPLY_MODE", "The apply mode is invalid.")
			return
		}
		stored := string(item.Value)
		if item.Secret {
			if s.bootstrapStore == nil {
				writeAPIError(w, r, 503, "SECRET_STORE_UNAVAILABLE", "Secret storage is unavailable.")
				return
			}
			sealed, err := s.bootstrapStore.SealString(
				"setting:"+item.Key, stored,
			)
			if err != nil {
				s.internalError(w, r, err)
				return
			}
			bytes, _ := json.Marshal(map[string]string{"ciphertext": sealed})
			stored = string(bytes)
		}
		var before any
		var version int
		err := tx.QueryRowContext(r.Context(),
			`SELECT value_json,version FROM settings WHERE key=$1`,
			item.Key).Scan(&before, &version)
		if errors.Is(err, sql.ErrNoRows) {
			before, version = nil, 0
		} else if err != nil {
			s.internalError(w, r, err)
			return
		}
		active, pending := stored, any(nil)
		if item.ApplyMode != "immediate" && version > 0 {
			active, pending = valueString(before), stored
		}
		next := version + 1
		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO settings(
			 key,value_json,secret,apply_mode,pending_value_json,
			 version,updated_by,updated_at
			 ) VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			 ON CONFLICT(key) DO UPDATE SET
			 value_json=excluded.value_json,secret=excluded.secret,
			 apply_mode=excluded.apply_mode,
			 pending_value_json=excluded.pending_value_json,
			 version=excluded.version,updated_by=excluded.updated_by,
			 updated_at=excluded.updated_at`,
			item.Key, active, item.Secret, item.ApplyMode, pending, next,
			userID, time.Now().UTC())
		if err == nil {
			_, err = tx.ExecContext(r.Context(),
				`INSERT INTO setting_versions(
				 id,setting_key,version,before_json,after_json,changed_by,reason
				 ) VALUES($1,$2,$3,$4,$5,$6,$7)`,
				uuid.NewString(), item.Key, next, before, stored, userID,
				item.Reason)
		}
		if err != nil {
			s.internalError(w, r, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		s.internalError(w, r, err)
		return
	}
	for _, item := range input.Settings {
		s.recordAdminAudit(
			r, "setting.update", "setting", item.Key, nil,
			maskedSettingValue(item.Value, item.Secret), item.Reason,
		)
	}
	writeJSON(w, 200, map[string]any{"updated": len(input.Settings)})
}

func (s *Server) settingHistory(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	rows, err := s.database.DB().QueryContext(r.Context(),
		`SELECT v.id,v.setting_key,v.version,v.before_json,v.after_json,
		 v.changed_by,v.reason,v.created_at,COALESCE(s.secret,FALSE)
		 FROM setting_versions v LEFT JOIN settings s ON s.key=v.setting_key
		 WHERE v.setting_key NOT IN ($1,$2)
		   AND ($3='' OR v.setting_key=$3)
		 ORDER BY v.created_at DESC LIMIT 500`,
		keycloakDedicatedSetting,
		keycloakClientSecretSetting,
		key,
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, settingKey, reason string
		var version int
		var before, after, user, created any
		var secret bool
		if err := rows.Scan(
			&id, &settingKey, &version, &before, &after, &user, &reason,
			&created, &secret,
		); err != nil {
			s.internalError(w, r, err)
			return
		}
		items = append(items, map[string]any{
			"id": id, "key": settingKey, "version": version,
			"before":     maskedSettingValue(before, secret),
			"after":      maskedSettingValue(after, secret),
			"changed_by": user, "reason": reason, "created_at": apiTime(created),
		})
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (s *Server) rollbackSetting(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Key     string `json:"key"`
		Version int    `json:"version"`
		Reason  string `json:"reason"`
	}
	if decodeJSON(r, &input) != nil || input.Key == "" || input.Version <= 0 {
		writeAPIError(w, r, 400, "INVALID_ROLLBACK", "key and version are required.")
		return
	}
	if dedicatedSetting(input.Key) {
		writeAPIError(
			w, r, http.StatusConflict, "DEDICATED_SETTING_ENDPOINT",
			dedicatedSettingEndpointDetail,
		)
		return
	}
	var target string
	if err := s.database.DB().QueryRowContext(r.Context(),
		`SELECT after_json FROM setting_versions
		 WHERE setting_key=$1 AND version=$2`,
		input.Key, input.Version).Scan(&target); errors.Is(err, sql.ErrNoRows) {
		writeAPIError(w, r, 404, "SETTING_VERSION_NOT_FOUND", "The setting version does not exist.")
		return
	} else if err != nil {
		s.internalError(w, r, err)
		return
	}
	var current string
	var version int
	if err := s.database.DB().QueryRowContext(r.Context(),
		`SELECT value_json,version FROM settings WHERE key=$1`,
		input.Key).Scan(&current, &version); err != nil {
		s.internalError(w, r, err)
		return
	}
	next := version + 1
	tx, err := s.database.DB().BeginTx(r.Context(), nil)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(),
		`UPDATE settings SET value_json=$1,pending_value_json=NULL,
		 version=$2,updated_by=$3,updated_at=$4 WHERE key=$5`,
		target, next, principalFromContext(r.Context()).User.ID,
		time.Now().UTC(), input.Key)
	if err == nil {
		_, err = tx.ExecContext(r.Context(),
			`INSERT INTO setting_versions(
			 id,setting_key,version,before_json,after_json,changed_by,reason
			 ) VALUES($1,$2,$3,$4,$5,$6,$7)`,
			uuid.NewString(), input.Key, next, current, target,
			principalFromContext(r.Context()).User.ID, input.Reason)
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		s.internalError(w, r, err)
		return
	}
	s.recordAdminAudit(r, "setting.rollback", "setting", input.Key, nil,
		map[string]any{"target_version": input.Version}, input.Reason)
	writeJSON(w, 200, map[string]any{"rolled_back": true, "version": next})
}

func validSettingKey(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '.' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func dedicatedSetting(key string) bool {
	return key == keycloakDedicatedSetting ||
		key == keycloakClientSecretSetting ||
		key == mail.KeyPassword
}

func valueString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		bytes, _ := json.Marshal(value)
		return string(bytes)
	}
}

func maskedSettingValue(value any, secret bool) any {
	if secret {
		return map[string]any{"configured": value != nil, "masked": true}
	}
	return rawJSON(value)
}
