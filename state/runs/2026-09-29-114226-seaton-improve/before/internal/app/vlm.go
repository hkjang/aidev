package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// VLM 추론 결과는 보정되지 않은 값이므로 단독으로는 자동 승인선을 넘지 못하게 한다.
// CV와 교차 검증된 좌석만 자동 승인 구간으로 올라간다.
const (
	vlmConfidenceCap     = .94
	vlmDefaultConfidence = .85
	vlmMaxAttempts       = 3
	vlmTileOverlap       = .12
	vlmMaxRequestBytes   = 4 << 20
	// 좌석 하나가 도면 너비의 이 비율을 넘으면 회의실·구역으로 보고 버린다.
	vlmMaxSeatExtent = .35
	vlmMinSeatExtent = .004
)

type vlmConfig struct {
	endpoint string
	model    string
	apiKey   string
	timeout  time.Duration
	maxSide  int
	maxSeats int
	tiles    int
	jsonMode bool
}

// vlmError는 재시도 여부와 CV 폴백 판단에 쓰이는 분류된 오류다.
type vlmError struct {
	Kind    string
	Message string
	Err     error
}

func (e *vlmError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *vlmError) Unwrap() error { return e.Err }

func (e *vlmError) retryable() bool {
	switch e.Kind {
	case "network", "rate_limit", "server":
		return true
	}
	return false
}

func vlmFail(kind, message string, err error) *vlmError {
	return &vlmError{Kind: kind, Message: message, Err: err}
}

// vlmConfigFrom은 설정값을 읽고 검증한다. 오프라인 배포를 유지하기 위해 주소는
// 관리자가 명시한 사내 엔드포인트만 허용한다.
func (s *Server) vlmConfigFrom(ctx context.Context) (vlmConfig, error) {
	cfg := vlmConfig{timeout: 120 * time.Second, maxSide: 1600, maxSeats: 400, tiles: 1, jsonMode: true}
	base, _ := s.getSetting(ctx, "ai.vlm_base_url")
	base = strings.TrimSpace(base)
	if base == "" {
		return cfg, vlmFail("config", "VLM 엔드포인트 주소를 설정하세요", nil)
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return cfg, vlmFail("config", "VLM 주소는 http 또는 https 형식이어야 합니다", err)
	}
	if !strings.Contains(parsed.Path, "/chat/completions") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/chat/completions"
	}
	cfg.endpoint = parsed.String()
	if cfg.model, _ = s.getSetting(ctx, "ai.vlm_model"); strings.TrimSpace(cfg.model) == "" {
		return cfg, vlmFail("config", "VLM 모델 이름을 설정하세요", nil)
	}
	cfg.model = strings.TrimSpace(cfg.model)
	cfg.apiKey, _ = s.getSetting(ctx, "ai.vlm_api_key")
	if v, _ := s.getSetting(ctx, "ai.vlm_timeout_seconds"); v != "" {
		cfg.timeout = time.Duration(clampInt(int(parseFloat(v, 120)), 10, 600)) * time.Second
	}
	if v, _ := s.getSetting(ctx, "ai.vlm_max_image_side"); v != "" {
		cfg.maxSide = clampInt(int(parseFloat(v, 1600)), 512, 4096)
	}
	if v, _ := s.getSetting(ctx, "ai.vlm_max_seats"); v != "" {
		cfg.maxSeats = clampInt(int(parseFloat(v, 400)), 1, maxDetectedSeats)
	}
	if v, _ := s.getSetting(ctx, "ai.vlm_tiles"); v != "" {
		cfg.tiles = clampInt(int(parseFloat(v, 1)), 1, 4)
	}
	if v, _ := s.getSetting(ctx, "ai.vlm_json_mode"); v != "" {
		cfg.jsonMode = v == "true"
	}
	return cfg, nil
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

type vlmResult struct {
	Objects    []detectedObject
	Warnings   []string
	Convention string
	Tiles      int
	RawSeats   int
}

