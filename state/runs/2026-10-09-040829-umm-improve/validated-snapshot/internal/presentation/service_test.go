package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// What is tested here is the orchestration: the order of the calls, what
// happens when one of them fails, and what ends up recorded either way. The
// database and the network have their own tests; standing them up here would
// make these slow without making them stronger.

type fakeSpaces struct {
	notes []store.Note
	edges []store.Edge
	name  string
	err   error
	// asked records the space the store was queried for, so a test can prove
	// the permission-carrying call actually happened.
	asked uuid.UUID
}

func (f *fakeSpaces) ListNotes(_ context.Context, _, spaceID uuid.UUID, _ string) ([]store.Note, []store.Edge, error) {
	f.asked = spaceID
	return f.notes, f.edges, f.err
}

func (f *fakeSpaces) ListSpaces(_ context.Context, _ uuid.UUID) ([]store.Space, error) {
	return []store.Space{{ID: f.asked, Name: f.name}}, nil
}

type recordedComplete struct {
	status        string
	source        string
	sources       []store.SlideSource
	thoughtCount  int
	excludedCount int
	trimmedCount  int
	failure       string
}

type fakeLinks struct {
	created   []string
	createErr error
	// made keeps the whole link each create produced, so a test can hand one
	// back to a retry the way the store would.
	made      []store.PresentationLink
	complete  []recordedComplete
	completeE error
	failed    store.PresentationLink
	failedErr error
}

func (f *fakeLinks) FailedPresentationLink(_ context.Context, _, _ uuid.UUID) (store.PresentationLink, error) {
	if f.failedErr != nil {
		return store.PresentationLink{}, f.failedErr
	}
	return f.failed, nil
}

func (f *fakeLinks) CreatePresentationLink(_ context.Context, _, spaceID uuid.UUID, ptiumID, title string, request json.RawMessage) (store.PresentationLink, error) {
	if f.createErr != nil {
		return store.PresentationLink{}, f.createErr
	}
	f.created = append(f.created, ptiumID)
	link := store.PresentationLink{ID: uuid.New(), SpaceID: spaceID, PtiumID: ptiumID, Title: title,
		Status: store.PresentationPending, Request: request}
	f.made = append(f.made, link)
	return link, nil
}

func (f *fakeLinks) CompletePresentationLink(_ context.Context, _, _ uuid.UUID, status, source string, sources []store.SlideSource, thoughtCount, excludedCount, trimmedCount int, failure, failureKind string) error {
	f.complete = append(f.complete, recordedComplete{status, source, sources, thoughtCount, excludedCount, trimmedCount, failure})
	return f.completeE
}

type fakeSettings struct{ cfg Config }

func (f fakeSettings) GetSetting(_ context.Context, _ string, dst any) error {
	raw, _ := json.Marshal(f.cfg)
	return json.Unmarshal(raw, dst)
}

type fakePtium struct {
	deckID     string
	createErr  error
	applyErr   error
	warnings   []string
	slideCount int
	applied    []string
	dryRuns    []bool
	created    int
}

func (f *fakePtium) CreateDeck(_ context.Context, _, _, _ string) (Deck, error) {
	if f.createErr != nil {
		return Deck{}, f.createErr
	}
	f.created++
	return Deck{ID: f.deckID}, nil
}

func (f *fakePtium) ApplySource(_ context.Context, _, source string, dryRun bool) (ApplyResult, error) {
	f.applied = append(f.applied, source)
	f.dryRuns = append(f.dryRuns, dryRun)
	if f.applyErr != nil {
		return ApplyResult{}, f.applyErr
	}
	return ApplyResult{SlideCount: f.slideCount, Warnings: f.warnings}, nil
}

func serviceWith(spaces *fakeSpaces, links *fakeLinks, ptium *fakePtium, cfg Config) *Service {
	return &Service{
		Spaces: spaces, Links: links, Settings: fakeSettings{cfg},
		NewPtium: func(Config) (Ptium, error) { return ptium, nil },
	}
}

func note(n byte, content string, x float64) store.Note {
	return store.Note{ID: id(n), Content: content, X: x}
}

func spaceFixture() *fakeSpaces {
	return &fakeSpaces{
		name: "회고 주기 재검토",
		notes: []store.Note{
			note(1, "회고 주기를 격주로 줄여 보자", 0),
			note(2, "주기가 짧으면 논의가 얕아진다", 400),
		},
		edges: []store.Edge{{SourceID: id(2), TargetID: id(1), Relation: store.RelationSupports, Origin: store.OriginManual}},
	}
}

