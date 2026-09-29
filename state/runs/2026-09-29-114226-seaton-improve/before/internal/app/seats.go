package app

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"
)

func (s *Server) listSeats(w http.ResponseWriter, r *http.Request) {
	mapID := r.URL.Query().Get("floorMapId")
	floorID := r.URL.Query().Get("floorId")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	org := r.URL.Query().Get("organizationId")
	rows, err := s.db.Query(r.Context(), `SELECT s.id,s.floor_map_id,s.seat_no,s.type,s.status,s.x,s.y,s.width,s.height,s.rotation,s.confidence,s.organization_id,COALESCE(o.name,''),e.id,COALESCE(e.employee_no,''),COALESCE(e.name,''),e.organization_id,COALESCE(eo.name,''),COALESCE(e.workplace,'')
	FROM seats s JOIN floor_maps m ON m.id=s.floor_map_id LEFT JOIN organizations o ON o.id=s.organization_id
	LEFT JOIN seat_assignments a ON a.seat_id=s.id AND a.ended_at IS NULL LEFT JOIN employees e ON e.id=a.employee_id
	LEFT JOIN organizations eo ON eo.id=e.organization_id
	WHERE ($1='' OR s.floor_map_id=$1) AND ($2='' OR m.floor_id=$2) AND ($3='' OR s.seat_no ILIKE '%%'||$3||'%%' OR e.name ILIKE '%%'||$3||'%%' OR e.employee_no ILIKE '%%'||$3||'%%') AND ($4='' OR e.organization_id=$4 OR s.organization_id=$4) ORDER BY s.seat_no`, mapID, floorID, q, org)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer rows.Close()
	items := []Seat{}
	for rows.Next() {
		var item Seat
		if rows.Scan(&item.ID, &item.FloorMapID, &item.SeatNo, &item.Type, &item.Status, &item.X, &item.Y, &item.Width, &item.Height, &item.Rotation, &item.Confidence, &item.OrganizationID, &item.OrganizationName, &item.EmployeeID, &item.EmployeeNo, &item.EmployeeName, &item.EmployeeOrganizationID, &item.EmployeeOrganizationName, &item.EmployeeWorkplace) == nil {
			items = append(items, item)
		}
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

type seatInput struct {
	FloorMapID     string  `json:"floorMapId"`
	SeatNo         string  `json:"seatNo"`
	Type           string  `json:"type"`
	Status         string  `json:"status"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	Rotation       float64 `json:"rotation"`
	OrganizationID *string `json:"organizationId"`
}

func validateSeat(in *seatInput) string {
	in.SeatNo = strings.TrimSpace(in.SeatNo)
	if in.Type == "" {
		in.Type = "fixed"
	}
	if in.Status == "" {
		in.Status = "available"
	}
	if in.Width == 0 {
		in.Width = .04
	}
	if in.Height == 0 {
		in.Height = .055
	}
	if in.FloorMapID == "" || in.SeatNo == "" {
		return "도면과 좌석번호는 필수입니다"
	}
	if in.X < 0 || in.X > 1 || in.Y < 0 || in.Y > 1 || in.Width <= 0 || in.X+in.Width > 1 || in.Height <= 0 || in.Y+in.Height > 1 {
		return "좌석 좌표는 0~1 범위여야 합니다"
	}
	return ""
}

func (s *Server) createSeat(w http.ResponseWriter, r *http.Request) {
	var in seatInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if msg := validateSeat(&in); msg != "" {
		writeError(w, 400, "invalid_seat", msg)
		return
	}
	id := newID()
	_, err := s.db.Exec(r.Context(), `INSERT INTO seats(id,floor_map_id,seat_no,type,status,x,y,width,height,rotation,organization_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, in.FloorMapID, in.SeatNo, in.Type, in.Status, in.X, in.Y, in.Width, in.Height, in.Rotation, in.OrganizationID)
	if err != nil {
		writeError(w, 409, "seat_conflict", "좌석 번호가 중복되었거나 값이 올바르지 않습니다")
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "seat.create", "seat", id, r.RemoteAddr, in)
	writeJSON(w, 201, map[string]string{"id": id})
}

func (s *Server) createSeatGrid(w http.ResponseWriter, r *http.Request) {
	var in struct {
		FloorMapID string  `json:"floorMapId"`
		Prefix     string  `json:"prefix"`
		Start      int     `json:"start"`
		Rows       int     `json:"rows"`
		Columns    int     `json:"columns"`
		X          float64 `json:"x"`
		Y          float64 `json:"y"`
		SeatWidth  float64 `json:"seatWidth"`
		SeatHeight float64 `json:"seatHeight"`
		GapX       float64 `json:"gapX"`
		GapY       float64 `json:"gapY"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Rows < 1 || in.Columns < 1 || in.Rows*in.Columns > 500 || in.FloorMapID == "" {
		writeError(w, 400, "invalid_grid", "한 번에 1~500개 좌석을 생성할 수 있습니다")
		return
	}
	if in.Start < 1 {
		in.Start = 1
	}
	if in.SeatWidth <= 0 {
		in.SeatWidth = .035
	}
	if in.SeatHeight <= 0 {
		in.SeatHeight = .05
	}
	if in.GapX < 0 {
		in.GapX = .008
	}
	if in.GapY < 0 {
		in.GapY = .012
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	ids := []string{}
	n := in.Start
	for row := 0; row < in.Rows; row++ {
		for col := 0; col < in.Columns; col++ {
			x := in.X + float64(col)*(in.SeatWidth+in.GapX)
			y := in.Y + float64(row)*(in.SeatHeight+in.GapY)
			if x+in.SeatWidth > 1 || y+in.SeatHeight > 1 {
				writeError(w, 400, "grid_out_of_bounds", "생성할 좌석이 도면 범위를 벗어납니다")
				return
			}
			id := newID()
			seatNo := fmt.Sprintf("%s%03d", in.Prefix, n)
			n++
			if _, err = tx.Exec(r.Context(), `INSERT INTO seats(id,floor_map_id,seat_no,x,y,width,height) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, in.FloorMapID, seatNo, x, y, in.SeatWidth, in.SeatHeight); err != nil {
				writeError(w, 409, "seat_conflict", "좌석 번호가 중복됩니다: "+seatNo)
				return
			}
			ids = append(ids, id)
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		notFoundOrServer(w, err)
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "seat.grid_create", "floor_map", in.FloorMapID, r.RemoteAddr, map[string]any{"count": len(ids)})
	writeJSON(w, 201, map[string]any{"ids": ids, "count": len(ids)})
}

func (s *Server) updateSeat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SeatNo         *string  `json:"seatNo"`
		Type           *string  `json:"type"`
		Status         *string  `json:"status"`
		X              *float64 `json:"x"`
		Y              *float64 `json:"y"`
		Width          *float64 `json:"width"`
		Height         *float64 `json:"height"`
		Rotation       *float64 `json:"rotation"`
		OrganizationID *string  `json:"organizationId"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	id := chi.URLParam(r, "seatID")
	tag, err := s.db.Exec(r.Context(), `UPDATE seats SET seat_no=COALESCE($2,seat_no),type=COALESCE($3,type),status=COALESCE($4,status),x=COALESCE($5,x),y=COALESCE($6,y),width=COALESCE($7,width),height=COALESCE($8,height),rotation=COALESCE($9,rotation),organization_id=CASE WHEN $10::text='' THEN NULL ELSE COALESCE($10,organization_id) END,updated_at=now() WHERE id=$1`, id, in.SeatNo, in.Type, in.Status, in.X, in.Y, in.Width, in.Height, in.Rotation, in.OrganizationID)
	if err != nil {
		writeError(w, 400, "invalid_seat", "좌석 값을 확인하세요")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, 404, "not_found", "좌석이 없습니다")
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "seat.update", "seat", id, r.RemoteAddr, in)
	w.WriteHeader(204)
}

func (s *Server) updateSeatsBulk(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Updates []struct {
			ID       string  `json:"id"`
			X        float64 `json:"x"`
			Y        float64 `json:"y"`
			Rotation float64 `json:"rotation"`
		} `json:"updates"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if len(in.Updates) < 1 || len(in.Updates) > 500 {
		writeError(w, http.StatusBadRequest, "invalid_bulk_update", "한 번에 1~500개 좌석을 이동할 수 있습니다")
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	ids := make([]string, 0, len(in.Updates))
	for _, update := range in.Updates {
		if update.ID == "" || update.X < 0 || update.X > 1 || update.Y < 0 || update.Y > 1 || update.Rotation < -360 || update.Rotation > 360 {
			writeError(w, http.StatusBadRequest, "invalid_position", "좌석 위치 또는 회전값을 확인하세요")
			return
		}
		tag, err := tx.Exec(r.Context(), `UPDATE seats SET x=$2,y=$3,rotation=$4,updated_at=now() WHERE id=$1 AND $2+width<=1 AND $3+height<=1`, update.ID, update.X, update.Y, update.Rotation)
		if err != nil || tag.RowsAffected() == 0 {
			writeError(w, http.StatusBadRequest, "position_out_of_bounds", "좌석이 도면 범위를 벗어나거나 존재하지 않습니다")
			return
		}
		ids = append(ids, update.ID)
	}
	if err = tx.Commit(r.Context()); err != nil {
		notFoundOrServer(w, err)
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "seat.bulk_move", "seat", "", r.RemoteAddr, map[string]any{"ids": ids, "count": len(ids)})
	writeJSON(w, http.StatusOK, map[string]int{"updated": len(ids)})
}

func (s *Server) deleteSeat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "seatID")
	tag, err := s.db.Exec(r.Context(), `DELETE FROM seats WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM seat_assignments a WHERE a.seat_id=seats.id AND a.ended_at IS NULL)`, id)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, 409, "seat_in_use", "배정 중인 좌석은 삭제할 수 없습니다")
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "seat.delete", "seat", id, r.RemoteAddr, nil)
	w.WriteHeader(204)
}

