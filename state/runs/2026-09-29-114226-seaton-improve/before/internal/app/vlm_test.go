package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// asVLMError는 분류된 VLM 오류를 꺼낸다.
func asVLMError(err error, target **vlmError) bool { return errors.As(err, target) }

func testServer() *Server {
	return &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func testVLMConfig(endpoint string) vlmConfig {
	return vlmConfig{
		endpoint: endpoint, model: "qwen2.5-vl-7b-instruct",
		timeout: 3 * time.Second, maxSide: 512, maxSeats: 32, tiles: 1, jsonMode: true,
	}
}

// chatResponse는 OpenAI 호환 서버가 돌려주는 형태로 본문을 감싼다.
func chatResponse(content, finish string) string {
	body, _ := json.Marshal(map[string]any{
		"choices": []any{map[string]any{
			"message": map[string]any{"content": content}, "finish_reason": finish,
		}},
	})
	return string(body)
}

func TestExtractJSONStripsFencesAndProse(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"순수 JSON", `{"seats":[]}`, `{"seats":[]}`},
		{"코드펜스", "```json\n{\"seats\":[]}\n```", `{"seats":[]}`},
		{"언어 없는 펜스", "```\n{\"seats\":[]}\n```", `{"seats":[]}`},
		{"앞뒤 설명", "Here you go:\n{\"seats\":[]}\nHope this helps!", `{"seats":[]}`},
		{"배열 최상위", `[{"x":0.1}]`, `[{"x":0.1}]`},
		{"중첩 객체", `{"a":{"b":1},"seats":[]}`, `{"a":{"b":1},"seats":[]}`},
		{"문자열 안의 중괄호", `{"note":"}{","seats":[]}`, `{"note":"}{","seats":[]}`},
	}
	for _, c := range cases {
		if got := extractJSON(c.input); got != c.want {
			t.Errorf("%s: %q → %q, 기대 %q", c.name, c.input, got, c.want)
		}
	}
}

func TestParseVLMSeatsCoordinateConventions(t *testing.T) {
	// 같은 좌석 두 개를 정규화·픽셀·0~1000 좌표로 각각 표현한다. per-mille 사례는
	// 픽셀 해석이 성립할 수 없도록(좌표 > 이미지 크기) 작은 이미지를 기준으로 준다.
	cases := []struct {
		name, body          string
		sentWidth, sentHigh int
		want                string
	}{
		{
			"정규화 xywh",
			`{"seats":[{"x":0.1,"y":0.2,"w":0.05,"h":0.06},{"x":0.5,"y":0.2,"w":0.05,"h":0.06}]}`,
			1000, 600, "normalized/xywh",
		},
		{
			"픽셀 corners (bbox_2d)",
			`{"seats":[{"bbox_2d":[100,120,150,156]},{"bbox_2d":[500,120,550,156]}]}`,
			1000, 600, "pixel/corners",
		},
		{
			"0~1000 corners",
			`{"seats":[{"bbox":[500,600,550,660]},{"bbox":[700,600,750,660]}]}`,
			400, 300, "per-mille/corners",
		},
	}
	for _, c := range cases {
		parsed, err := parseVLMSeats(c.body, c.sentWidth, c.sentHigh)
		if err != nil {
			t.Fatalf("%s: 예상치 못한 오류 %v", c.name, err)
		}
		if parsed.Raw != 2 || len(parsed.Objects) != 2 {
			t.Fatalf("%s: 상자 2개를 기대했으나 raw=%d parsed=%d", c.name, parsed.Raw, len(parsed.Objects))
		}
		if parsed.Convention != c.want {
			t.Errorf("%s: 좌표계 %q, 기대 %q", c.name, parsed.Convention, c.want)
		}
		for _, object := range parsed.Objects {
			if object.X < 0 || object.Y < 0 || object.X+object.W > 1.0001 || object.Y+object.H > 1.0001 {
				t.Errorf("%s: 정규화 범위를 벗어남 %#v", c.name, object)
			}
			if object.Source != "vlm" {
				t.Errorf("%s: 출처가 vlm이 아니다: %q", c.name, object.Source)
			}
			if object.Confidence > vlmConfidenceCap {
				t.Errorf("%s: VLM 단독 신뢰도 상한을 넘었다: %.3f", c.name, object.Confidence)
			}
		}
	}
}

