package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hkjang/invenqor/server/internal/apitime"
	"github.com/hkjang/invenqor/server/internal/classify"
)

type classificationRuleView struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Priority    int             `json:"priority"`
	Enabled     bool            `json:"enabled"`
	SystemRule  bool            `json:"system_rule"`
	Confidence  float64         `json:"confidence"`
	Match       classify.Match  `json:"match"`
	Assign      classify.Assign `json:"assign"`
	Assets      int64           `json:"assets"`
}

func (s *Server) listClassificationRules(
	response http.ResponseWriter,
	request *http.Request,
) {
	rules, err := s.ingestService.Classifier().Rules(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	// Counting the assets each rule currently explains is what turns a rule list
	// into something an administrator can reason about.
	usage, err := s.classificationUsage(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	views := make([]classificationRuleView, 0, len(rules))
	for _, rule := range rules {
		views = append(views, classificationRuleView{
			ID:          rule.ID,
			Name:        rule.Name,
			Description: rule.Description,
			Priority:    rule.Priority,
			Enabled:     rule.Enabled,
			SystemRule:  rule.SystemRule,
			Confidence:  rule.Confidence,
			Match:       rule.Match,
			Assign:      rule.Assign,
			Assets:      usage[rule.ID],
		})
	}
	summary, err := s.classificationSummary(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"rules":   views,
		"summary": summary,
	})
}

func (s *Server) classificationUsage(
	ctx context.Context,
) (map[string]int64, error) {
	rows, err := s.database.DB().QueryContext(
		ctx,
		`SELECT classification_rules_json FROM assets
		  WHERE deleted_at IS NULL AND classification_source = 'rule'`,
	)
	if err != nil {
		return nil, fmt.Errorf("count rule usage: %w", err)
	}
	defer rows.Close()
	usage := map[string]int64{}
	for rows.Next() {
		var encoded any
		if err := rows.Scan(&encoded); err != nil {
			return nil, err
		}
		applied := []string{}
		text := ""
		switch typed := encoded.(type) {
		case string:
			text = typed
		case []byte:
			text = string(typed)
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		if json.Unmarshal([]byte(text), &applied) != nil {
			continue
		}
		for _, rule := range applied {
			usage[rule]++
		}
	}
	return usage, rows.Err()
}

func (s *Server) classificationSummary(
	ctx context.Context,
) (map[string]any, error) {
	summary := map[string]any{}
	for key, query := range map[string]string{
		"assets": `SELECT COUNT(*) FROM assets WHERE deleted_at IS NULL`,
		"classified": `SELECT COUNT(*) FROM assets
		  WHERE deleted_at IS NULL AND classification_source = 'rule'`,
		"manual": `SELECT COUNT(*) FROM assets
		  WHERE deleted_at IS NULL AND manual_fields_json <> '[]'`,
		"unclassified": `SELECT COUNT(*) FROM assets
		  WHERE deleted_at IS NULL AND classification_source = ''`,
		"inferred_relations": `SELECT COUNT(*) FROM asset_relations
		  WHERE source = 'inferred' AND status = 'active'`,
		"proposed_relations": `SELECT COUNT(*) FROM asset_relations
		  WHERE status = 'proposed'`,
	} {
		count, err := scalarCount(ctx, s.database.DB(), query)
		if err != nil {
			return nil, fmt.Errorf("summarise classification: %w", err)
		}
		summary[key] = count
	}
	return summary, nil
}

func (s *Server) updateClassificationRule(
	response http.ResponseWriter,
	request *http.Request,
) {
	ruleID := chi.URLParam(request, "ruleID")
	var input struct {
		Enabled  *bool  `json:"enabled"`
		Priority *int   `json:"priority"`
		Reason   string `json:"reason"`
	}
	if ruleID == "" || decodeJSON(request, &input) != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_REQUEST", "The request body is invalid.",
		)
		return
	}
	if input.Enabled == nil && input.Priority == nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"NO_CHANGE", "Provide enabled or priority.",
		)
		return
	}
	if input.Priority != nil && (*input.Priority < 1 || *input.Priority > 999) {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_PRIORITY", "Priority must be between 1 and 999.",
		)
		return
	}
	var before classificationRuleView
	if err := s.database.DB().QueryRowContext(
		request.Context(),
		`SELECT id, name, priority, enabled FROM asset_classification_rules
		  WHERE id = $1`,
		ruleID,
	).Scan(&before.ID, &before.Name, &before.Priority, &before.Enabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeAPIError(
				response, request, http.StatusNotFound,
				"RULE_NOT_FOUND", "The classification rule does not exist.",
			)
			return
		}
		s.internalError(response, request, err)
		return
	}
	enabled := before.Enabled
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	priority := before.Priority
	if input.Priority != nil {
		priority = *input.Priority
	}
	if _, err := s.database.DB().ExecContext(
		request.Context(),
		`UPDATE asset_classification_rules
		    SET enabled = $1, priority = $2, updated_at = $3, updated_by = $4
		  WHERE id = $5`,
		enabled,
		priority,
		time.Now().UTC(),
		principalFromContext(request.Context()).User.ID,
		ruleID,
	); err != nil {
		s.internalError(response, request, err)
		return
	}
	// The engine caches the rule set, so an edit has to invalidate it or the
	// change would appear to do nothing for the next half minute.
	s.ingestService.Classifier().Invalidate()
	s.recordAdminAudit(
		request,
		"asset.classification_rule.update",
		"asset_classification_rule",
		ruleID,
		map[string]any{"enabled": before.Enabled, "priority": before.Priority},
		map[string]any{"enabled": enabled, "priority": priority, "name": before.Name},
		input.Reason,
	)
	writeJSON(response, http.StatusOK, map[string]any{
		"updated":  true,
		"enabled":  enabled,
		"priority": priority,
	})
}

