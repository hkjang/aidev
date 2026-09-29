package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	engineCV     = "cv"
	engineVLM    = "vlm"
	engineHybrid = "hybrid"
	// 분석 잡이 이 시간을 넘기면 프로세스가 죽은 것으로 보고 실패 처리한다.
	analysisJobTimeout = 30 * time.Minute
)

// 교차 검증된 좌석만 자동 승인 구간에 올린다. CV 단독은 소폭 감점,
// VLM 단독은 상한을 두어 반드시 검토를 거치게 한다.
const (
	fusionAgreementBase = .90
	fusionCVOnlyFactor  = .93
	fusionVLMOnlyCap    = .88
)

func validEngine(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case engineVLM:
		return engineVLM
	case engineHybrid:
		return engineHybrid
	default:
		return engineCV
	}
}

// runningAnalyses는 같은 도면을 동시에 두 번 분석하지 못하게 막는다.
type runningAnalyses struct {
	mu  sync.Mutex
	ids map[string]bool
}

func (r *runningAnalyses) claim(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ids == nil {
		r.ids = map[string]bool{}
	}
	if r.ids[id] {
		return false
	}
	r.ids[id] = true
	return true
}

func (r *runningAnalyses) release(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.ids, id)
}

type detectionOutcome struct {
	Objects  []detectedObject
	Grid     *seatGrid
	Lattice  bool
	Filled   int
	Engine   string
	Warnings []string
	Details  map[string]any
}

// vlmRunner는 VLM 호출을 주입 가능한 형태로 감싼다. 설정 읽기와 실제 추론을
// 분리해 폴백 경로를 데이터베이스 없이 검증할 수 있게 하는 것이 목적이다.
type vlmRunner func(ctx context.Context, img image.Image) (vlmResult, error)

type detectionPlan struct {
	engine    string
	fusionIoU float64
	model     string
	runVLM    vlmRunner
}

// detectionPlanFor는 설정에서 실행 계획을 만든다. VLM 설정이 잘못되어 있으면
// 그 오류를 그대로 돌려주는 실행기를 넣어 폴백 경로가 일관되게 동작하게 한다.
func (s *Server) detectionPlanFor(ctx context.Context, engine string) detectionPlan {
	plan := detectionPlan{engine: engine, fusionIoU: .35}
	if v, _ := s.getSetting(ctx, "ai.fusion_iou"); v != "" {
		plan.fusionIoU = math.Max(.1, math.Min(.9, parseFloat(v, .35)))
	}
	if engine == engineCV {
		return plan
	}
	cfg, err := s.vlmConfigFrom(ctx)
	if err != nil {
		plan.runVLM = func(context.Context, image.Image) (vlmResult, error) {
			return vlmResult{}, err
		}
		return plan
	}
	plan.model = cfg.model
	plan.runVLM = func(callCtx context.Context, img image.Image) (vlmResult, error) {
		return s.detectWithVLM(callCtx, img, cfg)
	}
	return plan
}

// detectWithCV는 오프라인 CV 결과를 outcome에 채운다.
func detectWithCV(img image.Image, outcome *detectionOutcome) {
	cv := detectSeats(img)
	for i := range cv.Objects {
		if cv.Objects[i].Filled {
			cv.Objects[i].Source = "grid-fill"
		} else {
			cv.Objects[i].Source = "cv"
		}
	}
	outcome.Objects = cv.Objects
	outcome.Grid = cv.Grid
	outcome.Lattice = cv.Lattice
	outcome.Filled = cv.Filled
	outcome.Details["cvSeats"] = len(cv.Objects)
	outcome.Details["cvLattice"] = cv.Lattice
}

