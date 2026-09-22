package server

import (
	"net/http/httptest"
	"testing"
)

func TestRAGGenerationUnavailableIsConflict(t *testing.T) {
	recorder := httptest.NewRecorder()
	ragIndexError(recorder, ragGenerationUnavailable("실제 HNSW 인덱스가 아직 준비되지 않았습니다"))
	if recorder.Code != 409 {
		t.Fatal("expected operational state reported as server fault", recorder.Code, recorder.Body.String())
	}
}
