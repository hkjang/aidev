package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

/*
Turning a space into a deck, end to end.

Preview compiles and shows; Create makes the deck. Someone looking at what
their space would become must not thereby change anything — not their space,
not Ptium — so the preview paths never write a link, never create a deck, and
pass dryRun down when they do reach Ptium at all.

The dependencies are interfaces rather than the concrete store and client so
that the orchestration — the order of the calls, what happens when one fails,
what gets recorded either way — can be tested without a database or a live
Ptium. The parts that genuinely need those have their own tests elsewhere.
*/

// Spaces is the part of the store this needs to read a space.
type Spaces interface {
	ListNotes(ctx context.Context, userID, spaceID uuid.UUID, query string) ([]store.Note, []store.Edge, error)
	ListSpaces(ctx context.Context, userID uuid.UUID) ([]store.Space, error)
}

// Links is the part of the store this needs to record what happened.
type Links interface {
	CreatePresentationLink(ctx context.Context, userID, spaceID uuid.UUID, ptiumID, title string, request json.RawMessage) (store.PresentationLink, error)
	CompletePresentationLink(ctx context.Context, userID, linkID uuid.UUID, status, source string, sources []store.SlideSource, thoughtCount, excludedCount, trimmedCount int, failure, failureKind string) error
	FailedPresentationLink(ctx context.Context, userID, linkID uuid.UUID) (store.PresentationLink, error)
}

// Naming is optional: with none, or with one that fails, a deck compiles with
// the headings umm derived itself, which is what it had before.

// Settings reads umm's configuration.
type Settings interface {
	GetSetting(ctx context.Context, key string, dst any) error
}

// Decrypter unwraps a stored credential. It stays an interface so this package
// keeps no hard dependency on key management, the same way the store does.
type Decrypter interface {
	Decrypt(string) (string, error)
}

// Ptium is what the service needs from a Ptium installation.
type Ptium interface {
	CreateDeck(ctx context.Context, title, templateID, language string) (Deck, error)
	ApplySource(ctx context.Context, deckID, source string, dryRun bool) (ApplyResult, error)
}

