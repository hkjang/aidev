package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/hkjang/seaton/internal/tracking"
	"github.com/jackc/pgx/v5"
)

// 방문 추적 스니펫과 콘텐츠 보안 정책(CSP)의 접점.
//
// 정책은 문서(index.html) 응답에 실리는 것만 브라우저가 따르므로, 스니펫과
// nonce 와 정책 조립은 모두 SPA 문서를 내보내는 자리에서 한다. 정적 자산과
// API 응답은 원래의 잠긴 정책을 그대로 받는다.

// cspReportPath 는 브라우저가 정책 위반을 신고하는 곳이다. 같은 오리진이라
// 정책에 더할 출처가 없고, 추적이 켜져 있을 때만 정책에 들어간다.
const cspReportPath = "/api/v1/tracking/csp-report"

// basePolicy 는 추적이 꺼져 있을 때의 정책이다. 추적을 끄면 이 문자열 그대로
// 돌아가야 한다 — 한 번 느슨해진 채 남는 정책이 이 기능의 가장 큰 위험이다.
const basePolicy = "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

// pagePolicy 는 이 경로의 문서에 실을 정책이다. 추적이 이 문서에 붙을 때만
// script-src 에 그 요청의 nonce 와 스니펫이 필요로 하는 출처를 더하고,
// 브라우저가 거부한 것을 신고할 report-uri 를 넣는다. 'unsafe-inline' 은
// 어떤 경우에도 넣지 않는다.
func pagePolicy(config tracking.Config, path, nonce string) string {
	if !config.Active(path) {
		return basePolicy
	}
	extraScripts, extraConnects, extraImages := config.PolicySources()
	scripts := append([]string{"'self'", "'nonce-" + nonce + "'"}, extraScripts...)
	connects := append([]string{"'self'"}, extraConnects...)
	images := append([]string{"'self'", "data:", "blob:"}, extraImages...)
	return "default-src 'self'; img-src " + strings.Join(images, " ") +
		"; style-src 'self' 'unsafe-inline'; script-src " + strings.Join(scripts, " ") +
		"; connect-src " + strings.Join(connects, " ") +
		"; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; report-uri " + cspReportPath
}

// newNonce 는 요청 하나에 쓰는 nonce 다. 정책의 base64-value 문법에 맞춘다.
func newNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// loadTracking 은 settings 의 tracking.* 를 읽는다. 읽지 못하면 꺼진 설정으로
// 다룬다 — 데이터베이스가 잠시 안 보인다고 정책이 느슨해져서는 안 된다.
func (s *Server) loadTracking(ctx context.Context) tracking.Config {
	rows, err := s.db.Query(ctx, `SELECT key,value FROM settings WHERE key LIKE $1`, tracking.KeyPrefix+"%")
	if err != nil {
		s.logger.Warn("추적 설정을 읽지 못했습니다", "error", err)
		return tracking.Config{}
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err == nil {
			values[key] = value
		}
	}
	return tracking.ReadConfig(values)
}

// servePage 는 SPA 문서를 내보낸다. 추적이 이 경로에 붙으면 nonce 를 만들어
// 스니펫의 모든 script 태그와 정책의 script-src 에 같은 값을 넣는다.
func (s *Server) servePage(w http.ResponseWriter, r *http.Request, document []byte) {
	config := s.trackingConfig(r.Context())
	if config.Active(r.URL.Path) {
		nonce := newNonce()
		w.Header().Set("Content-Security-Policy", pagePolicy(config, r.URL.Path, nonce))
		document = tracking.Inject(document, config.Snippet(nonce), config.Placement)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(document)
}

// validateTrackingSettings 는 저장하려는 트랜잭션 안의 tracking.* 값을 통째로
// 읽어 검증한다. 요청이 일부 키만 보내도 나머지와 합친 결과가 말이 되어야 한다.
func (s *Server) validateTrackingSettings(ctx context.Context, tx pgx.Tx) error {
	rows, err := tx.Query(ctx, `SELECT key,value FROM settings WHERE key LIKE $1`, tracking.KeyPrefix+"%")
	if err != nil {
		return err
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		values[key] = value
	}
	return tracking.ReadConfig(values).Validate()
}

// cspReport 는 브라우저의 정책 위반 신고를 받는다. 자격 증명 없이 오는 요청이라
// 인증도 CSRF 도 요구하지 않고, 유계 메모리 목록만 바꾼다. 추적이 꺼져 있으면
// 정책에 report-uri 가 없으므로 온 것은 남의 신고다 — 버린다.
func (s *Server) cspReport(w http.ResponseWriter, r *http.Request) {
	if !s.trackingConfig(r.Context()).Enabled {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var body struct {
		Report struct {
			BlockedURI         string `json:"blocked-uri"`
			EffectiveDirective string `json:"effective-directive"`
			ViolatedDirective  string `json:"violated-directive"`
			DocumentURI        string `json:"document-uri"`
		} `json:"csp-report"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_report", "신고 형식을 읽지 못했습니다")
		return
	}
	directive := body.Report.EffectiveDirective
	if directive == "" {
		directive = body.Report.ViolatedDirective
	}
	page := body.Report.DocumentURI
	if parsed, err := url.Parse(page); err == nil && parsed.Path != "" {
		page = parsed.Path
	}
	s.violations.Record(body.Report.BlockedURI, directive, page)
	w.WriteHeader(http.StatusNoContent)
}

// listTrackingViolations 는 차단된 출처를 관리 화면에 보여 준다. 지금 설정이
// 이미 허용하는 것은 allowed 로 표시된다.
func (s *Server) listTrackingViolations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.violations.List(s.trackingConfig(r.Context()))})
}

func (s *Server) forgetTrackingViolations(w http.ResponseWriter, r *http.Request) {
	s.violations.Forget()
	w.WriteHeader(http.StatusNoContent)
}

// momentoProxy 는 /momento/* 를 Momento 수집기로 넘긴다. 로더(tracker.js)와
// 수집 요청(collect/v1/events)만 지나가고, 이 앱의 세션 쿠키·인증 헤더는 벗겨
// 수집기에 닿지 않는다. 프록시 구성이 아니면 이 경로는 없다.
func (s *Server) momentoProxy(w http.ResponseWriter, r *http.Request) {
	target := s.trackingConfig(r.Context()).ProxyTarget()
	if target == nil {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, tracking.ProxyPrefix)
	loader := r.Method == http.MethodGet && rest == "/tracker.js"
	collect := (r.Method == http.MethodPost || r.Method == http.MethodOptions) && rest == "/collect/v1/events"
	if !loader && !collect {
		http.NotFound(w, r)
		return
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.Out.URL.Path, request.Out.URL.RawPath = rest, ""
			request.SetURL(target)
			request.SetXForwarded()
			request.Out.Header.Del("Cookie")
			request.Out.Header.Del("Authorization")
		},
		ModifyResponse: func(response *http.Response) error {
			// 수집기가 세운 쿠키나 정책이 이 앱의 것과 섞이지 않게 한다.
			response.Header.Del("Set-Cookie")
			response.Header.Del("Content-Security-Policy")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			s.logger.Warn("Momento 수집기에 연결하지 못했습니다", "error", err, "target", target.String())
			writeError(w, http.StatusBadGateway, "tracking_upstream", "추적 수집기에 연결하지 못했습니다")
		},
	}
	proxy.ServeHTTP(w, r)
}
