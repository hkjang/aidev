package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/store"
	"github.com/jackc/pgx/v5"
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)
var sensitiveCredentialPathPattern = regexp.MustCompile(`^/api/v1/(?:api-keys(?:/[^/]+/rotate)?|webhooks(?:/[^/]+/rotate-secret)?|handoff/claims)$`)

const idempotencyPendingLease = 2 * time.Minute
const idempotencyLeaseRefresh = 30 * time.Second

func idempotencySupported(method, path string) bool {
	if !strings.HasPrefix(path, "/api/v1/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	switch method {
	case http.MethodPost:
		// Capture is a single segment and belongs here for the same reason the
		// canvas mutations do: a thought written on a train has to survive the
		// tunnel, and a retry must not leave two copies of it.
		if len(parts) == 1 && parts[0] == "capture" {
			return true
		}
		return len(parts) == 3 && parts[1] != "" && ((parts[0] == "spaces" && (parts[2] == "notes" || parts[2] == "edges")) || (parts[0] == "notes" && parts[2] == "comments"))
	case http.MethodPut:
		return (len(parts) == 2 && parts[0] == "notes" && parts[1] != "") || (len(parts) == 3 && parts[0] == "comments" && parts[1] != "" && parts[2] == "resolve")
	case http.MethodDelete:
		return len(parts) == 2 && parts[1] != "" && (parts[0] == "notes" || parts[0] == "comments")
	default:
		return false
	}
}

func idempotencySuccessStatus(method string) int {
	switch method {
	case http.MethodPost:
		return http.StatusCreated
	case http.MethodDelete:
		return http.StatusNoContent
	default:
		return http.StatusOK
	}
}

func idempotencyRequestIdentity(r *http.Request, body []byte) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(r.URL.RequestURI()))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write(body)
	return r.URL.EscapedPath() + "#sha256:" + hex.EncodeToString(digest.Sum(nil))
}

// compressResponses gzips responses that are large enough to be worth it.
//
// A canvas of two thousand thoughts is about a megabyte of JSON, and the
// documented install exposes umm directly rather than behind a proxy that would
// compress for it — so without this, that megabyte crosses the wire every time
// someone opens the space.
//
// Responses that carry a freshly minted secret are left alone. Compressing a
// response whose length an attacker can watch while varying what it echoes is
// the shape of BREACH, and umm's CSRF defence is origin-based rather than a
// token in the body, so these endpoints are the only ones where a secret appears
// at all. Excluding them costs nothing: they are single small objects.
func (s *Server) compressResponses(next http.Handler) http.Handler {
	// Named by what Go's mime table actually returns, not by what the type is
	// "supposed" to be: a .js file is served as text/javascript, and listing only
	// application/javascript left the 435 kB bundle uncompressed while the
	// stylesheet beside it shrank. The list is checked against a real response
	// rather than assumed.
	compressor := middleware.Compress(5,
		"application/json", "application/javascript", "application/manifest+json",
		"text/javascript", "text/html", "text/css", "text/markdown", "text/plain",
		"image/svg+xml",
	)
	compressed := compressor(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sensitiveCredentialPathPattern.MatchString(r.URL.EscapedPath()) {
			next.ServeHTTP(w, r)
			return
		}
		compressed.ServeHTTP(w, r)
	})
}

