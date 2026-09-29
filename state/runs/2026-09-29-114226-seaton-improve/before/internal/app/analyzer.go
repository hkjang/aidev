package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

const detectionEngine = "offline-cv-v2"

type detectedObject struct {
	X, Y, W, H, Confidence float64
	Row, Col               int
	Filled                 bool
	// Source는 좌석을 찾아낸 근거다: cv, vlm, cv+vlm(교차 검증), grid-fill.
	Source string
}

// analyzeFloorMap은 분석을 큐에 넣고 즉시 202를 돌려준다. VLM 호출은 수십 초가
// 걸릴 수 있어 HTTP 응답을 붙잡고 있으면 서버 WriteTimeout과 리버스 프록시에서
// 끊긴다. 진행 상황은 analysis_jobs 를 폴링해 확인한다.
func (s *Server) analyzeFloorMap(w http.ResponseWriter, r *http.Request) {
	mapID := chi.URLParam(r, "mapID")
	var status string
	if err := s.db.QueryRow(r.Context(), `SELECT status FROM floor_maps WHERE id=$1`, mapID).Scan(&status); err != nil {
		notFoundOrServer(w, err)
		return
	}
	if status == "published" {
		writeError(w, 409, "published_map", "게시 중인 도면은 다시 분석할 수 없습니다")
		return
	}
	engine := engineCV
	if v, _ := s.getSetting(r.Context(), "ai.engine"); v != "" {
		engine = validEngine(v)
	}
	// 요청에서 엔진을 한 번만 다르게 지정할 수 있게 한다. 잘못된 값은 설정값으로 되돌린다.
	if requested := strings.TrimSpace(r.URL.Query().Get("engine")); requested != "" {
		engine = validEngine(requested)
	}
	if engine != engineCV {
		if _, err := s.vlmConfigFrom(r.Context()); err != nil {
			var typed *vlmError
			message := "VLM 설정을 확인하세요"
			if errors.As(err, &typed) {
				message = typed.Message
			}
			writeError(w, 400, "vlm_config_invalid", message)
			return
		}
	}
	threshold := .8
	if v, _ := s.getSetting(r.Context(), "ai.confidence_threshold"); v != "" {
		threshold = parseFloat(v, .8)
	}
	autoThreshold := .95
	if v, _ := s.getSetting(r.Context(), "ai.auto_approve_threshold"); v != "" {
		autoThreshold = parseFloat(v, .95)
	}
	if !s.analyses.claim(mapID) {
		writeError(w, 409, "analysis_running", "이 도면은 이미 분석 중입니다")
		return
	}
	u, _ := userFrom(r)
	jobID := newID()
	if _, err := s.db.Exec(r.Context(), `INSERT INTO analysis_jobs(id,floor_map_id,status,engine,confidence_threshold,created_by) VALUES($1,$2,'queued',$3,$4,$5)`, jobID, mapID, engine, threshold, u.ID); err != nil {
		s.analyses.release(mapID)
		notFoundOrServer(w, err)
		return
	}
	if _, err := s.db.Exec(r.Context(), `UPDATE floor_maps SET status='analyzing' WHERE id=$1`, mapID); err != nil {
		s.analyses.release(mapID)
		s.failAnalysis(r.Context(), jobID, mapID, err)
		notFoundOrServer(w, err)
		return
	}
	// 요청 컨텍스트는 응답과 함께 취소되므로 값만 물려받고 취소는 끊어낸다.
	jobCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), analysisJobTimeout)
	go func() {
		defer cancel()
		s.runAnalysis(jobCtx, jobID, mapID, engine, threshold, autoThreshold, u.ID)
	}()
	s.audit(r.Context(), u.ID, "floor_map.analyze_queued", "floor_map", mapID, r.RemoteAddr, map[string]any{"jobId": jobID, "engine": engine})
	writeJSON(w, http.StatusAccepted, map[string]any{
		"jobId": jobID, "engine": engine, "status": "queued",
		"statusUrl": "/api/v1/analysis-jobs/" + jobID,
		"message":   "도면 분석을 시작했습니다. 완료되면 결과가 표시됩니다.",
	})
}