func (s *Server) assignSeat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EmployeeID string `json:"employeeId"`
		SeatID     string `json:"seatId"`
		Reason     string `json:"reason"`
		Source     string `json:"source"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.EmployeeID == "" || in.SeatID == "" {
		writeError(w, 400, "required_fields", "직원과 좌석을 선택하세요")
		return
	}
	if in.Source == "" {
		in.Source = "manual"
	}
	u, _ := userFrom(r)
	if err := s.performAssignment(r.Context(), u, in.EmployeeID, in.SeatID, in.Reason, in.Source); err != nil {
		writeError(w, 409, "assignment_conflict", err.Error())
		return
	}
	s.audit(r.Context(), u.ID, "assignment.create", "seat", in.SeatID, r.RemoteAddr, in)
	w.WriteHeader(204)
}

func (s *Server) performAssignment(ctx context.Context, u User, employeeID, seatID, reason, source string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var previous *string
	_ = tx.QueryRow(ctx, `SELECT seat_id FROM seat_assignments WHERE employee_id=$1 AND ended_at IS NULL FOR UPDATE`, employeeID).Scan(&previous)
	var occupied string
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT employee_id FROM seat_assignments WHERE seat_id=$1 AND ended_at IS NULL),'')`, seatID).Scan(&occupied); err != nil {
		return err
	}
	if occupied != "" && occupied != employeeID {
		return fmt.Errorf("이미 다른 직원에게 배정된 좌석입니다")
	}
	if previous != nil && *previous == seatID {
		return nil
	}
	_, err = tx.Exec(ctx, `UPDATE seat_assignments SET ended_at=now() WHERE employee_id=$1 AND ended_at IS NULL`, employeeID)
	if err != nil {
		return err
	}
	if previous != nil {
		_, _ = tx.Exec(ctx, `UPDATE seats SET status='available',updated_at=now() WHERE id=$1 AND type<>'unavailable'`, *previous)
	}
	_, err = tx.Exec(ctx, `INSERT INTO seat_assignments(id,employee_id,seat_id,assigned_by,reason,source) VALUES($1,$2,$3,$4,$5,$6)`, newID(), employeeID, seatID, u.ID, reason, source)
	if err != nil {
		return err
	}
	_, _ = tx.Exec(ctx, `UPDATE seats SET status='assigned',updated_at=now() WHERE id=$1`, seatID)
	_, err = tx.Exec(ctx, `INSERT INTO seat_history(id,employee_id,previous_seat_id,new_seat_id,changed_by,reason,source) VALUES($1,$2,$3,$4,$5,$6,$7)`, newID(), employeeID, previous, seatID, u.ID, reason, source)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) unassignSeat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "seatID")
	u, _ := userFrom(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var employeeID string
	err = tx.QueryRow(r.Context(), `UPDATE seat_assignments SET ended_at=now() WHERE seat_id=$1 AND ended_at IS NULL RETURNING employee_id`, id).Scan(&employeeID)
	if err == pgx.ErrNoRows {
		writeError(w, 404, "not_assigned", "배정된 직원이 없습니다")
		return
	}
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	_, _ = tx.Exec(r.Context(), `UPDATE seats SET status='available',updated_at=now() WHERE id=$1`, id)
	_, _ = tx.Exec(r.Context(), `INSERT INTO seat_history(id,employee_id,previous_seat_id,changed_by,reason,source) VALUES($1,$2,$3,$4,$5,'manual')`, newID(), employeeID, id, u.ID, "관리자 배정 해제")
	if tx.Commit(r.Context()) != nil {
		notFoundOrServer(w, err)
		return
	}
	s.audit(r.Context(), u.ID, "assignment.delete", "seat", id, r.RemoteAddr, nil)
	w.WriteHeader(204)
}

