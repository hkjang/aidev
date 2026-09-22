package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type runbookRemoteError struct{ Status int }

func (e *runbookRemoteError) Error() string {
	return fmt.Sprintf("격리 실행기 응답 오류 (HTTP %d)", e.Status)
}

type runbookRemote struct {
	runner runbookRunner
	client *http.Client
	token  string
}
type runbookRemoteState struct {
	Status    string
	Output    string
	Truncated bool
}

type runbookSafetyError struct{ message string }

func (e *runbookSafetyError) Error() string { return e.message }

func (s *Server) newRunbookRemote(r runbookRunner) (*runbookRemote, error) {
	token, e := s.decrypt(str(r.Config, "token"))
	if e != nil || token == "" {
		return nil, approvalProblem(400, "실행기 자격 증명을 확인하세요")
	}
	pool, e := x509.SystemCertPool()
	if e != nil {
		pool = x509.NewCertPool()
	}
	if ca := str(r.Config, "ca_pem"); ca != "" && !pool.AppendCertsFromPEM([]byte(ca)) {
		return nil, approvalProblem(400, "실행기 CA 인증서를 확인하세요")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}, MaxIdleConns: 8, MaxIdleConnsPerHost: 4, IdleConnTimeout: 30 * time.Second, ResponseHeaderTimeout: 20 * time.Second}
	return &runbookRemote{runner: r, token: token, client: &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (r *runbookRemote) close() { r.client.CloseIdleConnections() }
func (r *runbookRemote) request(ctx context.Context, method, path string, input any) ([]byte, error) {
	var body io.Reader
	if input != nil {
		body = bytes.NewReader(jsonValue(input))
	}
	req, e := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(str(r.runner.Config, "base_url"), "/")+path, body)
	if e != nil {
		return nil, fmt.Errorf("격리 실행기 요청 주소를 확인하세요")
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, e := r.client.Do(req)
	if e != nil {
		return nil, fmt.Errorf("격리 실행기 연결에 실패했습니다. 인증서·주소·제한 시간을 확인하세요")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &runbookRemoteError{response.StatusCode}
	}
	raw, e := io.ReadAll(io.LimitReader(response.Body, (2<<20)+1))
	if e != nil {
		return nil, fmt.Errorf("격리 실행기 응답을 읽지 못했습니다")
	}
	if len(raw) > 2<<20 {
		if strings.Contains(path, "/stdout/") || strings.Contains(path, "/log?") {
			return raw[:2<<20], nil
		}
		return nil, fmt.Errorf("격리 실행기 응답이 2MB 제한을 초과했습니다")
	}
	return raw, nil
}
func (r *runbookRemote) json(ctx context.Context, method, path string, input any) (map[string]any, error) {
	raw, e := r.request(ctx, method, path, input)
	if e != nil {
		return nil, e
	}
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	e = json.Unmarshal(raw, &out)
	if e != nil {
		return nil, fmt.Errorf("격리 실행기의 JSON 응답이 잘못되었습니다")
	}
	return out, nil
}
func runbookObject(m map[string]any, key string) map[string]any {
	value, _ := m[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}
func runbookArray(m map[string]any, key string) []any { value, _ := m[key].([]any); return value }

func (r *runbookRemote) preflight(ctx context.Context, action runbookAction) (map[string]any, error) {
	if r.runner.Kind == "awx" {
		return r.awxPreflight(ctx, action)
	}
	return r.kubernetesPreflight(ctx, action)
}
func (r *runbookRemote) awxPreflight(ctx context.Context, action runbookAction) (map[string]any, error) {
	template, e := r.json(ctx, "GET", fmt.Sprintf("/api/v2/job_templates/%d/", action.TemplateID), nil)
	if e != nil {
		return nil, e
	}
	if number(template, "id", 0) != action.TemplateID || number(template, "timeout", 0) <= 0 || number(template, "timeout", 0) > action.Timeout {
		return nil, approvalProblem(400, "AWX 템플릿에 madi 단계 제한 시간 이하의 양수 timeout을 설정하세요")
	}
	if boolean(template, "ask_inventory_on_launch") || boolean(template, "ask_credential_on_launch") || boolean(template, "ask_scm_branch_on_launch") {
		return nil, approvalProblem(400, "AWX 템플릿의 실행 시 인벤토리·자격증명·브랜치 변경을 비활성화하세요")
	}
	if !boolean(template, "ask_variables_on_launch") {
		return nil, approvalProblem(400, "선언된 매개변수와 실행 추적 ID를 반영하도록 AWX 템플릿의 extra_vars 실행 시 입력을 허용하세요")
	}
	projectID := number(template, "project", 0)
	if projectID < 1 {
		return nil, approvalProblem(400, "AWX 템플릿의 고정 프로젝트를 확인하세요")
	}
	project, e := r.json(ctx, "GET", fmt.Sprintf("/api/v2/projects/%d/", projectID), nil)
	if e != nil {
		return nil, e
	}
	if boolean(project, "scm_update_on_launch") || str(project, "scm_revision") == "" {
		return nil, approvalProblem(400, "AWX 프로젝트를 고정 SCM revision으로 동기화하고 실행 시 업데이트를 끄세요")
	}
	eeID := number(template, "execution_environment", 0)
	if eeID < 1 {
		return nil, approvalProblem(400, "AWX 템플릿에 digest가 고정된 Execution Environment를 명시하세요")
	}
	ee, e := r.json(ctx, "GET", fmt.Sprintf("/api/v2/execution_environments/%d/", eeID), nil)
	if e != nil {
		return nil, e
	}
	if !runbookImage.MatchString(str(ee, "image")) {
		return nil, approvalProblem(400, "AWX Execution Environment 이미지도 @sha256 digest로 고정하세요")
	}
	launch, e := r.json(ctx, "GET", fmt.Sprintf("/api/v2/job_templates/%d/launch/", action.TemplateID), nil)
	if e != nil {
		return nil, e
	}
	if len(runbookArray(launch, "passwords_needed_to_start")) > 0 {
		return nil, approvalProblem(400, "AWX 템플릿에서 대화형 비밀번호 입력을 제거하세요")
	}
	fields := map[string]any{}
	for _, key := range []string{"id", "modified", "name", "job_type", "inventory", "project", "playbook", "timeout", "execution_environment", "survey_enabled", "ask_variables_on_launch", "limit", "verbosity", "job_tags", "skip_tags", "become_enabled", "forks", "job_slice_count"} {
		fields[key] = template[key]
	}
	fields["extra_vars_hash"] = digest(str(template, "extra_vars"))
	fields["project_revision"] = project["scm_revision"]
	return map[string]any{"template": fields, "project_id": projectID, "execution_environment": map[string]any{"id": eeID, "image": ee["image"], "modified": ee["modified"]}, "isolation": "AWX 실행 노드·인벤토리 정책"}, nil
}
func (r *runbookRemote) launch(ctx context.Context, executionID string, index int, step runbookPlanStep) (string, error) {
	if r.runner.Kind == "kubernetes" {
		return r.kubernetesLaunch(ctx, executionID, index, step)
	}
	vars := map[string]any{"madi_execution_id": executionID, "madi_execution_step": index}
	for key, value := range step.Parameters {
		vars[key] = value
	}
	body := map[string]any{"extra_vars": vars}
	result, e := r.json(ctx, "POST", fmt.Sprintf("/api/v2/job_templates/%d/launch/", step.Action.TemplateID), body)
	if e != nil {
		return "", e
	}
	id := number(result, "id", number(result, "job", 0))
	if id < 1 {
		return "", fmt.Errorf("AWX 실행 결과에 Job ID가 없어 실행 여부를 확정할 수 없습니다")
	}
	if len(runbookObject(result, "ignored_fields")) > 0 {
		return strconv.Itoa(id), fmt.Errorf("AWX가 실행 매개변수를 무시했습니다. 작업을 중지하고 템플릿을 확인하세요")
	}
	return strconv.Itoa(id), nil
}
func (r *runbookRemote) poll(ctx context.Context, externalID string, step runbookPlanStep, output bool) (runbookRemoteState, error) {
	if r.runner.Kind == "kubernetes" {
		return r.kubernetesPoll(ctx, externalID, step, output)
	}
	id, e := strconv.Atoi(externalID)
	if e != nil || id < 1 {
		return runbookRemoteState{}, fmt.Errorf("AWX Job ID를 확인하세요")
	}
	result, e := r.json(ctx, "GET", fmt.Sprintf("/api/v2/jobs/%d/", id), nil)
	if e != nil {
		return runbookRemoteState{}, e
	}
	if number(result, "job_template", 0) != step.Action.TemplateID {
		return runbookRemoteState{}, &runbookSafetyError{"AWX 작업의 템플릿이 승인 대상과 다릅니다"}
	}
	if revision := str(result, "scm_revision"); output && revision != "" && revision != str(runbookObject(step.Remote, "template"), "project_revision") {
		return runbookRemoteState{}, &runbookSafetyError{"AWX 작업의 SCM revision이 승인된 원본과 다릅니다"}
	}
	state := runbookRemoteState{Status: "running"}
	switch str(result, "status") {
	case "successful":
		state.Status = "succeeded"
	case "failed", "error":
		state.Status = "failed"
	case "canceled":
		state.Status = "cancelled"
	case "new", "pending", "waiting", "running":
	default:
		return state, fmt.Errorf("알 수 없는 AWX 작업 상태입니다")
	}
	if !output {
		return state, nil
	}
	raw, e := r.request(ctx, "GET", fmt.Sprintf("/api/v2/jobs/%d/stdout/?format=txt", id), nil)
	if e != nil {
		return state, e
	}
	state.Output = strings.ToValidUTF8(string(raw), "�")
	state.Truncated = len(raw) >= 2<<20
	return state, nil
}
func (r *runbookRemote) cancel(ctx context.Context, externalID string) error {
	if r.runner.Kind == "kubernetes" {
		return r.kubernetesCancel(ctx, externalID)
	}
	id, e := strconv.Atoi(externalID)
	if e != nil || id < 1 {
		return fmt.Errorf("AWX Job ID를 확인하세요")
	}
	_, e = r.json(ctx, "POST", fmt.Sprintf("/api/v2/jobs/%d/cancel/", id), map[string]any{})
	return e
}
func runbookJobName(executionID string, index int) string {
	return "madi-" + strings.ReplaceAll(executionID, "-", "") + "-" + strconv.Itoa(index)
}
func (r *runbookRemote) namespacePath() string {
	return "/api/v1/namespaces/" + url.PathEscape(str(r.runner.Config, "namespace"))
}
