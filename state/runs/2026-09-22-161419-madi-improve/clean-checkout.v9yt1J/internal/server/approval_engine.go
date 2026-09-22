package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed approval.sql
var approvalSchema string

func (s *Server) migrateApproval(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, approvalSchema)
	return e
}

type approvalGate struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
	Role string `json:"role,omitempty"`
}
type approvalStage struct {
	Name  string         `json:"name"`
	Mode  string         `json:"mode"`
	Gates []approvalGate `json:"gates"`
}
type approvalPolicy struct {
	ID           string          `json:"id"`
	WorkspaceID  string          `json:"workspace_id"`
	SpaceID      string          `json:"space_id"`
	ResourceKind string          `json:"resource_kind"`
	Name         string          `json:"name"`
	Enabled      bool            `json:"enabled"`
	Version      int64           `json:"version"`
	Stages       []approvalStage `json:"stages"`
}
type approvalRequest struct {
	ID              string         `json:"id"`
	WorkspaceID     string         `json:"workspace_id"`
	DocumentID      string         `json:"document_id"`
	ResourceKind    string         `json:"resource_kind"`
	ResourceID      string         `json:"resource_id"`
	ResourceVersion int64          `json:"resource_version"`
	ResourceHash    string         `json:"resource_hash"`
	Snapshot        map[string]any `json:"snapshot"`
	PolicyID        string         `json:"policy_id"`
	PolicyVersion   int64          `json:"policy_version"`
	PolicyRevision  int64          `json:"policy_revision"`
	Policy          approvalPolicy `json:"policy"`
	RequesterID     string         `json:"requester_id"`
	OwnerID         string         `json:"owner_id"`
	Status          string         `json:"status"`
	CurrentStage    int            `json:"current_stage"`
	Version         int64          `json:"version"`
	Comment         string         `json:"comment"`
}
type approvalFailure struct {
	Status  int
	Message string
}

func (e *approvalFailure) Error() string               { return e.Message }
func approvalProblem(status int, message string) error { return &approvalFailure{status, message} }

// Caller must acquire the document lock before calling this function. Settings
// changes update the clock while holding the same settings row write lock.
func approvalConfigurationTx(ctx context.Context, tx pgx.Tx) (map[string]any, int64, error) {
	var raw []byte
	var revision int64
	if e := tx.QueryRow(ctx, `SELECT data FROM settings WHERE id=1 FOR SHARE`).Scan(&raw); e != nil {
		return nil, 0, e
	}
	cfg := defaultSettings()
	if e := json.Unmarshal(raw, &cfg); e != nil {
		return nil, 0, e
	}
	if e := tx.QueryRow(ctx, `SELECT revision FROM approval_policy_clock WHERE id=1`).Scan(&revision); e != nil {
		return nil, 0, e
	}
	if !boolean(cfg, "approval_enabled") {
		return nil, 0, approvalProblem(404, "검토·승인 기능이 활성화되어 있지 않습니다")
	}
	return cfg, revision, nil
}

