package ingest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

type AssetRecord struct {
	AssetID     string          `json:"asset_id"`
	Category    string          `json:"category"`
	Source      string          `json:"source"`
	CollectedAt uint64          `json:"collected_at"`
	Payload     json.RawMessage `json:"payload"`
}

type CollectionError struct {
	Collector string `json:"collector"`
	Message   string `json:"message"`
}

type Snapshot struct {
	SchemaVersion uint32            `json:"schema_version"`
	AgentID       string            `json:"agent_id"`
	CollectedAt   uint64            `json:"collected_at"`
	DurationMS    uint64            `json:"duration_ms"`
	Records       []AssetRecord     `json:"records"`
	Errors        []CollectionError `json:"errors"`
}

type AssetChange struct {
	Kind     string       `json:"kind"`
	AssetID  string       `json:"asset_id"`
	Category string       `json:"category"`
	Record   *AssetRecord `json:"record,omitempty"`
}

type Envelope struct {
	SchemaVersion    uint32            `json:"schema_version"`
	EventID          string            `json:"event_id"`
	AgentID          string            `json:"agent_id"`
	CreatedAt        uint64            `json:"created_at"`
	Kind             string            `json:"kind"`
	SnapshotHash     string            `json:"snapshot_hash"`
	Snapshot         *Snapshot         `json:"snapshot,omitempty"`
	Changes          []AssetChange     `json:"changes,omitempty"`
	CollectionErrors []CollectionError `json:"collection_errors,omitempty"`
}

func DecodeEnvelope(raw []byte) (Envelope, error) {
	var envelope Envelope
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return Envelope{}, fmt.Errorf("decode event envelope: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Envelope{}, errors.New("event body contains multiple JSON values")
	}
	if err := envelope.Validate(); err != nil {
		return Envelope{}, err
	}
	return envelope, nil
}

func (e Envelope) Validate() error {
	if e.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version %d", e.SchemaVersion)
	}
	if _, err := uuid.Parse(e.EventID); err != nil {
		return errors.New("event_id must be a UUID")
	}
	if _, err := uuid.Parse(e.AgentID); err != nil {
		return errors.New("agent_id must be a UUID")
	}
	if e.CreatedAt == 0 {
		return errors.New("created_at is required")
	}
	if e.Kind != "inventory" && e.Kind != "heartbeat" {
		return errors.New("kind must be inventory or heartbeat")
	}
	if e.Kind == "heartbeat" && (e.Snapshot != nil || len(e.Changes) > 0) {
		return errors.New("heartbeat cannot contain inventory records")
	}
	if e.Snapshot != nil {
		if e.Snapshot.SchemaVersion != 1 {
			return errors.New("snapshot schema_version must be 1")
		}
		if e.Snapshot.AgentID != e.AgentID {
			return errors.New("snapshot agent_id does not match envelope")
		}
		for index := range e.Snapshot.Records {
			if err := e.Snapshot.Records[index].Validate(); err != nil {
				return fmt.Errorf("snapshot record %d: %w", index, err)
			}
		}
	}
	for index, change := range e.Changes {
		switch change.Kind {
		case "added", "updated":
			if change.Record == nil {
				return fmt.Errorf("change %d requires record", index)
			}
			if change.AssetID != change.Record.AssetID ||
				change.Category != change.Record.Category {
				return fmt.Errorf("change %d record identity does not match", index)
			}
			if err := change.Record.Validate(); err != nil {
				return fmt.Errorf("change %d: %w", index, err)
			}
		case "removed":
			if strings.TrimSpace(change.AssetID) == "" ||
				strings.TrimSpace(change.Category) == "" {
				return fmt.Errorf("change %d removal identity is required", index)
			}
		default:
			return fmt.Errorf("change %d has unsupported kind %q", index, change.Kind)
		}
	}
	for _, collectionError := range append(
		append([]CollectionError{}, e.CollectionErrors...),
		snapshotErrors(e.Snapshot)...,
	) {
		if strings.TrimSpace(collectionError.Collector) == "" {
			return errors.New("collector error name is required")
		}
	}
	return nil
}

func (r AssetRecord) Validate() error {
	if strings.TrimSpace(r.AssetID) == "" {
		return errors.New("asset_id is required")
	}
	if strings.TrimSpace(r.Category) == "" {
		return errors.New("category is required")
	}
	if strings.TrimSpace(r.Source) == "" {
		return errors.New("source is required")
	}
	if r.CollectedAt == 0 {
		return errors.New("collected_at is required")
	}
	if len(r.Payload) == 0 || !json.Valid(r.Payload) {
		return errors.New("payload must be valid JSON")
	}
	var object map[string]any
	if err := json.Unmarshal(r.Payload, &object); err != nil || object == nil {
		return errors.New("payload must be a JSON object")
	}
	return nil
}

func snapshotErrors(snapshot *Snapshot) []CollectionError {
	if snapshot == nil {
		return nil
	}
	return snapshot.Errors
}
