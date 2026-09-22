package server

import "context"

func (s *Server) expireDistributions(ctx context.Context) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return
	}
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, `UPDATE knowledge_distribution_exports d SET status='expired' WHERE status IN ('queued','awaiting_review','ready') AND (expires_at<=clock_timestamp() OR NOT EXISTS(SELECT 1 FROM export_runs e WHERE e.id=d.export_id AND e.status='ready' AND e.expires_at>clock_timestamp()) OR NOT EXISTS(SELECT 1 FROM knowledge_distribution_policy p WHERE p.id=1 AND p.enabled AND p.revision=d.policy_revision) OR NOT EXISTS(SELECT 1 FROM knowledge_distribution_keys k WHERE k.id=d.signing_key_id AND k.revoked_at IS NULL))`)
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE knowledge_distribution_exports d SET status='failed' FROM automation_jobs j WHERE d.job_id=j.id AND d.status='queued' AND j.status IN ('failed','cancelled')`)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `DELETE FROM knowledge_distribution_artifacts a USING knowledge_distribution_exports d WHERE a.export_id=d.id AND d.status IN ('expired','revoked','failed')`)
	}
	if e == nil {
		_ = tx.Commit(ctx)
	}
}
