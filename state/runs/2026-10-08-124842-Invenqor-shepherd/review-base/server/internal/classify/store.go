package classify

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Store loads the rule set and applies it to assets. The rule set is cached
// because it is read on every ingested record and changes only when an
// administrator edits it; the cache is invalidated by version, never by clock.
type Store struct {
	database *sql.DB
	mutex    sync.RWMutex
	cached   []Rule
	loadedAt time.Time
	now      func() time.Time
}

func NewStore(database *sql.DB) *Store {
	return &Store{database: database, now: func() time.Time { return time.Now().UTC() }}
}

const ruleCacheTTL = 30 * time.Second

// Rules returns the enabled and disabled rule set in run order.
func (store *Store) Rules(ctx context.Context) ([]Rule, error) {
	store.mutex.RLock()
	if store.cached != nil && store.now().Sub(store.loadedAt) < ruleCacheTTL {
		rules := store.cached
		store.mutex.RUnlock()
		return rules, nil
	}
	store.mutex.RUnlock()
	rules, err := store.load(ctx)
	if err != nil {
		return nil, err
	}
	store.mutex.Lock()
	store.cached = rules
	store.loadedAt = store.now()
	store.mutex.Unlock()
	return rules, nil
}

// Invalidate drops the cache so an administrative edit takes effect at once.
func (store *Store) Invalidate() {
	store.mutex.Lock()
	store.cached = nil
	store.mutex.Unlock()
}

func (store *Store) load(ctx context.Context) ([]Rule, error) {
	rows, err := store.database.QueryContext(
		ctx,
		`SELECT id, name, description, priority, enabled, system_rule,
		        confidence, match_json, assign_json
		   FROM asset_classification_rules
		  ORDER BY priority, name`,
	)
	if err != nil {
		return nil, fmt.Errorf("load classification rules: %w", err)
	}
	defer rows.Close()
	rules := make([]Rule, 0, 24)
	for rows.Next() {
		var rule Rule
		var matchRaw, assignRaw any
		if err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Description, &rule.Priority,
			&rule.Enabled, &rule.SystemRule, &rule.Confidence,
			&matchRaw, &assignRaw,
		); err != nil {
			return nil, err
		}
		if rule.Match, err = DecodeMatch(textOf(matchRaw)); err != nil {
			return nil, fmt.Errorf("rule %s has an invalid match: %w", rule.Name, err)
		}
		if rule.Assign, err = DecodeAssign(textOf(assignRaw)); err != nil {
			return nil, fmt.Errorf("rule %s has an invalid assignment: %w", rule.Name, err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	Sort(rules)
	return rules, nil
}

func textOf(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

// AssetContext is what the caller already knows about the asset row.
type AssetContext struct {
	AssetID     string
	Category    string
	Name        string
	Type        string
	Environment string
	Criticality string
	Payload     string
}

// ClassifyAndStore applies the rule set to one asset and writes the outcome with
// its provenance. It returns the result so the caller can act on the
// relationship decision in the same transaction.
func (store *Store) ClassifyAndStore(
	ctx context.Context,
	transaction *sql.Tx,
	rules []Rule,
	asset AssetContext,
) (Result, error) {
	var manualRaw any
	if err := transaction.QueryRowContext(
		ctx,
		"SELECT manual_fields_json FROM assets WHERE id = $1",
		asset.AssetID,
	).Scan(&manualRaw); err != nil {
		return Result{}, fmt.Errorf("read manual field list: %w", err)
	}
	manual := []string{}
	if text := textOf(manualRaw); strings.TrimSpace(text) != "" {
		_ = json.Unmarshal([]byte(text), &manual)
	}
	attributes := map[string]any{}
	if strings.TrimSpace(asset.Payload) != "" {
		_ = json.Unmarshal([]byte(asset.Payload), &attributes)
	}
	result := Apply(rules, Subject{
		Category:     asset.Category,
		Name:         asset.Name,
		Type:         asset.Type,
		Environment:  asset.Environment,
		Criticality:  asset.Criticality,
		Attributes:   attributes,
		ManualFields: manual,
	})
	if len(result.AppliedRules) == 0 {
		return result, nil
	}
	appliedJSON, err := json.Marshal(result.AppliedRules)
	if err != nil {
		return Result{}, err
	}
	tagsJSON, err := json.Marshal(result.Tags)
	if err != nil {
		return Result{}, err
	}
	// COALESCE keeps a value the rules did not decide, and the manual guard has
	// already removed anything a person owns from the result.
	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE assets SET
		   type = CASE WHEN $1 = '' THEN type ELSE $1 END,
		   environment = CASE WHEN $2 = '' THEN environment ELSE $2 END,
		   criticality = CASE WHEN $3 = '' THEN criticality ELSE $3 END,
		   owner_department = CASE WHEN $4 = '' THEN owner_department ELSE $4 END,
		   location = CASE WHEN $5 = '' THEN location ELSE $5 END,
		   tags_json = $6,
		   classification_source = 'rule',
		   classification_confidence = $7,
		   classification_rules_json = $8,
		   classified_at = $9,
		   updated_at = $9
		 WHERE id = $10`,
		result.Type,
		result.Environment,
		result.Criticality,
		result.OwnerDepartment,
		result.Location,
		string(tagsJSON),
		result.Confidence,
		string(appliedJSON),
		store.now(),
		asset.AssetID,
	); err != nil {
		return Result{}, fmt.Errorf("store classification: %w", err)
	}
	return result, nil
}

// MarkManual records that a person has taken ownership of these fields, so no
// later automatic pass moves them back.
func MarkManual(
	ctx context.Context,
	transaction *sql.Tx,
	assetID string,
	fields []string,
) error {
	if len(fields) == 0 {
		return nil
	}
	var existingRaw any
	if err := transaction.QueryRowContext(
		ctx,
		"SELECT manual_fields_json FROM assets WHERE id = $1",
		assetID,
	).Scan(&existingRaw); err != nil {
		return fmt.Errorf("read manual field list: %w", err)
	}
	existing := []string{}
	if text := textOf(existingRaw); strings.TrimSpace(text) != "" {
		_ = json.Unmarshal([]byte(text), &existing)
	}
	for _, field := range fields {
		existing = appendUnique(existing, field)
	}
	encoded, err := json.Marshal(existing)
	if err != nil {
		return err
	}
	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE assets SET manual_fields_json = $1,
		   classification_source = CASE WHEN classification_source = ''
		     THEN 'manual' ELSE classification_source END
		 WHERE id = $2`,
		string(encoded),
		assetID,
	); err != nil {
		return fmt.Errorf("store manual field list: %w", err)
	}
	return nil
}
