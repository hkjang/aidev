package presentation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hkjang/umm/internal/textutil"
)

/*
Talking to Ptium.

Two calls make a deck, and the split matters. Creating one is `POST
/api/v1/presentations`, which returns a draft with no slides; the slides come
from `PUT /api/v1/presentations/{id}/source`, which compiles the deck source
this package wrote. Nothing here posts a `prompt`, so nothing here asks Ptium
to write anything — Ptium lays out and renders what umm already decided.

Ptium adjusts source it cannot honour exactly rather than rejecting it — a
layout the template lacks, text with nowhere to go — and names each adjustment
in `warnings`. Those are carried back rather than dropped: a deck that quietly
lost a bullet is worse than one that says it did.
*/

// MaxResponseBody caps what is read from Ptium. A deck's source can be long,
// but nothing this client asks for returns megabytes, and an unbounded read
// from another service is how one bad response takes umm down with it.
const MaxResponseBody = 4 << 20

// DefaultTimeout is what a call gets when the caller sets none. Compiling a
// deck is slower than an ordinary request and faster than generation, which
// umm never waits on.
const DefaultTimeout = 30 * time.Second

// Client calls one Ptium installation.
type Client struct {
	BaseURL string
	// APIKey is a `ptium_*` key or an OIDC access token. Sent as a bearer
	// credential, which Ptium accepts for both.
	APIKey string
	HTTP   *http.Client
}

// ErrNotConfigured is returned when no Ptium is set up, so a caller can tell
// "the administrator has not connected one" from "the call failed".
var ErrNotConfigured = errors.New("ptium is not configured")