func (s *Server) verifyWriteOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		p, ok := auth.PrincipalFrom(r.Context())
		if ok && p.AuthType == "api_key" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
			writeProblem(w, r, http.StatusForbidden, "cross-site-write", "교차 사이트 요청 차단", "다른 사이트에서 시작된 변경 요청은 허용되지 않습니다.", nil)
			return
		}
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		parsed, err := parseBrowserOrigin(origin)
		if err != nil {
			writeProblem(w, r, http.StatusForbidden, "invalid-origin", "Origin 확인 실패", "요청 출처를 확인할 수 없습니다.", nil)
			return
		}
		allowed := sameOrigin(parsed, effectiveRequestScheme(r), r.Host)
		if !allowed && s.Store != nil {
			var general struct {
				PublicURL string `json:"public_url"`
			}
			if s.Store.GetSetting(r.Context(), "general", &general) == nil {
				configured, parseErr := url.Parse(general.PublicURL)
				allowed = parseErr == nil && sameOrigin(parsed, configured.Scheme, configured.Host)
			}
		}
		if !allowed {
			writeProblem(w, r, http.StatusForbidden, "invalid-origin", "Origin 확인 실패", "허용된 서비스 주소에서 요청을 다시 시도해 주세요.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseBrowserOrigin(raw string) (*url.URL, error) {
	origin, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || origin.Host == "" || origin.User != nil ||
		(origin.Scheme != "http" && origin.Scheme != "https") ||
		(origin.Path != "" && origin.Path != "/") || origin.RawQuery != "" || origin.Fragment != "" {
		return nil, errors.New("invalid browser origin")
	}
	return origin, nil
}

func effectiveRequestScheme(r *http.Request) string {
	if forwarded := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))); forwarded == "http" || forwarded == "https" {
		return forwarded
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func sameOrigin(origin *url.URL, targetScheme, targetHost string) bool {
	target, err := url.Parse(strings.ToLower(strings.TrimSpace(targetScheme)) + "://" + targetHost)
	if err != nil || target.Host == "" || !strings.EqualFold(origin.Scheme, target.Scheme) ||
		!strings.EqualFold(origin.Hostname(), target.Hostname()) {
		return false
	}
	return originPort(origin) == originPort(target)
}

func originPort(origin *url.URL) string {
	if port := origin.Port(); port != "" {
		return port
	}
	if strings.EqualFold(origin.Scheme, "https") {
		return "443"
	}
	return "80"
}

func (s *Server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.PrincipalFrom(r.Context())
		if !ok || p.AuthType != "session" {
			writeProblem(w, r, http.StatusForbidden, "interactive-session-required", "브라우저 세션 필요", "이 작업은 로그인한 브라우저 세션에서만 수행할 수 있습니다.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// idempotency serializes supported Canvas mutations with a per-user advisory
// lock. The store completes each successful response in the domain transaction;
// a renewable pending lease blocks concurrent duplicates while allowing a retry
// to recover automatically when the process stops before the mutation begins.
func (s *Server) idempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" || r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodPost && sensitiveCredentialPathPattern.MatchString(r.URL.EscapedPath()) {
			writeProblem(w, r, http.StatusBadRequest, "idempotency-not-supported", "재시도 키 사용 불가", "한 번만 공개되는 자격 증명 응답에는 Idempotency-Key를 사용할 수 없습니다.", nil)
			return
		}
		if !idempotencySupported(r.Method, r.URL.EscapedPath()) {
			writeProblem(w, r, http.StatusBadRequest, "idempotency-not-supported", "재시도 키 사용 불가", "Idempotency-Key는 오프라인 Canvas 변경 endpoint에서만 사용할 수 있습니다.", nil)
			return
		}
		if !idempotencyKeyPattern.MatchString(key) {
			writeProblem(w, r, http.StatusBadRequest, "invalid-idempotency-key", "재시도 키 오류", "Idempotency-Key는 8~128자의 영문, 숫자, 점, 밑줄, 콜론 또는 하이픈이어야 합니다.", nil)
			return
		}
		requestBody, err := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
		if err != nil {
			writeProblem(w, r, http.StatusBadRequest, "invalid-request-body", "요청 본문 오류", "재시도 요청 본문을 읽을 수 없습니다.", nil)
			return
		}
		if len(requestBody) > 1<<20 {
			writeProblem(w, r, http.StatusRequestEntityTooLarge, "request-too-large", "요청 본문 초과", "요청 본문은 1 MiB 이하여야 합니다.", nil)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(requestBody))
		requestIdentity := idempotencyRequestIdentity(r, requestBody)
		p, ok := auth.PrincipalFrom(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		tx, err := s.Store.Pool.Begin(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "안전한 재시도 상태를 준비하지 못했습니다.")
			return
		}
		defer tx.Rollback(r.Context())
		lockKey := p.User.ID.String() + ":" + key
		if _, err = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, lockKey); err != nil {
			writeError(w, http.StatusServiceUnavailable, "안전한 재시도 잠금을 얻지 못했습니다.")
			return
		}
		if _, err = tx.Exec(r.Context(), `DELETE FROM idempotency_records WHERE user_id=$1 AND idempotency_key=$2 AND expires_at<=now()`, p.User.ID, key); err != nil {
			writeError(w, http.StatusServiceUnavailable, "만료된 재시도 상태를 정리하지 못했습니다.")
			return
		}
		var method, path, state string
		var status *int
		var stored []byte
		var expiresAt time.Time
		err = tx.QueryRow(r.Context(), `SELECT method,path,state,response_status,response_body,expires_at FROM idempotency_records WHERE user_id=$1 AND idempotency_key=$2 AND expires_at>now()`, p.User.ID, key).Scan(&method, &path, &state, &status, &stored, &expiresAt)
		if err == nil {
			if err = tx.Commit(r.Context()); err != nil {
				writeError(w, http.StatusServiceUnavailable, "재시도 상태를 확정하지 못했습니다.")
				return
			}
			if method != r.Method || path != requestIdentity {
				writeProblem(w, r, http.StatusConflict, "idempotency-key-reused", "재시도 키 충돌", "같은 Idempotency-Key를 다른 요청에 사용할 수 없습니다.", nil)
				return
			}
			if state == "pending" {
				retryAfter := int((time.Until(expiresAt) + time.Second - 1) / time.Second)
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				writeProblem(w, r, http.StatusTooEarly, "idempotency-in-progress", "재시도 처리 중", "같은 요청이 아직 처리 중입니다. Retry-After 이후 자동으로 다시 시도합니다.", nil)
				return
			}
			if state != "completed" || status == nil {
				writeError(w, http.StatusServiceUnavailable, "저장된 재시도 결과가 올바르지 않습니다.")
				return
			}
			w.Header().Set("Idempotency-Replayed", "true")
			if len(stored) > 0 {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
			}
			w.WriteHeader(*status)
			_, _ = w.Write(stored)
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusServiceUnavailable, "재시도 상태를 확인하지 못했습니다.")
			return
		}
		var reservationCreatedAt time.Time
		if err = tx.QueryRow(r.Context(), `INSERT INTO idempotency_records(user_id,idempotency_key,method,path,state,response_status,expires_at) VALUES($1,$2,$3,$4,'pending',NULL,now()+$5*interval '1 second') RETURNING created_at`, p.User.ID, key, r.Method, requestIdentity, idempotencyPendingLease.Seconds()).Scan(&reservationCreatedAt); err != nil {
			writeError(w, http.StatusServiceUnavailable, "재시도 요청을 예약하지 못했습니다.")
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "재시도 요청을 확정하지 못했습니다.")
			return
		}
		reservation := store.IdempotencyReservation{
			UserID: p.User.ID, Key: key, Method: r.Method, Path: requestIdentity,
			CreatedAt: reservationCreatedAt, Status: idempotencySuccessStatus(r.Method),
		}
		r = r.WithContext(store.WithIdempotencyReservation(r.Context(), reservation))
		stopLease := s.maintainIdempotencyLease(reservation)
		defer stopLease()

		recorder := httptest.NewRecorder()
		next.ServeHTTP(recorder, r)
		stopLease()
		result := recorder.Result()
		responseBody := bytes.Clone(recorder.Body.Bytes())
		recordable := result.StatusCode >= 200 && result.StatusCode < 300 && len(responseBody) <= 1<<20
		if recordable && len(responseBody) > 0 {
			var body any
			recordable = json.Unmarshal(responseBody, &body) == nil
		}
		if recordable {
			verifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			var persistedStatus int
			var persistedBody []byte
			verifyErr := s.Store.Pool.QueryRow(verifyCtx, `SELECT response_status,response_body FROM idempotency_records WHERE user_id=$1 AND idempotency_key=$2 AND method=$3 AND path=$4 AND state='completed' AND created_at=$5 AND expires_at>now()`, p.User.ID, key, r.Method, requestIdentity, reservationCreatedAt).Scan(&persistedStatus, &persistedBody)
			cancel()
			if verifyErr != nil || persistedStatus != result.StatusCode {
				slog.Error("atomic idempotency completion unavailable", "user_id", p.User.ID, "path", r.URL.EscapedPath(), "error", verifyErr)
				writeError(w, http.StatusServiceUnavailable, "변경은 처리되었지만 안전한 재시도 결과를 확인하지 못했습니다. 같은 키로 다시 시도해 주세요.")
				return
			}
			responseBody = persistedBody
		} else {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, cleanupErr := s.Store.Pool.Exec(cleanupCtx, `DELETE FROM idempotency_records WHERE user_id=$1 AND idempotency_key=$2 AND method=$3 AND path=$4 AND state='pending' AND created_at=$5`, p.User.ID, key, r.Method, requestIdentity, reservationCreatedAt)
			cancel()
			if cleanupErr != nil {
				slog.Warn("idempotency pending reservation cleanup failed", "user_id", p.User.ID, "path", r.URL.EscapedPath(), "error", cleanupErr)
			}
		}
		for header, values := range result.Header {
			for _, value := range values {
				w.Header().Add(header, value)
			}
		}
		w.WriteHeader(result.StatusCode)
		_, _ = w.Write(responseBody)
	})
}

func (s *Server) maintainIdempotencyLease(reservation store.IdempotencyReservation) func() {
	return s.maintainIdempotencyLeaseEvery(reservation, idempotencyLeaseRefresh)
}

func (s *Server) maintainIdempotencyLeaseEvery(reservation store.IdempotencyReservation, refreshInterval time.Duration) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				command, err := s.Store.Pool.Exec(ctx, `UPDATE idempotency_records SET expires_at=now()+$6*interval '1 second' WHERE user_id=$1 AND idempotency_key=$2 AND method=$3 AND path=$4 AND state='pending' AND created_at=$5`, reservation.UserID, reservation.Key, reservation.Method, reservation.Path, reservation.CreatedAt, idempotencyPendingLease.Seconds())
				cancel()
				if err != nil {
					slog.Warn("idempotency lease refresh failed", "user_id", reservation.UserID, "path", reservation.Path, "error", err)
					continue
				}
				if command.RowsAffected() == 0 {
					return
				}
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(stop)
			<-done
		})
	}
}
