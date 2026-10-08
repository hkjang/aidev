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

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConcurrentPasswordFailuresRespectGlobalCeilingIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db := isolatedHTTPStore(t, dsn, 1)
	replicaPool, err := pgxpool.NewWithConfig(ctx, db.Pool.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(replicaPool.Close)
	replica := &store.Store{Pool: replicaPool}

	username := "parallel_login_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err = db.BootstrapAdmin(ctx, username, "correct-password"); err != nil {
		t.Fatal(err)
	}
	var userID uuid.UUID
	if err = db.Pool.QueryRow(ctx, `SELECT id FROM users WHERE username=$1`, username).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err = db.PutSetting(ctx, "security", map[string]any{
		"login_max_failures":    3,
		"login_lockout_minutes": 15,
	}, userID); err != nil {
		t.Fatal(err)
	}

	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService}
	replicaAuth := &auth.Service{Store: replica}
	servers := []*Server{server, {Store: replica, Auth: replicaAuth}}
	const attempts = 12
	start := make(chan struct{})
	statuses := make(chan int, attempts)
	var workers sync.WaitGroup
	for index := 0; index < attempts; index++ {
		workers.Add(1)
		go func(server *Server) {
			defer workers.Done()
			<-start
			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"`+username+`","password":"wrong"}`)).WithContext(ctx)
			request.Header.Set("Content-Type", "application/json")
			request.RemoteAddr = "203.0.113.81:44321"
			response := httptest.NewRecorder()
			server.login(response, request)
			statuses <- response.Code
		}(servers[index%len(servers)])
	}
	close(start)
	workers.Wait()
	close(statuses)

	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusUnauthorized] != 3 || counts[http.StatusTooManyRequests] != attempts-3 {
		t.Fatalf("parallel guesses crossed the configured ceiling: statuses=%v", counts)
	}

	identities := store.LoginIdentities(username, "203.0.113.81")
	rows, err := db.Pool.Query(ctx, `
		SELECT identity,failure_count,COALESCE(locked_until>now(),false)
		FROM login_attempts WHERE identity=ANY($1)`, identities)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type counter struct {
		failures int
		locked   bool
	}
	counters := map[string]counter{}
	for rows.Next() {
		var identity string
		var value counter
		if err = rows.Scan(&identity, &value.failures, &value.locked); err != nil {
			t.Fatal(err)
		}
		counters[identity] = value
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if got := counters["ip:203.0.113.81"]; got.failures != 3 || !got.locked {
		t.Fatalf("source counter did not stop at the ceiling: %#v", got)
	}
	if got := counters[store.LoginAccountIdentity(username)]; got.failures != 3 || got.locked {
		t.Fatalf("account counter changed outside accepted attempts: %#v", got)
	}
	var sessions int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE user_id=$1`, userID).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 {
		t.Fatalf("failed parallel logins created %d sessions", sessions)
	}
}
