package httpapi

import (
	"testing"
	"time"
)

func TestRateLimiterSpendsThenRefuses(t *testing.T) {
	limiter := newRateLimiter()
	const perMinute = 60
	for attempt := range perMinute {
		if allowed, _ := limiter.allow("caller", perMinute); !allowed {
			t.Fatalf("request %d was refused while the bucket still had tokens", attempt)
		}
	}
	allowed, wait := limiter.allow("caller", perMinute)
	if allowed {
		t.Fatal("expected the bucket to be empty after spending its whole capacity")
	}
	if wait < time.Second {
		t.Fatalf("Retry-After must be at least a second, got %s", wait)
	}
}

func TestRateLimiterIsolatesCallers(t *testing.T) {
	limiter := newRateLimiter()
	for range 30 {
		limiter.allow("noisy", 30)
	}
	if allowed, _ := limiter.allow("quiet", 30); !allowed {
		t.Fatal("one caller exhausting its bucket must not affect another")
	}
}

func TestRateLimiterTreatsZeroAsUnlimited(t *testing.T) {
	limiter := newRateLimiter()
	for range 1000 {
		if allowed, _ := limiter.allow("caller", 0); !allowed {
			t.Fatal("a rate of zero disables the limit and must always allow")
		}
	}
}

func TestSecurityPolicyClampsOutOfRangeValues(t *testing.T) {
	policy := securityPolicy{LoginMaxFailures: 0, LoginLockoutMinutes: 100000, APIRatePerMinute: 1, AIRatePerMinute: -3, AIDailyLimit: -1}.normalized()
	if policy.LoginMaxFailures != 8 || policy.LoginLockoutMinutes != 15 || policy.APIRatePerMinute != 600 || policy.AIRatePerMinute != 6 || policy.AIDailyLimit != 80 {
		t.Fatalf("out of range settings must fall back to defaults, got %+v", policy)
	}
	// Zero is a meaningful choice for the daily cap: it means no cap at all.
	if got := (securityPolicy{AIDailyLimit: 0}).normalized().AIDailyLimit; got != 0 {
		t.Fatalf("a daily limit of zero means unlimited and must be preserved, got %d", got)
	}
}
