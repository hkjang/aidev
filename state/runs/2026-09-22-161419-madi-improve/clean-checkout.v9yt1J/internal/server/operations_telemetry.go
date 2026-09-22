package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	collector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

// No global provider, environment detector, OTEL_* variables or request-context
// propagator is installed. Only these explicitly constructed spans leave madi.
type operationsRuntime struct {
	ctx                context.Context
	started            time.Time
	mu                 sync.RWMutex
	generation         *telemetryGeneration
	fingerprint        string
	errorsEnabled      bool
	retention          int
	configurationError string
	lastResult         string
	lastSent           time.Time
	traces             chan operationTrace
	errors             chan operationError
	sent               atomic.Uint64
	dropped            atomic.Uint64
	failed             atomic.Uint64
	errorDropped       atomic.Uint64
}
type telemetryGeneration struct {
	ctx      context.Context
	cancel   context.CancelFunc
	provider *sdktrace.TracerProvider
	client   *operationCollector
}
type operationTrace struct {
	generation *telemetryGeneration
	span       *tracepb.Span
}
type operationError struct {
	RequestID, Route, Method, Kind string
	Status                         int
	Duration                       int64
	Created                        time.Time
}
type operationCollector struct {
	endpoint, token string
	client          *http.Client
}
type operationExporter struct {
	runtime    *operationsRuntime
	generation *telemetryGeneration
}

func (e *operationExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		if e.generation.ctx.Err() != nil {
			e.runtime.dropped.Add(1)
			continue
		}
		sc := span.SpanContext()
		tid, sid := sc.TraceID(), sc.SpanID()
		out := &tracepb.Span{TraceId: append([]byte(nil), tid[:]...), SpanId: append([]byte(nil), sid[:]...), Flags: uint32(sc.TraceFlags()), Name: span.Name(), Kind: tracepb.Span_SPAN_KIND_SERVER, StartTimeUnixNano: uint64(span.StartTime().UnixNano()), EndTimeUnixNano: uint64(span.EndTime().UnixNano())}
		for _, attr := range span.Attributes() {
			key := string(attr.Key)
			switch key {
			case "http.request.method", "http.route", "madi.request_id":
				out.Attributes = append(out.Attributes, stringAttribute(key, attr.Value.AsString()))
			case "http.response.status_code":
				out.Attributes = append(out.Attributes, &commonpb.KeyValue{Key: key, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: attr.Value.AsInt64()}}})
			}
		}
		select {
		case e.runtime.traces <- operationTrace{e.generation, out}:
		default:
			e.runtime.dropped.Add(1)
		}
	}
	return nil
}
func (*operationExporter) Shutdown(context.Context) error { return nil }
func stringAttribute(key, value string) *commonpb.KeyValue {
	return &commonpb.KeyValue{Key: key, Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: value}}}
}

func newOperationCollector(cfg map[string]any) (*operationCollector, error) {
	if err := validateOperationsSettings(cfg); err != nil {
		return nil, err
	}
	if str(cfg, "otel_endpoint") == "" {
		return nil, errors.New("저장된 OTLP 수신 주소가 없습니다")
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if ca := str(cfg, "otel_ca_pem"); ca != "" && !pool.AppendCertsFromPEM([]byte(ca)) {
		return nil, errors.New("CA 인증서를 읽지 못했습니다")
	}
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: time.Duration(number(cfg, "otel_timeout_seconds", 5)) * time.Second, MaxIdleConns: 2, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second}
	return &operationCollector{str(cfg, "otel_endpoint"), str(cfg, "otel_auth_token"), &http.Client{Transport: transport, Timeout: time.Duration(number(cfg, "otel_timeout_seconds", 5)) * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("OTLP redirect denied") }}}, nil
}
func (c *operationCollector) close() {
	if t, ok := c.client.Transport.(*http.Transport); ok {
		t.CloseIdleConnections()
	}
}
func (c *operationCollector) send(ctx context.Context, version string, spans []*tracepb.Span) error {
	payload := &collector.ExportTraceServiceRequest{ResourceSpans: []*tracepb.ResourceSpans{{Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{stringAttribute("service.name", "madi"), stringAttribute("service.version", version)}}, ScopeSpans: []*tracepb.ScopeSpans{{Scope: &commonpb.InstrumentationScope{Name: "madi.http", Version: "1"}, Spans: spans}}}}}
	body, err := proto.Marshal(payload)
	if err != nil {
		return errors.New("Trace 직렬화 실패")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return errors.New("OTLP 요청 생성 실패")
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return errors.New("설정 변경 또는 종료로 전송 취소")
		}
		return errors.New("OTLP 연결 실패: 주소·TLS 인증서·수신 서버 상태를 확인하세요")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New("OTLP 수신 서버가 성공하지 않은 HTTP 상태를 반환했습니다")
	}
	if err != nil || len(data) > 64<<10 {
		return errors.New("OTLP 응답 크기 또는 읽기 오류")
	}
	var result collector.ExportTraceServiceResponse
	if len(data) > 0 && proto.Unmarshal(data, &result) != nil {
		return errors.New("OTLP protobuf 응답 형식이 올바르지 않습니다")
	}
	if result.PartialSuccess != nil && result.PartialSuccess.RejectedSpans > 0 {
		return errors.New("수신 서버가 일부 Trace를 거부했습니다")
	}
	return nil
}