func approvalPolicyTx(ctx context.Context, tx pgx.Tx, cfg map[string]any, wid, space, kind string) (approvalPolicy, error) {
	var out approvalPolicy
	var stages []byte
	e := tx.QueryRow(ctx, `WITH RECURSIVE ancestors AS (
 SELECT id,parent_id,0 depth,ARRAY[id] path FROM spaces WHERE id=NULLIF($2,'')::uuid AND workspace_id=$1
 UNION ALL SELECT s.id,s.parent_id,a.depth+1,a.path||s.id FROM spaces s JOIN ancestors a ON s.id=a.parent_id WHERE s.workspace_id=$1 AND a.depth<100 AND NOT s.id=ANY(a.path)
 ) SELECT p.id::text,p.workspace_id::text,COALESCE(p.space_id::text,''),p.resource_kind,p.name,p.enabled,p.version,p.stages FROM approval_policies p
 WHERE p.workspace_id=$1 AND p.resource_kind=$3 AND p.enabled AND (p.space_id IS NULL OR p.space_id IN(SELECT id FROM ancestors))
 ORDER BY COALESCE((SELECT depth FROM ancestors WHERE id=p.space_id),101),p.id LIMIT 1 FOR SHARE OF p`, wid, space, kind).Scan(&out.ID, &out.WorkspaceID, &out.SpaceID, &out.ResourceKind, &out.Name, &out.Enabled, &out.Version, &stages)
	if errors.Is(e, pgx.ErrNoRows) {
		if oneOf(kind, "runbook", "impact_exception", "knowledge_distribution", "learning_step") {
			return out, approvalProblem(409, "이 리소스에는 관리자가 명시적인 승인 정책을 설정해야 합니다")
		}
		return approvalPolicy{WorkspaceID: wid, ResourceKind: kind, Name: "기본 문서 검토", Enabled: true, Stages: []approvalStage{{Name: "검토 및 승인", Mode: "all", Gates: []approvalGate{{Name: "다른 검토 담당자", Kind: "role", Role: str(cfg, "reviewer_role")}}}}}, nil
	}
	if e != nil {
		return out, e
	}
	e = json.Unmarshal(stages, &out.Stages)
	if e == nil {
		e = approvalValidatePolicy(&out)
	}
	return out, e
}

func approvalValidatePolicy(policy *approvalPolicy) error {
	policy.Name = strings.TrimSpace(policy.Name)
	if !validID(policy.WorkspaceID) || (policy.SpaceID != "" && !validID(policy.SpaceID)) || !oneOf(policy.ResourceKind, "document", "runbook", "sql_query_plan", "impact_exception", "knowledge_distribution", "learning_step") || policy.Name == "" || len(policy.Name) > 250 {
		return approvalProblem(400, "정책 이름·워크스페이스·공간·대상 유형을 확인하세요")
	}
	if len(policy.Stages) == 0 || len(policy.Stages) > 20 {
		return approvalProblem(400, "승인 정책은 1~20단계로 구성하세요")
	}
	for _, stage := range policy.Stages {
		if strings.TrimSpace(stage.Name) == "" || len(stage.Name) > 250 || !oneOf(stage.Mode, "any", "all") || len(stage.Gates) == 0 || len(stage.Gates) > 20 {
			return approvalProblem(400, "각 단계 이름·병렬 승인 방식·1~20개 검토 대상을 확인하세요")
		}
		for _, gate := range stage.Gates {
			if strings.TrimSpace(gate.Name) == "" || len(gate.Name) > 250 || !oneOf(gate.Kind, "user", "team", "role", "document_reviewer") {
				return approvalProblem(400, "검토 대상 이름과 유형을 확인하세요")
			}
			if oneOf(gate.Kind, "user", "team") && !validID(gate.ID) {
				return approvalProblem(400, "검토 사용자 또는 팀을 선택하세요")
			}
			if gate.Kind == "role" && !oneOf(gate.Role, "admin", "editor", "commenter", "viewer") {
				return approvalProblem(400, "검토 역할을 확인하세요")
			}
		}
	}
	return nil
}

