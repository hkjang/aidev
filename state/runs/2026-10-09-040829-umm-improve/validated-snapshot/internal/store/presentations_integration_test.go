package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// This layer's whole job is access control and keeping a link and its sources
// in step. Neither can be shown without a real database: an in-memory fake
// would be asserting that the fake behaves, and the SQL is where the rule
// actually lives.

func presentationFixture(t *testing.T) (*Store, uuid.UUID, uuid.UUID, []uuid.UUID) {
	t.Helper()
	db, userID, spaceID := retrievalSpace(t)
	ctx := context.Background()

	notes := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for i, id := range notes {
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,$4)`,
			id, spaceID, userID, []string{"문제", "원인", "대안"}[i]); err != nil {
			t.Fatal(err)
		}
	}
	return db, userID, spaceID, notes
}

func TestPresentationLinkRoundTripIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	ctx := context.Background()

	link, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_abc", "회고 주기 재검토", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Written before compiling, so a compile that fails still leaves a record
	// rather than a deck in Ptium umm has never heard of.
	if link.Status != PresentationPending {
		t.Fatalf("a new link should be pending, got %q", link.Status)
	}

	sources := []SlideSource{
		{SlidePosition: 1, NoteID: notes[0]},
		{SlidePosition: 2, NoteID: notes[1]},
		{SlidePosition: 2, NoteID: notes[2]},
	}
	const source = "# 회고\n@cover\n"
	// Three counts that mean three different things, given three different
	// values so that reading them back proves they did not get crossed: used,
	// held out of analysis by their author, and left out only by a length cap.
	if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationReady, source, sources, 3, 1, 7, "", ""); err != nil {
		t.Fatal(err)
	}

	links, err := db.ListPresentationLinks(ctx, userID, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].Status != PresentationReady || links[0].ThoughtCount != 3 || links[0].ExcludedCount != 1 {
		t.Fatalf("link not recorded as expected: %+v", links)
	}
	if links[0].TrimmedCount != 7 {
		t.Fatalf("trimmed count came back as %d, want 7: %+v", links[0].TrimmedCount, links[0])
	}
	// The list is read far more often than any one source, so it must not carry
	// megabytes of deck text to draw a few rows.
	if links[0].CompiledSource != "" {
		t.Fatalf("the list carried the compiled source: %q", links[0].CompiledSource)
	}

	got, err := db.PresentationLinkSource(ctx, userID, link.ID)
	if err != nil || got != source {
		t.Fatalf("source round trip: %q, %v", got, err)
	}

	stored, err := db.PresentationSources(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 3 {
		t.Fatalf("expected three source rows, got %+v", stored)
	}
}

// Recompiling replaces the mapping. Adding to it would leave a slide claiming
// thoughts that are no longer on it.
func TestRecompilingReplacesTheSourceMappingIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	ctx := context.Background()

	link, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_recompile", "덱", nil)
	if err != nil {
		t.Fatal(err)
	}
	first := []SlideSource{{SlidePosition: 1, NoteID: notes[0]}, {SlidePosition: 2, NoteID: notes[1]}}
	if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationReady, "a", first, 2, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	second := []SlideSource{{SlidePosition: 1, NoteID: notes[2]}}
	if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationReady, "b", second, 1, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}

	stored, err := db.PresentationSources(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].NoteID != notes[2] {
		t.Fatalf("the old mapping survived a recompile: %+v", stored)
	}
}

// The other direction, and the reason the table earns its place: someone
// editing a note that decks quote is making a different decision from someone
// editing one nobody has used.
func TestANoteKnowsWhichTalksUsedItIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	ctx := context.Background()

	for _, name := range []string{"pt_one", "pt_two"} {
		link, err := db.CreatePresentationLink(ctx, userID, spaceID, name, name, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationReady, "x",
			[]SlideSource{{SlidePosition: 1, NoteID: notes[0]}}, 1, 0, 0, "", ""); err != nil {
			t.Fatal(err)
		}
	}

	used, err := db.PresentationsUsingNote(ctx, userID, notes[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(used) != 2 {
		t.Fatalf("expected both talks, got %+v", used)
	}
	unused, err := db.PresentationsUsingNote(ctx, userID, notes[1])
	if err != nil {
		t.Fatal(err)
	}
	if len(unused) != 0 {
		t.Fatalf("a note nobody used reported %d talks", len(unused))
	}
}

// A thought from another space would credit a slide to something the deck's
// readers cannot see.
func TestASlideCannotCiteAThoughtFromAnotherSpaceIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	ctx := context.Background()

	otherSpace := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'다른 공간')`, otherSpace, userID); err != nil {
		t.Fatal(err)
	}
	outsider := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO notes(id,space_id,author_id,content) VALUES($1,$2,$3,'남의 생각')`,
		outsider, otherSpace, userID); err != nil {
		t.Fatal(err)
	}

	link, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_outsider", "덱", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = db.CompletePresentationLink(ctx, userID, link.ID, PresentationReady, "x", []SlideSource{
		{SlidePosition: 1, NoteID: notes[0]},
		{SlidePosition: 1, NoteID: outsider},
	}, 2, 0, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}

	stored, err := db.PresentationSources(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].NoteID != notes[0] {
		t.Fatalf("a thought from another space was cited: %+v", stored)
	}
}

func TestAStrangerCannotMakeOrReadALinkIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	ctx := context.Background()

	stranger := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,'stranger_pres'::citext,'stranger')`, stranger); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, stranger)

	if _, err := db.CreatePresentationLink(ctx, stranger, spaceID, "pt_nope", "덱", nil); err == nil {
		t.Fatal("a stranger made a deck from someone else's space")
	}

	link, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_private", "덱", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationReady, "비밀 소스",
		[]SlideSource{{SlidePosition: 1, NoteID: notes[0]}}, 1, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}

	if err := db.CompletePresentationLink(ctx, stranger, link.ID, PresentationFailed, "x", nil, 0, 0, 0, "hacked", ""); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("a stranger wrote to someone else's link: %v", err)
	}
	if links, err := db.ListPresentationLinks(ctx, stranger, spaceID); err != nil || len(links) != 0 {
		t.Fatalf("a stranger listed someone else's talks: %+v %v", links, err)
	}
	if sources, err := db.PresentationSources(ctx, stranger, link.ID); err != nil || len(sources) != 0 {
		t.Fatalf("a stranger read someone else's slide sources: %+v %v", sources, err)
	}
	if _, err := db.PresentationLinkSource(ctx, stranger, link.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("a stranger read someone else's deck source: %v", err)
	}
	if used, err := db.PresentationsUsingNote(ctx, stranger, notes[0]); err != nil || len(used) != 0 {
		t.Fatalf("a stranger saw which talks used someone else's note: %+v %v", used, err)
	}
}

