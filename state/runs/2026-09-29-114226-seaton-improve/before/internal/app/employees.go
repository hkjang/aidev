package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) listOrganizations(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id,COALESCE(external_id,''),name,parent_id,color FROM organizations ORDER BY name`)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, external, name, color string
		var parent *string
		if rows.Scan(&id, &external, &name, &parent, &color) == nil {
			items = append(items, map[string]any{"id": id, "externalId": external, "name": name, "parentId": parent, "color": color})
		}
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (s *Server) upsertOrganization(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID         string  `json:"id"`
		ExternalID string  `json:"externalId"`
		Name       string  `json:"name"`
		ParentID   *string `json:"parentId"`
		Color      string  `json:"color"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, 400, "name_required", "조직명은 필수입니다")
		return
	}
	if in.ID == "" {
		in.ID = newID()
	}
	if in.Color == "" {
		in.Color = "#2563EB"
	}
	_, err := s.db.Exec(r.Context(), `INSERT INTO organizations(id,external_id,name,parent_id,color) VALUES($1,NULLIF($2,''),$3,$4,$5) ON CONFLICT(id) DO UPDATE SET external_id=EXCLUDED.external_id,name=EXCLUDED.name,parent_id=EXCLUDED.parent_id,color=EXCLUDED.color,updated_at=now()`, in.ID, in.ExternalID, in.Name, in.ParentID, in.Color)
	if err != nil {
		writeError(w, 409, "organization_conflict", "조직 ID 또는 외부 ID가 중복되었습니다")
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "organization.upsert", "organization", in.ID, r.RemoteAddr, in)
	writeJSON(w, 200, map[string]string{"id": in.ID})
}

