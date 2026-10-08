package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/dream"
	"github.com/hkjang/umm/internal/store"
)

// securityPolicy holds the operator tunable half of the security settings.
// Zero or out of range values fall back to the defaults so a partially written
// settings row can never disable a protection.
type securityPolicy struct {
	LoginMaxFailures    int `json:"login_max_failures"`
	LoginLockoutMinutes int `json:"login_lockout_minutes"`
	APIRatePerMinute    int `json:"api_rate_per_minute"`
	AIRatePerMinute     int `json:"ai_rate_per_minute"`
	AIDailyLimit        int `json:"ai_daily_limit"`
}

func (p securityPolicy) normalized() securityPolicy {
	p.LoginMaxFailures = clampSetting(p.LoginMaxFailures, 3, 100, 8)
	p.LoginLockoutMinutes = clampSetting(p.LoginLockoutMinutes, 1, 24*60, 15)
	p.APIRatePerMinute = clampSetting(p.APIRatePerMinute, 30, 100000, 600)
	p.AIRatePerMinute = clampSetting(p.AIRatePerMinute, 1, 600, 6)
	p.AIDailyLimit = store.NormalizeAIDailyLimit(p.AIDailyLimit)
	return p
}

func clampSetting(value, low, high, fallback int) int {
	if value < low || value > high {
		return fallback
	}
	return value
}

func (p securityPolicy) lockout() time.Duration {
	return time.Duration(p.LoginLockoutMinutes) * time.Minute
}

// securityPolicyTTL bounds how long a request keeps using settings that an
// administrator has already changed. Explicit invalidation on save makes the
// common case immediate; the TTL only covers changes made on another instance.
const securityPolicyTTL = 30 * time.Second

func (s *Server) securityPolicy(ctx context.Context) securityPolicy {
	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	if !s.policyLoadedAt.IsZero() && time.Since(s.policyLoadedAt) < securityPolicyTTL {
		return s.policy
	}
	var policy securityPolicy
	_ = s.Store.GetSetting(ctx, "security", &policy)
	s.policy = policy.normalized()
	s.policyLoadedAt = time.Now()
	return s.policy
}

func (s *Server) invalidateSecurityPolicy() {
	s.policyMu.Lock()
	s.policyLoadedAt = time.Time{}
	s.policyMu.Unlock()
}

// rateLimiter is a per-instance token bucket keyed by caller identity.
//
// It is deliberately in memory: a database round trip per request would cost
// more than the burst it prevents. In a horizontally scaled deployment each
// instance therefore enforces the configured rate independently, so the
// effective ceiling is the rate multiplied by the replica count. The
// database-backed login lockout and AI daily cap are the limits that must hold
// globally, and those are counted in PostgreSQL.
type rateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*rateBucket
	lastSweep time.Time
}

type rateBucket struct {
	tokens  float64
	updated time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{buckets: map[string]*rateBucket{}, lastSweep: time.Now()}
}

// allow spends one token. It returns how long the caller should wait when the
// bucket is empty.
func (l *rateLimiter) allow(key string, perMinute int) (bool, time.Duration) {
	if perMinute <= 0 {
		return true, 0
	}
	capacity := math.Max(1, float64(perMinute))
	refillPerSecond := capacity / 60
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweepLocked(now)
	bucket := l.buckets[key]
	if bucket == nil {
		bucket = &rateBucket{tokens: capacity, updated: now}
		l.buckets[key] = bucket
	}
	bucket.tokens = math.Min(capacity, bucket.tokens+now.Sub(bucket.updated).Seconds()*refillPerSecond)
	bucket.updated = now
	if bucket.tokens < 1 {
		wait := time.Duration((1 - bucket.tokens) / refillPerSecond * float64(time.Second))
		return false, max(wait, time.Second)
	}
	bucket.tokens--
	return true, 0
}

