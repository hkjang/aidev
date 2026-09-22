package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
)

var mcpVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26"}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations"`
	Scope       string         `json:"-"`
}

func mcpTools() []mcpTool {
	text := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	uuid := func(description string) map[string]any {
		return map[string]any{"type": "string", "format": "uuid", "description": description}
	}
	makeTool := func(name, description, scope string, props map[string]any, required ...string) mcpTool {
		if required == nil {
			required = []string{}
		}
		readOnly := strings.HasSuffix(scope, ":read") && name != "create_knowledge_package" && name != "export_knowledge_package"
		return mcpTool{Name: name, Description: description, Scope: scope, InputSchema: map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}, Annotations: map[string]any{"readOnlyHint": readOnly, "destructiveHint": name == "delete_document", "openWorldHint": false}}
	}
	base := []mcpTool{
		makeTool("search_documents", "현재 키의 워크스페이스에서 접근 가능한 문서를 검색하여 메타데이터와 최대 600자의 스니펫을 반환합니다. 전체 원문은 document:read 권한의 get_document로 별도 조회하세요.", "search:read", map[string]any{"query": text("검색어"), "workspace_id": uuid("생략하면 API 키의 워크스페이스"), "tag": text("태그 필터")}, "query"),
		makeTool("search_workspace", "워크스페이스 지식의 제한된 검색 스니펫과 메타데이터를 조회합니다. 데이터베이스 결과에는 database:read 권한도 필요합니다.", "search:read", map[string]any{"query": text("검색어"), "workspace_id": uuid("워크스페이스")}, "query"),
		makeTool("get_document", "Markdown 원문과 문서 속성을 조회합니다.", "document:read", map[string]any{"document_id": uuid("문서 ID")}, "document_id"),
		makeTool("create_document", "워크스페이스에 Markdown 문서를 생성합니다.", "document:write", map[string]any{"workspace_id": uuid("생략하면 API 키의 워크스페이스"), "title": text("문서 제목"), "markdown": text("Markdown 원문"), "visibility": map[string]any{"type": "string", "enum": []string{"workspace", "private"}}, "tags": map[string]any{"type": "array", "items": text("태그")}}, "title"),
		makeTool("update_document", "기존 문서를 수정합니다. 먼저 조회한 version을 제공하여 동시 수정 충돌을 방지하세요.", "document:write", map[string]any{"document_id": uuid("문서 ID"), "version": map[string]any{"type": "integer", "minimum": 1}, "title": text("문서 제목"), "markdown": text("Markdown 원문"), "tags": map[string]any{"type": "array", "items": text("태그")}}, "document_id", "version"),
		makeTool("delete_document", "문서를 휴지통으로 이동합니다. 영구 삭제하지 않습니다.", "document:write", map[string]any{"document_id": uuid("문서 ID")}, "document_id"),
		makeTool("get_backlinks", "문서를 참조하는 접근 가능한 문서 목록을 조회합니다.", "document:read", map[string]any{"document_id": uuid("문서 ID")}, "document_id"),
		makeTool("get_graph", "워크스페이스의 문서 연결 그래프를 조회합니다.", "document:read", map[string]any{"workspace_id": uuid("생략하면 API 키의 워크스페이스")}),
		makeTool("search_knowledge_at_date", "담당자가 명시한 업무 유효기간의 문서 버전만 현재 ACL로 검색합니다. 수정 시각·승인 인증이 아니며 미등록 자료는 유효하다고 추정하지 않습니다.", "document:read", map[string]any{"workspace_id": uuid("워크스페이스"), "date": text("업무 기준일 YYYY-MM-DD"), "query": text("일반 검색어·현재 조직 용어 사전 적용"), "after": uuid("이전 결과의 next_after"), "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 50}}, "date"),
		makeTool("get_document_at_date", "현재 접근 권한과 민감정보 정책으로 명시된 업무 기준일의 원문 버전을 조회합니다. hash는 내용 해시이며 사실성·승인 인증이 아닙니다.", "document:read", map[string]any{"document_id": uuid("문서"), "date": text("업무 기준일 YYYY-MM-DD"), "revision": map[string]any{"type": "integer", "minimum": 1, "description": "검색 결과 validity_revision; 등록 변경 시409"}}, "document_id", "date"),
		makeTool("get_document_impact", "현재 원문과 이전 버전을 비교해 현재 ACL 안의 명시적 역방향 의존·Runbook·본인 패키지를 확인합니다. 규칙 기반 후보이며 관련 자료를 자동 수정하지 않습니다.", "document:read", map[string]any{"document_id": uuid("변경 원문"), "from_version": map[string]any{"type": "integer", "minimum": 1}, "depth": map[string]any{"type": "integer", "minimum": 1, "maximum": 5}}, "document_id", "from_version"),
		makeTool("create_document_proposal", "원문을 바꾸지 않고 변경안을 현재 문서 열람자에게 공유합니다. 먼저 읽은 base_version과 공유 consent:true가 필요합니다. AI 도움 표시는 작성자의 분류입니다.", "document:write", map[string]any{"document_id": uuid("문서"), "base_version": map[string]any{"type": "integer", "minimum": 1}, "markdown": text("제안 Markdown"), "reason": text("변경 이유"), "provenance": map[string]any{"type": "string", "enum": []string{"human", "ai_assisted"}}, "consent": map[string]any{"type": "boolean"}}, "document_id", "base_version", "markdown", "reason", "provenance", "consent"),
		makeTool("get_document_proposal", "현재 문서 권한과 보호 정책으로 변경안·기준/현재 원문·차이·처리 이력을 조회합니다.", "document:read", map[string]any{"proposal_id": uuid("변경안")}, "proposal_id"),
		makeTool("merge_document_proposal", "차이를 확인한 변경안을 같은 기준 버전에서만 원자적으로 반영합니다. revision과 consent:true가 필요하며 문서 게시 승인은 별개입니다.", "document:write", map[string]any{"proposal_id": uuid("변경안"), "revision": map[string]any{"type": "integer", "minimum": 1}, "consent": map[string]any{"type": "boolean"}, "note": text("반영 의견")}, "proposal_id", "revision", "consent"),
		makeTool("create_knowledge_package", "목적·필수 정책·선택 근거를 예산에 맞춰 암호화 보관합니다. 문서별 최신 version과 보관 consent:true가 필요합니다. 원문은 아직 전달하지 않으며 inspect 후 대상/동의를 명시해 export하세요. 도구 실행 권한은 부여하지 않습니다.", "document:read", map[string]any{"workspace_id": uuid("워크스페이스"), "purpose": text("업무 목적"), "allowed_scope": text("허용된 검토 범위"), "model": text("사용 모델 식별자"), "token_budget": map[string]any{"type": "integer", "minimum": 512, "maximum": 262144}, "counter": map[string]any{"type": "string", "enum": []string{"estimate", "responses"}}, "consent": map[string]any{"type": "boolean"}, "model_consent": map[string]any{"type": "boolean", "description": "responses 계산에 원문 전송 동의; ai:execute도 필요"}, "documents": map[string]any{"type": "array", "minItems": 1, "maxItems": 32, "items": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"id": uuid("문서"), "version": map[string]any{"type": "integer", "minimum": 1}, "mandatory": map[string]any{"type": "boolean"}, "reason": text("포함 이유")}, "required": []string{"id", "version"}}}}, "purpose", "model", "token_budget", "counter", "consent", "documents"),
		makeTool("inspect_knowledge_package", "본인의 지식 패키지 최신성·만료·토큰 예산을 확인합니다. 원문은 반환하지 않습니다.", "document:read", map[string]any{"package_id": uuid("지식 패키지 ID")}, "package_id"),
		makeTool("export_knowledge_package", "현재 ACL·원문 버전을 다시 확인하고 완성된 지식 묶음을 반환합니다. 전달 대상과 consent:true가 필수입니다. 이미 받은 사본은 권한 변경으로 회수되지 않으며 어떤 실행 권한도 부여하지 않습니다.", "document:read", map[string]any{"package_id": uuid("지식 패키지 ID"), "target": text("전달 대상 에이전트 또는 모델"), "consent": map[string]any{"type": "boolean"}}, "package_id", "target", "consent"),
		makeTool("list_databases", "워크스페이스의 데이터베이스 목록을 조회합니다.", "database:read", map[string]any{"workspace_id": uuid("생략하면 API 키의 워크스페이스")}),
		makeTool("query_database", "접근 가능한 데이터베이스의 행 목록을 조회합니다.", "database:read", map[string]any{"database_id": uuid("데이터베이스 ID")}, "database_id"),
		makeTool("create_database_row", "데이터베이스에 속성 값을 가진 새 행을 생성합니다.", "database:write", map[string]any{"database_id": uuid("데이터베이스 ID"), "values": map[string]any{"type": "object", "description": "속성 ID를 키로 하는 값"}}, "database_id", "values"),
	}
	for i := range base {
		if base[i].Name != "create_knowledge_package" {
			continue
		}
		props := base[i].InputSchema["properties"].(map[string]any)
		props["documents"].(map[string]any)["minItems"] = 0
		props["attachments"] = map[string]any{"type": "array", "maxItems": 32, "items": packageAttachmentOpenAPISchema()}
		required := []string{}
		for _, key := range base[i].InputSchema["required"].([]string) {
			if key != "documents" {
				required = append(required, key)
			}
		}
		base[i].InputSchema["required"] = required
		base[i].Description += " 문서와 첨부 조각 합계1~32개가 필요합니다. 첨부는 현재 추출·해시·위치와 원문 ACL을 검증하며 문서 동의에 자동 포함되지 않습니다."
	}
	base = append(base, mcpTool{Name: "get_impact_exception", Description: "변경 원문과 대상의 현재 ACL에서 예외 승인 근거·과거 결정·현재 유효 여부를 읽습니다. 문서·게시·실행 권한을 변경하지 않습니다. 예외 신청/결정은 사람의 로그인 화면에서만 가능합니다.", Scope: "document:read", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"exception_id"}, "properties": map[string]any{"exception_id": map[string]any{"type": "string", "format": "uuid"}}}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}})
	base = append(base, knowledgeQuestionMCPTools()...)
	base = append(base, structuredDraftMCPTools()...)
	base = append(base, knowledgeConflictMCPTools()...)
	base = append(base, documentQueryMCPTools()...)
	base = append(base, knowledgePathMCPTools()...)
	return append(base, systemStatusMCPTools()...)
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func mcpResult(w http.ResponseWriter, id json.RawMessage, result any) {
	jsonResponse(w, 200, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}
func mcpError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	jsonResponse(w, 200, map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}})
}

func (s *Server) mcp(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	if p.TokenID == "" {
		apiError(w, 401, "MCP 연결에는 범위가 지정된 API 키가 필요합니다.")
		return
	}
	if !s.sameOrigin(r) {
		apiError(w, 403, "허용되지 않은 요청 출처입니다.")
		return
	}
	version := r.Header.Get("MCP-Protocol-Version")
	if version != "" && !slices.Contains(mcpVersions, version) {
		apiError(w, 400, "지원하는 MCP 버전: "+strings.Join(mcpVersions, ", "))
		return
	}
	if accept := r.Header.Get("Accept"); accept != "" && !strings.Contains(accept, "application/json") && !strings.Contains(accept, "*/*") {
		apiError(w, 406, "Accept에 application/json을 포함하세요.")
		return
	}
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		apiError(w, 415, "Content-Type: application/json이 필요합니다.")
		return
	}
	var in mcpRequest
	if err := decode(r, &in); err != nil {
		mcpError(w, nil, -32700, "유효한 단일 JSON-RPC 객체가 필요합니다.")
		return
	}
	if in.JSONRPC != "2.0" || in.Method == "" {
		mcpError(w, in.ID, -32600, "JSON-RPC 2.0 요청이 필요합니다.")
		return
	}
	if len(in.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	var id any
	if json.Unmarshal(in.ID, &id) != nil {
		mcpError(w, nil, -32600, "유효한 요청 ID가 필요합니다.")
		return
	}
	switch id.(type) {
	case string, float64:
	default:
		mcpError(w, nil, -32600, "요청 ID는 문자열 또는 숫자여야 합니다.")
		return
	}
	switch in.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if json.Unmarshal(in.Params, &params) != nil || params.ProtocolVersion == "" {
			mcpError(w, in.ID, -32602, "protocolVersion이 필요합니다.")
			return
		}
		selected := params.ProtocolVersion
		if !slices.Contains(mcpVersions, selected) {
			selected = mcpVersions[0]
		}
		mcpResult(w, in.ID, map[string]any{"protocolVersion": selected, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]any{"name": "madi", "version": s.Version}, "instructions": "모든 도구는 API 키의 권한, 워크스페이스, 사용자 문서 접근 권한을 적용합니다. 문서 수정 전에 최신 version을 조회하세요."})
	case "ping":
		mcpResult(w, in.ID, map[string]any{})
	case "tools/list":
		out := []mcpTool{}
		for _, tool := range mcpTools() {
			if hasIntegrationScope(p, tool.Scope) {
				out = append(out, tool)
			}
		}
		mcpResult(w, in.ID, map[string]any{"tools": out})
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if json.Unmarshal(in.Params, &params) != nil || params.Name == "" {
			mcpError(w, in.ID, -32602, "도구 이름과 arguments를 확인하세요.")
			return
		}
		var found *mcpTool
		for _, tool := range mcpTools() {
			if tool.Name == params.Name {
				found = &tool
				break
			}
		}
		if found == nil {
			mcpError(w, in.ID, -32602, "알 수 없는 도구입니다.")
			return
		}
		if !hasIntegrationScope(p, found.Scope) {
			mcpResult(w, in.ID, mcpToolResult(nil, "API 키의 권한 범위를 벗어난 도구입니다."))
			return
		}
		if params.Arguments == nil {
			params.Arguments = map[string]any{}
		}
		method, path, body, err := mcpDispatch(params.Name, params.Arguments, p.WorkspaceID)
		if err != nil {
			mcpError(w, in.ID, -32602, err.Error())
			return
		}
		encoded, _ := json.Marshal(body)
		internal, err := http.NewRequestWithContext(context.WithValue(r.Context(), integrationCountedTokenKey{}, p.TokenID), method, path, bytes.NewReader(encoded))
		if err != nil {
			mcpError(w, in.ID, -32603, "요청을 구성하지 못했습니다.")
			return
		}
		internal.Header = r.Header.Clone()
		internal.Header.Set("Content-Type", "application/json")
		internal.Header.Set("X-Madi-Request", "1")
		internal.Host = r.Host
		internal.RemoteAddr = r.RemoteAddr
		internal.TLS = r.TLS
		recorder := httptest.NewRecorder()
		s.mux.ServeHTTP(recorder, internal)
		var result any
		if json.Unmarshal(recorder.Body.Bytes(), &result) != nil {
			mcpResult(w, in.ID, mcpToolResult(nil, "도구 응답을 해석하지 못했습니다."))
			return
		}
		if recorder.Code >= 400 {
			mcpResult(w, in.ID, mcpToolResult(result, "요청을 처리하지 못했습니다."))
			return
		}
		mcpResult(w, in.ID, mcpToolResult(result, ""))
	default:
		mcpError(w, in.ID, -32601, "지원하지 않는 MCP 메서드입니다.")
	}
}

func mcpToolResult(value any, message string) map[string]any {
	if value == nil {
		value = map[string]any{"error": message}
	}
	encoded, _ := json.Marshal(value)
	return map[string]any{"content": []map[string]any{{"type": "text", "text": string(encoded)}}, "structuredContent": map[string]any{"result": value}, "isError": message != ""}
}

func mcpDispatch(name string, args map[string]any, defaultWorkspace string) (string, string, map[string]any, error) {
	wid := str(args, "workspace_id")
	if wid == "" {
		wid = defaultWorkspace
	}
	bad := func(message string) (string, string, map[string]any, error) {
		return "", "", nil, fmt.Errorf("%s", message)
	}
	for _, field := range []string{"workspace_id", "document_id", "database_id", "package_id", "proposal_id", "exception_id"} {
		if value, ok := args[field]; ok {
			id, ok := value.(string)
			if !ok || !validID(id) {
				return bad(field + "는 UUID여야 합니다.")
			}
		}
	}
	for _, tool := range mcpTools() {
		if tool.Name != name {
			continue
		}
		props := tool.InputSchema["properties"].(map[string]any)
		for key := range args {
			if _, ok := props[key]; !ok {
				return bad("지원하지 않는 매개변수: " + key)
			}
		}
		for _, key := range tool.InputSchema["required"].([]string) {
			if _, ok := args[key]; !ok {
				return bad("필수 매개변수: " + key)
			}
		}
	}
	body := map[string]any{}
	for key, value := range args {
		if key != "document_id" && key != "database_id" && key != "package_id" && key != "proposal_id" {
			body[key] = value
		}
	}
	switch name {
	case "search_documents", "search_workspace":
		query, ok := args["query"].(string)
		if !ok || strings.TrimSpace(query) == "" {
			return bad("query 검색어를 입력하세요.")
		}
		values := url.Values{"workspace_id": {wid}, "q": {query}, "tag": {str(args, "tag")}}
		if name == "search_documents" {
			values.Set("type", "document")
		}
		return "GET", "/api/v1/search?" + values.Encode(), nil, nil
	case "get_document":
		return "GET", "/api/v1/documents/" + str(args, "document_id"), nil, nil
	case "list_knowledge_paths":
		return "GET", "/api/v1/knowledge-paths?" + url.Values{"workspace_id": {wid}}.Encode(), nil, nil
	case "get_knowledge_path":
		if !validID(str(args, "path_id")) {
			return bad("지식 경로 ID를 확인하세요")
		}
		return "GET", "/api/v1/knowledge-paths/" + str(args, "path_id"), nil, nil
	case "list_document_queries":
		return "GET", "/api/v1/documents/" + str(args, "document_id") + "/queries", nil, nil
	case "execute_document_query":
		return "POST", "/api/v1/documents/" + str(args, "document_id") + "/queries/execute", body, nil
	case "get_system_status":
		return "GET", "/api/v1/documents/" + str(args, "document_id") + "/system-status", nil, nil
	case "get_impact_exception":
		return "GET", "/api/v1/knowledge/impact-exceptions/" + str(args, "exception_id"), nil, nil
	case "get_managed_question":
		if !validID(str(args, "question_id")) {
			return bad("관리 질문 ID를 확인하세요")
		}
		return "GET", "/api/v1/knowledge/questions/" + str(args, "question_id"), nil, nil
	case "list_knowledge_conflicts":
		return "GET", "/api/v1/knowledge/conflicts?" + url.Values{"workspace_id": {wid}}.Encode(), nil, nil
	case "list_structured_drafts":
		if str(args, "after") != "" && !validID(str(args, "after")) {
			return bad("목록 위치를 확인하세요")
		}
		return "GET", "/api/v1/knowledge/structured-drafts?" + url.Values{"workspace_id": {wid}, "after": {str(args, "after")}}.Encode(), nil, nil
	case "get_structured_draft":
		if !validID(str(args, "draft_id")) {
			return bad("구조화 초안 ID를 확인하세요")
		}
		return "GET", "/api/v1/knowledge/structured-drafts/" + str(args, "draft_id"), nil, nil
	case "get_knowledge_conflict":
		if !validID(str(args, "run_id")) {
			return bad("모순 검토 ID를 확인하세요")
		}
		return "GET", "/api/v1/knowledge/conflicts/" + str(args, "run_id"), nil, nil
	case "list_managed_questions":
		if str(args, "after") != "" && !validID(str(args, "after")) {
			return bad("다음 목록 위치를 확인하세요")
		}
		return "GET", "/api/v1/knowledge/questions?" + url.Values{"workspace_id": {wid}, "after": {str(args, "after")}}.Encode(), nil, nil
	case "report_system_status":
		return "POST", "/api/v1/documents/" + str(args, "document_id") + "/system-status/reports", body, nil
	case "create_document":
		body["workspace_id"] = wid
		return "POST", "/api/v1/documents", body, nil
	case "update_document":
		return "PUT", "/api/v1/documents/" + str(args, "document_id"), body, nil
	case "delete_document":
		return "DELETE", "/api/v1/documents/" + str(args, "document_id"), nil, nil
	case "get_backlinks":
		return "GET", "/api/v1/documents/" + str(args, "document_id") + "/backlinks", nil, nil
	case "get_graph":
		return "GET", "/api/v1/graph?workspace_id=" + url.QueryEscape(wid), nil, nil
	case "search_knowledge_at_date":
		values := url.Values{"workspace_id": {wid}, "date": {str(args, "date")}, "q": {str(args, "query")}}
		for _, key := range []string{"after", "limit"} {
			if v, ok := args[key]; ok {
				values.Set(key, fmt.Sprint(v))
			}
		}
		return "GET", "/api/v1/knowledge/time-search?" + values.Encode(), nil, nil
	case "get_document_at_date":
		values := url.Values{"date": {str(args, "date")}}
		if v, ok := args["revision"]; ok {
			values.Set("revision", fmt.Sprint(v))
		}
		return "GET", "/api/v1/documents/" + str(args, "document_id") + "/valid-at?" + values.Encode(), nil, nil
	case "get_document_impact":
		from, depth := number(args, "from_version", -1), number(args, "depth", 1)
		if from < 1 || from > 2147483647 || depth < 1 || depth > 5 || fmt.Sprint(args["from_version"]) != fmt.Sprint(from) {
			return bad("이전 버전과 1~5단계 깊이를 확인하세요")
		}
		if value, ok := args["depth"]; ok && fmt.Sprint(value) != fmt.Sprint(depth) {
			return bad("탐색 깊이는 1~5 정수여야 합니다")
		}
		return "GET", fmt.Sprintf("/api/v1/documents/%s/impact?from=%d&depth=%d", str(args, "document_id"), from, depth), nil, nil
	case "create_document_proposal":
		return "POST", "/api/v1/documents/" + str(args, "document_id") + "/proposals", body, nil
	case "get_document_proposal":
		return "GET", "/api/v1/knowledge/proposals/" + str(args, "proposal_id"), nil, nil
	case "merge_document_proposal":
		return "POST", "/api/v1/knowledge/proposals/" + str(args, "proposal_id") + "/merge", body, nil
	case "create_knowledge_package":
		body["workspace_id"] = wid
		return "POST", "/api/v1/knowledge/packages", body, nil
	case "inspect_knowledge_package":
		return "GET", "/api/v1/knowledge/packages/" + str(args, "package_id"), nil, nil
	case "export_knowledge_package":
		return "POST", "/api/v1/knowledge/packages/" + str(args, "package_id") + "/export", body, nil
	case "list_databases":
		return "GET", "/api/v1/databases?workspace_id=" + url.QueryEscape(wid), nil, nil
	case "query_database":
		return "GET", "/api/v1/databases/" + str(args, "database_id") + "/rows", nil, nil
	case "create_database_row":
		return "POST", "/api/v1/databases/" + str(args, "database_id") + "/rows", body, nil
	}
	return bad("지원하지 않는 도구입니다.")
}
