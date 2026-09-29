package app

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// seatGrid는 도면별 좌석 격자 보정값이다. 모든 값은 도면 대비 비율 좌표이며,
// 스냅과 정렬이 임의의 고정 간격 대신 실제 책상 열 간격을 따르게 하는 데 쓰인다.
type seatGrid struct {
	OriginX float64 `json:"originX"`
	OriginY float64 `json:"originY"`
	PitchX  float64 `json:"pitchX"`
	PitchY  float64 `json:"pitchY"`
	// Source는 격자의 출처다. manual(관리자 보정)은 재분석이 덮어쓰지 않고,
	// cv(자동 추론)는 다시 분석할 때 갱신된다. 잘못 추론된 격자가 영구히
	// 남지 않게 하려면 이 구분이 필요하다.
	Source string `json:"source,omitempty"`
}

const (
	gridSourceManual = "manual"
	gridSourceAuto   = "cv"
)

const (
	minGridPitch = .004
	maxGridPitch = .5
)

func (g seatGrid) valid() bool {
	return g.OriginX >= 0 && g.OriginX < 1 && g.OriginY >= 0 && g.OriginY < 1 &&
		g.PitchX >= minGridPitch && g.PitchX <= maxGridPitch &&
		g.PitchY >= minGridPitch && g.PitchY <= maxGridPitch
}

// parseSeatGrid는 저장된 jsonb를 읽는다. 값이 없거나 범위를 벗어나면 nil을 돌려
// API 응답에서 "격자 미설정"으로 나타난다.
func parseSeatGrid(raw []byte) *seatGrid {
	if len(raw) == 0 {
		return nil
	}
	var g seatGrid
	if json.Unmarshal(raw, &g) != nil || !g.valid() {
		return nil
	}
	return &g
}

// snap은 비율 좌표를 가장 가까운 격자 교점으로 옮긴다. 원점보다 앞선 좌석도
// 격자를 음의 방향으로 연장해 처리한다.
func (g seatGrid) snap(x, y float64) (float64, float64) {
	return g.OriginX + math.Round((x-g.OriginX)/g.PitchX)*g.PitchX,
		g.OriginY + math.Round((y-g.OriginY)/g.PitchY)*g.PitchY
}

