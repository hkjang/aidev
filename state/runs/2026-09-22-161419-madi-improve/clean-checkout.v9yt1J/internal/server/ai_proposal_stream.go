package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// A structured proposal is untrusted output, not a tool invocation. The caller
// validates its closed schema and only then sends a reviewable result. No model
// output reaches a write handler here. Quiet streams are also revoked promptly.
func (s *Server) streamAIProposal(w http.ResponseWriter, r *http.Request, wid, system, input string, sources []aiSource, guard func() error, finish func(string) (any, error), validatedOutputOnly ...bool) {
	p := current(r)
	cfg, err := s.effectiveSettings(r.Context(), wid)
	if err != nil {
		respond(w, nil, err)
		return
	}
	endpoint, err := aiEndpoint(str(cfg, "ai_base_url"))
	tokens := number(cfg, "ai_max_tokens", 4096)
	if err != nil || !boolean(cfg, "ai_enabled") || str(cfg, "ai_model") == "" || tokens < 1 || tokens > 262144 {
		apiError(w, 503, "관리자가 AI 공급자와 모델을 활성화해야 합니다")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	check := func() error {
		if err := s.validateAIStream(r.WithContext(ctx), p, wid, sources, cfg); err != nil {
			return err
		}
		if guard != nil {
			return guard()
		}
		return nil
	}
	if err = check(); err != nil {
		apiError(w, 403, err.Error())
		return
	}
	revoked := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if e := check(); e != nil {
					select {
					case revoked <- e:
					default:
					}
					cancel()
					return
				}
			}
		}
	}()
	payload := map[string]any{"model": str(cfg, "ai_model"), "stream": true, "max_tokens": tokens, "messages": []map[string]string{{"role": "system", "content": system}, {"role": "user", "content": input}}}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonValue(payload)))
	if err != nil {
		apiError(w, 503, "AI 요청을 구성하지 못했습니다")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if secret := str(cfg, "ai_api_key"); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
		req.Header.Set("api-key", secret)
	}
	client := integrationHTTPClient(0)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 45 * time.Second
	client.Transport = transport
	defer transport.CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		select {
		case e := <-revoked:
			apiError(w, 403, e.Error())
		default:
			apiError(w, 502, "AI 공급자 연결이 중단되었습니다")
		}
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		apiError(w, 502, fmt.Sprintf("AI 공급자가 HTTP %d 오류를 반환했습니다", response.StatusCode))
		return
	}
	if !strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream") {
		apiError(w, 502, "AI 공급자는 스트리밍 응답을 반환해야 합니다")
		return
	}
	if err = check(); err != nil {
		apiError(w, 403, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)
	controller := http.NewResponseController(w)
	send := func(value any) error {
		if _, e := fmt.Fprintf(w, "data: %s\n\n", jsonValue(value)); e != nil {
			return e
		}
		return controller.Flush()
	}
	if send(map[string]any{"phase": "generating", "automatic_apply": false}) != nil {
		return
	}
	var output strings.Builder
	err = streamAIResponse(ctx, response.Body, func(delta string) error {
		if output.Len()+len(delta) > 65536 {
			return errors.New("AI 제안은 64KiB 이하여야 합니다")
		}
		if e := check(); e != nil {
			return e
		}
		output.WriteString(delta)
		// Structured extraction may create new sensitive values spanning several
		// deltas. Keep provider streaming/cancellation, but do not disclose its
		// raw JSON before the caller validates the complete schema and PII policy.
		if len(validatedOutputOnly) > 0 && validatedOutputOnly[0] {
			return send(map[string]any{"phase": "generating", "automatic_apply": false})
		}
		return send(map[string]any{"text": delta})
	})
	select {
	case e := <-revoked:
		err = e
	default:
	}
	if err == nil {
		err = check()
	}
	if err == nil {
		var result any
		result, err = finish(output.String())
		if err == nil {
			err = check()
		}
		if err == nil {
			err = send(map[string]any{"proposal": result, "automatic_apply": false})
		}
	}
	if err != nil && r.Context().Err() == nil {
		message := "AI 제안이 완료되지 않았거나 검증에 실패했습니다. 조건을 확인하고 다시 요청하세요"
		if errors.Is(err, errAIStreamChanged) {
			message = errAIStreamChanged.Error()
		}
		_ = send(map[string]any{"error": message, "retract": true})
	}
	if r.Context().Err() == nil {
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		_ = controller.Flush()
	}
	s.audit(r, "AI_PROPOSAL", wid, map[string]any{"success": err == nil, "source_count": len(sources), "max_tokens": tokens})
}
