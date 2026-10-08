package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hkjang/invenqor/server/internal/diagnostics"
	"github.com/hkjang/invenqor/server/internal/ingest"
	"github.com/hkjang/invenqor/server/internal/storage"
)

// RunSpoolReplay continually checks the existing PostgreSQL connection. It
// never opens SQLite or changes database handles during a runtime outage.
func (s *Server) RunSpoolReplay(ctx context.Context, interval time.Duration) {
	if s.spool == nil {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if s.database.Mode() != storage.ModePostgresDegraded {
			if count, err := s.ReplaySpool(ctx); err != nil {
				s.logger.Warn("event_spool_replay_failed", "error", err)
			} else if count > 0 {
				s.logger.Info("event_spool_replayed", "events", count)
			}
		} else {
			pingContext, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := s.database.Ping(pingContext)
			cancel()
			if err == nil {
				s.logger.Info("postgres_connection_recovered")
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) ReplaySpool(ctx context.Context) (int, error) {
	if s.spool == nil {
		return 0, nil
	}
	lease, acquired, err := s.spool.AcquireReplayLease()
	if err != nil {
		return 0, err
	}
	if !acquired {
		return 0, nil
	}
	defer lease.Release()
	paths, err := s.spool.Pending()
	if err != nil {
		return 0, err
	}
	replayed := 0
	for _, path := range paths {
		if err := lease.Renew(); err != nil {
			return replayed, err
		}
		raw, err := s.spool.Read(path)
		if err != nil {
			return replayed, err
		}
		envelope, err := ingest.DecodeEnvelope(raw)
		if err != nil {
			if discardErr := s.discardSpooledEvent(
				ctx,
				path,
				"SPOOLED_EVENT_INVALID",
				"A corrupt or legacy spool segment was discarded after schema validation failed.",
				"",
				err,
			); discardErr != nil {
				return replayed, discardErr
			}
			continue
		}
		agent, err := s.agentService.GetByExternalID(ctx, envelope.AgentID)
		if errors.Is(err, sql.ErrNoRows) {
			if discardErr := s.discardSpooledEvent(
				ctx,
				path,
				"SPOOLED_EVENT_AGENT_NOT_FOUND",
				"A spooled event was discarded because its Agent no longer exists.",
				envelope.AgentID,
				err,
			); discardErr != nil {
				return replayed, discardErr
			}
			continue
		}
		if err != nil {
			return replayed, err
		}
		if strings.EqualFold(agent.Status, "blocked") {
			if discardErr := s.discardSpooledEvent(
				ctx,
				path,
				"SPOOLED_EVENT_AGENT_BLOCKED",
				"A spooled event was discarded because its Agent is blocked.",
				envelope.AgentID,
				errors.New("agent status is blocked"),
			); discardErr != nil {
				return replayed, discardErr
			}
			continue
		}
		if _, err := s.ingestService.Process(
			ctx, agent, envelope, raw, "",
		); err != nil {
			return replayed, err
		}
		if err := s.spool.Acknowledge(path); err != nil {
			return replayed, err
		}
		replayed++
	}
	return replayed, nil
}

func (s *Server) discardSpooledEvent(
	ctx context.Context,
	path string,
	eventCode string,
	message string,
	agentID string,
	cause error,
) error {
	segment := filepath.Base(path)
	details := map[string]any{"segment": segment}
	if cause != nil {
		details["error"] = cause.Error()
	}
	if s.diagnosticStore != nil {
		if err := s.diagnosticStore.Record(ctx, diagnostics.Event{
			Level:     "warning",
			Component: "event_spool",
			EventCode: eventCode,
			Message:   message,
			AgentID:   agentID,
			Details:   details,
		}); err != nil {
			return fmt.Errorf("record discarded spool diagnostic: %w", err)
		}
	}
	s.logger.Warn(
		"event_spool_segment_discarded",
		"event_code", eventCode,
		"segment", segment,
		"agent_id", agentID,
	)
	if err := s.spool.Acknowledge(path); err != nil {
		return fmt.Errorf("discard spool segment: %w", err)
	}
	return nil
}
