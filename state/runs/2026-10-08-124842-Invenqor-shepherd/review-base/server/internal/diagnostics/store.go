package diagnostics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/apitime"
	"github.com/hkjang/invenqor/server/internal/storage"
)

const (
	retentionPeriod = 30 * 24 * time.Hour
	retentionCount  = 10_000
)

var (
	invenqorSecret = regexp.MustCompile(`ivq_(?:at|et|ec|sk)_[A-Za-z0-9_-]+`)
	bearerSecret   = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]+`)
	urlPassword    = regexp.MustCompile(`(://[^:/\s]+:)[^@\s]+(@)`)
)

type Event struct {
	Level      string         `json:"level"`
	Component  string         `json:"component"`
	EventCode  string         `json:"event_code"`
	Message    string         `json:"message"`
	RequestID  string         `json:"request_id"`
	InstanceID string         `json:"instance_id"`
	AgentID    string         `json:"agent_id"`
	SourceIP   string         `json:"source_ip"`
	Details    map[string]any `json:"details"`
}

type Item struct {
	ID         string         `json:"id"`
	OccurredAt apitime.Time   `json:"occurred_at"`
	Level      string         `json:"level"`
	Component  string         `json:"component"`
	EventCode  string         `json:"event_code"`
	Message    string         `json:"message"`
	RequestID  string         `json:"request_id"`
	InstanceID string         `json:"instance_id"`
	AgentID    string         `json:"agent_id"`
	SourceIP   string         `json:"source_ip"`
	Details    map[string]any `json:"details"`
}

type Filter struct {
	Level      string
	Component  string
	InstanceID string
	Query      string
	Limit      int
}

type Store struct {
	database   *sql.DB
	instanceID string
	writes     atomic.Uint64
}

func NewStore(database *sql.DB) *Store {
	instanceID, _ := os.Hostname()
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "server-" + uuid.NewString()[:8]
	}
	if len(instanceID) > 128 {
		instanceID = instanceID[:128]
	}
	return &Store{database: database, instanceID: instanceID}
}

func (store *Store) InstanceID() string {
	return store.instanceID
}