func (s *Server) failAnalysis(ctx context.Context, jobID, mapID string, err error) {
	msg := "analysis failed"
	if err != nil {
		msg = err.Error()
	}
	_, _ = s.db.Exec(ctx, `UPDATE analysis_jobs SET status='failed',error=$2,completed_at=now() WHERE id=$1`, jobID, msg)
	_, _ = s.db.Exec(ctx, `UPDATE floor_maps SET status='failed' WHERE id=$1`, mapID)
}

// rasterizePDF는 PDF 첫 페이지를 PNG로 변환해 바이트와 픽셀 크기를 함께 돌려준다.
// 좌석 오버레이 배경과 CV 분석이 같은 래스터를 공유하도록 하는 것이 목적이다.
func rasterizePDF(ctx context.Context, data []byte) ([]byte, int, int, error) {
	dir, err := os.MkdirTemp("", "seaton-pdf-")
	if err != nil {
		return nil, 0, 0, err
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "map.pdf")
	if err = os.WriteFile(input, data, 0o600); err != nil {
		return nil, 0, 0, err
	}
	output := filepath.Join(dir, "page")
	cmd := exec.CommandContext(ctx, "pdftoppm", "-f", "1", "-singlefile", "-png", "-r", "150", input, output)
	if raw, e := cmd.CombinedOutput(); e != nil {
		return nil, 0, 0, fmt.Errorf("PDF 변환 실패: %s", strings.TrimSpace(string(raw)))
	}
	raster, err := os.ReadFile(output + ".png")
	if err != nil {
		return nil, 0, 0, err
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(raster))
	if err != nil {
		return nil, 0, 0, err
	}
	return raster, cfg.Width, cfg.Height, nil
}

// ensurePreview는 PDF 도면의 래스터 미리보기를 돌려주고, 아직 없으면 만들어 저장한다.
// 좌석 오버레이와 CV 분석이 완전히 같은 픽셀 기준을 쓰도록 보장한다.
func (s *Server) ensurePreview(ctx context.Context, mapID string, preview []byte) ([]byte, error) {
	if len(preview) > 0 {
		return preview, nil
	}
	var data []byte
	if err := s.db.QueryRow(ctx, `SELECT file_data FROM floor_maps WHERE id=$1`, mapID).Scan(&data); err != nil {
		return nil, err
	}
	raster, width, height, err := rasterizePDF(ctx, data)
	if err != nil {
		return nil, err
	}
	if _, e := s.db.Exec(ctx, `UPDATE floor_maps SET preview_data=$2,preview_width=$3,preview_height=$4 WHERE id=$1`, mapID, raster, width, height); e != nil {
		s.logger.Warn("도면 미리보기 저장 실패", "error", e, "floorMapId", mapID)
	}
	return raster, nil
}

// overlayImage는 좌석 좌표의 기준이 되는 래스터를 디코딩한다.
func (s *Server) overlayImage(ctx context.Context, mapID string, data, preview []byte, contentType string) (image.Image, error) {
	if strings.HasPrefix(contentType, "image/") {
		img, _, err := image.Decode(bytes.NewReader(data))
		return img, err
	}
	if contentType != "application/pdf" {
		return nil, fmt.Errorf("지원하지 않는 도면 형식입니다")
	}
	raster, err := s.ensurePreview(ctx, mapID, preview)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(raster))
}

const (
	maxDetectionSide = 1400
	maxDetectedSeats = 500
)

// binaryMap은 도면을 잉크/배경 두 값으로 나눈 작업용 비트맵이다.
type binaryMap struct {
	w, h int
	ink  []bool
}