// Config is umm's `ptium` settings section.
type Config struct {
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"api_key"`
	TemplateID     string `json:"template_id"`
	Language       string `json:"language"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// Service compiles spaces into decks.
type Service struct {
	Spaces   Spaces
	Links    Links
	Settings Settings
	// Cipher unwraps the stored Ptium credential. Secrets are written to
	// app_settings encrypted and prefixed "enc:", so without this the bridge
	// sends the ciphertext as a bearer token and Ptium answers 401 — which
	// reads exactly like a wrong key and sends the operator to check the one
	// thing that was right.
	Cipher Decrypter
	// NewPtium builds a client from the configured address. Injected so a test
	// can hand back a stub without standing up an HTTP server, and so the
	// credential is read at call time rather than cached past a rotation.
	NewPtium func(cfg Config) (Ptium, error)
	// Namer proposes a heading for each group of thoughts that were put together
	// by position rather than by anything the person said. Optional in the
	// strongest sense: nil, or one that fails, leaves the deck exactly as it
	// compiled. Polish that breaks must not break the thing it was polishing.
	Namer Namer
	// Sectioner proposes where a long talk divides into parts. Optional on the
	// same terms.
	Sectioner Sectioner
}

// Request is what a person asked for.
type Request struct {
	SpaceID uuid.UUID
	// Title overrides the space's name.
	Title string
	// Only restricts the deck to a selection, so a cluster or a few chosen
	// thoughts can become a talk without the rest of the space.
	Only []uuid.UUID
	// IncludeExcluded overrides the note-level mark. Off unless asked.
	IncludeExcluded bool
	// OneSlidePerThought turns off grouping thoughts by where they sit.
	OneSlidePerThought bool
	// SectionDeck asks the chat model where a long talk divides into parts. It
	// adds heading slides and moves nothing: the order is a follows chain or the
	// layout the person made, and rearranging that would override what they
	// said.
	SectionDeck bool
	// NameGroups asks the chat model to name each group of thoughts that were
	// put together by position. Opt-in, because it sends those thoughts to the
	// gateway and because it makes the deck stop being the same every time.
	// Nothing else about the deck changes: the sentences on the slides stay the
	// person's either way.
	NameGroups bool
	// MaxSlides caps how long the talk may be. Zero means no cap.
	MaxSlides int
}

// asked is what the person chose, in the form it is stored beside the deck.
//
// Only the choices. The space and the title are already columns on the link,
// and taking them from there rather than from here keeps one answer to "which
// space is this deck of" instead of two that can disagree.
//
// The two that reach a model — naming groups, dividing into parts — are kept
// like the rest. A retry is meant to produce the talk that was asked for, and
// leaving them out would quietly hand back a deck with neither, which is the
// same silent substitution this exists to stop. That the model may name a group
// differently the second time is what asking a model for headings means, and
// the screen already says the headings are its.
type asked struct {
	Only               []uuid.UUID `json:"only,omitempty"`
	IncludeExcluded    bool        `json:"includeExcluded,omitempty"`
	OneSlidePerThought bool        `json:"oneSlidePerThought,omitempty"`
	NameGroups         bool        `json:"nameGroups,omitempty"`
	SectionDeck        bool        `json:"sectionDeck,omitempty"`
	MaxSlides          int         `json:"maxSlides,omitempty"`
}

// recorded renders what was asked for, to be kept with the deck it produced.
func (r Request) recorded() json.RawMessage {
	raw, err := json.Marshal(asked{
		Only:               r.Only,
		IncludeExcluded:    r.IncludeExcluded,
		OneSlidePerThought: r.OneSlidePerThought,
		NameGroups:         r.NameGroups,
		SectionDeck:        r.SectionDeck,
		MaxSlides:          r.MaxSlides,
	})
	if err != nil {
		// Nothing in here can fail to marshal. If that ever changes, recording
		// nothing is what a link written before this looks like, and a retry
		// still makes a deck.
		return nil
	}
	return raw
}

// asRequest rebuilds what to compile from a link that failed.
//
// A link written before umm kept the choices — or one whose record cannot be
// read — decodes to none of them, which is exactly what a retry did then: the
// whole space, uncapped, with every switch off. That is worse than the talk
// they asked for and better than refusing to retry at all.
func asRequest(link store.PresentationLink) Request {
	req := Request{SpaceID: link.SpaceID, Title: link.Title}
	var a asked
	if len(link.Request) == 0 || json.Unmarshal(link.Request, &a) != nil {
		return req
	}
	req.Only = a.Only
	req.IncludeExcluded = a.IncludeExcluded
	req.OneSlidePerThought = a.OneSlidePerThought
	req.NameGroups = a.NameGroups
	req.SectionDeck = a.SectionDeck
	req.MaxSlides = a.MaxSlides
	return req
}

// Preview is what a space would become, without anything having happened.
type Preview struct {
	Storyline Storyline `json:"storyline"`
	Source    string    `json:"source"`
	// SlideCount and Warnings come from Ptium compiling against the real
	// template, so a preview is measured rather than guessed. They are absent
	// when no Ptium is configured, which is not an error: the storyline is
	// still worth showing.
	SlideCount int      `json:"slideCount"`
	Warnings   []string `json:"warnings"`
	// Checked says whether Ptium actually saw this. Without it a preview with
	// no warnings and one that was never checked look identical.
	Checked bool `json:"checked"`
}

// Result is a deck that now exists.
type Result struct {
	Link     store.PresentationLink `json:"link"`
	Warnings []string               `json:"warnings"`
}

// ErrNothingToPresent is returned when a space has no thought that could reach
// a slide, so the API answers 400 rather than making an empty deck.
var ErrNothingToPresent = errors.New("nothing in this space can become a talk")

// Preview compiles the space and, if Ptium is configured, asks it what would
// happen. Nothing is written and no deck is created.
func (s *Service) Preview(ctx context.Context, userID uuid.UUID, req Request) (Preview, error) {
	story, source, err := s.compile(ctx, userID, req)
	if err != nil {
		return Preview{}, err
	}
	// Checked stays false, and that is not a shortcoming to be papered over.
	// Ptium can only dry-run source against a deck, because the template is
	// what decides whether a slide fits, and it has no endpoint that validates
	// source without one. Creating a throwaway deck to check against would
	// leave litter in Ptium every time someone looked at a preview.
	//
	// So a first preview shows umm's own work: the order, the grouping, and the
	// person's own sentences. That is the part they are there to review. Use
	// PreviewAgainst once a deck exists to have Ptium measure it.
	return Preview{Storyline: story, Source: source, Warnings: []string{}}, nil
}

// Outline compiles the space and writes it as a Markdown document.
//
// The same compile as a deck preview, so a person who has read the preview
// knows exactly what they are getting: the order is theirs either way. Nothing
// is written and nothing is sent anywhere — this never reaches Ptium, and does
// not need it configured.
func (s *Service) Outline(ctx context.Context, userID uuid.UUID, req Request) (string, error) {
	story, _, err := s.compile(ctx, userID, req)
	if err != nil {
		return "", err
	}
	return WriteOutline(story), nil
}

// PreviewAgainst compiles the space and checks it against an existing deck,
// changing neither.
func (s *Service) PreviewAgainst(ctx context.Context, userID uuid.UUID, req Request, deckID string) (Preview, error) {
	preview, err := s.Preview(ctx, userID, req)
	if err != nil {
		return Preview{}, err
	}
	client, _, err := s.client(ctx)
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return preview, nil
		}
		return Preview{}, err
	}
	result, err := client.ApplySource(ctx, deckID, preview.Source, true)
	if err != nil {
		return Preview{}, err
	}
	preview.SlideCount = result.SlideCount
	if result.Warnings != nil {
		preview.Warnings = result.Warnings
	}
	preview.Checked = true
	return preview, nil
}

