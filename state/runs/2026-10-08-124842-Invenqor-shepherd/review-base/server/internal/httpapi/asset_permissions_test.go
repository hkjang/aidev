package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/auth"
)

func TestAssetExportAndRestoreUseTheirDedicatedPermissions(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	adminCookie, adminCSRF := authenticateInitialAdmin(t, server, runtime)

	writerRole := uuid.NewString()
	deleteExportRole := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO roles(id,name,description) VALUES
		 ($1,'asset_writer_only','write without export or delete'),
		 ($2,'asset_delete_export','delete and export without write')`,
		writerRole,
		deleteExportRole,
	); err != nil {
		t.Fatal(err)
	}
	for _, grant := range []struct {
		role       string
		permission string
	}{
		{writerRole, "assets.read"},
		{writerRole, "assets.write"},
		{deleteExportRole, "assets.read"},
		{deleteExportRole, "assets.delete"},
		{deleteExportRole, "assets.export"},
	} {
		if _, err := runtime.DB().Exec(
			`INSERT INTO role_permissions(role_id,permission_name) VALUES($1,$2)`,
			grant.role,
			grant.permission,
		); err != nil {
			t.Fatal(err)
		}
	}

	writerCookie, writerCSRF := createAndLoginPermissionUser(
		t,
		server,
		adminCookie,
		adminCSRF,
		"writer.only",
		"asset_writer_only",
	)
	exportDenied := httptest.NewRecorder()
	exportRequest := httptest.NewRequest(http.MethodGet, "/api/v1/assets.csv", nil)
	exportRequest.AddCookie(writerCookie)
	server.Handler().ServeHTTP(exportDenied, exportRequest)
	if exportDenied.Code != http.StatusForbidden {
		t.Fatalf(
			"writer-only CSV status = %d body = %s, want 403",
			exportDenied.Code,
			exportDenied.Body.String(),
		)
	}
	for _, path := range []string{
		"/api/v1/dashboard/statistics",
		"/api/v1/assets/visualization",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(writerCookie)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf(
				"writer-only %s status = %d body = %s, want 403",
				path,
				response.Code,
				response.Body.String(),
			)
		}
	}

	assetID := uuid.NewString()
	now := time.Now().UTC()
	if _, err := runtime.DB().Exec(
		`INSERT INTO assets(
		 id,asset_key,name,type,status,criticality,environment,source,
		 first_seen_at,last_seen_at,deleted_at
		 ) VALUES($1,$2,'permission-asset','host','deleted','normal',
		 'other','manual',$3,$3,$3)`,
		assetID,
		uuid.NewString(),
		now,
	); err != nil {
		t.Fatal(err)
	}
	restoreDenied := performAuthenticatedJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/assets/"+assetID+"/restore",
		map[string]any{},
		writerCookie,
		writerCSRF,
	)
	if restoreDenied.Code != http.StatusForbidden {
		t.Fatalf(
			"writer-only restore status = %d body = %s, want 403",
			restoreDenied.Code,
			restoreDenied.Body.String(),
		)
	}

	managerCookie, managerCSRF := createAndLoginPermissionUser(
		t,
		server,
		adminCookie,
		adminCSRF,
		"delete.export",
		"asset_delete_export",
	)
	exportAllowed := httptest.NewRecorder()
	exportRequest = httptest.NewRequest(http.MethodGet, "/api/v1/assets.csv", nil)
	exportRequest.AddCookie(managerCookie)
	server.Handler().ServeHTTP(exportAllowed, exportRequest)
	if exportAllowed.Code != http.StatusOK ||
		exportAllowed.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf(
			"delete/export CSV = %d/%q body = %s",
			exportAllowed.Code,
			exportAllowed.Header().Get("Content-Type"),
			exportAllowed.Body.String(),
		)
	}
	restoreAllowed := performAuthenticatedJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/assets/"+assetID+"/restore",
		map[string]any{},
		managerCookie,
		managerCSRF,
	)
	if restoreAllowed.Code != http.StatusOK {
		t.Fatalf(
			"delete/export restore status = %d body = %s",
			restoreAllowed.Code,
			restoreAllowed.Body.String(),
		)
	}
}

func createAndLoginPermissionUser(
	t *testing.T,
	server *Server,
	adminCookie *http.Cookie,
	adminCSRF string,
	username string,
	role string,
) (*http.Cookie, string) {
	t.Helper()
	password := "PermissionBoundary!42"
	created := performAuthenticatedJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/admin/users",
		map[string]any{
			"username": username,
			"password": password,
			"roles":    []string{role},
			"reason":   "permission boundary test",
		},
		adminCookie,
		adminCSRF,
	)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %s status = %d body = %s", username, created.Code, created.Body.String())
	}
	login := performJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/auth/local/login",
		map[string]string{"username": username, "password": password},
		nil,
	)
	if login.Code != http.StatusOK {
		t.Fatalf("login %s status = %d body = %s", username, login.Code, login.Body.String())
	}
	var payload struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range login.Result().Cookies() {
		if cookie.Name == auth.SessionCookie {
			return cookie, payload.CSRF
		}
	}
	t.Fatal("login response omitted session cookie")
	return nil, ""
}