func TestAnUnknownStatusIsRefusedIntegration(t *testing.T) {
	db, userID, spaceID, _ := presentationFixture(t)
	ctx := context.Background()

	link, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_status", "덱", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Refused in Go rather than left to the CHECK constraint, so the API can
	// answer 400 instead of 500.
	if err := db.CompletePresentationLink(ctx, userID, link.ID, "exploded", "x", nil, 0, 0, 0, "", ""); !errors.Is(err, ErrUnknownPresentationStatus) {
		t.Fatalf("an unknown status was accepted: %v", err)
	}
}

// What was asked for comes back with the deck that failed.
//
// Retrying compiles the space again into the deck Ptium already opened, and the
// request that produced it is gone by then: it lived in the HTTP call that
// failed. Without this row a retry asks for the whole space with every default
// — a different talk, into the same deck, saying nothing about it.
func TestAFailedLinkCarriesWhatWasAskedForIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	ctx := context.Background()

	wanted := json.RawMessage(`{"only":["` + notes[0].String() + `"],"maxSlides":20,"sectionDeck":true}`)
	link, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_asked", "임원 보고", wanted)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationFailed, "", nil, 0, 0, 0, "시간 초과", "timed-out"); err != nil {
		t.Fatal(err)
	}

	failed, err := db.FailedPresentationLink(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err := json.Unmarshal(failed.Request, &stored); err != nil {
		t.Fatalf("what was asked for did not survive the round trip: %q (%v)", failed.Request, err)
	}
	if stored["maxSlides"] != float64(20) || stored["sectionDeck"] != true {
		t.Fatalf("a choice was lost: %+v", stored)
	}
	only, ok := stored["only"].([]any)
	if !ok || len(only) != 1 || only[0] != notes[0].String() {
		t.Fatalf("the selection was lost: %+v", stored["only"])
	}

	// A deck made before umm kept the request reads as no choices at all, which
	// is what a retry did then. Not null: nothing downstream should have to
	// tell "asked for nothing" from "cannot be read".
	plain, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_plain", "덱", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain.Request) != "{}" {
		t.Fatalf("a link with nothing asked for recorded %q", plain.Request)
	}
}

