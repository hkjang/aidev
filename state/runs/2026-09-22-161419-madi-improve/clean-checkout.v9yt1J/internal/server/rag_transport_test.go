package server

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRAGEmbeddingProtocolAndTLS(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" || r.Header.Get("Authorization") != "Bearer private-test" {
			t.Error("unexpected contract")
		}
		var in map[string]any
		json.NewDecoder(r.Body).Decode(&in)
		if in["encoding_format"] != "float" || in["model"] != "local-embedding" || in["stream"] != nil || in["dimensions"] != nil {
			t.Error(in)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"index":1,"embedding":[0,1]},{"index":0,"embedding":[1,0]}]}`)
	}))
	defer upstream.Close()
	p := ragProvider{BaseURL: upstream.URL + "/v1", Model: "local-embedding", APIKey: "private-test", CA: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: upstream.Certificate().Raw}))}
	v, e := ragEmbeddings(t.Context(), p, []string{"문서 하나", "문서 둘"})
	if e != nil || len(v) != 2 || v[0][0] != 1 || v[1][1] != 1 {
		t.Fatal(v, e)
	}
	if value, ok := ragCosine(v[0], v[1]); !ok || value != 0 {
		t.Fatal(value, ok)
	}
	if value, ok := ragCosine(v[0], v[0]); !ok || value != 1 {
		t.Fatal(value, ok)
	}
	u, e := ragEndpoint("https://llm.internal/deployments/embed?api-version=2026-01-01", "embeddings")
	if e != nil || u != "https://llm.internal/deployments/embed/embeddings?api-version=2026-01-01" {
		t.Fatal(u, e)
	}
}

func TestRAGEmbeddingRejectsAmbiguousVectors(t *testing.T) {
	for name, body := range map[string]string{
		"missing-index":  `{"data":[{"embedding":[1,2]},{"index":1,"embedding":[1,2]}]}`,
		"duplicate":      `{"data":[{"index":0,"embedding":[1,2]},{"index":0,"embedding":[1,2]}]}`,
		"zero":           `{"data":[{"index":0,"embedding":[0,0]},{"index":1,"embedding":[1,2]}]}`,
		"dimensions":     `{"data":[{"index":0,"embedding":[1]},{"index":1,"embedding":[1,2]}]}`,
		"range":          `{"data":[{"index":0,"embedding":[1,2]},{"index":2,"embedding":[1,2]}]}`,
		"count":          `{"data":[{"index":0,"embedding":[1,2]}]}`,
		"provider-error": `{"error":{"message":"private-provider-echo"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, body)
			}))
			defer up.Close()
			_, e := ragEmbeddings(t.Context(), ragProvider{BaseURL: up.URL, Model: "local", AllowHTTP: true}, []string{"one", "two"})
			if e == nil || strings.Contains(e.Error(), "private-provider-echo") {
				t.Fatal(e)
			}
		})
	}
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer target.Close()
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer up.Close()
	_, e := ragEmbeddings(context.Background(), ragProvider{BaseURL: up.URL, Model: "local", AllowHTTP: true}, []string{"one"})
	if e == nil || called {
		t.Fatal("followed embedding redirect", e)
	}
	if _, e = ragEmbeddings(t.Context(), ragProvider{BaseURL: up.URL, Model: "local"}, []string{"one"}); e == nil {
		t.Fatal("plaintext silently enabled")
	}
}
