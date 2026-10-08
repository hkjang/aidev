package intelligence

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestProviderDefaultsToLocalAlgorithm(t *testing.T) {
	var provider Provider
	if got := provider.Algorithm(); got != LocalAlgorithm {
		t.Fatalf("an unconfigured provider must stay offline, got %q", got)
	}
	vectors, algorithm := provider.Embed(context.Background(), []string{"생각"})
	if algorithm != LocalAlgorithm || len(vectors) != 1 || len(vectors[0]) != Dimensions {
		t.Fatalf("expected one local vector of %d dimensions, got %d vectors labelled %q", Dimensions, len(vectors), algorithm)
	}
}

func TestProviderWithoutModelStaysLocal(t *testing.T) {
	provider := Provider{Remote: &RemoteConfig{BaseURL: "https://gateway.internal", Model: "  "}}
	if got := provider.Algorithm(); got != LocalAlgorithm {
		t.Fatalf("a base URL without a model must not switch the algorithm, got %q", got)
	}
}

func TestProviderUsesTheGatewayAndNormalises(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("credential was not forwarded, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"index": 0, "embedding": []float32{3, 4}}}})
	}))
	defer server.Close()

	provider := Provider{Remote: &RemoteConfig{BaseURL: server.URL, APIKey: "secret", Model: "text-embed", Timeout: 5 * time.Second}}
	algorithm := provider.Algorithm()
	if !strings.HasPrefix(algorithm, "gateway:text-embed:") {
		t.Fatalf("expected the gateway algorithm label, got %q", algorithm)
	}
	vectors, actualAlgorithm := provider.Embed(context.Background(), []string{"생각"})
	if actualAlgorithm != algorithm {
		t.Fatalf("expected the gateway label %q, got %q", algorithm, actualAlgorithm)
	}
	if math.Abs(float64(vectors[0][0])-0.6) > 1e-6 || math.Abs(float64(vectors[0][1])-0.8) > 1e-6 {
		t.Fatalf("gateway vectors must be normalised for Cosine, got %v", vectors[0])
	}
}

func TestProviderAlgorithmIncludesCanonicalGatewayIdentity(t *testing.T) {
	first := Provider{Remote: &RemoteConfig{BaseURL: "https://gateway-a.internal/v1", APIKey: "first", Model: "shared-model", Timeout: time.Second}}
	equivalent := Provider{Remote: &RemoteConfig{BaseURL: "https://gateway-a.internal/v1/embeddings", APIKey: "second", Model: "shared-model", Timeout: 2 * time.Second}}
	managed := Provider{Remote: &RemoteConfig{BaseURL: "https://gateway-a.internal/v1", Model: "shared-model", SettingsManaged: true}}
	second := Provider{Remote: &RemoteConfig{BaseURL: "https://gateway-b.internal/v1", APIKey: "first", Model: "shared-model", Timeout: time.Second}}

	if first.Algorithm() != equivalent.Algorithm() {
		t.Fatalf("equivalent embedding endpoints must share a vector space: %q != %q", first.Algorithm(), equivalent.Algorithm())
	}
	if first.Algorithm() != managed.Algorithm() {
		t.Fatalf("settings ownership must not change the vector space: %q != %q", first.Algorithm(), managed.Algorithm())
	}
	if first.Algorithm() == second.Algorithm() {
		t.Fatalf("different gateways with the same model label shared an algorithm: %q", first.Algorithm())
	}
	if strings.Contains(first.Algorithm(), "gateway-a.internal") {
		t.Fatalf("the persisted algorithm must fingerprint, not disclose, the endpoint: %q", first.Algorithm())
	}
}