// Create makes the deck.
//
// The link is written before the deck is compiled and updated afterwards,
// whichever way it goes. Recording only on success would leave a deck in Ptium
// that umm has no record of — invisible in umm and unexplained in Ptium — and
// a failure nobody can look up afterwards.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req Request) (Result, error) {
	story, source, err := s.compile(ctx, userID, req)
	if err != nil {
		return Result{}, err
	}
	client, cfg, err := s.client(ctx)
	if err != nil {
		return Result{}, err
	}

	deck, err := client.CreateDeck(ctx, story.Title, cfg.TemplateID, cfg.Language)
	if err != nil {
		return Result{}, err
	}
	link, err := s.Links.CreatePresentationLink(ctx, userID, req.SpaceID, deck.ID, story.Title, req.recorded())
	if err != nil {
		// The deck exists in Ptium and umm could not record it. Said plainly
		// rather than swallowed, because the two are now out of step and only a
		// person can decide what to do about it.
		return Result{}, fmt.Errorf("%w: deck %s: %v", ErrDeckNotRecorded, deck.ID, err)
	}

	result, applyErr := client.ApplySource(ctx, deck.ID, source, false)
	if applyErr != nil {
		// Both: the kind decides the sentence a person reads and whether
		// retrying can help, and the message is what whoever fixes it needs.
		classified := Classify(applyErr)
		if err := s.Links.CompletePresentationLink(ctx, userID, link.ID, store.PresentationFailed, source, nil,
			len(story.usedThoughts()), len(story.Excluded), len(story.Trimmed), applyErr.Error(), string(classified.Kind)); err != nil {
			return Result{}, fmt.Errorf("%w (and recording the failure also failed: %v)", applyErr, err)
		}
		// The deck is open in Ptium and empty. Saying so is what stops the next
		// press from making another one.
		return Result{}, &DeckLeftBehind{DeckID: deck.ID, Err: applyErr}
	}

	if err := s.Links.CompletePresentationLink(ctx, userID, link.ID, store.PresentationReady, source,
		story.SlideSources(), len(story.usedThoughts()), len(story.Excluded), len(story.Trimmed), "", ""); err != nil {
		return Result{}, err
	}
	link.Status = store.PresentationReady
	link.CompiledSource = source
	link.ThoughtCount = len(story.usedThoughts())
	link.ExcludedCount = len(story.Excluded)
	link.TrimmedCount = len(story.Trimmed)

	warnings := result.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return Result{Link: link, Warnings: warnings}, nil
}