func (s *Server) StartOperations(ctx context.Context) {
	op := &operationsRuntime{ctx: ctx, started: time.Now(), traces: make(chan operationTrace, 2048), errors: make(chan operationError, 2048), retention: 7}
	if !s.operations.CompareAndSwap(nil, op) {
		return
	}
	s.reloadOperations(op)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		defer func() {
			op.mu.Lock()
			generation := op.generation
			op.generation = nil
			op.mu.Unlock()
			stopTelemetryGeneration(generation)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.reloadOperations(op)
			}
		}
	}()
	go s.exportOperations(op)
	go s.storeOperationErrors(op)
}
func stopTelemetryGeneration(g *telemetryGeneration) {
	if g != nil {
		g.cancel()
		g.client.close()
		_ = g.provider.Shutdown(context.Background())
	}
}
func (s *Server) reloadOperations(op *operationsRuntime) {
	ctx, cancel := context.WithTimeout(op.ctx, 2*time.Second)
	defer cancel()
	cfg, err := s.settings(ctx)
	if err == nil {
		err = validateOperationsSettings(cfg)
	}
	fingerprint := "invalid"
	if err == nil {
		snapshot := map[string]any{}
		for key := range defaultOperationsSettings() {
			snapshot[key] = cfg[key]
		}
		fingerprint = digest(string(jsonValue(snapshot)))
	}
	op.mu.Lock()
	if fingerprint == op.fingerprint {
		op.mu.Unlock()
		return
	}
	old := op.generation
	op.generation = nil
	op.fingerprint = fingerprint
	op.configurationError = ""
	op.errorsEnabled = err == nil && boolean(cfg, "operations_errors_enabled")
	if err != nil {
		op.configurationError = "운영 설정을 안전하게 읽지 못해 전송과 오류 수집을 중지했습니다"
	}
	if err == nil {
		op.retention = number(cfg, "operations_retention_days", 7)
	}
	op.mu.Unlock()
	stopTelemetryGeneration(old)
	if err != nil || !boolean(cfg, "otel_enabled") || op.ctx.Err() != nil {
		return
	}
	client, err := newOperationCollector(cfg)
	if err != nil {
		return
	}
	genCtx, genCancel := context.WithCancel(op.ctx)
	gen := &telemetryGeneration{ctx: genCtx, cancel: genCancel, client: client}
	rate, _ := operationNumber(cfg["otel_sample_rate"])
	gen.provider = sdktrace.NewTracerProvider(sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", "madi"), attribute.String("service.version", s.Version))), sdktrace.WithSampler(sdktrace.TraceIDRatioBased(rate)), sdktrace.WithRawSpanLimits(sdktrace.SpanLimits{AttributeValueLengthLimit: 360, AttributeCountLimit: 4, EventCountLimit: 0, LinkCountLimit: 0, AttributePerEventCountLimit: 0, AttributePerLinkCountLimit: 0}), sdktrace.WithSyncer(&operationExporter{op, gen}))
	op.mu.Lock()
	op.generation = gen
	op.mu.Unlock()
}
func (s *Server) safeOperationRoute(pattern string) string {
	for _, registered := range s.apiRoutes {
		if pattern == registered {
			return pattern
		}
	}
	switch pattern {
	case "GET /", "GET /healthz", "GET /readyz", "GET /api/v1/public", "GET /api/v1/openapi.json", "POST /api/v1/auth/login", "POST /api/v1/auth/ldap/login", "GET /api/v1/auth/saml/start", "GET /api/v1/auth/saml/metadata", "POST /api/v1/auth/saml/acs", "GET /api/v1/auth/oidc/start", "GET /api/v1/auth/oidc/callback", "POST /api/v1/capture-hooks/{id}":
		return pattern
	}
	return "unmatched"
}
func (s *Server) recordOperation(id, pattern, method string, status int, start time.Time, panicked bool) {
	op := s.operations.Load()
	if op == nil || op.ctx.Err() != nil {
		return
	}
	if !validID(id) {
		return
	}
	if !oneOf(method, "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT", "TRACE") {
		method = "OTHER"
	}
	route := s.safeOperationRoute(pattern)
	if status < 100 || status > 599 {
		status = 500
	}
	if panicked {
		status = 500
	}
	end := time.Now()
	op.mu.RLock()
	generation, errorsEnabled := op.generation, op.errorsEnabled
	op.mu.RUnlock()
	if generation != nil && generation.ctx.Err() == nil {
		_, span := generation.provider.Tracer("madi.http").Start(context.Background(), route, trace.WithTimestamp(start), trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attribute.String("http.request.method", method), attribute.String("http.route", route), attribute.String("madi.request_id", id), attribute.Int("http.response.status_code", status)))
		span.End(trace.WithTimestamp(end))
	}
	if status >= 400 && errorsEnabled {
		kind := "http"
		if panicked {
			kind = "panic"
		}
		select {
		case op.errors <- operationError{id, route, method, kind, status, max(0, end.Sub(start).Milliseconds()), end}:
		default:
			op.errorDropped.Add(1)
		}
	}
}
func (s *Server) exportOperations(op *operationsRuntime) {
	for {
		select {
		case <-op.ctx.Done():
			return
		case first := <-op.traces:
			batch := []*tracepb.Span{first.span}
			for len(batch) < 128 {
				select {
				case next := <-op.traces:
					if next.generation == first.generation {
						batch = append(batch, next.span)
					} else {
						op.dropped.Add(1)
					}
				default:
					goto send
				}
			}
		send:
			if first.generation.ctx.Err() != nil {
				op.dropped.Add(uint64(len(batch)))
				continue
			}
			err := first.generation.client.send(first.generation.ctx, s.Version, batch)
			op.mu.Lock()
			if err == nil {
				op.lastSent = time.Now()
				op.lastResult = "OTLP 수신 서버 전송 성공"
				op.sent.Add(uint64(len(batch)))
			} else {
				op.lastResult = err.Error()
				op.failed.Add(uint64(len(batch)))
			}
			op.mu.Unlock()
		}
	}
}
func (s *Server) storeOperationErrors(op *operationsRuntime) {
	cleanup := time.NewTicker(time.Minute)
	defer cleanup.Stop()
	for {
		select {
		case <-op.ctx.Done():
			return
		case event := <-op.errors:
			op.mu.RLock()
			enabled := op.errorsEnabled
			op.mu.RUnlock()
			if !enabled {
				op.errorDropped.Add(1)
				continue
			}
			ctx, cancel := context.WithTimeout(op.ctx, 2*time.Second)
			_, err := s.DB.Exec(ctx, `INSERT INTO operations_http_errors(request_id,route,method,status,duration_ms,kind,created_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, event.RequestID, event.Route, event.Method, event.Status, event.Duration, event.Kind, event.Created)
			cancel()
			if err != nil {
				op.errorDropped.Add(1)
			}
		case <-cleanup.C:
			op.mu.RLock()
			days := op.retention
			op.mu.RUnlock()
			ctx, cancel := context.WithTimeout(op.ctx, 3*time.Second)
			_, _ = s.DB.Exec(ctx, `DELETE FROM operations_http_errors WHERE created_at<now()-($1::int*interval '1 day') OR request_id IN (SELECT request_id FROM operations_http_errors ORDER BY created_at DESC OFFSET 50000)`, days)
			cancel()
		}
	}
}
func (s *Server) registerTelemetryOperations() {
	s.admin("GET /api/v1/admin/operations/health", s.operationsHealth)
	s.admin("GET /api/v1/admin/operations/errors", func(w http.ResponseWriter, r *http.Request) {
		rows, err := s.rows(r.Context(), `SELECT to_jsonb(e) FROM operations_http_errors e ORDER BY created_at DESC LIMIT 200`)
		respond(w, rows, err)
	})
	s.admin("GET /api/v1/admin/operations/telemetry", s.operationsTelemetryStatus)
	s.admin("POST /api/v1/admin/operations/telemetry/test", s.testOperationsTelemetry)
}
func (s *Server) operationStatus() map[string]any {
	status := map[string]any{"running": false, "enabled": false, "sent_spans": 0, "failed_spans": 0, "dropped_spans": 0, "dropped_errors": 0, "queue_capacity": 2048, "notice": "Trace에는 HTTP 메서드·등록 경로 템플릿·요청 ID·상태·시간과 서비스 버전만 포함됩니다. 본문·질문·사용자·문서 ID·주소 쿼리·비밀값은 전송하지 않습니다. 설정 변경·종료 시 대기 Trace는 폐기하며 실패 자동 재전송은 하지 않습니다."}
	if op := s.operations.Load(); op != nil {
		op.mu.RLock()
		status["running"] = op.ctx.Err() == nil
		status["enabled"] = op.generation != nil && op.generation.ctx.Err() == nil
		status["configuration_error"] = op.configurationError
		status["last_result"] = op.lastResult
		if !op.lastSent.IsZero() {
			status["last_sent_at"] = op.lastSent
		}
		status["errors_enabled"] = op.errorsEnabled
		status["uptime_seconds"] = int64(time.Since(op.started).Seconds())
		op.mu.RUnlock()
		status["sent_spans"] = op.sent.Load()
		status["failed_spans"] = op.failed.Load()
		status["dropped_spans"] = op.dropped.Load()
		status["dropped_errors"] = op.errorDropped.Load()
		status["queued_spans"] = len(op.traces)
	}
	return status
}
func (s *Server) operationsTelemetryStatus(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, 200, s.operationStatus())
}
func (s *Server) operationsHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	started := time.Now()
	err := s.DB.Ping(ctx)
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	pool := s.DB.Stat()
	classes := map[string]any{}
	for i := 1; i <= 5; i++ {
		classes[strconv.Itoa(i)+"xx"] = s.responseClasses[i].Load()
	}
	jsonResponse(w, 200, map[string]any{"version": s.Version, "checked_at": time.Now(), "database": map[string]any{"ready": err == nil, "latency_ms": time.Since(started).Milliseconds(), "connections": pool.TotalConns(), "idle_connections": pool.IdleConns(), "max_connections": pool.MaxConns()}, "process": map[string]any{"heap_bytes": mem.HeapAlloc, "goroutines": runtime.NumGoroutine()}, "requests": s.requests.Load(), "errors": s.errors.Load(), "responses": classes, "telemetry": s.operationStatus()})
}
func (s *Server) testOperationsTelemetry(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.settings(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	client, err := newOperationCollector(cfg)
	if err != nil {
		apiError(w, 400, err.Error())
		return
	}
	defer client.close()
	tid := newID()
	raw, _ := hex.DecodeString(digest(tid))
	traceID := raw[:16]
	spanID := raw[16:24]
	now := time.Now()
	span := &tracepb.Span{TraceId: traceID, SpanId: spanID, Name: "madi.telemetry.diagnostic", Kind: tracepb.Span_SPAN_KIND_INTERNAL, StartTimeUnixNano: uint64(now.UnixNano()), EndTimeUnixNano: uint64(now.Add(time.Microsecond).UnixNano()), Attributes: []*commonpb.KeyValue{stringAttribute("madi.request_id", tid)}}
	if err = client.send(r.Context(), s.Version, []*tracepb.Span{span}); err != nil {
		apiError(w, 502, err.Error())
		return
	}
	s.audit(r, "TELEMETRY_TEST", "operations", map[string]any{"result": "success"})
	jsonResponse(w, 200, map[string]any{"ok": true, "content_sent": false, "message": "저장된 수신 주소로 고정 진단 Trace 1개를 실제 전송했습니다. 문서·질문·개인정보는 보내지 않았습니다."})
}
