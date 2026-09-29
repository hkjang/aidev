package app

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type dashboardIssue struct {
	ID               string     `json:"id"`
	Kind             string     `json:"kind"`
	Severity         string     `json:"severity"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	EmployeeID       string     `json:"employeeId,omitempty"`
	EmployeeNo       string     `json:"employeeNo,omitempty"`
	EmployeeName     string     `json:"employeeName,omitempty"`
	OrganizationName string     `json:"organizationName,omitempty"`
	SeatID           string     `json:"seatId,omitempty"`
	SeatNo           string     `json:"seatNo,omitempty"`
	FloorID          string     `json:"floorId,omitempty"`
	FloorName        string     `json:"floorName,omitempty"`
	Confidence       *float64   `json:"confidence,omitempty"`
	OccurredAt       *time.Time `json:"occurredAt,omitempty"`
	Action           string     `json:"action"`
}

// actionCountKeys는 관리자가 처리해야 할 항목이다. 대시보드와 상단 배지가
// 같은 정의를 쓰도록 키 목록만 두고 SQL 은 dashboardQueries 에서 가져온다.
// 쿼리를 복사해 두면 한쪽만 고쳤을 때 배지와 카드가 조용히 어긋난다.
var actionCountKeys = []string{
	"unassignedEmployees",
	"retiredAssignments",
	"organizationMismatch",
	"lowConfidenceSeats",
}

func actionRequiredTotal(counts map[string]int) int {
	total := 0
	for _, key := range actionCountKeys {
		total += counts[key]
	}
	return total
}

// countAll은 집계 쿼리들을 제한된 동시성으로 실행한다. 무제한으로 띄우면
// 대시보드 한 번에 풀의 연결을 모두 잡아, 관련 없는 요청까지 그 뒤에 줄 선다.
// pgxpool 기본 MaxConns 는 max(4, NumCPU) 라 여유를 남겨 둔다.
const countConcurrency = 4

func (s *Server) countAll(ctx context.Context, queries map[string]string) map[string]int {
	type result struct {
		key   string
		count int
	}
	channel := make(chan result, len(queries))
	slots := make(chan struct{}, countConcurrency)
	var group sync.WaitGroup
	for key, query := range queries {
		group.Add(1)
		go func(key, query string) {
			defer group.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			var count int
			if err := s.db.QueryRow(ctx, query).Scan(&count); err != nil {
				s.logger.Error("집계 쿼리 실패", "key", key, "error", err)
				return
			}
			channel <- result{key, count}
		}(key, query)
	}
	group.Wait()
	close(channel)
	counts := map[string]int{}
	for item := range channel {
		counts[item.key] = item.count
	}
	return counts
}

// actionCount는 상단 배지 전용의 가벼운 집계다. 화면을 옮길 때마다 대시보드
// 전체(13개 쿼리)를 부르지 않도록 필요한 항목만 센다.
func (s *Server) actionCount(w http.ResponseWriter, r *http.Request) {
	subset := map[string]string{}
	for _, key := range actionCountKeys {
		subset[key] = dashboardQueries[key]
	}
	counts := s.countAll(r.Context(), subset)
	writeJSON(w, http.StatusOK, map[string]any{
		"actionRequired": actionRequiredTotal(counts), "counts": counts,
	})
}

var dashboardQueries = map[string]string{
	"unassignedEmployees": `SELECT count(*) FROM employees e WHERE e.status='active' AND NOT EXISTS(SELECT 1 FROM seat_assignments a WHERE a.employee_id=e.id AND a.ended_at IS NULL)`,
	"unusedSeats":         `SELECT count(*) FROM seats s JOIN floor_maps m ON m.id=s.floor_map_id WHERE m.is_active AND s.status='available' AND NOT EXISTS(SELECT 1 FROM seat_assignments a WHERE a.seat_id=s.id AND a.ended_at IS NULL)`,
	"retiredAssignments":  `SELECT count(*) FROM seat_assignments a JOIN employees e ON e.id=a.employee_id WHERE a.ended_at IS NULL AND e.status='retired'`,
	"organizationMismatch": `SELECT count(*) FROM seat_assignments a JOIN employees e ON e.id=a.employee_id JOIN seats s ON s.id=a.seat_id
			WHERE a.ended_at IS NULL AND s.organization_id IS NOT NULL AND e.organization_id IS DISTINCT FROM s.organization_id`,
	"lowConfidenceSeats": `SELECT count(*) FROM seats s JOIN floor_maps m ON m.id=s.floor_map_id WHERE m.is_active AND s.confidence IS NOT NULL AND s.confidence < 0.95`,
	"totalEmployees":     `SELECT count(*) FROM employees WHERE status='active'`,
	"assignedEmployees":  `SELECT count(*) FROM employees e WHERE e.status='active' AND EXISTS(SELECT 1 FROM seat_assignments a WHERE a.employee_id=e.id AND a.ended_at IS NULL)`,
	"totalSeats":         `SELECT count(*) FROM seats s JOIN floor_maps m ON m.id=s.floor_map_id WHERE m.is_active AND s.type IN ('fixed','shared')`,
	"assignedSeats":      `SELECT count(*) FROM seats s JOIN floor_maps m ON m.id=s.floor_map_id WHERE m.is_active AND EXISTS(SELECT 1 FROM seat_assignments a WHERE a.seat_id=s.id AND a.ended_at IS NULL)`,
	"mapsInReview":       `SELECT count(*) FROM floor_maps WHERE status='review'`,
	"buildings":          `SELECT count(*) FROM buildings`,
	"floors":             `SELECT count(*) FROM floors`,
	"activeMaps":         `SELECT count(*) FROM floor_maps WHERE is_active`,
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	counts := s.countAll(r.Context(), dashboardQueries)
	counts["actionRequired"] = actionRequiredTotal(counts)

	oidcEnabled, _ := s.getSetting(r.Context(), "oidc.enabled")
	hrEnabled, _ := s.getSetting(r.Context(), "hr.sync_enabled")
	var lastSyncStatus string
	var lastSyncAt *time.Time
	_ = s.db.QueryRow(r.Context(), `SELECT status,completed_at FROM employee_sync_runs ORDER BY started_at DESC LIMIT 1`).Scan(&lastSyncStatus, &lastSyncAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"counts": counts,
		"readiness": map[string]any{
			"hasBuilding":     counts["buildings"] > 0,
			"hasFloor":        counts["floors"] > 0,
			"hasPublishedMap": counts["activeMaps"] > 0,
			"hasEmployees":    counts["totalEmployees"] > 0,
			"oidcEnabled":     oidcEnabled == "true",
			"hrEnabled":       hrEnabled == "true",
		},
		"integration": map[string]any{"lastSyncStatus": lastSyncStatus, "lastSyncAt": lastSyncAt},
	})
}

func (s *Server) dashboardIssues(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if parsed, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && parsed >= 1 && parsed <= 500 {
		limit = parsed
	}
	kind := r.URL.Query().Get("kind")
	items := make([]dashboardIssue, 0)
	appendRows := func(query string, scan func(pgx.Rows) (dashboardIssue, error)) error {
		rows, err := s.db.Query(r.Context(), query, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, err := scan(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
			if len(items) >= limit {
				break
			}
		}
		return rows.Err()
	}
	if kind == "" || kind == "retired_assignment" {
		err := appendRows(`SELECT a.id,e.id,e.employee_no,e.name,COALESCE(o.name,''),s.id,s.seat_no,f.id,f.name,a.assigned_at
			FROM seat_assignments a JOIN employees e ON e.id=a.employee_id JOIN seats s ON s.id=a.seat_id
			JOIN floor_maps m ON m.id=s.floor_map_id JOIN floors f ON f.id=m.floor_id LEFT JOIN organizations o ON o.id=e.organization_id
			WHERE a.ended_at IS NULL AND e.status='retired' ORDER BY a.assigned_at LIMIT $1`, func(rows pgx.Rows) (dashboardIssue, error) {
			var i dashboardIssue
			var at time.Time
			err := rows.Scan(&i.ID, &i.EmployeeID, &i.EmployeeNo, &i.EmployeeName, &i.OrganizationName, &i.SeatID, &i.SeatNo, &i.FloorID, &i.FloorName, &at)
			i.Kind = "retired_assignment"
			i.Severity = "critical"
			i.Title = "퇴직자 좌석 점유"
			i.Description = i.EmployeeName + "님의 " + i.SeatNo + " 좌석을 해제해야 합니다."
			i.OccurredAt = &at
			i.Action = "release"
			return i, err
		})
		if err != nil {
			notFoundOrServer(w, err)
			return
		}
	}
	if len(items) < limit && (kind == "" || kind == "organization_mismatch") {
		err := appendRows(`SELECT a.id,e.id,e.employee_no,e.name,COALESCE(eo.name,''),s.id,s.seat_no,f.id,f.name,a.assigned_at,COALESCE(so.name,'')
			FROM seat_assignments a JOIN employees e ON e.id=a.employee_id JOIN seats s ON s.id=a.seat_id
			JOIN floor_maps m ON m.id=s.floor_map_id JOIN floors f ON f.id=m.floor_id LEFT JOIN organizations eo ON eo.id=e.organization_id LEFT JOIN organizations so ON so.id=s.organization_id
			WHERE a.ended_at IS NULL AND s.organization_id IS NOT NULL AND e.organization_id IS DISTINCT FROM s.organization_id ORDER BY a.assigned_at LIMIT $1`, func(rows pgx.Rows) (dashboardIssue, error) {
			var i dashboardIssue
			var at time.Time
			var seatOrg string
			err := rows.Scan(&i.ID, &i.EmployeeID, &i.EmployeeNo, &i.EmployeeName, &i.OrganizationName, &i.SeatID, &i.SeatNo, &i.FloorID, &i.FloorName, &at, &seatOrg)
			i.Kind = "organization_mismatch"
			i.Severity = "warning"
			i.Title = "조직 영역 불일치"
			i.Description = i.EmployeeName + "님은 " + i.OrganizationName + " 소속이지만 좌석 영역은 " + seatOrg + "입니다."
			i.OccurredAt = &at
			i.Action = "align_organization"
			return i, err
		})
		if err != nil {
			notFoundOrServer(w, err)
			return
		}
	}
	if len(items) < limit && (kind == "" || kind == "unassigned_employee") {
		err := appendRows(`SELECT e.id,e.employee_no,e.name,COALESCE(o.name,''),e.updated_at FROM employees e LEFT JOIN organizations o ON o.id=e.organization_id
			WHERE e.status='active' AND NOT EXISTS(SELECT 1 FROM seat_assignments a WHERE a.employee_id=e.id AND a.ended_at IS NULL) ORDER BY e.updated_at DESC LIMIT $1`, func(rows pgx.Rows) (dashboardIssue, error) {
			var i dashboardIssue
			var at time.Time
			err := rows.Scan(&i.ID, &i.EmployeeNo, &i.EmployeeName, &i.OrganizationName, &at)
			i.EmployeeID = i.ID
			i.Kind = "unassigned_employee"
			i.Severity = "warning"
			i.Title = "좌석 미배정"
			i.Description = i.EmployeeName + "님에게 좌석이 배정되지 않았습니다."
			i.OccurredAt = &at
			i.Action = "assign"
			return i, err
		})
		if err != nil {
			notFoundOrServer(w, err)
			return
		}
	}
	if len(items) < limit && (kind == "" || kind == "low_confidence") {
		err := appendRows(`SELECT s.id,s.seat_no,f.id,f.name,s.confidence,s.updated_at FROM seats s JOIN floor_maps m ON m.id=s.floor_map_id JOIN floors f ON f.id=m.floor_id
			WHERE m.is_active AND s.confidence IS NOT NULL AND s.confidence<0.95 ORDER BY s.confidence,s.updated_at DESC LIMIT $1`, func(rows pgx.Rows) (dashboardIssue, error) {
			var i dashboardIssue
			var at time.Time
			err := rows.Scan(&i.ID, &i.SeatNo, &i.FloorID, &i.FloorName, &i.Confidence, &at)
			i.SeatID = i.ID
			i.Kind = "low_confidence"
			i.Severity = "info"
			i.Title = "AI 좌석 확인"
			i.Description = i.FloorName + " " + i.SeatNo + " 좌석의 인식 결과를 확인해 주세요."
			i.OccurredAt = &at
			i.Action = "approve"
			return i, err
		})
		if err != nil {
			notFoundOrServer(w, err)
			return
		}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (s *Server) resolveDashboardIssue(w http.ResponseWriter, r *http.Request) {
	kind, id := chi.URLParam(r, "kind"), chi.URLParam(r, "issueID")
	u, _ := userFrom(r)
	switch kind {
	case "retired-assignment":
		tx, err := s.db.Begin(r.Context())
		if err != nil {
			notFoundOrServer(w, err)
			return
		}
		defer tx.Rollback(r.Context())
		var employeeID, seatID string
		err = tx.QueryRow(r.Context(), `UPDATE seat_assignments SET ended_at=now() WHERE id=$1 AND ended_at IS NULL RETURNING employee_id,seat_id`, id).Scan(&employeeID, &seatID)
		if err != nil {
			notFoundOrServer(w, err)
			return
		}
		_, _ = tx.Exec(r.Context(), `UPDATE seats SET status='available',updated_at=now() WHERE id=$1`, seatID)
		_, err = tx.Exec(r.Context(), `INSERT INTO seat_history(id,employee_id,previous_seat_id,changed_by,reason,source) VALUES($1,$2,$3,$4,'처리필요에서 퇴직자 좌석 해제','dashboard')`, newID(), employeeID, seatID, u.ID)
		if err == nil {
			err = tx.Commit(r.Context())
		}
		if err != nil {
			notFoundOrServer(w, err)
			return
		}
	case "low-confidence":
		tag, err := s.db.Exec(r.Context(), `UPDATE seats SET confidence=1,metadata=jsonb_set(metadata,'{reviewed}', 'true'::jsonb),updated_at=now() WHERE id=$1`, id)
		if err != nil || tag.RowsAffected() == 0 {
			notFoundOrServer(w, err)
			return
		}
	case "organization-mismatch":
		tag, err := s.db.Exec(r.Context(), `UPDATE seats s SET organization_id=e.organization_id,updated_at=now() FROM seat_assignments a JOIN employees e ON e.id=a.employee_id WHERE a.id=$1 AND a.seat_id=s.id AND a.ended_at IS NULL`, id)
		if err != nil || tag.RowsAffected() == 0 {
			notFoundOrServer(w, err)
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "unsupported_issue", "자동 처리할 수 없는 항목입니다")
		return
	}
	s.audit(r.Context(), u.ID, "dashboard.issue.resolve", kind, id, r.RemoteAddr, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resolveAllRetiredAssignments(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	rows, err := tx.Query(r.Context(), `SELECT a.id,a.employee_id,a.seat_id FROM seat_assignments a JOIN employees e ON e.id=a.employee_id WHERE a.ended_at IS NULL AND e.status='retired' FOR UPDATE`)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	type target struct{ id, employee, seat string }
	targets := []target{}
	for rows.Next() {
		var t target
		if rows.Scan(&t.id, &t.employee, &t.seat) == nil {
			targets = append(targets, t)
		}
	}
	rows.Close()
	for _, t := range targets {
		_, err = tx.Exec(r.Context(), `UPDATE seat_assignments SET ended_at=now() WHERE id=$1`, t.id)
		if err != nil {
			break
		}
		_, _ = tx.Exec(r.Context(), `UPDATE seats SET status='available',updated_at=now() WHERE id=$1`, t.seat)
		_, err = tx.Exec(r.Context(), `INSERT INTO seat_history(id,employee_id,previous_seat_id,changed_by,reason,source) VALUES($1,$2,$3,$4,'퇴직자 좌석 일괄 해제','dashboard_bulk')`, newID(), t.employee, t.seat, u.ID)
		if err != nil {
			break
		}
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	s.audit(r.Context(), u.ID, "dashboard.retired.release_all", "assignment", "", r.RemoteAddr, map[string]int{"count": len(targets)})
	writeJSON(w, http.StatusOK, map[string]int{"resolved": len(targets)})
}