// detectWithVLM은 도면을 비전 모델에 보내 좌석 경계 상자를 받는다. 도면이 크면
// 타일로 나눠 보내 작은 책상이 뭉개지는 것을 막고, 결과는 전역 좌표로 합친다.
func (s *Server) detectWithVLM(ctx context.Context, img image.Image, cfg vlmConfig) (vlmResult, error) {
	result := vlmResult{Tiles: cfg.tiles * cfg.tiles}
	bounds := img.Bounds()
	if bounds.Dx() < 8 || bounds.Dy() < 8 {
		return result, vlmFail("config", "도면 이미지가 너무 작습니다", nil)
	}
	type tileJob struct{ rect image.Rectangle }
	jobs := []tileJob{}
	if cfg.tiles <= 1 {
		jobs = append(jobs, tileJob{bounds})
	} else {
		stepX := float64(bounds.Dx()) / float64(cfg.tiles)
		stepY := float64(bounds.Dy()) / float64(cfg.tiles)
		padX, padY := stepX*vlmTileOverlap, stepY*vlmTileOverlap
		for row := 0; row < cfg.tiles; row++ {
			for col := 0; col < cfg.tiles; col++ {
				rect := image.Rect(
					bounds.Min.X+int(math.Max(0, float64(col)*stepX-padX)),
					bounds.Min.Y+int(math.Max(0, float64(row)*stepY-padY)),
					bounds.Min.X+int(math.Min(float64(bounds.Dx()), float64(col+1)*stepX+padX)),
					bounds.Min.Y+int(math.Min(float64(bounds.Dy()), float64(row+1)*stepY+padY)),
				)
				if rect.Dx() > 8 && rect.Dy() > 8 {
					jobs = append(jobs, tileJob{rect})
				}
			}
		}
	}
	collected := []detectedObject{}
	failures := 0
	var lastErr error
	for index, job := range jobs {
		tile := cropImage(img, job.rect)
		parsed, err := s.requestVLMBoxes(ctx, tile, cfg)
		if err != nil {
			failures++
			lastErr = err
			s.logger.Warn("VLM 타일 분석 실패", "tile", index, "error", err)
			continue
		}
		if parsed.Convention != "" {
			result.Convention = parsed.Convention
		}
		result.RawSeats += parsed.Raw
		result.Warnings = append(result.Warnings, parsed.Warnings...)
		boxes := parsed.Objects
		// 타일 내부 비율을 도면 전체 비율로 옮긴다.
		offsetX := float64(job.rect.Min.X-bounds.Min.X) / float64(bounds.Dx())
		offsetY := float64(job.rect.Min.Y-bounds.Min.Y) / float64(bounds.Dy())
		scaleX := float64(job.rect.Dx()) / float64(bounds.Dx())
		scaleY := float64(job.rect.Dy()) / float64(bounds.Dy())
		for _, box := range boxes {
			box.X = offsetX + box.X*scaleX
			box.Y = offsetY + box.Y*scaleY
			box.W *= scaleX
			box.H *= scaleY
			collected = append(collected, box)
		}
	}
	if failures == len(jobs) {
		if lastErr == nil {
			lastErr = vlmFail("protocol", "VLM 응답을 해석하지 못했습니다", nil)
		}
		return result, lastErr
	}
	if failures > 0 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("타일 %d개 중 %d개 분석에 실패해 해당 영역 좌석이 빠졌을 수 있습니다", len(jobs), failures))
	}
	merged := suppressOverlaps(collected, .55)
	if len(merged) > cfg.maxSeats {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("VLM이 좌석 %d개를 보고해 상한 %d개까지만 사용했습니다", len(merged), cfg.maxSeats))
		merged = merged[:cfg.maxSeats]
	}
	if len(merged) == 0 {
		return result, vlmFail("empty", "VLM이 좌석을 하나도 찾지 못했습니다", nil)
	}
	result.Objects = merged
	return result, nil
}

