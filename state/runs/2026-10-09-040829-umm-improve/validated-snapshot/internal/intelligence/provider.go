package intelligence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hkjang/umm/internal/textutil"
)

// LocalAlgorithm identifies the offline character n-gram embedding. It is the
// default and the fallback: umm must keep working with no AI gateway at all.
const LocalAlgorithm = "umm-local-chargram-v1"

const maxEmbeddingResponseBody = 32 << 20

// RemoteConfig points at an OpenAI compatible /v1/embeddings endpoint.
type RemoteConfig struct {
	BaseURL         string
	APIKey          string
	Model           string
	Timeout         time.Duration
	SettingsManaged bool
}

// Provider produces embeddings for a batch of texts. A zero Provider embeds
// locally, which is what every deployment gets until an administrator sets an
// embedding model in the AI gateway settings.
type Provider struct {
	Remote *RemoteConfig
	Client *http.Client
}

// Algorithm names the vectors this provider would produce. The endpoint
// fingerprint distinguishes gateways that expose the same model label but do
// not necessarily implement the same vector space. API keys and timeouts are
// deliberately excluded because changing transport policy does not change a
// model's vectors.
func (p Provider) Algorithm() string {
	if model := p.model(); model != "" {
		endpoint := strings.TrimSpace(p.Remote.BaseURL)
		if canonical, err := EmbeddingsEndpoint(endpoint); err == nil {
			endpoint = canonical
		}
		identity := sha256.Sum256([]byte(endpoint))
		return fmt.Sprintf("gateway:%s:%x", model, identity)
	}
	return LocalAlgorithm
}

func (p Provider) model() string {
	if p.Remote == nil {
		return ""
	}
	if strings.TrimSpace(p.Remote.BaseURL) == "" {
		return ""
	}
	return strings.TrimSpace(p.Remote.Model)
}

// Model returns the configured gateway model, or an empty string when the
// local algorithm is in use.
func (p Provider) Model() string { return p.model() }

// Embed returns one unit vector per input. The returned algorithm is the one
// actually used: when the gateway is unreachable the local vectors are labelled
// as local, so a degraded call can never be mistaken for a gateway result.
func (p Provider) Embed(ctx context.Context, texts []string) ([][]float32, string) {
	if len(texts) == 0 {
		return nil, p.Algorithm()
	}
	if p.model() == "" {
		return embedLocal(texts), LocalAlgorithm
	}
	vectors, err := p.embedRemote(ctx, texts)
	if err != nil {
		return embedLocal(texts), LocalAlgorithm
	}
	return vectors, p.Algorithm()
}

// EmbedStrict behaves like Embed but reports the gateway error instead of
// falling back, so an administrator testing the configuration sees the reason.
func (p Provider) EmbedStrict(ctx context.Context, texts []string) ([][]float32, error) {
	if p.model() == "" {
		return embedLocal(texts), nil
	}
	return p.embedRemote(ctx, texts)
}

func embedLocal(texts []string) [][]float32 {
	out := make([][]float32, len(texts))
	for i, text := range texts {
		out[i] = Embed(text)
	}
	return out
}

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (p Provider) embedRemote(ctx context.Context, texts []string) ([][]float32, error) {
	endpoint, err := EmbeddingsEndpoint(p.Remote.BaseURL)
	if err != nil {
		return nil, err
	}
	client := p.Client
	if client == nil {
		timeout := p.Remote.Timeout
		if timeout <= 0 {
			timeout = 45 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	raw, err := json.Marshal(embeddingRequest{Model: p.model(), Input: texts})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(p.Remote.APIKey); key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxEmbeddingResponseBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxEmbeddingResponseBody {
		return nil, errors.New("embedding response is too large")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		snippet := textutil.LimitUTF8Bytes(string(body), 300)
		return nil, fmt.Errorf("embedding gateway status %d: %s", response.StatusCode, snippet)
	}
	var parsed embeddingResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) != len(texts) {
		return nil, fmt.Errorf("embedding gateway returned %d vectors for %d inputs", len(parsed.Data), len(texts))
	}
	out := make([][]float32, len(texts))
	dimensions := 0
	for _, item := range parsed.Data {
		if item.Index < 0 || item.Index >= len(out) {
			return nil, errors.New("embedding gateway returned an out of range index")
		}
		if len(item.Embedding) == 0 {
			return nil, errors.New("embedding gateway returned an empty vector")
		}
		if dimensions == 0 {
			dimensions = len(item.Embedding)
		} else if len(item.Embedding) != dimensions {
			return nil, fmt.Errorf("embedding gateway returned inconsistent dimensions: %d and %d", dimensions, len(item.Embedding))
		}
		out[item.Index] = Normalize(item.Embedding)
	}
	for _, vector := range out {
		if vector == nil {
			return nil, errors.New("embedding gateway skipped an input")
		}
	}
	return out, nil
}

// Normalize scales a vector to unit length so Cosine stays a plain dot product.
func Normalize(vector []float32) []float32 {
	var norm float64
	for _, value := range vector {
		norm += float64(value) * float64(value)
	}
	if norm == 0 {
		return vector
	}
	scale := float32(math.Sqrt(norm))
	out := make([]float32, len(vector))
	for i, value := range vector {
		out[i] = value / scale
	}
	return out
}

// EmbeddingsEndpoint resolves the gateway base URL the same way the chat
// endpoint does, so administrators configure one base URL for both.
func EmbeddingsEndpoint(baseURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || !(u.Scheme == "http" || u.Scheme == "https") || u.Host == "" {
		return "", errors.New("invalid AI gateway URL")
	}
	path := strings.TrimRight(u.Path, "/")
	switch {
	case strings.HasSuffix(path, "/embeddings"):
	case strings.HasSuffix(path, "/v1"):
		path += "/embeddings"
	default:
		path += "/v1/embeddings"
	}
	u.Path = path
	u.RawPath = ""
	u.Fragment = ""
	return u.String(), nil
}