func (s *Server) bulkAssignments(w http.ResponseWriter, r *http.Request) {
	rows, ok := readSpreadsheet(w, r)
	if !ok {
		return
	}
	if len(rows) < 2 {
		writeError(w, 400, "empty_file", "등록할 데이터가 없습니다")
		return
	}
	u, _ := userFrom(r)
	success := 0
	failures := []map[string]any{}
	for i, row := range rows[1:] {
		if len(row) < 2 {
			continue
		}
		empNo := strings.TrimSpace(row[0])
		seatNo := strings.TrimSpace(row[1])
		if empNo == "" && seatNo == "" {
			continue
		}
		fail := func(reason string) {
			failures = append(failures, map[string]any{"row": i + 2, "employeeNo": empNo, "seatNo": seatNo, "error": reason})
		}
		var empID string
		if err := s.db.QueryRow(r.Context(), `SELECT id FROM employees WHERE employee_no=$1`, empNo).Scan(&empID); err != nil {
			fail("사번을 찾을 수 없습니다")
			continue
		}
		// 좌석 번호는 도면마다 따로 매겨진다. 게시된 도면으로 좁히지 않으면 아직
		// 검토 중인 새 버전의 좌석에 배정되어 좌석맵에는 보이지 않는다.
		var seatID string
		err := s.db.QueryRow(r.Context(), `SELECT s.id FROM seats s JOIN floor_maps m ON m.id=s.floor_map_id
                        WHERE s.seat_no=$1 AND m.is_active ORDER BY m.published_at DESC NULLS LAST LIMIT 1`, seatNo).Scan(&seatID)
		if err != nil {
			fail("게시된 도면에서 좌석 번호를 찾을 수 없습니다")
			continue
		}
		if err := s.performAssignment(r.Context(), u, empID, seatID, "일괄 등록", "bulk"); err != nil {
			fail(assignmentFailure(err))
			continue
		}
		success++
	}
	writeJSON(w, 200, map[string]any{"success": success, "failed": len(failures), "failures": failures})
}

