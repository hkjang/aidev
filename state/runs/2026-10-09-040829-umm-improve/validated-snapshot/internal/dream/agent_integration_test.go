package dream

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// The assistant is only as safe as what it is actually offered on the wire.
// The unit test checks the Go tool list; this one checks the JSON the gateway
// receives, and that a thought marked as excluded from AI stays unreachable
// through a tool call the same way it does through /ai/ask.
func TestThoughtAgentReadsOnlyAndHonorsExclusionIntegration(t *testing.T) {
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

	userID, spaceID := uuid.New(), uuid.New()
	visibleID, hiddenID := uuid.New(), uuid.New()
	username := "agent_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'조력자 공간')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM spaces WHERE id=$1`, spaceID)
	if _, err = db.Pool.Exec(ctx, `
		INSERT INTO notes(id,space_id,author_id,content,ai_excluded) VALUES
		($1,$3,$4,'검색 노출 생각: 회고 주기를 격주로 바꾸는 실험',false),
		($2,$3,$4,'비공개 생각: 회고 주기 관련 개인 기록',true)`,
		visibleID, hiddenID, spaceID, userID); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var sentToolNames []string
	var turns int
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var sent struct {
			Tools []struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tools"`
		}
		_ = json.Unmarshal(raw, &sent)
		mu.Lock()
		for _, tool := range sent.Tools {
			sentToolNames = append(sentToolNames, tool.Function.Name)
		}
		turns++
		turn := turns
		mu.Unlock()

		if turn == 1 {
			// The model asks to look, exactly as a real one does.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{"finish_reason": "tool_calls", "message": map[string]any{
					"role": "assistant",
					"tool_calls": []any{map[string]any{
						"id": "call_1", "type": "function",
						"function": map[string]any{"name": "search_thoughts", "arguments": `{"query":"회고 주기"}`},
					}},
				}}},
				"usage": map[string]any{"prompt_tokens": 30, "completion_tokens": 12},
			})
			return
		}
		// Echo what the tool returned so the test can assert on what the model saw.
		var received struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(raw, &received)
		var seen strings.Builder
		for _, message := range received.Messages {
			if message.Role == "tool" {
				seen.WriteString(message.Content)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{
				"role": "assistant", "content": "조회한 내용: " + seen.String(),
			}}},
			"usage": map[string]any{"prompt_tokens": 40, "completion_tokens": 20},
		})
	}))
	defer gateway.Close()

	var previousDream Config
	if db.GetSetting(ctx, "dream", &previousDream) == nil {
		defer db.PutSetting(context.Background(), "dream", previousDream, userID)
	}
	var previousGateway GatewayConfig
	if db.GetSetting(ctx, "ai_gateway", &previousGateway) == nil {
		defer db.PutSetting(context.Background(), "ai_gateway", previousGateway, userID)
	}
	if err = db.PutSetting(ctx, "dream", Config{Model: "agent-model", TokenLimit: 512}, userID); err != nil {
		t.Fatal(err)
	}
	if err = db.PutSetting(ctx, "ai_gateway", GatewayConfig{BaseURL: gateway.URL, TimeoutSeconds: 5}, userID); err != nil {
		t.Fatal(err)
	}

	service := &Service{Store: db}
	result, err := service.RunAgent(ctx, userID, "회고 주기에 대해 내가 무슨 생각을 했는지 살펴봐 줘")
	if err != nil {
		t.Fatal(err)
	}

	// The boundary, checked on the wire rather than in the tool list.
	mu.Lock()
	offered := append([]string(nil), sentToolNames...)
	mu.Unlock()
	if len(offered) == 0 {
		t.Fatal("no tools were sent to the gateway; the assistant could not look at anything")
	}
	for _, name := range offered {
		for _, write := range []string{"create", "update", "delete", "merge", "connect", "move", "write"} {
			if strings.Contains(strings.ToLower(name), write) {
				t.Fatalf("the gateway was offered %q, which can change something", name)
			}
		}
	}

	if !strings.Contains(result.Answer, "검색 노출 생각") {
		t.Errorf("the visible thought never reached the model: %q", result.Answer)
	}
	if strings.Contains(result.Answer, "비공개 생각") {
		t.Fatalf("a thought marked as excluded from AI reached the model: %q", result.Answer)
	}
	if result.Excluded == 0 {
		t.Error("the excluded thought was not counted, so the person cannot tell something was held back")
	}
	if len(result.Steps) != 1 || result.Steps[0].Tool != "search_thoughts" {
		t.Errorf("steps did not record the lookup: %+v", result.Steps)
	}
	if result.Truncated {
		t.Error("a two-turn run reported itself as cut short")
	}
	if result.Model != "agent-model" {
		t.Errorf("model = %q", result.Model)
	}
}

