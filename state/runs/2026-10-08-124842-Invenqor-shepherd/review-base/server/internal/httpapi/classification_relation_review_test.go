package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// openapi.yaml declares {relationId} on
// POST /api/v1/assets/relations/{relationId}/{decision} as `format: uuid` and
// promises 200, 400 and 404 — no 500. reviewProposedRelation put
// chi.URLParam(request, "relationID") straight into
// `UPDATE asset_relations ... WHERE id = $6`, and asset_relations.id is a UUID
// column on PostgreSQL and TEXT on the SQLite fallback, so the same review
// request ended differently per dialect: a spelling PostgreSQL's uuid input
// rejects failed with SQLSTATE 22P02 and surfaced as a 500, the brace-wrapped
// and unhyphenated spellings were folded into a hit and really did review the
// proposal, and an upper-case id — the declared form — was folded by
// PostgreSQL where SQLite's comparison matched nothing and answered 404.
//
// These tests go through the real router against the real Runtime and read back
// the rows the handler writes, so a 404 that had already reviewed is caught.

// insertProposedRelation creates two assets and a proposed relationship between
// them, and returns the relationship id.
func insertProposedRelation(t *testing.T, server *Server) string {
	t.Helper()
	source, target := relationValidationAssets(t, server)
	relationID := uuid.NewString()
	if _, err := server.database.DB().Exec(
		`INSERT INTO asset_relations(
			id, source_asset_id, relation_type, target_asset_id, source,
			confidence, derivation, status
		 ) VALUES($1,$2,'duplicate_of',$3,'inferred',0.6,'machine_identity',
		          'proposed')`,
		relationID, source, target,
	); err != nil {
		t.Fatalf("insert proposed relation: %v", err)
	}
	return relationID
}

// relationStatus reads the column the review moves.
func relationStatus(t *testing.T, server *Server, id string) string {
	t.Helper()
	var status string
	if err := server.database.DB().QueryRow(
		`SELECT status FROM asset_relations WHERE id=$1`, id,
	).Scan(&status); err != nil {
		t.Fatalf("read relation status %s: %v", id, err)
	}
	return status
}

// A relationID that is not the declared canonical UUID has to be refused before
// the UPDATE runs, with the same 404 an unknown proposal already gets, on both
// dialects — and it must leave the proposal and the audit trail alone.
func TestProposedRelationReviewAnswers404ForNonCanonicalRelationID(t *testing.T) {
	for _, decision := range []string{"approve", "reject"} {
		t.Run(decision, func(t *testing.T) {
			runtime := newRuntime(t)
			server := testServer(t, runtime)
			cookie, csrf := authenticateInitialAdmin(t, server, runtime)
			relationID := insertProposedRelation(t, server)

			for name, spelling := range assetIDSpellings(relationID) {
				t.Run(name, func(t *testing.T) {
					audits := countRows(t, server,
						`SELECT COUNT(*) FROM audit_logs WHERE resource_type='asset_relation'`,
					)
					response := performAuthenticatedJSON(
						t, server, http.MethodPost,
						"/api/v1/assets/relations/"+spelling+"/"+decision,
						map[string]any{"reason": "확인"}, cookie, csrf,
					)
					if response.Code != http.StatusNotFound ||
						errorCode(t, response) != "PROPOSAL_NOT_FOUND" {
						t.Fatalf(
							"review response = %d %q, want 404 PROPOSAL_NOT_FOUND: %s",
							response.Code, errorCode(t, response), response.Body.String(),
						)
					}
					if status := relationStatus(t, server, relationID); status != "proposed" {
						t.Fatalf("relation status = %q, want unchanged \"proposed\"", status)
					}
					if after := countRows(t, server,
						`SELECT COUNT(*) FROM audit_logs WHERE resource_type='asset_relation'`,
					); after != audits {
						t.Fatalf("relation audits = %d, want unchanged %d", after, audits)
					}
				})
			}
		})
	}
}

// The declared form includes an upper-case id, which both dialects have to
// review the same way, and the audit trail has to name it canonically.
func TestProposedRelationReviewNormalisesUpperCaseRelationID(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	relationID := insertProposedRelation(t, server)

	response := performAuthenticatedJSON(
		t, server, http.MethodPost,
		"/api/v1/assets/relations/"+strings.ToUpper(relationID)+"/approve",
		map[string]any{"reason": "확인"}, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf(
			"upper-case approve = %d, want 200: %s",
			response.Code, response.Body.String(),
		)
	}
	if status := relationStatus(t, server, relationID); status != "active" {
		t.Fatalf("approved relation status = %q, want \"active\"", status)
	}
	// The audit trail must not carry the caller's spelling: resource_id is TEXT
	// on both dialects, so an upper-case id would be stored and re-exported
	// verbatim.
	var resourceID string
	if err := server.database.DB().QueryRow(
		`SELECT resource_id FROM audit_logs
		  WHERE resource_type='asset_relation' AND action='asset.relation.approve'`,
	).Scan(&resourceID); err != nil {
		t.Fatalf("read approve audit: %v", err)
	}
	if resourceID != relationID {
		t.Fatalf("audit resource_id = %q, want canonical %q", resourceID, relationID)
	}
}