// reclassifyAssets replays the current rule set over the inventory already
// stored. A rule change that only affects assets collected in the future is not
// useful: the point of editing a taxonomy is to fix what is already wrong.
func (s *Server) reclassifyAssets(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		Reason string `json:"reason"`
	}
	if decodeJSON(request, &input) != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_REQUEST", "The request body is invalid.",
		)
		return
	}
	classifier := s.ingestService.Classifier()
	classifier.Invalidate()
	rules, err := classifier.Rules(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	// Read the work list before opening a transaction: SQLite serialises writers
	// and a query on another connection would deadlock against it.
	type target struct {
		assetID     string
		category    string
		name        string
		assetType   string
		environment string
		criticality string
		payload     string
	}
	rows, err := s.database.DB().QueryContext(
		request.Context(),
		`SELECT a.id,
		        COALESCE((SELECT s.category FROM asset_sources s
		                   WHERE s.asset_id = a.id AND s.deleted_at IS NULL
		                   ORDER BY CASE WHEN s.category = 'system' THEN 0 ELSE 1 END,
		                            s.first_seen_at LIMIT 1), ''),
		        a.name, a.type, a.environment, a.criticality,
		        a.attributes_json
		   FROM assets a
		  WHERE a.deleted_at IS NULL
		  ORDER BY CASE WHEN a.type = 'host' THEN 0 ELSE 1 END, a.name
		  LIMIT 20000`,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	targets := make([]target, 0, 256)
	for rows.Next() {
		var item target
		var payload any
		if err := rows.Scan(
			&item.assetID, &item.category, &item.name, &item.assetType,
			&item.environment, &item.criticality, &payload,
		); err != nil {
			rows.Close()
			s.internalError(response, request, err)
			return
		}
		switch typed := payload.(type) {
		case string:
			item.payload = typed
		case []byte:
			item.payload = string(typed)
		}
		targets = append(targets, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		s.internalError(response, request, err)
		return
	}

	transaction, err := s.database.DB().BeginTx(request.Context(), nil)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer transaction.Rollback()
	changed := 0
	for _, item := range targets {
		result, err := classifier.ClassifyAndStore(
			request.Context(),
			transaction,
			rules,
			classify.AssetContext{
				AssetID:     item.assetID,
				Category:    item.category,
				Name:        item.name,
				Type:        item.assetType,
				Environment: item.environment,
				Criticality: item.criticality,
				Payload:     item.payload,
			},
		)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		if len(result.AppliedRules) > 0 {
			changed++
		}
	}
	if err := transaction.Commit(); err != nil {
		s.internalError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request,
		"asset.classification.reclassify",
		"asset_classification_rule",
		"all",
		nil,
		map[string]any{"examined": len(targets), "classified": changed},
		input.Reason,
	)
	writeJSON(response, http.StatusOK, map[string]any{
		"examined":   len(targets),
		"classified": changed,
	})
}

type proposedRelation struct {
	ID           string       `json:"id"`
	RelationType string       `json:"relation_type"`
	Derivation   string       `json:"derivation"`
	Confidence   float64      `json:"confidence"`
	CreatedAt    apitime.Time `json:"created_at"`
	Source       struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Environment string `json:"environment"`
	} `json:"source"`
	Target struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Environment string `json:"environment"`
	} `json:"target"`
}

