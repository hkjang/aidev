package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Getting umm onto a real embedding model is three steps of friction: know that
// the default is lexical, know that a gateway is running somewhere, and know
// what the model is called. The first two are answered elsewhere. This answers
// the third by asking the gateways that would plausibly be there.
//
// The candidate list is fixed in the binary and never taken from a request.
// Probing an address someone supplies would turn the administrator screen into a
// way of reaching whatever the server can reach, and there is no version of this
// feature that needs it: the point is to find the sidecar umm itself documents.

// discoveryCandidates are the places an embedding gateway lives when someone
// followed umm's own instructions, or already had one running.
var discoveryCandidates = []string{
	// The sidecar in umm's compose file.
	"http://embeddings:11434",
	// A text-embeddings-inference style sidecar under the same service name.
	"http://embeddings:8080",
	// Ollama on the machine hosting the container, and beside the process.
	"http://host.docker.internal:11434",
	"http://127.0.0.1:11434",
}

// GatewayCandidate is one address that answered, and what it offers.
type GatewayCandidate struct {
	BaseURL string           `json:"baseUrl"`
	Models  []CandidateModel `json:"models"`
}

// CandidateModel is a model the gateway lists.
type CandidateModel struct {
	Name string `json:"name"`
	// LikelyEmbedding is a hint from the name, not a finding. A gateway lists
	// every model it serves and nothing in the response says which ones embed, so
	// this narrows a long list rather than deciding anything. The test and the
	// quality measurement are what settle it.
	LikelyEmbedding bool `json:"likelyEmbedding"`
}

// embeddingNameHints are substrings that appear in the names of widely used
// embedding models. Matching one is a reason to try a model first, not evidence.
var embeddingNameHints = []string{
	"embed", "bge", "gte", "e5", "minilm", "nomic", "paraphrase", "sentence", "arctic", "mxbai",
}

// DiscoverGateways probes the known addresses and reports what answered.
//
// Every candidate is tried concurrently with a short timeout, because the common
// case is that most of them are not there and an operator should not wait out
// four sequential connection failures to learn it.
func DiscoverGateways(ctx context.Context, client *http.Client) []GatewayCandidate {
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	type result struct {
		index     int
		candidate GatewayCandidate
		found     bool
	}
	results := make(chan result, len(discoveryCandidates))
	for index, baseURL := range discoveryCandidates {
		go func(index int, baseURL string) {
			models, err := listModels(ctx, client, baseURL)
			if err != nil || len(models) == 0 {
				results <- result{index: index}
				return
			}
			results <- result{index: index, candidate: GatewayCandidate{BaseURL: baseURL, Models: models}, found: true}
		}(index, baseURL)
	}
	found := make([]GatewayCandidate, 0, len(discoveryCandidates))
	order := map[string]int{}
	for range discoveryCandidates {
		r := <-results
		if r.found {
			found = append(found, r.candidate)
			order[r.candidate.BaseURL] = r.index
		}
	}
	// Report in the order the candidates are declared, so the sidecar umm
	// documents comes first rather than whichever host answered quickest.
	sort.Slice(found, func(i, j int) bool { return order[found[i].BaseURL] < order[found[j].BaseURL] })
	return found
}

// maxModelListBody bounds what a probe will read. A candidate address is not
// necessarily an embedding gateway — it might be anything listening on that
// port — so the response is treated as untrusted input.
const maxModelListBody = 1 << 20

func listModels(ctx context.Context, client *http.Client, baseURL string) ([]CandidateModel, error) {
	endpoint, err := modelsEndpoint(baseURL)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("model list status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxModelListBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxModelListBody {
		return nil, fmt.Errorf("model list is too large")
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err = json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	models := make([]CandidateModel, 0, len(payload.Data))
	for _, entry := range payload.Data {
		name := strings.TrimSpace(entry.ID)
		if name == "" {
			continue
		}
		models = append(models, CandidateModel{Name: name, LikelyEmbedding: looksLikeEmbeddingModel(name)})
	}
	// Likely embedding models first: a gateway commonly serves a dozen chat
	// models and two that embed, and the operator is looking for the two.
	sort.SliceStable(models, func(i, j int) bool { return models[i].LikelyEmbedding && !models[j].LikelyEmbedding })
	return models, nil
}

func looksLikeEmbeddingModel(name string) bool {
	lower := strings.ToLower(name)
	for _, hint := range embeddingNameHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// modelsEndpoint builds the OpenAI-compatible model listing URL, mirroring how
// EmbeddingsEndpoint treats a base address so the two agree about what a gateway
// URL means.
func modelsEndpoint(baseURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || !(u.Scheme == "http" || u.Scheme == "https") || u.Host == "" {
		return "", fmt.Errorf("invalid gateway URL")
	}
	path := strings.TrimRight(u.Path, "/")
	switch {
	case strings.HasSuffix(path, "/models"):
	case strings.HasSuffix(path, "/v1"):
		path += "/models"
	default:
		path += "/v1/models"
	}
	u.Path, u.RawPath, u.Fragment = path, "", ""
	return u.String(), nil
}
