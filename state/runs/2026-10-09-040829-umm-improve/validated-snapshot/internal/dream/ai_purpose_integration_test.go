package dream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// The column existing and the column being filled are different facts.
//
// Every call site passes a purpose, and the compiler checks the type but not
// the value: a site could pass the wrong constant, or a new one could be added
// with whatever was nearest. So this drives the real call paths and reads back
// what landed, rather than asserting that the code says what it says.
func TestAICallsRecordWhatTheyWereForIntegration(t *testing.T) {
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

	userID, spaceID, noteID := uuid.New(), uuid.New(), uuid.New()
	username := "purpose_" + userID.String()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, userID, username); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'용도 공간')`, spaceID, userID); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM spaces WHERE id=$1`, spaceID)
	if _, err = db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'회고 주기를 격주로 줄여 보자')`,
		noteID, spaceID, userID); err != nil {
		t.Fatal(err)
	}

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{
				"role": "assistant", "content": "답",
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
	if err = db.PutSetting(ctx, "dream", Config{Model: "purpose-model", TokenLimit: 512}, userID); err != nil {
		t.Fatal(err)
	}
	if err = db.PutSetting(ctx, "ai_gateway", GatewayConfig{BaseURL: gateway.URL, TimeoutSeconds: 5}, userID); err != nil {
		t.Fatal(err)
	}

	service := &Service{Store: db}

	// One call down each path a person can take, driven for real.
	if _, err := service.Assist(ctx, userID, []uuid.UUID{noteID}, "summarize"); err != nil {
		t.Fatalf("assist: %v", err)
	}
	if _, err := service.Ask(ctx, userID, "회고 주기에 대해 뭘 적었지?"); err != nil {
		t.Fatalf("ask: %v", err)
	}
	// Complete is the shared door two deck features use, and the purpose comes
	// from the caller. Both are driven so a single label for both would show.
	if _, err := service.Complete(ctx, userID, store.PurposeDeckHeadings, "system", "user", 100); err != nil {
		t.Fatalf("headings: %v", err)
	}
	if _, err := service.Complete(ctx, userID, store.PurposeDeckSections, "system", "user", 100); err != nil {
		t.Fatalf("sections: %v", err)
	}

	rows, err := db.Pool.Query(ctx, `SELECT purpose,count(*) FROM ai_calls WHERE user_id=$1 GROUP BY purpose`, userID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]int{}
	for rows.Next() {
		var purpose string
		var n int
		if err := rows.Scan(&purpose, &n); err != nil {
			t.Fatal(err)
		}
		got[purpose] = n
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	for _, want := range []store.Purpose{store.PurposeAssist, store.PurposeAsk,
		store.PurposeDeckHeadings, store.PurposeDeckSections} {
		if got[string(want)] == 0 {
			t.Errorf("no call was recorded as %q; recorded: %v", want, got)
		}
	}
	// Nothing landed unlabelled, which is what a missed call site looks like.
	if got[""] != 0 {
		t.Errorf("%d calls were recorded with no purpose: %v", got[""], got)
	}
}