// A degraded gateway must never be mistaken for a successful one: the returned
// algorithm has to say the vectors are local, or they would be stored under the
// gateway label and compared against incompatible vectors later.
func TestProviderFallsBackAndReportsLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream is down", http.StatusBadGateway)
	}))
	defer server.Close()

	provider := Provider{Remote: &RemoteConfig{BaseURL: server.URL, Model: "text-embed", Timeout: time.Second}}
	vectors, algorithm := provider.Embed(context.Background(), []string{"생각"})
	if algorithm != LocalAlgorithm {
		t.Fatalf("a failed gateway call must report local vectors, got %q", algorithm)
	}
	if len(vectors) != 1 || len(vectors[0]) != Dimensions {
		t.Fatalf("expected a usable local fallback vector, got %d vectors", len(vectors))
	}
	if _, err := provider.EmbedStrict(context.Background(), []string{"생각"}); err == nil {
		t.Fatal("EmbedStrict must surface the gateway error for the settings screen")
	}
}

func TestProviderBoundsGatewayErrorOnUTF8Boundary(t *testing.T) {
	body := strings.Repeat("a", 299) + "한tail"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	provider := Provider{Remote: &RemoteConfig{BaseURL: server.URL, Model: "text-embed", Timeout: time.Second}}
	_, err := provider.EmbedStrict(context.Background(), []string{"생각"})
	if err == nil {
		t.Fatal("EmbedStrict must surface the gateway error")
	}
	want := "embedding gateway status 502: " + strings.Repeat("a", 299)
	if err.Error() != want || !utf8.ValidString(err.Error()) {
		t.Fatalf("gateway error = %q, want valid UTF-8 %q", err, want)
	}
}

func TestProviderRejectsAMismatchedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"index": 0, "embedding": []float32{1}}}})
	}))
	defer server.Close()

	provider := Provider{Remote: &RemoteConfig{BaseURL: server.URL, Model: "m", Timeout: time.Second}}
	if _, err := provider.EmbedStrict(context.Background(), []string{"a", "b"}); err == nil {
		t.Fatal("a response with fewer vectors than inputs must be an error, not a silent misalignment")
	}
}

func TestProviderRejectsInconsistentDimensionsAndFallsBack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
			{"index": 0, "embedding": []float32{1, 0}},
			{"index": 1, "embedding": []float32{1, 0, 0}},
		}})
	}))
	defer server.Close()

	provider := Provider{Remote: &RemoteConfig{BaseURL: server.URL, Model: "m", Timeout: time.Second}}
	if _, err := provider.EmbedStrict(context.Background(), []string{"a", "b"}); err == nil || !strings.Contains(err.Error(), "inconsistent dimensions") {
		t.Fatalf("mixed embedding dimensions must be rejected, got %v", err)
	}
	vectors, algorithm := provider.Embed(context.Background(), []string{"a", "b"})
	if algorithm != LocalAlgorithm || len(vectors) != 2 {
		t.Fatalf("malformed remote space must fall back for both inputs: algorithm=%q vectors=%d", algorithm, len(vectors))
	}
	if len(vectors[0]) != Dimensions || len(vectors[1]) != Dimensions {
		t.Fatalf("malformed remote space must fall back as one local space: algorithm=%q dimensions=%d/%d", algorithm, len(vectors[0]), len(vectors[1]))
	}
}

func TestEmbeddingsEndpoint(t *testing.T) {
	cases := map[string]string{
		"https://gw.internal":               "https://gw.internal/v1/embeddings",
		"https://gw.internal/":              "https://gw.internal/v1/embeddings",
		"https://gw.internal/v1":            "https://gw.internal/v1/embeddings",
		"https://gw.internal/v1/embeddings": "https://gw.internal/v1/embeddings",
		"https://gw.internal/openai/v1":     "https://gw.internal/openai/v1/embeddings",
	}
	for input, want := range cases {
		got, err := EmbeddingsEndpoint(input)
		if err != nil || got != want {
			t.Fatalf("%s: want %s, got %s (err=%v)", input, want, got, err)
		}
	}
	for _, invalid := range []string{"", "ftp://gw", "not a url", "/relative"} {
		if _, err := EmbeddingsEndpoint(invalid); err == nil {
			t.Fatalf("expected %q to be rejected", invalid)
		}
	}
}