// compile reads the space and turns it into a talk.
func (s *Service) compile(ctx context.Context, userID uuid.UUID, req Request) (Storyline, string, error) {
	// ListNotes carries the permission check, so a space the caller cannot read
	// comes back empty rather than compiled.
	notes, edges, err := s.Spaces.ListNotes(ctx, userID, req.SpaceID, "")
	if err != nil {
		return Storyline{}, "", err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = s.spaceName(ctx, userID, req.SpaceID)
	}

	thoughts := make([]Thought, 0, len(notes))
	for _, note := range notes {
		thoughts = append(thoughts, Thought{
			ID: note.ID, Title: note.Title, Content: note.Content, Kind: note.Kind,
			X: note.X, Y: note.Y, Width: note.Width, Height: note.Height, AIExcluded: note.AIExcluded,
		})
	}
	links := make([]Link, 0, len(edges))
	for _, edge := range edges {
		links = append(links, Link{From: edge.SourceID, To: edge.TargetID, Relation: edge.Relation,
			Origin: edge.Origin, Reason: edge.Reason})
	}

	story := Compile(thoughts, links, Options{Title: title, Only: req.Only,
		IncludeExcluded: req.IncludeExcluded, OneSlidePerThought: req.OneSlidePerThought,
		MaxSlides: req.MaxSlides})
	// Only the headings, only the groups, and only when asked. The slides
	// themselves are already final at this point — a thought reaches its slide
	// unchanged whether or not a model is involved.
	if req.NameGroups || req.SectionDeck {
		var cfg Config
		_ = s.Settings.GetSetting(ctx, "ptium", &cfg)
		if req.NameGroups {
			story.NamedHeadings = nameGroups(ctx, s.Namer, &story, cfg.Language)
		}
		// After naming: the headings a part is proposed over should be the ones
		// that will actually be on the slides, or the model is dividing a talk
		// it cannot see.
		if req.SectionDeck {
			story.Sections = sectionDeck(ctx, s.Sectioner, &story, cfg.Language)
			// Part headings are slides too. A person who asked for twenty and
			// was handed twenty-four did not get the length they asked for, so
			// the cap is applied again over the finished deck. A part whose
			// slides all fall out of the length loses its heading with them,
			// which means dividing a long talk into parts costs content slides
			// — that is what asking for a length means.
			if fit(&story, req.MaxSlides) > 0 {
				// And then the deck says what is still in it. "AI가 3개 부로
				// 나눴습니다" over a deck holding two headings, or "묶음 제목
				// 15개를 AI가 지었습니다" over twelve, is the same deck saying
				// two different things — and these two counts are the whole of
				// what tells a person how much of the shape is the model's, so
				// the numbers they cannot check themselves are the ones that
				// have to be right. Recounted from the slides rather than
				// adjusted by the cut, because the cut does not know which of
				// the slides it dropped a model had touched.
				story.Sections = story.parts()
				story.NamedHeadings = story.named()
			}
		}
	}
	if len(story.Slides) == 0 {
		return Storyline{}, "", ErrNothingToPresent
	}
	// Trace is always on. A deck whose slides cannot say where they came from
	// is the thing this whole feature exists to avoid.
	return story, WriteSource(story, SourceOptions{Trace: true}), nil
}

func (s *Service) spaceName(ctx context.Context, userID, spaceID uuid.UUID) string {
	spaces, err := s.Spaces.ListSpaces(ctx, userID)
	if err != nil {
		return ""
	}
	for _, space := range spaces {
		if space.ID == spaceID {
			return space.Name
		}
	}
	return ""
}

// client builds a Ptium client from the current settings.
func (s *Service) client(ctx context.Context) (Ptium, Config, error) {
	var cfg Config
	if err := s.Settings.GetSetting(ctx, "ptium", &cfg); err != nil {
		return nil, cfg, err
	}
	if cfg.BaseURL == "" {
		return nil, cfg, ErrNotConfigured
	}
	key, err := s.credential(cfg.APIKey)
	if err != nil {
		return nil, cfg, err
	}

	build := s.NewPtium
	if build == nil {
		build = func(c Config) (Ptium, error) {
			return NewClient(c.BaseURL, c.APIKey, time.Duration(c.TimeoutSeconds)*time.Second)
		}
	}
	// The decrypted key is handed to the builder, never written back to cfg,
	// so a plaintext credential does not travel any further than the client
	// that needs it.
	withKey := cfg
	withKey.APIKey = key
	client, err := build(withKey)
	if err != nil {
		return nil, cfg, err
	}
	return client, cfg, nil
}

