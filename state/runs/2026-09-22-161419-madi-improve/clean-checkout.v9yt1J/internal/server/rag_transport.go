package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Provider values come from encrypted administrator settings, never from an
// untrusted search request. Embeddings have a JSON response (unlike chat SSE).
// Contract: https://developers.openai.com/api/reference/ruby/resources/embeddings/methods/create
type ragProvider struct {
	BaseURL, Model, APIKey, CA string
	AllowHTTP                  bool
	Dimensions                 int
}

func ragEndpoint(base, suffix string) (string, error) {
	u, e := url.Parse(strings.TrimSpace(base))
	if e != nil || u.Hostname() == "" || !oneOf(u.Scheme, "https", "http") || u.User != nil || u.Fragment != "" || len(base) > 4096 {
		return "", errors.New("검색 AI 공급자의 API 주소를 확인하세요")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/"+suffix) {
		u.Path += "/" + suffix
	}
	return u.String(), nil
}

func ragHTTP(p ragProvider) (*http.Client, error) {
	u, e := url.Parse(p.BaseURL)
	if e != nil || (u.Scheme == "http" && !p.AllowHTTP) {
		return nil, errors.New("암호화되지 않은 검색 AI 연결은 관리자의 명시적 허용이 필요합니다")
	}
	roots, _ := x509.SystemCertPool()
	if roots == nil {
		roots = x509.NewCertPool()
	}
	if p.CA != "" && !roots.AppendCertsFromPEM([]byte(p.CA)) {
		return nil, errors.New("검색 AI CA 인증서 형식을 확인하세요")
	}
	transport := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 45 * time.Second, DisableKeepAlives: true}
	return &http.Client{Transport: transport, Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
}

// Return vectors in input order using the explicit index field. Never silently
// associate a shuffled/malformed provider response with a different document.
func ragEmbeddings(ctx context.Context, p ragProvider, inputs []string) ([][]float32, error) {
	if p.Model == "" || len(p.Model) > 256 || len(inputs) < 1 || len(inputs) > 32 || p.Dimensions < 0 || p.Dimensions > 8192 {
		return nil, errors.New("임베딩 모델·입력 개수·벡터 차원을 확인하세요")
	}
	for _, text := range inputs {
		if strings.TrimSpace(text) == "" || len(text) > 8192 {
			return nil, errors.New("임베딩 조각은 비어 있지 않은 8192바이트 이하 텍스트여야 합니다")
		}
	}
	endpoint, e := ragEndpoint(p.BaseURL, "embeddings")
	if e != nil {
		return nil, e
	}
	client, e := ragHTTP(p)
	if e != nil {
		return nil, e
	}
	payload := map[string]any{"model": p.Model, "input": inputs, "encoding_format": "float"}
	// Omit dimensions unless explicitly configured: many local models and older
	// OpenAI-compatible providers do not support that optional request field.
	if p.Dimensions > 0 {
		payload["dimensions"] = p.Dimensions
	}
	req, e := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonValue(payload)))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
		req.Header.Set("api-key", p.APIKey)
	}
	res, e := client.Do(req)
	if e != nil {
		return nil, errors.New("임베딩 공급자에 연결할 수 없습니다. 주소·네트워크·인증서를 확인하세요")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("임베딩 공급자가 HTTP %d 오류를 반환했습니다", res.StatusCode)
	}
	if !strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "application/json") {
		return nil, errors.New("임베딩 공급자가 JSON 벡터 응답을 반환하지 않았습니다")
	}
	raw, e := io.ReadAll(io.LimitReader(res.Body, (16<<20)+1))
	if e != nil || len(raw) > 16<<20 {
		return nil, errors.New("임베딩 응답 크기 또는 읽기 오류입니다")
	}
	var body struct {
		Data []struct {
			Index  *int      `json:"index"`
			Vector []float32 `json:"embedding"`
		} `json:"data"`
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(raw, &body) != nil || len(body.Data) != len(inputs) || (len(body.Error) > 0 && string(body.Error) != "null") {
		return nil, errors.New("임베딩 응답의 입력별 벡터 개수가 맞지 않습니다")
	}
	out := make([][]float32, len(inputs))
	dimension := p.Dimensions
	for _, item := range body.Data {
		if item.Index == nil || *item.Index < 0 || *item.Index >= len(inputs) || out[*item.Index] != nil || len(item.Vector) < 1 || len(item.Vector) > 8192 {
			return nil, errors.New("임베딩 응답의 순서·중복·차원이 올바르지 않습니다")
		}
		if dimension == 0 {
			dimension = len(item.Vector)
		}
		if len(item.Vector) != dimension {
			return nil, errors.New("임베딩 모델의 벡터 차원이 변경되었습니다. 설정과 색인을 확인하세요")
		}
		var norm float64
		for _, v := range item.Vector {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return nil, errors.New("유효하지 않은 임베딩 벡터입니다")
			}
			norm += float64(v) * float64(v)
		}
		if norm == 0 {
			return nil, errors.New("0 벡터 임베딩은 검색에 사용할 수 없습니다")
		}
		out[*item.Index] = item.Vector
	}
	return out, nil
}

func ragCosine(a, b []float32) (float64, bool) {
	if len(a) == 0 || len(a) != len(b) {
		return 0, false
	}
	var dot, aa, bb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
			return 0, false
		}
		dot += x * y
		aa += x * x
		bb += y * y
	}
	if aa == 0 || bb == 0 {
		return 0, false
	}
	return max(-1, min(1, dot/math.Sqrt(aa*bb))), true
}
