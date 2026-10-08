package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/presentation"
	"github.com/hkjang/umm/internal/store"
	"github.com/jackc/pgx/v5"
)

// Turning a space into a talk.
//
// Three things a person does: see what their space would become, make the
// deck, and later ask where a slide's sentences came from. The first must
// change nothing, which is why it is a GET and reaches neither Ptium's
// database nor umm's.

// presentations returns the service, built from the store each time so the
// Ptium address and credential are read fresh rather than cached past a change.
func (s *Server) presentations(r *http.Request) *presentation.Service {
	svc := &presentation.Service{Spaces: s.Store, Links: s.Store, Settings: s.Store, Cipher: s.Cipher}
	// The namer is what proposes a heading for a group of thoughts nobody
	// connected. Absent when there is no Dream service to reach a chat model
	// through, which is the same as having no chat model configured: the deck
	// compiles with the headings umm derived itself.
	if s.Dreams != nil {
		svc.Namer = presentation.GatewayNamer{AI: s.Dreams, UserID: principal(r).User.ID}
		svc.Sectioner = presentation.GatewaySectioner{AI: s.Dreams, UserID: principal(r).User.ID}
	}
	return svc
}

// presentationRequest is what a caller may ask for.
type presentationRequest struct {
	Title string `json:"title"`
	// OneSlidePerThought turns off grouping thoughts by where they sit, for a
	// space whose arrangement means nothing.
	OneSlidePerThought bool `json:"oneSlidePerThought"`
	// NameGroups asks the chat model to name each group. Opt-in: it sends those
	// thoughts to the gateway, and it makes the deck stop being the same every
	// time. The sentences on the slides are the person's either way.
	NameGroups bool `json:"nameGroups"`
	// SectionDeck asks where a long talk divides into parts. It adds heading
	// slides and moves nothing.
	SectionDeck bool `json:"sectionDeck"`
	// MaxSlides caps how long the talk may be. Zero, the default, means no cap:
	// a deck is a whole space until somebody says how long they have.
	MaxSlides int `json:"maxSlides"`
	// NoteIDs restricts the deck to a selection, so a cluster or a few chosen
	// thoughts can become a talk without the rest of the space.
	NoteIDs []uuid.UUID `json:"noteIds"`
	// IncludeExcluded overrides the note-level "keep this out of analysis" mark.
	// It has to be asked for: a thought held back is being held back from having
	// things done to it, and putting it on a slide is one of those things.
	IncludeExcluded bool `json:"includeExcluded"`
	// DeckID, when given, has Ptium check the compiled source against a deck
	// that already exists. Nothing is written either way.
	DeckID string `json:"deckId"`
}

// maxSlidesParam reads a length cap off the query string.
//
// Anything that is not a number is no cap rather than an error: a preview is
// the cheapest thing in this feature, and refusing to draw one over a typo in
// an optional parameter costs more than ignoring it.
func maxSlidesParam(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return clampMaxSlides(n)
}

// clampMaxSlides keeps a cap inside what a talk can be.
//
// Negative and zero both mean no cap. The ceiling is not a limit on decks —
// an uncapped deck of any size still compiles — it only stops a cap larger
// than any real talk from being stored as though somebody chose it.
func clampMaxSlides(n int) int {
	if n <= 0 {
		return 0
	}
	if n > maxSlidesCeiling {
		return maxSlidesCeiling
	}
	return n
}

// maxSlidesCeiling is well past the longest talk anyone gives; past this a cap
// is not shortening anything.
const maxSlidesCeiling = 500

func (s *Server) previewPresentation(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	req := presentation.Request{
		SpaceID:            spaceID,
		Title:              r.URL.Query().Get("title"),
		IncludeExcluded:    r.URL.Query().Get("includeExcluded") == "true",
		OneSlidePerThought: r.URL.Query().Get("oneSlidePerThought") == "true",
		NameGroups:         r.URL.Query().Get("nameGroups") == "true",
		SectionDeck:        r.URL.Query().Get("sectionDeck") == "true",
		MaxSlides:          maxSlidesParam(r.URL.Query().Get("maxSlides")),
	}

	svc := s.presentations(r)
	deckID := r.URL.Query().Get("deckId")
	preview, err := presentation.Preview{}, error(nil)
	if deckID != "" {
		preview, err = svc.PreviewAgainst(r.Context(), principal(r).User.ID, req, deckID)
	} else {
		preview, err = svc.Preview(r.Context(), principal(r).User.ID, req)
	}
	if err != nil {
		writePresentationError(w, r, err, "발표 구성을 만들지 못했습니다.")
		return
	}
	writeJSON(w, 200, preview)
}