func TestParseVLMSeatsNormalizesPixelCornersToSamePlace(t *testing.T) {
	// 픽셀 corners와 정규화 xywh가 같은 좌석을 가리키면 결과도 같아야 한다.
	pixelParse, err := parseVLMSeats(`{"seats":[{"bbox_2d":[200,150,300,240]}]}`, 1000, 600)
	if err != nil {
		t.Fatalf("픽셀 해석 실패: %v", err)
	}
	normalizedParse, err := parseVLMSeats(`{"seats":[{"x":0.2,"y":0.25,"w":0.1,"h":0.15}]}`, 1000, 600)
	if err != nil {
		t.Fatalf("정규화 해석 실패: %v", err)
	}
	pixel, normalized := pixelParse.Objects, normalizedParse.Objects
	if len(pixel) != 1 || len(normalized) != 1 {
		t.Fatalf("각각 1개를 기대했으나 %d / %d", len(pixel), len(normalized))
	}
	for _, pair := range [][2]float64{
		{pixel[0].X, normalized[0].X}, {pixel[0].Y, normalized[0].Y},
		{pixel[0].W, normalized[0].W}, {pixel[0].H, normalized[0].H},
	} {
		if math.Abs(pair[0]-pair[1]) > 1e-9 {
			t.Fatalf("좌표가 어긋난다: %v", pair)
		}
	}
}

func TestParseVLMSeatsRejectsAndRepairsBadBoxes(t *testing.T) {
	body := `{"seats":[
		{"x":0.1,"y":0.1,"w":0.05,"h":0.05},
		{"x":0.2,"y":0.2,"w":0,"h":0.05},
		{"x":0.3,"y":0.3,"w":0.9,"h":0.05},
		{"x":0.4,"y":0.4,"w":0.0001,"h":0.0001},
		{"x":0.5,"y":0.5,"w":0.30,"h":0.02},
		{"x":1.9,"y":0.6,"w":0.05,"h":0.05}
	]}`
	parsed, err := parseVLMSeats(body, 1000, 600)
	if err != nil {
		t.Fatalf("예상치 못한 오류: %v", err)
	}
	objects, raw := parsed.Objects, parsed.Raw
	if len(parsed.Warnings) == 0 {
		t.Fatal("대부분의 상자를 버렸는데 경고가 없다")
	}
	if raw != 6 {
		t.Fatalf("원본 상자 6개를 기대했으나 %d", raw)
	}
	// 유효한 것은 첫 번째 하나뿐이다: 0 너비·과대·과소·비정상 종횡비·범위 초과는 버려진다.
	if len(objects) != 1 {
		t.Fatalf("유효 상자 1개를 기대했으나 %d개: %#v", len(objects), objects)
	}
	if math.Abs(objects[0].X-.1) > 1e-9 {
		t.Fatalf("살아남은 상자가 기대와 다르다: %#v", objects[0])
	}
}

func TestParseVLMSeatsSwapsInvertedCorners(t *testing.T) {
	parsed, err := parseVLMSeats(`{"seats":[{"bbox_2d":[300,240,200,150]}]}`, 1000, 600)
	if err != nil {
		t.Fatalf("예상치 못한 오류: %v", err)
	}
	objects := parsed.Objects
	if len(objects) != 1 {
		t.Fatalf("상자 1개를 기대했으나 %d", len(objects))
	}
	if math.Abs(objects[0].X-.2) > 1e-9 || math.Abs(objects[0].Y-.25) > 1e-9 {
		t.Fatalf("역순 좌표를 바로잡지 못했다: %#v", objects[0])
	}
}

