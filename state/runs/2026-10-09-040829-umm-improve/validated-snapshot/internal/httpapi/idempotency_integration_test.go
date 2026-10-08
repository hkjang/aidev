package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
)

func TestIdempotencyPendingLeaseRecoversAfterCrashIntegration(t *testing.T) {
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
	username := "idempotency_lease_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	authService := &auth.Service{Store: db}
	session, err := authService.CreateSession(ctx, userID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}

	spaceID := uuid.New()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'idempotency lease')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := &Server{Store: db}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		note, createErr := db.CreateNote(r.Context(), userID, store.Note{SpaceID: spaceID, Content: "recover after crash"})
		if createErr != nil {
			writeError(w, http.StatusInternalServerError, createErr.Error())
			return
		}
		writeJSON(w, http.StatusCreated, note)
	})
	handler := authService.Middleware(auth.Require(server.idempotency(next)))
	body := `{"content":"recover after crash"}`
	path := "/api/v1/spaces/" + spaceID.String() + "/notes"
	requestFor := func(key string) *http.Request {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Idempotency-Key", key)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		return request
	}

	staleKey := "lease-expired-12345678"
	staleIdentity := idempotencyRequestIdentity(requestFor(staleKey), []byte(body))
	if _, err = db.Pool.Exec(ctx, `INSERT INTO idempotency_records(user_id,idempotency_key,method,path,state,created_at,expires_at) VALUES($1,$2,'POST',$3,'pending',now()-interval '3 minutes',now()-interval '1 second')`, userID, staleKey, staleIdentity); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, requestFor(staleKey))
	if response.Code != http.StatusCreated || calls != 1 {
		t.Fatalf("expired reservation response=%d calls=%d body=%s", response.Code, calls, response.Body.String())
	}
	var state string
	var completedExpiry time.Time
	if err = db.Pool.QueryRow(ctx, `SELECT state,expires_at FROM idempotency_records WHERE user_id=$1 AND idempotency_key=$2`, userID, staleKey).Scan(&state, &completedExpiry); err != nil {
		t.Fatal(err)
	}
	if state != "completed" || time.Until(completedExpiry) < 23*time.Hour {
		t.Fatalf("recovered reservation state=%s expiry=%s", state, completedExpiry)
	}
	replayed := httptest.NewRecorder()
	handler.ServeHTTP(replayed, requestFor(staleKey))
	if replayed.Code != http.StatusCreated || replayed.Header().Get("Idempotency-Replayed") != "true" || calls != 1 || !bytes.Equal(response.Body.Bytes(), replayed.Body.Bytes()) {
		t.Fatalf("completed replay response=%d replayed=%q calls=%d equal-body=%t", replayed.Code, replayed.Header().Get("Idempotency-Replayed"), calls, bytes.Equal(response.Body.Bytes(), replayed.Body.Bytes()))
	}

	activeKey := "lease-active-12345678"
	activeIdentity := idempotencyRequestIdentity(requestFor(activeKey), []byte(body))
	if _, err = db.Pool.Exec(ctx, `INSERT INTO idempotency_records(user_id,idempotency_key,method,path,state,expires_at) VALUES($1,$2,'POST',$3,'pending',now()+interval '2 minutes')`, userID, activeKey, activeIdentity); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, requestFor(activeKey))
	retryAfter, parseErr := strconv.Atoi(response.Header().Get("Retry-After"))
	if response.Code != http.StatusTooEarly || calls != 1 || parseErr != nil || retryAfter < 1 || retryAfter > int(idempotencyPendingLease/time.Second) {
		t.Fatalf("active reservation response=%d calls=%d retry-after=%q body=%s", response.Code, calls, response.Header().Get("Retry-After"), response.Body.String())
	}

	heartbeatKey := "lease-heartbeat-12345678"
	heartbeatIdentity := path + "#sha256:heartbeat"
	var heartbeatCreatedAt time.Time
	if err = db.Pool.QueryRow(ctx, `INSERT INTO idempotency_records(user_id,idempotency_key,method,path,state,expires_at) VALUES($1,$2,'POST',$3,'pending',now()+interval '1 second') RETURNING created_at`, userID, heartbeatKey, heartbeatIdentity).Scan(&heartbeatCreatedAt); err != nil {
		t.Fatal(err)
	}
	stopHeartbeat := server.maintainIdempotencyLeaseEvery(store.IdempotencyReservation{
		UserID: userID, Key: heartbeatKey, Method: http.MethodPost,
		Path: heartbeatIdentity, CreatedAt: heartbeatCreatedAt,
	}, 10*time.Millisecond)
	deadline := time.Now().Add(2 * time.Second)
	for {
		var heartbeatExpiry time.Time
		if err = db.Pool.QueryRow(ctx, `SELECT expires_at FROM idempotency_records WHERE user_id=$1 AND idempotency_key=$2`, userID, heartbeatKey).Scan(&heartbeatExpiry); err != nil {
			stopHeartbeat()
			t.Fatal(err)
		}
		if time.Until(heartbeatExpiry) > time.Minute {
			break
		}
		if time.Now().After(deadline) {
			stopHeartbeat()
			t.Fatalf("live reservation lease was not renewed: %s", heartbeatExpiry)
		}
		time.Sleep(10 * time.Millisecond)
	}
	stopHeartbeat()

	deletable, err := db.CreateNote(ctx, userID, store.Note{SpaceID: spaceID, Content: "delete exactly once"})
	if err != nil {
		t.Fatal(err)
	}
	deleteCalls := 0
	deleteServer := &Server{Store: db}
	deleteNext := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalls++
		if deleteErr := db.DeleteNote(r.Context(), userID, deletable.ID); deleteErr != nil {
			writeError(w, http.StatusInternalServerError, deleteErr.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	deleteHandler := authService.Middleware(auth.Require(deleteServer.idempotency(deleteNext)))
	deleteKey := "delete-once-12345678"
	deleteRequest := func() *http.Request {
		request := httptest.NewRequest(http.MethodDelete, "/api/v1/notes/"+deletable.ID.String(), nil)
		request.Header.Set("Idempotency-Key", deleteKey)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		return request
	}
	deleted := httptest.NewRecorder()
	deleteHandler.ServeHTTP(deleted, deleteRequest())
	replayedDelete := httptest.NewRecorder()
	deleteHandler.ServeHTTP(replayedDelete, deleteRequest())
	if deleted.Code != http.StatusNoContent || replayedDelete.Code != http.StatusNoContent || replayedDelete.Header().Get("Idempotency-Replayed") != "true" || deleteCalls != 1 || deleted.Body.Len() != 0 || replayedDelete.Body.Len() != 0 {
		t.Fatalf("atomic delete first=%d replay=%d replayed=%q calls=%d bodies=%d/%d", deleted.Code, replayedDelete.Code, replayedDelete.Header().Get("Idempotency-Replayed"), deleteCalls, deleted.Body.Len(), replayedDelete.Body.Len())
	}

	missingReservation := store.IdempotencyReservation{
		UserID: userID, Key: "missing-reservation", Method: http.MethodPost,
		Path: path + "#sha256:missing", CreatedAt: time.Now().UTC(), Status: http.StatusCreated,
	}
	_, err = db.CreateNote(store.WithIdempotencyReservation(ctx, missingReservation), userID, store.Note{SpaceID: spaceID, Content: "must roll back without completion"})
	if err == nil {
		t.Fatal("note mutation committed without an idempotency reservation")
	}
	var rolledBack int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM notes WHERE space_id=$1 AND content='must roll back without completion'`, spaceID).Scan(&rolledBack); err != nil {
		t.Fatal(err)
	}
	if rolledBack != 0 {
		t.Fatalf("%d note mutations survived failed atomic completion", rolledBack)
	}
}
