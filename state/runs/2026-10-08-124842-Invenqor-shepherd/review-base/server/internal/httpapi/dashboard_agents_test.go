package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Blocking is how an operator retires a machine: the console offers 차단 next to
// each Agent and nothing ever deletes the row. So a fleet accumulates blocked
// Agents, and if they still count towards "needs attention" that tile only ever
// goes up - a warning permanently on is a warning nobody reads, and clicking
// 차단 to tidy up appears to do nothing.
//
// The release distribution already counts only unblocked Agents. This holds the
// dashboard to the same idea of who is in the fleet.
func TestBlockedAgentsAreNotCountedAsNeedingAttention(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, _ := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()

	insert := func(hostname string, lastSeen time.Time, blocked bool) {
		t.Helper()
		var blockedAt any
		status := "active"
		if blocked {
			blockedAt = now
			status = "blocked"
		}
		if _, err := runtime.DB().Exec(
			`INSERT INTO agents(id, agent_id, hostname, status, last_seen_at, blocked_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			uuid.NewString(), uuid.NewString(), hostname, status, lastSeen, blockedAt,
		); err != nil {
			t.Fatal(err)
		}
	}

	insert("reporting", now, false)
	insert("silent", now.Add(-2*time.Hour), false)
	insert("retired", now.Add(-90*24*time.Hour), true)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/statistics", nil)
	request.AddCookie(cookie)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("dashboard = %d body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Agents struct {
			Total     int64 `json:"total"`
			Healthy   int64 `json:"healthy"`
			Attention int64 `json:"attention"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}

	if payload.Agents.Total != 2 {
		t.Errorf(
			"agents.total = %d, want 2: a blocked Agent is out of service and not "+
				"part of the fleet expected to report",
			payload.Agents.Total,
		)
	}
	if payload.Agents.Healthy != 1 {
		t.Errorf("agents.healthy = %d, want 1", payload.Agents.Healthy)
	}
	if payload.Agents.Attention != 1 {
		t.Errorf(
			"agents.attention = %d, want 1: only the silent Agent needs looking "+
				"at, and blocking the retired one must clear it from this count",
			payload.Agents.Attention,
		)
	}
}
