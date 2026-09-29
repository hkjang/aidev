package app

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Server) getSetting(ctx context.Context, key string) (string, error) {
	var value string
	var secret bool
	err := s.db.QueryRow(ctx, `SELECT value,secret FROM settings WHERE key=$1`, key).Scan(&value, &secret)
	if err != nil {
		return "", err
	}
	if secret && value != "" {
		return s.keys.Decrypt(value)
	}
	return value, nil
}

func (s *Server) listSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT key,value,secret FROM settings ORDER BY key`)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer rows.Close()
	items := []Setting{}
	for rows.Next() {
		var item Setting
		if err := rows.Scan(&item.Key, &item.Value, &item.Secret); err != nil {
			continue
		}
		item.Configured = item.Value != ""
		if item.Secret && item.Configured {
			item.Value = "********"
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Settings map[string]string `json:"settings"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	keys := make([]string, 0, len(in.Settings))
	for key := range in.Settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	for _, key := range keys {
		value := strings.TrimSpace(in.Settings[key])
		var secret bool
		if err := tx.QueryRow(r.Context(), `SELECT secret FROM settings WHERE key=$1`, key).Scan(&secret); err != nil {
			if err == pgx.ErrNoRows {
				writeError(w, http.StatusBadRequest, "unknown_setting", "지원하지 않는 설정입니다: "+key)
				return
			}
			notFoundOrServer(w, err)
			return
		}
		if secret && (value == "********" || value == "") {
			continue
		}
		if secret {
			value, err = s.keys.Encrypt(value)
			if err != nil {
				notFoundOrServer(w, err)
				return
			}
		}
		u, _ := userFrom(r)
		if _, err = tx.Exec(r.Context(), `UPDATE settings SET value=$2,updated_at=now(),updated_by=$3 WHERE key=$1`, key, value, u.ID); err != nil {
			notFoundOrServer(w, err)
			return
		}
	}
	if err := s.validateTrackingSettings(r.Context(), tx); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_setting", err.Error())
		return
	}
	if err := s.validateMCPOAuthSettings(r.Context(), tx); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_setting", err.Error())
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		notFoundOrServer(w, err)
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "settings.update", "settings", "", r.RemoteAddr, map[string]any{"keys": keys})
	writeJSON(w, http.StatusOK, map[string]any{"updated": keys})
}
