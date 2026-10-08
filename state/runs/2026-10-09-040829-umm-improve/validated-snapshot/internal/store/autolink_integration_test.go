package store

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/intelligence"
)

// Auto-link is the first thing in umm that writes into a person's memory without
// being asked to. These tests hold the two properties that makes acceptable: it
// refuses to guess on a backend that cannot tell meaning from vocabulary, and
// everything it writes is marked as a guess and can be undone.

// autoLinkSpace builds a workspace with two clearly distinct subjects, so a
// backend that works has something to find and a backend that does not has
// something to get wrong.
func autoLinkSpace(t *testing.T) (*Store, uuid.UUID, uuid.UUID) {
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
	name := "autolink_" + strings.ReplaceAll(ownerID.String(), "-", "")
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,$2::citext,$2::text)`, ownerID, name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, ownerID) })
	if _, err = db.Pool.Exec(ctx, `INSERT INTO user_preferences(user_id) VALUES($1)`, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'auto link')`, spaceID, ownerID); err != nil {
		t.Fatal(err)
	}
	for _, content := range autoLinkNotes {
		if _, err = db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,$4)`,
			uuid.New(), spaceID, ownerID, content); err != nil {
			t.Fatal(err)
		}
	}
	return db, ownerID, spaceID
}

// Deliberately not sentences from the quality dataset. They used to be, which
// meant the stub matched them on its topic branch and the workspace geometry
// below never applied — the fixture was silently borrowing the shape of a
// different dataset, and any change here did nothing.
var autoLinkNotes = []string{
	"배포 승인은 당번을 정해 돌아가며 맡는다",
	"배포 창구를 하나로 좁히면 대기가 길어진다",
	"긴급 배포는 사후 검토를 반드시 남긴다",
	"화분에 물 주는 요일을 정해 두었다",
	"베란다 채광이 오후에만 들어온다",
	"흙이 마르는 속도가 계절마다 다르다",
}

// The gate that matters most. umm's own measurement says the offline algorithm
// ranks shared vocabulary above shared meaning, so anything it proposed would be
// a connection between notes that happen to use the same words.
func TestAutoLinkRefusesALexicalBackendIntegration(t *testing.T) {
	db, ownerID, spaceID := autoLinkSpace(t)
	ctx := context.Background()

	result, err := db.SuggestLinks(ctx, ownerID, spaceID)
	if err != nil {
		t.Fatalf("suggest links: %v", err)
	}
	if result.Outcome != OutcomeBackendNotSemantic {
		t.Fatalf("outcome=%q, want %q: umm proposed connections from a backend that cannot judge them",
			result.Outcome, OutcomeBackendNotSemantic)
	}
	if len(result.Edges) != 0 {
		t.Fatalf("%d edges were written despite the refusal", len(result.Edges))
	}
	// A refusal still has to serialise as a list. A caller that iterates the
	// result should not crash on the paths where umm declined to run.
	if result.Edges == nil {
		t.Error("edges was null rather than an empty list")
	}
	var written int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM note_edges WHERE space_id=$1`, spaceID).Scan(&written); err != nil {
		t.Fatal(err)
	}
	if written != 0 {
		t.Fatalf("%d edges reached the database", written)
	}
}

