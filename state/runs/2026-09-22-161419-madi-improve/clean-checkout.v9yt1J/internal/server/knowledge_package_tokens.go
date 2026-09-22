package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// https://developers.openai.com/api/reference/resources/responses/subresources/input_tokens/methods/count
// This is a token-count metadata request (JSON), not an AI text completion. No
// hidden fallback sends source text elsewhere when a local provider lacks it.
func responsePackageTokens(ctx context.Context, cfg map[string]any, allowHTTP bool, text string) (int, error) {
	if len(text) > 2<<20 {
		return 0, errors.New("토큰 계산 입력은 2MiB 이하여야 합니다")
	}
	endpoint, err := aiEndpoint(str(cfg, "ai_base_url"))
	if err != nil {
		return 0, err
	}
	endpoint = strings.TrimSuffix(endpoint, "/chat/completions") + "/responses/input_tokens"
	client, err := ragHTTP(ragProvider{BaseURL: endpoint, AllowHTTP: allowHTTP})
	if err != nil {
		return 0, err
	}
	client.Timeout = 15 * time.Second
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonValue(map[string]any{"model": str(cfg, "ai_model"), "input": text})))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if secret := str(cfg, "ai_api_key"); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
		req.Header.Set("api-key", secret)
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, errors.New("모델 토큰 계산 연결이 중단되었습니다")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return 0, fmt.Errorf("모델 토큰 계산 HTTP %d: 공급자가 input_tokens를 지원하는지 확인하세요. 추정치로 자동 대체하지 않았습니다", res.StatusCode)
	}
	if !strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "application/json") {
		return 0, errors.New("모델 토큰 계산 응답은 JSON이어야 합니다")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 65537))
	if err != nil || len(raw) > 65536 {
		return 0, errors.New("모델 토큰 계산 응답 크기를 확인하세요")
	}
	var result struct {
		Count  *int   `json:"input_tokens"`
		Object string `json:"object"`
	}
	if json.Unmarshal(raw, &result) != nil || result.Count == nil || *result.Count < 1 || *result.Count > 16<<20 || result.Object != "response.input_tokens" {
		return 0, errors.New("모델 토큰 계산 응답 값이 유효하지 않습니다")
	}
	return *result.Count, nil
}
