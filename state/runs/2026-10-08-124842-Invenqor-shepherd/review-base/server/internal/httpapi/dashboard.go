package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

type statisticBucket struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type dailyStatistic struct {
	Date   string `json:"date"`
	Events int64  `json:"events"`
	Failed int64  `json:"failed"`
}

func (s *Server) dashboardStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UTC()
	assetScope := ""
	if r.URL.Query().Get("scope") == "managed" {
		assetScope = " AND type <> 'process'"
	}
	assetTotal, err := scalarCount(
		ctx, s.database.DB(),
		`SELECT COUNT(*) FROM assets WHERE deleted_at IS NULL`+assetScope,
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	seen24h, err := scalarCount(
		ctx, s.database.DB(),
		`SELECT COUNT(*) FROM assets
		 WHERE deleted_at IS NULL AND last_seen_at >= $1`+assetScope,
		now.Add(-24*time.Hour),
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	// The fleet expected to report, which is not every row ever written.
	// Blocking is how an operator retires a machine - the console offers it next
	// to each Agent and nothing deletes the row - so counting blocked Agents here
	// made "needs attention" climb forever and made clicking 차단 appear to do
	// nothing. The release distribution already counts only unblocked Agents.
	agentTotal, err := scalarCount(
		ctx, s.database.DB(),
		`SELECT COUNT(*) FROM agents WHERE blocked_at IS NULL`,
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	agentHealthy, err := scalarCount(
		ctx, s.database.DB(),
		`SELECT COUNT(*) FROM agents
		 WHERE blocked_at IS NULL AND last_seen_at >= $1`,
		now.Add(-30*time.Minute),
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	event24h, err := scalarCount(
		ctx, s.database.DB(),
		`SELECT COUNT(*) FROM agent_events WHERE received_at >= $1`,
		now.Add(-24*time.Hour),
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	failed24h, err := scalarCount(
		ctx, s.database.DB(),
		`SELECT COUNT(*) FROM agent_events
		 WHERE received_at >= $1 AND processing_status = 'failed'`,
		now.Add(-24*time.Hour),
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	assetDimensions := make(map[string][]statisticBucket)
	for key, column := range map[string]string{
		"by_type":        "type",
		"by_status":      "status",
		"by_environment": "environment",
		"by_criticality": "criticality",
		"by_source":      "source",
	} {
		buckets, err := groupedCounts(
			ctx,
			s.database.DB(),
			`SELECT COALESCE(NULLIF(`+column+`, ''), 'unknown'), COUNT(*)
			 FROM assets WHERE deleted_at IS NULL`+assetScope+`
			 GROUP BY `+column+` ORDER BY COUNT(*) DESC, `+column+` LIMIT 12`,
		)
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		assetDimensions[key] = buckets
	}
	agentStatus, err := groupedCounts(
		ctx,
		s.database.DB(),
		`SELECT COALESCE(NULLIF(status, ''), 'unknown'), COUNT(*)
		 FROM agents GROUP BY status ORDER BY COUNT(*) DESC, status`,
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	agentOS, err := groupedCounts(
		ctx,
		s.database.DB(),
		`SELECT COALESCE(NULLIF(os_name, ''), 'unknown'), COUNT(*)
		 FROM agents GROUP BY os_name ORDER BY COUNT(*) DESC, os_name LIMIT 10`,
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	daily, err := dailyEventCounts(ctx, s.database.DB(), now)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"generated_at": now,
		"assets": map[string]any{
			"total":          assetTotal,
			"seen_24h":       seen24h,
			"stale":          assetTotal - seen24h,
			"by_type":        assetDimensions["by_type"],
			"by_status":      assetDimensions["by_status"],
			"by_environment": assetDimensions["by_environment"],
			"by_criticality": assetDimensions["by_criticality"],
			"by_source":      assetDimensions["by_source"],
		},
		"agents": map[string]any{
			"total":     agentTotal,
			"healthy":   agentHealthy,
			"attention": agentTotal - agentHealthy,
			"by_status": agentStatus,
			"by_os":     agentOS,
		},
		"collection": map[string]any{
			"events_24h": event24h,
			"failed_24h": failed24h,
			"daily":      daily,
		},
	})
}

func scalarCount(
	ctx context.Context,
	database *sql.DB,
	query string,
	args ...any,
) (int64, error) {
	var count int64
	err := database.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func groupedCounts(
	ctx context.Context,
	database *sql.DB,
	query string,
	args ...any,
) ([]statisticBucket, error) {
	rows, err := database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]statisticBucket, 0)
	for rows.Next() {
		var item statisticBucket
		if err := rows.Scan(&item.Label, &item.Count); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// dateExpression truncates a timestamp column to its calendar day on both
// engines. SQLite holds the driver's Go time layout ("2006-01-02 15:04:05.999
// -0700 MST"), which SQLite's own DATE() cannot parse and silently returns NULL
// for - so every daily series was empty in the start-up fallback mode. Casting
// to text and taking the leading ten characters yields the same ISO day on
// PostgreSQL and SQLite alike.
func dateExpression(column string) string {
	return "substr(CAST(" + column + " AS TEXT),1,10)"
}

func dailyEventCounts(
	ctx context.Context,
	database *sql.DB,
	now time.Time,
) ([]dailyStatistic, error) {
	start := now.AddDate(0, 0, -6)
	day := dateExpression("received_at")
	rows, err := database.QueryContext(
		ctx,
		`SELECT `+day+`, COUNT(*),
		 SUM(CASE WHEN processing_status = 'failed' THEN 1 ELSE 0 END)
		 FROM agent_events WHERE received_at >= $1
		 GROUP BY `+day+` ORDER BY `+day,
		start,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byDate := make(map[string]dailyStatistic)
	for rows.Next() {
		var date any
		var events, failed int64
		if err := rows.Scan(&date, &events, &failed); err != nil {
			return nil, err
		}
		key := dateString(date)
		byDate[key] = dailyStatistic{Date: key, Events: events, Failed: failed}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]dailyStatistic, 0, 7)
	for day := 6; day >= 0; day-- {
		key := now.AddDate(0, 0, -day).Format("2006-01-02")
		item := byDate[key]
		item.Date = key
		result = append(result, item)
	}
	return result, nil
}

func dateString(value any) string {
	switch typed := value.(type) {
	case time.Time:
		return typed.UTC().Format("2006-01-02")
	case string:
		if len(typed) >= 10 {
			return typed[:10]
		}
	case []byte:
		if len(typed) >= 10 {
			return string(typed[:10])
		}
	}
	return ""
}