// credential unwraps a stored secret, leaving a plain one alone.
func (s *Service) credential(stored string) (string, error) {
	if !strings.HasPrefix(stored, "enc:") {
		return stored, nil
	}
	if s.Cipher == nil {
		return "", errors.New("ptium credential is encrypted but no key is available to read it")
	}
	return s.Cipher.Decrypt(strings.TrimPrefix(stored, "enc:"))
}

// SlideSources maps each slide's position to the thoughts on it.
//
// Positions are one-based and count only the slides umm wrote, matching what
// Ptium compiles from this source.
func (s Storyline) SlideSources() []store.SlideSource {
	var out []store.SlideSource
	position := 0
	if s.Title != "" {
		position++ // the cover, which quotes nothing
	}
	for _, slide := range s.Slides {
		position++
		for _, id := range slide.From {
			out = append(out, store.SlideSource{SlidePosition: position, NoteID: id})
		}
	}
	return out
}

// parts is how many part headings are in the deck as it stands.
func (s Storyline) parts() int {
	count := 0
	for _, slide := range s.Slides {
		if slide.Sectioned {
			count++
		}
	}
	return count
}

// named is how many headings in the deck as it stands were proposed by a model
// rather than derived from the person's own words.
func (s Storyline) named() int {
	count := 0
	for _, slide := range s.Slides {
		if slide.Named {
			count++
		}
	}
	return count
}

// usedThoughts is every thought that reached a slide, counted once.
func (s Storyline) usedThoughts() map[uuid.UUID]bool {
	seen := map[uuid.UUID]bool{}
	for _, slide := range s.Slides {
		for _, id := range slide.From {
			seen[id] = true
		}
	}
	return seen
}

// Retry compiles the space again into the deck a failed attempt already made.
//
// Making a deck is two calls: Ptium opens one, then umm compiles source into
// it. When the second fails — a space large enough to run past the timeout is
// the usual way — the deck exists in Ptium and umm records the attempt as
// failed. Pressing the button again made another deck, so a space that failed
// four times left four empty ones behind and nothing said so.
//
// This applies the source to the deck that is already there. Ptium's source
// endpoint replaces rather than appends, so a partly-compiled deck ends up the
// same as one compiled once, and running this twice is the same as running it
// once.
func (s *Service) Retry(ctx context.Context, userID uuid.UUID, linkID uuid.UUID) (Result, error) {
	link, err := s.Links.FailedPresentationLink(ctx, userID, linkID)
	if err != nil {
		return Result{}, err
	}
	// Recompiled rather than replayed from the stored source: the thinking has
	// probably moved on since it failed, and a retry that resurrects an old
	// deck source would put stale sentences on the slides.
	//
	// Recompiled from what was asked for, though, not from the defaults. A talk
	// built from six selected thoughts and capped at twenty slides used to come
	// back as the whole space, uncapped, into the same deck and without a word
	// about it — the space had moved on, which is the point, but the request had
	// not.
	story, source, err := s.compile(ctx, userID, asRequest(link))
	if err != nil {
		return Result{}, err
	}
	client, _, err := s.client(ctx)
	if err != nil {
		return Result{}, err
	}

	result, applyErr := client.ApplySource(ctx, link.PtiumID, source, false)
	if applyErr != nil {
		classified := Classify(applyErr)
		if err := s.Links.CompletePresentationLink(ctx, userID, link.ID, store.PresentationFailed, source, nil,
			len(story.usedThoughts()), len(story.Excluded), len(story.Trimmed), applyErr.Error(), string(classified.Kind)); err != nil {
			return Result{}, fmt.Errorf("%w (and recording the failure also failed: %v)", applyErr, err)
		}
		return Result{}, applyErr
	}

	if err := s.Links.CompletePresentationLink(ctx, userID, link.ID, store.PresentationReady, source,
		story.SlideSources(), len(story.usedThoughts()), len(story.Excluded), len(story.Trimmed), "", ""); err != nil {
		return Result{}, err
	}
	link.Status = store.PresentationReady
	link.Error, link.FailureKind = "", ""
	link.CompiledSource = source
	link.ThoughtCount = len(story.usedThoughts())
	link.ExcludedCount = len(story.Excluded)
	link.TrimmedCount = len(story.Trimmed)

	warnings := result.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return Result{Link: link, Warnings: warnings}, nil
}