// NewClient builds a client, or reports that there is nothing to talk to.
func NewClient(baseURL, apiKey string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, ErrNotConfigured
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("ptium base url is not a url: %q", baseURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("ptium base url must be http or https, got %q", parsed.Scheme)
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Client{BaseURL: baseURL, APIKey: strings.TrimSpace(apiKey), HTTP: &http.Client{Timeout: timeout}}, nil
}

// Deck is the part of a Ptium presentation umm keeps a handle on. Everything
// else about it stays in Ptium, where it can be edited without umm's copy
// going stale.
type Deck struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type deckEnvelope struct {
	Data Deck `json:"data"`
}

// CreateDeck opens an empty deck to compile into.
//
// Deliberately without a prompt: this asks Ptium for somewhere to put slides,
// not for slides. Passing the person's thoughts here instead would hand them to
// a model and get paraphrases back.
func (c *Client) CreateDeck(ctx context.Context, title, templateID, language string) (Deck, error) {
	body := map[string]any{"title": fallbackTitle(title)}
	if templateID = strings.TrimSpace(templateID); templateID != "" {
		body["templateId"] = templateID
	}
	if language = strings.TrimSpace(language); language != "" {
		body["language"] = language
	}

	var envelope deckEnvelope
	if err := c.do(ctx, http.MethodPost, "/api/v1/presentations", body, &envelope); err != nil {
		return Deck{}, err
	}
	if strings.TrimSpace(envelope.Data.ID) == "" {
		return Deck{}, errors.New("ptium returned a deck with no id")
	}
	return envelope.Data, nil
}

// ApplyResult is what compiling source produced.
type ApplyResult struct {
	SlideCount int      `json:"slideCount"`
	Warnings   []string `json:"warnings"`
}

type applyEnvelope struct {
	Data ApplyResult `json:"data"`
}

// ApplySource compiles deck source into slides.
//
// With dryRun the deck is untouched and only the result comes back, which is
// what makes a storyline preview honest: the slide count and the warnings are
// Ptium's, measured against the real template, not umm's guess at them.
func (c *Client) ApplySource(ctx context.Context, deckID, source string, dryRun bool) (ApplyResult, error) {
	if strings.TrimSpace(deckID) == "" {
		return ApplyResult{}, errors.New("no deck id")
	}
	if strings.TrimSpace(source) == "" {
		return ApplyResult{}, errors.New("refusing to compile an empty deck")
	}
	body := map[string]any{"source": source, "dryRun": dryRun}

	var envelope applyEnvelope
	path := "/api/v1/presentations/" + url.PathEscape(deckID) + "/source"
	if err := c.do(ctx, http.MethodPut, path, body, &envelope); err != nil {
		return ApplyResult{}, err
	}
	return envelope.Data, nil
}

// StatusError is a reply Ptium actually sent. Its presence means the request
// reached Ptium and came back — which is the difference between a connection
// that does not work and one that works but answered something unexpected.
type StatusError struct {
	Status int
	Detail string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("ptium status %d: %s", e.Status, e.Detail)
}

// ShapeError is a reply that arrived and could not be read as documented.
//
// Ptium answered, so the address and the key are right; only the body was not
// what this version expects. Reported apart from a failure to connect, because
// telling someone their connection is broken when it is not sends them looking
// in the wrong place — and the templates envelope has already changed shape
// once between versions.
type ShapeError struct {
	Err error
}

func (e *ShapeError) Error() string {
	return "ptium returned something that is not the documented response: " + e.Err.Error()
}

func (e *ShapeError) Unwrap() error { return e.Err }

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	// A read carries no body. Sending "null" with a JSON content type on a GET
	// is the kind of thing a strict server or a proxy in front of one rejects,
	// and there is nothing to send.
	var reader io.Reader
	hasBody := body != nil
	if hasBody {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if hasBody {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Accept", "application/json")
	if c.APIKey != "" {
		// Bearer rather than X-API-Key: Ptium accepts either for a ptium_* key
		// and only Bearer for an OIDC token, so one header covers both and there
		// is no chance of sending the two together, which Ptium rejects.
		request.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("ptium is unreachable: %w", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, MaxResponseBody+1))
	if err != nil {
		return err
	}
	if len(payload) > MaxResponseBody {
		return errors.New("ptium response is too large")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return &StatusError{Status: response.StatusCode, Detail: problemDetail(payload)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return &ShapeError{Err: err}
	}
	return nil
}

// problemDetail pulls the readable part out of an error body.
//
// Ptium answers failures with RFC 9457 problem details, and the whole document
// in a log line buries the one sentence that says what went wrong. A body that
// is not a problem document is passed through, trimmed, rather than replaced
// with a message that hides it.
func problemDetail(payload []byte) string {
	var problem struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(payload, &problem) == nil {
		switch {
		case strings.TrimSpace(problem.Detail) != "":
			return textutil.LimitUTF8Bytes(problem.Detail, 300)
		case strings.TrimSpace(problem.Title) != "":
			return textutil.LimitUTF8Bytes(problem.Title, 300)
		}
	}
	return textutil.LimitUTF8Bytes(strings.TrimSpace(string(payload)), 300)
}

// fallbackTitle keeps Ptium's 200-character limit from turning a long space
// name into a rejected request, and never sends an empty one.
func fallbackTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "umm"
	}
	return textutil.LimitUTF8Bytes(title, 200)
}

// Template is one of the designs a deck can be generated into.
type Template struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind,omitempty"`
}

// data is the array itself, not an object wrapping one. Checked against a
// running Ptium rather than guessed: the guess parsed cleanly against a stub
// and failed on the first real connection test, with an unmarshal error an
// administrator could do nothing with.
type templateEnvelope struct {
	Data []Template `json:"data"`
}

// Templates lists the designs this Ptium offers.
//
// Also what the connection test calls. It is an authenticated read, so a reply
// proves three things at once — the address answers, it is a Ptium, and the
// credential is accepted — which is what an administrator actually wants to
// know before someone tries to make a deck and gets a 401 they cannot read.
func (c *Client) Templates(ctx context.Context) ([]Template, error) {
	var envelope templateEnvelope
	if err := c.do(ctx, http.MethodGet, "/api/v1/templates", nil, &envelope); err != nil {
		return nil, err
	}
	return envelope.Data, nil
}
