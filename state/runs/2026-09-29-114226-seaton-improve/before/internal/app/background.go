package app

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
)

func (s *Server) RunBackground(ctx context.Context) {
	cleanup := time.NewTicker(30 * time.Minute)
	scheduler := time.NewTicker(time.Minute)
	defer cleanup.Stop()
	defer scheduler.Stop()
	// 재시작으로 끊긴 분석 잡을 먼저 정리한다. 그대로 두면 도면이 analyzing
	// 상태에 갇혀 재분석이 막힌다.
	s.reclaimStaleAnalyses(ctx)
	var lastScheduled time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanup.C:
			_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at<now(); DELETE FROM oidc_states WHERE expires_at<now();`)
			if err != nil {
				s.logger.Error("background cleanup failed", "error", err)
			}
			s.reclaimStaleAnalyses(ctx)
		case now := <-scheduler.C:
			enabled, _ := s.getSetting(ctx, "hr.sync_enabled")
			expression, _ := s.getSetting(ctx, "hr.schedule")
			if enabled != "true" || expression == "" {
				continue
			}
			schedule, err := cron.ParseStandard(expression)
			if err != nil {
				s.logger.Error("invalid HR sync schedule", "schedule", expression, "error", err)
				continue
			}
			previousMinute := now.Truncate(time.Minute).Add(-time.Minute)
			due := schedule.Next(previousMinute)
			if due.After(now) || (!lastScheduled.IsZero() && !due.After(lastScheduled)) {
				continue
			}
			lastScheduled = due
			go func() {
				if _, err := s.runEmployeeSync(ctx); err != nil {
					s.logger.Error("scheduled HR sync failed", "error", err)
				}
			}()
		}
	}
}
