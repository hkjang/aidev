package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

func TestUserManagementLifecycleAndSafety(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	adminCookie, csrf := authenticateInitialAdmin(t, server, runtime)

	rolesRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/roles",
		nil,
	)
	rolesRequest.AddCookie(adminCookie)
	rolesResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(rolesResponse, rolesRequest)
	if rolesResponse.Code != http.StatusOK ||
		!containsBody(rolesResponse, `"viewer"`) ||
		!containsBody(rolesResponse, `"users.manage"`) {
		t.Fatalf(
			"roles status = %d body = %s",
			rolesResponse.Code,
			rolesResponse.Body.String(),
		)
	}

	create := performAuthenticatedJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/admin/users",
		map[string]any{
			"username":     "managed.user",
			"display_name": "Managed User",
			"email":        "managed@example.test",
			"password":     "ManagedUser!42",
			"roles":        []string{"viewer"},
			"reason":       "test lifecycle",
		},
		adminCookie,
		csrf,
	)
	if create.Code != http.StatusCreated {
		t.Fatalf(
			"create status = %d body = %s",
			create.Code,
			create.Body.String(),
		)
	}
	var created struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created user: %v", err)
	}

	login := performJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/auth/local/login",
		map[string]string{
			"username": "managed.user",
			"password": "ManagedUser!42",
		},
		nil,
	)
	if login.Code != http.StatusOK {
		t.Fatalf(
			"managed login status = %d body = %s",
			login.Code,
			login.Body.String(),
		)
	}

	deactivate := performAuthenticatedJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/admin/users/"+created.User.ID,
		map[string]any{
			"active": false,
			"roles":  []string{"operator"},
			"reason": "access review",
		},
		adminCookie,
		csrf,
	)
	if deactivate.Code != http.StatusOK {
		t.Fatalf(
			"deactivate status = %d body = %s",
			deactivate.Code,
			deactivate.Body.String(),
		)
	}
	inactiveLogin := performJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/auth/local/login",
		map[string]string{
			"username": "managed.user",
			"password": "ManagedUser!42",
		},
		nil,
	)
	if inactiveLogin.Code != http.StatusForbidden {
		t.Fatalf(
			"inactive login status = %d, want 403",
			inactiveLogin.Code,
		)
	}

	selfDelete := performAuthenticatedJSON(
		t,
		server,
		http.MethodDelete,
		"/api/v1/admin/users/"+currentUserID(t, server, adminCookie),
		map[string]any{},
		adminCookie,
		csrf,
	)
	if selfDelete.Code != http.StatusConflict ||
		!containsBody(selfDelete, "SELF_DELETE") {
		t.Fatalf(
			"self delete status = %d body = %s",
			selfDelete.Code,
			selfDelete.Body.String(),
		)
	}

	var audits int
	if err := runtime.DB().QueryRow(
		`SELECT COUNT(*) FROM audit_logs
		  WHERE action IN ('user.create','user.update')`,
	).Scan(&audits); err != nil {
		t.Fatalf("count user audit entries: %v", err)
	}
	if audits != 2 {
		t.Fatalf("user audit entry count = %d, want 2", audits)
	}
}

func TestUserPasswordResetRevokesSessions(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	adminCookie, csrf := authenticateInitialAdmin(t, server, runtime)
	create := performAuthenticatedJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/admin/users",
		map[string]any{
			"username": "password.user",
			"password": "OriginalPassword!42",
			"roles":    []string{"viewer"},
		},
		adminCookie,
		csrf,
	)
	var created struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if create.Code != http.StatusCreated ||
		json.Unmarshal(create.Body.Bytes(), &created) != nil {
		t.Fatalf(
			"create status = %d body = %s",
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
			"username": "password.user",
			"password": "OriginalPassword!42",
		},
		nil,
	)
	var userCookie *http.Cookie
	for _, cookie := range login.Result().Cookies() {
		if cookie.Name == "invenqor_session" {
			userCookie = cookie
		}
	}
	if login.Code != http.StatusOK || userCookie == nil {
		t.Fatalf("login status = %d body = %s", login.Code, login.Body.String())
	}
	reset := performAuthenticatedJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/admin/users/"+created.User.ID+"/password",
		map[string]string{
			"password": "ReplacementPassword!84",
			"reason":   "credential rotation",
		},
		adminCookie,
		csrf,
	)
	if reset.Code != http.StatusOK {
		t.Fatalf(
			"reset status = %d body = %s",
			reset.Code,
			reset.Body.String(),
		)
	}
	me := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	me.AddCookie(userCookie)
	meResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(meResponse, me)
	if meResponse.Code != http.StatusUnauthorized {
		t.Fatalf("old session status = %d, want 401", meResponse.Code)
	}
	newLogin := performJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/auth/local/login",
		map[string]string{
			"username": "password.user",
			"password": "ReplacementPassword!84",
		},
		nil,
	)
	if newLogin.Code != http.StatusOK {
		t.Fatalf(
			"new password login status = %d body = %s",
			newLogin.Code,
			newLogin.Body.String(),
		)
	}
}

func currentUserID(
	t *testing.T,
	server *Server,
	cookie *http.Cookie,
) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	var payload struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if response.Code != http.StatusOK ||
		json.Unmarshal(response.Body.Bytes(), &payload) != nil {
		t.Fatalf(
			"me status = %d body = %s",
			response.Code,
			response.Body.String(),
		)
	}
	return payload.User.ID
}

func containsBody(response *httptest.ResponseRecorder, value string) bool {
	return strings.Contains(response.Body.String(), value)
}
