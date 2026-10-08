package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeBody(t *testing.T, raw string, target any) error {
	t.Helper()
	request := httptest.NewRequest("POST", "/", strings.NewReader(raw))
	return decodeJSON(httptest.NewRecorder(), request, target)
}

func TestNoteWriteRequestAcceptsMinimalPayload(t *testing.T) {
	var body noteWriteRequest
	err := decodeBody(t, `{"content":"새 생각","title":"","color":"yellow","kind":"thought","x":120,"y":160,"width":240,"height":160,"rotation":0}`, &body)
	if err != nil {
		t.Fatalf("minimal note payload rejected: %v", err)
	}
	if note := body.note(); note.Content != "새 생각" || note.Width != 240 || note.Height != 160 {
		t.Fatalf("note payload mapped incorrectly: %#v", note)
	}
}

func TestNoteWriteRequestIgnoresServerManagedFields(t *testing.T) {
	var body noteWriteRequest
	err := decodeBody(t, `{"id":"","spaceId":"","authorId":"","content":"새 생각","title":"","color":"yellow","kind":"thought","source":"user","x":120,"y":160,"width":240,"height":160,"rotation":0,"version":0,"createdAt":"","updatedAt":"","relatedCount":0}`, &body)
	if err != nil {
		t.Fatalf("response-shaped note payload rejected: %v", err)
	}
	if note := body.note(); note.ID.String() != "00000000-0000-0000-0000-000000000000" || note.Content != "새 생각" {
		t.Fatalf("server-managed fields were not ignored: %#v", note)
	}
}

func TestEdgeWriteRequestIgnoresServerManagedFields(t *testing.T) {
	var body edgeWriteRequest
	err := decodeBody(t, `{"id":"","spaceId":"","source":"7fbd99aa-cf6b-4cb1-9d3a-606602111234","target":"7fbd99aa-cf6b-4cb1-9d3a-606602115678","relation":"related"}`, &body)
	if err != nil {
		t.Fatalf("response-shaped edge payload rejected: %v", err)
	}
	if body.SourceID == body.TargetID || body.Relation != "related" {
		t.Fatalf("edge payload mapped incorrectly: %#v", body)
	}
}

func TestWriteRequestsRemainStrict(t *testing.T) {
	var body noteWriteRequest
	if err := decodeBody(t, `{"content":"새 생각","unexpected":true}`, &body); err == nil {
		t.Fatal("unknown note field was accepted")
	}
}
