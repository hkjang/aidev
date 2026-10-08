package httpapi

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/ingest"
	"github.com/hkjang/invenqor/server/internal/spool"
)

func TestReplaySpoolDiscardsPermanentFailuresWithSharedDiagnostics(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	manager, err := spool.OpenDirectory(filepath.Join(t.TempDir(), "event-spool"))
	if err != nil {
		t.Fatal(err)
	}
	server.spool = manager

	invalidAgentID := uuid.NewString()
	invalidEventID := uuid.NewString()
	if _, err := manager.Append(
		invalidAgentID,
		invalidEventID,
		[]byte(`{"token":"ivq_at_should_never_be_logged"`),
	); err != nil {
		t.Fatal(err)
	}

	missingAgentID := uuid.NewString()
	appendHeartbeat(t, manager, missingAgentID, uuid.NewString())
	blockedAgentID := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO agents(
		 id,agent_id,hostname,status,auth_method,blocked_at
		 ) VALUES($1,$2,'blocked-spool-agent','blocked','bearer',$3)`,
		uuid.NewString(),
		blockedAgentID,
		time.Now().UTC(),
	); err != nil {
		t.Fatal(err)
	}
	appendHeartbeat(t, manager, blockedAgentID, uuid.NewString())

	replayed, err := server.ReplaySpool(context.Background())
	if err != nil || replayed != 0 {
		t.Fatalf("ReplaySpool() = %d/%v, want 0/nil", replayed, err)
	}
	pending, err := manager.Pending()
	if err != nil || len(pending) != 0 {
		t.Fatalf("permanent spool failures remain pending = %#v/%v", pending, err)
	}
	for _, eventCode := range []string{
		"SPOOLED_EVENT_INVALID",
		"SPOOLED_EVENT_AGENT_NOT_FOUND",
		"SPOOLED_EVENT_AGENT_BLOCKED",
	} {
		var instanceID, details string
		if err := runtime.DB().QueryRow(
			`SELECT instance_id,details_json FROM diagnostic_logs
			 WHERE event_code=$1`,
			eventCode,
		).Scan(&instanceID, &details); err != nil {
			t.Fatalf("read %s diagnostic: %v", eventCode, err)
		}
		if instanceID == "" || !strings.Contains(details, `"segment"`) {
			t.Fatalf("%s diagnostic = %q/%s", eventCode, instanceID, details)
		}
		if strings.Contains(details, "should_never_be_logged") {
			t.Fatalf("%s diagnostic leaked raw event: %s", eventCode, details)
		}
	}
}

func appendHeartbeat(
	t *testing.T,
	manager *spool.Manager,
	agentID string,
	eventID string,
) {
	t.Helper()
	raw, err := json.Marshal(ingest.Envelope{
		SchemaVersion: 1,
		EventID:       eventID,
		AgentID:       agentID,
		CreatedAt:     uint64(time.Now().Unix()),
		Kind:          "heartbeat",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(agentID, eventID, raw); err != nil {
		t.Fatal(err)
	}
}