// sweepLocked drops buckets that have refilled completely, which bounds memory
// without needing a separate goroutine.
func (l *rateLimiter) sweepLocked(now time.Time) {
	if now.Sub(l.lastSweep) < time.Minute {
		return
	}
	l.lastSweep = now
	for key, bucket := range l.buckets {
		if now.Sub(bucket.updated) > 5*time.Minute {
			delete(l.buckets, key)
		}
	}
}

// rateLimitKey prefers the authenticated principal so one shared office address
// cannot exhaust everyone's budget, and falls back to the peer address for
// unauthenticated traffic such as the login endpoint.
func rateLimitKey(r *http.Request) string {
	if p, ok := auth.PrincipalFrom(r.Context()); ok {
		return "user:" + p.User.ID.String() + ":" + p.AuthType
	}
	return "addr:" + clientIP(r)
}

func clientIP(r *http.Request) string {
	if address, ok := remoteAddressIP(r.RemoteAddr); ok {
		return address.String()
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// rateLimit protects the API from a single caller monopolising the service.
func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		policy := s.securityPolicy(r.Context())
		allowed, wait := s.limiter.allow(rateLimitKey(r), policy.APIRatePerMinute)
		if !allowed {
			writeRetryAfter(w, wait)
			writeProblem(w, r, http.StatusTooManyRequests, "rate-limited", "요청이 너무 많습니다",
				"짧은 시간에 너무 많은 요청이 들어왔습니다. 잠시 후 다시 시도해 주세요.",
				map[string]any{"retryAfterSeconds": int(wait.Seconds())})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// aiQuota guards the interactive endpoints from short bursts. The Dream
// service enforces the durable daily quota immediately before every gateway
// call, including calls made by the scheduled worker.
func (s *Server) aiQuota(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		policy := s.securityPolicy(r.Context())
		p := principal(r)
		allowed, wait := s.limiter.allow("ai:"+p.User.ID.String(), policy.AIRatePerMinute)
		if !allowed {
			writeRetryAfter(w, wait)
			writeProblem(w, r, http.StatusTooManyRequests, "ai-rate-limited", "AI 요청이 너무 잦습니다",
				"AI 생성 요청은 분당 "+strconv.Itoa(policy.AIRatePerMinute)+"회로 제한됩니다. 잠시 후 다시 시도해 주세요.",
				map[string]any{"retryAfterSeconds": int(wait.Seconds()), "limitPerMinute": policy.AIRatePerMinute})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeAIQuotaProblem(w http.ResponseWriter, r *http.Request, err error) bool {
	var quota *dream.AIQuotaError
	_ = errors.As(err, &quota)
	if errors.Is(err, dream.ErrAIDailyLimit) {
		limit, used := 0, 0
		if quota != nil {
			limit, used = quota.Limit, quota.Used
		}
		writeRetryAfter(w, time.Minute)
		writeProblem(w, r, http.StatusTooManyRequests, "ai-daily-limit", "오늘의 AI 사용량을 모두 썼습니다",
			"하루 AI 생성 한도 "+strconv.Itoa(limit)+"회를 모두 사용했습니다. 내일 다시 시도하거나 관리자에게 한도 조정을 요청해 주세요.",
			map[string]any{"dailyLimit": limit, "used": used})
		return true
	}
	if errors.Is(err, dream.ErrAIQuotaUnavailable) {
		slog.Error("AI quota persistence failed", "error", err, "user_id", principal(r).User.ID)
		writeRetryAfter(w, 5*time.Second)
		writeProblem(w, r, http.StatusServiceUnavailable, "ai-quota-unavailable", "AI 사용 한도를 확인하지 못했습니다",
			"사용량 보호를 위해 요청을 시작하지 않았습니다. 잠시 후 다시 시도해 주세요.",
			map[string]any{"retryAfterSeconds": 5})
		return true
	}
	return false
}

func writeRetryAfter(w http.ResponseWriter, wait time.Duration) {
	seconds := int(wait.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
}
