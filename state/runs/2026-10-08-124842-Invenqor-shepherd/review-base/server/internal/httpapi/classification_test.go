package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestClassificationRulesAreListedWithUsageAndSummary(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	response := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/settings/classification", nil, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Rules []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Priority   int    `json:"priority"`
			Enabled    bool   `json:"enabled"`
			SystemRule bool   `json:"system_rule"`
			Match      struct {
				Categories []string `json:"categories"`
				NameTokens []string `json:"name_tokens"`
			} `json:"match"`
			Assign struct {
				Type         string `json:"type"`
				RelateToHost bool   `json:"relate_to_host"`
			} `json:"assign"`
		} `json:"rules"`
		Summary map[string]int64 `json:"summary"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Rules) < 15 {
		t.Fatalf("rule count = %d, want the seeded taxonomy", len(payload.Rules))
	}
	// The list must arrive in run order, or an administrator cannot reason about
	// which rule wins.
	for index := 1; index < len(payload.Rules); index++ {
		if payload.Rules[index-1].Priority > payload.Rules[index].Priority {
			t.Fatalf("rules are not in priority order at %d", index)
		}
	}
	if _, found := payload.Summary["proposed_relations"]; !found {
		t.Fatalf("summary = %+v", payload.Summary)
	}
	tokenRuleFound := false
	for _, rule := range payload.Rules {
		if len(rule.Match.NameTokens) > 0 {
			tokenRuleFound = true
		}
	}
	if !tokenRuleFound {
		t.Fatal("no environment rule uses token matching")
	}
}

func TestDisablingARuleTakesEffectImmediatelyAndIsAudited(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	const productionRule = "20000000-0000-0000-0000-000000000020"
	update := performAuthenticatedJSON(
		t, server, http.MethodPatch,
		"/api/v1/admin/settings/classification/rules/"+productionRule,
		map[string]any{"enabled": false, "reason": "테스트"},
		cookie, csrf,
	)
	if update.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", update.Code, update.Body.String())
	}
	listed := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/settings/classification", nil, cookie, csrf,
	)
	if !strings.Contains(listed.Body.String(), `"enabled":false`) {
		t.Fatalf("the disabled rule is still listed as enabled: %s", listed.Body.String())
	}
	audit := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/admin/audit?limit=20", nil, cookie, csrf,
	)
	if !strings.Contains(audit.Body.String(), "asset.classification_rule.update") {
		t.Fatalf("the rule change was not audited: %s", audit.Body.String())
	}

	missing := performAuthenticatedJSON(
		t, server, http.MethodPatch,
		"/api/v1/admin/settings/classification/rules/"+uuid.NewString(),
		map[string]any{"enabled": false}, cookie, csrf,
	)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unknown rule status = %d", missing.Code)
	}
	invalid := performAuthenticatedJSON(
		t, server, http.MethodPatch,
		"/api/v1/admin/settings/classification/rules/"+productionRule,
		map[string]any{"priority": 0}, cookie, csrf,
	)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid priority status = %d", invalid.Code)
	}
}

// Editing a taxonomy is only useful if it can be replayed over the inventory
// that is already stored.
func TestReclassifyAppliesRulesToExistingAssets(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	// An asset that predates the rule set: default environment and criticality.
	assetID := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO assets(
			id, asset_key, name, type, status, criticality, environment,
			confidence, attributes_json, custom_fields_json, source,
			first_seen_at, last_seen_at, created_at, updated_at
		 ) VALUES($1,'legacy-key','db-prd-77','host','active','normal','other',
		          1.0,'{}','{}','agent',
		          CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,
		          CURRENT_TIMESTAMP)`,
		assetID,
	); err != nil {
		t.Fatal(err)
	}
	// The classification engine reads the collector category from asset_sources.
	if _, err := runtime.DB().Exec(
		`INSERT INTO asset_sources(
			id, asset_id, category, source_asset_id, source_name, payload_json,
			collected_at, first_seen_at, last_seen_at
		 ) VALUES($1,$2,'system','legacy','test','{}',
		          CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		uuid.NewString(), assetID,
	); err != nil {
		t.Fatal(err)
	}

	run := performAuthenticatedJSON(
		t, server, http.MethodPost,
		"/api/v1/admin/settings/classification/reclassify",
		map[string]any{"reason": "규칙 적용"}, cookie, csrf,
	)
	if run.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", run.Code, run.Body.String())
	}
	var summary struct {
		Examined   int `json:"examined"`
		Classified int `json:"classified"`
	}
	if err := json.Unmarshal(run.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Examined < 1 || summary.Classified < 1 {
		t.Fatalf("summary = %+v", summary)
	}
	var environment, criticality, source string
	if err := runtime.DB().QueryRow(
		`SELECT environment, criticality, classification_source
		   FROM assets WHERE id = $1`,
		assetID,
	).Scan(&environment, &criticality, &source); err != nil {
		t.Fatal(err)
	}
	if environment != "production" || criticality != "high" || source != "rule" {
		t.Fatalf(
			"reclassified asset = %q/%q/%q",
			environment, criticality, source,
		)
	}
}

// An operator edit has to win over the rule set, and the reclassify pass is the
// hardest case: it runs over everything with the newest rules.
func TestReclassifyRespectsOperatorEdits(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	created := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/assets",
		map[string]any{
			"name": "svc-prd-01", "type": "host", "environment": "other",
		},
		cookie, csrf,
	)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", created.Code, created.Body.String())
	}
	var payload struct {
		Asset struct {
			ID string `json:"id"`
		} `json:"asset"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	// The operator sets the environment by hand.
	updated := performAuthenticatedJSON(
		t, server, http.MethodPatch, "/api/v1/assets/"+payload.Asset.ID,
		map[string]any{"environment": "dr", "reason": "운영자 지정"},
		cookie, csrf,
	)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d body = %s", updated.Code, updated.Body.String())
	}

	run := performAuthenticatedJSON(
		t, server, http.MethodPost,
		"/api/v1/admin/settings/classification/reclassify",
		map[string]any{}, cookie, csrf,
	)
	if run.Code != http.StatusOK {
		t.Fatalf("reclassify status = %d body = %s", run.Code, run.Body.String())
	}
	var environment string
	if err := runtime.DB().QueryRow(
		"SELECT environment FROM assets WHERE id = $1", payload.Asset.ID,
	).Scan(&environment); err != nil {
		t.Fatal(err)
	}
	if environment != "dr" {
		t.Fatalf(
			"reclassification overwrote the operator's environment with %q",
			environment,
		)
	}
}

