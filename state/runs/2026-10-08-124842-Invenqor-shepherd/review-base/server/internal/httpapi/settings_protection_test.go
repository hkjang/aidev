package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGenericSettingsCannotExposeOrOverwriteKeycloakState(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	if _, err := runtime.DB().Exec(
		`INSERT INTO settings(key,value_json,secret,apply_mode,version)
		 VALUES($1,$2,FALSE,'new_login',1),
		       ($3,$4,TRUE,'immediate',1)`,
		keycloakDedicatedSetting,
		`{"enabled":false}`,
		keycloakClientSecretSetting,
		`{"sealed":"v1.not-a-real-secret.envelope"}`,
	); err != nil {
		t.Fatal(err)
	}

	listed := performAuthenticatedJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/admin/settings",
		nil,
		cookie,
		csrf,
	)
	if listed.Code != http.StatusOK {
		t.Fatalf("list settings status = %d body = %s", listed.Code, listed.Body.String())
	}
	var payload struct {
		Items []struct {
			Key string `json:"key"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, item := range payload.Items {
		if dedicatedSetting(item.Key) {
			t.Fatalf("generic settings exposed dedicated key %q", item.Key)
		}
	}

	for _, key := range []string{
		keycloakDedicatedSetting,
		keycloakClientSecretSetting,
	} {
		updated := performAuthenticatedJSON(
			t,
			server,
			http.MethodPatch,
			"/api/v1/admin/settings",
			map[string]any{"settings": []map[string]any{{
				"key": key, "value": map[string]any{"overwritten": true},
				"secret": false, "apply_mode": "immediate", "reason": "test",
			}}},
			cookie,
			csrf,
		)
		if updated.Code != http.StatusConflict {
			t.Fatalf(
				"generic update %q status = %d body = %s",
				key,
				updated.Code,
				updated.Body.String(),
			)
		}
	}

	rolledBack := performAuthenticatedJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/admin/settings/rollback",
		map[string]any{
			"key": keycloakDedicatedSetting, "version": 1, "reason": "test",
		},
		cookie,
		csrf,
	)
	if rolledBack.Code != http.StatusConflict {
		t.Fatalf(
			"generic rollback status = %d body = %s",
			rolledBack.Code,
			rolledBack.Body.String(),
		)
	}

	var stored string
	if err := runtime.DB().QueryRow(
		"SELECT value_json FROM settings WHERE key=$1",
		keycloakDedicatedSetting,
	).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	// Compared as a value, not as text: value_json is JSONB on PostgreSQL, which
	// re-serialises what it is given - key order is normalised and a space appears
	// after each colon - so a byte comparison here fails on an unchanged setting
	// and passes only because SQLite stores the text verbatim.
	var decoded map[string]any
	if err := json.Unmarshal([]byte(stored), &decoded); err != nil {
		t.Fatalf("stored setting is not JSON: %s", stored)
	}
	if len(decoded) != 1 || decoded["enabled"] != false {
		t.Fatalf("dedicated setting was changed to %s", stored)
	}
}