// runDetection은 계획된 엔진으로 좌석 후보를 찾는다. hybrid는 CV의 기하학적
// 정확도와 VLM의 의미 이해를 교차 검증해 신뢰도를 보정하고, VLM이 어떤 이유로든
// 실패하면 항상 CV 결과로 물러나 분석 자체는 완료된다.
func runDetection(ctx context.Context, plan detectionPlan, img image.Image) detectionOutcome {
	outcome := detectionOutcome{Engine: plan.engine, Details: map[string]any{}}
	if plan.engine == engineCV || plan.engine == engineHybrid {
		detectWithCV(img, &outcome)
	}
	if plan.engine == engineCV {
		return outcome
	}
	var result vlmResult
	err := error(vlmFail("config", "VLM 실행기가 준비되지 않았습니다", nil))
	if plan.runVLM != nil {
		result, err = plan.runVLM(ctx, img)
	}
	if err == nil {
		outcome.Details["vlmSeats"] = len(result.Objects)
		outcome.Details["vlmRawSeats"] = result.RawSeats
		outcome.Details["vlmCoordinates"] = result.Convention
		outcome.Details["vlmTiles"] = result.Tiles
		if plan.model != "" {
			outcome.Details["vlmModel"] = plan.model
		}
		outcome.Warnings = append(outcome.Warnings, result.Warnings...)
		if plan.engine == engineVLM {
			outcome.Objects = result.Objects
			return outcome
		}
		fused, stats := fuseDetections(outcome.Objects, result.Objects, plan.fusionIoU, outcome.Grid)
		outcome.Objects = fused
		for key, value := range stats {
			outcome.Details[key] = value
		}
		// 두 엔진이 각각 좌석을 찾았는데 하나도 겹치지 않으면 좌표계나 모델 문제다.
		if stats["agreed"] == 0 && stats["cvOnly"] > 0 && (stats["vlmOnly"] > 0 || stats["vlmOffLattice"] > 0) {
			outcome.Warnings = append(outcome.Warnings,
				"CV와 VLM 결과가 한 좌석도 겹치지 않았습니다. 일치 판정 IoU를 낮추거나 VLM 모델을 확인하세요")
		}
		if stats["vlmOffLattice"] > 0 {
			outcome.Warnings = append(outcome.Warnings, fmt.Sprintf(
				"VLM이 책상 격자를 벗어난 위치에 보고한 %d건은 오검출로 보아 제외했습니다", stats["vlmOffLattice"]))
		}
		if stats["vlmOnly"] > stats["agreed"] {
			outcome.Warnings = append(outcome.Warnings, fmt.Sprintf(
				"VLM만 찾은 좌석 %d건이 교차 검증된 %d건보다 많습니다. 오검출이 섞였을 수 있으니 검토 목록을 확인하세요",
				stats["vlmOnly"], stats["agreed"]))
		}
		return outcome
	}
	// 여기까지 왔다면 VLM 경로가 실패했다.
	message := "VLM 분석 실패"
	var typed *vlmError
	if errors.As(err, &typed) {
		message = typed.Message
		outcome.Details["vlmErrorKind"] = typed.Kind
	} else {
		message = err.Error()
	}
	outcome.Details["fallback"] = engineCV
	if plan.engine == engineVLM {
		outcome.Warnings = append(outcome.Warnings, message+" · CV 분석으로 대체했습니다")
		detectWithCV(img, &outcome)
		return outcome
	}
	outcome.Warnings = append(outcome.Warnings, message+" · CV 결과만 사용했습니다")
	return outcome
}

// offLattice는 CV가 복원한 격자에서 벗어난 상자인지 본다. 격자가 뚜렷한 도면에서
// 격자 밖에 홀로 놓인 VLM 상자는 실측에서 대부분 환각이었다.
func offLattice(grid *seatGrid, object detectedObject) bool {
	if grid == nil || !grid.valid() {
		return false
	}
	residual := func(value, origin, pitch float64) float64 {
		steps := (value - origin) / pitch
		return math.Abs(steps - math.Round(steps))
	}
	return residual(object.X, grid.OriginX, grid.PitchX) > .35 ||
		residual(object.Y, grid.OriginY, grid.PitchY) > .35
}

