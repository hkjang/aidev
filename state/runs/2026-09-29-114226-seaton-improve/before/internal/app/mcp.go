package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}
type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}
type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) mcp(w http.ResponseWriter, r *http.Request) {
	if scopes, _ := r.Context().Value(apiScopesContextKey).([]string); len(scopes) > 0 && !containsString(scopes, "mcp") {
		writeError(w, 403, "insufficient_scope", "mcp 범위가 있는 API 키가 필요합니다")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req mcpRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.JSONRPC != "2.0" {
		writeJSON(w, 400, mcpResponse{JSONRPC: "2.0", ID: req.ID, Error: &mcpError{-32600, "Invalid Request"}})
		return
	}
	switch req.Method {
	case "initialize":
		writeJSON(w, 200, mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]string{"name": "SeatOn", "version": s.version}, "instructions": "SeatOn 사내 좌석 검색·현황·관리 도구입니다. 변경 도구는 미리보기 또는 명시적 권한을 요구합니다."}})
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
	case "ping":
		writeJSON(w, 200, mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}})
	case "tools/list":
		writeJSON(w, 200, mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": mcpTools()}})
	case "tools/call":
		s.mcpToolCall(w, r, req)
	default:
		writeJSON(w, 200, mcpResponse{JSONRPC: "2.0", ID: req.ID, Error: &mcpError{-32601, "Method not found"}})
	}
}