// assignmentFailure는 일괄 배정 실패 사유를 사람이 읽을 수 있는 문장으로 바꾼다.
// 데이터베이스 오류 원문을 그대로 화면에 내보내면 관리자가 무엇을 고쳐야 할지
// 알 수 없다.
func assignmentFailure(err error) string {
	if strings.Contains(err.Error(), "이미 다른 직원") {
		return "이미 다른 직원에게 배정된 좌석입니다"
	}
	return "배정하지 못했습니다"
}

func readSpreadsheet(w http.ResponseWriter, r *http.Request) ([][]string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 15<<20)
	if err := r.ParseMultipartForm(15 << 20); err != nil {
		writeError(w, 400, "invalid_upload", "파일을 읽지 못했습니다")
		return nil, false
	}
	f, h, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "file_required", "파일을 선택하세요")
		return nil, false
	}
	defer f.Close()
	name := strings.ToLower(h.Filename)
	if strings.HasSuffix(name, ".xlsx") {
		book, err := excelize.OpenReader(f)
		if err != nil {
			writeError(w, 400, "invalid_excel", "Excel 파일을 읽지 못했습니다")
			return nil, false
		}
		defer book.Close()
		sheets := book.GetSheetList()
		if len(sheets) == 0 {
			return [][]string{}, true
		}
		rows, err := book.GetRows(sheets[0])
		if err != nil {
			writeError(w, 400, "invalid_excel", "Excel 시트를 읽지 못했습니다")
			return nil, false
		}
		return rows, true
	}
	if strings.HasSuffix(name, ".csv") {
		rows, err := csv.NewReader(f).ReadAll()
		if err != nil && err != io.EOF {
			writeError(w, 400, "invalid_csv", "CSV 파일을 읽지 못했습니다")
			return nil, false
		}
		return rows, true
	}
	writeError(w, 400, "unsupported_file", "CSV 또는 XLSX 파일만 사용할 수 있습니다")
	return nil, false
}

var _ = strconv.Itoa
var _ = time.Now