func TestAFailedCompileKeepsWhatPtiumSaidIntegration(t *testing.T) {
	db, userID, spaceID, _ := presentationFixture(t)
	ctx := context.Background()

	link, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_failed", "덱", nil)
	if err != nil {
		t.Fatal(err)
	}
	const why = "템플릿에 해당 레이아웃이 없습니다"
	if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationFailed, "", nil, 0, 0, 0, why, ""); err != nil {
		t.Fatal(err)
	}
	links, err := db.ListPresentationLinks(ctx, userID, spaceID)
	if err != nil || len(links) != 1 {
		t.Fatalf("list: %+v %v", links, err)
	}
	// A failure that is still explainable after the job is long gone.
	if links[0].Status != PresentationFailed || links[0].Error != why {
		t.Fatalf("the reason was lost: %+v", links[0])
	}
}

// Deleting a space is a decision about the space; the deck lives in Ptium and
// is unaffected, but umm's record of it has nothing left to hang from.
func TestDeletingASpaceTakesItsLinksIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	ctx := context.Background()

	link, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_cascade", "덱", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationReady, "x",
		[]SlideSource{{SlidePosition: 1, NoteID: notes[0]}}, 1, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `DELETE FROM spaces WHERE id=$1`, spaceID); err != nil {
		t.Fatal(err)
	}

	var remaining int
	if err := db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM presentation_sources WHERE presentation_link_id=$1`, link.ID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("%d source rows outlived their space", remaining)
	}
}

// Whether a deck is still true.
//
// The whole feature turns on one distinction: moving a thought must not make a
// slide stale, and rewriting one must. Notes bump their version on any update
// and x, y, width, height and rotation share that statement, so the obvious
// signal would report every deck as stale the moment anyone dragged a note —
// on this canvas, permanently, and the warning would come to mean nothing.

func freshDeck(t *testing.T, db *Store, userID, spaceID uuid.UUID, notes []uuid.UUID) PresentationLink {
	t.Helper()
	ctx := context.Background()
	link, err := db.CreatePresentationLink(ctx, userID, spaceID, "pt_"+uuid.NewString(), "덱", nil)
	if err != nil {
		t.Fatal(err)
	}
	sources := make([]SlideSource, 0, len(notes))
	for i, id := range notes {
		sources = append(sources, SlideSource{SlidePosition: i + 2, NoteID: id})
	}
	if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationReady, "x", sources, len(notes), 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	return link
}

func TestAFreshDeckHasNoStaleSlidesIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	link := freshDeck(t, db, userID, spaceID, notes)

	stale, err := db.StaleSlides(context.Background(), userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("a deck compiled a moment ago is already stale: %+v", stale)
	}
}

// The distinction the feature lives or dies on.
func TestMovingAThoughtDoesNotMakeASlideStaleIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	link := freshDeck(t, db, userID, spaceID, notes)
	ctx := context.Background()

	// Exactly what dragging a note across the canvas does, version bump and all.
	if _, err := db.Pool.Exec(ctx,
		`UPDATE notes SET x=1234, y=567, width=400, height=300, rotation=2, version=version+1, updated_at=now() WHERE id=$1`,
		notes[0]); err != nil {
		t.Fatal(err)
	}

	stale, err := db.StaleSlides(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("moving a thought marked the deck stale: %+v", stale)
	}
}

func TestRewritingAThoughtMakesItsSlideStaleIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	link := freshDeck(t, db, userID, spaceID, notes)
	ctx := context.Background()

	if _, err := db.Pool.Exec(ctx,
		`UPDATE notes SET content='생각이 바뀌었다', version=version+1, updated_at=now() WHERE id=$1`, notes[1]); err != nil {
		t.Fatal(err)
	}

	stale, err := db.StaleSlides(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0].NoteID != notes[1] {
		t.Fatalf("the rewritten thought was not reported: %+v", stale)
	}
	if stale[0].Reason != StaleChanged {
		t.Fatalf("reason: %q", stale[0].Reason)
	}
	// What it says now, so a person can see the change without leaving the deck.
	if stale[0].Content != "생각이 바뀌었다" {
		t.Fatalf("the current text was not carried: %q", stale[0].Content)
	}
	// And it names the slide, not just the deck.
	if stale[0].SlidePosition != 3 {
		t.Fatalf("slide position: %d", stale[0].SlidePosition)
	}
}

// A retitled thought is a rewritten one: the title is what a slide is called.
func TestRetitlingAThoughtMakesItsSlideStaleIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	link := freshDeck(t, db, userID, spaceID, notes)
	ctx := context.Background()

	if _, err := db.Pool.Exec(ctx, `UPDATE notes SET title='새 제목' WHERE id=$1`, notes[0]); err != nil {
		t.Fatal(err)
	}
	stale, err := db.StaleSlides(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0].NoteID != notes[0] {
		t.Fatalf("a retitled thought was not reported: %+v", stale)
	}
}

// A deleted thought and a rewritten one call for different things — one slide
// can be brought up to date, the other has lost its source entirely — so they
// are not collapsed into "out of date".
func TestADeletedThoughtIsReportedAsDeletedIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	link := freshDeck(t, db, userID, spaceID, notes)
	ctx := context.Background()

	if _, err := db.Pool.Exec(ctx, `UPDATE notes SET deleted_at=now() WHERE id=$1`, notes[2]); err != nil {
		t.Fatal(err)
	}
	stale, err := db.StaleSlides(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0].Reason != StaleDeleted {
		t.Fatalf("a deleted thought: %+v", stale)
	}
	if stale[0].Content != "" {
		t.Fatalf("a deleted thought reported text: %q", stale[0].Content)
	}
}

// Claiming a slide is stale when nobody knows is worse than saying nothing.
func TestASlideCompiledBeforeFingerprintsIsNeverStaleIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	link := freshDeck(t, db, userID, spaceID, notes)
	ctx := context.Background()

	// Exactly the state an upgrade leaves behind.
	if _, err := db.Pool.Exec(ctx,
		`UPDATE presentation_sources SET note_fingerprint='' WHERE presentation_link_id=$1`, link.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `UPDATE notes SET content='완전히 다른 내용' WHERE id=$1`, notes[0]); err != nil {
		t.Fatal(err)
	}

	stale, err := db.StaleSlides(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("a deck of unknown freshness was reported as stale: %+v", stale)
	}
}

func TestStaleCountsAreDrawnInOneQueryIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	ctx := context.Background()
	changed := freshDeck(t, db, userID, spaceID, notes)
	untouched := freshDeck(t, db, userID, spaceID, notes[:1])

	if _, err := db.Pool.Exec(ctx, `UPDATE notes SET content='바뀜' WHERE id=$1`, notes[1]); err != nil {
		t.Fatal(err)
	}

	counts, err := db.StaleCounts(ctx, userID, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if counts[changed.ID] != 1 {
		t.Fatalf("the affected deck counts %d slides", counts[changed.ID])
	}
	// A deck that does not quote the changed thought is not affected by it.
	if _, ok := counts[untouched.ID]; ok {
		t.Fatalf("a deck that never quoted the change was reported: %+v", counts)
	}
}

// Recompiling brings a deck back up to date rather than leaving the warning
// standing for a slide that has just been rewritten from the current text.
func TestRecompilingClearsTheWarningIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	link := freshDeck(t, db, userID, spaceID, notes)
	ctx := context.Background()

	if _, err := db.Pool.Exec(ctx, `UPDATE notes SET content='고쳐 씀' WHERE id=$1`, notes[0]); err != nil {
		t.Fatal(err)
	}
	if stale, _ := db.StaleSlides(ctx, userID, link.ID); len(stale) != 1 {
		t.Fatalf("expected one stale slide first, got %+v", stale)
	}

	sources := []SlideSource{{SlidePosition: 2, NoteID: notes[0]}}
	if err := db.CompletePresentationLink(ctx, userID, link.ID, PresentationReady, "x", sources, 1, 0, 0, "", ""); err != nil {
		t.Fatal(err)
	}
	stale, err := db.StaleSlides(ctx, userID, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("recompiling left the warning standing: %+v", stale)
	}
}

func TestAStrangerCannotSeeWhichSlidesWentStaleIntegration(t *testing.T) {
	db, userID, spaceID, notes := presentationFixture(t)
	link := freshDeck(t, db, userID, spaceID, notes)
	ctx := context.Background()

	stranger := uuid.New()
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name) VALUES($1,'stale_stranger'::citext,'x')`, stranger); err != nil {
		t.Fatal(err)
	}
	defer db.Pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, stranger)

	if _, err := db.Pool.Exec(ctx, `UPDATE notes SET content='바뀜' WHERE id=$1`, notes[0]); err != nil {
		t.Fatal(err)
	}
	if stale, err := db.StaleSlides(ctx, stranger, link.ID); err != nil || len(stale) != 0 {
		t.Fatalf("a stranger read someone else's stale slides: %+v %v", stale, err)
	}
	if counts, err := db.StaleCounts(ctx, stranger, spaceID); err != nil || len(counts) != 0 {
		t.Fatalf("a stranger counted someone else's stale slides: %+v %v", counts, err)
	}
}