// grayscale은 분석 비용을 억제하기 위해 긴 변을 maxDetectionSide로 줄이면서
// 8비트 명도 배열을 만든다.
func grayscale(src image.Image) ([]uint8, int, int) {
	b := src.Bounds()
	scale := 1.0
	if maxSide := math.Max(float64(b.Dx()), float64(b.Dy())); maxSide > maxDetectionSide {
		scale = maxDetectionSide / maxSide
	}
	w, h := int(float64(b.Dx())*scale), int(float64(b.Dy())*scale)
	if w < 1 || h < 1 {
		return nil, 0, 0
	}
	pix := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		sy := b.Min.Y + int(float64(y)/scale)
		for x := 0; x < w; x++ {
			sx := b.Min.X + int(float64(x)/scale)
			pix[y*w+x] = color.GrayModel.Convert(src.At(sx, sy)).(color.Gray).Y
		}
	}
	return pix, w, h
}

// otsuThreshold는 명도 히스토그램의 클래스 간 분산을 최대화하는 임계값을 찾는다.
// 스캔 도면처럼 전체가 밝거나 어두운 경우에도 선을 놓치지 않게 한다.
func otsuThreshold(pix []uint8) int {
	var hist [256]int
	for _, v := range pix {
		hist[v]++
	}
	total, sum := len(pix), 0.0
	for i, c := range hist {
		sum += float64(i) * float64(c)
	}
	best, bestVariance, weightBelow, sumBelow := 0, -1.0, 0, 0.0
	for t := 0; t < 256; t++ {
		weightBelow += hist[t]
		if weightBelow == 0 {
			continue
		}
		weightAbove := total - weightBelow
		if weightAbove == 0 {
			break
		}
		sumBelow += float64(t) * float64(hist[t])
		meanBelow := sumBelow / float64(weightBelow)
		meanAbove := (sum - sumBelow) / float64(weightAbove)
		variance := float64(weightBelow) * float64(weightAbove) * (meanBelow - meanAbove) * (meanBelow - meanAbove)
		if variance > bestVariance {
			bestVariance, best = variance, t
		}
	}
	return best
}

func binarize(pix []uint8, w, h int) *binaryMap {
	threshold := otsuThreshold(pix)
	if threshold < 60 {
		threshold = 60
	}
	if threshold > 180 {
		threshold = 180
	}
	ink := make([]bool, len(pix))
	count := 0
	// otsuThreshold는 임계값 자체를 어두운 쪽 클래스에 포함시키므로 <= 로 비교한다.
	for i, v := range pix {
		if int(v) <= threshold {
			ink[i] = true
			count++
		}
	}
	// 도면은 대부분 흰 배경이다. 잉크 비율이 비정상적으로 높으면 임계값이
	// 배경까지 삼킨 것이므로 보수적인 고정값으로 되돌린다.
	if float64(count) > .45*float64(len(pix)) {
		for i, v := range pix {
			ink[i] = v < 105
		}
	}
	return &binaryMap{w: w, h: h, ink: ink}
}

// suppressLongLines는 벽·파티션·치수선처럼 길게 이어진 획을 지운다. 이것이
// 없으면 책상 윤곽이 벽선과 이어져 하나의 거대한 덩어리가 되고, 크기 필터에
// 걸려 좌석 후보가 전부 사라진다.
func (m *binaryMap) suppressLongLines(minLength int) {
	remove := make([]bool, len(m.ink))
	scan := func(length int, at func(i int) int) {
		start := -1
		for i := 0; i <= length; i++ {
			if i < length && m.ink[at(i)] {
				if start < 0 {
					start = i
				}
				continue
			}
			if start >= 0 {
				if i-start >= minLength {
					for j := start; j < i; j++ {
						remove[at(j)] = true
					}
				}
				start = -1
			}
		}
	}
	for y := 0; y < m.h; y++ {
		row := y * m.w
		scan(m.w, func(i int) int { return row + i })
	}
	for x := 0; x < m.w; x++ {
		scan(m.h, func(i int) int { return i*m.w + x })
	}
	for i, drop := range remove {
		if drop {
			m.ink[i] = false
		}
	}
}

