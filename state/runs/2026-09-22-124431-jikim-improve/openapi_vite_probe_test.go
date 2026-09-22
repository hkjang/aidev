package httpapi

import (
 "context"
 "net/http"
 "net/http/httptest"
 "os/exec"
 "sync/atomic"
 "testing"
 "time"
)

func TestOpenAPIViteProbe(t *testing.T) {
 mux := http.NewServeMux()
 quietServer().routes(mux)
 var arrivals atomic.Int32
 upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path == "/api/openapi.json" { arrivals.Add(1) }
  mux.ServeHTTP(w, r)
 }))
 defer upstream.Close()
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()
 cmd := exec.CommandContext(ctx, "/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-124431-jikim-improve/tools/node_modules/node/bin/node", "/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-124431-jikim-improve/openapi-vite-probe.mjs", "/home/hkjang/.cache/auto-improve-wt/jikim/web", upstream.URL)
 output, err := cmd.CombinedOutput()
 t.Log(string(output))
 if err != nil { t.Fatal(err) }
 if arrivals.Load() != 4 { t.Fatalf("upstream arrivals=%d, want 4", arrivals.Load()) }
}
