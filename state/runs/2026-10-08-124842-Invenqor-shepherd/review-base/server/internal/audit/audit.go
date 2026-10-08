package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Entry struct {
	ActorType    string
	ActorID      string
	ActorName    string
	Action       string
	ResourceType string
	ResourceID   string
	RequestID    string
	SourceIP     string
	UserAgent    string
	Result       string
	Reason       string
	Before       any
	After        any
	Metadata     any
}

type Recorder struct{}

func (Recorder) Record(ctx context.Context, database DBTX, entry Entry) error {
	before, err := optionalJSON(entry.Before)
	if err != nil {
		return fmt.Errorf("encode audit before value: %w", err)
	}
	after, err := optionalJSON(entry.After)
	if err != nil {
		return fmt.Errorf("encode audit after value: %w", err)
	}
	metadata, err := jsonOrEmpty(entry.Metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	_, err = database.ExecContext(
		ctx,
		`INSERT INTO audit_logs(
			id, actor_type, actor_id, actor_name, action, resource_type,
			resource_id, request_id, source_ip, user_agent, result, reason,
			before_json, after_json, metadata_json
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		uuid.NewString(),
		entry.ActorType,
		nullString(entry.ActorID),
		entry.ActorName,
		entry.Action,
		entry.ResourceType,
		nullString(entry.ResourceID),
		entry.RequestID,
		entry.SourceIP,
		entry.UserAgent,
		entry.Result,
		entry.Reason,
		before,
		after,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("record audit entry: %w", err)
	}
	return nil
}

func optionalJSON(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func jsonOrEmpty(value any) (string, error) {
	if value == nil {
		return "{}", nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