func mcpTools() []map[string]any {
	return []map[string]any{
		{"name": "search_employees", "description": "이름, 사번, 이메일 또는 조직명으로 직원을 검색하고 현재 좌석을 반환합니다.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"query": map[string]string{"type": "string", "description": "검색어"}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 50, "default": 10}}, "required": []string{"query"}}},
		{"name": "list_available_seats", "description": "건물/층의 사용 가능한 좌석을 조회합니다.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"floor_id": map[string]string{"type": "string"}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 30}}}},
		{"name": "get_floor_map", "description": "층 도면과 좌석 배치 메타데이터를 조회합니다. 이미지 원문 대신 안전한 내부 URL과 비율 좌표를 반환합니다.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"floor_id": map[string]string{"type": "string"}}, "required": []string{"floor_id"}}},
		{"name": "get_action_items", "description": "미배정, 퇴직자 점유, 조직 불일치 등 관리자 처리 필요 건수를 조회합니다.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
		{"name": "assign_seat", "description": "좌석 관리자 권한과 write 범위로 직원을 좌석에 배정합니다. 모든 변경은 감사 및 좌석 이력에 기록됩니다.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"employee_id": map[string]string{"type": "string"}, "seat_id": map[string]string{"type": "string"}, "reason": map[string]string{"type": "string"}}, "required": []string{"employee_id", "seat_id", "reason"}}},
	}
}

func (s *Server) mcpToolCall(w http.ResponseWriter, r *http.Request, req mcpRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if json.Unmarshal(req.Params, &params) != nil {
		writeJSON(w, 200, mcpResponse{JSONRPC: "2.0", ID: req.ID, Error: &mcpError{-32602, "Invalid params"}})
		return
	}
	result, err := s.executeMCPTool(r, params.Name, params.Arguments)
	if err != nil {
		// 도구를 부르는 쪽은 사람이 아니라 모델이다. 모델이 스스로 고쳐 다시 부를 수
		// 있도록 무엇이 잘못됐는지 말해 주는 문장만 내보내고, 그 밖의 내부 오류는
		// 일반 문장으로 바꾼다. 데이터베이스 오류 원문에는 테이블·제약 이름이 그대로
		// 들어 있어 그대로 내보내면 스키마가 새어 나간다.
		message := "요청을 처리하지 못했습니다"
		var fault errMCP
		if errors.As(err, &fault) {
			message = fault.Error()
		} else {
			s.logger.Error("mcp tool failed", "tool", params.Name, "error", err)
		}
		writeJSON(w, 200, mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"content": []map[string]string{{"type": "text", "text": message}}, "isError": true}})
		return
	}
	raw, _ := json.Marshal(result)
	writeJSON(w, 200, mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"content": []map[string]string{{"type": "text", "text": string(raw)}}, "structuredContent": result, "isError": false}})
}

func (s *Server) executeMCPTool(r *http.Request, name string, args map[string]any) (any, error) {
	limit := intArg(args, "limit", 20, 1, 100)
	switch name {
	case "search_employees":
		q := strings.TrimSpace(stringArg(args, "query"))
		// 스키마가 필수라고 알리는 값은 실제로도 요구해야 한다. 그러지 않으면
		// 검색어를 빠뜨린 호출이 직원 명부를 통째로 돌려준다.
		if q == "" {
			return nil, errMCP("query는 필수입니다")
		}
		rows, err := s.db.Query(r.Context(), `SELECT e.id,e.employee_no,e.name,COALESCE(o.name,''),COALESCE(se.seat_no,''),COALESCE(b.name,''),COALESCE(f.name,'') FROM employees e LEFT JOIN organizations o ON o.id=e.organization_id LEFT JOIN seat_assignments a ON a.employee_id=e.id AND a.ended_at IS NULL LEFT JOIN seats se ON se.id=a.seat_id LEFT JOIN floor_maps m ON m.id=se.floor_map_id LEFT JOIN floors f ON f.id=m.floor_id LEFT JOIN buildings b ON b.id=f.building_id WHERE e.status='active' AND (e.name ILIKE '%%'||$1||'%%' OR e.employee_no ILIKE '%%'||$1||'%%' OR e.email ILIKE '%%'||$1||'%%' OR o.name ILIKE '%%'||$1||'%%') ORDER BY e.name LIMIT $2`, q, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, no, nm, org, seat, building, floor string
			_ = rows.Scan(&id, &no, &nm, &org, &seat, &building, &floor)
			items = append(items, map[string]any{"id": id, "employeeNo": no, "name": nm, "organization": org, "seatNo": seat, "building": building, "floor": floor})
		}
		return map[string]any{"items": items}, nil
	case "list_available_seats":
		floorID := stringArg(args, "floor_id")
		rows, err := s.db.Query(r.Context(), `SELECT s.id,s.seat_no,b.name,f.name FROM seats s JOIN floor_maps m ON m.id=s.floor_map_id JOIN floors f ON f.id=m.floor_id JOIN buildings b ON b.id=f.building_id WHERE s.status='available' AND m.is_active AND ($1='' OR f.id=$1) ORDER BY b.name,f.sort_order,s.seat_no LIMIT $2`, floorID, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, no, b, f string
			_ = rows.Scan(&id, &no, &b, &f)
			items = append(items, map[string]any{"id": id, "seatNo": no, "building": b, "floor": f})
		}
		return map[string]any{"items": items}, nil
	case "get_floor_map":
		floorID := stringArg(args, "floor_id")
		if floorID == "" {
			return nil, errMCP("floor_id는 필수입니다")
		}
		var mapID, version, building, floor string
		err := s.db.QueryRow(r.Context(), `SELECT m.id,m.version,b.name,f.name FROM floor_maps m JOIN floors f ON f.id=m.floor_id JOIN buildings b ON b.id=f.building_id WHERE m.floor_id=$1 AND m.is_active`, floorID).Scan(&mapID, &version, &building, &floor)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errMCP("그 층에 게시된 도면이 없습니다. list_available_seats로 층을 먼저 확인하세요")
		}
		if err != nil {
			return nil, err
		}
		rows, err := s.db.Query(r.Context(), `SELECT s.id,s.seat_no,s.type,s.status,s.x,s.y,s.width,s.height,COALESCE(e.name,'') FROM seats s LEFT JOIN seat_assignments a ON a.seat_id=s.id AND a.ended_at IS NULL LEFT JOIN employees e ON e.id=a.employee_id WHERE s.floor_map_id=$1 ORDER BY s.seat_no`, mapID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var id, no, typ, status, employee string
			var x, y, ww, hh float64
			_ = rows.Scan(&id, &no, &typ, &status, &x, &y, &ww, &hh, &employee)
			items = append(items, map[string]any{"id": id, "seatNo": no, "type": typ, "status": status, "x": x, "y": y, "width": ww, "height": hh, "employee": employee})
		}
		return map[string]any{"id": mapID, "version": version, "building": building, "floor": floor, "contentUrl": "/api/v1/floor-maps/" + mapID + "/content", "previewUrl": "/api/v1/floor-maps/" + mapID + "/preview", "seats": items}, nil
	case "get_action_items":
		u, _ := userFrom(r)
		if !u.CanManageSeats() {
			return nil, errMCP("좌석 관리자 권한이 필요합니다")
		}
		return s.actionCounts(r)
	case "assign_seat":
		u, _ := userFrom(r)
		if !u.CanManageSeats() {
			return nil, errMCP("좌석 관리자 권한이 필요합니다")
		}
		if scopes, _ := r.Context().Value(apiScopesContextKey).([]string); len(scopes) > 0 && !containsString(scopes, "write") {
			return nil, errMCP("write 범위가 필요합니다")
		}
		emp, seat, reason := stringArg(args, "employee_id"), stringArg(args, "seat_id"), stringArg(args, "reason")
		if emp == "" || seat == "" || reason == "" {
			return nil, errMCP("employee_id, seat_id, reason은 필수입니다")
		}
		// 없는 식별자를 그대로 넣으면 외래키 위반 오류가 나고, 그 원문에는 테이블과
		// 제약 이름이 들어 있다. 무엇이 없는지 먼저 확인해 알려 준다.
		var exists bool
		if err := s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1)`, emp).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, errMCP("employee_id에 해당하는 직원이 없습니다")
		}
		if err := s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM seats WHERE id=$1)`, seat).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, errMCP("seat_id에 해당하는 좌석이 없습니다")
		}
		if err := s.performAssignment(r.Context(), u, emp, seat, reason, "mcp"); err != nil {
			return nil, errMCP(assignmentFailure(err))
		}
		s.audit(r.Context(), u.ID, "assignment.create", "seat", seat, r.RemoteAddr, map[string]string{"source": "mcp", "employeeId": emp})
		return map[string]any{"applied": true, "employeeId": emp, "seatId": seat}, nil
	default:
		return nil, errMCP("알 수 없는 도구입니다: " + name)
	}
}