func approvalValidatePolicyTargetsTx(ctx context.Context, tx pgx.Tx, policy approvalPolicy) error {
	var valid bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE id=$1) AND ($2='' OR EXISTS(SELECT 1 FROM spaces WHERE id=NULLIF($2,'')::uuid AND workspace_id=$1))`, policy.WorkspaceID, policy.SpaceID).Scan(&valid); e != nil {
		return e
	}
	if !valid {
		return approvalProblem(400, "정책의 워크스페이스와 공간을 확인하세요")
	}
	for _, stage := range policy.Stages {
		for _, gate := range stage.Gates {
			if gate.Kind == "user" {
				if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members m JOIN users u ON u.id=m.user_id WHERE m.workspace_id=$1 AND u.id=$2 AND u.kind='user' AND NOT u.disabled)`, policy.WorkspaceID, gate.ID).Scan(&valid); e != nil {
					return e
				}
				if !valid {
					return approvalProblem(400, "검토 사용자는 워크스페이스의 활성 일반 사용자여야 합니다")
				}
			}
			if gate.Kind == "team" {
				if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM teams WHERE workspace_id=$1 AND id=$2)`, policy.WorkspaceID, gate.ID).Scan(&valid); e != nil {
					return e
				}
				if !valid {
					return approvalProblem(400, "팀·부서는 같은 워크스페이스에 있어야 합니다")
				}
			}
		}
	}
	return nil
}

func (s *Server) approvalCandidatesTx(ctx context.Context, tx pgx.Tx, gate approvalGate, resource ApprovalResource, requesterID string) ([]string, error) {
	rows, e := tx.Query(ctx, `SELECT u.id::text FROM users u JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=$1
 WHERE u.kind='user' AND NOT u.disabled AND u.id<>$3 AND u.id<>$4 AND (
 ($5='user' AND u.id=NULLIF($6,'')::uuid) OR
 ($5='team' AND EXISTS(SELECT 1 FROM teams t JOIN team_members tm ON tm.team_id=t.id WHERE t.workspace_id=$1 AND t.id=NULLIF($6,'')::uuid AND tm.user_id=u.id)) OR
 ($5='document_reviewer' AND EXISTS(SELECT 1 FROM knowledge_document_meta k WHERE k.document_id=NULLIF($2,'')::uuid AND k.reviewer_id=u.id)) OR
 ($5='role' AND (m.role IN ('owner','admin') OR ($7 IN ('editor','commenter','viewer') AND m.role='editor') OR ($7 IN ('commenter','viewer') AND m.role='commenter') OR ($7='viewer' AND m.role='viewer')))
 ) ORDER BY u.id LIMIT 201`, resource.WorkspaceID, resource.DocumentID, resource.OwnerID, requesterID, gate.Kind, gate.ID, gate.Role)
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, id)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return nil, e
	}
	rows.Close()
	if len(ids) > 200 {
		return nil, approvalProblem(400, "검토 대상은 gate당 최대 200명입니다. 팀 또는 지정 사용자로 범위를 좁히세요")
	}
	adapter, ok := s.approvalAdapters[resource.Kind]
	if !ok {
		return nil, approvalProblem(409, "승인 대상 모듈이 등록되어 있지 않습니다")
	}
	allowed := []string{}
	for _, uid := range ids {
		valid, e := adapter.CanReview(ctx, tx, uid, resource)
		if e != nil {
			return nil, e
		}
		if valid {
			allowed = append(allowed, uid)
		}
	}
	return allowed, nil
}

func approvalDocumentSnapshot(doc map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"id", "workspace_id", "title", "markdown", "tags", "aliases", "visibility", "owner_id", "parent_id", "space_id", "block_metadata"} {
		out[key] = doc[key]
	}
	return out
}
func approvalSnapshotHash(snapshot map[string]any) string { return digest(string(jsonValue(snapshot))) }

func approvalReadRequestTx(ctx context.Context, tx pgx.Tx, id string, lock bool) (approvalRequest, error) {
	var out approvalRequest
	var snapshot, policy []byte
	query := `SELECT id::text,workspace_id::text,COALESCE(document_id::text,''),resource_kind,resource_id::text,resource_version,resource_hash,snapshot,COALESCE(policy_id::text,''),policy_version,policy_revision,policy_snapshot,requester_id::text,owner_id::text,status,current_stage,version,comment FROM approval_requests WHERE id=$1`
	if lock {
		query += " FOR UPDATE"
	}
	e := tx.QueryRow(ctx, query, id).Scan(&out.ID, &out.WorkspaceID, &out.DocumentID, &out.ResourceKind, &out.ResourceID, &out.ResourceVersion, &out.ResourceHash, &snapshot, &out.PolicyID, &out.PolicyVersion, &out.PolicyRevision, &policy, &out.RequesterID, &out.OwnerID, &out.Status, &out.CurrentStage, &out.Version, &out.Comment)
	if e != nil {
		return out, e
	}
	if e = json.Unmarshal(snapshot, &out.Snapshot); e != nil {
		return out, e
	}
	e = json.Unmarshal(policy, &out.Policy)
	if e == nil {
		e = approvalValidatePolicy(&out.Policy)
	}
	if e == nil && (out.Policy.WorkspaceID != out.WorkspaceID || out.Policy.ResourceKind != out.ResourceKind) {
		e = approvalProblem(409, "검토 정책 snapshot 범위가 일치하지 않습니다")
	}
	return out, e
}

// Generic immutable resource submission. The caller holds the source document
// lock and applies any resource-specific status mutation in this transaction.
func (s *Server) StartApprovalTx(ctx context.Context, tx pgx.Tx, p *Principal, kind, resourceID, comment string) (ApprovalRequest, error) {
	var out approvalRequest
	adapter, ok := s.approvalAdapters[kind]
	if !ok {
		return out, approvalProblem(409, "승인 대상 모듈이 등록되어 있지 않습니다")
	}
	resource, e := adapter.Lock(ctx, tx, p, resourceID, true)
	if e != nil {
		return out, e
	}
	if resource.Kind != kind || resource.ID != resourceID || !validID(resource.WorkspaceID) || !validID(resource.OwnerID) || (p.WorkspaceID != "" && p.WorkspaceID != resource.WorkspaceID) {
		return out, approvalProblem(403, "승인 대상의 워크스페이스 범위가 일치하지 않습니다")
	}
	cfg, revision, e := approvalConfigurationTx(ctx, tx)
	if e != nil {
		return out, e
	}
	policy, e := approvalPolicyTx(ctx, tx, cfg, resource.WorkspaceID, resource.SpaceID, kind)
	if e != nil {
		return out, e
	}
	if len(comment) > 10000 {
		return out, approvalProblem(400, "검토 요청 의견은 10,000자 이하입니다")
	}
	assignments := map[[2]int][]string{}
	total := 0
	for si, stage := range policy.Stages {
		for gi, gate := range stage.Gates {
			ids, e := s.approvalCandidatesTx(ctx, tx, gate, resource, p.ID)
			if e != nil {
				return out, e
			}
			if len(ids) == 0 {
				return out, approvalProblem(409, fmt.Sprintf("%s / %s에 대상 조회 권한이 있는 다른 검토 담당자가 없습니다", stage.Name, gate.Name))
			}
			total += len(ids)
			if total > 2000 {
				return out, approvalProblem(400, "요청당 검토 후보는 최대 2,000명입니다")
			}
			assignments[[2]int{si, gi}] = ids
		}
	}
	if adapter.Submit != nil {
		identity := resource
		resource, e = adapter.Submit(ctx, tx, p, resource)
		if e != nil {
			return out, e
		}
		if resource.Kind != identity.Kind || resource.ID != identity.ID || resource.WorkspaceID != identity.WorkspaceID || resource.SpaceID != identity.SpaceID || resource.OwnerID != identity.OwnerID || resource.DocumentID != identity.DocumentID || resource.Version < identity.Version {
			return out, approvalProblem(409, "승인 대상의 범위 또는 소유자가 제출 중 변경되었습니다")
		}
	}
	if len(jsonValue(resource.Snapshot)) > 5<<20 {
		return out, approvalProblem(400, "승인 snapshot은 5MB 이하입니다")
	}
	out = approvalRequest{ID: newID(), WorkspaceID: policy.WorkspaceID, DocumentID: resource.DocumentID, ResourceKind: kind, ResourceID: resourceID, ResourceVersion: resource.Version, ResourceHash: approvalSnapshotHash(resource.Snapshot), Snapshot: resource.Snapshot, PolicyID: policy.ID, PolicyVersion: policy.Version, PolicyRevision: revision, Policy: policy, RequesterID: p.ID, OwnerID: resource.OwnerID, Status: "pending", Version: 1, Comment: comment}
	if _, e = tx.Exec(ctx, `UPDATE approval_requests SET status='superseded',version=version+1,updated_at=now(),completed_at=now() WHERE resource_kind=$1 AND resource_id=$2 AND status='pending'`, kind, resourceID); e != nil {
		return out, e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO approval_requests(id,workspace_id,document_id,resource_kind,resource_id,resource_version,resource_hash,snapshot,policy_id,policy_version,policy_revision,policy_snapshot,requester_id,owner_id,status,comment) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid,$10,$11,$12,$13,$14,'pending',$15)`, out.ID, out.WorkspaceID, out.DocumentID, kind, resourceID, resource.Version, out.ResourceHash, jsonValue(resource.Snapshot), policy.ID, policy.Version, revision, jsonValue(policy), p.ID, out.OwnerID, comment); e != nil {
		return out, e
	}
	for position, ids := range assignments {
		for _, uid := range ids {
			if _, e = tx.Exec(ctx, `INSERT INTO approval_assignments(request_id,stage_index,gate_index,user_id) VALUES($1,$2,$3,$4)`, out.ID, position[0], position[1], uid); e != nil {
				return out, e
			}
		}
	}
	if e = approvalNotifyStageTx(ctx, tx, out); e != nil {
		return out, e
	}
	return out, nil
}

