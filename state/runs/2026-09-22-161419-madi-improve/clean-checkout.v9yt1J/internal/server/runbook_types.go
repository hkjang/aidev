package server

import (
	"context"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed runbook.sql
var runbookSchema string

func (s *Server) migrateRunbook(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, runbookSchema)
	return e
}

type runbookParameter struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Options  []string `json:"options"`
	Pattern  string   `json:"pattern"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
}
type runbookAction struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Roles      []string           `json:"roles"`
	Teams      []string           `json:"teams"`
	Parameters []runbookParameter `json:"parameters"`
	Timeout    int                `json:"timeout_seconds"`
	TemplateID int                `json:"template_id"`
	Image      string             `json:"image"`
	Argv       []string           `json:"argv"`
	CPUMilli   int                `json:"cpu_milli"`
	MemoryMi   int                `json:"memory_mi"`
}
type runbookRunner struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Enabled     bool            `json:"enabled"`
	Revision    int64           `json:"revision"`
	Config      map[string]any  `json:"config"`
	Actions     []runbookAction `json:"actions"`
}
type runbookStep struct {
	Name       string         `json:"name"`
	RunnerID   string         `json:"runner_id"`
	ActionID   string         `json:"action_id"`
	Parameters map[string]any `json:"parameters"`
}
type runbookDefinition struct {
	DocumentID      string        `json:"document_id"`
	Version         int64         `json:"version"`
	Purpose         string        `json:"purpose"`
	Prerequisites   string        `json:"prerequisites"`
	Validation      string        `json:"validation"`
	Rollback        string        `json:"rollback"`
	Steps           []runbookStep `json:"steps"`
	ValidationSteps []runbookStep `json:"validation_steps"`
	RollbackSteps   []runbookStep `json:"rollback_steps"`
}
type runbookPlanStep struct {
	Name           string         `json:"name"`
	RunnerID       string         `json:"runner_id"`
	RunnerRevision int64          `json:"runner_revision"`
	Kind           string         `json:"kind"`
	Action         runbookAction  `json:"action"`
	Parameters     map[string]any `json:"parameters"`
	Argv           []string       `json:"argv,omitempty"`
	Remote         map[string]any `json:"remote"`
}
type runbookPlan struct {
	DocumentID        string            `json:"document_id"`
	DocumentVersion   int64             `json:"document_version"`
	DocumentHash      string            `json:"document_hash"`
	DefinitionVersion int64             `json:"definition_version"`
	DefinitionHash    string            `json:"definition_hash"`
	SettingsRevision  int64             `json:"settings_revision"`
	Phase             string            `json:"phase"`
	Steps             []runbookPlanStep `json:"steps"`
	Title             string            `json:"title"`
}

var runbookName = regexp.MustCompile(`^[a-z][a-z0-9_\-]{0,63}$`)
var runbookImage = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9./:_-]*@sha256:[a-f0-9]{64}$`)
var runbookNamespace = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func validateRunbookRunner(r *runbookRunner) error {
	normalizeRunbookRunnerLists(r)
	if !validID(r.WorkspaceID) || strings.TrimSpace(r.Name) == "" || len(r.Name) > 200 || !oneOf(r.Kind, "awx", "kubernetes") {
		return approvalProblem(400, "실행기 이름·워크스페이스·유형을 확인하세요")
	}
	endpoint, e := url.Parse(str(r.Config, "base_url"))
	if e != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return approvalProblem(400, "실행기는 인증서 검증이 가능한 HTTPS 주소만 허용합니다")
	}
	for key, value := range r.Config {
		if !oneOf(key, "base_url", "token", "token_configured", "ca_pem", "namespace", "runtime_class") {
			return approvalProblem(400, "지원하지 않는 실행기 설정: "+key)
		}
		if key != "token_configured" {
			if v, ok := value.(string); !ok || len(v) > 256<<10 {
				return approvalProblem(400, "실행기 설정은 제한된 문자열이어야 합니다")
			}
		}
	}
	if ca := str(r.Config, "ca_pem"); ca != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(ca)) {
			return approvalProblem(400, "실행기 CA 인증서를 확인하세요")
		}
	}
	if r.Enabled && str(r.Config, "token") == "" {
		return approvalProblem(400, "사용할 실행기의 Bearer 자격 증명을 입력하세요")
	}
	if strings.ContainsAny(str(r.Config, "token"), "\x00\r\n") {
		return approvalProblem(400, "실행기 Bearer 자격 증명에는 줄바꿈을 사용할 수 없습니다")
	}
	if r.Kind == "kubernetes" && (!runbookNamespace.MatchString(str(r.Config, "namespace")) || oneOf(str(r.Config, "namespace"), "default", "kube-system", "kube-public", "kube-node-lease")) {
		return approvalProblem(400, "시스템·기본 namespace가 아닌 전용 실행 namespace를 지정하세요")
	}
	if v := str(r.Config, "runtime_class"); v != "" && !runbookNamespace.MatchString(v) {
		return approvalProblem(400, "RuntimeClass 이름을 확인하세요")
	}
	if len(r.Actions) < 1 || len(r.Actions) > 50 {
		return approvalProblem(400, "실행기에는 1~50개 허용 작업을 등록하세요")
	}
	seen := map[string]bool{}
	for i := range r.Actions {
		a := &r.Actions[i]
		if !runbookName.MatchString(a.ID) || seen[a.ID] || strings.TrimSpace(a.Name) == "" || len(a.Name) > 200 {
			return approvalProblem(400, "중복되지 않는 작업 ID와 이름을 입력하세요")
		}
		seen[a.ID] = true
		if a.Timeout == 0 {
			a.Timeout = 300
		}
		if a.Timeout < 5 || a.Timeout > 1800 {
			return approvalProblem(400, "단계 제한 시간은 5~1,800초입니다")
		}
		if len(a.Roles)+len(a.Teams) == 0 {
			return approvalProblem(400, "작업을 실행할 역할 또는 팀을 명시적으로 허용하세요")
		}
		for _, role := range a.Roles {
			if !oneOf(role, "admin", "editor") {
				return approvalProblem(400, "실행 역할은 관리자 또는 편집자만 허용합니다")
			}
		}
		for _, team := range a.Teams {
			if !validID(team) {
				return approvalProblem(400, "실행 팀 ID를 확인하세요")
			}
		}
		if len(a.Parameters) > 30 {
			return approvalProblem(400, "작업 매개변수는 30개 이하입니다")
		}
		params := map[string]bool{}
		for _, p := range a.Parameters {
			if !runbookName.MatchString(p.Name) || params[p.Name] || strings.HasPrefix(p.Name, "madi_") || strings.HasPrefix(p.Name, "ansible_") || strings.HasPrefix(p.Name, "awx_") || strings.HasPrefix(p.Name, "tower_") || !oneOf(p.Type, "string", "integer", "boolean", "enum") || len(p.Label) > 200 {
				return approvalProblem(400, "매개변수 이름·형식 또는 예약 변수를 확인하세요")
			}
			params[p.Name] = true
			if p.Type == "string" {
				if p.Pattern == "" || len(p.Pattern) > 256 {
					return approvalProblem(400, "문자열 매개변수에는 허용 패턴을 명시하세요")
				}
				if _, e := regexp.Compile("^(?:" + p.Pattern + ")$"); e != nil {
					return approvalProblem(400, "문자열 매개변수 패턴을 확인하세요")
				}
			}
			if p.Type == "enum" && (len(p.Options) == 0 || len(p.Options) > 100) {
				return approvalProblem(400, "선택 매개변수는 1~100개 옵션을 등록하세요")
			}
			options := map[string]bool{}
			for _, value := range p.Options {
				if value == "" || options[value] || len(value) > 1000 || strings.ContainsAny(value, "\x00\r\n") {
					return approvalProblem(400, "선택 옵션 값을 확인하세요")
				}
				options[value] = true
			}
			if p.Min != nil && (math.IsNaN(*p.Min) || math.IsInf(*p.Min, 0)) || p.Max != nil && (math.IsNaN(*p.Max) || math.IsInf(*p.Max, 0)) || p.Min != nil && p.Max != nil && *p.Min > *p.Max {
				return approvalProblem(400, "숫자 매개변수 범위를 확인하세요")
			}
		}
		if r.Kind == "awx" {
			if a.TemplateID < 1 {
				return approvalProblem(400, "AWX의 고정 Job Template ID를 입력하세요")
			}
			continue
		}
		if !runbookImage.MatchString(a.Image) || len(a.Argv) < 1 || len(a.Argv) > 50 || !strings.HasPrefix(a.Argv[0], "/") || strings.Contains(a.Argv[0], "${") {
			return approvalProblem(400, "Kubernetes 작업은 정확한 이미지 digest와 고정 절대경로 실행 파일이 필요합니다")
		}
		if a.CPUMilli < 10 || a.CPUMilli > 8000 || a.MemoryMi < 16 || a.MemoryMi > 16384 {
			return approvalProblem(400, "CPU는 10~8,000m, 메모리는 16~16,384Mi 범위로 제한하세요")
		}
		for _, arg := range a.Argv {
			if len(arg) > 2000 || strings.ContainsAny(arg, "\x00\r\n") {
				return approvalProblem(400, "고정 argv를 확인하세요")
			}
			if strings.Contains(arg, "${") {
				name := strings.TrimSuffix(strings.TrimPrefix(arg, "${"), "}")
				if arg != "${"+name+"}" || !params[name] {
					return approvalProblem(400, "매개변수는 argv의 독립 항목 ${name}으로만 전달하세요")
				}
			}
		}
		if oneOf(path.Base(a.Argv[0]), "sh", "bash", "dash", "zsh", "ksh", "python", "python3", "node", "perl", "ruby") && (len(a.Argv) < 2 || !strings.HasPrefix(a.Argv[1], "/") || strings.Contains(a.Argv[1], "${")) {
			return approvalProblem(400, "인터프리터는 이미지에 내장된 고정 절대경로 스크립트만 실행할 수 있습니다")
		}
	}
	return nil
}