// dilated는 1픽셀 팽창한 사본을 만든다. 끊긴 윤곽선을 이어 붙여 하나의 책상이
// 여러 조각으로 쪼개지는 것을 막는다.
func (m *binaryMap) dilated() []bool {
	out := make([]bool, len(m.ink))
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			if !m.ink[y*m.w+x] {
				continue
			}
			for dy := -1; dy <= 1; dy++ {
				ny := y + dy
				if ny < 0 || ny >= m.h {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					nx := x + dx
					if nx >= 0 && nx < m.w {
						out[ny*m.w+nx] = true
					}
				}
			}
		}
	}
	return out
}

type component struct{ minX, minY, maxX, maxY int }

// connectedComponents는 8방향으로 이어진 덩어리의 경계 상자를 모은다.
func connectedComponents(mask []bool, w, h int) []component {
	seen := make([]bool, len(mask))
	stack := make([]int, 0, 256)
	out := []component{}
	for i := range mask {
		if seen[i] || !mask[i] {
			continue
		}
		seen[i] = true
		stack = append(stack[:0], i)
		c := component{minX: i % w, maxX: i % w, minY: i / w, maxY: i / w}
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			px, py := p%w, p/w
			if px < c.minX {
				c.minX = px
			}
			if px > c.maxX {
				c.maxX = px
			}
			if py < c.minY {
				c.minY = py
			}
			if py > c.maxY {
				c.maxY = py
			}
			for dy := -1; dy <= 1; dy++ {
				ny := py + dy
				if ny < 0 || ny >= h {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					nx := px + dx
					if nx < 0 || nx >= w {
						continue
					}
					ni := ny*w + nx
					if !seen[ni] && mask[ni] {
						seen[ni] = true
						stack = append(stack, ni)
					}
				}
			}
		}
		out = append(out, c)
	}
	return out
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// estimatePitch는 같은 행(또는 열)에 놓인 후보들의 인접 간격 중앙값으로 책상
// 반복 주기를 추정한다. 개별 사각형 인식보다 사무실 도면에서 훨씬 안정적이다.
func estimatePitch(primary, secondary []float64, band, minGap float64) float64 {
	gaps := []float64{}
	for i := range primary {
		nearest := math.Inf(1)
		for j := range primary {
			if i == j || math.Abs(secondary[i]-secondary[j]) > band {
				continue
			}
			if d := primary[j] - primary[i]; d >= minGap && d < nearest {
				nearest = d
			}
		}
		if !math.IsInf(nearest, 1) {
			gaps = append(gaps, nearest)
		}
	}
	return median(gaps)
}

// latticePhase는 주기가 주어졌을 때 관측값에 가장 잘 맞는 격자 위상을 원형
// 평균으로 구한다. 이상치 몇 개에 흔들리지 않는다.
func latticePhase(values []float64, pitch float64) float64 {
	if pitch <= 0 {
		return 0
	}
	sinSum, cosSum := 0.0, 0.0
	for _, v := range values {
		angle := 2 * math.Pi * v / pitch
		sinSum += math.Sin(angle)
		cosSum += math.Cos(angle)
	}
	phase := math.Atan2(sinSum, cosSum) * pitch / (2 * math.Pi)
	for phase < 0 {
		phase += pitch
	}
	return math.Mod(phase, pitch)
}

type seatCandidate struct {
	centerX, centerY float64
	minX, minY       int
	bw, bh           int
	density          float64
}

type detectionResult struct {
	Objects []detectedObject
	Grid    *seatGrid
	Lattice bool
	Filled  int
}

// seatCluster는 크기가 비슷한 후보 묶음과 그 묶음이 이루는 격자를 담는다.
type seatCluster struct {
	members                   []int
	medianWidth, medianHeight float64
	pitchX, pitchY            float64
	phaseX, phaseY            float64
	lattice                   bool
	score                     float64
}

// 인접 후보의 면적이 이 배수 이상 벌어지면 다른 종류의 도형으로 본다.
const sizeClusterGap = 1.8

