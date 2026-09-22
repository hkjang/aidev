package server

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOperationsSettingsBounds(t *testing.T) {
	if err := validateOperationsSettings(defaultSettings()); err != nil {
		t.Fatal(err)
	}
	for _, patch := range []map[string]any{{"otel_enabled": true}, {"otel_endpoint": "http://127.0.0.1:9999/v1/traces"}, {"otel_endpoint": "https://user:secret@example.test/v1/traces"}, {"otel_endpoint": "https://example.test/v1/traces?token=secret"}, {"otel_endpoint": "https://example.test/v1/traces#secret"}, {"otel_ca_pem": "bad"}, {"otel_auth_token": "secret\r\nInjected: value"}, {"otel_sample_rate": -0.1}, {"otel_sample_rate": 1.1}, {"otel_timeout_seconds": 1.5}, {"operations_retention_days": 0}, {"feature_flags": map[string]any{"unknown": true}}, {"feature_flags": map[string]any{"canvas": "false"}}, {"feature_flags": map[string]any{"canvas": nil}}} {
		cfg := defaultSettings()
		for k, v := range patch {
			cfg[k] = v
		}
		if validateOperationsSettings(cfg) == nil {
			t.Fatalf("invalid settings accepted: %v", patch)
		}
	}
	cfg := defaultSettings()
	cfg["otel_endpoint"] = "http://127.0.0.1:9999/v1/traces"
	cfg["otel_allow_http"] = true
	cfg["otel_enabled"] = true
	if err := validateOperationsSettings(cfg); err != nil {
		t.Fatal(err)
	}
}
func TestOperationsFeatureCeilingsAndActualModules(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	for _, feature := range featureCatalogue() {
		if !s.canFeature(ctx, p, wid, feature.ID) {
			t.Fatalf("default off: %s", feature.ID)
		}
	}
	if s.canFeature(ctx, p, newID(), "canvas") || s.canFeature(ctx, p, wid, "unknown") {
		t.Fatal("unknown scope granted")
	}
	canvas := testJSONObject(t, admin.request("POST", "/api/v1/canvases", map[string]any{"workspace_id": wid, "title": "보존되는 원본", "data": canvasData{}}, 200))
	path := "/api/v1/canvases/" + str(canvas, "id")
	db := testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "수식 정책", "properties": []any{map[string]any{"id": "n", "name": "원본", "type": "number"}, map[string]any{"id": "f", "name": "수식", "type": "formula", "expression": `prop("n") * 2`}}}, 200))
	dbID := str(db, "id")
	admin.request("POST", "/api/v1/databases/"+dbID+"/rows", map[string]any{"values": map[string]any{"n": 9}}, 200)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"feature_flags": map[string]any{"canvas": false, "database-formula": false}}, 200)
	admin.request("GET", path, nil, 404)
	admin.request("POST", "/api/v1/canvases", map[string]any{"workspace_id": wid, "title": "금지", "data": canvasData{}}, 403)
	result := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+dbID+"/query", map[string]any{}, 200))
	row := result["rows"].([]any)[0].(map[string]any)
	if row["values"].(map[string]any)["n"] != float64(9) || row["computed_values"].(map[string]any)["f"] != nil || row["errors"].(map[string]any)["f"] == nil {
		t.Fatalf("formula ceiling not applied: %v", row)
	}
	ws := testJSONObject(t, admin.request("GET", "/api/v1/workspaces/"+wid+"/settings", nil, 200))
	ws = testJSONObject(t, admin.request("PUT", "/api/v1/workspaces/"+wid+"/settings", map[string]any{"version": ws["version"], "data": map[string]any{"feature_flags": map[string]any{"canvas": true}}}, 200))
	userPath := "/api/v1/admin/features/users/" + p.ID
	uf := testJSONObject(t, admin.request("PUT", userPath, map[string]any{"version": 0, "data": map[string]any{"canvas": true}}, 200))
	if s.canFeature(ctx, p, wid, "canvas") {
		t.Fatal("lower-level true bypassed service false")
	}
	admin.request("PUT", userPath, map[string]any{"version": 0, "data": map[string]any{}}, 409)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"feature_flags": map[string]any{}}, 200)
	if !s.canFeature(ctx, p, wid, "canvas") {
		t.Fatal("feature not reenabled")
	}
	admin.request("GET", path, nil, 200)
	result = testJSONObject(t, admin.request("POST", "/api/v1/databases/"+dbID+"/query", map[string]any{}, 200))
	row = result["rows"].([]any)[0].(map[string]any)
	if row["computed_values"].(map[string]any)["f"] != float64(18) {
		t.Fatal("preserved formula not recomputed")
	}
	uf = testJSONObject(t, admin.request("PUT", userPath, map[string]any{"version": uf["version"], "data": map[string]any{"canvas": false}}, 200))
	if s.canFeature(ctx, p, wid, "canvas") {
		t.Fatal("user false bypassed")
	}
	admin.request("POST", userPath+"/history/1/restore", map[string]any{"version": uf["version"]}, 200)
	if !s.canFeature(ctx, p, wid, "canvas") {
		t.Fatal("user history not restored")
	}
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/settings", map[string]any{"version": ws["version"], "data": map[string]any{"feature_flags": map[string]any{"canvas": false}}}, 200)
	if s.canFeature(ctx, p, wid, "canvas") {
		t.Fatal("workspace false bypassed user true")
	}
	_, err := s.DB.Exec(ctx, `UPDATE user_feature_flags SET data='{"canvas":"not-a-boolean"}' WHERE user_id=$1`, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.canFeature(ctx, p, wid, "canvas") {
		t.Fatal("malformed config failed open")
	}
}

func operationTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{R: 16, G: 100, B: 90, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func operationUpload(t *testing.T, client *integrationTestClient, wid string, data []byte, want int) map[string]any {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, _ := writer.CreateFormFile("file", "untrusted.png")
	file.Write(data)
	writer.Close()
	request, _ := http.NewRequest("POST", client.base+"/api/v1/workspaces/"+wid+"/branding/assets", &body)
	request.Header.Set("X-Madi-Request", "1")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := client.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != want {
		t.Fatalf("upload got %d want%d: %s", response.StatusCode, want, raw)
	}
	return testJSONObject(t, raw)
}
func TestOperationsBrandImagesAndIsolation(t *testing.T) {
	if whiteContrast("#087768") < 4.5 || whiteContrast("#ffffff") >= 4.5 {
		t.Fatal("contrast calculation")
	}
	for _, data := range [][]byte{[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), bytes.Repeat([]byte("x"), (1<<20)+1), operationTestPNG(t, 2049, 2)} {
		if _, _, _, err := sanitizeBrandImage(data); err == nil {
			t.Fatal("unsafe image accepted")
		}
	}
	clean, width, height, err := sanitizeBrandImage(append(operationTestPNG(t, 1024, 256), []byte("PRIVATE_METADATA_SENTINEL")...))
	if err != nil || width != 512 || height != 128 || bytes.Contains(clean, []byte("PRIVATE_METADATA_SENTINEL")) {
		t.Fatalf("canonical image: %dx%d %v", width, height, err)
	}
	s, admin, ctx, _, wid := jobTestFixture(t)
	upload := operationUpload(t, admin, wid, operationTestPNG(t, 48, 48), 200)
	other := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "격리 브랜딩"}, 200))
	wsPath := "/api/v1/workspaces/" + wid + "/settings"
	snapshot := testJSONObject(t, admin.request("GET", wsPath, nil, 200))
	admin.request("PUT", wsPath, map[string]any{"version": snapshot["version"], "data": map[string]any{"logo_url": "https://tracker.test/logo.png"}}, 400)
	admin.request("PUT", wsPath, map[string]any{"version": snapshot["version"], "data": map[string]any{"theme_primary": "#ffffff"}}, 400)
	admin.request("PUT", wsPath, map[string]any{"version": snapshot["version"], "data": map[string]any{"logo_url": brandAssetURL(str(other, "id"), str(upload, "digest"))}}, 400)
	snapshot = testJSONObject(t, admin.request("PUT", wsPath, map[string]any{"version": snapshot["version"], "data": map[string]any{"logo_url": upload["url"], "favicon_url": upload["url"], "theme_primary": "#087768"}}, 200))
	if !strings.Contains(string(jsonValue(snapshot)), str(upload, "digest")) {
		t.Fatal("brand not saved")
	}
	admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "brand-viewer@example.test", "name": "브랜드 열람자", "role": "viewer", "password": "Brand-viewer-password-2026!"}, 200)
	viewer := newIntegrationTestClient(t, admin.base)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": "brand-viewer@example.test", "password": "Brand-viewer-password-2026!"}, 200)
	viewer.request("GET", str(upload, "url"), nil, 404)
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "brand-viewer@example.test", "role": "viewer"}, 200)
	response, err := viewer.client.Get(viewer.base + str(upload, "url"))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "image/png" || !strings.Contains(response.Header.Get("Cache-Control"), "no-store") {
		t.Fatal("brand member response")
	}
	operationUpload(t, viewer, wid, operationTestPNG(t, 48, 48), 403)
	_, err = s.DB.Exec(ctx, `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=(SELECT id FROM users WHERE email='brand-viewer@example.test')`, wid)
	if err != nil {
		t.Fatal(err)
	}
	viewer.request("GET", str(upload, "url"), nil, 404)
	var count int
	s.DB.QueryRow(context.Background(), "SELECT count(*) FROM workspace_brand_assets WHERE workspace_id=$1", wid).Scan(&count)
	if count != 1 {
		t.Fatal("original assets not preserved")
	}
}

func operationWait(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("operation condition timed out")
}

func TestOperationsPluginFeatureCancelsRunningJob(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	manifest := pluginTestManifest()
	pluginTestUpload(t, admin, pluginTestZIP(t, manifest, nil), "", 200)
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/plugins/"+manifest.ID, map[string]any{"enabled": true, "capabilities": []string{"document:read"}}, 200)
	plugin := *p
	plugin.PluginID, plugin.WorkspaceID, plugin.ScopeRestricted, plugin.Scopes = manifest.ID, wid, true, []string{"document:read"}
	jobCtx := context.WithValue(ctx, principalKey, &plugin)
	started, done := make(chan struct{}), make(chan struct{})
	s.RegisterJobHandler("test.feature.cancel", func(ctx context.Context, _ Job) (map[string]any, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	id, err := s.EnqueueJob(jobCtx, nil, "test.feature.cancel", p.ID, wid, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.claimJob(ctx)
	if err != nil || job.ID != id {
		t.Fatalf("job claim %v %v", job, err)
	}
	go func() { s.runJob(ctx, job); close(done) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("job not started")
	}
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"feature_flags": map[string]any{"plugins": false}}, 200)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("feature-disabled plugin job kept running")
	}
	var status string
	if err = s.DB.QueryRow(ctx, "SELECT status FROM automation_jobs WHERE id=$1", id).Scan(&status); err != nil || status != "cancelled" {
		t.Fatalf("job must not retry automatically: %s %v", status, err)
	}
	admin.request("POST", "/api/v1/plugins/"+manifest.ID+"/bridge", map[string]any{"workspace_id": wid, "operation": "documents.list", "args": map[string]any{}}, 403)
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/plugins/"+manifest.ID, map[string]any{"enabled": false, "capabilities": []string{"document:read"}}, 200)
}