// A model that never stops calling tools must still end, and the person must be
// told the answer was cut short rather than concluded.
func TestThoughtAgentStopsAtStepLimitIntegration(t *testing.T) {
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
	username := "agent_loop_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)

	var mu sync.Mutex
	var turns int
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		turns++
		mu.Unlock()
		// The last turn is sent without tools; answer it as a real model would.
		var sent struct {
			Tools []any `json:"tools"`
		}
		_ = json.Unmarshal(raw, &sent)
		if len(sent.Tools) == 0 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{
					"role": "assistant", "content": "지금까지 확인한 범위에서의 답입니다.",
				}}},
				"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"finish_reason": "tool_calls", "message": map[string]any{
				"role": "assistant",
				"tool_calls": []any{map[string]any{
					"id": "call_loop", "type": "function",
					"function": map[string]any{"name": "find_open_questions", "arguments": `{}`},
				}},
			}}},
			"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5},
		})
	}))
	defer gateway.Close()

	var previousDream Config
	if db.GetSetting(ctx, "dream", &previousDream) == nil {
		defer db.PutSetting(context.Background(), "dream", previousDream, userID)
	}
	var previousGateway GatewayConfig
	if db.GetSetting(ctx, "ai_gateway", &previousGateway) == nil {
		defer db.PutSetting(context.Background(), "ai_gateway", previousGateway, userID)
	}
	if err = db.PutSetting(ctx, "dream", Config{Model: "agent-loop-model", TokenLimit: 256}, userID); err != nil {
		t.Fatal(err)
	}
	if err = db.PutSetting(ctx, "ai_gateway", GatewayConfig{BaseURL: gateway.URL, TimeoutSeconds: 5}, userID); err != nil {
		t.Fatal(err)
	}

	service := &Service{Store: db}
	result, err := service.RunAgent(ctx, userID, "계속 찾아봐")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated {
		t.Error("the run hit the step limit but did not say the answer was cut short")
	}
	if result.Answer == "" {
		t.Error("hitting the limit lost everything that was learned")
	}
	mu.Lock()
	total := turns
	mu.Unlock()
	if total != maxAgentSteps+1 {
		t.Errorf("model calls = %d, want %d (%d steps plus one final answer)", total, maxAgentSteps+1, maxAgentSteps)
	}
	if len(result.Steps) != maxAgentSteps {
		t.Errorf("steps = %d, want %d", len(result.Steps), maxAgentSteps)
	}
}

// A look that costs nothing is a look someone can run in a loop. Each turn is a
// model call and must be charged to the person who asked; this failed once,
// because the quota was consumed against a nil user, which skips it entirely.
func TestThoughtAgentChargesEveryTurnToThePersonIntegration(t *testing.T) {
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
	username := "agent_quota_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var sent struct {
			Tools []any `json:"tools"`
		}
		_ = json.Unmarshal(raw, &sent)
		if len(sent.Tools) == 0 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"role": "assistant", "content": "답입니다."}}},
				"usage":   map[string]any{"prompt_tokens": 5, "completion_tokens": 5},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"finish_reason": "tool_calls", "message": map[string]any{
				"role": "assistant",
				"tool_calls": []any{map[string]any{
					"id": "call_cost", "type": "function",
					"function": map[string]any{"name": "find_contradictions", "arguments": `{}`},
				}},
			}}},
			"usage": map[string]any{"prompt_tokens": 5, "completion_tokens": 5},
		})
	}))
	defer gateway.Close()

	var previousDream Config
	if db.GetSetting(ctx, "dream", &previousDream) == nil {
		defer db.PutSetting(context.Background(), "dream", previousDream, userID)
	}
	var previousGateway GatewayConfig
	if db.GetSetting(ctx, "ai_gateway", &previousGateway) == nil {
		defer db.PutSetting(context.Background(), "ai_gateway", previousGateway, userID)
	}
	if err = db.PutSetting(ctx, "dream", Config{Model: "agent-cost-model", TokenLimit: 256}, userID); err != nil {
		t.Fatal(err)
	}
	if err = db.PutSetting(ctx, "ai_gateway", GatewayConfig{BaseURL: gateway.URL, TimeoutSeconds: 5}, userID); err != nil {
		t.Fatal(err)
	}

	service := &Service{Store: db}
	if _, err = service.RunAgent(ctx, userID, "상충하는 생각이 있는지 살펴봐"); err != nil {
		t.Fatal(err)
	}

	var consumed int
	if err = db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM ai_quota_reservations WHERE user_id=$1 AND consumed_at IS NOT NULL`,
		userID).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if consumed != maxAgentSteps+1 {
		t.Fatalf("charged %d model calls to the person, but the run made %d; a free look can be run in a loop",
			consumed, maxAgentSteps+1)
	}
}
