package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Before the memory graph, `relation` was free text taken straight from a
// request body. An ordinary user could POST relation='dreamed' and produce an
// edge asserting that umm's Dream layer had discovered the connection, and a
// 5000-character relation was accepted and stored for every reader to render.
//
// These tests hold both closed at the store boundary, which is where every
// caller — HTTP, MCP, import — passes through.

func graphSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID, [2]uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Close() })

	ownerID, spaceID := uuid.New(), uuid.New()
	name := "graph_owner_" + strings.ReplaceAll(ownerID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, ownerID, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, ownerID) })
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1)`, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'memory graph')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	var notes [2]uuid.UUID
	for index := range notes {
		notes[index] = uuid.New()
		if _, err = db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,$4)`,
			notes[index], spaceID, ownerID, "생각 "+string(rune('가'+index))); err != nil {
			t.Fatal(err)
		}
	}
	return db, ownerID, spaceID, notes
}

// The exact request that used to succeed.
func TestClientCannotForgeDreamProvenanceIntegration(t *testing.T) {
	db, ownerID, spaceID, notes := graphSpace(t)
	ctx := context.Background()

	for _, claim := range []string{"dreamed", "expanded", "auto"} {
		_, err := db.CreateEdge(ctx, ownerID, Edge{
			SpaceID: spaceID, SourceID: notes[0], TargetID: notes[1], Relation: Relation(claim),
		})
		if !errors.Is(err, ErrUnknownRelation) {
			t.Errorf("relation %q was accepted; a client can claim provenance again (err=%v)", claim, err)
		}
	}

	var stored int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM note_edges WHERE space_id=$1`, spaceID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 0 {
		t.Fatalf("%d edges were written despite every request being rejected", stored)
	}
}

// Even a valid relation must not let the caller choose where the edge came from.
func TestOriginIsSetByTheCodePathNotTheCallerIntegration(t *testing.T) {
	db, ownerID, spaceID, notes := graphSpace(t)
	ctx := context.Background()

	// A caller that fills in Origin and Confidence anyway — the shape a handler
	// would produce if someone later bound them straight from JSON.
	forged := 0.99
	edge, err := db.CreateEdge(ctx, ownerID, Edge{
		SpaceID: spaceID, SourceID: notes[0], TargetID: notes[1],
		Relation: RelationSupports, Origin: OriginDream, Confidence: &forged,
	})
	if err != nil {
		t.Fatalf("create edge: %v", err)
	}
	if edge.Origin != OriginManual {
		t.Errorf("origin=%q; the web path must record a drawn line as manual", edge.Origin)
	}
	if edge.Confidence != nil {
		t.Errorf("confidence=%v; a person drawing a line is not stating a probability", *edge.Confidence)
	}

	var origin string
	var confidence *float64
	if err = db.Pool.QueryRow(ctx, `SELECT origin,confidence FROM note_edges WHERE id=$1`, edge.ID).Scan(&origin, &confidence); err != nil {
		t.Fatal(err)
	}
	if origin != string(OriginManual) || confidence != nil {
		t.Errorf("stored origin=%q confidence=%v; the database kept the forged values", origin, confidence)
	}
}

// An agent writing through MCP is making an assertion, like a person, but it is
// not a person. Once agents write into a memory the owner has to be able to see
// which parts of it they wrote.
func TestAgentEdgesAreDistinguishableFromHandDrawnIntegration(t *testing.T) {
	db, ownerID, spaceID, notes := graphSpace(t)
	ctx := context.Background()

	agentEdge, err := db.CreateAgentEdge(ctx, ownerID, Edge{
		SpaceID: spaceID, SourceID: notes[0], TargetID: notes[1], Relation: RelationSupports,
	})
	if err != nil {
		t.Fatalf("create agent edge: %v", err)
	}
	if agentEdge.Origin != OriginAgent {
		t.Fatalf("origin=%q, want %q", agentEdge.Origin, OriginAgent)
	}
	if agentEdge.Origin.Inferred() {
		t.Error("an agent asserted this connection; it is not umm's own guess")
	}
}

// One edge per pair was right when relations had no meaning. Two notes can
// genuinely both support and refine one another, and the old constraint made the
// second write fail.
func TestTypedEdgesCoexistOnOnePairIntegration(t *testing.T) {
	db, ownerID, spaceID, notes := graphSpace(t)
	ctx := context.Background()

	for _, relation := range []Relation{RelationSupports, RelationRefines, RelationFollows} {
		if _, err := db.CreateEdge(ctx, ownerID, Edge{
			SpaceID: spaceID, SourceID: notes[0], TargetID: notes[1], Relation: relation,
		}); err != nil {
			t.Fatalf("second relation %q on the same pair: %v", relation, err)
		}
	}
	var count int
	if err := db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM note_edges WHERE source_note_id=$1 AND target_note_id=$2`,
		notes[0], notes[1]).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("expected three typed edges on the pair, got %d", count)
	}
	// The same meaning twice is still one connection.
	if _, err := db.CreateEdge(ctx, ownerID, Edge{
		SpaceID: spaceID, SourceID: notes[0], TargetID: notes[1], Relation: RelationSupports,
	}); err == nil {
		t.Error("the same relation was recorded twice on one pair")
	}
}

