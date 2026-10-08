package dream

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

func TestScheduledGatewayHonorsDurableDailyQuotaIntegration(t *testing.T) {
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
	limit, err := db.AIDailyLimit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if limit == 0 {
		t.Skip("daily AI quota is disabled in this integration database")
	}

	var gatewayCalls atomic.Int64
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		gatewayCalls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message":       map[string]string{"content": `{"content":"두 생각을 연결한 새로운 한국어 제안입니다.","type":"connection","rationale":"두 원본을 연결했습니다.","suggestedAction":"작게 시험합니다.","sourceRefs":[1,2]}`},
				"finish_reason": "stop",
			}},
			"usage": map[string]int{"prompt_tokens": 10, "completion_tokens": 20},
		})
	}))
	defer gateway.Close()
	service := &Service{Store: db}
	cfg := Config{Model: "quota-model", TokenLimit: 512}
	gatewayConfig := GatewayConfig{BaseURL: gateway.URL, TimeoutSeconds: 2}
	sources := []sourceNote{{ID: uuid.New(), Content: "첫 번째 생각"}, {ID: uuid.New(), Content: "두 번째 생각"}}

	blockedUser := uuid.New()
	blockedName := "scheduled_quota_" + blockedUser.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, blockedUser, blockedName); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, blockedUser)
	if _, err = db.Pool.Exec(ctx, `
		INSERT INTO ai_quota_reservations(user_id,created_at,expires_at,consumed_at)
		SELECT $1,now()-interval '1 minute',now()+interval '23 hours',now()-interval '1 minute'
		FROM generate_series(1,$2)`, blockedUser, limit); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, _, err = service.callGatewayWithGuidance(ctx, blockedUser, cfg, gatewayConfig, sources, "free", "")
	if !errors.Is(err, ErrAIDailyLimit) {
		t.Fatalf("scheduled generation crossed the daily quota: %v", err)
	}
	if gatewayCalls.Load() != 0 {
		t.Fatalf("quota-blocked scheduled generation contacted the gateway %d times", gatewayCalls.Load())
	}

	consumedUser := uuid.New()
	consumedName := "durable_quota_" + consumedUser.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, consumedUser, consumedName); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, consumedUser)
	if _, _, _, _, _, err = service.callGatewayWithGuidance(ctx, consumedUser, cfg, gatewayConfig, sources, "free", ""); err != nil {
		t.Fatal(err)
	}
	var consumed, calls int
	if err = db.Pool.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM ai_quota_reservations WHERE user_id=$1 AND consumed_at IS NOT NULL AND expires_at>now()+interval '23 hours'),
		  (SELECT count(*) FROM ai_calls WHERE user_id=$1)`, consumedUser).Scan(&consumed, &calls); err != nil {
		t.Fatal(err)
	}
	if gatewayCalls.Load() != 1 || consumed != 1 || calls != 0 {
		t.Fatalf("gateway usage was not durable before optional logging: gateway=%d consumed=%d calls=%d", gatewayCalls.Load(), consumed, calls)
	}
}

func TestRecordAICallOutlivesRequestCancellationIntegration(t *testing.T) {
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
	username := "canceled_log_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	defer db.Pool.Exec(ctx, `DELETE FROM ai_calls WHERE user_id=$1`, userID)

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	service := &Service{Store: db}
	service.recordAICall(canceled, userID, uuid.Nil, store.PurposeAssist, "quota-model", 1, 2, time.Millisecond, nil, GatewayConfig{}, "")
	var calls int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM ai_calls WHERE user_id=$1`, userID).Scan(&calls); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("canceled request lost %d AI call logs", 1-calls)
	}
}

func TestEvaluationHonorsTheInitiatingUsersDailyQuotaIntegration(t *testing.T) {
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
	limit, err := db.AIDailyLimit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if limit == 0 {
		t.Skip("daily AI quota is disabled in this integration database")
	}

	userID := uuid.New()
	username := "eval_quota_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)

	var gatewayCalls atomic.Int64
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		gatewayCalls.Add(1)
		http.Error(w, "quota should block this request", http.StatusInternalServerError)
	}))
	defer gateway.Close()
	var previousDream Config
	if err = db.GetSetting(ctx, "dream", &previousDream); err != nil {
		t.Fatal(err)
	}
	defer db.PutSetting(context.Background(), "dream", previousDream, userID)
	var previousGateway GatewayConfig
	if err = db.GetSetting(ctx, "ai_gateway", &previousGateway); err != nil {
		t.Fatal(err)
	}
	defer db.PutSetting(context.Background(), "ai_gateway", previousGateway, userID)
	if err = db.PutSetting(ctx, "dream", Config{Model: "eval-quota-model", TokenLimit: 512}, userID); err != nil {
		t.Fatal(err)
	}
	if err = db.PutSetting(ctx, "ai_gateway", GatewayConfig{BaseURL: gateway.URL, TimeoutSeconds: 2}, userID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `
		INSERT INTO ai_quota_reservations(user_id,created_at,expires_at,consumed_at)
		SELECT $1,now()-interval '1 minute',now()+interval '23 hours',now()-interval '1 minute'
		FROM generate_series(1,$2)`, userID, limit); err != nil {
		t.Fatal(err)
	}

	service := &Service{Store: db}
	_, err = service.Evaluate(ctx, userID, EvalRequest{
		DreamType:  "connection",
		InputNotes: []string{"첫 번째 평가 생각", "두 번째 평가 생각"},
	})
	if !errors.Is(err, ErrAIDailyLimit) {
		t.Fatalf("evaluation bypassed the daily quota: %v", err)
	}
	if calls := gatewayCalls.Load(); calls != 0 {
		t.Fatalf("quota-blocked evaluation contacted the gateway %d times", calls)
	}
}
