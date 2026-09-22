package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

func agentToolSchema(name string) map[string]any {
	props := map[string]any{}
	required := []string{}
	add := func(key, kind string) { props[key] = map[string]any{"type": kind}; required = append(required, key) }
	description := ""
	switch name {
	case "search_documents":
		description = "허용된 지식 범위에서 현재 읽을 수 있는 Markdown 문서 검색. 출처 버전과 정확한 조각을 반환합니다."
		add("query", "string")
	case "get_document":
		description = "허용된 문서의 지정 바이트 위치부터 최대 16KB를 읽습니다."
		add("document_id", "string")
		add("start_byte", "integer")
	case "get_graph":
		description = "허용된 문서 사이의 연결과 제목을 조회합니다(최대 100개)."
		add("query", "string")
	case "query_database":
		description = "허용된 DB의 명시 속성과 필터를 안전하게 조회합니다. SQL을 실행하지 않습니다."
		add("database_id", "string")
		props["property_ids"] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "maxItems": 20}
		required = append(required, "property_ids")
		add("limit", "integer")
		props["filters"] = map[string]any{"type": "array", "maxItems": 10, "items": map[string]any{"type": "object", "properties": map[string]any{"property_id": map[string]any{"type": "string"}, "operator": map[string]any{"type": "string", "enum": []string{"eq", "contains", "gt", "gte", "lt", "lte"}}, "value": map[string]any{"type": []string{"string", "number", "boolean", "null"}}}, "required": []string{"property_id", "operator", "value"}, "additionalProperties": false}}
		required = append(required, "filters")
	case "create_document":
		description = "새 문서 작성을 제안합니다. 실제 사용자가 이 개별 내용을 확인하기 전에는 저장하지 않습니다."
		add("space_id", "string")
		add("title", "string")
		add("markdown", "string")
		props["visibility"] = map[string]any{"type": "string", "enum": []string{"private", "workspace"}}
		required = append(required, "visibility")
	case "update_document":
		description = "읽은 문서의 지정 버전을 수정하는 계획입니다. 현재 사용자 확인이 필요하고 동시 변경 시 실패합니다."
		add("document_id", "string")
		add("expected_version", "integer")
		add("title", "string")
		add("markdown", "string")
	}
	return map[string]any{"type": "function", "function": map[string]any{"name": name, "description": description, "parameters": map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}}}
}

// Streaming deltas are untrusted fragments. Only a complete, bounded assistant
// turn may result in tool execution; no partial JSON is ever executed.
func agentModelTurn(ctx context.Context, cfg map[string]any, a workspaceAgent, messages []agentMessage, delta func(string) error) (agentMessage, error) {
	out := agentMessage{Role: "assistant"}
	endpoint, e := aiEndpoint(str(cfg, "ai_base_url"))
	if e != nil {
		return out, e
	}
	tools := []map[string]any{}
	for _, name := range a.Tools {
		tools = append(tools, agentToolSchema(name))
	}
	maxTokens := min(a.MaxTokens, number(cfg, "ai_max_tokens", 4096))
	if maxTokens < 1 || maxTokens > 262144 || str(cfg, "ai_model") == "" || !boolean(cfg, "ai_enabled") {
		return out, errors.New("AI 공급자·모델·토큰 설정을 확인하세요")
	}
	payload := map[string]any{"model": str(cfg, "ai_model"), "messages": messages, "stream": true, "max_tokens": maxTokens, "tools": tools, "tool_choice": "auto", "parallel_tool_calls": false}
	raw := jsonValue(payload)
	if len(raw) > 4<<20 {
		return out, errors.New("Agent 대화 컨텍스트가 4MB 한도를 초과했습니다")
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if e != nil {
		return out, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if key := str(cfg, "ai_api_key"); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := integrationHTTPClient(10 * time.Minute)
	resp, e := client.Do(req)
	if e != nil {
		return out, errors.New("Agent AI 공급자에 연결하지 못했습니다")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("Agent AI 공급자 응답 오류 HTTP %d", resp.StatusCode)
	}
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		return out, errors.New("AI 공급자는 스트리밍 tool_calls SSE를 지원해야 합니다")
	}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 8<<20))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	calls := map[int]*agentToolCall{}
	finished, done := false, false
	bytesUsed := 0
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			done = true
			break
		}
		if data == "" {
			continue
		}
		var event struct {
			Error   json.RawMessage `json:"error"`
			Choices []struct {
				Index  int     `json:"index"`
				Finish *string `json:"finish_reason"`
				Delta  struct {
					Content   *string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(data), &event) != nil || len(event.Error) > 0 && string(event.Error) != "null" {
			return out, errors.New("AI 공급자의 스트림 형식이 올바르지 않습니다")
		}
		for _, choice := range event.Choices {
			if choice.Index != 0 {
				return out, errors.New("Agent는 단일 응답 선택지만 지원합니다")
			}
			if finished && choice.Delta.Content != nil {
				return out, errors.New("종료 뒤 AI 응답이 도착했습니다")
			}
			if choice.Delta.Content != nil {
				value := *choice.Delta.Content
				bytesUsed += len(value)
				if bytesUsed > 1<<20 {
					return out, errors.New("Agent 단계 출력은 최대 1MB입니다")
				}
				out.Content += value
				if value != "" && delta != nil {
					if e = delta(value); e != nil {
						return out, e
					}
				}
			}
			for _, part := range choice.Delta.ToolCalls {
				if finished || part.Index < 0 || part.Index >= 8 {
					return out, errors.New("Agent 도구 호출은 단계별 최대 8개입니다")
				}
				call := calls[part.Index]
				if call == nil {
					call = &agentToolCall{Type: "function"}
					calls[part.Index] = call
				}
				if part.Type != "" && part.Type != "function" {
					return out, errors.New("허용되지 않은 도구 유형입니다")
				}
				call.ID += part.ID
				call.Function.Name += part.Function.Name
				call.Function.Arguments += part.Function.Arguments
				if len(call.ID) > 200 || len(call.Function.Name) > 100 || len(call.Function.Arguments) > 65536 {
					return out, errors.New("Agent 도구 입력 한도를 초과했습니다")
				}
			}
			if choice.Finish != nil && *choice.Finish != "" {
				if !oneOf(*choice.Finish, "stop", "tool_calls") {
					return out, errors.New("AI 응답이 토큰 한도 또는 공급자 정책으로 중단되었습니다")
				}
				finished = true
			}
		}
	}
	if e = scanner.Err(); e != nil {
		return out, errors.New("AI 스트림을 읽는 중 연결이 종료되었습니다")
	}
	if !done || !finished {
		return out, errors.New("완료되지 않은 AI 응답입니다. 도구를 실행하지 않았습니다")
	}
	indices := []int{}
	for index := range calls {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	ids := map[string]bool{}
	for i, index := range indices {
		call := calls[index]
		if i != index || call.ID == "" || ids[call.ID] || !slicesContains(a.Tools, call.Function.Name) || !json.Valid([]byte(call.Function.Arguments)) {
			return out, errors.New("AI 도구 호출 ID·이름·JSON이 올바르지 않습니다")
		}
		ids[call.ID] = true
		out.ToolCalls = append(out.ToolCalls, *call)
	}
	return out, nil
}
func slicesContains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
