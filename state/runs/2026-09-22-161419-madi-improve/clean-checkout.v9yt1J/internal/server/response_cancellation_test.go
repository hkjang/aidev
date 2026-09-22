package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponseCancellationIsNotDatabaseFailure(t *testing.T) {
	for _, err := range []error{context.Canceled, fmt.Errorf("private driver argument: %w", context.Canceled)} {
		w := httptest.NewRecorder()
		respond(w, nil, err)
		if w.Code != http.StatusRequestTimeout || !strings.Contains(w.Body.String(), "요청이 취소되었습니다") || strings.Contains(w.Body.String(), "private") || w.Header().Get("Retry-After") != "" {
			t.Fatalf("cancelled response: status=%d body=%s", w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	respond(w, nil, context.DeadlineExceeded)
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Retry-After") != "1" {
		t.Fatalf("existing deadline contract changed: %d", w.Code)
	}
}
