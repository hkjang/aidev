package app

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

type planCanvas struct{ img *image.Gray }

func newPlan(w, h int, background uint8) *planCanvas {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = background
	}
	return &planCanvas{img: img}
}

func (p *planCanvas) hLine(x0, x1, y int, ink uint8) {
	for x := x0; x <= x1; x++ {
		p.img.SetGray(x, y, color.Gray{Y: ink})
	}
}

func (p *planCanvas) vLine(x, y0, y1 int, ink uint8) {
	for y := y0; y <= y1; y++ {
		p.img.SetGray(x, y, color.Gray{Y: ink})
	}
}

// outline은 책상처럼 얇은 선으로 그린 빈 사각형이다.
func (p *planCanvas) outline(x, y, w, h int, ink uint8) {
	p.hLine(x, x+w-1, y, ink)
	p.hLine(x, x+w-1, y+h-1, ink)
	p.vLine(x, y, y+h-1, ink)
	p.vLine(x+w-1, y, y+h-1, ink)
}

func (p *planCanvas) filled(x, y, w, h int, ink uint8) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			p.img.SetGray(x+dx, y+dy, color.Gray{Y: ink})
		}
	}
}

const (
	planW, planH           = 800, 600
	deskW, deskH           = 40, 30
	pitchXpx, pitchYpx     = 100, 80
	firstColX, firstRowY   = 60, 80
	deskColumns, deskRows  = 5, 4
	missingCol, missingRow = 2, 1
)

// officePlan은 외벽·복도벽, 5×4 책상 격자, 도면 기호 노이즈를 갖춘 합성 도면을
// 만든다. missingCol/missingRow 자리의 책상은 흔적만 남겨 격자 보간을 검증한다.
func officePlan() *planCanvas {
	p := newPlan(planW, planH, 255)
	// 외벽과 복도벽 — 길게 이어진 획이라 전처리에서 제거되어야 한다.
	p.outline(10, 10, planW-20, planH-20, 0)
	p.vLine(540, 10, planH-11, 0)
	for row := 0; row < deskRows; row++ {
		for col := 0; col < deskColumns; col++ {
			x, y := firstColX+col*pitchXpx, firstRowY+row*pitchYpx
			if col == missingCol && row == missingRow {
				// 가려진 책상: 좌석 후보로 잡히기엔 작지만 잉크는 남아 있다.
				p.filled(x+10, y+10, 8, 7, 0)
				continue
			}
			p.outline(x, y, deskW, deskH, 0)
		}
	}
	// 책상 격자 밖의 글자·라벨 노이즈.
	p.filled(620, 470, 6, 8, 0)
	p.filled(640, 470, 6, 8, 0)
	p.filled(600, 500, 34, 9, 0)
	return p
}

func TestDetectSeatsRecoversDeskGrid(t *testing.T) {
	result := detectSeats(officePlan().img)
	if !result.Lattice {
		t.Fatal("책상 반복 격자를 찾지 못했다")
	}
	expected := deskColumns*deskRows - 1
	if len(result.Objects) != expected+1 {
		t.Fatalf("좌석 후보 %d개를 기대했으나 %d개", expected+1, len(result.Objects))
	}
	if result.Filled != 1 {
		t.Fatalf("격자 보간 좌석 1개를 기대했으나 %d개", result.Filled)
	}
	for _, object := range result.Objects {
		if object.X < 0 || object.Y < 0 || object.X+object.W > 1 || object.Y+object.H > 1 {
			t.Fatalf("정규화 범위를 벗어난 좌석: %#v", object)
		}
		if object.Filled {
			// 흔적만 있는 자리는 자동 승인되지 않고 검토 대상으로 남아야 한다.
			if object.Confidence >= .95 {
				t.Fatalf("보간 좌석이 자동 승인 신뢰도를 넘었다: %.3f", object.Confidence)
			}
			continue
		}
		if object.Confidence < .95 {
			t.Fatalf("격자에 정확히 놓인 좌석의 신뢰도가 낮다: %.3f", object.Confidence)
		}
	}
}