// fuseDetections는 CV와 VLM 결과를 IoU로 짝지어 신뢰도를 재산정한다. 기하학은
// 실제 잉크에서 나온 CV 상자를 우선하고, 두 엔진이 합의한 좌석만 높은 신뢰도를
// 준다. CV가 격자를 세운 도면에서는 격자를 벗어난 VLM 단독 상자를 버려
// 오검출이 좌석으로 늘어나는 것을 막는다.
func fuseDetections(cvObjects, vlmObjects []detectedObject, iou float64, grid *seatGrid) ([]detectedObject, map[string]int) {
	stats := map[string]int{"agreed": 0, "cvOnly": 0, "vlmOnly": 0, "vlmOffLattice": 0}
	matched := make([]bool, len(vlmObjects))
	fused := make([]detectedObject, 0, len(cvObjects)+len(vlmObjects))
	for _, cvObject := range cvObjects {
		bestIndex, bestScore := -1, iou
		for index, vlmObject := range vlmObjects {
			if matched[index] {
				continue
			}
			if score := intersectionOverUnion(cvObject, vlmObject); score > bestScore {
				bestIndex, bestScore = index, score
			}
		}
		if bestIndex < 0 {
			cvObject.Confidence = math.Min(.99, cvObject.Confidence*fusionCVOnlyFactor)
			if cvObject.Source == "" {
				cvObject.Source = "cv"
			}
			fused = append(fused, cvObject)
			stats["cvOnly"]++
			continue
		}
		matched[bestIndex] = true
		agreement := (bestScore - iou) / math.Max(.0001, 1-iou)
		cvObject.Confidence = math.Min(.99, math.Max(cvObject.Confidence,
			fusionAgreementBase+.09*math.Min(1, agreement)))
		cvObject.Source = "cv+vlm"
		fused = append(fused, cvObject)
		stats["agreed"]++
	}
	for index, vlmObject := range vlmObjects {
		if matched[index] {
			continue
		}
		if offLattice(grid, vlmObject) {
			stats["vlmOffLattice"]++
			continue
		}
		vlmObject.Confidence = math.Min(fusionVLMOnlyCap, vlmObject.Confidence)
		vlmObject.Source = "vlm"
		fused = append(fused, vlmObject)
		stats["vlmOnly"]++
	}
	// 두 엔진이 미세하게 어긋난 중복을 마지막으로 한 번 더 정리한다.
	fused = suppressOverlaps(fused, .6)
	sort.Slice(fused, func(i, j int) bool {
		if math.Abs(fused[i].Y-fused[j].Y) > .02 {
			return fused[i].Y < fused[j].Y
		}
		return fused[i].X < fused[j].X
	})
	return fused, stats
}

// analysisJobStatus는 프런트가 진행 상황을 폴링할 수 있게 잡 상태를 돌려준다.
func (s *Server) analysisJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	var status, engine, mapID string
	var detected, review int
	var failure *string
	var warnings []string
	var details []byte
	var createdAt time.Time
	var completedAt *time.Time
	err := s.db.QueryRow(r.Context(), `SELECT floor_map_id,status,engine,detected_count,review_count,error,warnings,details,created_at,completed_at FROM analysis_jobs WHERE id=$1`, jobID).
		Scan(&mapID, &status, &engine, &detected, &review, &failure, &warnings, &details, &createdAt, &completedAt)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	if warnings == nil {
		warnings = []string{}
	}
	payload := map[string]any{
		"jobId": jobID, "floorMapId": mapID, "status": status, "engine": engine,
		"detected": detected, "needsReview": review, "warnings": warnings,
		"createdAt": createdAt, "completedAt": completedAt,
	}
	if failure != nil {
		payload["error"] = *failure
	}
	if len(details) > 0 {
		var parsed map[string]any
		if json.Unmarshal(details, &parsed) == nil {
			payload["details"] = parsed
			if message, ok := parsed["message"].(string); ok {
				payload["message"] = message
			}
		}
	}
	writeJSON(w, http.StatusOK, payload)
}