func (s *Server) createPresentation(w http.ResponseWriter, r *http.Request) {
	// notes:write rather than notes:read: this sends the person's thoughts to
	// another service and creates something there. A key that may only read
	// must not be able to do that.
	if !requireScope(w, r, "notes:write") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	var body presentationRequest
	if r.ContentLength > 0 {
		if decodeJSON(w, r, &body) != nil {
			return
		}
	}

	result, err := s.presentations(r).Create(r.Context(), principal(r).User.ID, presentation.Request{
		SpaceID:            spaceID,
		Title:              body.Title,
		Only:               body.NoteIDs,
		IncludeExcluded:    body.IncludeExcluded,
		OneSlidePerThought: body.OneSlidePerThought,
		NameGroups:         body.NameGroups,
		SectionDeck:        body.SectionDeck,
		MaxSlides:          clampMaxSlides(body.MaxSlides),
	})
	if err != nil {
		writePresentationError(w, r, err, "발표 자료를 만들지 못했습니다.")
		return
	}
	writeJSON(w, 201, result)
}

func (s *Server) listPresentations(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	links, err := s.Store.ListPresentationLinks(r.Context(), principal(r).User.ID, spaceID)
	if err != nil {
		writeError(w, 500, "발표 자료 목록을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"presentations": s.decks(r, links)})
}

// deckLink is a stored link plus where the deck can be opened.
type deckLink struct {
	store.PresentationLink
	// URL is computed on every read and never stored. An administrator who
	// moves Ptium would otherwise leave every past link pointing at an address
	// that no longer answers, with no way to tell which ones were stale.
	URL string `json:"url,omitempty"`
	// StaleSlides is how many of this deck's slides quote a thought that has
	// since been rewritten or deleted. A deck made from someone's thinking
	// stops being true when the thinking moves on, and only umm is in a
	// position to notice.
	StaleSlides int `json:"staleSlides,omitempty"`
}

// decks turns stored links into what a client reads: where to open each deck,
// and how much of it has gone stale.
//
// One function rather than each handler assembling its own, because they did
// not agree: the list of a space's talks reported a deck as having a changed
// slide while the same deck, reached from the note that changed, reported
// none. The note view is where it matters most — you have just rewritten the
// thought — and it was the one saying nothing was wrong.
func (s *Server) decks(r *http.Request, links []store.PresentationLink) []deckLink {
	rows := s.withDeckURLs(r, links)

	// Staleness is counted per space, so ask once per space rather than once
	// per deck. In practice that is a single query.
	spaces := map[uuid.UUID]bool{}
	for _, link := range links {
		spaces[link.SpaceID] = true
	}
	counts := map[uuid.UUID]int{}
	for spaceID := range spaces {
		found, err := s.Store.StaleCounts(r.Context(), principal(r).User.ID, spaceID)
		if err != nil {
			// A count that cannot be read is left absent rather than reported
			// as zero: "nothing has changed" is a claim, and this is not in a
			// position to make it.
			return rows
		}
		for id, count := range found {
			counts[id] = count
		}
	}
	for i := range rows {
		rows[i].StaleSlides = counts[rows[i].ID]
	}
	return rows
}

func (s *Server) withDeckURLs(r *http.Request, links []store.PresentationLink) []deckLink {
	var cfg presentation.Config
	// A missing or unreadable setting is not an error here: the list is still
	// worth showing, just without somewhere to click through to.
	_ = s.Store.GetSetting(r.Context(), "ptium", &cfg)
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")

	out := make([]deckLink, 0, len(links))
	for _, link := range links {
		row := deckLink{PresentationLink: link}
		if base != "" && link.PtiumID != "" {
			row.URL = base + "/presentations/" + url.PathEscape(link.PtiumID) + "/editor"
		}
		out = append(out, row)
	}
	return out
}

// presentationSources answers "where did this slide's sentences come from".
//
// The question that matters most about a deck made this way, because the
// slides carry the person's own words: being able to get back to the note is
// being able to check that nothing was put in their mouth.
// retryPresentation compiles the space again into the deck a failed attempt
// already made, rather than making another one.
//
// notes:write, like creating: this changes a deck in Ptium.
func (s *Server) retryPresentation(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	linkID, ok := parseID(w, r, "linkID")
	if !ok {
		return
	}
	result, err := s.presentations(r).Retry(r.Context(), principal(r).User.ID, linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, 404, "다시 시도할 수 있는 실패한 발표 자료가 없습니다.")
			return
		}
		writePresentationError(w, r, err, "발표 자료를 다시 만들지 못했습니다.")
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) presentationSources(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	linkID, ok := parseID(w, r, "linkID")
	if !ok {
		return
	}
	userID := principal(r).User.ID
	sources, err := s.Store.PresentationSources(r.Context(), userID, linkID)
	if err != nil {
		writeError(w, 500, "슬라이드 출처를 불러오지 못했습니다.")
		return
	}
	source, err := s.Store.PresentationLinkSource(r.Context(), userID, linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, 404, "발표 자료를 찾을 수 없습니다.")
			return
		}
		writeError(w, 500, "슬라이드 출처를 불러오지 못했습니다.")
		return
	}
	// What has changed since, alongside where each slide came from: the two
	// answer one question between them — is this deck still what I think.
	stale, err := s.Store.StaleSlides(r.Context(), userID, linkID)
	if err != nil {
		writeError(w, 500, "슬라이드 출처를 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"sources": sources, "source": source, "stale": stale})
}

// notePresentations answers the other direction: which talks quote this
// thought. Someone about to edit a note that two decks quote is making a
// different decision from someone editing one nobody has used.
func (s *Server) notePresentations(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	noteID, ok := parseID(w, r, "noteID")
	if !ok {
		return
	}
	links, err := s.Store.PresentationsUsingNote(r.Context(), principal(r).User.ID, noteID)
	if err != nil {
		writeError(w, 500, "이 생각이 쓰인 발표 자료를 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"presentations": s.decks(r, links)})
}

// writePresentationError turns the service's errors into something the reader
// can act on.
//
// Before this, every failure that was not one of the three named cases came
// back as one 502 carrying the Go error that produced it:
//
//	발표 자료를 만들지 못했습니다. ptium is unreachable: Post "http://ptium.internal:8080/…": dial tcp: connection refused
//	발표 자료를 만들지 못했습니다. ptium status 401: api key is invalid or expired
//	발표 자료를 만들지 못했습니다. … json: cannot unmarshal array into Go value of type presentation.deckEnvelope
//
// All three say what happened. None says what to do, they are all the same
// shape so the screen cannot tell them apart, and two of them show a person who
// wanted slides an internal address and a Go type name.
//
// So each kind gets its own problem type, its own sentence about who fixes it,
// and — only where Ptium said something worth repeating — Ptium's own words.
// The Go error is kept as `technical`, which the screen shows to administrators
// and nobody else.
func writePresentationError(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	switch {
	case errors.Is(err, presentation.ErrNotConfigured):
		writeProblem(w, r, 409, "ptium-not-configured", "Ptium이 연결되어 있지 않습니다",
			"Ptium이 연결되어 있지 않습니다. 서비스 관리자에서 주소를 설정해 주세요.", nil)
		return
	case errors.Is(err, presentation.ErrNothingToPresent):
		writeProblem(w, r, 400, "presentation-nothing-to-present", "발표로 만들 생각이 없습니다",
			"발표로 만들 생각이 없습니다.", nil)
		return
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, 404, "공간을 찾을 수 없습니다.")
		return
	}

	failure := presentation.Classify(err)
	status, title, detail := presentationFailureMessage(failure, fallback)
	extra := map[string]any{"failure": string(failure.Kind)}
	if failure.Status != 0 {
		extra["ptiumStatus"] = failure.Status
	}
	// Ptium's own words, kept separate from umm's sentence so the screen can
	// show them as what they are — another service talking — rather than
	// pasting them into the middle of a Korean sentence.
	if failure.Detail != "" {
		extra["ptiumDetail"] = failure.Detail
	}
	// The deck Ptium opened before the compile failed. It is really there and
	// empty, and saying which one it is turns "it broke" into something to
	// finish: the screen offers the deck and a retry into it rather than a
	// button that quietly makes a second one.
	if deck := presentation.LeftBehindDeck(err); deck != "" {
		extra["ptiumId"] = deck
		extra["deckLeftBehind"] = true
	}
	// The Go error names internal hosts, Go types and SQL constraints. An
	// administrator is the one who can use that; for anyone else it is noise
	// they cannot act on and something they should not be shown.
	if failure.Technical != "" && principal(r).User.Role == "admin" {
		extra["technical"] = failure.Technical
	}
	writeProblem(w, r, status, "ptium-"+string(failure.Kind), title, detail, extra)
}

// presentationFailureMessage says who fixes each kind and how.
//
// Cut by who can act, not by status code: a rejected deck is the author's to
// fix and Ptium already named the slide, a refused credential is the
// administrator's, and a service that is down is nobody's until it is back.
func presentationFailureMessage(failure presentation.Failure, fallback string) (int, string, string) {
	switch failure.Kind {
	case presentation.FailureUnreachable:
		return 502, "Ptium에 연결하지 못했습니다",
			"Ptium 서버에 연결하지 못했습니다. 서버가 내려가 있거나 주소가 닿지 않습니다. 잠시 뒤 다시 시도해 보시고, 계속되면 서비스 관리자에게 알려 주세요."
	case presentation.FailureTimedOut:
		return 504, "Ptium이 제때 답하지 않았습니다",
			"Ptium이 정해진 시간 안에 답하지 않았습니다. 생각이 많은 공간은 오래 걸릴 수 있습니다. 다시 시도해 보시고, 계속되면 관리자에게 제한 시간을 늘려 달라고 요청하세요."
	case presentation.FailureUnauthorized:
		return 502, "Ptium이 umm의 자격 증명을 거부했습니다",
			"Ptium이 umm의 API 키를 받아들이지 않았습니다. 키가 만료됐거나 잘못 설정된 것이며, 여기서 고칠 수 있는 것이 아닙니다. 서비스 관리자에게 Ptium 키를 다시 설정해 달라고 알려 주세요."
	case presentation.FailureNoAPI:
		return 502, "그 주소에 Ptium API가 없습니다",
			"설정된 주소가 응답은 했지만 Ptium API가 아닙니다. 주소가 잘못됐거나 앞단의 프록시가 가로채고 있습니다. 서비스 관리자에게 Ptium 주소를 확인해 달라고 알려 주세요."
	case presentation.FailureRejected:
		return 422, "Ptium이 이 발표 구성을 받아들이지 않았습니다",
			"Ptium이 이 발표 구성을 슬라이드로 만들 수 없다고 답했습니다. 아래 Ptium의 설명이 어느 슬라이드가 문제인지 알려 줍니다. 생각을 줄이거나 나눈 뒤 다시 시도해 보세요."
	case presentation.FailureRemote:
		return 502, "Ptium 쪽에서 오류가 났습니다",
			"Ptium이 요청을 받았지만 처리하는 중에 오류가 났습니다. umm 설정 문제가 아니므로 잠시 뒤 다시 시도해 보시고, 계속되면 Ptium 쪽 로그를 관리자에게 확인 요청하세요."
	case presentation.FailureUnexpected:
		return 502, "Ptium이 예상과 다른 응답을 보냈습니다",
			"주소와 키는 맞지만 Ptium이 umm이 아는 것과 다른 형식으로 답했습니다. 두 서비스의 버전이 맞지 않을 때 생깁니다. 서비스 관리자에게 Ptium 버전을 확인해 달라고 알려 주세요."
	case presentation.FailureNotRecorded:
		return 500, "Ptium에는 만들어졌지만 umm이 기록하지 못했습니다",
			"덱은 Ptium에 실제로 만들어졌고 umm이 그 사실을 저장하지 못했습니다. 그대로 다시 시도하면 덱이 하나 더 생깁니다. Ptium에서 방금 만들어진 덱을 먼저 확인해 주세요."
	default:
		return 502, "발표 자료를 만들지 못했습니다", fallback
	}
}
