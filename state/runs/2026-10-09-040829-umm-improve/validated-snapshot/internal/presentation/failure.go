package presentation

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
)

/*
Why making a deck failed, in terms of what to do about it.

Every one of these used to reach a person as the Go error that produced it:

	ptium is unreachable: Post "http://ptium.internal:8080/api/v1/presentations": dial tcp: connection refused
	ptium status 401: api key is invalid or expired
	ptium returned ...: json: cannot unmarshal array into Go value of type presentation.deckEnvelope

Each says what happened and none says what to do, and two of them say things a
person who just wanted slides should not be shown at all: the address of an
internal service, and the name of a Go type.

The distinction that matters to the reader is not the status code. It is who
can fix it. A wrong API key is the administrator's; a layout the template lacks
is the author's, and Ptium already said which slide; a service that is down is
nobody's until it comes back. So the kinds below are cut that way.
*/

// FailureKind is what sort of problem this is, from the point of view of
// whoever has to do something about it.
type FailureKind string

const (
	// FailureUnreachable: nothing answered. Nobody can fix this from inside umm.
	FailureUnreachable FailureKind = "unreachable"
	// FailureUnauthorized: Ptium answered and refused umm's credential.
	FailureUnauthorized FailureKind = "unauthorized"
	// FailureNoAPI: something answered but it is not the Ptium API — usually the
	// address points at a web page, a proxy, or the wrong path.
	FailureNoAPI FailureKind = "no-api"
	// FailureRejected: Ptium understood the request and would not do it. This is
	// the one where Ptium's own words are worth repeating verbatim, because they
	// name the slide and the reason.
	FailureRejected FailureKind = "rejected"
	// FailureRemote: Ptium broke on its own side.
	FailureRemote FailureKind = "remote-error"
	// FailureUnexpected: Ptium answered something this version cannot read.
	FailureUnexpected FailureKind = "unexpected-response"
	// FailureNotRecorded: the deck exists in Ptium and umm failed to write it
	// down. The two are now out of step, which is why it is not folded into the
	// others — the deck is really there and retrying makes a second one.
	FailureNotRecorded FailureKind = "not-recorded"
	// FailureTimedOut: umm gave up waiting.
	FailureTimedOut FailureKind = "timed-out"
	// FailureOther: anything not recognised.
	FailureOther FailureKind = "other"
)

// ErrDeckNotRecorded marks the case where Ptium made a deck and umm could not
// store the link. Typed rather than a formatted string so the API layer can
// tell it apart without matching on wording.
var ErrDeckNotRecorded = errors.New("ptium made a deck that umm could not record")

// DeckLeftBehind marks a failure that happened after Ptium had already opened
// the deck. The deck is really there and empty, so the answer is not "try
// again" — trying again used to make a second one.
type DeckLeftBehind struct {
	DeckID string
	Err    error
}

func (e *DeckLeftBehind) Error() string {
	return "ptium deck " + e.DeckID + " was created and left empty: " + e.Err.Error()
}

func (e *DeckLeftBehind) Unwrap() error { return e.Err }

// LeftBehindDeck returns the deck a failure left in Ptium, if it left one.
func LeftBehindDeck(err error) string {
	var left *DeckLeftBehind
	if errors.As(err, &left) {
		return left.DeckID
	}
	return ""
}

// Failure is a classified error: what kind it is, and the part of it that is
// safe and useful to show the person who asked for the deck.
type Failure struct {
	Kind FailureKind
	// Detail is Ptium's own explanation when it gave one, already trimmed by
	// problemDetail. Empty when the only explanation is a Go error, because a Go
	// error is not an explanation.
	Detail string
	// Status is Ptium's HTTP status when it answered, or 0.
	Status int
	// Technical is the underlying error text. It names internal hosts, Go types
	// and SQL constraints, so it is for operators and logs — never for someone
	// who just wanted slides.
	Technical string
}

// Classify sorts an error from this package into something answerable.
func Classify(err error) Failure {
	if err == nil {
		return Failure{Kind: FailureOther}
	}
	failure := Failure{Kind: FailureOther, Technical: err.Error()}

	var status *StatusError
	if errors.As(err, &status) {
		failure.Status, failure.Detail = status.Status, showableDetail(status.Detail)
		switch {
		case status.Status == 401 || status.Status == 403:
			failure.Kind = FailureUnauthorized
			// Ptium's wording here is about umm's credential, not about anything
			// the reader did, and repeating it invites them to go looking for a
			// key they do not have.
			failure.Detail = ""
		case status.Status == 404:
			// Not "this deck is missing": every call this package makes to a
			// correct Ptium exists, so a 404 means the address is wrong.
			failure.Kind = FailureNoAPI
			failure.Detail = ""
		case status.Status >= 500:
			failure.Kind = FailureRemote
		case status.Status >= 400:
			failure.Kind = FailureRejected
		}
		return failure
	}

	var shape *ShapeError
	if errors.As(err, &shape) {
		// The Go type name in here is the whole of the message, so there is
		// nothing worth passing on.
		failure.Kind, failure.Detail = FailureUnexpected, ""
		return failure
	}

	if errors.Is(err, ErrDeckNotRecorded) {
		failure.Kind = FailureNotRecorded
		return failure
	}

	if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
		failure.Kind = FailureTimedOut
		return failure
	}

	// A transport failure arrives wrapped in *url.Error, whose message contains
	// the full request URL. That is an internal address, so the kind is kept and
	// the message is not.
	var transport *url.Error
	if errors.As(err, &transport) {
		failure.Kind = FailureUnreachable
		return failure
	}
	if strings.Contains(err.Error(), "ptium is unreachable") {
		failure.Kind = FailureUnreachable
	}
	return failure
}

func isTimeout(err error) bool {
	var timeout net.Error
	return errors.As(err, &timeout) && timeout.Timeout()
}

// Host is the part of a configured base URL that is safe to show an
// administrator diagnosing a connection, without the path a credential can
// hide in.
func Host(baseURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

// showableDetail drops a body that is not an explanation.
//
// When a proxy in front of Ptium answers instead of Ptium, the body is an HTML
// error page. Passing that through puts "<html><title>502 Bad Gateway</title>"
// under a sentence that says "here is what Ptium told us", which is both wrong
// and useless — the proxy is not Ptium and the page says nothing the status
// code did not.
func showableDetail(detail string) string {
	trimmed := strings.TrimSpace(detail)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") || strings.HasPrefix(lower, "<") {
		return ""
	}
	return trimmed
}