func TestPreviewChangesNothing(t *testing.T) {
	spaces, links, ptium := spaceFixture(), &fakeLinks{}, &fakePtium{deckID: "pt_1"}
	svc := serviceWith(spaces, links, ptium, Config{BaseURL: "https://ptium.internal"})

	preview, err := svc.Preview(context.Background(), uuid.New(), Request{SpaceID: id(9)})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(preview.Storyline.Slides) == 0 || preview.Source == "" {
		t.Fatalf("preview produced nothing: %+v", preview)
	}
	// Looking at what a space would become must not create anything.
	if ptium.created != 0 || len(ptium.applied) != 0 {
		t.Fatalf("preview touched ptium: created=%d applied=%d", ptium.created, len(ptium.applied))
	}
	if len(links.created) != 0 || len(links.complete) != 0 {
		t.Fatalf("preview wrote a link: %+v %+v", links.created, links.complete)
	}
	// Never checked is not the same as checked and clean.
	if preview.Checked {
		t.Fatal("a preview with no deck to compile against reported itself as checked")
	}
}

// A preview is umm's own work and is worth showing even when there is nowhere
// to send it. Refusing here would make the storyline unreachable for anyone who
// has not connected Ptium yet.
func TestPreviewWorksWithoutPtium(t *testing.T) {
	svc := serviceWith(spaceFixture(), &fakeLinks{}, &fakePtium{}, Config{})
	preview, err := svc.Preview(context.Background(), uuid.New(), Request{SpaceID: id(9)})
	if err != nil {
		t.Fatalf("preview without ptium failed: %v", err)
	}
	if len(preview.Storyline.Slides) == 0 {
		t.Fatal("no storyline")
	}
}

func TestPreviewAgainstDryRunsAndReportsWhatPtiumSaid(t *testing.T) {
	ptium := &fakePtium{deckID: "pt_1", slideCount: 3, warnings: []string{"레이아웃 없음"}}
	svc := serviceWith(spaceFixture(), &fakeLinks{}, ptium, Config{BaseURL: "https://ptium.internal"})

	preview, err := svc.PreviewAgainst(context.Background(), uuid.New(), Request{SpaceID: id(9)}, "pt_1")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(ptium.dryRuns) != 1 || !ptium.dryRuns[0] {
		t.Fatalf("a preview compiled for real: %+v", ptium.dryRuns)
	}
	if preview.SlideCount != 3 || len(preview.Warnings) != 1 || !preview.Checked {
		t.Fatalf("ptium's answer did not reach the preview: %+v", preview)
	}
}

func TestCreateMakesTheDeckAndRecordsIt(t *testing.T) {
	spaces, links, ptium := spaceFixture(), &fakeLinks{}, &fakePtium{deckID: "pt_new", slideCount: 2}
	svc := serviceWith(spaces, links, ptium, Config{BaseURL: "https://ptium.internal", Language: "ko"})

	result, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(links.created) != 1 || links.created[0] != "pt_new" {
		t.Fatalf("the deck was not recorded: %+v", links.created)
	}
	if len(ptium.dryRuns) != 1 || ptium.dryRuns[0] {
		t.Fatalf("create was a dry run: %+v", ptium.dryRuns)
	}
	if len(links.complete) != 1 || links.complete[0].status != store.PresentationReady {
		t.Fatalf("final status: %+v", links.complete)
	}
	if result.Link.Status != store.PresentationReady {
		t.Fatalf("returned link status: %+v", result.Link)
	}
	// A deck whose slides cannot say where they came from is what this whole
	// feature exists to avoid, so the mapping is never optional.
	if len(links.complete[0].sources) == 0 {
		t.Fatal("no slide-to-thought mapping was recorded")
	}
	if !strings.Contains(links.complete[0].source, TracePrefix) {
		t.Fatal("the stored source carries no trace comments")
	}
}

// Recording only on success would leave a deck in Ptium that umm has never
// heard of: invisible in umm, unexplained in Ptium, and a failure nobody can
// look up afterwards.
func TestAFailedCompileIsStillRecorded(t *testing.T) {
	links := &fakeLinks{}
	ptium := &fakePtium{deckID: "pt_bad", applyErr: errors.New("ptium status 422: 레이아웃 없음")}
	svc := serviceWith(spaceFixture(), links, ptium, Config{BaseURL: "https://ptium.internal"})

	if _, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)}); err == nil {
		t.Fatal("a failed compile was reported as success")
	}
	if len(links.created) != 1 {
		t.Fatalf("the deck was not recorded before compiling: %+v", links.created)
	}
	if len(links.complete) != 1 || links.complete[0].status != store.PresentationFailed {
		t.Fatalf("the failure was not recorded: %+v", links.complete)
	}
	if !strings.Contains(links.complete[0].failure, "레이아웃 없음") {
		t.Fatalf("ptium's explanation was lost: %+v", links.complete[0])
	}
}

