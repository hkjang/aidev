package server

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// Called by saveDocument before taking its target FOR UPDATE lock. An origin
// target is one of these two documents. Taking both in UUID order avoids the
// cross-target A->B/B->A lock inversion during concurrent merges.
func (s *Server) prepareProposalConflictMergeTx(r *http.Request, tx pgx.Tx) error {
	intent, ok := r.Context().Value(documentProposalMergeKey{}).(documentProposalMerge)
	if !ok {
		return nil
	}
	return lockProposalConflictSourcesTx(r, tx, intent.ID, true)
}
func lockProposalConflictSourcesTx(r *http.Request, tx pgx.Tx, id string, write bool) error {
	var sources []string
	e := tx.QueryRow(r.Context(), `SELECT ARRAY[o.left_id::text,o.right_id::text] FROM knowledge_proposal_conflict_origins o JOIN knowledge_proposals p ON p.id=o.proposal_id WHERE o.proposal_id=$1 AND madi_document_allowed($2,p.document_id,false) AND madi_proposal_origin_allowed($2,p.id)`, id, current(r).ID).Scan(&sources)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	lock := " FOR SHARE"
	if write {
		lock = " FOR UPDATE"
	}
	rows, e := tx.Query(r.Context(), `SELECT id FROM documents WHERE id=ANY($1::uuid[]) ORDER BY id`+lock, sources)
	if e != nil {
		return e
	}
	for rows.Next() {
	}
	e = rows.Err()
	rows.Close()
	return e
}
func (s *Server) proposalConflictOriginTx(r *http.Request, tx pgx.Tx, id string) (map[string]any, error) {
	if e := lockProposalConflictSourcesTx(r, tx, id, false); e != nil {
		return nil, e
	}
	var cid, wid, actor, left, right, lh, rh string
	var revision, lv, rv int
	e := tx.QueryRow(r.Context(), `SELECT o.candidate_id::text,d.workspace_id::text,o.actor_id::text,o.candidate_revision,o.left_id::text,o.right_id::text,o.left_version,o.right_version,o.left_hash,o.right_hash FROM knowledge_proposal_conflict_origins o JOIN knowledge_proposals p ON p.id=o.proposal_id JOIN documents d ON d.id=p.document_id WHERE o.proposal_id=$1 FOR SHARE OF o`, id).Scan(&cid, &wid, &actor, &revision, &left, &right, &lv, &rv, &lh, &rh)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	sources, e := s.conflictSourcesTx(r, tx, wid, []string{left, right})
	if e != nil {
		return nil, e
	}
	c, e := conflictCandidateTx(r, tx, cid, false, false)
	if e != nil {
		return nil, e
	}
	// Recheck the encrypted captured excerpt against today's protection policy;
	// the private report itself is never returned to a proposal reader.
	if _, e = s.conflictCandidateViewTx(r, tx, c, false); e != nil {
		return nil, e
	}
	stale := sources[left].Version != lv || sources[right].Version != rv || sources[left].Hash != lh || sources[right].Hash != rh || c.Revision != revision
	out := map[string]any{"kind": "conflict_comparison", "stale": stale, "left_document_id": left, "right_document_id": right, "left_version": lv, "right_version": rv, "candidate_revision": revision, "notice": "두 원문의 차이에 대한 개인 판단에서 연결한 제안입니다. 사실성 확인·게시 승인을 의미하지 않으며 두 원문 접근 권한이 모두 필요합니다."}
	if actor == current(r).ID {
		out["candidate_id"] = cid
		out["run_id"] = c.RunID
	}
	return out, nil
}
func (s *Server) validateProposalConflictCreationTx(r *http.Request, tx pgx.Tx, target, cid string, revision int, consent bool) (*conflictCandidate, error) {
	if cid == "" {
		if revision != 0 || consent {
			return nil, approvalProblem(400, "비교 출처 ID와 동의가 일치하지 않습니다")
		}
		return nil, nil
	}
	if !personalAccessRequest(current(r)) || !validID(cid) || revision < 1 || !consent {
		return nil, approvalProblem(400, "본인 비교 후보와 현재 판단 버전·출처 연결 동의를 확인하세요")
	}
	c, e := conflictCandidateTx(r, tx, cid, true, false)
	if e != nil {
		return nil, e
	}
	if target != c.LeftID && target != c.RightID {
		return nil, approvalProblem(400, "비교한 두 원문 중 하나에만 변경안을 연결할 수 있습니다")
	}
	view, e := s.conflictCandidateViewTx(r, tx, c, false)
	if e != nil {
		return nil, e
	}
	if c.Revision != revision || boolean(view, "stale") {
		return nil, approvalProblem(409, "비교 원문 또는 판단 기록이 변경되었습니다. 새 분석으로 확인하세요")
	}
	return &c, nil
}
func insertProposalConflictOriginTx(r *http.Request, tx pgx.Tx, id string, c *conflictCandidate) error {
	if c == nil {
		return nil
	}
	_, e := tx.Exec(r.Context(), `INSERT INTO knowledge_proposal_conflict_origins(proposal_id,candidate_id,candidate_revision,left_id,right_id,left_version,right_version,left_hash,right_hash,actor_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, c.ID, c.Revision, c.LeftID, c.RightID, c.LeftVersion, c.RightVersion, c.LeftHash, c.RightHash, current(r).ID)
	return e
}

// After canonical merge the target's version/hash has intentionally changed;
// revalidate current actor and both source ACLs without treating our own write as
// stale. The source rows and candidate are still locked by the prepare phase.
func (s *Server) proposalConflictFinalACLTx(r *http.Request, tx pgx.Tx, id string) error {
	var wid, left, right string
	e := tx.QueryRow(r.Context(), `SELECT d.workspace_id::text,o.left_id::text,o.right_id::text FROM knowledge_proposal_conflict_origins o JOIN knowledge_proposals p ON p.id=o.proposal_id JOIN documents d ON d.id=p.document_id WHERE o.proposal_id=$1`, id).Scan(&wid, &left, &right)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	return s.conflictCurrentACLTx(r, tx, wid, []string{left, right})
}
