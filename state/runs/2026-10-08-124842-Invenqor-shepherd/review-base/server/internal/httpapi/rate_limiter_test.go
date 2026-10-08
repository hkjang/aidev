package httpapi

import (
	"fmt"
	"testing"
	"time"
)

// The rate limiters are keyed by whatever identifies the caller: an agent ID
// for ingest, an API key ID for the external API, and a source address for
// enrollment and preflight - which need no credentials at all. Nothing ever
// removed a key again, so a Server that runs for months kept one record for
// every address that ever reached it, and a single host holding an IPv6 /64
// can spend addresses faster than an operator notices the memory going.
func TestTheRateLimiterForgetsAKeyWhoseWindowHasPassed(t *testing.T) {
	limiter := newAgentRateLimiter(5, time.Minute)
	clock := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return clock }

	for index := range 1000 {
		limiter.Allow(fmt.Sprintf("203.0.113.%d", index))
	}
	if len(limiter.entries) != 1000 {
		t.Fatalf("expected the addresses of the current window to be counted, got %d", len(limiter.entries))
	}

	// Two windows on, every one of those counters means nothing: the caller
	// would be allowed again whether or not the record survived.
	clock = clock.Add(2 * time.Minute)
	limiter.Allow("198.51.100.1")
	if len(limiter.entries) != 1 {
		t.Fatalf("expected the passed windows to be swept, got %d entries", len(limiter.entries))
	}
}

// The sweep must not cost a full pass over the map on every request, or a
// burst of distinct addresses becomes quadratic work on the request path.
func TestTheRateLimiterSweepsAtMostOncePerWindow(t *testing.T) {
	limiter := newAgentRateLimiter(5, time.Minute)
	clock := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return clock }

	limiter.Allow("203.0.113.1")
	swept := limiter.sweptAt
	clock = clock.Add(30 * time.Second)
	limiter.Allow("203.0.113.2")
	if !limiter.sweptAt.Equal(swept) {
		t.Fatalf("expected no second sweep inside the same window, swept at %s", limiter.sweptAt)
	}
	if len(limiter.entries) != 2 {
		t.Fatalf("expected both live counters to be kept, got %d", len(limiter.entries))
	}
}

// Sweeping must not hand a caller a fresh allowance. Nothing pinned the limit
// itself before, so this also states the rule the enrollment, ingest and API
// key endpoints all rely on.
func TestTheRateLimiterRefusesAKeyOverItsLimitUntilTheWindowPasses(t *testing.T) {
	limiter := newAgentRateLimiter(3, time.Minute)
	clock := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return clock }

	for attempt := range 3 {
		clock = clock.Add(time.Second)
		if !limiter.Allow("agent-1") {
			t.Fatalf("attempt %d was refused inside the limit", attempt+1)
		}
	}
	clock = clock.Add(time.Second)
	if limiter.Allow("agent-1") {
		t.Fatal("expected the fourth attempt inside the window to be refused")
	}

	clock = clock.Add(time.Minute)
	if !limiter.Allow("agent-1") {
		t.Fatal("expected a new window to allow the caller again")
	}
}