func TestParseVLMSeatsDeduplicatesRepeatedBoxes(t *testing.T) {
	// 모델이 같은 좌석을 반복 출력하는 흔한 퇴화 모드.
	repeated := strings.Repeat(`{"x":0.1,"y":0.1,"w":0.05,"h":0.05},`, 20)
	body := `{"seats":[` + repeated + `{"x":0.5,"y":0.5,"w":0.05,"h":0.05}]}`
	parsed, err := parseVLMSeats(body, 1000, 600)
	if err != nil {
		t.Fatalf("예상치 못한 오류: %v", err)
	}
	objects, raw := parsed.Objects, parsed.Raw
	if raw != 21 {
		t.Fatalf("원본 21개를 기대했으나 %d", raw)
	}
	if len(objects) != 2 {
		t.Fatalf("중복 제거 후 2개를 기대했으나 %d", len(objects))
	}
}

func TestParseVLMSeatsEmptyAndInvalid(t *testing.T) {
	if parsed, err := parseVLMSeats(`{"seats":[]}`, 800, 600); err != nil || len(parsed.Objects) != 0 {
		t.Fatalf("좌석 없음 응답은 오류가 아니어야 한다: %v / %d", err, len(parsed.Objects))
	}
	if _, err := parseVLMSeats(`좌석을 찾을 수 없습니다`, 800, 600); err == nil {
		t.Fatal("JSON이 아닌 응답을 통과시켰다")
	}
	// 어떤 좌표계로도 해석되지 않는 값.
	if _, err := parseVLMSeats(`{"seats":[{"bbox":[90000,90000,95000,95000]}]}`, 800, 600); err == nil {
		t.Fatal("범위를 벗어난 좌표를 통과시켰다")
	}
}

func TestFuseDetectionsCalibratesConfidence(t *testing.T) {
	cvObjects := []detectedObject{
		{X: .10, Y: .10, W: .05, H: .05, Confidence: .90, Source: "cv"}, // VLM과 합의
		{X: .50, Y: .50, W: .05, H: .05, Confidence: .97, Source: "cv"}, // CV 단독
	}
	vlmObjects := []detectedObject{
		{X: .102, Y: .101, W: .05, H: .05, Confidence: .80, Source: "vlm"}, // 위 좌석과 겹침
		{X: .80, Y: .80, W: .05, H: .05, Confidence: .94, Source: "vlm"},   // VLM 단독
	}
	fused, stats := fuseDetections(cvObjects, vlmObjects, .35, nil)
	if stats["agreed"] != 1 || stats["cvOnly"] != 1 || stats["vlmOnly"] != 1 {
		t.Fatalf("융합 통계가 기대와 다르다: %#v", stats)
	}
	if len(fused) != 3 {
		t.Fatalf("좌석 3개를 기대했으나 %d개", len(fused))
	}
	bySource := map[string]detectedObject{}
	for _, object := range fused {
		bySource[object.Source] = object
	}
	agreed, ok := bySource["cv+vlm"]
	if !ok {
		t.Fatal("교차 검증 좌석이 없다")
	}
	if agreed.Confidence < fusionAgreementBase {
		t.Fatalf("합의 좌석 신뢰도가 올라가지 않았다: %.3f", agreed.Confidence)
	}
	cvOnly := bySource["cv"]
	if cvOnly.Confidence >= .97 {
		t.Fatalf("CV 단독 좌석에 감점이 없다: %.3f", cvOnly.Confidence)
	}
	vlmOnly := bySource["vlm"]
	if vlmOnly.Confidence > fusionVLMOnlyCap {
		t.Fatalf("VLM 단독 좌석이 상한을 넘었다: %.3f", vlmOnly.Confidence)
	}
	// VLM 단독은 자동 승인선(0.95)을 절대 넘지 않아야 한다.
	if vlmOnly.Confidence >= .95 {
		t.Fatalf("VLM 단독 좌석이 자동 승인 구간에 들어갔다: %.3f", vlmOnly.Confidence)
	}
}

