package spool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestAppendIsSecureAndIdempotent(t *testing.T) {
	manager, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := manager.Append("agent-1", "event-1", []byte(`{"one":1}`))
	if err != nil || duplicate {
		t.Fatalf("first Append() = %v/%v", duplicate, err)
	}
	duplicate, err = manager.Append("agent-1", "event-1", []byte(`{"two":2}`))
	if err != nil || !duplicate {
		t.Fatalf("second Append() = %v/%v", duplicate, err)
	}
	paths, err := manager.Pending()
	if err != nil || len(paths) != 1 {
		t.Fatalf("Pending() = %#v/%v", paths, err)
	}
	bytes, err := manager.Read(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(bytes) != `{"one":1}` {
		t.Fatalf("spool segment was overwritten: %s", bytes)
	}
	info, err := os.Stat(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("segment mode = %o, want 600", info.Mode().Perm())
	}
	directoryInfo, err := os.Stat(filepath.Dir(paths[0]))
	if err != nil {
		t.Fatal(err)
	}
	if directoryInfo.Mode().Perm() != 0o700 {
		t.Fatalf("spool directory mode = %o, want 700", directoryInfo.Mode().Perm())
	}
	if err := manager.Acknowledge(paths[0]); err != nil {
		t.Fatal(err)
	}
}

func TestAppendPublishesOneCompleteSegmentUnderConcurrentRetries(t *testing.T) {
	manager, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	const writers = 16
	type appendResult struct {
		duplicate bool
		err       error
	}
	results := make(chan appendResult, writers)
	start := make(chan struct{})
	var group sync.WaitGroup
	validPayloads := make(map[string]struct{}, writers)
	for index := 0; index < writers; index++ {
		payload := fmt.Sprintf(`{"event_id":"event-1","writer":%d}`, index)
		validPayloads[payload] = struct{}{}
		group.Add(1)
		go func(raw string) {
			defer group.Done()
			<-start
			duplicate, appendErr := manager.Append(
				"agent-1", "event-1", []byte(raw),
			)
			results <- appendResult{duplicate: duplicate, err: appendErr}
		}(payload)
	}
	close(start)
	group.Wait()
	close(results)
	winners := 0
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if !result.duplicate {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("non-duplicate writers = %d, want 1", winners)
	}
	paths, err := manager.Pending()
	if err != nil || len(paths) != 1 {
		t.Fatalf("Pending() = %#v/%v", paths, err)
	}
	raw, err := manager.Read(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, valid := validPayloads[string(raw)]; !valid || !json.Valid(raw) {
		t.Fatalf("published segment is partial or foreign: %q", raw)
	}
	entries, err := os.ReadDir(manager.Directory())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Fatalf("successful append left staging file %q", entry.Name())
		}
	}
}

func TestPendingIgnoresAndReplayReclaimsCrashedStagingSegment(t *testing.T) {
	directory := t.TempDir()
	manager, err := OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(directory, stagingPrefix+"crashed.tmp")
	if err := os.WriteFile(stagingPath, []byte(`{"event_id":`), 0o600); err != nil {
		t.Fatal(err)
	}
	paths, err := manager.Pending()
	if err != nil || len(paths) != 0 {
		t.Fatalf("Pending() exposed partial staging file = %#v/%v", paths, err)
	}
	old := time.Now().Add(-2 * stagingCleanupGrace)
	if err := os.Chtimes(stagingPath, old, old); err != nil {
		t.Fatal(err)
	}
	lease, acquired, err := manager.AcquireReplayLease()
	if err != nil || !acquired {
		t.Fatalf("AcquireReplayLease() = %#v/%v/%v", lease, acquired, err)
	}
	if err := lease.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stagingPath); !os.IsNotExist(err) {
		t.Fatalf("crashed staging file remains after replay cleanup: %v", err)
	}
}

func TestStagingCleanupDoesNotRemovePausedWriter(t *testing.T) {
	directory := t.TempDir()
	manager, err := OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := os.CreateTemp(directory, stagingPrefix+"*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	path := writer.Name()
	locked, err := tryLockReplayFile(writer)
	if err != nil || !locked {
		t.Fatalf("stage writer lock = %v/%v", locked, err)
	}
	old := time.Now().Add(-2 * stagingCleanupGrace)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	lease, acquired, err := manager.AcquireReplayLease()
	if err != nil || !acquired {
		t.Fatalf("AcquireReplayLease() = %#v/%v/%v", lease, acquired, err)
	}
	if err := lease.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("paused writer stage was removed: %v", err)
	}
	if err := unlockReplayFile(writer); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	lease, acquired, err = manager.AcquireReplayLease()
	if err != nil || !acquired {
		t.Fatalf("second AcquireReplayLease() = %#v/%v/%v", lease, acquired, err)
	}
	if err := lease.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("abandoned writer stage remains: %v", err)
	}
}