// The deck exists and umm could not record it. Saying so plainly is the only
// honest option: the two are out of step and only a person can decide what to
// do about it.
func TestADeckUmmCannotRecordSaysWhichDeck(t *testing.T) {
	links := &fakeLinks{createErr: errors.New("permission denied")}
	ptium := &fakePtium{deckID: "pt_orphan"}
	svc := serviceWith(spaceFixture(), links, ptium, Config{BaseURL: "https://ptium.internal"})

	_, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)})
	if err == nil {
		t.Fatal("an unrecorded deck was reported as success")
	}
	if !strings.Contains(err.Error(), "pt_orphan") {
		t.Fatalf("the orphaned deck is unidentifiable: %v", err)
	}
	// And nothing was compiled into a deck umm cannot track.
	if len(ptium.applied) != 0 {
		t.Fatalf("compiled into an unrecorded deck: %+v", ptium.applied)
	}
}

func TestCreateRefusesWithoutPtium(t *testing.T) {
	svc := serviceWith(spaceFixture(), &fakeLinks{}, &fakePtium{}, Config{})
	_, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("got %v, want ErrNotConfigured", err)
	}
}

// An empty space must not produce an empty deck. A deck with no slides in
// someone's Ptium account is litter they have to clean up.
func TestNothingToPresentIsRefusedBeforeAnythingIsMade(t *testing.T) {
	spaces := &fakeSpaces{name: "빈 공간"}
	links, ptium := &fakeLinks{}, &fakePtium{deckID: "pt_empty"}
	svc := serviceWith(spaces, links, ptium, Config{BaseURL: "https://ptium.internal"})

	if _, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)}); !errors.Is(err, ErrNothingToPresent) {
		t.Fatalf("got %v, want ErrNothingToPresent", err)
	}
	if ptium.created != 0 || len(links.created) != 0 {
		t.Fatalf("an empty space still made something: ptium=%d links=%+v", ptium.created, links.created)
	}
}

// Every thought held back from analysis being the only content is the same
// case: nothing may reach a slide, so nothing may be created.
func TestASpaceOfHeldBackThoughtsMakesNothing(t *testing.T) {
	spaces := &fakeSpaces{name: "비공개", notes: []store.Note{{ID: id(1), Content: "개인적인 메모", AIExcluded: true}}}
	ptium := &fakePtium{deckID: "pt_x"}
	svc := serviceWith(spaces, &fakeLinks{}, ptium, Config{BaseURL: "https://ptium.internal"})

	if _, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)}); !errors.Is(err, ErrNothingToPresent) {
		t.Fatalf("got %v, want ErrNothingToPresent", err)
	}
	if ptium.created != 0 {
		t.Fatal("a deck was made out of thoughts that were held back")
	}
}

func TestTheSpaceNameBecomesTheTitle(t *testing.T) {
	spaces := spaceFixture()
	spaces.asked = id(9)
	svc := serviceWith(spaces, &fakeLinks{}, &fakePtium{deckID: "pt_1"}, Config{})

	preview, err := svc.Preview(context.Background(), uuid.New(), Request{SpaceID: id(9)})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Storyline.Title != "회고 주기 재검토" {
		t.Fatalf("title: %q", preview.Storyline.Title)
	}
}

func TestAnExplicitTitleWins(t *testing.T) {
	svc := serviceWith(spaceFixture(), &fakeLinks{}, &fakePtium{}, Config{})
	preview, err := svc.Preview(context.Background(), uuid.New(), Request{SpaceID: id(9), Title: "임원 보고"})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Storyline.Title != "임원 보고" {
		t.Fatalf("title: %q", preview.Storyline.Title)
	}
}

// The store call carries the permission check, so a space the caller cannot
// read comes back empty and compiles to nothing rather than being compiled
// anyway.
func TestAStoreErrorStopsEverything(t *testing.T) {
	spaces := spaceFixture()
	spaces.err = errors.New("no")
	ptium := &fakePtium{deckID: "pt_1"}
	svc := serviceWith(spaces, &fakeLinks{}, ptium, Config{BaseURL: "https://ptium.internal"})

	if _, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)}); err == nil {
		t.Fatal("a store failure was ignored")
	}
	if ptium.created != 0 {
		t.Fatal("a deck was made from a space that could not be read")
	}
}