func TestFuseDetectionsWithoutVLMResults(t *testing.T) {
	cvObjects := []detectedObject{{X: .1, Y: .1, W: .05, H: .05, Confidence: .98, Source: "cv"}}
	fused, stats := fuseDetections(cvObjects, nil, .35, nil)
	if len(fused) != 1 || stats["cvOnly"] != 1 || stats["agreed"] != 0 {
		t.Fatalf("VLM 결과가 없을 때 CV 결과가 유지되지 않았다: %#v / %#v", fused, stats)
	}
}

func TestDetectWithVLMSuccess(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		if r.Header.Get("Authorization") != "Bearer secret-key" {
			t.Errorf("API 키가 전달되지 않았다: %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "qwen2.5-vl-7b-instruct" {
			t.Errorf("모델 이름이 전달되지 않았다: %v", body["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, chatResponse(`{"seats":[{"x":0.1,"y":0.1,"w":0.06,"h":0.06,"confidence":0.9}]}`, "stop"))
	}))
	defer server.Close()
	cfg := testVLMConfig(server.URL)
	cfg.apiKey = "secret-key"
	result, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), cfg)
	if err != nil {
		t.Fatalf("예상치 못한 오류: %v", err)
	}
	if len(result.Objects) != 1 {
		t.Fatalf("좌석 1개를 기대했으나 %d개", len(result.Objects))
	}
	if requests != 1 {
		t.Fatalf("요청 1회를 기대했으나 %d회", requests)
	}
}

func TestDetectWithVLMAuthFailureDoesNotRetry(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	_, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), testVLMConfig(server.URL))
	if err == nil {
		t.Fatal("인증 실패가 오류로 보고되지 않았다")
	}
	var typed *vlmError
	if !asVLMError(err, &typed) || typed.Kind != "auth" {
		t.Fatalf("auth 오류를 기대했으나 %v", err)
	}
	if requests != 1 {
		t.Fatalf("인증 실패는 재시도하지 않아야 한다. 요청 %d회", requests)
	}
}

func TestDetectWithVLMRetriesServerErrorThenSucceeds(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&requests, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, chatResponse(`{"seats":[{"x":0.2,"y":0.2,"w":0.06,"h":0.06}]}`, "stop"))
	}))
	defer server.Close()
	result, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), testVLMConfig(server.URL))
	if err != nil {
		t.Fatalf("재시도 후에도 실패했다: %v", err)
	}
	if len(result.Objects) != 1 {
		t.Fatalf("좌석 1개를 기대했으나 %d개", len(result.Objects))
	}
	if requests != 2 {
		t.Fatalf("요청 2회를 기대했으나 %d회", requests)
	}
}

func TestDetectWithVLMFallsBackWhenResponseFormatUnsupported(t *testing.T) {
	var sawResponseFormat, sawPlain int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["response_format"]; ok {
			atomic.AddInt32(&sawResponseFormat, 1)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"message":"response_format is not supported by this model"}}`)
			return
		}
		atomic.AddInt32(&sawPlain, 1)
		_, _ = io.WriteString(w, chatResponse("```json\n{\"seats\":[{\"x\":0.3,\"y\":0.3,\"w\":0.06,\"h\":0.06}]}\n```", "stop"))
	}))
	defer server.Close()
	result, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), testVLMConfig(server.URL))
	if err != nil {
		t.Fatalf("response_format 미지원 서버에서 실패했다: %v", err)
	}
	if len(result.Objects) != 1 {
		t.Fatalf("좌석 1개를 기대했으나 %d개", len(result.Objects))
	}
	if sawResponseFormat != 1 || sawPlain != 1 {
		t.Fatalf("response_format 재시도 흐름이 기대와 다르다: %d / %d", sawResponseFormat, sawPlain)
	}
}

func TestDetectWithVLMRepairsNonJSONReply(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&requests, 1) == 1 {
			_, _ = io.WriteString(w, chatResponse("도면에서 좌석을 찾았습니다만 형식을 지키지 못했습니다.", "stop"))
			return
		}
		_, _ = io.WriteString(w, chatResponse(`{"seats":[{"x":0.4,"y":0.4,"w":0.06,"h":0.06}]}`, "stop"))
	}))
	defer server.Close()
	result, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), testVLMConfig(server.URL))
	if err != nil {
		t.Fatalf("복구 재시도가 동작하지 않았다: %v", err)
	}
	if len(result.Objects) != 1 || requests != 2 {
		t.Fatalf("좌석 %d개 / 요청 %d회", len(result.Objects), requests)
	}
}