func TestDetectSeatsGridMatchesSeatPositions(t *testing.T) {
	result := detectSeats(officePlan().img)
	if result.Grid == nil {
		t.Fatal("격자 보정값을 돌려주지 않았다")
	}
	if got, want := result.Grid.PitchX, float64(pitchXpx)/planW; math.Abs(got-want) > .002 {
		t.Fatalf("가로 간격 %.4f, 기대 %.4f", got, want)
	}
	if got, want := result.Grid.PitchY, float64(pitchYpx)/planH; math.Abs(got-want) > .002 {
		t.Fatalf("세로 간격 %.4f, 기대 %.4f", got, want)
	}
	// 추론한 격자로 스냅해도 좌석이 제자리에 남아야 한다.
	for _, object := range result.Objects {
		x, y := result.Grid.snap(object.X, object.Y)
		if math.Abs(x-object.X) > .01 || math.Abs(y-object.Y) > .01 {
			t.Fatalf("격자 스냅이 좌석을 옮겼다: (%.4f,%.4f) → (%.4f,%.4f)", object.X, object.Y, x, y)
		}
	}
}

// 책상 윤곽이 벽선과 이어져 있어도 각각 분리되어야 한다. 이전 구현은 벽에 붙은
// 책상들이 하나의 거대한 덩어리로 합쳐져 크기 필터에 전부 걸러졌다.
func TestDetectSeatsSeparatesWallAttachedDesks(t *testing.T) {
	p := newPlan(planW, planH, 255)
	const rowY = 100
	p.hLine(0, planW-1, rowY, 0) // 책상 윗변을 모두 관통하는 벽선
	for col := 0; col < deskColumns; col++ {
		p.outline(firstColX+col*pitchXpx, rowY, deskW, deskH, 0)
	}
	result := detectSeats(p.img)
	if len(result.Objects) != deskColumns {
		t.Fatalf("벽선에 붙은 책상 %d개를 기대했으나 %d개", deskColumns, len(result.Objects))
	}
	// 한 줄뿐이면 세로 주기를 세울 수 없으므로 전부 검토 대상이어야 한다.
	if result.Lattice {
		t.Fatal("한 줄짜리 배치에서 격자를 세웠다")
	}
	for _, object := range result.Objects {
		if object.Confidence >= .95 {
			t.Fatalf("격자 없이 자동 승인된 좌석: %.3f", object.Confidence)
		}
	}
}

// 스캔 도면처럼 대비가 낮아도 Otsu 이진화로 책상을 찾아야 한다.
// 고정 임계값(<105)만 쓰던 이전 구현은 아무것도 찾지 못했다.
func TestDetectSeatsHandlesLowContrastPlan(t *testing.T) {
	p := newPlan(planW, planH, 205)
	for row := 0; row < deskRows; row++ {
		for col := 0; col < deskColumns; col++ {
			p.outline(firstColX+col*pitchXpx, firstRowY+row*pitchYpx, deskW, deskH, 140)
		}
	}
	result := detectSeats(p.img)
	if len(result.Objects) < deskColumns*deskRows {
		t.Fatalf("저대비 도면에서 좌석 %d개 이상을 기대했으나 %d개", deskColumns*deskRows, len(result.Objects))
	}
}

func TestDetectSeatsIgnoresBlankPlan(t *testing.T) {
	result := detectSeats(newPlan(400, 300, 255).img)
	if len(result.Objects) != 0 || result.Grid != nil {
		t.Fatalf("빈 도면에서 좌석을 만들었다: %#v", result)
	}
}

func TestSeatGridSnapAndValidation(t *testing.T) {
	grid := seatGrid{OriginX: .1, OriginY: .2, PitchX: .05, PitchY: .08}
	if !grid.valid() {
		t.Fatal("정상 격자를 거부했다")
	}
	x, y := grid.snap(.163, .118)
	if math.Abs(x-.15) > 1e-9 || math.Abs(y-.12) > 1e-9 {
		t.Fatalf("스냅 결과 (%.4f,%.4f), 기대 (0.15,0.12)", x, y)
	}
	if (seatGrid{PitchX: .001, PitchY: .05}).valid() {
		t.Fatal("너무 촘촘한 간격을 허용했다")
	}
	if parseSeatGrid(nil) != nil || parseSeatGrid([]byte(`{}`)) != nil {
		t.Fatal("빈 격자를 유효한 값으로 읽었다")
	}
	if parseSeatGrid([]byte(`{"originX":0.1,"originY":0.2,"pitchX":0.05,"pitchY":0.08}`)) == nil {
		t.Fatal("저장된 격자를 읽지 못했다")
	}
}

