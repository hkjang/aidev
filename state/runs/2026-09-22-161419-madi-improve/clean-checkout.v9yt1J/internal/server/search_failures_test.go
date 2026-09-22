package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSearchFailureClassification(t *testing.T) {
	w := httptest.NewRecorder()
	searchFailure(w, context.Background(), context.DeadlineExceeded)
	var out map[string]any
	if json.Unmarshal(w.Body.Bytes(), &out) != nil || w.Code != 504 || str(out, "outcome") != "timeout" {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, item := range []struct {
		status  int
		outcome string
	}{{400, "invalid_query"}, {403, "scope_unavailable"}} {
		w = httptest.NewRecorder()
		searchAPIError(w, item.status, "범위를 확인하세요")
		if json.Unmarshal(w.Body.Bytes(), &out) != nil || str(out, "outcome") != item.outcome {
			t.Fatal(w.Body.String())
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	w = httptest.NewRecorder()
	searchFailure(w, cancelled, errors.New("private database detail"))
	if w.Code != 504 {
		t.Fatal(w.Code)
	}
}

func TestPostgresSearchIncompleteGramFallbackAndStaleCursor(t *testing.T) {
	s, admin, viewer, wid, _ := collaborationTestSetup(t)
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "정규화 원문", "markdown": "쿠버네티스장애 대응"}, 200))
	id := str(doc, "id")
	if _, e := s.indexSearchDocument(t.Context(), id); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(t.Context(), `UPDATE search_folded_documents SET grams_complete=false,gram_vector=''::tsvector WHERE document_id=$1`, id); e != nil {
		t.Fatal(e)
	}
	result := testJSONObject(t, viewer.request("GET", "/api/v1/search?workspace_id="+wid+"&type=document&q="+url.QueryEscape("쿠 버 네 티 스 장 애"), nil, 200))
	if len(result["results"].([]any)) != 1 {
		t.Fatal("incomplete bigram filter dropped exact result", result)
	}
	// No-match does not pretend current unindexed data is a completed normalized scan.
	result = testJSONObject(t, viewer.request("GET", "/api/v1/search?workspace_id="+wid+"&type=document&q="+url.QueryEscape("아직없는검색질의"), nil, 200))
	if str(result, "outcome") != "index_pending" {
		t.Fatal("missing accessible seed projection not disclosed", result)
	}
	doc2 := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "정규화 원문 2", "markdown": "쿠버네티스장애 대응"}, 200))
	if _, e := s.indexSearchDocument(t.Context(), str(doc2, "id")); e != nil {
		t.Fatal(e)
	}
	base := "/api/v1/search?workspace_id=" + wid + "&type=document&limit=1&q=" + url.QueryEscape("쿠 버 네 티 스 장 애")
	result = testJSONObject(t, viewer.request("GET", base, nil, 200))
	cursor := str(result, "next_cursor")
	if cursor == "" {
		t.Fatal("no next cursor")
	}
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/search-dictionary", map[string]any{"revision": 0, "confirm_shared": true, "entries": []searchDictionaryEntry{{Canonical: "쿠버네티스", Aliases: []string{"k8s"}}}}, 200)
	changed := testJSONObject(t, viewer.request("GET", base+"&cursor="+url.QueryEscape(cursor), nil, 409))
	if str(changed, "outcome") != "search_changed" {
		t.Fatal(changed)
	}
}
