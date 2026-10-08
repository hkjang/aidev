package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
)

type gatedPreferenceBody struct {
	reader  *strings.Reader
	started chan<- struct{}
	release <-chan struct{}
	once    sync.Once
}

func (body *gatedPreferenceBody) Read(p []byte) (int, error) {
	body.once.Do(func() {
		body.started <- struct{}{}
		<-body.release
	})
	return body.reader.Read(p)
}

func TestConcurrentPartialPreferenceUpdatesDoNotClobberIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Close()
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	userID := uuid.New()
	username := "preference_atomic_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id,locale,theme) VALUES($1,'ko','light')`, userID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	router := chi.NewRouter()
	router.Put("/api/v1/preferences", server.putPreferences)
	handler := authService.Middleware(auth.Require(router))

	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseBodies := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseBodies()
	responses := make(chan *httptest.ResponseRecorder, 2)
	send := func(payload string) {
		body := &gatedPreferenceBody{reader: strings.NewReader(payload), started: started, release: release}
		request := httptest.NewRequest(http.MethodPut, "/api/v1/preferences", body)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		responses <- response
	}
	go send(`{"locale":"en"}`)
	go send(`{"theme":"dark"}`)

	// Both old handlers read the same row before asking for their request body.
	// Releasing the bodies together makes the stale-snapshot overwrite
	// deterministic, while the transactional implementation serializes them.
	for range 2 {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("preference request did not reach body decoding")
		}
	}
	releaseBodies()
	for range 2 {
		select {
		case response := <-responses:
			if response.Code != http.StatusOK {
				t.Fatalf("preference update response=%d body=%s", response.Code, response.Body.String())
			}
		case <-time.After(5 * time.Second):
			t.Fatal("preference update did not complete")
		}
	}

	var locale, theme string
	if err = db.Pool.QueryRow(ctx, `SELECT locale,theme FROM user_preferences WHERE user_id=$1`, userID).Scan(&locale, &theme); err != nil {
		t.Fatal(err)
	}
	if locale != "en" || theme != "dark" {
		t.Fatalf("concurrent patches clobbered each other: locale=%q theme=%q", locale, theme)
	}
}

func TestPreferencesPatchPreservesOmittedFieldsAndClearsPause(t *testing.T) {
	pauseUntil := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	current := preferences{Locale: "ko", Theme: "dark", DreamPauseUntil: &pauseUntil}
	var patch preferencesPatch
	if err := decodeBody(t, `{"locale":"en","dream_pause_until":null}`, &patch); err != nil {
		t.Fatal(err)
	}
	if err := patch.apply(&current); err != nil {
		t.Fatal(err)
	}
	if current.Locale != "en" || current.Theme != "dark" || current.DreamPauseUntil != nil {
		t.Fatalf("patch did not preserve omission or clear null: %#v", current)
	}
}