// requestVLMBoxes는 한 장의 이미지에 대해 모델을 호출하고 검증된 상자를 돌려준다.
func (s *Server) requestVLMBoxes(ctx context.Context, img image.Image, cfg vlmConfig) (vlmParse, error) {
	scaled := downscaleImage(img, cfg.maxSide)
	dataURI, err := encodeImageDataURI(scaled)
	if err != nil {
		return vlmParse{}, vlmFail("config", "도면 이미지를 인코딩하지 못했습니다", err)
	}
	width, height := scaled.Bounds().Dx(), scaled.Bounds().Dy()
	prompt := vlmPrompt()
	useResponseFormat := cfg.jsonMode
	var lastErr *vlmError
	repaired := false
	for attempt := 0; attempt < vlmMaxAttempts; attempt++ {
		if attempt > 0 {
			// 지수 백오프에 지터를 섞어 동시 재시도가 겹치지 않게 한다.
			delay := time.Duration(800*int64(1<<uint(attempt-1)))*time.Millisecond +
				time.Duration(int64(attempt)*137)*time.Millisecond
			select {
			case <-ctx.Done():
				return vlmParse{}, vlmFail("network", "VLM 호출이 취소되었습니다", ctx.Err())
			case <-time.After(delay):
			}
		}
		instruction := prompt
		if repaired {
			instruction = prompt + "\n\nYour previous reply was not valid JSON. Reply with the JSON object ONLY."
		}
		content, finish, callErr := s.callVLM(ctx, cfg, instruction, dataURI, useResponseFormat)
		if callErr != nil {
			lastErr = callErr
			if callErr.Kind == "unsupported_response_format" && useResponseFormat {
				useResponseFormat = false
				attempt--
				continue
			}
			if !callErr.retryable() {
				return vlmParse{}, callErr
			}
			continue
		}
		parsed, parseErr := parseVLMSeats(content, width, height)
		if parseErr != nil {
			lastErr = parseErr
			if !repaired {
				repaired = true
				continue
			}
			return vlmParse{}, parseErr
		}
		if finish == "length" {
			if len(parsed.Objects) == 0 {
				// 잘린 응답에서 상자를 하나도 못 건졌다면 재시도할 여지가 있다.
				lastErr = vlmFail("protocol", "VLM 응답이 최대 길이에서 잘렸습니다. 좌석 상한을 낮추거나 타일 수를 늘리세요", nil)
				continue
			}
			parsed.Warnings = append(parsed.Warnings,
				"VLM 응답이 최대 길이에서 잘려 일부 좌석이 빠졌을 수 있습니다")
		}
		return parsed, nil
	}
	if lastErr == nil {
		lastErr = vlmFail("network", "VLM 호출이 반복 실패했습니다", nil)
	}
	return vlmParse{}, lastErr
}

// vlmPrompt는 Qwen2.5-VL 7B로 실측해 고른 형식이다. 같은 도면(책상 30개)에서
// 프롬프트별 F1을 5회씩 측정한 결과:
//
//	JSON 스키마에 예시 좌표를 넣어 요청            0.00  (모델이 예시 숫자를 그대로 복사)
//	스키마만 서술하고 정규화 좌표 요청              0.00  (균일 격자를 발명)
//	네이티브 grounding · 배열 반환                0.30
//	네이티브 grounding · 객체 래퍼                0.37~0.41  ← 채택
//	위에 "최대 N개" 상한 문구 추가                 0.22  (상한을 넣으면 오히려 나빠짐)
//
// 좌석 수 상한은 프롬프트가 아니라 서버에서 잘라낸다. 좌표계는 모델이 픽셀로
// 답하든 정규화로 답하든 파서가 판별한다.
func vlmPrompt() string {
	return `Locate every individual desk (workstation) drawn in this office floor plan.
Ignore walls, doors, corridors, meeting rooms, title blocks, legends, dimension lines and text labels.
Output JSON only: an object with a single key "objects" whose value is a list where each element has "bbox_2d" and "label".`
}

type vlmChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (s *Server) callVLM(ctx context.Context, cfg vlmConfig, prompt, dataURI string, useResponseFormat bool) (string, string, *vlmError) {
	maxTokens := clampInt(1024+cfg.maxSeats*40, 1024, 16384)
	body := map[string]any{
		"model":       cfg.model,
		"temperature": 0,
		"max_tokens":  maxTokens,
		"messages": []any{
			map[string]any{"role": "system", "content": "You are a precise floor-plan object detector. You always answer with a single JSON object."},
			map[string]any{"role": "user", "content": []any{
				map[string]any{"type": "text", "text": prompt},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": dataURI}},
			}},
		},
	}
	if useResponseFormat {
		body["response_format"] = map[string]any{"type": "json_object"}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", "", vlmFail("config", "VLM 요청을 만들지 못했습니다", err)
	}
	requestCtx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, cfg.endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", "", vlmFail("config", "VLM 주소가 올바르지 않습니다", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if cfg.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
	}
	response, err := (&http.Client{Timeout: cfg.timeout}).Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
			return "", "", vlmFail("network", fmt.Sprintf("VLM 응답이 %.0f초 안에 오지 않았습니다", cfg.timeout.Seconds()), err)
		}
		return "", "", vlmFail("network", "VLM 서버에 연결할 수 없습니다", err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return "", "", vlmFail("network", "VLM 응답을 읽지 못했습니다", err)
	}
	switch {
	case response.StatusCode == http.StatusUnauthorized, response.StatusCode == http.StatusForbidden:
		return "", "", vlmFail("auth", "VLM 인증에 실패했습니다. API 키를 확인하세요", nil)
	case response.StatusCode == http.StatusNotFound:
		return "", "", vlmFail("config", "VLM 엔드포인트 경로를 찾을 수 없습니다. 주소가 /v1 까지 포함되었는지 확인하세요", nil)
	case response.StatusCode == http.StatusTooManyRequests:
		return "", "", vlmFail("rate_limit", "VLM 서버가 요청 한도를 초과했습니다", nil)
	case response.StatusCode == http.StatusBadRequest:
		detail := vlmErrorDetail(payload)
		if useResponseFormat && mentionsResponseFormat(detail) {
			return "", "", vlmFail("unsupported_response_format", detail, nil)
		}
		return "", "", vlmFail("protocol", "VLM이 요청을 거부했습니다: "+detail, nil)
	case response.StatusCode >= 500:
		return "", "", vlmFail("server", fmt.Sprintf("VLM 서버 오류 %d", response.StatusCode), nil)
	case response.StatusCode < 200 || response.StatusCode >= 300:
		return "", "", vlmFail("protocol", fmt.Sprintf("VLM 응답 코드 %d", response.StatusCode), nil)
	}
	var parsed vlmChatResponse
	if err = json.Unmarshal(payload, &parsed); err != nil {
		return "", "", vlmFail("protocol", "VLM 응답이 JSON 형식이 아닙니다", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", "", vlmFail("protocol", "VLM 오류: "+parsed.Error.Message, nil)
	}
	if len(parsed.Choices) == 0 {
		return "", "", vlmFail("protocol", "VLM이 빈 응답을 돌려주었습니다", nil)
	}
	return parsed.Choices[0].Message.Content, parsed.Choices[0].FinishReason, nil
}

func vlmErrorDetail(payload []byte) string {
	var parsed vlmChatResponse
	if json.Unmarshal(payload, &parsed) == nil && parsed.Error != nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	detail := strings.TrimSpace(string(payload))
	if len(detail) > 300 {
		detail = detail[:300]
	}
	if detail == "" {
		detail = "상세 정보 없음"
	}
	return detail
}

