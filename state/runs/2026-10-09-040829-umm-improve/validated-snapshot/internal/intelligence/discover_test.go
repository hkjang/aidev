package intelligence

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModelsEndpointMatchesEmbeddingsEndpointConventions(t *testing.T) {
	for _, testCase := range []struct{ base, want string }{
		{"http://embeddings:11434", "http://embeddings:11434/v1/models"},
		{"http://embeddings:11434/", "http://embeddings:11434/v1/models"},
		{"http://gateway/v1", "http://gateway/v1/models"},
		{"http://gateway/v1/models", "http://gateway/v1/models"},
	} {
		got, err := modelsEndpoint(testCase.base)
		if err != nil {
			t.Errorf("%s: %v", testCase.base, err)
			continue
		}
		if got != testCase.want {
			t.Errorf("%s -> %s, want %s", testCase.base, got, testCase.want)
		}
	}
	// The same addresses the embeddings endpoint refuses must be refused here,
	// or discovery becomes a way to reach schemes the rest of umm will not.
	for _, bad := range []string{"", "ftp://gateway", "not a url", "//gateway", "file:///etc/passwd"} {
		if _, err := modelsEndpoint(bad); err == nil {
			t.Errorf("%q was accepted as a gateway address", bad)
		}
	}
}

// A gateway typically serves a dozen chat models and one or two that embed, so
// the hint exists to put the two the operator wants at the top. It is a hint:
// the connection test is what settles whether a model embeds.
func TestEmbeddingNameHintNarrowsTheList(t *testing.T) {
	for _, name := range []string{
		"bge-m3", "paraphrase-multilingual", "text-embedding-3-small",
		"nomic-embed-text", "intfloat/multilingual-e5-small", "mxbai-embed-large",
	} {
		if !looksLikeEmbeddingModel(name) {
			t.Errorf("%q is a widely used embedding model and was not hinted", name)
		}
	}
	for _, name := range []string{"qwen3.6:27b", "gemma4:31b", "gpt-oss:20b", "llama3"} {
		if looksLikeEmbeddingModel(name) {
			t.Errorf("%q is a chat model and was hinted as an embedding one", name)
		}
	}
}

func TestDiscoverReportsWhatAGatewayOffers(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"qwen3.6:27b"},{"id":"bge-m3"},{"id":"gemma4:31b"}]}`))
	}))
	defer gateway.Close()

	models, err := listModels(context.Background(), gateway.Client(), gateway.URL)
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected every served model, got %d", len(models))
	}
	// The embedding model has to surface first, or the hint does no work.
	if models[0].Name != "bge-m3" || !models[0].LikelyEmbedding {
		t.Errorf("the likely embedding model was not first: %+v", models)
	}
}

// A candidate address is not necessarily an embedding gateway — it may be
// anything listening on that port — so its response is untrusted input.
func TestDiscoverySurvivesSomethingElseOnThePort(t *testing.T) {
	for _, handler := range []http.HandlerFunc{
		func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<html>hello</html>")) },
		func(w http.ResponseWriter, r *http.Request) { http.Error(w, "nope", http.StatusTeapot) },
		func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"data":"not-a-list"}`)) },
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "999999999")
			_, _ = w.Write([]byte(`{"data":[`))
		},
	} {
		server := httptest.NewServer(handler)
		models, err := listModels(context.Background(), server.Client(), server.URL)
		if err == nil && len(models) > 0 {
			t.Errorf("a non-gateway response produced %d models", len(models))
		}
		server.Close()
	}
}

// The candidate list is compiled in. If it ever became reachable from a request,
// the administrator screen would turn into a way of probing whatever the server
// can reach.
func TestDiscoveryCandidatesAreFixed(t *testing.T) {
	if len(discoveryCandidates) == 0 {
		t.Fatal("no candidates to probe")
	}
	for _, candidate := range discoveryCandidates {
		if _, err := modelsEndpoint(candidate); err != nil {
			t.Errorf("built-in candidate %q is not a usable address: %v", candidate, err)
		}
	}
}