// With a backend that does separate meaning, umm proposes — and everything it
// writes says so.
func TestAutoLinkMarksEverythingItWritesAsInferredIntegration(t *testing.T) {
	db, ownerID, spaceID := autoLinkSpace(t)
	ctx := context.Background()
	stopGateway := useSemanticStub(t, db, ownerID)
	defer stopGateway()
	assertNotesEmbeddedByTheStub(t, db, ownerID, spaceID)

	result, err := db.SuggestLinks(ctx, ownerID, spaceID)
	if err != nil {
		t.Fatalf("suggest links: %v", err)
	}
	if result.Outcome != OutcomeSuggested {
		notes, _, _ := db.ListNotes(ctx, ownerID, spaceID, "")
		vectors := db.loadEmbeddings(ctx, notes)
		var algorithms []string
		rows, _ := db.Pool.Query(ctx, `SELECT DISTINCT algorithm FROM note_embeddings e JOIN notes n ON n.id=e.note_id WHERE n.space_id=$1`, spaceID)
		for rows.Next() {
			var a string
			_ = rows.Scan(&a)
			algorithms = append(algorithms, a)
		}
		rows.Close()
		var scores []float64
		for i := 0; i < len(notes); i++ {
			for j := i + 1; j < len(notes); j++ {
				scores = append(scores, intelligence.Cosine(vectors[notes[i].ID], vectors[notes[j].ID]))
			}
		}
		t.Fatalf("outcome=%q considered=%d notes=%d\n  stored algorithms=%v\n  scores=%.3f",
			result.Outcome, result.Considered, len(notes), algorithms, scores)
	}
	if len(result.Edges) == 0 {
		t.Fatal("a workspace with two clear subjects produced no suggestion")
	}
	for _, edge := range result.Edges {
		if edge.Origin != OriginAuto {
			t.Errorf("suggestion recorded as %q; it must be distinguishable from a drawn line", edge.Origin)
		}
		if !edge.Origin.Inferred() {
			t.Error("a suggestion must report itself as inferred")
		}
		if edge.Confidence == nil {
			t.Fatal("a guess with no score cannot be weighed or ranked")
		}
		if *edge.Confidence < 0.5 || *edge.Confidence > 1 {
			t.Errorf("confidence %.3f is outside the range the schema allows", *edge.Confidence)
		}
	}

	// Suggestions must not connect the two subjects to each other: that is the
	// failure a lexical backend produces, and the reason for the gate.
	var crossSubject int
	if err = db.Pool.QueryRow(ctx, `
		SELECT count(*) FROM note_edges e
		JOIN notes a ON a.id=e.source_note_id
		JOIN notes b ON b.id=e.target_note_id
		WHERE e.space_id=$1 AND (a.content LIKE '%자전거%' OR a.content LIKE '%라이딩%') != (b.content LIKE '%자전거%' OR b.content LIKE '%라이딩%')
	`, spaceID).Scan(&crossSubject); err != nil {
		t.Fatal(err)
	}
	if crossSubject > 0 {
		t.Errorf("%d suggestions joined the two unrelated subjects", crossSubject)
	}
}

// umm must not propose a connection someone already drew, and must not propose
// the same one again on a second run.
func TestAutoLinkDoesNotRepeatItselfIntegration(t *testing.T) {
	db, ownerID, spaceID := autoLinkSpace(t)
	ctx := context.Background()
	stopGateway := useSemanticStub(t, db, ownerID)
	defer stopGateway()
	assertNotesEmbeddedByTheStub(t, db, ownerID, spaceID)

	first, err := db.SuggestLinks(ctx, ownerID, spaceID)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if len(first.Edges) == 0 {
		t.Fatal("first run proposed nothing")
	}
	second, err := db.SuggestLinks(ctx, ownerID, spaceID)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(second.Edges) != 0 {
		t.Fatalf("a second run proposed %d connections it had already proposed", len(second.Edges))
	}
	if second.Outcome != OutcomeNoCandidates {
		t.Errorf("outcome=%q, want %q", second.Outcome, OutcomeNoCandidates)
	}
}

