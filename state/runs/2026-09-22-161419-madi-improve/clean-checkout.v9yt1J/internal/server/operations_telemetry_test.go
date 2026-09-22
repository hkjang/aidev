package server

import (
	"context"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	collector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestOperationsOTLPProtocolTLSRedirectAndPrivacy(t *testing.T) {
	var received atomic.Int32
	var payload atomic.Value
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		payload.Store(body)
		if r.URL.Path != "/v1/traces" || r.Header.Get("Content-Type") != "application/x-protobuf" || r.Header.Get("Authorization") != "Bearer test-private-secret" {
			t.Error("invalid wire headers")
		}
		var request collector.ExportTraceServiceRequest
		if proto.Unmarshal(body, &request) != nil || len(request.ResourceSpans) != 1 || request.ResourceSpans[0].Resource.Attributes[0].GetValue().GetStringValue() != "madi" {
			t.Error("invalid protobuf OTLP request")
		}
		received.Add(1)
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Write([]byte{})
	}))
	defer server.Close()
	cfg := defaultSettings()
	cfg["otel_endpoint"] = server.URL + "/v1/traces"
	cfg["otel_auth_token"] = "test-private-secret"
	client, err := newOperationCollector(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if client.send(context.Background(), "test", []*tracepb.Span{}) == nil {
		t.Fatal("untrusted TLS accepted")
	}
	client.close()
	cfg["otel_ca_pem"] = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}))
	client, err = newOperationCollector(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client.close()
	if err = client.send(context.Background(), "test", []*tracepb.Span{{TraceId: make([]byte, 16), SpanId: make([]byte, 8), Name: "fixed"}}); err != nil {
		t.Fatal(err)
	}
	if received.Load() != 1 || strings.Contains(string(payload.Load().([]byte)), "test-private-secret") {
		t.Fatal("wire secret exposed in payload")
	}
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, server.URL+"/v1/traces", 302) }))
	defer redirect.Close()
	cfg["otel_endpoint"] = redirect.URL
	cfg["otel_allow_http"] = true
	client2, err := newOperationCollector(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer client2.close()
	if client2.send(context.Background(), "test", []*tracepb.Span{}) == nil || received.Load() != 1 {
		t.Fatal("redirect followed")
	}
}

func TestOperationsRealSDKRuntimeDisableAndErrors(t *testing.T) {
	t.Setenv("OTEL_TRACES_SAMPLER", "always_off")
	t.Setenv("OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT", "0")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "PRIVATE_ENV_ATTRIBUTE=PRIVATE_ENV_PAYLOAD")
	s, admin, _, _, wid := jobTestFixture(t)
	requests := make(chan *collector.ExportTraceServiceRequest, 32)
	started, cancelled := make(chan struct{}, 1), make(chan struct{}, 1)
	var slow atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var request collector.ExportTraceServiceRequest
		if proto.Unmarshal(body, &request) != nil {
			t.Error("invalid real SDK wire")
		}
		if slow.Load() {
			select {
			case started <- struct{}{}:
			default:
			}
			<-r.Context().Done()
			select {
			case cancelled <- struct{}{}:
			default:
			}
			return
		}
		select {
		case requests <- &request:
		default:
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.StartOperations(ctx)
	op := s.operations.Load()
	if op == nil || op.generation != nil {
		t.Fatal("telemetry must default off")
	}
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"otel_endpoint": server.URL + "/v1/traces", "otel_allow_http": true, "otel_enabled": true, "otel_sample_rate": 1, "otel_timeout_seconds": 10, "otel_auth_token": "TRACE_SECRET_937"}, 200)
	operationWait(t, func() bool { op.mu.RLock(); defer op.mu.RUnlock(); return op.generation != nil })
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "TRACE_PRIVATE_TITLE_938", "markdown": "TRACE_PRIVATE_BODY_938", "visibility": "private"}, 200))
	admin.request("GET", "/api/v1/documents/"+str(doc, "id")+"?secret=TRACE_QUERY_938", nil, 200)
	admin.request("GET", "/api/v1/documents/"+newID(), nil, 404)
	operationWait(t, func() bool { return len(requests) >= 3 })
	foundRoute := false
	for len(requests) > 0 {
		request := <-requests
		raw := string(jsonValue(request))
		for _, secret := range []string{"TRACE_SECRET_937", "TRACE_PRIVATE_TITLE_938", "TRACE_PRIVATE_BODY_938", "TRACE_QUERY_938", "PRIVATE_ENV_PAYLOAD", str(doc, "id"), wid} {
			if strings.Contains(raw, secret) {
				t.Fatalf("trace leaked %s", secret)
			}
		}
		for _, rs := range request.ResourceSpans {
			for _, ss := range rs.ScopeSpans {
				for _, span := range ss.Spans {
					if span.Name == "GET /api/v1/documents/{id}" {
						foundRoute = true
						if len(span.Attributes) != 4 {
							t.Fatal("environment variables overrode explicit span limits")
						}
					}
					if len(span.TraceId) != 16 || len(span.SpanId) != 8 {
						t.Fatal("SDK trace ID invalid")
					}
					for _, a := range span.Attributes {
						if !oneOf(a.Key, "http.request.method", "http.route", "madi.request_id", "http.response.status_code") {
							t.Fatal("unexpected attribute")
						}
					}
				}
			}
		}
	}
	if !foundRoute {
		t.Fatal("registered route template absent")
	}
	operationWait(t, func() bool {
		var n int
		s.DB.QueryRow(ctx, "SELECT count(*) FROM operations_http_errors WHERE route='GET /api/v1/documents/{id}' AND status=404").Scan(&n)
		return n > 0
	})
	var errorsBody string
	errorsBody = string(admin.request("GET", "/api/v1/admin/operations/errors", nil, 200))
	if strings.Contains(errorsBody, "TRACE_") {
		t.Fatal("error log leaked payload")
	}
	slow.Store(true)
	s.recordOperation(newID(), "GET /api/v1/documents/{id}", "GET", 200, time.Now(), false)
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("slow collector not called")
	}
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"otel_enabled": false}, 200)
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("disabled telemetry did not cancel in-flight")
	}
	operationWait(t, func() bool { op.mu.RLock(); defer op.mu.RUnlock(); return op.generation == nil })
	status := testJSONObject(t, admin.request("GET", "/api/v1/admin/operations/telemetry", nil, 200))
	if status["enabled"] != false || status["sent_spans"].(float64) < 3 {
		t.Fatal("actual telemetry counters invalid")
	}
	health := testJSONObject(t, admin.request("GET", "/api/v1/admin/operations/health", nil, 200))
	if health["database"].(map[string]any)["ready"] != true {
		t.Fatal("actual DB health missing")
	}
	safe := s.safeOperationRoute("GET /documents/TRACE_PRIVATE_ID?secret=TRACE_SECRET")
	if safe != "unmatched" {
		t.Fatal("unregistered actual path leaked")
	}
}