func (store *Store) Record(ctx context.Context, event Event) error {
	event.Level = strings.ToLower(strings.TrimSpace(event.Level))
	if event.Level != "info" && event.Level != "warning" &&
		event.Level != "error" {
		event.Level = "error"
	}
	event.Component = bounded(strings.TrimSpace(event.Component), 64)
	event.EventCode = bounded(strings.TrimSpace(event.EventCode), 128)
	event.Message = bounded(Sanitize(event.Message), 2_000)
	event.RequestID = bounded(strings.TrimSpace(event.RequestID), 128)
	event.AgentID = bounded(strings.TrimSpace(event.AgentID), 128)
	event.SourceIP = bounded(strings.TrimSpace(event.SourceIP), 128)
	event.InstanceID = store.instanceID
	if event.Component == "" {
		event.Component = "server"
	}
	if event.EventCode == "" {
		event.EventCode = "SERVER_DIAGNOSTIC"
	}
	details, err := json.Marshal(sanitizeValue(event.Details))
	if err != nil {
		return fmt.Errorf("encode diagnostic details: %w", err)
	}
	if len(details) > 16*1024 {
		details = []byte(`{"truncated":true}`)
	}
	_, err = store.database.ExecContext(
		ctx,
		`INSERT INTO diagnostic_logs(
			id,occurred_at,level,component,event_code,message,request_id,
			instance_id,agent_id,source_ip,details_json
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		uuid.NewString(),
		time.Now().UTC(),
		event.Level,
		event.Component,
		event.EventCode,
		event.Message,
		event.RequestID,
		event.InstanceID,
		event.AgentID,
		event.SourceIP,
		string(details),
	)
	if err != nil {
		return fmt.Errorf("store diagnostic event: %w", err)
	}
	if store.writes.Add(1)%100 == 0 {
		store.prune(ctx)
	}
	return nil
}

// List returns the matching events newest first, the total the filter matches
// (so the console can say the limit truncated the answer rather than implying
// there is nothing more), and the facets its filter controls are built from.
func (store *Store) List(
	ctx context.Context,
	filter Filter,
) ([]Item, int64, Facets, error) {
	if filter.Limit < 1 {
		filter.Limit = 100
	}
	if filter.Limit > 500 {
		filter.Limit = 500
	}
	where := " WHERE 1=1"
	arguments := make([]any, 0, 5)
	addCondition := func(column string, value any) {
		arguments = append(arguments, value)
		where += fmt.Sprintf(" AND %s=$%d", column, len(arguments))
	}
	if filter.Level != "" {
		addCondition("level", filter.Level)
	}
	if filter.Component != "" {
		addCondition("component", filter.Component)
	}
	if filter.InstanceID != "" {
		addCondition("instance_id", filter.InstanceID)
	}
	if query := strings.ToLower(strings.TrimSpace(filter.Query)); query != "" {
		arguments = append(arguments, storage.LikeContains(query))
		where += fmt.Sprintf(
			` AND LOWER(event_code || ' ' || message || ' ' ||
			 request_id || ' ' || agent_id || ' ' || source_ip || ' ' ||
			 instance_id) LIKE $%d`+storage.LikeEscapeClause,
			len(arguments),
		)
	}
	var total int64
	if err := store.database.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM diagnostic_logs"+where,
		arguments...,
	).Scan(&total); err != nil {
		return nil, 0, Facets{}, fmt.Errorf("count diagnostic events: %w", err)
	}
	statement := `SELECT id,occurred_at,level,component,event_code,message,
		 request_id,instance_id,agent_id,source_ip,details_json
		 FROM diagnostic_logs` + where
	arguments = append(arguments, filter.Limit)
	statement += fmt.Sprintf(
		" ORDER BY occurred_at DESC LIMIT $%d",
		len(arguments),
	)
	rows, err := store.database.QueryContext(
		ctx,
		statement,
		arguments...,
	)
	if err != nil {
		return nil, 0, Facets{}, fmt.Errorf("list diagnostic events: %w", err)
	}
	defer rows.Close()
	items := make([]Item, 0, filter.Limit)
	for rows.Next() {
		var item Item
		var details any
		if err := rows.Scan(
			&item.ID,
			&item.OccurredAt,
			&item.Level,
			&item.Component,
			&item.EventCode,
			&item.Message,
			&item.RequestID,
			&item.InstanceID,
			&item.AgentID,
			&item.SourceIP,
			&details,
		); err != nil {
			return nil, 0, Facets{}, err
		}
		item.Details = decodeDetails(details)
		items = append(items, item)
	}
	facets, err := store.facets(ctx)
	if err != nil {
		return nil, 0, Facets{}, err
	}
	return items, total, facets, rows.Err()
}

// Since returns the recorded events for the named components inside a time
// window, newest first. The caller aggregates in Go so the same query works on
// PostgreSQL and on the SQLite start-up fallback.
func (store *Store) Since(
	ctx context.Context,
	components []string,
	since time.Time,
	limit int,
) ([]Item, error) {
	if len(components) == 0 {
		return nil, nil
	}
	if limit < 1 {
		limit = 1_000
	}
	if limit > 5_000 {
		limit = 5_000
	}
	arguments := make([]any, 0, len(components)+2)
	placeholders := make([]string, 0, len(components))
	for _, component := range components {
		arguments = append(arguments, component)
		placeholders = append(
			placeholders,
			fmt.Sprintf("$%d", len(arguments)),
		)
	}
	arguments = append(arguments, since.UTC())
	sinceArgument := len(arguments)
	arguments = append(arguments, limit)
	statement := fmt.Sprintf(
		`SELECT id,occurred_at,level,component,event_code,message,
		 request_id,instance_id,agent_id,source_ip,details_json
		 FROM diagnostic_logs
		 WHERE component IN (%s) AND occurred_at >= $%d
		 ORDER BY occurred_at DESC LIMIT $%d`,
		strings.Join(placeholders, ","),
		sinceArgument,
		len(arguments),
	)
	rows, err := store.database.QueryContext(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list diagnostic activity: %w", err)
	}
	defer rows.Close()
	items := make([]Item, 0, 64)
	for rows.Next() {
		var item Item
		var details any
		if err := rows.Scan(
			&item.ID,
			&item.OccurredAt,
			&item.Level,
			&item.Component,
			&item.EventCode,
			&item.Message,
			&item.RequestID,
			&item.InstanceID,
			&item.AgentID,
			&item.SourceIP,
			&details,
		); err != nil {
			return nil, err
		}
		item.Details = decodeDetails(details)
		items = append(items, item)
	}
	return items, rows.Err()
}

// Facets are the values the log actually holds. The console builds its filter
// controls from these: the component list used to be written by hand in the
// console, so components the server had started recording - agent_preflight and
// keycloak among them - could not be filtered at all.
type Facets struct {
	Instances  []string `json:"instances"`
	Components []string `json:"components"`
	EventCodes []string `json:"event_codes"`
}

func (store *Store) facets(ctx context.Context) (Facets, error) {
	instances, err := store.distinct(ctx, "instance_id", 200)
	if err != nil {
		return Facets{}, err
	}
	components, err := store.distinct(ctx, "component", 100)
	if err != nil {
		return Facets{}, err
	}
	codes, err := store.distinct(ctx, "event_code", 300)
	if err != nil {
		return Facets{}, err
	}
	return Facets{
		Instances:  instances,
		Components: components,
		EventCodes: codes,
	}, nil
}

// distinct lists the values present in one column. The column name is never
// caller-supplied - it comes from the fixed set above.
func (store *Store) distinct(
	ctx context.Context,
	column string,
	limit int,
) ([]string, error) {
	rows, err := store.database.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT %s FROM diagnostic_logs
			 GROUP BY %s ORDER BY COUNT(*) DESC, %s LIMIT %d`,
			column, column, column, limit,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("list distinct %s: %w", column, err)
	}
	defer rows.Close()
	result := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (store *Store) prune(ctx context.Context) {
	_, _ = store.database.ExecContext(
		ctx,
		"DELETE FROM diagnostic_logs WHERE occurred_at < $1",
		time.Now().UTC().Add(-retentionPeriod),
	)
	rows, err := store.database.QueryContext(
		ctx,
		`SELECT id FROM diagnostic_logs
		 ORDER BY occurred_at DESC LIMIT 1000 OFFSET $1`,
		retentionCount,
	)
	if err != nil {
		return
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	_ = rows.Close()
	for _, id := range ids {
		_, _ = store.database.ExecContext(
			ctx,
			"DELETE FROM diagnostic_logs WHERE id=$1",
			id,
		)
	}
}

func Sanitize(value string) string {
	value = invenqorSecret.ReplaceAllString(value, "[REDACTED]")
	value = bearerSecret.ReplaceAllString(value, "${1}[REDACTED]")
	return urlPassword.ReplaceAllString(value, "${1}[REDACTED]${2}")
}

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case string:
		return Sanitize(typed)
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") ||
				strings.Contains(lower, "secret") ||
				strings.Contains(lower, "password") ||
				strings.Contains(lower, "authorization") {
				result[key] = "[REDACTED]"
			} else {
				result[key] = sanitizeValue(child)
			}
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, child := range typed {
			result[index] = sanitizeValue(child)
		}
		return result
	default:
		return value
	}
}

func decodeDetails(value any) map[string]any {
	var raw []byte
	switch typed := value.(type) {
	case string:
		raw = []byte(typed)
	case []byte:
		raw = typed
	default:
		raw, _ = json.Marshal(typed)
	}
	result := map[string]any{}
	_ = json.Unmarshal(raw, &result)
	return result
}

func bounded(value string, maximum int) string {
	if len(value) > maximum {
		return value[:maximum]
	}
	return value
}