// A suggestion has to be answerable both ways, or it is just clutter someone
// cannot clear.
func TestSuggestionCanBeAcceptedOrDismissedIntegration(t *testing.T) {
	db, ownerID, spaceID := autoLinkSpace(t)
	ctx := context.Background()
	stopGateway := useSemanticStub(t, db, ownerID)
	defer stopGateway()
	assertNotesEmbeddedByTheStub(t, db, ownerID, spaceID)

	result, err := db.SuggestLinks(ctx, ownerID, spaceID)
	if err != nil || len(result.Edges) < 2 {
		t.Fatalf("need at least two suggestions to test both answers: %d (%v)", len(result.Edges), err)
	}

	accepted, err := db.AcceptSuggestion(ctx, ownerID, result.Edges[0].ID)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if accepted.Origin != OriginManual {
		t.Errorf("accepted suggestion still reads as %q", accepted.Origin)
	}
	if accepted.Confidence != nil {
		t.Error("a connection someone stood behind still carries a machine score")
	}

	if err = db.DeleteEdge(ctx, ownerID, result.Edges[1].ID); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	var remaining int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM note_edges WHERE id=$1`, result.Edges[1].ID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Error("a dismissed suggestion is still in the graph")
	}

	// Accepting is not a way to launder an edge someone else made: only inferred
	// ones can be accepted, so a second accept finds nothing.
	if _, err = db.AcceptSuggestion(ctx, ownerID, result.Edges[0].ID); err == nil {
		t.Error("an already-accepted edge was accepted again")
	}
}

// The property the first version of this file missed. Auto-link skips pairs that
// are already connected, so deleting a suggestion removed the very record that
// kept umm from proposing it — and the next run brought it straight back. A
// suggestion that cannot be got rid of is worse than no suggestion.
func TestDismissedSuggestionsStayDismissedIntegration(t *testing.T) {
	db, ownerID, spaceID := autoLinkSpace(t)
	ctx := context.Background()
	stopGateway := useSemanticStub(t, db, ownerID)
	defer stopGateway()
	assertNotesEmbeddedByTheStub(t, db, ownerID, spaceID)

	first, err := db.SuggestLinks(ctx, ownerID, spaceID)
	if err != nil || len(first.Edges) == 0 {
		t.Fatalf("first run produced nothing: %v", err)
	}
	dismissed := first.Edges[0]
	if err = db.DeleteEdge(ctx, ownerID, dismissed.ID); err != nil {
		t.Fatalf("dismiss: %v", err)
	}

	second, err := db.SuggestLinks(ctx, ownerID, spaceID)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	for _, edge := range second.Edges {
		samePair := (edge.SourceID == dismissed.SourceID && edge.TargetID == dismissed.TargetID) ||
			(edge.SourceID == dismissed.TargetID && edge.TargetID == dismissed.SourceID)
		if samePair {
			t.Fatal("a suggestion that was turned down came back on the next run")
		}
	}

	// Deleting a line a person drew is a different act: they may want it
	// proposed again later, so it must not leave a permanent dismissal.
	drawn, err := db.CreateEdge(ctx, ownerID, Edge{
		SpaceID: spaceID, SourceID: dismissed.SourceID, TargetID: dismissed.TargetID, Relation: RelationSupports,
	})
	if err != nil {
		t.Fatalf("draw a connection: %v", err)
	}
	if err = db.DeleteEdge(ctx, ownerID, drawn.ID); err != nil {
		t.Fatalf("delete a drawn connection: %v", err)
	}
	var dismissals int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM link_dismissals WHERE space_id=$1`, spaceID).Scan(&dismissals); err != nil {
		t.Fatal(err)
	}
	if dismissals != 1 {
		t.Errorf("expected only the inferred edge to leave a dismissal, found %d", dismissals)
	}
}

// A workspace too small to describe a distribution should be left alone rather
// than have its two closest notes joined for want of anything better.
func TestAutoLinkDeclinesATinyWorkspaceIntegration(t *testing.T) {
	db, ownerID, spaceID := autoLinkSpace(t)
	ctx := context.Background()
	stopGateway := useSemanticStub(t, db, ownerID)
	defer stopGateway()

	if _, err := db.Pool.Exec(ctx,
		`UPDATE notes SET deleted_at=now() WHERE space_id=$1 AND id IN (SELECT id FROM notes WHERE space_id=$1 LIMIT 3)`,
		spaceID); err != nil {
		t.Fatal(err)
	}
	result, err := db.SuggestLinks(ctx, ownerID, spaceID)
	if err != nil {
		t.Fatalf("suggest links: %v", err)
	}
	if result.Outcome != OutcomeTooFewNotes {
		t.Fatalf("outcome=%q, want %q", result.Outcome, OutcomeTooFewNotes)
	}
}