// clusterBySize는 면적 순으로 정렬한 뒤 면적이 급증하는 지점에서 끊어 묶는다.
func clusterBySize(candidates []seatCandidate) [][]int {
	order := make([]int, len(candidates))
	for i := range order {
		order[i] = i
	}
	area := func(i int) float64 { return float64(candidates[i].bw * candidates[i].bh) }
	sort.Slice(order, func(a, b int) bool { return area(order[a]) < area(order[b]) })
	clusters := [][]int{}
	current := []int{}
	for position, index := range order {
		if position > 0 && area(index) > area(order[position-1])*sizeClusterGap {
			clusters = append(clusters, current)
			current = []int{}
		}
		current = append(current, index)
	}
	if len(current) > 0 {
		clusters = append(clusters, current)
	}
	return clusters
}

// evaluateCluster는 묶음의 대표 크기와 격자 적합도를 구한다.
func evaluateCluster(candidates []seatCandidate, members []int, w, h int) seatCluster {
	cluster := seatCluster{members: members}
	widths := make([]float64, 0, len(members))
	heights := make([]float64, 0, len(members))
	xs := make([]float64, 0, len(members))
	ys := make([]float64, 0, len(members))
	for _, index := range members {
		c := candidates[index]
		widths = append(widths, float64(c.bw))
		heights = append(heights, float64(c.bh))
		xs = append(xs, c.centerX)
		ys = append(ys, c.centerY)
	}
	cluster.medianWidth, cluster.medianHeight = median(widths), median(heights)
	if cluster.medianWidth <= 0 || cluster.medianHeight <= 0 {
		return cluster
	}
	cluster.pitchX = estimatePitch(xs, ys, cluster.medianHeight*.6, cluster.medianWidth*.5)
	cluster.pitchY = estimatePitch(ys, xs, cluster.medianWidth*.6, cluster.medianHeight*.5)
	cluster.lattice = cluster.pitchX >= cluster.medianWidth*.5 && cluster.pitchY >= cluster.medianHeight*.5 &&
		cluster.pitchX <= float64(w)/2 && cluster.pitchY <= float64(h)/2
	fit := 0.0
	if cluster.lattice {
		cluster.phaseX = latticePhase(xs, cluster.pitchX)
		cluster.phaseY = latticePhase(ys, cluster.pitchY)
		for i := range xs {
			residualX := math.Abs(xs[i]-(cluster.phaseX+math.Round((xs[i]-cluster.phaseX)/cluster.pitchX)*cluster.pitchX)) / (cluster.pitchX / 2)
			residualY := math.Abs(ys[i]-(cluster.phaseY+math.Round((ys[i]-cluster.phaseY)/cluster.pitchY)*cluster.pitchY)) / (cluster.pitchY / 2)
			fit += (2 - math.Min(1, residualX) - math.Min(1, residualY)) / 2
		}
		fit /= float64(len(xs))
	}
	// 규모와 격자 정합도를 함께 본다. 격자를 못 세운 묶음도 후보로는 남긴다.
	cluster.score = float64(len(members)) * (.4 + .6*fit)
	return cluster
}

// chooseSeatCluster는 좌석으로 볼 묶음을 고른다. 점수가 비슷하면 더 큰 도형을
// 택한다. 같은 자리에 책상과 의자가 겹쳐 있을 때 워크스테이션 외형인 책상이
// 좌석이기 때문이다.
func chooseSeatCluster(candidates []seatCandidate, w, h int) seatCluster {
	best := seatCluster{}
	for _, members := range clusterBySize(candidates) {
		if len(members) == 0 {
			continue
		}
		cluster := evaluateCluster(candidates, members, w, h)
		if cluster.medianWidth <= 0 {
			continue
		}
		if best.medianWidth <= 0 {
			best = cluster
			continue
		}
		bestArea := best.medianWidth * best.medianHeight
		area := cluster.medianWidth * cluster.medianHeight
		// 점수가 15% 안쪽으로 붙으면 더 큰 도형 쪽을 택한다.
		if cluster.score > best.score*1.15 ||
			(cluster.score > best.score*.85 && area > bestArea) {
			best = cluster
		}
	}
	return best
}