func TestSlideSourcesCountThePositionsPtiumWillProduce(t *testing.T) {
	story := Compile([]Thought{
		{ID: id(1), Content: "주장"},
		{ID: id(2), Content: "근거", X: 400},
	}, []Link{{From: id(2), To: id(1), Relation: store.RelationSupports}}, Options{Title: "제목"})

	sources := story.SlideSources()
	if len(sources) == 0 {
		t.Fatal("no sources")
	}
	// The cover is position 1 and quotes nothing, so the first slide with
	// content is position 2 — the same numbering Ptium compiles from this
	// source, which is what makes the mapping mean anything.
	for _, src := range sources {
		if src.SlidePosition < 2 {
			t.Fatalf("a source was credited to the cover: %+v", src)
		}
	}
}

func TestWithoutATitleTheFirstSlideIsPositionOne(t *testing.T) {
	story := Compile([]Thought{{ID: id(1), Content: "주장"}}, nil, Options{})
	sources := story.SlideSources()
	if len(sources) != 1 || sources[0].SlidePosition != 1 {
		t.Fatalf("positions are off when there is no cover: %+v", sources)
	}
}

type fakeCipher struct{ calls int }

func (f *fakeCipher) Decrypt(value string) (string, error) {
	f.calls++
	return "plain:" + value, nil
}

// Secrets are written to app_settings encrypted and prefixed "enc:". Sending
// that string as a bearer credential got a 401 from a real Ptium — which reads
// exactly like a wrong key and sends whoever is debugging it to check the one
// thing that was right.
func TestAStoredCredentialIsDecryptedBeforeItIsSent(t *testing.T) {
	cipher := &fakeCipher{}
	ptium := &fakePtium{deckID: "pt_1"}
	var sentKey string
	svc := &Service{
		Spaces: spaceFixture(), Links: &fakeLinks{},
		Settings: fakeSettings{Config{BaseURL: "https://ptium.internal", APIKey: "enc:ABC123"}},
		Cipher:   cipher,
		NewPtium: func(cfg Config) (Ptium, error) { sentKey = cfg.APIKey; return ptium, nil },
	}

	if _, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if cipher.calls != 1 {
		t.Fatalf("the credential was not decrypted (%d calls)", cipher.calls)
	}
	if sentKey != "plain:ABC123" {
		t.Fatalf("ptium was given %q", sentKey)
	}
	if strings.HasPrefix(sentKey, "enc:") {
		t.Fatal("the ciphertext was sent as a bearer credential")
	}
}

// A credential that was never encrypted still has to work, so an operator who
// pasted one in before encryption was on is not locked out.
func TestAPlainCredentialIsSentAsItIs(t *testing.T) {
	cipher := &fakeCipher{}
	var sentKey string
	svc := &Service{
		Spaces: spaceFixture(), Links: &fakeLinks{},
		Settings: fakeSettings{Config{BaseURL: "https://ptium.internal", APIKey: "ptium_plain"}},
		Cipher:   cipher,
		NewPtium: func(cfg Config) (Ptium, error) { sentKey = cfg.APIKey; return &fakePtium{deckID: "pt_1"}, nil },
	}
	if _, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if cipher.calls != 0 {
		t.Fatal("a plain credential was run through the cipher")
	}
	if sentKey != "ptium_plain" {
		t.Fatalf("ptium was given %q", sentKey)
	}
}

// Without a key, saying so beats sending ciphertext and getting a 401 that
// blames the credential.
func TestAnUnreadableCredentialSaysSoRatherThanFailingAsA401(t *testing.T) {
	svc := &Service{
		Spaces: spaceFixture(), Links: &fakeLinks{},
		Settings: fakeSettings{Config{BaseURL: "https://ptium.internal", APIKey: "enc:ABC123"}},
		NewPtium: func(Config) (Ptium, error) { return &fakePtium{deckID: "pt_1"}, nil },
	}
	_, err := svc.Create(context.Background(), uuid.New(), Request{SpaceID: id(9)})
	if err == nil || !strings.Contains(err.Error(), "encrypted") {
		t.Fatalf("got %v, want an error naming the unreadable credential", err)
	}
}

