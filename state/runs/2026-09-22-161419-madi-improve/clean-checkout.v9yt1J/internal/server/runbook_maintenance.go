package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// StartRunbookMaintenance reconciles cancelled/abandoned queue deliveries. It
// never launches a remote job. Old credentials are retained solely to stop an
// already identified job even after a runner is disabled or reconfigured.
func (s *Server) StartRunbookMaintenance(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			if err := s.reconcileRunbooks(ctx); err != nil && ctx.Err() == nil {
				slog.Error("runbook_reconcile", "error", "격리 작업 중지 상태를 확인하지 못했습니다")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *Server) reconcileRunbooks(ctx context.Context) error {
	// A short durable reservation avoids holding a pool connection while calling
	// the runner. Concurrent replicas can safely repeat cancellation, never launch.
	rows, err := s.DB.Query(ctx, `WITH candidates AS (
	 SELECT e.id FROM runbook_executions e LEFT JOIN automation_jobs j ON j.id=e.job_id
	 WHERE e.status IN ('queued','running','unknown') AND e.reconcile_after<=now()
	 AND (e.cancel_requested OR (e.status IN ('queued','running') AND (j.id IS NULL OR j.status IN ('failed','cancelled','succeeded'))))
	 ORDER BY e.reconcile_after LIMIT 10 FOR UPDATE OF e SKIP LOCKED
	) UPDATE runbook_executions e SET reconcile_after=now()+interval '90 seconds' FROM candidates c WHERE e.id=c.id RETURNING e.id::text`)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err = s.haltRunbookExecution(id, "취소되거나 작업자가 종료된 실행의 중지를 확인합니다"); err != nil {
			slog.Warn("runbook_external_stop_unconfirmed", "execution_id", id)
		}
	}
	return nil
}

func (s *Server) haltRunbookExecution(id, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	var raw []byte
	var status string
	if err := s.DB.QueryRow(ctx, `SELECT snapshot,status FROM runbook_executions WHERE id=$1`, id).Scan(&raw, &status); err != nil {
		return err
	}
	if !oneOf(status, "queued", "running", "unknown") {
		return nil
	}
	var plan runbookPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return err
	}
	terminal := "cancelled"
	var last error
	for index, step := range plan.Steps {
		var state, external string
		if err := s.DB.QueryRow(ctx, `SELECT state,external_id FROM runbook_execution_steps WHERE execution_id=$1 AND step_index=$2`, id, index).Scan(&state, &external); err != nil {
			return err
		}
		if oneOf(state, "succeeded", "failed", "cancelled") {
			continue
		}
		if state == "queued" {
			// CAS closes the race with a worker about to launch. A worker that
			// already owns launching must be stopped using its remote handle.
			tag, err := s.DB.Exec(ctx, `UPDATE runbook_execution_steps SET state='cancelled',completed_at=now(),last_error=$3 WHERE execution_id=$1 AND step_index=$2 AND state='queued'`, id, index, reason)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 1 {
				continue
			}
			if err = s.DB.QueryRow(ctx, `SELECT state,external_id FROM runbook_execution_steps WHERE execution_id=$1 AND step_index=$2`, id, index).Scan(&state, &external); err != nil {
				return err
			}
		}
		runner, err := s.loadRunbookRunner(ctx, step.RunnerID, step.RunnerRevision)
		if err != nil {
			terminal = "unknown"
			last = err
			continue
		}
		remote, err := s.newRunbookRemote(runner)
		if err != nil {
			terminal = "unknown"
			last = err
			continue
		}
		result, err := s.stopRunbookStep(id, index, step, remote, external, reason)
		remote.close()
		if err != nil || result == "unknown" {
			terminal = "unknown"
			last = err
		}
	}
	if err := s.runbookFinish(id, terminal, reason, plan); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return last
}