func TestDetectWithVLMTimeoutIsClassifiedAsNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	cfg := testVLMConfig(server.URL)
	cfg.timeout = 300 * time.Millisecond
	_, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), cfg)
	if err == nil {
		t.Fatal("타임아웃이 오류로 보고되지 않았다")
	}
	var typed *vlmError
	if !asVLMError(err, &typed) || typed.Kind != "network" {
		t.Fatalf("network 오류를 기대했으나 %v", err)
	}
}

func TestDetectWithVLMEmptyResultIsTyped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, chatResponse(`{"seats":[]}`, "stop"))
	}))
	defer server.Close()
	_, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), testVLMConfig(server.URL))
	var typed *vlmError
	if !asVLMError(err, &typed) || typed.Kind != "empty" {
		t.Fatalf("empty 오류를 기대했으나 %v", err)
	}
}

func TestDetectWithVLMTilingRemapsCoordinates(t *testing.T) {
	// 모든 타일이 자기 좌표계의 좌상단 좌석을 보고하면, 전역 좌표로 흩어져야 한다.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, chatResponse(`{"seats":[{"x":0.05,"y":0.05,"w":0.1,"h":0.1}]}`, "stop"))
	}))
	defer server.Close()
	cfg := testVLMConfig(server.URL)
	cfg.tiles = 2
	result, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), cfg)
	if err != nil {
		t.Fatalf("타일 분석 실패: %v", err)
	}
	if len(result.Objects) != 4 {
		t.Fatalf("타일 4개에서 좌석 4개를 기대했으나 %d개: %#v", len(result.Objects), result.Objects)
	}
	seen := map[string]bool{}
	for _, object := range result.Objects {
		if object.X < 0 || object.Y < 0 || object.X+object.W > 1.0001 || object.Y+object.H > 1.0001 {
			t.Fatalf("전역 좌표 범위를 벗어남: %#v", object)
		}
		key := fmt.Sprintf("%.2f/%.2f", object.X, object.Y)
		if seen[key] {
			t.Fatalf("타일 좌표가 전역으로 옮겨지지 않아 중복됐다: %s", key)
		}
		seen[key] = true
	}
}

func TestDetectWithVLMPartialTileFailureWarns(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// 첫 타일만 계속 실패시켜 부분 실패 경고를 확인한다.
		if atomic.AddInt32(&requests, 1) <= vlmMaxAttempts {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, chatResponse(`{"seats":[{"x":0.05,"y":0.05,"w":0.1,"h":0.1}]}`, "stop"))
	}))
	defer server.Close()
	cfg := testVLMConfig(server.URL)
	cfg.tiles = 2
	result, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), cfg)
	if err != nil {
		t.Fatalf("일부 타일 성공 시에는 결과를 돌려야 한다: %v", err)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "실패") {
		t.Fatalf("부분 실패 경고가 없다: %#v", result.Warnings)
	}
	if len(result.Objects) != 3 {
		t.Fatalf("성공한 타일 3개의 좌석을 기대했으나 %d개", len(result.Objects))
	}
}