func approvalNotifyStageTx(ctx context.Context, tx pgx.Tx, request approvalRequest) error {
	_, e := tx.Exec(ctx, `INSERT INTO notifications(id,user_id,title,document_id)
 SELECT gen_random_uuid(),a.user_id,$3,NULLIF($4,'')::uuid FROM approval_assignments a WHERE a.request_id=$1 AND a.stage_index=$2 GROUP BY a.user_id`, request.ID, request.CurrentStage, request.Policy.Name+" · "+request.Policy.Stages[request.CurrentStage].Name+" 검토 요청", request.DocumentID)
	return e
}

func approvalCurrentTx(ctx context.Context, tx pgx.Tx, request approvalRequest, resource ApprovalResource) (bool, string, error) {
	cfg, revision, e := approvalConfigurationTx(ctx, tx)
	if e != nil {
		return false, "", e
	}
	policy, e := approvalPolicyTx(ctx, tx, cfg, request.WorkspaceID, resource.SpaceID, request.ResourceKind)
	if e != nil {
		return false, "", e
	}
	if request.PolicyRevision != revision || request.PolicyID != policy.ID || request.PolicyVersion != policy.Version {
		return false, "승인 정책이 변경되었습니다. 현재 정책으로 다시 검토를 요청하세요", nil
	}
	if request.ResourceVersion != resource.Version || approvalSnapshotHash(resource.Snapshot) != request.ResourceHash {
		return false, "검토 요청 후 대상 내용이 변경되었습니다. 새 버전으로 다시 검토를 요청하세요", nil
	}
	return true, "", nil
}