// listProposedRelations is the review queue. An inference the product is not
// sure about is never applied silently; a person accepts or rejects it.
func (s *Server) listProposedRelations(
	response http.ResponseWriter,
	request *http.Request,
) {
	rows, err := s.database.DB().QueryContext(
		request.Context(),
		`SELECT r.id, r.relation_type, r.derivation, r.confidence, r.created_at,
		        s.id, s.name, s.type, s.environment,
		        t.id, t.name, t.type, t.environment
		   FROM asset_relations r
		   JOIN assets s ON s.id = r.source_asset_id
		   JOIN assets t ON t.id = r.target_asset_id
		  WHERE r.status = 'proposed'
		  ORDER BY r.confidence DESC, r.created_at DESC
		  LIMIT 200`,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer rows.Close()
	items := make([]proposedRelation, 0)
	for rows.Next() {
		var item proposedRelation
		if err := rows.Scan(
			&item.ID, &item.RelationType, &item.Derivation, &item.Confidence,
			&item.CreatedAt,
			&item.Source.ID, &item.Source.Name, &item.Source.Type,
			&item.Source.Environment,
			&item.Target.ID, &item.Target.Name, &item.Target.Type,
			&item.Target.Environment,
		); err != nil {
			s.internalError(response, request, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) reviewProposedRelation(
	response http.ResponseWriter,
	request *http.Request,
) {
	rawRelationID := chi.URLParam(request, "relationID")
	decision := chi.URLParam(request, "decision")
	if rawRelationID == "" || (decision != "approve" && decision != "reject") {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_REQUEST", "The decision must be approve or reject.",
		)
		return
	}
	// openapi.yaml declares {relationId} as `format: uuid` and promises 200, 400
	// and 404 here. asset_relations.id is a UUID column on PostgreSQL and TEXT on
	// the SQLite fallback, so any other spelling ended the same request
	// differently per dialect: PostgreSQL's uuid input rejected `not-a-uuid` and
	// the `urn:uuid:` form with SQLSTATE 22P02, surfacing as a 500 this path never
	// declares, and it folded the brace-wrapped and unhyphenated forms into a hit
	// — so a review addressed that way really did approve or reject the proposal
	// and record an audit entry — where the fallback matched nothing and answered
	// 404. An upper-case id, which is the declared form, diverged the other way:
	// PostgreSQL folded it into a hit and SQLite's comparison did not.
	//
	// Refusing before the UPDATE settles it with the 404 an unknown proposal
	// already gets, and the normalised id is what reaches the SQL and the audit
	// trail, so no non-canonical spelling is stored or re-exported.
	relationID, ok := canonicalUUID(rawRelationID)
	if !ok {
		writeAPIError(
			response, request, http.StatusNotFound,
			"PROPOSAL_NOT_FOUND", "The proposal does not exist or was already reviewed.",
		)
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	if decodeJSON(request, &input) != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_REQUEST", "The request body is invalid.",
		)
		return
	}
	reviewedAt := time.Now().UTC()
	status := "active"
	source := "manual"
	// Rejecting closes the relationship; approving leaves it open.
	var validTo any
	if decision == "reject" {
		status = "rejected"
		source = "inferred"
		validTo = reviewedAt
	}
	// An approved proposal becomes a manual relationship: a person vouched for it,
	// so a later automatic pass must not revise it.
	//
	// valid_to is decided here rather than by a CASE over the same parameter that
	// reviewed_at uses. PostgreSQL deduces a parameter's type from every place it
	// appears, and inside a CASE whose other branch is a bare NULL it deduced a
	// different type than the timestamp column gave it, so every approval and
	// every rejection failed with "inconsistent types deduced for parameter $3"
	// and the console got a 500. SQLite has no such deduction, so the tests passed.
	result, err := s.database.DB().ExecContext(
		request.Context(),
		`UPDATE asset_relations
		    SET status = $1, source = $2, reviewed_at = $3, reviewed_by = $4,
		        valid_to = $5
		  WHERE id = $6 AND status = 'proposed'`,
		status,
		source,
		reviewedAt,
		principalFromContext(request.Context()).User.ID,
		validTo,
		relationID,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		writeAPIError(
			response, request, http.StatusNotFound,
			"PROPOSAL_NOT_FOUND", "The proposal does not exist or was already reviewed.",
		)
		return
	}
	s.recordAdminAudit(
		request,
		"asset.relation."+decision,
		"asset_relation",
		relationID,
		map[string]any{"status": "proposed"},
		map[string]any{"status": status},
		input.Reason,
	)
	writeJSON(response, http.StatusOK, map[string]any{"status": status})
}
