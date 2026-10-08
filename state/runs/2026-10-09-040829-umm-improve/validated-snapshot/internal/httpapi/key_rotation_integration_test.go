package httpapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/cryptoutil"
	"github.com/hkjang/umm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func legacyV1Ciphertext(t *testing.T, key []byte, plain string) string {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	sealed := aead.Seal(nonce, nonce, []byte(plain), []byte("umm:v1"))
	return base64.RawURLEncoding.EncodeToString(sealed)
}

func isolatedHTTPStore(t *testing.T, dsn string, maxConns ...int32) *store.Store {
	t.Helper()
	ctx := context.Background()
	adminConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(adminPool.Close)

	schema := "httpapi_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err = adminPool.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE")
	})

	testConfig := adminConfig.Copy()
	testConfig.ConnConfig.RuntimeParams["search_path"] = identifier + ", public"
	if len(maxConns) > 0 {
		testConfig.MaxConns = maxConns[0]
		if testConfig.MinConns > testConfig.MaxConns {
			testConfig.MinConns = testConfig.MaxConns
		}
	}
	testPool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(testPool.Close)
	db := &store.Store{Pool: testPool}
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRotateEncryptionKeyRewrapsUnprefixedLegacyPromptIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	oldKey := []byte("0123456789abcdef0123456789abcdef")
	newKey := []byte("abcdef0123456789abcdef0123456789")
	keyring, err := cryptoutil.NewWithPrevious(newKey, oldKey)
	if err != nil {
		t.Fatal(err)
	}
	db.Cipher = keyring

	adminID := uuid.New()
	username := "key_rotation_" + strings.ReplaceAll(adminID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, adminID, username); err != nil {
		t.Fatal(err)
	}
	legacyPrompt := legacyV1Ciphertext(t, oldKey, "historical prompt")
	currentUnprefixed, err := keyring.Encrypt("unwrapped current prompt")
	if err != nil {
		t.Fatal(err)
	}
	var legacyCallID, currentCallID int64
	if err = db.Pool.QueryRow(ctx, `INSERT INTO ai_calls(user_id,model,status,prompt_ciphertext) VALUES($1,'legacy-model','success',$2) RETURNING id`, adminID, legacyPrompt).Scan(&legacyCallID); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `INSERT INTO ai_calls(user_id,model,status,prompt_ciphertext) VALUES($1,'current-model','success',$2) RETURNING id`, adminID, currentUnprefixed).Scan(&currentCallID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, adminID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db, Cipher: keyring}
	router := chi.NewRouter()
	router.Get("/status", server.encryptionKeyStatus)
	router.Post("/rotate", server.rotateEncryptionKey)
	handler := authService.Middleware(auth.Require(auth.RequireAdmin(router)))
	send := func(method, path string) (*httptest.ResponseRecorder, map[string]any) {
		t.Helper()
		request := httptest.NewRequest(method, path, nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		var payload map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode %s response: %v body=%s", path, err, response.Body.String())
		}
		return response, payload
	}

	before, beforePayload := send(http.MethodGet, "/status")
	if before.Code != http.StatusOK || beforePayload["pendingRotation"] != float64(2) || beforePayload["unreadable"] != float64(0) {
		t.Fatalf("status before rotation=%d payload=%#v", before.Code, beforePayload)
	}
	rotated, rotatedPayload := send(http.MethodPost, "/rotate")
	if rotated.Code != http.StatusOK || rotatedPayload["rotated"] != float64(2) {
		t.Fatalf("rotation response=%d payload=%#v", rotated.Code, rotatedPayload)
	}

	primaryOnly, err := cryptoutil.New(newKey)
	if err != nil {
		t.Fatal(err)
	}
	for callID, expected := range map[int64]string{
		legacyCallID:  "historical prompt",
		currentCallID: "unwrapped current prompt",
	} {
		var stored string
		if err = db.Pool.QueryRow(ctx, `SELECT prompt_ciphertext FROM ai_calls WHERE id=$1`, callID).Scan(&stored); err != nil {
			t.Fatal(err)
		}
		plain, decryptErr := primaryOnly.Decrypt(strings.TrimPrefix(stored, "enc:"))
		if decryptErr != nil || plain != expected || !strings.HasPrefix(stored, "enc:v2."+primaryOnly.KeyID()+".") {
			t.Fatalf("rotated prompt is not primary-key ciphertext: prefix=%q plain=%q err=%v", fmt.Sprintf("%.24s", stored), plain, decryptErr)
		}
	}
	after, afterPayload := send(http.MethodGet, "/status")
	if after.Code != http.StatusOK || afterPayload["pendingRotation"] != float64(0) || afterPayload["unreadable"] != float64(0) {
		t.Fatalf("status after rotation=%d payload=%#v", after.Code, afterPayload)
	}
}