func mentionsResponseFormat(detail string) bool {
	lower := strings.ToLower(detail)
	for _, needle := range []string{"response_format", "guided", "json_object", "json schema", "structured"} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

type vlmSeatRaw struct {
	X          *float64  `json:"x"`
	Y          *float64  `json:"y"`
	W          *float64  `json:"w"`
	H          *float64  `json:"h"`
	Width      *float64  `json:"width"`
	Height     *float64  `json:"height"`
	Bbox       []float64 `json:"bbox"`
	Bbox2D     []float64 `json:"bbox_2d"`
	Box        []float64 `json:"box"`
	Confidence *float64  `json:"confidence"`
	Score      *float64  `json:"score"`
}

type vlmPayload struct {
	Seats   []vlmSeatRaw `json:"seats"`
	Objects []vlmSeatRaw `json:"objects"`
	Items   []vlmSeatRaw `json:"items"`
	Boxes   []vlmSeatRaw `json:"boxes"`
}

// extractJSON은 모델이 붙인 코드펜스나 앞뒤 설명을 걷어내고 JSON 본문만 남긴다.
func extractJSON(content string) string {
	trimmed := strings.TrimSpace(content)
	if fence := strings.Index(trimmed, "```"); fence >= 0 {
		rest := trimmed[fence+3:]
		if newline := strings.IndexByte(rest, '\n'); newline >= 0 {
			rest = rest[newline+1:]
		}
		if end := strings.Index(rest, "```"); end >= 0 {
			rest = rest[:end]
		}
		trimmed = strings.TrimSpace(rest)
	}
	start := strings.IndexAny(trimmed, "{[")
	if start < 0 {
		return trimmed
	}
	open := trimmed[start]
	close := byte('}')
	if open == '[' {
		close = ']'
	}
	depth, inString, escaped := 0, false, false
	for i := start; i < len(trimmed); i++ {
		c := trimmed[i]
		switch {
		case escaped:
			escaped = false
		case c == '\\' && inString:
			escaped = true
		case c == '"':
			inString = !inString
		case inString:
		case c == open:
			depth++
		case c == close:
			depth--
			if depth == 0 {
				return trimmed[start : i+1]
			}
		}
	}
	return trimmed[start:]
}

type boxQuad struct {
	a, b, c, d float64
	confidence float64
	explicitWH bool
}

func collectQuads(seats []vlmSeatRaw) []boxQuad {
	quads := []boxQuad{}
	for _, seat := range seats {
		confidence := vlmDefaultConfidence
		if seat.Confidence != nil {
			confidence = *seat.Confidence
		} else if seat.Score != nil {
			confidence = *seat.Score
		}
		width, height := seat.W, seat.Width
		if width == nil {
			width = height
		}
		heightValue := seat.H
		if heightValue == nil {
			heightValue = seat.Height
		}
		if seat.X != nil && seat.Y != nil && width != nil && heightValue != nil {
			quads = append(quads, boxQuad{*seat.X, *seat.Y, *width, *heightValue, confidence, true})
			continue
		}
		for _, candidate := range [][]float64{seat.Bbox2D, seat.Bbox, seat.Box} {
			if len(candidate) == 4 {
				quads = append(quads, boxQuad{candidate[0], candidate[1], candidate[2], candidate[3], confidence, false})
				break
			}
		}
	}
	return quads
}

// 상자 대부분이 버려지면 좌표 해석 자체가 틀렸을 가능성이 높다. 아래 비율을
// 밑돌면 해석 실패로 보고 재시도하고, 그 위에서도 낮으면 경고만 남긴다.
const (
	vlmMinValidFraction   = .15
	vlmCleanValidFraction = .6
)

type vlmParse struct {
	Objects    []detectedObject
	Convention string
	Raw        int
	Warnings   []string
}

type coordinateReading struct {
	scaleName     string
	arrayCorners  bool
	objects       []detectedObject
	validFraction float64
}

// parseVLMSeats는 응답 본문에서 좌석 상자를 뽑아낸다. 모델이 정규화 좌표를
// 지킨다는 보장이 없어(픽셀 좌표나 0~1000 좌표로 답하는 모델이 있다) 후보
// 해석을 모두 적용해 살아남는 상자가 가장 많은 쪽을 고른다. 배열 필드는
// Qwen 계열 관례대로 [x1,y1,x2,y2]를 우선 시도한다.
func parseVLMSeats(content string, sentWidth, sentHeight int) (vlmParse, *vlmError) {
	body := extractJSON(content)
	if body == "" {
		return vlmParse{}, vlmFail("protocol", "VLM 응답에 JSON이 없습니다", nil)
	}
	var seats []vlmSeatRaw
	var payload vlmPayload
	if err := json.Unmarshal([]byte(body), &payload); err == nil {
		for _, group := range [][]vlmSeatRaw{payload.Seats, payload.Objects, payload.Items, payload.Boxes} {
			if len(group) > 0 {
				seats = group
				break
			}
		}
		if seats == nil && strings.Contains(body, "\"seats\"") {
			// {"seats": []} 는 정상적인 "좌석 없음" 응답이다.
			return vlmParse{Convention: "normalized"}, nil
		}
	}
	if seats == nil {
		var bare []vlmSeatRaw
		if err := json.Unmarshal([]byte(body), &bare); err != nil {
			return vlmParse{}, vlmFail("protocol", "VLM 응답 구조를 해석하지 못했습니다", err)
		}
		seats = bare
	}
	quads := collectQuads(seats)
	if len(quads) == 0 {
		return vlmParse{Convention: "normalized"}, nil
	}
	maximum, arrayQuads := 0.0, 0
	for _, quad := range quads {
		if !quad.explicitWH {
			arrayQuads++
		}
		for _, value := range []float64{quad.a, quad.b, quad.c, quad.d} {
			maximum = math.Max(maximum, math.Abs(value))
		}
	}
	type coordinateScale struct {
		name       string
		divX, divY float64
	}
	scales := []coordinateScale{{"normalized", 1, 1}}
	if maximum > 1.5 {
		scales = append(scales,
			coordinateScale{"pixel", float64(sentWidth), float64(sentHeight)},
			coordinateScale{"per-mille", 1000, 1000})
	}
	best := coordinateReading{}
	for _, scale := range scales {
		if scale.divX <= 0 || scale.divY <= 0 {
			continue
		}
		// corners를 먼저 평가해 동점이면 관례에 맞는 해석이 이기게 한다.
		for _, corners := range []bool{true, false} {
			reading := decodeReading(quads, scale.name, scale.divX, scale.divY, corners)
			if reading.validFraction > best.validFraction {
				best = reading
			}
		}
	}
	if len(best.objects) == 0 || best.validFraction < vlmMinValidFraction {
		return vlmParse{Raw: len(quads)}, vlmFail("protocol",
			fmt.Sprintf("VLM이 돌려준 좌표를 해석하지 못했습니다 (상자 %d개, 최대값 %.1f)", len(quads), maximum), nil)
	}
	form := "xywh"
	if arrayQuads > 0 && best.arrayCorners {
		form = "corners"
	}
	result := vlmParse{
		Objects:    suppressOverlaps(best.objects, .55),
		Convention: best.scaleName + "/" + form,
		Raw:        len(quads),
	}
	if best.validFraction < vlmCleanValidFraction {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"VLM이 보고한 상자 %d개 중 %d개만 좌석으로 인정됐습니다. 프롬프트나 모델을 점검하세요",
			len(quads), len(best.objects)))
	}
	return result, nil
}