// detectSeats는 도면에서 좌석 후보를 찾는다. 전처리로 벽선을 제거해 책상을
// 분리하고, 찾아낸 후보들의 반복 주기로 격자를 복원해 신뢰도를 매기며, 격자
// 위에서 빠진 자리는 잉크가 있는 곳만 보간한다.
func detectSeats(src image.Image) detectionResult {
	pix, w, h := grayscale(src)
	if pix == nil {
		return detectionResult{}
	}
	bm := binarize(pix, w, h)
	longest := float64(w)
	if h > w {
		longest = float64(h)
	}
	bm.suppressLongLines(int(math.Max(24, longest*.18)))
	candidates := []seatCandidate{}
	for _, c := range connectedComponents(bm.dilated(), w, h) {
		// 팽창으로 넓어진 1픽셀을 되돌린다.
		minX, minY, maxX, maxY := c.minX+1, c.minY+1, c.maxX-1, c.maxY-1
		bw, bh := maxX-minX+1, maxY-minY+1
		if bw < 8 || bh < 8 || bw > w/6 || bh > h/6 {
			continue
		}
		if ratio := float64(bw) / float64(bh); ratio < .3 || ratio > 3.2 {
			continue
		}
		ink := 0
		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				if bm.ink[y*w+x] {
					ink++
				}
			}
		}
		density := float64(ink) / float64(bw*bh)
		if density < .03 || density > .9 {
			continue
		}
		candidates = append(candidates, seatCandidate{
			centerX: float64(minX) + float64(bw)/2, centerY: float64(minY) + float64(bh)/2,
			minX: minX, minY: minY, bw: bw, bh: bh, density: density,
		})
	}
	if len(candidates) == 0 {
		return detectionResult{}
	}
	// 도면에는 책상마다 의자가 딸려 있고 치수 기호·글자도 섞여 있다. 크기가
	// 비슷한 것끼리 묶은 뒤 격자를 가장 잘 이루는 묶음을 좌석으로 택한다.
	// 중앙값 하나로 걸러내면 의자 무리가 다수일 때 의자를 좌석으로 오인한다.
	best := chooseSeatCluster(candidates, w, h)
	if len(best.members) == 0 {
		return detectionResult{}
	}
	chosen := make([]seatCandidate, 0, len(best.members))
	for _, index := range best.members {
		chosen = append(chosen, candidates[index])
	}
	candidates = chosen
	medianWidth, medianHeight := best.medianWidth, best.medianHeight
	pitchX, pitchY := best.pitchX, best.pitchY
	lattice := best.lattice
	phaseX, phaseY := best.phaseX, best.phaseY
	fitOf := func(value, phase, pitch float64) (int, float64) {
		index := int(math.Round((value - phase) / pitch))
		residual := math.Abs(value-(phase+float64(index)*pitch)) / (pitch / 2)
		return index, 1 - math.Min(1, residual)
	}
	objects := make([]detectedObject, 0, len(candidates))
	occupied := map[[2]int]bool{}
	for _, c := range candidates {
		sizeScore := 1 - math.Min(1, (math.Abs(float64(c.bw)-medianWidth)/medianWidth+
			math.Abs(float64(c.bh)-medianHeight)/medianHeight)/1.2)
		densityScore := 1 - math.Min(1, math.Abs(c.density-.2)/.45)
		object := detectedObject{
			X: float64(c.minX) / float64(w), Y: float64(c.minY) / float64(h),
			W: float64(c.bw) / float64(w), H: float64(c.bh) / float64(h),
		}
		if lattice {
			col, fitX := fitOf(c.centerX, phaseX, pitchX)
			row, fitY := fitOf(c.centerY, phaseY, pitchY)
			object.Col, object.Row = col, row
			object.Confidence = .62 + .20*(fitX+fitY)/2 + .12*sizeScore + .06*densityScore
			occupied[[2]int{row, col}] = true
		} else {
			// 격자를 세울 수 없으면 자동 승인 없이 항상 검토 대상으로 남긴다.
			object.Confidence = .60 + .15*sizeScore + .08*densityScore
		}
		if object.Confidence > .99 {
			object.Confidence = .99
		}
		objects = append(objects, object)
	}
	filled := 0
	if lattice {
		filled = fillLatticeGaps(bm, &objects, occupied, phaseX, phaseY, pitchX, pitchY, medianWidth, medianHeight)
	}
	sort.Slice(objects, func(i, j int) bool {
		if lattice && objects[i].Row != objects[j].Row {
			return objects[i].Row < objects[j].Row
		}
		if !lattice && math.Abs(objects[i].Y-objects[j].Y) > .02 {
			return objects[i].Y < objects[j].Y
		}
		return objects[i].X < objects[j].X
	})
	if len(objects) > maxDetectedSeats {
		objects = objects[:maxDetectedSeats]
	}
	result := detectionResult{Objects: objects, Lattice: lattice, Filled: filled}
	if lattice {
		// 좌석 좌표는 좌상단 기준이므로 격자 원점도 중심에서 반 칸 물러난다.
		originX := math.Mod(phaseX-medianWidth/2, pitchX)
		originY := math.Mod(phaseY-medianHeight/2, pitchY)
		for originX < 0 {
			originX += pitchX
		}
		for originY < 0 {
			originY += pitchY
		}
		grid := seatGrid{
			OriginX: originX / float64(w), OriginY: originY / float64(h),
			PitchX: pitchX / float64(w), PitchY: pitchY / float64(h),
		}
		if grid.valid() {
			result.Grid = &grid
		}
	}
	return result
}