func TestDetectionMessage(t *testing.T) {
	cv := detectionOutcome{Engine: engineCV, Lattice: true}
	if got := detectionMessage(cv, 0, 0, nil); got == "" {
		t.Fatal("빈 안내 문구")
	}
	if got := detectionMessage(cv, 3, 0, nil); got == "" {
		t.Fatal("빈 결과 문구")
	}
	if detectionMessage(cv, 3, 2, nil) == detectionMessage(cv, 3, 0, nil) {
		t.Fatal("격자 보간 건수를 안내하지 않았다")
	}
	hybrid := detectionOutcome{Engine: engineHybrid}
	message := detectionMessage(hybrid, 5, 0, map[string]int{"cv+vlm": 3, "cv": 1, "vlm": 1})
	if !strings.Contains(message, "교차 검증") {
		t.Fatalf("하이브리드 결과에 교차 검증 요약이 없다: %s", message)
	}
	if !strings.Contains(detectionMessage(detectionOutcome{Engine: engineVLM}, 2, 0, nil), "검토 대상") {
		t.Fatal("VLM 단독 결과에 검토 안내가 없다")
	}
}

// 실제 CAD 도면에는 책상마다 의자가 딸려 있다. 의자 수가 책상 수와 같아서
// 중앙값 크기 필터만으로는 의자를 좌석으로 오인했다. 크기 군집 중 격자를
// 이루는 쪽을 고르고, 비슷하면 더 큰 도형을 택해야 한다.
func TestDetectSeatsPrefersDesksOverChairs(t *testing.T) {
	p := newPlan(planW, planH, 255)
	p.outline(10, 10, planW-20, planH-20, 0)
	for row := 0; row < deskRows; row++ {
		for col := 0; col < deskColumns; col++ {
			x, y := firstColX+col*pitchXpx, firstRowY+row*pitchYpx
			p.outline(x, y, deskW, deskH, 0)
			// 책상 아래 의자: 책상과 같은 개수, 같은 격자 주기.
			p.outline(x+deskW/2-9, y+deskH+4, 18, 16, 0)
		}
	}
	result := detectSeats(p.img)
	expected := deskColumns * deskRows
	if len(result.Objects) != expected {
		t.Fatalf("책상 %d개를 기대했으나 %d개", expected, len(result.Objects))
	}
	// 책상 크기(40x30)가 잡혀야 하고 의자 크기(18x16)가 잡히면 안 된다.
	for _, object := range result.Objects {
		width := object.W * planW
		if width < float64(deskW)*.8 {
			t.Fatalf("의자를 좌석으로 인식했다: 폭 %.1fpx", width)
		}
	}
	if result.Grid == nil {
		t.Fatal("격자를 찾지 못했다")
	}
	// 격자가 책상 좌표와 맞물려야 정렬이 좌석을 옮기지 않는다.
	for _, object := range result.Objects {
		x, y := result.Grid.snap(object.X, object.Y)
		if math.Abs(x-object.X) > .01 || math.Abs(y-object.Y) > .01 {
			t.Fatalf("격자 스냅이 책상을 옮겼다: (%.4f,%.4f) → (%.4f,%.4f)", object.X, object.Y, x, y)
		}
	}
}

func TestSeatGridSourceControlsOverwrite(t *testing.T) {
	manual := parseSeatGrid([]byte(`{"originX":0.1,"originY":0.2,"pitchX":0.05,"pitchY":0.08,"source":"manual"}`))
	if manual == nil || manual.Source != gridSourceManual {
		t.Fatalf("관리자 격자의 출처를 읽지 못했다: %#v", manual)
	}
	auto := parseSeatGrid([]byte(`{"originX":0.1,"originY":0.2,"pitchX":0.05,"pitchY":0.08}`))
	if auto == nil || auto.Source != "" {
		t.Fatalf("출처 없는 격자를 잘못 읽었다: %#v", auto)
	}
}