func runbookParameters(a runbookAction, input map[string]any) (map[string]any, []string, error) {
	out := map[string]any{}
	known := map[string]bool{}
	for _, p := range a.Parameters {
		known[p.Name] = true
		value, exists := input[p.Name]
		if !exists {
			if p.Required {
				return nil, nil, approvalProblem(400, p.Name+" 값은 필수입니다")
			}
			continue
		}
		valid := false
		switch p.Type {
		case "string", "enum":
			v, ok := value.(string)
			valid = ok && len(v) <= 1000 && !strings.ContainsAny(v, "\x00\r\n") && !strings.Contains(v, "{{") && !strings.Contains(v, "{%")
			if valid {
				if p.Type == "enum" {
					valid = slices.Contains(p.Options, v)
				} else {
					pattern, e := regexp.Compile("^(?:" + p.Pattern + ")$")
					valid = e == nil && pattern.MatchString(v)
				}
			}
		case "boolean":
			_, valid = value.(bool)
		case "integer":
			n, ok := value.(float64)
			if !ok {
				if i, yes := value.(int); yes {
					n = float64(i)
					ok = true
					value = n
				}
			}
			valid = ok && math.Trunc(n) == n && math.Abs(n) <= 9007199254740991 && (p.Min == nil || n >= *p.Min) && (p.Max == nil || n <= *p.Max)
		}
		if !valid {
			return nil, nil, approvalProblem(400, p.Name+" 값이 관리자의 허용 형식과 일치하지 않습니다")
		}
		out[p.Name] = value
	}
	for key := range input {
		if !known[key] {
			return nil, nil, approvalProblem(400, "허용되지 않은 실행 매개변수: "+key)
		}
	}
	argv := append([]string{}, a.Argv...)
	for i, arg := range argv {
		if strings.HasPrefix(arg, "${") && strings.HasSuffix(arg, "}") {
			name := arg[2 : len(arg)-1]
			value, ok := out[name]
			if !ok {
				return nil, nil, approvalProblem(400, "argv에 사용되는 매개변수 "+name+" 값은 필수입니다")
			}
			if n, ok := value.(float64); ok {
				argv[i] = strconv.FormatFloat(n, 'f', 0, 64)
			} else {
				argv[i] = fmt.Sprint(value)
			}
		}
	}
	return out, argv, nil
}