func TestMaskedSettingSaveSerializesWithKeyRotationIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	oldKey := []byte("0123456789abcdef0123456789abcdef")
	newKey := []byte("abcdef0123456789abcdef0123456789")
	oldCipher, err := cryptoutil.New(oldKey)
	if err != nil {
		t.Fatal(err)
	}
	keyring, err := cryptoutil.NewWithPrevious(newKey, oldKey)
	if err != nil {
		t.Fatal(err)
	}
	db.Cipher = keyring

	adminID := uuid.New()
	username := "setting_rotation_" + strings.ReplaceAll(adminID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, adminID, username); err != nil {
		t.Fatal(err)
	}
	legacySecret, err := oldCipher.Encrypt("oidc-secret-before-rotation")
	if err != nil {
		t.Fatal(err)
	}
	var oidc map[string]any
	if err = db.GetSetting(ctx, "oidc", &oidc); err != nil {
		t.Fatal(err)
	}
	oidc["client_id"] = "before-rotation"
	oidc["client_secret"] = "enc:" + legacySecret
	if err = db.PutSetting(ctx, "oidc", oidc, adminID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, adminID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db, Cipher: keyring}
	router := chi.NewRouter()
	router.Put("/settings/{section}", server.putAdminSetting)
	router.Post("/rotate", server.rotateEncryptionKey)
	handler := authService.Middleware(auth.Require(auth.RequireAdmin(router)))

	maskedSave := make(map[string]any, len(oidc))
	for key, value := range oidc {
		maskedSave[key] = value
	}
	maskedSave["client_id"] = "saved-after-rotation"
	maskedSave["client_secret"] = secretMask
	maskedBody, err := json.Marshal(maskedSave)
	if err != nil {
		t.Fatal(err)
	}
	send := func(method, path string, body []byte) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(string(body)))
		if len(body) > 0 {
			request.Header.Set("Content-Type", "application/json")
		}
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	blockTx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blockTx.Rollback(context.Background())
	if err = db.LockSettingTx(ctx, blockTx, "oidc"); err != nil {
		t.Fatal(err)
	}

	rotationDone := make(chan *httptest.ResponseRecorder, 1)
	go func() { rotationDone <- send(http.MethodPost, "/rotate", nil) }()
	select {
	case response := <-rotationDone:
		t.Fatalf("key rotation bypassed the setting lock: status=%d body=%s", response.Code, response.Body.String())
	case <-time.After(100 * time.Millisecond):
	}

	saveDone := make(chan *httptest.ResponseRecorder, 1)
	go func() { saveDone <- send(http.MethodPut, "/settings/oidc", maskedBody) }()
	select {
	case response := <-saveDone:
		t.Fatalf("masked setting save bypassed the setting lock: status=%d body=%s", response.Code, response.Body.String())
	case <-time.After(100 * time.Millisecond):
	}
	if err = blockTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	for operation, done := range map[string]<-chan *httptest.ResponseRecorder{"rotation": rotationDone, "masked save": saveDone} {
		select {
		case response := <-done:
			if response.Code != http.StatusOK {
				t.Fatalf("%s response=%d body=%s", operation, response.Code, response.Body.String())
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s did not resume after the setting lock was released", operation)
		}
	}

	var stored map[string]any
	if err = db.GetSetting(ctx, "oidc", &stored); err != nil {
		t.Fatal(err)
	}
	if stored["client_id"] != "saved-after-rotation" {
		t.Fatalf("masked save did not preserve its non-secret update: %#v", stored)
	}
	storedSecret, _ := stored["client_secret"].(string)
	primaryOnly, err := cryptoutil.New(newKey)
	if err != nil {
		t.Fatal(err)
	}
	plain, decryptErr := primaryOnly.Decrypt(strings.TrimPrefix(storedSecret, "enc:"))
	if decryptErr != nil || plain != "oidc-secret-before-rotation" || !strings.HasPrefix(storedSecret, "enc:v2."+primaryOnly.KeyID()+".") {
		t.Fatalf("masked save restored stale ciphertext: prefix=%q plain=%q err=%v", fmt.Sprintf("%.24s", storedSecret), plain, decryptErr)
	}
}