func TestDetectWithVLMSeatCapIsEnforced(t *testing.T) {
	seats := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		// 겹치지 않도록 격자로 흩어 놓는다.
		x := float64(i%8) * .12
		y := float64(i/8) * .18
		seats = append(seats, fmt.Sprintf(`{"x":%.3f,"y":%.3f,"w":0.05,"h":0.05}`, x, y))
	}
	body := `{"seats":[` + strings.Join(seats, ",") + `]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, chatResponse(body, "stop"))
	}))
	defer server.Close()
	cfg := testVLMConfig(server.URL)
	cfg.maxSeats = 10
	result, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), cfg)
	if err != nil {
		t.Fatalf("예상치 못한 오류: %v", err)
	}
	if len(result.Objects) != 10 {
		t.Fatalf("상한 10개를 기대했으나 %d개", len(result.Objects))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("좌석 상한 초과 경고가 없다")
	}
}

func TestDownscaleImagePreservesAspectAndBounds(t *testing.T) {
	source := image.NewGray(image.Rect(0, 0, 1200, 600))
	scaled := downscaleImage(source, 600)
	if scaled.Bounds().Dx() != 600 || scaled.Bounds().Dy() != 300 {
		t.Fatalf("축소 크기가 기대와 다르다: %v", scaled.Bounds())
	}
	// 이미 작은 이미지는 확대하지 않는다.
	small := downscaleImage(image.NewGray(image.Rect(0, 0, 100, 80)), 600)
	if small.Bounds().Dx() != 100 || small.Bounds().Dy() != 80 {
		t.Fatalf("작은 이미지를 확대했다: %v", small.Bounds())
	}
}

func TestEncodeImageDataURI(t *testing.T) {
	uri, err := encodeImageDataURI(probeFloorPlan())
	if err != nil {
		t.Fatalf("인코딩 실패: %v", err)
	}
	if !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Fatalf("PNG 데이터 URI를 기대했으나 %.40s", uri)
	}
}

func TestValidEngine(t *testing.T) {
	for input, want := range map[string]string{
		"cv": engineCV, "vlm": engineVLM, "hybrid": engineHybrid,
		"VLM": engineVLM, " hybrid ": engineHybrid, "": engineCV, "nonsense": engineCV,
	} {
		if got := validEngine(input); got != want {
			t.Errorf("validEngine(%q) = %q, 기대 %q", input, got, want)
		}
	}
}

func TestRunningAnalysesPreventsDoubleClaim(t *testing.T) {
	var running runningAnalyses
	if !running.claim("map-1") {
		t.Fatal("첫 점유가 거부됐다")
	}
	if running.claim("map-1") {
		t.Fatal("중복 점유를 허용했다")
	}
	if !running.claim("map-2") {
		t.Fatal("다른 도면 점유가 거부됐다")
	}
	running.release("map-1")
	if !running.claim("map-1") {
		t.Fatal("해제 후 재점유가 거부됐다")
	}
}

// 도면에 책상 격자를 그려 CV가 반드시 좌석을 찾을 수 있는 입력을 만든다.
func fusionTestPlan() image.Image { return officePlan().img }

func TestRunDetectionCVOnlyNeverCallsVLM(t *testing.T) {
	called := false
	plan := detectionPlan{engine: engineCV, fusionIoU: .35,
		runVLM: func(context.Context, image.Image) (vlmResult, error) {
			called = true
			return vlmResult{}, nil
		}}
	outcome := runDetection(context.Background(), plan, fusionTestPlan())
	if called {
		t.Fatal("CV 전용 엔진이 VLM을 호출했다")
	}
	if len(outcome.Objects) == 0 {
		t.Fatal("CV 좌석을 찾지 못했다")
	}
	if len(outcome.Warnings) != 0 {
		t.Fatalf("CV 전용에서 경고가 발생했다: %#v", outcome.Warnings)
	}
}

func TestRunDetectionVLMFailureFallsBackToCV(t *testing.T) {
	plan := detectionPlan{engine: engineVLM, fusionIoU: .35,
		runVLM: func(context.Context, image.Image) (vlmResult, error) {
			return vlmResult{}, vlmFail("network", "VLM 서버에 연결할 수 없습니다", nil)
		}}
	outcome := runDetection(context.Background(), plan, fusionTestPlan())
	if len(outcome.Objects) == 0 {
		t.Fatal("VLM 실패 시 CV 결과로 대체되지 않았다")
	}
	if outcome.Details["fallback"] != engineCV {
		t.Fatalf("폴백 사실이 기록되지 않았다: %#v", outcome.Details)
	}
	if outcome.Details["vlmErrorKind"] != "network" {
		t.Fatalf("VLM 오류 분류가 기록되지 않았다: %#v", outcome.Details)
	}
	if len(outcome.Warnings) == 0 || !strings.Contains(outcome.Warnings[0], "대체") {
		t.Fatalf("대체 사실을 경고로 알리지 않았다: %#v", outcome.Warnings)
	}
	for _, object := range outcome.Objects {
		if object.Source != "cv" && object.Source != "grid-fill" {
			t.Fatalf("폴백 좌석의 출처가 잘못됐다: %q", object.Source)
		}
	}
}

func TestRunDetectionHybridSurvivesVLMFailure(t *testing.T) {
	plan := detectionPlan{engine: engineHybrid, fusionIoU: .35,
		runVLM: func(context.Context, image.Image) (vlmResult, error) {
			return vlmResult{}, vlmFail("auth", "VLM 인증에 실패했습니다", nil)
		}}
	outcome := runDetection(context.Background(), plan, fusionTestPlan())
	if len(outcome.Objects) == 0 {
		t.Fatal("하이브리드에서 VLM이 실패하자 좌석이 사라졌다")
	}
	// 교차 검증이 없었으므로 융합 통계는 기록되지 않는다.
	if _, ok := outcome.Details["agreed"]; ok {
		t.Fatal("VLM 실패인데 융합 통계가 기록됐다")
	}
	if len(outcome.Warnings) == 0 {
		t.Fatal("VLM 실패를 경고하지 않았다")
	}
}

func TestRunDetectionHybridMarksAgreementAndDisagreement(t *testing.T) {
	planImage := fusionTestPlan()
	cvOnly := runDetection(context.Background(), detectionPlan{engine: engineCV}, planImage)
	if len(cvOnly.Objects) < 3 {
		t.Fatalf("융합 시험에 필요한 CV 좌석이 부족하다: %d", len(cvOnly.Objects))
	}
	// CV가 찾은 좌석 중 둘은 VLM도 찾았다고 하고, 겹치지 않는 좌석 하나를 더 준다.
	echo := []detectedObject{cvOnly.Objects[0], cvOnly.Objects[1]}
	for i := range echo {
		echo[i].Confidence = .8
		echo[i].Source = "vlm"
	}
	// 책상 격자에서 벗어난 위치의 VLM 단독 상자는 오검출로 보아 제외되어야 한다.
	echo = append(echo, detectedObject{X: .90, Y: .93, W: .04, H: .04, Confidence: .9, Source: "vlm"})
	plan := detectionPlan{engine: engineHybrid, fusionIoU: .35, model: "test-vlm",
		runVLM: func(context.Context, image.Image) (vlmResult, error) {
			return vlmResult{Objects: echo, Convention: "normalized/xywh", RawSeats: len(echo), Tiles: 1}, nil
		}}
	outcome := runDetection(context.Background(), plan, planImage)
	agreed, _ := outcome.Details["agreed"].(int)
	vlmOnly, _ := outcome.Details["vlmOnly"].(int)
	offLatticeCount, _ := outcome.Details["vlmOffLattice"].(int)
	if agreed != 2 {
		t.Fatalf("교차 검증 2건을 기대했으나 %d건: %#v", agreed, outcome.Details)
	}
	if vlmOnly+offLatticeCount != 1 {
		t.Fatalf("VLM 단독 상자 1건이 집계되지 않았다: 유지 %d건 제외 %d건", vlmOnly, offLatticeCount)
	}
	if offLatticeCount != 1 {
		t.Fatalf("격자를 벗어난 VLM 상자를 제외하지 않았다: %#v", outcome.Details)
	}
	if outcome.Details["vlmModel"] != "test-vlm" {
		t.Fatalf("모델 이름이 기록되지 않았다: %#v", outcome.Details)
	}
	for _, object := range outcome.Objects {
		if object.Source == "vlm" && object.Confidence >= .95 {
			t.Fatalf("VLM 단독 좌석이 자동 승인 구간에 들어갔다: %.3f", object.Confidence)
		}
	}
}

// jsonMode를 끄면 response_format을 아예 보내지 않아야 한다. 모델에 따라 JSON
// 모드가 인식 품질을 떨어뜨리는 경우가 있어 운영자가 끌 수 있어야 한다.
func TestDetectWithVLMHonoursJSONModeOff(t *testing.T) {
	var sawResponseFormat int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["response_format"]; ok {
			atomic.AddInt32(&sawResponseFormat, 1)
		}
		_, _ = io.WriteString(w, chatResponse(`{"objects":[{"bbox_2d":[100,100,160,160]}]}`, "stop"))
	}))
	defer server.Close()
	cfg := testVLMConfig(server.URL)
	cfg.jsonMode = false
	result, err := testServer().detectWithVLM(context.Background(), probeFloorPlan(), cfg)
	if err != nil {
		t.Fatalf("예상치 못한 오류: %v", err)
	}
	if sawResponseFormat != 0 {
		t.Fatalf("jsonMode가 꺼졌는데 response_format을 보냈다: %d회", sawResponseFormat)
	}
	if len(result.Objects) != 1 {
		t.Fatalf("좌석 1개를 기대했으나 %d개", len(result.Objects))
	}
}

// Qwen 계열의 네이티브 grounding 응답(객체 래퍼 + bbox_2d 픽셀 좌표)을
// 그대로 해석해야 한다. 실측에서 이 형식이 가장 정확했다.
func TestParseVLMSeatsAcceptsQwenNativeObjectWrapper(t *testing.T) {
	body := `{"objects":[{"bbox_2d":[100,120,220,200],"label":"desk"},{"bbox_2d":[400,120,520,200],"label":"desk"}]}`
	parsed, err := parseVLMSeats(body, 1000, 600)
	if err != nil {
		t.Fatalf("네이티브 응답 해석 실패: %v", err)
	}
	if len(parsed.Objects) != 2 {
		t.Fatalf("좌석 2개를 기대했으나 %d개", len(parsed.Objects))
	}
	if parsed.Convention != "pixel/corners" {
		t.Fatalf("픽셀 corners 좌표계를 기대했으나 %q", parsed.Convention)
	}
	first := parsed.Objects[0]
	if math.Abs(first.X-.1) > 1e-9 || math.Abs(first.W-.12) > 1e-9 {
		t.Fatalf("좌표 변환이 틀렸다: %#v", first)
	}
}

// CV가 격자를 세운 도면에서 격자를 벗어난 VLM 단독 상자는 버려야 한다. 실측에서
// 이런 상자가 대부분 환각이어서 하이브리드의 정밀도를 떨어뜨렸다.
func TestFuseDetectionsDropsOffLatticeVLMBoxes(t *testing.T) {
	grid := &seatGrid{OriginX: .1, OriginY: .1, PitchX: .2, PitchY: .2}
	cvObjects := []detectedObject{{X: .1, Y: .1, W: .05, H: .05, Confidence: .95, Source: "cv"}}
	vlmObjects := []detectedObject{
		{X: .3, Y: .1, W: .05, H: .05, Confidence: .9, Source: "vlm"},   // 격자 위 → 유지
		{X: .37, Y: .16, W: .05, H: .05, Confidence: .9, Source: "vlm"}, // 격자 밖 → 제거
	}
	fused, stats := fuseDetections(cvObjects, vlmObjects, .35, grid)
	if stats["vlmOnly"] != 1 || stats["vlmOffLattice"] != 1 {
		t.Fatalf("격자 밖 상자를 걸러내지 못했다: %#v", stats)
	}
	if len(fused) != 2 {
		t.Fatalf("좌석 2개를 기대했으나 %d개", len(fused))
	}
	// 격자가 없으면(격자를 못 세운 도면) 걸러내지 않는다.
	_, noGrid := fuseDetections(cvObjects, vlmObjects, .35, nil)
	if noGrid["vlmOnly"] != 2 || noGrid["vlmOffLattice"] != 0 {
		t.Fatalf("격자가 없을 때는 유지해야 한다: %#v", noGrid)
	}
}