func (s *Server) listEmployees(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	org := r.URL.Query().Get("organizationId")
	status := r.URL.Query().Get("status")
	assignment := r.URL.Query().Get("assignment")
	limit := 100
	if v, _ := strconv.Atoi(r.URL.Query().Get("limit")); v > 0 && v <= 500 {
		limit = v
	}
	rows, err := s.db.Query(r.Context(), `SELECT e.id,e.employee_no,e.name,COALESCE(e.email,''),e.organization_id,COALESCE(o.name,''),COALESCE(e.title,''),COALESCE(e.position,''),COALESCE(e.workplace,''),e.status,a.seat_id,COALESCE(se.seat_no,'') FROM employees e LEFT JOIN organizations o ON o.id=e.organization_id LEFT JOIN seat_assignments a ON a.employee_id=e.id AND a.ended_at IS NULL LEFT JOIN seats se ON se.id=a.seat_id WHERE ($1='' OR e.name ILIKE '%%'||$1||'%%' OR e.employee_no ILIKE '%%'||$1||'%%' OR e.email ILIKE '%%'||$1||'%%' OR o.name ILIKE '%%'||$1||'%%') AND ($2='' OR e.organization_id=$2) AND ($3='' OR e.status=$3) AND ($4='' OR ($4='assigned' AND a.seat_id IS NOT NULL) OR ($4='unassigned' AND a.seat_id IS NULL)) ORDER BY e.name LIMIT $5`, q, org, status, assignment, limit)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer rows.Close()
	items := []Employee{}
	for rows.Next() {
		var item Employee
		if rows.Scan(&item.ID, &item.EmployeeNo, &item.Name, &item.Email, &item.OrganizationID, &item.OrganizationName, &item.Title, &item.Position, &item.Workplace, &item.Status, &item.SeatID, &item.SeatNo) == nil {
			items = append(items, item)
		}
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

type employeeInput struct {
	ID                     string  `json:"id"`
	EmployeeNo             string  `json:"employeeNo"`
	Name                   string  `json:"name"`
	Email                  string  `json:"email"`
	OrganizationID         *string `json:"organizationId"`
	OrganizationExternalID string  `json:"organizationExternalId"`
	OrganizationName       string  `json:"organizationName"`
	Title                  string  `json:"title"`
	Position               string  `json:"position"`
	Workplace              string  `json:"workplace"`
	Status                 string  `json:"status"`
}

func (s *Server) saveEmployee(r *http.Request, in *employeeInput) (string, error) {
	if in.ID == "" {
		_ = s.db.QueryRow(r.Context(), `SELECT id FROM employees WHERE employee_no=$1`, in.EmployeeNo).Scan(&in.ID)
		if in.ID == "" {
			in.ID = newID()
		}
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if in.OrganizationID == nil && in.OrganizationName != "" {
		orgID := newID()
		external := in.OrganizationExternalID
		if external == "" {
			external = "import:" + strings.ToLower(strings.ReplaceAll(in.OrganizationName, " ", "-"))
		}
		err := s.db.QueryRow(r.Context(), `INSERT INTO organizations(id,external_id,name) VALUES($1,$2,$3) ON CONFLICT(external_id) DO UPDATE SET name=EXCLUDED.name,updated_at=now() RETURNING id`, orgID, external, in.OrganizationName).Scan(&orgID)
		if err != nil {
			return "", err
		}
		in.OrganizationID = &orgID
	}
	_, err := s.db.Exec(r.Context(), `INSERT INTO employees(id,employee_no,name,email,organization_id,title,position,workplace,status) VALUES($1,$2,$3,NULLIF($4,''),$5,NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),$9) ON CONFLICT(employee_no) DO UPDATE SET name=EXCLUDED.name,email=EXCLUDED.email,organization_id=EXCLUDED.organization_id,title=EXCLUDED.title,position=EXCLUDED.position,workplace=EXCLUDED.workplace,status=EXCLUDED.status,updated_at=now()`, in.ID, in.EmployeeNo, in.Name, in.Email, in.OrganizationID, in.Title, in.Position, in.Workplace, in.Status)
	return in.ID, err
}

func (s *Server) upsertEmployee(w http.ResponseWriter, r *http.Request) {
	var in employeeInput
	if !decodeJSON(w, r, &in) {
		return
	}
	in.EmployeeNo = strings.TrimSpace(in.EmployeeNo)
	in.Name = strings.TrimSpace(in.Name)
	if in.EmployeeNo == "" || in.Name == "" {
		writeError(w, 400, "required_fields", "사번과 이름은 필수입니다")
		return
	}
	id, err := s.saveEmployee(r, &in)
	if err != nil {
		writeError(w, 409, "employee_conflict", "직원 정보가 중복되었거나 올바르지 않습니다")
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "employee.upsert", "employee", id, r.RemoteAddr, in)
	writeJSON(w, 200, map[string]string{"id": id})
}

func (s *Server) importEmployees(w http.ResponseWriter, r *http.Request) {
	rows, ok := readSpreadsheet(w, r)
	if !ok {
		return
	}
	if len(rows) < 2 {
		writeError(w, 400, "empty_file", "등록할 직원이 없습니다")
		return
	}
	headers := map[string]int{}
	for i, h := range rows[0] {
		headers[strings.ToLower(strings.TrimSpace(h))] = i
	}
	find := func(row []string, names ...string) string {
		for _, name := range names {
			if i, ok := headers[name]; ok && i < len(row) {
				return strings.TrimSpace(row[i])
			}
		}
		return ""
	}
	success := 0
	failures := []map[string]any{}
	for i, row := range rows[1:] {
		in := employeeInput{EmployeeNo: find(row, "employeeno", "employee_no", "사번"), Name: find(row, "name", "이름", "성명"), Email: find(row, "email", "이메일"), OrganizationExternalID: find(row, "organizationid", "organization_id", "조직코드"), OrganizationName: find(row, "organization", "organizationname", "조직명", "부서"), Title: find(row, "title", "직급"), Position: find(row, "position", "직책"), Workplace: find(row, "workplace", "근무지"), Status: find(row, "status", "재직상태")}
		if in.Status == "재직" {
			in.Status = "active"
		} else if in.Status == "휴직" {
			in.Status = "leave"
		} else if in.Status == "퇴직" {
			in.Status = "retired"
		}
		if in.EmployeeNo == "" || in.Name == "" {
			failures = append(failures, map[string]any{"row": i + 2, "error": "사번/이름 누락"})
			continue
		}
		if _, err := s.saveEmployee(r, &in); err != nil {
			failures = append(failures, map[string]any{"row": i + 2, "error": err.Error()})
		} else {
			success++
		}
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "employee.import", "employee", "", r.RemoteAddr, map[string]int{"success": success, "failed": len(failures)})
	writeJSON(w, 200, map[string]any{"success": success, "failed": len(failures), "failures": failures})
}

// listHistory는 좌석 변경 이력을 조회한다. 감사 목적의 화면이라 사람/좌석
// 검색과 방식·기간 필터가 필요하고, 화면에서 "몇 건 중 몇 건"을 보여줄 수
// 있도록 필터에 걸린 전체 건수도 함께 돌려준다.
//
// from/to 는 시각(RFC3339)으로 받는다. 날짜만 받아 서버 시간대로 해석하면
// 사용자가 자기 시간대 기준으로 고른 "오늘"이 서버에서는 다른 날이 되어
// 방금 만든 기록이 조회되지 않는다. 경계 계산은 사용자의 시간대를 아는
// 브라우저가 맡고, 서버는 받은 구간을 그대로 쓴다. to 는 열린 구간이다.
// 이력 조회 상한. COUNT 를 이 값에서 끊어 감사 테이블이 커져도 전체 스캔이
// 되지 않게 한다. 넘어가면 화면에 "N+"로 보여준다.
const historyCountCap = 5000

// listHistory는 좌석 변경 이력을 조회한다. 감사 목적의 화면이라 사람/좌석
// 검색과 방식·기간 필터가 필요하고, 화면에서 "몇 건 중 몇 건"을 보여줄 수
// 있도록 필터에 걸린 건수도 함께 돌려준다.
//
// from/to 는 시각(RFC3339)으로 받는다. 날짜만 받아 서버 시간대로 해석하면
// 사용자가 자기 시간대 기준으로 고른 "오늘"이 서버에서는 다른 날이 되어
// 방금 만든 기록이 조회되지 않는다. 경계 계산은 사용자의 시간대를 아는
// 브라우저가 맡고, 서버는 받은 구간을 그대로 쓴다. to 는 열린 구간이다.
func (s *Server) listHistory(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v, _ := strconv.Atoi(r.URL.Query().Get("limit")); v > 0 && v <= 500 {
		limit = v
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	// 시각은 여기서 검증한다. DB 오류로 넘기면 일시적인 장애까지 "조건이 잘못됐다"로
	// 보고하게 된다.
	parseBound := func(value string) (any, bool) {
		if value == "" {
			return nil, true
		}
		at, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, false
		}
		return at, true
	}
	fromAt, okFrom := parseBound(from)
	toAt, okTo := parseBound(to)
	if !okFrom || !okTo {
		writeError(w, http.StatusBadRequest, "invalid_filter", "기간은 RFC3339 시각이어야 합니다")
		return
	}
	// 조건을 실제로 주어진 것만 붙인다. ($n='' OR ...) 형태는 인덱스를 타지 못해
	// 기간을 좁혀도 이력 전체를 훑게 된다.
	conditions := []string{}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(clause, len(args)))
	}
	if query != "" {
		add(`(e.name ILIKE '%%'||$%[1]d||'%%' OR e.employee_no ILIKE '%%'||$%[1]d||'%%'
			OR ps.seat_no ILIKE '%%'||$%[1]d||'%%' OR ns.seat_no ILIKE '%%'||$%[1]d||'%%')`, query)
	}
	if source != "" {
		add(`h.source=$%d`, source)
	}
	if fromAt != nil {
		add(`h.changed_at >= $%d`, fromAt)
	}
	if toAt != nil {
		add(`h.changed_at < $%d`, toAt)
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	const joins = `FROM seat_history h
	LEFT JOIN employees e ON e.id=h.employee_id
	LEFT JOIN seats ps ON ps.id=h.previous_seat_id
	LEFT JOIN seats ns ON ns.id=h.new_seat_id
	LEFT JOIN users u ON u.id=h.changed_by`
	// 상한까지만 세고 끊는다. 감사 테이블은 계속 쌓이므로 무제한 COUNT 는
	// 화면을 열 때마다 전체 스캔이 된다.
	total := 0
	countArgs := append(append([]any{}, args...), historyCountCap)
	countSQL := fmt.Sprintf(`SELECT count(*) FROM (SELECT 1 %s %s LIMIT $%d) capped`,
		joins, where, len(countArgs))
	if err := s.db.QueryRow(r.Context(), countSQL, countArgs...).Scan(&total); err != nil {
		notFoundOrServer(w, err)
		return
	}
	listArgs := append(append([]any{}, args...), limit)
	listSQL := fmt.Sprintf(`SELECT h.id,h.changed_at,COALESCE(e.employee_no,''),COALESCE(e.name,''),COALESCE(ps.seat_no,''),COALESCE(ns.seat_no,''),COALESCE(u.display_name,'System'),COALESCE(h.reason,''),h.source %s %s ORDER BY h.changed_at DESC LIMIT $%d`,
		joins, where, len(listArgs))
	rows, err := s.db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, employeeNo, name, previous, next, actor, reason, source string
		var changed any
		if rows.Scan(&id, &changed, &employeeNo, &name, &previous, &next, &actor, &reason, &source) == nil {
			items = append(items, map[string]any{"id": id, "changedAt": changed, "employeeNo": employeeNo, "employeeName": name, "previousSeat": previous, "newSeat": next, "actor": actor, "reason": reason, "source": source})
		}
	}
	writeJSON(w, 200, map[string]any{
		"items": items, "total": total, "limit": limit,
		// 상한에 걸리면 화면이 "5000+"처럼 표기할 수 있게 알린다.
		"totalCapped": total >= historyCountCap,
	})
}