// decodeReading은 하나의 좌표 해석을 전체 상자에 적용하고 살아남은 비율로 점수를
// 낸다. x/y/w/h 키가 명시된 상자는 형식이 확정이므로 배열 해석과 무관하게 다룬다.
func decodeReading(quads []boxQuad, scaleName string, divX, divY float64, arrayCorners bool) coordinateReading {
	reading := coordinateReading{scaleName: scaleName, arrayCorners: arrayCorners}
	for _, quad := range quads {
		x, y := quad.a/divX, quad.b/divY
		var w, h float64
		if arrayCorners && !quad.explicitWH {
			x2, y2 := quad.c/divX, quad.d/divY
			if x2 < x {
				x, x2 = x2, x
			}
			if y2 < y {
				y, y2 = y2, y
			}
			w, h = x2-x, y2-y
		} else {
			w, h = quad.c/divX, quad.d/divY
		}
		if object, ok := validateVLMBox(x, y, w, h, quad.confidence); ok {
			reading.objects = append(reading.objects, object)
		}
	}
	if len(quads) > 0 {
		reading.validFraction = float64(len(reading.objects)) / float64(len(quads))
	}
	return reading
}

// validateVLMBox는 상자를 도면 범위 안으로 정리하고 좌석으로 볼 수 없는 크기를 버린다.
func validateVLMBox(x, y, w, h, confidence float64) (detectedObject, bool) {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(w) || math.IsNaN(h) {
		return detectedObject{}, false
	}
	if w <= 0 || h <= 0 || x < -.02 || y < -.02 || x > 1.02 || y > 1.02 {
		return detectedObject{}, false
	}
	x, y = math.Max(0, x), math.Max(0, y)
	w, h = math.Min(w, 1-x), math.Min(h, 1-y)
	if w < vlmMinSeatExtent || h < vlmMinSeatExtent {
		return detectedObject{}, false
	}
	if w > vlmMaxSeatExtent || h > vlmMaxSeatExtent {
		return detectedObject{}, false
	}
	if ratio := w / h; ratio > 6 || ratio < 1.0/6 {
		return detectedObject{}, false
	}
	if math.IsNaN(confidence) || confidence <= 0 || confidence > 1 {
		confidence = vlmDefaultConfidence
	}
	return detectedObject{X: x, Y: y, W: w, H: h, Confidence: math.Min(vlmConfidenceCap, confidence), Source: "vlm"}, true
}

