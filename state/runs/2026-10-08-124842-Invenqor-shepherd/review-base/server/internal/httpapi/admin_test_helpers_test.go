package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hkjang/invenqor/server/internal/auth"
	"github.com/hkjang/invenqor/server/internal/storage"
	"github.com/hkjang/invenqor/server/internal/storagetest"
)

func authenticateInitialAdmin(
	t *testing.T,
	server *Server,
	runtime *storage.Runtime,
) (*http.Cookie, string) {
	t.Helper()
	token, err := os.ReadFile(
		filepath.Join(storagetest.StateDir(t, runtime), "initial-admin.token"),
	)
	if err != nil {
		t.Fatalf("read bootstrap token: %v", err)
	}
	create := performJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/bootstrap/admin",
		map[string]string{
			"username":     "admin.user",
			"password":     "CorrectHorse!42",
			"display_name": "Administrator",
		},
		map[string]string{
			"X-Invenqor-Bootstrap-Token": strings.TrimSpace(string(token)),
		},
	)
	if create.Code != http.StatusCreated {
		t.Fatalf(
			"create initial admin status = %d body = %s",
			create.Code,
			create.Body.String(),
		)
	}
	login := performJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/auth/local/login",
		map[string]string{
			"username": "admin.user",
			"password": "CorrectHorse!42",
		},
		nil,
	)
	if login.Code != http.StatusOK {
		t.Fatalf(
			"login status = %d body = %s",
			login.Code,
			login.Body.String(),
		)
	}
	var payload struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	for _, cookie := range login.Result().Cookies() {
		if cookie.Name == auth.SessionCookie {
			return cookie, payload.CSRFToken
		}
	}
	t.Fatal("login response omitted session cookie")
	return nil, ""
}

func performAuthenticatedJSON(
	t *testing.T,
	server *Server,
	method string,
	path string,
	body any,
	cookie *http.Cookie,
	csrf string,
) *httptest.ResponseRecorder {
	t.Helper()
	bytesBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(bytesBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", csrf)
	request.AddCookie(cookie)
	authenticated := httptest.NewRecorder()
	server.Handler().ServeHTTP(authenticated, request)
	return authenticated
}
