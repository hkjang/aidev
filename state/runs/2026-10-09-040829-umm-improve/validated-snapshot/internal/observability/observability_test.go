package observability

import (
	"strings"
	"testing"
	"time"
)

func TestRegistrySnapshotAndPrometheus(t *testing.T) {
	registry := NewRegistry()
	registry.Begin()
	registry.Observe("GET", "/api/v1/today", 200, 12*time.Millisecond)
	registry.Begin()
	registry.Observe("POST", "/api/v1/notes/{noteID}", 503, 300*time.Millisecond)

	snapshot := registry.Snapshot()
	if snapshot["requests"] != uint64(2) || snapshot["serverErrors"] != uint64(1) || snapshot["inFlight"] != int64(0) {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	var output strings.Builder
	registry.Prometheus(&output, `0.7.0"test`)
	metrics := output.String()
	for _, expected := range []string{
		`umm_build_info{version="0.7.0\"test"} 1`,
		`umm_http_requests_total{method="GET",route="/api/v1/today",status="2xx"} 1`,
		`umm_http_requests_total{method="POST",route="/api/v1/notes/{noteID}",status="5xx"} 1`,
		`umm_http_request_duration_seconds_bucket`,
	} {
		if !strings.Contains(metrics, expected) {
			t.Errorf("metrics missing %q\n%s", expected, metrics)
		}
	}
}