func TestPendingOrdersByDurableServerArrivalAndUsesEventIDOnlyForTies(t *testing.T) {
	manager, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	type event struct {
		EventID   string `json:"event_id"`
		CreatedAt uint64 `json:"created_at"`
	}
	// Filename order is event-a, event-b, event-z. Replay order must instead
	// use durable server arrival (mtime), with event ID only as a stable tie.
	inputs := []struct {
		fileEventID string
		payload     event
		arrival     time.Duration
	}{
		{"event-a", event{EventID: "0002", CreatedAt: 100}, 3 * time.Second},
		{"event-z", event{EventID: "0003", CreatedAt: 300}, time.Second},
		{"event-b", event{EventID: "0001", CreatedAt: 200}, 2 * time.Second},
	}
	arrivalBase := time.Now().Add(-time.Hour)
	for _, input := range inputs {
		raw, marshalErr := json.Marshal(input.payload)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if _, appendErr := manager.Append("agent-1", input.fileEventID, raw); appendErr != nil {
			t.Fatal(appendErr)
		}
		path := filepath.Join(
			manager.Directory(), "agent-1_"+input.fileEventID+".json",
		)
		arrival := arrivalBase.Add(input.arrival)
		if err := os.Chtimes(path, arrival, arrival); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := manager.Pending()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"0003", "0001", "0002"}
	for index, path := range paths {
		raw, readErr := manager.Read(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		var got event
		if json.Unmarshal(raw, &got) != nil || got.EventID != want[index] {
			t.Fatalf("Pending()[%d] = %s, want event %s", index, raw, want[index])
		}
	}
}

func TestSharedReplayLockAndCrashRecovery(t *testing.T) {
	directory := t.TempDir()
	first, err := OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	second, err := OpenDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"event_id":"event-1","created_at":100}`)
	if duplicate, err := first.Append("agent-1", "event-1", raw); err != nil || duplicate {
		t.Fatalf("Append() = %v, %v", duplicate, err)
	}

	firstLease, acquired, err := first.AcquireReplayLease()
	if err != nil || !acquired {
		t.Fatalf("first AcquireReplayLease() = %#v/%v/%v", firstLease, acquired, err)
	}
	if lease, acquired, err := second.AcquireReplayLease(); err != nil || acquired || lease != nil {
		t.Fatalf("second concurrent lease = %#v/%v/%v", lease, acquired, err)
	}
	if duplicate, err := second.Append("agent-1", "event-1", raw); err != nil || !duplicate {
		t.Fatalf("Append() during replay lock = %v/%v, want duplicate", duplicate, err)
	}

	// Closing the descriptor models a crashed process: the kernel releases the
	// advisory lock even though the persistent lock file remains.
	if err := firstLease.file.Close(); err != nil {
		t.Fatal(err)
	}
	firstLease.file = nil
	secondLease, acquired, err := second.AcquireReplayLease()
	if err != nil || !acquired {
		t.Fatalf("lease after crash = %#v/%v/%v", secondLease, acquired, err)
	}
	pending, err := second.Pending()
	if err != nil || len(pending) != 1 {
		t.Fatalf("Pending() after crash = %#v/%v", pending, err)
	}
	got, err := second.Read(pending[0])
	if err != nil || string(got) != string(raw) {
		t.Fatalf("recovered Read() = %s/%v", got, err)
	}
	if err := second.Acknowledge(pending[0]); err != nil {
		t.Fatal(err)
	}
	if err := secondLease.Release(); err != nil {
		t.Fatal(err)
	}
	pending, err = second.Pending()
	if err != nil || len(pending) != 0 {
		t.Fatalf("Pending() after replay = %#v/%v", pending, err)
	}
	if _, err := os.Stat(filepath.Join(directory, ".replay.lock")); err != nil {
		t.Fatalf("persistent lock file was removed: %v", err)
	}
}