func TestProposedRelationsAreReviewable(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	first, second := uuid.NewString(), uuid.NewString()
	for index, id := range []string{first, second} {
		if _, err := runtime.DB().Exec(
			`INSERT INTO assets(
				id, asset_key, name, type, status, criticality, environment,
				confidence, attributes_json, custom_fields_json, source,
				first_seen_at, last_seen_at, created_at, updated_at
			 ) VALUES($1,$2,$3,'host','active','normal','production',
			          1.0,'{}','{}','agent',
			          CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,
			          CURRENT_TIMESTAMP)`,
			id,
			"clone-key-"+string(rune('a'+index)),
			"clone-0"+string(rune('1'+index)),
		); err != nil {
			t.Fatal(err)
		}
	}
	relationID := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO asset_relations(
			id, source_asset_id, relation_type, target_asset_id, source,
			confidence, derivation, status
		 ) VALUES($1,$2,'duplicate_of',$3,'inferred',0.6,'machine_identity',
		          'proposed')`,
		relationID, first, second,
	); err != nil {
		t.Fatal(err)
	}

	listed := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/assets/relations/proposed",
		nil, cookie, csrf,
	)
	if listed.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", listed.Code, listed.Body.String())
	}
	var queue struct {
		Items []struct {
			ID           string  `json:"id"`
			RelationType string  `json:"relation_type"`
			Derivation   string  `json:"derivation"`
			Confidence   float64 `json:"confidence"`
			Source       struct {
				Name string `json:"name"`
			} `json:"source"`
			Target struct {
				Name string `json:"name"`
			} `json:"target"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &queue); err != nil {
		t.Fatal(err)
	}
	if len(queue.Items) != 1 {
		t.Fatalf("queue = %+v", queue.Items)
	}
	// The queue has to name both ends: "duplicate_of" between two identifiers is
	// not a decision anyone can make.
	if queue.Items[0].Source.Name == "" || queue.Items[0].Target.Name == "" {
		t.Fatalf("queue entry is missing asset names: %+v", queue.Items[0])
	}

	approve := performAuthenticatedJSON(
		t, server, http.MethodPost,
		"/api/v1/assets/relations/"+relationID+"/approve",
		map[string]any{"reason": "확인"}, cookie, csrf,
	)
	if approve.Code != http.StatusOK {
		t.Fatalf("approve status = %d body = %s", approve.Code, approve.Body.String())
	}
	var status, source string
	var reviewedBy any
	if err := runtime.DB().QueryRow(
		"SELECT status, source, reviewed_by FROM asset_relations WHERE id = $1",
		relationID,
	).Scan(&status, &source, &reviewedBy); err != nil {
		t.Fatal(err)
	}
	// An approved proposal becomes a human decision, so later automatic passes
	// must leave it alone.
	if status != "active" || source != "manual" || reviewedBy == nil {
		t.Fatalf("approved relation = %q/%q/%v", status, source, reviewedBy)
	}
	// A second review of the same proposal must not silently succeed.
	repeat := performAuthenticatedJSON(
		t, server, http.MethodPost,
		"/api/v1/assets/relations/"+relationID+"/reject",
		map[string]any{}, cookie, csrf,
	)
	if repeat.Code != http.StatusNotFound {
		t.Fatalf("repeat review status = %d, want 404", repeat.Code)
	}
}
