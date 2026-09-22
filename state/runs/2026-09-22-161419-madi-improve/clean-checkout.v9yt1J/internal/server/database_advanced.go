package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed database_advanced.sql
var databaseAdvancedSchema string

type databaseQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Server) migrateDatabaseAdvanced(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, databaseAdvancedSchema)
	return e
}
func (s *Server) registerDatabaseAdvanced() {
	s.handle("POST /api/v1/databases/{id}/query", s.queryAdvancedDatabase)
	s.handle("POST /api/v1/databases/{id}/rows/{rowId}/ai/{propertyId}", s.generateDatabaseAIProperty)
	s.handle("POST /api/v1/databases/{id}/rows/{rowId}/button/{propertyId}", s.executeDatabaseButton)
}
func isAdvancedPropertyType(kind string) bool {
	return oneOf(kind, "formula", "relation", "rollup", "ai", "created_time", "updated_time", "created_by", "updated_by", "progress", "button")
}
func advancedProperties(value any) ([]map[string]any, error) {
	var properties []map[string]any
	e := json.Unmarshal(jsonValue(value), &properties)
	return properties, e
}
func advancedPropertyID(properties []map[string]any, reference string) (string, error) {
	for _, p := range properties {
		if str(p, "id") == reference {
			return reference, nil
		}
	}
	found := ""
	for _, p := range properties {
		if str(p, "name") == reference {
			if found != "" {
				return "", fmt.Errorf("같은 이름의 속성이 있습니다. 속성 ID를 사용하세요: %s", reference)
			}
			found = str(p, "id")
		}
	}
	if found == "" {
		return "", fmt.Errorf("참조하는 속성이 없습니다: %s", reference)
	}
	return found, nil
}
func validateAdvancedProperties(properties []map[string]any) error {
	lookup := map[string]map[string]any{}
	deps := map[string][]string{}
	for _, p := range properties {
		lookup[str(p, "id")] = p
	}
	for _, p := range properties {
		id, kind := str(p, "id"), str(p, "type")
		switch kind {
		case "formula":
			expression := str(p, "expression")
			ast, e := compileFormula(expression)
			if e != nil {
				return fmt.Errorf("%s: %w", str(p, "name"), e)
			}
			for _, ref := range formulaDependencies(ast) {
				target, e := advancedPropertyID(properties, ref)
				if e != nil {
					return e
				}
				deps[id] = append(deps[id], target)
			}
		case "relation":
			if !validID(str(p, "target_database_id")) {
				return errors.New("관계 속성의 대상 데이터베이스 ID를 확인하세요")
			}
		case "rollup":
			relation, ok := lookup[str(p, "relation_property_id")]
			if !ok || str(relation, "type") != "relation" {
				return errors.New("롤업에 사용할 관계 속성을 선택하세요")
			}
			if !oneOf(str(p, "aggregation"), "count", "sum", "avg", "min", "max", "unique") {
				return errors.New("롤업 집계 함수를 확인하세요")
			}
			if str(p, "aggregation") != "count" && str(p, "target_property_id") == "" {
				return errors.New("집계할 대상 속성을 선택하세요")
			}
			deps[id] = append(deps[id], str(p, "relation_property_id"))
		case "ai":
			if len(str(p, "prompt")) == 0 || len(str(p, "prompt")) > 4096 {
				return errors.New("AI 속성 지시문은 1~4096바이트로 입력하세요")
			}
			refs, ok := p["source_property_ids"].([]any)
			if !ok || len(refs) == 0 || len(refs) > 30 {
				return errors.New("AI 속성에 사용할 원본 속성을 1~30개 선택하세요")
			}
			seen := map[string]bool{}
			for _, raw := range refs {
				ref, ok := raw.(string)
				if !ok || ref == id || lookup[ref] == nil || seen[ref] {
					return errors.New("AI 속성의 원본 속성 ID가 올바르지 않습니다")
				}
				seen[ref] = true
			}
		case "button":
			if !oneOf(str(p, "action"), "create_document") {
				return errors.New("버튼 작업은 create_document를 지원합니다")
			}
			if len(str(p, "template")) > 32000 {
				return errors.New("버튼 문서 템플릿은 32000바이트 이하여야 합니다")
			}
		}
	}
	visiting, done := map[string]bool{}, map[string]bool{}
	var walk func(string, int) error
	walk = func(id string, depth int) error {
		if depth > formulaMaxDepth {
			return errors.New("속성 참조는 32단계까지 지원합니다")
		}
		if visiting[id] {
			return fmt.Errorf("속성 참조에 순환이 있습니다: %s", str(lookup[id], "name"))
		}
		if done[id] {
			return nil
		}
		visiting[id] = true
		for _, dep := range deps[id] {
			if e := walk(dep, depth+1); e != nil {
				return e
			}
		}
		visiting[id] = false
		done[id] = true
		return nil
	}
	for id := range deps {
		if e := walk(id, 0); e != nil {
			return e
		}
	}
	return nil
}
func validateAdvancedValue(property map[string]any, value any) error {
	switch str(property, "type") {
	case "relation":
		values, ok := value.([]any)
		if !ok || len(values) > 100 {
			return errors.New("관계 값은 최대 100개의 행 ID 배열이어야 합니다")
		}
		seen := map[string]bool{}
		for _, raw := range values {
			id, ok := raw.(string)
			if !ok || !validID(id) || seen[id] {
				return errors.New("관계 행 ID가 올바르지 않거나 중복입니다")
			}
			seen[id] = true
		}
		return nil
	case "progress":
		n, e := formulaNumber(value)
		if e != nil || n < 0 || n > 100 {
			return errors.New("진행률은 0~100 사이의 숫자여야 합니다")
		}
		return nil
	case "ai":
		text, ok := value.(string)
		if !ok || len(text) > formulaMaxOutput {
			return errors.New("AI 속성은 64KB 이하 텍스트여야 합니다")
		}
		return nil
	case "formula", "rollup", "created_time", "updated_time", "created_by", "updated_by", "button":
		return errors.New("계산 속성은 직접 수정할 수 없습니다")
	}
	return errors.New("지원하지 않는 고급 속성 유형입니다")
}
func (s *Server) validateAdvancedRelations(r *http.Request, q databaseQuerier, workspaceID, databaseID string, properties []map[string]any) error {
	targets := map[string][]map[string]any{}
	for _, p := range properties {
		if str(p, "type") != "relation" {
			continue
		}
		target := str(p, "target_database_id")
		if target == databaseID {
			targets[target] = properties
			continue
		}
		var wid string
		var raw []byte
		if e := q.QueryRow(r.Context(), "SELECT workspace_id::text,properties FROM databases WHERE id=$1 FOR SHARE", target).Scan(&wid, &raw); e != nil {
			return errors.New("관계 대상 데이터베이스를 찾을 수 없습니다")
		}
		if wid != workspaceID || !s.canWorkspace(r.Context(), current(r), wid, false) || !s.canDatabase(r, target, false) {
			return errors.New("관계는 접근 가능한 같은 워크스페이스의 데이터베이스만 연결할 수 있습니다")
		}
		var props []map[string]any
		if e := json.Unmarshal(raw, &props); e != nil {
			return e
		}
		targets[target] = props
	}
	lookup := map[string]map[string]any{}
	for _, p := range properties {
		lookup[str(p, "id")] = p
	}
	for _, p := range properties {
		if str(p, "type") != "rollup" || str(p, "aggregation") == "count" {
			continue
		}
		relation := lookup[str(p, "relation_property_id")]
		targetProps := targets[str(relation, "target_database_id")]
		if _, e := advancedPropertyID(targetProps, str(p, "target_property_id")); e != nil {
			return fmt.Errorf("롤업 대상: %w", e)
		}
	}
	return nil
}
func (s *Server) validateAdvancedRowRelations(r *http.Request, q databaseQuerier, databaseID string, properties []map[string]any, values map[string]any) error {
	var wid string
	if e := q.QueryRow(r.Context(), "SELECT workspace_id::text FROM databases WHERE id=$1", databaseID).Scan(&wid); e != nil {
		return e
	}
	for _, p := range properties {
		if str(p, "type") != "relation" {
			continue
		}
		value, exists := values[str(p, "id")]
		if !exists || value == nil {
			continue
		}
		if text, ok := value.(string); ok && text == "" {
			continue
		}
		if e := validateAdvancedValue(p, value); e != nil {
			return e
		}
		ids := listStrings(value)
		if len(ids) == 0 {
			continue
		}
		target := str(p, "target_database_id")
		var targetWid string
		if e := q.QueryRow(r.Context(), "SELECT workspace_id::text FROM databases WHERE id=$1 FOR SHARE", target).Scan(&targetWid); e != nil {
			return errors.New("관계 대상 데이터베이스가 없습니다")
		}
		if targetWid != wid || !s.canWorkspace(r.Context(), current(r), targetWid, false) || !s.canDatabase(r, target, false) {
			return errors.New("관계 대상에 접근할 수 없습니다")
		}
		var count int
		if e := q.QueryRow(r.Context(), "SELECT count(*) FROM database_rows WHERE database_id=$1 AND id=ANY($2::uuid[])", target, ids).Scan(&count); e != nil {
			return e
		}
		if count != len(ids) {
			return errors.New("관계 값에는 대상 데이터베이스에 존재하는 행만 연결할 수 있습니다")
		}
	}
	return nil
}