// The database is the last line: even a write that bypasses ParseRelation must
// not be able to store a meaning or a provenance outside the vocabulary.
func TestDatabaseRejectsValuesOutsideTheVocabularyIntegration(t *testing.T) {
	db, ownerID, spaceID, notes := graphSpace(t)
	ctx := context.Background()

	for _, bad := range []struct{ relation, origin string }{
		{"dreamed", "manual"},
		{strings.Repeat("A", 5000), "manual"},
		{"related", "impersonated"},
	} {
		_, err := db.Pool.Exec(ctx,
			`INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,created_by) VALUES($1,$2,$3,$4,$5,$6)`,
			spaceID, notes[0], notes[1], bad.relation, bad.origin, ownerID)
		if err == nil {
			t.Errorf("the database accepted relation=%.20q origin=%q", bad.relation, bad.origin)
		}
	}
}

// A guess with no score cannot be ranked or filtered, and a score on a drawn
// line is a number nobody produced.
func TestConfidenceIsTiedToInferenceIntegration(t *testing.T) {
	db, ownerID, spaceID, notes := graphSpace(t)
	ctx := context.Background()

	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,confidence,created_by) VALUES($1,$2,$3,'related','auto',NULL,$4)`,
		spaceID, notes[0], notes[1], ownerID); err == nil {
		t.Error("an inferred edge was stored without a confidence")
	}
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,confidence,created_by) VALUES($1,$2,$3,'related','auto',1.4,$4)`,
		spaceID, notes[0], notes[1], ownerID); err == nil {
		t.Error("a confidence above 1 was stored")
	}
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,confidence,created_by) VALUES($1,$2,$3,'related','auto',0.62,$4)`,
		spaceID, notes[0], notes[1], ownerID); err != nil {
		t.Errorf("a well-formed inferred edge was rejected: %v", err)
	}
}

// Every place an edge is read has to carry its provenance, or the interface
// shows a connection without saying where it came from — which is the whole
// point of recording it.
//
// This is the defect that reached a release: Backlinks was missed when the Edge
// struct gained origin and confidence, and nothing failed. The fields simply
// arrived empty, so the canvas rendered a connection with a separator and
// nothing after it, while the release notes said umm now showed the origin.
//
// The table below is the guard. Every function that hands an Edge to a caller
// belongs in it; a read path added without an entry is one nobody is checking.
func TestEveryEdgeReadCarriesProvenanceIntegration(t *testing.T) {
	db, ownerID, spaceID, notes := graphSpace(t)
	ctx := context.Background()

	drawn, err := db.CreateEdge(ctx, ownerID, Edge{
		SpaceID: spaceID, SourceID: notes[0], TargetID: notes[1], Relation: RelationSupports,
	})
	if err != nil {
		t.Fatalf("create edge: %v", err)
	}

	readPaths := []struct {
		name string
		read func() (Edge, error)
	}{
		{"CreateEdge", func() (Edge, error) { return drawn, nil }},
		{"Backlinks", func() (Edge, error) {
			links, err := db.Backlinks(ctx, ownerID, notes[1])
			if err != nil {
				return Edge{}, err
			}
			for _, link := range links {
				if link.Edge.ID == drawn.ID {
					return link.Edge, nil
				}
			}
			return Edge{}, errors.New("the connection did not appear as a backlink")
		}},
		{"ListNotes", func() (Edge, error) {
			_, edges, err := db.ListNotes(ctx, ownerID, spaceID, "")
			if err != nil {
				return Edge{}, err
			}
			for _, edge := range edges {
				if edge.ID == drawn.ID {
					return edge, nil
				}
			}
			return Edge{}, errors.New("the connection was not among the space edges")
		}},
	}

	for _, path := range readPaths {
		t.Run(path.name, func(t *testing.T) {
			edge, err := path.read()
			if err != nil {
				t.Fatalf("%s: %v", path.name, err)
			}
			if edge.Origin == "" {
				t.Fatalf("%s returned an edge with no origin; the interface cannot say who made it", path.name)
			}
			if edge.Origin != OriginManual {
				t.Errorf("origin=%q, want %q", edge.Origin, OriginManual)
			}
			if edge.Relation != RelationSupports {
				t.Errorf("relation=%q, want %q", edge.Relation, RelationSupports)
			}
			if edge.SpaceID != spaceID {
				t.Errorf("spaceId=%v, want %v", edge.SpaceID, spaceID)
			}
		})
	}
}

// The inferred half of the same guard: a suggestion read back anywhere has to
// keep both the mark that umm guessed it and the score, or the interface cannot
// present it as a suggestion.
func TestInferredEdgesKeepTheirScoreWhenReadBackIntegration(t *testing.T) {
	db, ownerID, spaceID, notes := graphSpace(t)
	ctx := context.Background()

	confidence := 0.73
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO note_edges(space_id,source_note_id,target_note_id,relation,origin,confidence,created_by)
		 VALUES($1,$2,$3,'related','auto',$4,$5)`,
		spaceID, notes[0], notes[1], confidence, ownerID); err != nil {
		t.Fatal(err)
	}

	links, err := db.Backlinks(ctx, ownerID, notes[1])
	if err != nil {
		t.Fatalf("backlinks: %v", err)
	}
	if len(links) == 0 {
		t.Fatal("the inferred connection did not appear as a backlink")
	}
	edge := links[0].Edge
	if !edge.Origin.Inferred() {
		t.Errorf("origin=%q; a suggestion read back must still report itself as inferred", edge.Origin)
	}
	if edge.Confidence == nil {
		t.Fatal("the score was dropped on the way out; the interface cannot show how strong the guess was")
	}
	if *edge.Confidence < confidence-1e-6 || *edge.Confidence > confidence+1e-6 {
		t.Errorf("confidence=%.3f, want %.3f", *edge.Confidence, confidence)
	}
}