func intersectionOverUnion(a, b detectedObject) float64 {
	left := math.Max(a.X, b.X)
	top := math.Max(a.Y, b.Y)
	right := math.Min(a.X+a.W, b.X+b.W)
	bottom := math.Min(a.Y+a.H, b.Y+b.H)
	if right <= left || bottom <= top {
		return 0
	}
	overlap := (right - left) * (bottom - top)
	union := a.W*a.H + b.W*b.H - overlap
	if union <= 0 {
		return 0
	}
	return overlap / union
}

// suppressOverlaps는 같은 좌석을 여러 번 보고하는 중복을 신뢰도 우선으로 제거한다.
func suppressOverlaps(objects []detectedObject, threshold float64) []detectedObject {
	sorted := append([]detectedObject(nil), objects...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Confidence != sorted[j].Confidence {
			return sorted[i].Confidence > sorted[j].Confidence
		}
		if sorted[i].Y != sorted[j].Y {
			return sorted[i].Y < sorted[j].Y
		}
		return sorted[i].X < sorted[j].X
	})
	kept := []detectedObject{}
	for _, candidate := range sorted {
		duplicate := false
		for _, existing := range kept {
			if intersectionOverUnion(candidate, existing) > threshold {
				duplicate = true
				break
			}
		}
		if !duplicate {
			kept = append(kept, candidate)
		}
	}
	sort.Slice(kept, func(i, j int) bool {
		if math.Abs(kept[i].Y-kept[j].Y) > .02 {
			return kept[i].Y < kept[j].Y
		}
		return kept[i].X < kept[j].X
	})
	return kept
}

func cropImage(src image.Image, rect image.Rectangle) image.Image {
	if sub, ok := src.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return sub.SubImage(rect)
	}
	out := image.NewNRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			out.Set(x, y, src.At(rect.Min.X+x, rect.Min.Y+y))
		}
	}
	return out
}

// downscaleImage는 박스 평균으로 축소한다. 도면의 얇은 선이 최근접 이웃 축소에서
// 사라지는 것을 막기 위해 평균을 쓴다.
func downscaleImage(src image.Image, maxSide int) *image.NRGBA {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	scale := 1.0
	if longest := math.Max(float64(width), float64(height)); longest > float64(maxSide) {
		scale = float64(maxSide) / longest
	}
	outWidth := int(math.Max(1, math.Round(float64(width)*scale)))
	outHeight := int(math.Max(1, math.Round(float64(height)*scale)))
	out := image.NewNRGBA(image.Rect(0, 0, outWidth, outHeight))
	stepX := float64(width) / float64(outWidth)
	stepY := float64(height) / float64(outHeight)
	for y := 0; y < outHeight; y++ {
		y0 := bounds.Min.Y + int(float64(y)*stepY)
		y1 := bounds.Min.Y + int(math.Ceil(float64(y+1)*stepY))
		if y1 <= y0 {
			y1 = y0 + 1
		}
		if y1 > bounds.Max.Y {
			y1 = bounds.Max.Y
		}
		for x := 0; x < outWidth; x++ {
			x0 := bounds.Min.X + int(float64(x)*stepX)
			x1 := bounds.Min.X + int(math.Ceil(float64(x+1)*stepX))
			if x1 <= x0 {
				x1 = x0 + 1
			}
			if x1 > bounds.Max.X {
				x1 = bounds.Max.X
			}
			var sumR, sumG, sumB uint64
			count := uint64(0)
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					r, g, b, _ := src.At(sx, sy).RGBA()
					sumR += uint64(r >> 8)
					sumG += uint64(g >> 8)
					sumB += uint64(b >> 8)
					count++
				}
			}
			if count == 0 {
				count = 1
			}
			out.SetNRGBA(x, y, color.NRGBA{
				R: uint8(sumR / count), G: uint8(sumG / count), B: uint8(sumB / count), A: 255,
			})
		}
	}
	return out
}

// encodeImageDataURI는 선 도면에 유리한 PNG를 우선 쓰고, 지나치게 커지면
// JPEG로 낮춰 요청 크기를 억제한다.
func encodeImageDataURI(img image.Image) (string, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return "", err
	}
	mime := "image/png"
	if buffer.Len() > vlmMaxRequestBytes {
		var fallback bytes.Buffer
		if err := jpeg.Encode(&fallback, img, &jpeg.Options{Quality: 85}); err == nil && fallback.Len() < buffer.Len() {
			buffer, mime = fallback, "image/jpeg"
		}
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}