// fillLatticeGaps는 격자 위에서 후보가 빠진 자리에 잉크가 실제로 있는 경우에만
// 좌석을 채운다. 가려진 책상을 살리면서 허위 좌석을 만들지 않기 위한 타협이다.
func fillLatticeGaps(bm *binaryMap, objects *[]detectedObject, occupied map[[2]int]bool,
	phaseX, phaseY, pitchX, pitchY, medianWidth, medianHeight float64) int {
	minRow, maxRow, minCol, maxCol := math.MaxInt32, math.MinInt32, math.MaxInt32, math.MinInt32
	for _, o := range *objects {
		minRow, maxRow = minInt(minRow, o.Row), maxInt(maxRow, o.Row)
		minCol, maxCol = minInt(minCol, o.Col), maxInt(maxCol, o.Col)
	}
	limit := maxInt(4, len(*objects)/4)
	filled := 0
	for row := minRow; row <= maxRow && filled < limit; row++ {
		for col := minCol; col <= maxCol && filled < limit; col++ {
			if occupied[[2]int{row, col}] {
				continue
			}
			left := phaseX + float64(col)*pitchX - medianWidth/2
			top := phaseY + float64(row)*pitchY - medianHeight/2
			minX, minY := int(math.Round(left)), int(math.Round(top))
			bw, bh := int(math.Round(medianWidth)), int(math.Round(medianHeight))
			if minX < 0 || minY < 0 || minX+bw > bm.w || minY+bh > bm.h || bw < 1 || bh < 1 {
				continue
			}
			ink := 0
			for y := minY; y < minY+bh; y++ {
				for x := minX; x < minX+bw; x++ {
					if bm.ink[y*bm.w+x] {
						ink++
					}
				}
			}
			density := float64(ink) / float64(bw*bh)
			if density < .04 {
				continue
			}
			*objects = append(*objects, detectedObject{
				X: float64(minX) / float64(bm.w), Y: float64(minY) / float64(bm.h),
				W: float64(bw) / float64(bm.w), H: float64(bh) / float64(bm.h),
				Confidence: .80 + .08*math.Min(1, density/.2),
				Row:        row, Col: col, Filled: true,
			})
			occupied[[2]int{row, col}] = true
			filled++
		}
	}
	return filled
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