// Retrying is asking for the same talk again.
//
// Making a deck is two calls, and when the second fails umm offers to compile
// into the deck Ptium already opened. It recompiles the space on purpose — the
// thinking has usually moved on — but the request had not moved on, and it was
// being thrown away: a talk built from two chosen thoughts and capped at one
// slide came back as the whole space, uncapped, into that same deck.
func TestARetryAsksForTheSameTalkAgain(t *testing.T) {
	spaces := &fakeSpaces{name: "회고 주기 재검토", notes: []store.Note{
		note(1, "회고 주기를 격주로 줄여 보자", 0),
		note(2, "회고 준비에 두 시간이 든다", 2000),
		note(3, "격주 회고는 논의가 얕아진다", 4000),
		note(4, "지난 분기 회고 기록", 6000),
	}}
	links := &fakeLinks{}
	ptium := &fakePtium{deckID: "pt_retry", applyErr: errors.New("ptium status 504: timed out")}
	svc := serviceWith(spaces, links, ptium, Config{BaseURL: "https://ptium.internal"})
	user := uuid.New()

	wanted := Request{SpaceID: id(9), Only: []uuid.UUID{id(1), id(2)}, MaxSlides: 1}
	if _, err := svc.Create(context.Background(), user, wanted); err == nil {
		t.Fatal("a failed compile was reported as success")
	}
	if len(links.made) != 1 {
		t.Fatalf("the deck was not recorded: %+v", links.made)
	}

	// The deck is open in Ptium and empty, so the person presses 다시 시도.
	links.failed = links.made[0]
	ptium.applyErr = nil
	if _, err := svc.Retry(context.Background(), user, links.made[0].ID); err != nil {
		t.Fatalf("retry: %v", err)
	}

	if len(ptium.applied) != 2 {
		t.Fatalf("the retry did not compile into the deck: %+v", ptium.applied)
	}
	if ptium.applied[1] != ptium.applied[0] {
		t.Fatalf("a retry made a different talk:\nasked for:\n%s\ngot:\n%s", ptium.applied[0], ptium.applied[1])
	}
	if strings.Contains(ptium.applied[1], "격주 회고는 논의가 얕아진다") {
		t.Fatal("a retry pulled in a thought the person had not selected")
	}
}

// Every switch, including the two that reach a model. A retry that quietly
// turned them off would hand back a deck with neither the names nor the parts
// that were asked for.
func TestARetryKeepsTheSwitchesThatWereOn(t *testing.T) {
	links := &fakeLinks{}
	ptium := &fakePtium{deckID: "pt_switches", applyErr: errors.New("ptium status 504: timed out")}
	svc := serviceWith(spaceFixture(), links, ptium, Config{BaseURL: "https://ptium.internal"})
	user := uuid.New()

	wanted := Request{SpaceID: id(9), OneSlidePerThought: true, IncludeExcluded: true,
		NameGroups: true, SectionDeck: true, MaxSlides: 12}
	if _, err := svc.Create(context.Background(), user, wanted); err == nil {
		t.Fatal("a failed compile was reported as success")
	}
	if len(links.made) != 1 {
		t.Fatalf("the deck was not recorded: %+v", links.made)
	}

	var stored asked
	if err := json.Unmarshal(links.made[0].Request, &stored); err != nil {
		t.Fatalf("what was asked for was not recorded readably: %v", err)
	}
	if !stored.IncludeExcluded || !stored.OneSlidePerThought || !stored.NameGroups ||
		!stored.SectionDeck || stored.MaxSlides != 12 {
		t.Fatalf("a choice was not recorded: %+v", stored)
	}

	again := asRequest(links.made[0])
	if again.SpaceID != id(9) || !again.OneSlidePerThought || !again.IncludeExcluded ||
		!again.NameGroups || !again.SectionDeck || again.MaxSlides != 12 {
		t.Fatalf("the request a retry would compile: %+v", again)
	}
}

// A deck recorded before umm kept the request still retries. It compiles the
// whole space with every default, which is exactly what a retry did then —
// worse than the talk that was asked for, and better than refusing.
func TestARetryOfADeckFromBeforeTheRequestWasKept(t *testing.T) {
	spaces := &fakeSpaces{name: "회고 주기 재검토", notes: []store.Note{
		note(1, "회고 주기를 격주로 줄여 보자", 0),
		note(2, "격주 회고는 논의가 얕아진다", 2000),
	}}
	links := &fakeLinks{failed: store.PresentationLink{ID: uuid.New(), SpaceID: id(9), Title: "회고 주기 재검토",
		PtiumID: "pt_old", Status: store.PresentationFailed}}
	ptium := &fakePtium{deckID: "pt_old"}
	svc := serviceWith(spaces, links, ptium, Config{BaseURL: "https://ptium.internal"})

	if _, err := svc.Retry(context.Background(), uuid.New(), links.failed.ID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(ptium.applied) != 1 || !strings.Contains(ptium.applied[0], "격주 회고는 논의가 얕아진다") {
		t.Fatalf("a link with no recorded request did not compile the whole space: %+v", ptium.applied)
	}
}