func (s *Server) updateFloorMapGrid(w http.ResponseWriter, r *http.Request) {
	mapID := chi.URLParam(r, "mapID")
	var in struct {
		OriginX *float64 `json:"originX"`
		OriginY *float64 `json:"originY"`
		PitchX  *float64 `json:"pitchX"`
		PitchY  *float64 `json:"pitchY"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	u, _ := userFrom(r)
	// 네 값이 모두 비면 보정을 해제한다.
	if in.OriginX == nil && in.OriginY == nil && in.PitchX == nil && in.PitchY == nil {
		tag, err := s.db.Exec(r.Context(), `UPDATE floor_maps SET grid='{}' WHERE id=$1`, mapID)
		if err != nil {
			notFoundOrServer(w, err)
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, 404, "not_found", "도면이 없습니다")
			return
		}
		s.audit(r.Context(), u.ID, "floor_map.grid_clear", "floor_map", mapID, r.RemoteAddr, nil)
		w.WriteHeader(204)
		return
	}
	if in.OriginX == nil || in.OriginY == nil || in.PitchX == nil || in.PitchY == nil {
		writeError(w, 400, "invalid_grid", "격자 원점과 간격을 모두 지정하세요")
		return
	}
	// 이 경로는 관리자가 직접 보정한 값이므로 출처를 서버가 확정한다.
	grid := seatGrid{OriginX: *in.OriginX, OriginY: *in.OriginY, PitchX: *in.PitchX, PitchY: *in.PitchY, Source: gridSourceManual}
	if !grid.valid() {
		writeError(w, 400, "invalid_grid", "격자 원점은 0~1, 간격은 0.004~0.5 범위여야 합니다")
		return
	}
	raw, _ := json.Marshal(grid)
	tag, err := s.db.Exec(r.Context(), `UPDATE floor_maps SET grid=$2 WHERE id=$1`, mapID, raw)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, 404, "not_found", "도면이 없습니다")
		return
	}
	s.audit(r.Context(), u.ID, "floor_map.grid_update", "floor_map", mapID, r.RemoteAddr, grid)
	writeJSON(w, 200, grid)
}

// alignSeatsToGrid는 도면의 격자에 맞춰 지정한 좌석들을 정렬한다. 여러 좌석을
// 눈대중으로 맞추는 반복 작업을 없애는 것이 목적이다.
func (s *Server) alignSeatsToGrid(w http.ResponseWriter, r *http.Request) {
	mapID := chi.URLParam(r, "mapID")
	var in struct {
		SeatIDs []string `json:"seatIds"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	var gridRaw []byte
	if err := s.db.QueryRow(r.Context(), `SELECT grid FROM floor_maps WHERE id=$1`, mapID).Scan(&gridRaw); err != nil {
		notFoundOrServer(w, err)
		return
	}
	grid := parseSeatGrid(gridRaw)
	if grid == nil {
		writeError(w, 409, "grid_missing", "먼저 도면의 좌석 격자를 보정하세요")
		return
	}
	// 좌석 목록을 비우면 도면 전체를 대상으로 한다.
	ids := in.SeatIDs
	if ids == nil {
		ids = []string{}
	}
	rows, err := s.db.Query(r.Context(), `SELECT id,x,y,width,height FROM seats WHERE floor_map_id=$1 AND (cardinality($2::text[])=0 OR id = ANY($2::text[]))`, mapID, ids)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	type target struct {
		id    string
		x, y  float64
		shift float64
	}
	targets := []target{}
	for rows.Next() {
		var id string
		var x, y, width, height float64
		if rows.Scan(&id, &x, &y, &width, &height) != nil {
			continue
		}
		nx, ny := grid.snap(x, y)
		nx, ny = clampUnit(nx, 1-width), clampUnit(ny, 1-height)
		targets = append(targets, target{id, nx, ny, math.Hypot(nx-x, ny-y)})
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		notFoundOrServer(w, err)
		return
	}
	if len(targets) == 0 {
		writeError(w, 400, "no_seats", "정렬할 좌석이 없습니다")
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	for _, t := range targets {
		if _, err = tx.Exec(r.Context(), `UPDATE seats SET x=$2,y=$3,updated_at=now() WHERE id=$1`, t.id, t.x, t.y); err != nil {
			notFoundOrServer(w, err)
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		notFoundOrServer(w, err)
		return
	}
	// 이동량을 함께 돌려준다. 격자가 어긋나 있으면 정렬이 좌석을 크게 옮기는데,
	// 그 사실을 관리자가 즉시 알아챌 수 있어야 한다.
	maxShift, sumShift := 0.0, 0.0
	for _, t := range targets {
		maxShift = math.Max(maxShift, t.shift)
		sumShift += t.shift
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "seat.grid_align", "floor_map", mapID, r.RemoteAddr,
		map[string]any{"count": len(targets), "maxShift": maxShift})
	response := map[string]any{
		"aligned":      len(targets),
		"maxShift":     math.Round(maxShift*10000) / 10000,
		"averageShift": math.Round(sumShift/float64(len(targets))*10000) / 10000,
	}
	// 좌석 한 칸에 준하는 이동이면 격자 자체를 의심해야 한다.
	if maxShift > math.Min(grid.PitchX, grid.PitchY)/2 {
		response["warning"] = "일부 좌석이 격자 간격의 절반 이상 이동했습니다. 격자 보정값을 다시 확인하세요"
	}
	writeJSON(w, 200, response)
}

func clampUnit(value, maximum float64) float64 {
	return math.Max(0, math.Min(maximum, value))
}