type errMCP string

func (e errMCP) Error() string                      { return string(e) }
func stringArg(m map[string]any, key string) string { v, _ := m[key].(string); return v }
func intArg(m map[string]any, key string, fallback, min, max int) int {
	v, ok := m[key].(float64)
	if !ok {
		return fallback
	}
	n := int(v)
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
func (s *Server) actionCounts(r *http.Request) (map[string]int, error) {
	counts := map[string]int{}
	queries := map[string]string{"unassignedEmployees": `SELECT count(*) FROM employees e WHERE e.status='active' AND NOT EXISTS(SELECT 1 FROM seat_assignments a WHERE a.employee_id=e.id AND a.ended_at IS NULL)`, "retiredAssignments": `SELECT count(*) FROM seat_assignments a JOIN employees e ON e.id=a.employee_id WHERE a.ended_at IS NULL AND e.status='retired'`, "organizationMismatch": `SELECT count(*) FROM seat_assignments a JOIN employees e ON e.id=a.employee_id JOIN seats s ON s.id=a.seat_id WHERE a.ended_at IS NULL AND s.organization_id IS NOT NULL AND e.organization_id IS DISTINCT FROM s.organization_id`}
	for key, q := range queries {
		var count int
		if err := s.db.QueryRow(r.Context(), q).Scan(&count); err != nil {
			return nil, err
		}
		counts[key] = count
	}
	return counts, nil
}