// useSemanticStub points the store at a gateway that separates the two subjects
// cleanly and clears umm's own quality measurement, so the tests above exercise
// the path a real sentence embedding model takes rather than the refusal.
func useSemanticStub(t *testing.T, db *Store, userID uuid.UUID) func() {
	t.Helper()
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		type embedding struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}
		payload := struct {
			Data []embedding `json:"data"`
		}{}
		for index, text := range request.Input {
			payload.Data = append(payload.Data, embedding{Embedding: semanticStubVector(text), Index: index})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	ctx := context.Background()
	var previous map[string]any
	hadPrevious := db.GetSetting(ctx, "ai_gateway", &previous) == nil
	// PutSetting deliberately preserves api_key so an administrator saving the
	// masked form does not wipe the stored secret. That also means a leftover
	// "enc:" key from another test survives, and a key this Store has no cipher
	// to decrypt closes the provider down to the local algorithm — silently. The
	// row has to be cleared, not overwritten.
	if _, err := db.Pool.Exec(ctx, `DELETE FROM app_settings WHERE key='ai_gateway'`); err != nil {
		t.Fatal(err)
	}
	if err := db.PutSetting(ctx, "ai_gateway", map[string]any{
		"base_url": gateway.URL, "embedding_model": "stub-semantic", "timeout_seconds": 10,
	}, userID); err != nil {
		t.Fatal(err)
	}
	report, err := db.MeasureEmbeddingQuality(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	// Without this the tests can pass or fail on whichever backend happened to be
	// configured, which is how a suite ends up measuring the wrong thing and
	// reporting success.
	if report.Algorithm == intelligence.LocalAlgorithm {
		var settings map[string]any
		_ = db.GetSetting(ctx, "ai_gateway", &settings)
		t.Fatalf("the stub gateway was not used; vectors came from the offline algorithm (settings=%v)", settings)
	}
	if !report.Semantic {
		t.Fatalf("the stub must clear umm's own gate, got discrimination=%+.3f accuracy=%.3f purity=%.3f",
			report.Discrimination, report.PairwiseAccuracy, report.NeighbourPurity)
	}
	return func() {
		if hadPrevious {
			_ = db.PutSetting(context.Background(), "ai_gateway", previous, userID)
		} else {
			db.Pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key='ai_gateway'`)
		}
		gateway.Close()
	}
}

// semanticStubVector stands in for a working sentence embedding: it knows the
// labelled quality data so the backend clears umm's gate, and it separates the
// two subjects in the test workspace. It is not an embedding algorithm.
func semanticStubVector(text string) []float64 {
	for index, pair := range intelligence.QualityPairs {
		side := 0
		switch text {
		case pair.A:
			side = 1
		case pair.B:
			side = 2
		default:
			continue
		}
		separation := map[intelligence.PairClass]float64{
			intelligence.ClassParaphrase:   0.10,
			intelligence.ClassRelated:      0.40,
			intelligence.ClassLexicalDecoy: 1.30,
			intelligence.ClassUnrelated:    1.45,
		}[pair.Class]
		angle := float64(index) * 0.37
		if side == 2 {
			angle += separation
		}
		return []float64{math.Cos(angle), math.Sin(angle), 0.15, 0, 0, 0, 0}
	}
	for topic, group := range intelligence.QualityTopics {
		for position, sentence := range group.Sentences {
			if sentence == text {
				angle := float64(topic)*(math.Pi/2) + float64(position)*0.05
				return []float64{0, 0, 0, math.Cos(angle), math.Sin(angle), 0, 0}
			}
		}
	}
	// The workspace notes: one direction per subject, spread within it so members
	// of a subject are close without being the same thought.
	//
	// The spread used to be 0.08 radians, which put same-subject pairs at cosine
	// 0.987 and above. Measured on real backends nothing legitimate lands there:
	// bge-m3 puts genuine duplicates at 0.943-0.986 and the next class down tops
	// out at 0.681. These six notes are three distinct thoughts about auth and
	// three about cycling — a real embedding separates them far more than that,
	// and a fixture that does not was quietly modelling every related pair as a
	// duplicate.
	for index, content := range autoLinkNotes {
		if content != text {
			continue
		}
		subject := 0.0
		if index >= 3 {
			subject = math.Pi / 2
		}
		angle := subject + float64(index%3)*0.45
		return []float64{0, 0, 0, 0, 0, math.Cos(angle), math.Sin(angle)}
	}
	return []float64{0, 0, 0, 0, 0, 1, 0}
}

// assertNotesEmbeddedByTheStub forces the workspace vectors and checks what
// produced them.
//
// Measuring the backend is not the same as embedding the notes: the quality
// report goes through the configured provider directly, while note vectors go
// through the cached provider and can silently fall back. A run that scored
// against local vectors reports "nothing stood out", which reads like a finding
// rather than a broken fixture — so this turns it into a specific failure.
func assertNotesEmbeddedByTheStub(t *testing.T, db *Store, ownerID, spaceID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	// Reading a space no longer indexes it — that is what v0.72.0 moved out of
	// the read path — so the precondition is established by asking the sweep to
	// do its work now rather than by listing the notes and hoping.
	if done := db.SweepEmbeddingsOnce(ctx, &spaceID); done == 0 {
		t.Fatal("the sweep found nothing to index, so this proves nothing about which backend embedded the notes")
	}
	rows, err := db.Pool.Query(ctx,
		`SELECT DISTINCT e.algorithm FROM note_embeddings e JOIN notes n ON n.id=e.note_id WHERE n.space_id=$1`, spaceID)
	if err != nil {
		t.Fatalf("read stored algorithms: %v", err)
	}
	defer rows.Close()
	var algorithms []string
	for rows.Next() {
		var algorithm string
		if err := rows.Scan(&algorithm); err != nil {
			t.Fatal(err)
		}
		algorithms = append(algorithms, algorithm)
		if algorithm == intelligence.LocalAlgorithm {
			t.Fatalf("workspace notes were embedded by the offline algorithm, not the stub; "+
				"any suggestion result would describe the wrong backend (algorithms=%v)", algorithms)
		}
	}
	if len(algorithms) == 0 {
		t.Fatal("no note vectors were stored; the run would score empty vectors")
	}
}