func (s *Server) approvalEligibleGatesTx(ctx context.Context, tx pgx.Tx, request approvalRequest, p *Principal, resource ApprovalResource) ([]int, error) {
	if p == nil || p.Kind != "user" || p.TokenID != "" || p.ScopeRestricted || p.ID == request.OwnerID || p.ID == request.RequesterID || request.Status != "pending" {
		return []int{}, nil
	}
	if request.CurrentStage < 0 || request.CurrentStage >= len(request.Policy.Stages) {
		return nil, approvalProblem(409, "검토 단계 정보가 유효하지 않습니다")
	}
	out := []int{}
	for index, gate := range request.Policy.Stages[request.CurrentStage].Gates {
		var assigned, decided bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM approval_assignments WHERE request_id=$1 AND stage_index=$2 AND gate_index=$3 AND user_id=$4),EXISTS(SELECT 1 FROM approval_decisions WHERE request_id=$1 AND stage_index=$2 AND gate_index=$3)`, request.ID, request.CurrentStage, index, p.ID).Scan(&assigned, &decided); e != nil {
			return nil, e
		}
		if !assigned || decided {
			continue
		}
		current, e := s.approvalCandidatesTx(ctx, tx, gate, resource, request.RequesterID)
		if e != nil {
			return nil, e
		}
		if slices.Contains(current, p.ID) {
			out = append(out, index)
		}
	}
	return out, nil
}

// The resource's source document must already be locked by the caller. The
// returned state is committed together with resource-specific publication.
func (s *Server) decideApprovalTx(ctx context.Context, tx pgx.Tx, p *Principal, resource ApprovalResource, requestID string, expected int64, gateIndex *int, reject bool, comment string) (approvalRequest, error) {
	var out approvalRequest
	if !validID(requestID) || expected < 1 {
		return out, approvalProblem(400, "검토한 요청 ID와 요청 버전을 함께 보내세요")
	}
	out, e := approvalReadRequestTx(ctx, tx, requestID, false)
	if e != nil {
		return out, e
	}
	if out.ResourceKind != resource.Kind || out.ResourceID != resource.ID || out.WorkspaceID != resource.WorkspaceID || out.Status != "pending" || out.Version != expected {
		return out, approvalProblem(409, "승인 요청 상태가 변경되었습니다. 검토 화면을 다시 불러오세요")
	}
	valid, reason, e := approvalCurrentTx(ctx, tx, out, resource)
	if e != nil {
		return out, e
	}
	if !valid {
		return out, approvalProblem(409, reason)
	}
	out, e = approvalReadRequestTx(ctx, tx, requestID, true)
	if e != nil {
		return out, e
	}
	if out.Status != "pending" || out.Version != expected {
		return out, approvalProblem(409, "승인 요청 상태가 변경되었습니다. 검토 화면을 다시 불러오세요")
	}
	eligible, e := s.approvalEligibleGatesTx(ctx, tx, out, p, resource)
	if e != nil {
		return out, e
	}
	if len(eligible) == 0 {
		return out, approvalProblem(403, "현재 단계의 다른 검토 담당자만 결정할 수 있습니다")
	}
	selected := -1
	if gateIndex != nil {
		selected = *gateIndex
	} else if len(eligible) == 1 {
		selected = eligible[0]
	}
	if !slices.Contains(eligible, selected) {
		return out, approvalProblem(400, "처리할 검토 대상을 선택하세요")
	}
	if len(comment) > 10000 || (reject && strings.TrimSpace(comment) == "") {
		return out, approvalProblem(400, "의견은 10,000자 이하이며 반려 사유는 필수입니다")
	}
	decision := "approved"
	if reject {
		decision = "rejected"
	}
	if _, e = tx.Exec(ctx, `INSERT INTO approval_decisions(id,request_id,stage_index,gate_index,user_id,decision,comment) VALUES($1,$2,$3,$4,$5,$6,$7)`, newID(), out.ID, out.CurrentStage, selected, p.ID, decision, comment); e != nil {
		return out, e
	}
	if reject {
		out.Status = "rejected"
	} else {
		var approved int
		if e = tx.QueryRow(ctx, `SELECT count(*) FROM approval_decisions WHERE request_id=$1 AND stage_index=$2 AND decision='approved'`, out.ID, out.CurrentStage).Scan(&approved); e != nil {
			return out, e
		}
		stage := out.Policy.Stages[out.CurrentStage]
		if stage.Mode == "any" || approved == len(stage.Gates) {
			if out.CurrentStage+1 == len(out.Policy.Stages) {
				out.Status = "approved"
			} else {
				out.CurrentStage++
				if e = approvalNotifyStageTx(ctx, tx, out); e != nil {
					return out, e
				}
			}
		}
	}
	out.Version++
	if _, e = tx.Exec(ctx, `UPDATE approval_requests SET status=$2,current_stage=$3,version=$4,updated_at=now(),completed_at=CASE WHEN $2='pending' THEN NULL ELSE now() END WHERE id=$1`, out.ID, out.Status, out.CurrentStage, out.Version); e != nil {
		return out, e
	}
	if out.Status != "pending" {
		title := out.Policy.Name + " 검토가 승인되었습니다"
		if reject {
			title = out.Policy.Name + " 검토가 반려되었습니다"
		}
		for _, uid := range slices.Compact([]string{out.RequesterID, out.OwnerID}) {
			if _, e = tx.Exec(ctx, `INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,$3,NULLIF($4,'')::uuid)`, newID(), uid, title, out.DocumentID); e != nil {
				return out, e
			}
		}
		if adapter := s.approvalAdapters[resource.Kind]; adapter.Complete != nil {
			if e = adapter.Complete(ctx, tx, p, out, resource); e != nil {
				return out, e
			}
		}
	}
	return out, nil
}