type advancedDatabase struct {
	id, workspaceID, spaceID string
	properties               []map[string]any
	lookup                   map[string]map[string]any
}
type advancedEngine struct {
	authorizeDatabase func(string) bool
	s                 *Server
	r                 *http.Request
	workspaceID       string
	budget            formulaBudget
	queries           int
	databases         map[string]*advancedDatabase
	rows              map[string]map[string]any
	values            map[string]any
	failures          map[string]error
	visiting          map[string]bool
	formulas          map[string]*formulaNode
	userNames         map[string]string
	featureCheckedAt  time.Time
	formulaFeature    bool
	formulaDependent  map[string]bool
}

func (e *advancedEngine) formulaEnabled(wid string) bool {
	if e.featureCheckedAt.IsZero() || time.Since(e.featureCheckedAt) > 250*time.Millisecond {
		e.formulaFeature = e.s.canFeature(e.r.Context(), current(e.r), wid, "database-formula")
		e.featureCheckedAt = time.Now()
	}
	return e.formulaFeature
}

func newAdvancedEngine(s *Server, r *http.Request) *advancedEngine {
	return &advancedEngine{s: s, r: r, budget: formulaBudget{steps: 200000}, databases: map[string]*advancedDatabase{}, rows: map[string]map[string]any{}, values: map[string]any{}, failures: map[string]error{}, visiting: map[string]bool{}, formulas: map[string]*formulaNode{}, userNames: map[string]string{}, formulaDependent: map[string]bool{}}
}
func (e *advancedEngine) database(id string) (*advancedDatabase, error) {
	if e.authorizeDatabase != nil && !e.authorizeDatabase(id) {
		return nil, errors.New("허용된 데이터베이스 지식 범위를 벗어난 참조입니다")
	}
	if cached := e.databases[id]; cached != nil {
		return cached, nil
	}
	e.queries++
	if e.queries > 2000 {
		return nil, errors.New("관계 조회 한도를 초과했습니다")
	}
	if !e.s.canDatabase(e.r, id, false) {
		return nil, errors.New("관계 대상 접근 권한이 없습니다")
	}
	raw, err := e.s.one(e.r.Context(), "SELECT to_jsonb(d) FROM databases d WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	wid := str(raw, "workspace_id")
	if e.workspaceID == "" {
		e.workspaceID = wid
	} else if wid != e.workspaceID {
		return nil, errors.New("워크스페이스를 벗어난 관계입니다")
	}
	props, err := advancedProperties(raw["properties"])
	if err != nil {
		return nil, err
	}
	db := &advancedDatabase{id: id, workspaceID: wid, spaceID: str(raw, "space_id"), properties: props, lookup: map[string]map[string]any{}}
	for _, p := range props {
		db.lookup[str(p, "id")] = p
	}
	e.databases[id] = db
	return db, nil
}
func (e *advancedEngine) row(databaseID, rowID string) (map[string]any, error) {
	key := databaseID + ":" + rowID
	if row := e.rows[key]; row != nil {
		return row, nil
	}
	if _, err := e.database(databaseID); err != nil {
		return nil, err
	}
	e.queries++
	if e.queries > 2000 {
		return nil, errors.New("관계 조회 한도를 초과했습니다")
	}
	row, err := e.s.one(e.r.Context(), "SELECT to_jsonb(v) FROM database_rows v WHERE database_id=$1 AND id=$2", databaseID, rowID)
	if err != nil {
		return nil, errors.New("연결한 행을 찾을 수 없습니다")
	}
	e.rows[key] = row
	return row, nil
}
func (e *advancedEngine) value(db *advancedDatabase, row map[string]any, propertyID string, depth int) (any, error) {
	key := db.id + ":" + str(row, "id") + ":" + propertyID
	if p := db.lookup[propertyID]; (p != nil && str(p, "type") == "formula") || e.formulaDependent[key] {
		e.formulaDependent[key] = true
		for parent := range e.visiting {
			e.formulaDependent[parent] = true
		}
		if !e.formulaEnabled(db.workspaceID) {
			return nil, errors.New("현재 기능 공개 정책에서 수식 계산이 비활성화되었습니다")
		}
	}
	if v, ok := e.values[key]; ok {
		return v, nil
	}
	if err := e.failures[key]; err != nil {
		return nil, err
	}
	if e.visiting[key] || depth > formulaMaxDepth {
		return nil, errors.New("관계 또는 수식에 순환 참조가 있습니다")
	}
	e.budget.steps--
	if e.budget.steps < 0 {
		return nil, errors.New("계산 한도를 초과했습니다. 조회 행 수를 줄이세요")
	}
	if e.r.Context().Err() != nil {
		return nil, e.r.Context().Err()
	}
	p := db.lookup[propertyID]
	if p == nil {
		return nil, errors.New("참조한 속성이 삭제되었습니다")
	}
	e.visiting[key] = true
	defer delete(e.visiting, key)
	raw, _ := row["values"].(map[string]any)
	var result any
	var err error
	switch str(p, "type") {
	case "formula":
		ast := e.formulas[db.id+":"+propertyID]
		if ast == nil {
			ast, err = compileFormula(str(p, "expression"))
			if err != nil {
				break
			}
			e.formulas[db.id+":"+propertyID] = ast
		}
		cellBudget := formulaBudget{steps: 1000}
		result, err = evaluateFormula(ast, func(ref string) (any, error) {
			id, ex := advancedPropertyID(db.properties, ref)
			if ex != nil {
				return nil, ex
			}
			return e.value(db, row, id, depth+1)
		}, &cellBudget, 0)
		e.budget.steps -= 1000 - cellBudget.steps
		if e.budget.steps < 0 {
			result, err = nil, errors.New("계산 한도를 초과했습니다. 조회 행 수를 줄이세요")
		}
	case "rollup":
		relation := db.lookup[str(p, "relation_property_id")]
		if relation == nil || str(relation, "type") != "relation" {
			err = errors.New("롤업의 관계 속성이 없습니다")
			break
		}
		target, ex := e.database(str(relation, "target_database_id"))
		if ex != nil {
			err = ex
			break
		}
		refs := listStrings(raw[str(relation, "id")])
		if len(refs) > 100 {
			err = errors.New("관계 행은 100개까지 집계할 수 있습니다")
			break
		}
		values := []any{}
		for _, rowID := range refs {
			related, ex := e.row(target.id, rowID)
			if ex != nil {
				err = ex
				break
			}
			if str(p, "aggregation") == "count" {
				values = append(values, float64(1))
				continue
			}
			targetID, ex := advancedPropertyID(target.properties, str(p, "target_property_id"))
			if ex != nil {
				err = ex
				break
			}
			value, ex := e.value(target, related, targetID, depth+1)
			if ex != nil {
				err = ex
				break
			}
			values = append(values, value)
		}
		if err == nil {
			result, err = aggregateAdvancedValues(str(p, "aggregation"), values)
		}
	case "created_time":
		result = row["created_at"]
	case "updated_time":
		result = row["updated_at"]
	case "created_by", "updated_by":
		userID := str(row, str(p, "type"))
		if userID == "" {
			result = nil
			break
		}
		name, ok := e.userNames[userID]
		if !ok {
			e.queries++
			if e.queries > 2000 {
				err = errors.New("사용자 조회 한도를 초과했습니다")
				break
			}
			if ex := e.s.DB.QueryRow(e.r.Context(), "SELECT u.name FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE u.id=$1 AND m.workspace_id=$2", userID, db.workspaceID).Scan(&name); ex != nil {
				name = "탈퇴한 사용자"
			}
			e.userNames[userID] = name
		}
		result = name
	case "button":
		result = nil
	default:
		result = raw[propertyID]
	}
	if err != nil {
		e.failures[key] = err
		return nil, err
	}
	if len(jsonValue(result)) > formulaMaxOutput {
		err = errors.New("계산 결과가 64KB를 초과했습니다")
		e.failures[key] = err
		return nil, err
	}
	e.values[key] = result
	return result, nil
}
func aggregateAdvancedValues(operation string, values []any) (any, error) {
	if operation == "count" {
		return float64(len(values)), nil
	}
	if operation == "unique" {
		seen := map[string]bool{}
		out := []any{}
		for _, v := range values {
			key := string(jsonValue(v))
			if !seen[key] {
				out = append(out, v)
				seen[key] = true
			}
		}
		return out, nil
	}
	if len(values) == 0 {
		return nil, nil
	}
	sum, min, max := float64(0), math.Inf(1), math.Inf(-1)
	for _, v := range values {
		n, e := formulaNumber(v)
		if e != nil {
			return nil, errors.New("선택한 롤업 집계에는 숫자 속성이 필요합니다")
		}
		sum += n
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	if math.IsNaN(sum) || math.IsInf(sum, 0) {
		return nil, errors.New("롤업 숫자 계산 범위를 초과했습니다")
	}
	switch operation {
	case "sum":
		return sum, nil
	case "avg":
		return sum / float64(len(values)), nil
	case "min":
		return min, nil
	case "max":
		return max, nil
	}
	return nil, errors.New("지원하지 않는 집계 함수입니다")
}
func (s *Server) enrichDatabaseRows(r *http.Request, id string, rows []map[string]any) ([]map[string]any, error) {
	engine := newAdvancedEngine(s, r)
	db, err := engine.database(id)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		engine.rows[id+":"+str(row, "id")] = row
	}
	for _, row := range rows {
		computed, failures, relations := map[string]any{}, map[string]any{}, map[string]any{}
		for _, p := range db.properties {
			propertyID, kind := str(p, "id"), str(p, "type")
			if isAdvancedPropertyType(kind) {
				v, ex := engine.value(db, row, propertyID, 0)
				if ex != nil {
					failures[propertyID] = ex.Error()
				} else {
					computed[propertyID] = v
				}
			}
			if kind == "relation" {
				labels := map[string]any{}
				raw, _ := row["values"].(map[string]any)
				for _, ref := range listStrings(raw[propertyID]) {
					target, ex := engine.database(str(p, "target_database_id"))
					if ex != nil {
						failures[propertyID] = ex.Error()
						break
					}
					related, ex := engine.row(target.id, ref)
					if ex != nil {
						failures[propertyID] = ex.Error()
						continue
					}
					rv, _ := related["values"].(map[string]any)
					title := ref
					for _, tp := range target.properties {
						if str(tp, "type") == "text" && formulaString(rv[str(tp, "id")]) != "" {
							title = formulaString(rv[str(tp, "id")])
							break
						}
					}
					labels[ref] = title
				}
				relations[propertyID] = labels
			}
		}
		row["computed_values"], row["errors"], row["relation_labels"] = computed, failures, relations
	}
	// A long relation scan may outlive a rollout change. Never publish an earlier
	// formula result after its current feature gate has been revoked.
	if !s.canFeature(r.Context(), current(r), db.workspaceID, "database-formula") {
		for _, row := range rows {
			computed, _ := row["computed_values"].(map[string]any)
			failures, _ := row["errors"].(map[string]any)
			for _, p := range db.properties {
				if str(p, "type") == "formula" || engine.formulaDependent[db.id+":"+str(row, "id")+":"+str(p, "id")] {
					id := str(p, "id")
					delete(computed, id)
					failures[id] = "현재 기능 공개 정책에서 수식·집계 계산이 비활성화되었습니다"
				}
			}
		}
	}
	return rows, nil
}

type advancedFilter struct {
	PropertyID string `json:"property_id"`
	Operator   string `json:"operator"`
	Value      any    `json:"value"`
}
type advancedSort struct {
	PropertyID string `json:"property_id"`
	Direction  string `json:"direction"`
}

func advancedCell(row map[string]any, id string) any {
	if computed, ok := row["computed_values"].(map[string]any); ok {
		if v, exists := computed[id]; exists {
			return v
		}
	}
	values, _ := row["values"].(map[string]any)
	return values[id]
}
func advancedCompare(a, b any) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	an, ae := formulaNumber(a)
	bn, be := formulaNumber(b)
	if ae == nil && be == nil {
		if an < bn {
			return -1
		}
		if an > bn {
			return 1
		}
		return 0
	}
	return strings.Compare(strings.ToLower(formulaString(a)), strings.ToLower(formulaString(b)))
}
func advancedFilterMatches(value any, f advancedFilter) bool {
	empty := value == nil || value == ""
	if a, ok := value.([]any); ok {
		empty = len(a) == 0
	}
	switch f.Operator {
	case "is_empty":
		return empty
	case "not_empty":
		return !empty
	case "eq":
		return formulaEqual(value, f.Value)
	case "neq":
		return !formulaEqual(value, f.Value)
	case "contains":
		return strings.Contains(strings.ToLower(formulaString(value)), strings.ToLower(formulaString(f.Value)))
	case "not_contains":
		return !strings.Contains(strings.ToLower(formulaString(value)), strings.ToLower(formulaString(f.Value)))
	case "gt":
		return !empty && advancedCompare(value, f.Value) > 0
	case "gte":
		return !empty && advancedCompare(value, f.Value) >= 0
	case "lt":
		return !empty && advancedCompare(value, f.Value) < 0
	case "lte":
		return !empty && advancedCompare(value, f.Value) <= 0
	case "in":
		a, ok := f.Value.([]any)
		if !ok {
			return false
		}
		for _, v := range a {
			if formulaEqual(value, v) {
				return true
			}
		}
	}
	return false
}
func (s *Server) queryAdvancedDatabase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDatabase(r, id, false) || !hasIntegrationScope(current(r), "database:read") {
		apiError(w, 403, "데이터베이스 조회 권한이 없습니다")
		return
	}
	var in struct {
		Filters []advancedFilter `json:"filters"`
		Sorts   []advancedSort   `json:"sorts"`
		Limit   int              `json:"limit"`
		Offset  int              `json:"offset"`
	}
	if decode(r, &in) != nil || len(in.Filters) > 30 || len(in.Sorts) > 10 || in.Offset < 0 || in.Offset > 10000 {
		apiError(w, 400, "조회 조건을 확인하세요")
		return
	}
	if in.Limit == 0 {
		in.Limit = 1000
	}
	if in.Limit < 1 || in.Limit > 1000 {
		apiError(w, 400, "한 번에 1~1000개 행을 조회할 수 있습니다")
		return
	}
	db, err := newAdvancedEngine(s, r).database(id)
	if err != nil {
		respond(w, nil, err)
		return
	}
	for _, f := range in.Filters {
		if db.lookup[f.PropertyID] == nil || !oneOf(f.Operator, "eq", "neq", "contains", "not_contains", "is_empty", "not_empty", "gt", "gte", "lt", "lte", "in") {
			apiError(w, 400, "필터 속성 또는 연산자를 확인하세요")
			return
		}
	}
	for _, sort := range in.Sorts {
		if db.lookup[sort.PropertyID] == nil || !oneOf(sort.Direction, "asc", "desc") {
			apiError(w, 400, "정렬 속성 또는 방향을 확인하세요")
			return
		}
	}
	rows, err := s.rows(r.Context(), "SELECT to_jsonb(v) FROM database_rows v WHERE database_id=$1 ORDER BY created_at,id LIMIT 10001", id)
	if err != nil {
		respond(w, nil, err)
		return
	}
	truncated := len(rows) > 10000
	if truncated {
		rows = rows[:10000]
	}
	rows, err = s.enrichDatabaseRows(r, id, rows)
	if err != nil {
		respond(w, nil, err)
		return
	}
	filtered := []map[string]any{}
	for _, row := range rows {
		match := true
		for _, f := range in.Filters {
			if !advancedFilterMatches(advancedCell(row, f.PropertyID), f) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, row)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		for _, rule := range in.Sorts {
			comparison := advancedCompare(advancedCell(filtered[i], rule.PropertyID), advancedCell(filtered[j], rule.PropertyID))
			if comparison != 0 {
				if rule.Direction == "desc" {
					return comparison > 0
				}
				return comparison < 0
			}
		}
		return false
	})
	total := len(filtered)
	start := min(in.Offset, total)
	end := min(start+in.Limit, total)
	s.audit(r, "DATABASE_QUERY", id, map[string]any{"filters": len(in.Filters), "rows": end - start})
	jsonResponse(w, 200, map[string]any{"rows": filtered[start:end], "total": total, "truncated": truncated, "offset": start, "limit": in.Limit})
}