func runbookRunnerTx(ctx context.Context, tx pgx.Tx, id string, revision int64) (runbookRunner, error) {
	var r runbookRunner
	var config, actions []byte
	if !validID(id) {
		return r, pgx.ErrNoRows
	}
	e := tx.QueryRow(ctx, `SELECT r.id::text,r.workspace_id::text,r.name,r.kind,r.enabled,r.revision,v.config,v.actions FROM runbook_runners r JOIN runbook_runner_versions v ON v.runner_id=r.id AND v.revision=CASE WHEN $2::bigint=0 THEN r.revision ELSE $2 END WHERE r.id=$1 FOR SHARE OF r,v`, id, revision).Scan(&r.ID, &r.WorkspaceID, &r.Name, &r.Kind, &r.Enabled, &r.Revision, &config, &actions)
	if e != nil {
		return r, e
	}
	if e = json.Unmarshal(config, &r.Config); e != nil {
		return r, e
	}
	e = json.Unmarshal(actions, &r.Actions)
	normalizeRunbookRunnerLists(&r)
	return r, e
}
func normalizeRunbookRunnerLists(r *runbookRunner) {
	if r.Actions == nil {
		r.Actions = []runbookAction{}
	}
	for i := range r.Actions {
		a := &r.Actions[i]
		if a.Roles == nil {
			a.Roles = []string{}
		}
		if a.Teams == nil {
			a.Teams = []string{}
		}
		if a.Argv == nil {
			a.Argv = []string{}
		}
		if a.Parameters == nil {
			a.Parameters = []runbookParameter{}
		}
		for j := range a.Parameters {
			if a.Parameters[j].Options == nil {
				a.Parameters[j].Options = []string{}
			}
		}
	}
}
func runbookActionAllowedTx(ctx context.Context, tx pgx.Tx, uid, wid string, action runbookAction) (bool, error) {
	var role, kind string
	var disabled bool
	e := tx.QueryRow(ctx, `SELECT m.role,u.kind,u.disabled FROM users u JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=$2 WHERE u.id=$1`, uid, wid).Scan(&role, &kind, &disabled)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if disabled || kind != "user" {
		return false, nil
	}
	if (oneOf(role, "owner", "admin") && slices.Contains(action.Roles, "admin")) || (oneOf(role, "owner", "admin", "editor") && slices.Contains(action.Roles, "editor")) {
		return true, nil
	}
	if len(action.Teams) == 0 {
		return false, nil
	}
	var allowed bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM team_members tm JOIN teams t ON t.id=tm.team_id WHERE tm.user_id=$1 AND t.workspace_id=$2 AND t.id::text=ANY($3))`, uid, wid, action.Teams).Scan(&allowed)
	return allowed, e
}
