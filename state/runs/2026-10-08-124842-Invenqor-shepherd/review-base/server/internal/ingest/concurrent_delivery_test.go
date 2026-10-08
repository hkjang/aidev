package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

// An agent whose delivery times out retries the same event, and with more than
// one server instance behind a load balancer the retry can land while the first
// attempt is still running. The same envelope then reaches two instances at
// once.
//
// Exactly one of them must do the work. The rest must either be told the event
// is already in progress or see it as a duplicate - never process it a second
// time, because that would double every asset the snapshot carries.
//
// Two separate things hold that today, which is worth knowing before changing
// either. Process runs in a single transaction, so the 'processing' status it
// writes is never committed and a second instance blocks on the conflicting
// insert rather than reading it; by the time it proceeds the event is already
// 'processed'. If that transaction is ever split, the status does become
// visible and the ErrInProgress branch takes over instead. Both were checked by
// making each change in turn - the invariant survived both, and neither layer
// is redundant if the other is removed.
//
// This pins the invariant rather than either mechanism, because it is the
// invariant a user notices: eight deliveries of one snapshot must not produce
// sixteen assets.
func TestTheSameEventDeliveredConcurrentlyIsProcessedOnce(t *testing.T) {
	runtime, agent, service := testService(t)
	envelope := inventoryEnvelope(agent.AgentID, []AssetRecord{
		record("host-source", "system", `{"hostname":"node-01"}`),
		record("pkg-source", "software.package", `{"name":"curl","version":"1"}`),
	})
	if err := envelope.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(envelope)

	const deliveries = 8
	results := make([]Result, deliveries)
	failures := make([]error, deliveries)
	start := make(chan struct{})
	var running sync.WaitGroup
	for index := range deliveries {
		running.Add(1)
		go func() {
			defer running.Done()
			// Released together so the deliveries genuinely overlap.
			<-start
			result, err := service.Process(
				context.Background(), agent, envelope, raw, "test",
			)
			results[index], failures[index] = result, err
		}()
	}
	close(start)
	running.Wait()

	processed, duplicates, inProgress := 0, 0, 0
	for index, err := range failures {
		switch {
		case err == nil && results[index].Duplicate:
			duplicates++
		case err == nil:
			processed++
		case errors.Is(err, ErrInProgress):
			inProgress++
		default:
			t.Errorf("delivery %d failed unexpectedly: %v", index, err)
		}
	}
	if processed != 1 {
		t.Errorf(
			"%d deliveries processed the event, want exactly 1 (%d duplicates, %d in progress)",
			processed, duplicates, inProgress,
		)
	}

	// The real damage a second processing run would do is in the data.
	assertCount(t, runtime, "agent_events", 1)
	assertCountWhere(t, runtime, "asset_sources", "deleted_at IS NULL", 2)
}
