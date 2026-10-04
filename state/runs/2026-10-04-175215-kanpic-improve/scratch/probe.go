package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type probe struct{ console slog.Handler }

func (h probe) Enabled(ctx context.Context, l slog.Level) bool { return h.console.Enabled(ctx, l) }
func (h probe) WithAttrs(a []slog.Attr) slog.Handler           { return probe{h.console.WithAttrs(a)} }
func (h probe) WithGroup(n string) slog.Handler                { return probe{h.console.WithGroup(n)} }
func (h probe) Handle(ctx context.Context, r slog.Record) error {
	_ = h.console.Handle(ctx, r)
	attributes := make(map[string]any)
	r.Attrs(func(attr slog.Attr) bool { attributes[attr.Key] = attr.Value.Any(); return true })
	data, err := json.Marshal(attributes)
	fmt.Printf("DB attributes = %s (marshal err=%v)\n", data, err)
	traceID, _ := attributes["trace_id"].(string)
	fmt.Printf("DB trace_id   = %q\n", traceID)
	return nil
}

func main() {
	var console bytes.Buffer
	logger := slog.New(probe{slog.NewJSONHandler(&console, nil)})
	wrapped := fmt.Errorf("상위: %w", errors.New("연결 거부"))
	logger.Error("request failed", "error", wrapped, "path", "/api/x", "duration_ms", 12)
	fmt.Printf("console       = %s", console.String())
	console.Reset()
	fmt.Println("--- With ---")
	logger.With("trace_id", "T-1", "actor", "kim").Info("http request", "method", "GET")
	fmt.Printf("console       = %s", console.String())
	console.Reset()
	fmt.Println("--- group / duration / ctx error ---")
	logger.Info("x", slog.Group("g", "a", 1), "dur", 3*time.Second, "ctxerr", context.Canceled)
	fmt.Printf("console       = %s", console.String())
}
