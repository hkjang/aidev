package observability

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var latencyBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

type metricKey struct {
	Method string
	Route  string
	Status string
}

type requestMetric struct {
	Count      uint64
	Duration   float64
	BucketHits []uint64
}

type gauge struct {
	Value float64
	Help  string
}

type Registry struct {
	mu       sync.RWMutex
	started  time.Time
	inFlight int64
	requests map[metricKey]*requestMetric
	gauges   map[string]gauge
}

func NewRegistry() *Registry {
	return &Registry{started: time.Now(), requests: map[metricKey]*requestMetric{}}
}

func (r *Registry) Begin() {
	r.mu.Lock()
	r.inFlight++
	r.mu.Unlock()
}

func (r *Registry) Observe(method, route string, status int, duration time.Duration) {
	if route == "" {
		route = "unmatched"
	}
	statusClass := strconv.Itoa(status/100) + "xx"
	key := metricKey{Method: method, Route: route, Status: statusClass}
	seconds := duration.Seconds()
	r.mu.Lock()
	r.inFlight--
	metric := r.requests[key]
	if metric == nil {
		metric = &requestMetric{BucketHits: make([]uint64, len(latencyBuckets))}
		r.requests[key] = metric
	}
	metric.Count++
	metric.Duration += seconds
	for index, upper := range latencyBuckets {
		if seconds <= upper {
			metric.BucketHits[index]++
		}
	}
	r.mu.Unlock()
}

func (r *Registry) Snapshot() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var total, errors uint64
	for key, value := range r.requests {
		total += value.Count
		if key.Status == "5xx" {
			errors += value.Count
		}
	}
	return map[string]any{"uptimeSeconds": int64(time.Since(r.started).Seconds()), "requests": total, "serverErrors": errors, "inFlight": r.inFlight}
}

// Gauge records a named point-in-time value that Prometheus should expose
// alongside the request metrics, such as the realtime subscriber count.
func (r *Registry) Gauge(name string, value float64, help string) {
	r.mu.Lock()
	if r.gauges == nil {
		r.gauges = map[string]gauge{}
	}
	r.gauges[name] = gauge{Value: value, Help: help}
	r.mu.Unlock()
}

func (r *Registry) Prometheus(w io.Writer, version string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fmt.Fprintln(w, "# HELP umm_build_info Build and release information.")
	fmt.Fprintln(w, "# TYPE umm_build_info gauge")
	fmt.Fprintf(w, "umm_build_info{version=\"%s\"} 1\n", escapeLabel(version))
	fmt.Fprintln(w, "# HELP umm_http_requests_in_flight Current in-flight HTTP requests.")
	fmt.Fprintln(w, "# TYPE umm_http_requests_in_flight gauge")
	fmt.Fprintf(w, "umm_http_requests_in_flight %d\n", r.inFlight)
	names := make([]string, 0, len(r.gauges))
	for name := range r.gauges {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		metric := r.gauges[name]
		fmt.Fprintf(w, "# HELP %s %s\n", name, metric.Help)
		fmt.Fprintf(w, "# TYPE %s gauge\n", name)
		fmt.Fprintf(w, "%s %g\n", name, metric.Value)
	}
	fmt.Fprintln(w, "# HELP umm_http_requests_total HTTP requests by method, route and status class.")
	fmt.Fprintln(w, "# TYPE umm_http_requests_total counter")
	fmt.Fprintln(w, "# HELP umm_http_request_duration_seconds HTTP request latency histogram.")
	fmt.Fprintln(w, "# TYPE umm_http_request_duration_seconds histogram")
	keys := make([]metricKey, 0, len(r.requests))
	for key := range r.requests {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Method+keys[i].Route+keys[i].Status < keys[j].Method+keys[j].Route+keys[j].Status
	})
	for _, key := range keys {
		metric := r.requests[key]
		labels := fmt.Sprintf("method=\"%s\",route=\"%s\",status=\"%s\"", escapeLabel(key.Method), escapeLabel(key.Route), escapeLabel(key.Status))
		fmt.Fprintf(w, "umm_http_requests_total{%s} %d\n", labels, metric.Count)
		for index, upper := range latencyBuckets {
			fmt.Fprintf(w, "umm_http_request_duration_seconds_bucket{%s,le=%q} %d\n", labels, strconv.FormatFloat(upper, 'f', -1, 64), metric.BucketHits[index])
		}
		fmt.Fprintf(w, "umm_http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", labels, metric.Count)
		fmt.Fprintf(w, "umm_http_request_duration_seconds_sum{%s} %g\n", labels, metric.Duration)
		fmt.Fprintf(w, "umm_http_request_duration_seconds_count{%s} %d\n", labels, metric.Count)
	}
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func Wrap(handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, "umm.http")
}

func Init(ctx context.Context, version string) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) == "" && strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")) == "" {
		return func(context.Context) error { return nil }, nil
	}
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	configured := resource.NewSchemaless(
		attribute.String("service.name", "umm"),
		attribute.String("service.version", version),
		attribute.String("deployment.environment.name", "self-hosted"),
	)
	res, err := resource.Merge(resource.Default(), configured)
	if err != nil {
		return nil, err
	}
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}