// runAnalysis는 요청과 분리된 컨텍스트에서 실제 분석을 수행한다. VLM 호출은
// 수십 초가 걸릴 수 있어 HTTP 응답을 붙잡아 둘 수 없기 때문이다.
func (s *Server) runAnalysis(ctx context.Context, jobID, mapID, engine string, threshold, autoThreshold float64, actorID string) {
	defer s.analyses.release(mapID)
	defer func() {
		if v := recover(); v != nil {
			s.logger.Error("분석 잡에서 패닉", "error", v, "jobId", jobID)
			s.failAnalysis(ctx, jobID, mapID, errors.New("분석 처리 중 내부 오류가 발생했습니다"))
		}
	}()
	if _, err := s.db.Exec(ctx, `UPDATE analysis_jobs SET status='running' WHERE id=$1`, jobID); err != nil {
		s.logger.Error("분석 잡 상태 갱신 실패", "error", err, "jobId", jobID)
	}
	var data, preview []byte
	var contentType string
	err := s.db.QueryRow(ctx, `SELECT file_data,content_type,preview_data FROM floor_maps WHERE id=$1`, mapID).Scan(&data, &contentType, &preview)
	if err != nil {
		s.failAnalysis(ctx, jobID, mapID, err)
		return
	}
	img, err := s.overlayImage(ctx, mapID, data, preview, contentType)
	if err != nil {
		s.failAnalysis(ctx, jobID, mapID, err)
		return
	}
	outcome := runDetection(ctx, s.detectionPlanFor(ctx, engine), img)
	prefix := "SEAT-"
	_ = s.db.QueryRow(ctx, `SELECT b.code||'-'||f.code||'-' FROM floor_maps m JOIN floors f ON f.id=m.floor_id JOIN buildings b ON b.id=f.building_id WHERE m.id=$1`, mapID).Scan(&prefix)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.failAnalysis(ctx, jobID, mapID, err)
		return
	}
	defer tx.Rollback(ctx)
	// 배정되지 않은 이전 자동 후보만 걷어낸다. 사람이 만든 좌석은 건드리지 않는다.
	if _, err = tx.Exec(ctx, `DELETE FROM seats WHERE floor_map_id=$1 AND confidence IS NOT NULL AND NOT EXISTS(SELECT 1 FROM seat_assignments a WHERE a.seat_id=seats.id)`, mapID); err != nil {
		s.failAnalysis(ctx, jobID, mapID, err)
		return
	}
	created, review, filled, bySource := 0, 0, 0, map[string]int{}
	for _, object := range outcome.Objects {
		if object.Confidence < threshold {
			continue
		}
		created++
		if object.Confidence < autoThreshold {
			review++
		}
		if object.Filled {
			filled++
		}
		source := object.Source
		if source == "" {
			source = outcome.Engine
		}
		bySource[source]++
		metadata, _ := json.Marshal(map[string]any{
			"source": source, "engine": outcome.Engine, "detector": detectionEngine,
			"row": object.Row, "col": object.Col, "gridFilled": object.Filled,
		})
		seatNo := fmt.Sprintf("%s%03d", prefix, created)
		if _, err = tx.Exec(ctx, `INSERT INTO seats(id,floor_map_id,seat_no,x,y,width,height,confidence,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(floor_map_id,seat_no) DO NOTHING`,
			newID(), mapID, seatNo, object.X, object.Y, object.W, object.H, object.Confidence, metadata); err != nil {
			s.failAnalysis(ctx, jobID, mapID, err)
			return
		}
	}
	// 관리자가 직접 보정한 격자만 보호하고, 이전 분석이 남긴 자동 격자는 갱신한다.
	// 잘못 추론된 격자를 영구 보존하면 격자 정렬이 좌석을 엉뚱한 자리로 옮긴다.
	if outcome.Grid != nil {
		outcome.Grid.Source = gridSourceAuto
		raw, _ := json.Marshal(outcome.Grid)
		if _, err = tx.Exec(ctx, `UPDATE floor_maps SET grid=$2 WHERE id=$1 AND COALESCE(grid->>'source',$3) <> $4`,
			mapID, raw, gridSourceAuto, gridSourceManual); err != nil {
			s.failAnalysis(ctx, jobID, mapID, err)
			return
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE floor_maps SET status='review' WHERE id=$1`, mapID); err != nil {
		s.failAnalysis(ctx, jobID, mapID, err)
		return
	}
	warnings := outcome.Warnings
	if created == 0 {
		warnings = append(warnings, "신뢰도 기준을 넘은 좌석이 없습니다. 기준값을 낮추거나 좌석 일괄 생성을 사용하세요")
	}
	details := map[string]any{}
	for key, value := range outcome.Details {
		details[key] = value
	}
	details["bySource"] = bySource
	details["threshold"] = threshold
	details["message"] = detectionMessage(outcome, created, filled, bySource)
	detailRaw, _ := json.Marshal(details)
	if warnings == nil {
		warnings = []string{}
	}
	if _, err = tx.Exec(ctx, `UPDATE analysis_jobs SET status='completed',detected_count=$2,review_count=$3,warnings=$4,details=$5,completed_at=now() WHERE id=$1`,
		jobID, created, review, warnings, detailRaw); err != nil {
		s.failAnalysis(ctx, jobID, mapID, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		s.failAnalysis(ctx, jobID, mapID, err)
		return
	}
	s.audit(ctx, actorID, "floor_map.analyze", "floor_map", mapID, "", map[string]any{
		"engine": outcome.Engine, "detector": detectionEngine, "detected": created,
		"review": review, "gridFilled": filled, "bySource": bySource, "warnings": warnings,
	})
	s.logger.Info("도면 분석 완료", "jobId", jobID, "floorMapId", mapID, "engine", outcome.Engine, "detected", created, "review", review)
}

func detectionMessage(outcome detectionOutcome, created, filled int, bySource map[string]int) string {
	if created == 0 {
		return "자동 인식 후보가 없습니다. 좌석 일괄 생성 도구로 보정해 주세요."
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "%d개의 좌석 후보를 생성했습니다.", created)
	switch outcome.Engine {
	case engineVLM:
		builder.WriteString(" 비전 모델 판독 결과이므로 모두 검토 대상입니다.")
	case engineHybrid:
		fmt.Fprintf(&builder, " CV·VLM 교차 검증 %d건, CV 단독 %d건, VLM 단독 %d건입니다.",
			bySource["cv+vlm"], bySource["cv"]+bySource["grid-fill"], bySource["vlm"])
	default:
		if outcome.Lattice {
			builder.WriteString(" 책상 배치 격자를 찾아 도면 격자 보정값에 반영했습니다.")
		} else {
			builder.WriteString(" 반복 격자를 찾지 못해 모든 후보를 검토 대상으로 남겼습니다.")
		}
	}
	if filled > 0 {
		fmt.Fprintf(&builder, " 격자에서 빠진 %d곳은 도면에 흔적이 있는 자리만 채웠으니 반드시 확인해 주세요.", filled)
	}
	return builder.String()
}

// reclaimStaleAnalyses는 프로세스가 재시작되어 끊긴 잡을 정리한다. 그대로 두면
// 도면이 영구히 analyzing 상태에 갇혀 재분석이 막힌다.
func (s *Server) reclaimStaleAnalyses(ctx context.Context) {
	// Go의 Duration 문자열("30m0s")은 Postgres interval 리터럴이 아니므로
	// 분 단위 정수로 넘겨 make_interval로 조립한다.
	rows, err := s.db.Query(ctx, `UPDATE analysis_jobs SET status='failed',error=COALESCE(error,'서비스가 재시작되어 분석이 중단되었습니다'),completed_at=now()
	WHERE status IN ('queued','running') AND created_at < now() - make_interval(mins => $1) RETURNING floor_map_id`,
		int(analysisJobTimeout.Minutes()))
	if err != nil {
		s.logger.Error("중단된 분석 잡 정리 실패", "error", err)
		return
	}
	mapIDs := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			mapIDs = append(mapIDs, id)
		}
	}
	rows.Close()
	if len(mapIDs) == 0 {
		return
	}
	if _, err = s.db.Exec(ctx, `UPDATE floor_maps SET status='uploaded' WHERE id = ANY($1::text[]) AND status='analyzing'`, mapIDs); err != nil {
		s.logger.Error("도면 상태 복구 실패", "error", err)
		return
	}
	s.logger.Warn("중단된 분석 잡을 정리했습니다", "count", len(mapIDs))
}

// testVLM은 설정 화면에서 사내 VLM 연결과 응답 형식을 미리 확인한다.
func (s *Server) testVLM(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.vlmConfigFrom(r.Context())
	if err != nil {
		var typed *vlmError
		if errors.As(err, &typed) {
			writeError(w, http.StatusBadRequest, "vlm_config_invalid", typed.Message)
			return
		}
		writeError(w, http.StatusBadRequest, "vlm_config_invalid", err.Error())
		return
	}
	// 좌석 4개가 놓인 작은 합성 도면으로 왕복을 검증한다.
	probe := probeFloorPlan()
	ctx, cancel := context.WithTimeout(r.Context(), cfg.timeout+10*time.Second)
	defer cancel()
	probeConfig := cfg
	probeConfig.tiles = 1
	probeConfig.maxSeats = 16
	started := time.Now()
	result, err := s.detectWithVLM(ctx, probe, probeConfig)
	elapsed := time.Since(started).Milliseconds()
	if err != nil {
		var typed *vlmError
		status, code, message := http.StatusBadGateway, "vlm_unreachable", err.Error()
		if errors.As(err, &typed) {
			message = typed.Message
			switch typed.Kind {
			case "auth":
				status, code = http.StatusBadGateway, "vlm_auth_failed"
			case "config":
				status, code = http.StatusBadRequest, "vlm_config_invalid"
			case "protocol":
				status, code = http.StatusBadGateway, "vlm_protocol_error"
			case "empty":
				// 연결은 됐지만 합성 도면에서 좌석을 찾지 못한 경우다.
				writeJSON(w, http.StatusOK, map[string]any{
					"ok": false, "model": cfg.model, "endpoint": cfg.endpoint, "elapsedMs": elapsed,
					"message": "연결은 성공했지만 시험 도면에서 좌석을 찾지 못했습니다. 모델이 도면 인식에 적합한지 확인하세요",
				})
				return
			}
		}
		writeError(w, status, code, message)
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "settings.vlm_test", "settings", "", r.RemoteAddr, map[string]any{"model": cfg.model, "seats": len(result.Objects)})
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "model": cfg.model, "endpoint": cfg.endpoint, "elapsedMs": elapsed,
		"seats": len(result.Objects), "coordinates": result.Convention,
		"message": fmt.Sprintf("연결 성공 · 시험 도면에서 좌석 %d개를 인식했습니다 (%s 좌표계)", len(result.Objects), result.Convention),
	})
}

// probeFloorPlan은 연결 시험용 최소 도면이다. 벽 하나와 책상 네 개를 그린다.
func probeFloorPlan() image.Image {
	const width, height = 640, 480
	plan := image.NewGray(image.Rect(0, 0, width, height))
	for i := range plan.Pix {
		plan.Pix[i] = 255
	}
	set := func(x, y int) {
		if x >= 0 && x < width && y >= 0 && y < height {
			plan.Pix[y*plan.Stride+x] = 0
		}
	}
	rect := func(x, y, w, h int) {
		for dx := 0; dx < w; dx++ {
			set(x+dx, y)
			set(x+dx, y+h-1)
		}
		for dy := 0; dy < h; dy++ {
			set(x, y+dy)
			set(x+w-1, y+dy)
		}
	}
	rect(20, 20, width-40, height-40)
	for index := 0; index < 4; index++ {
		rect(80+(index%2)*220, 120+(index/2)*160, 120, 80)
	}
	return plan
}
