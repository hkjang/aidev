package presentation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

// Classify decides what a person is told and whether they are offered a retry,
// so what matters is that each error lands in the kind whose sentence is true
// of it — and that nothing which names an internal host or a Go type is carried
// into the part of the answer a reader sees.

func TestClassifySortsByWhoCanFixIt(t *testing.T) {
	for _, c := range []struct {
		name string
		err  error
		want FailureKind
	}{
		{"nothing answered", fmt.Errorf("ptium is unreachable: %w",
			&url.Error{Op: "Post", URL: "http://ptium.internal:8080/api/v1/presentations", Err: errors.New("connect: connection refused")}),
			FailureUnreachable},
		{"credential refused", &StatusError{Status: 401, Detail: "api key is invalid or expired"}, FailureUnauthorized},
		{"forbidden is also the credential", &StatusError{Status: 403, Detail: "forbidden"}, FailureUnauthorized},
		{"nothing at that address", &StatusError{Status: 404, Detail: "not found"}, FailureNoAPI},
		{"ptium refused the deck", &StatusError{Status: 422, Detail: `slide 3: layout "two-column" is not in template basic`}, FailureRejected},
		{"ptium broke", &StatusError{Status: 500, Detail: "internal error"}, FailureRemote},
		{"answered something else", &ShapeError{Err: errors.New("json: cannot unmarshal array into Go value of type presentation.deckEnvelope")}, FailureUnexpected},
		{"deck made, not recorded", fmt.Errorf("%w: deck abc: %v", ErrDeckNotRecorded, errors.New("duplicate key")), FailureNotRecorded},
		{"gave up waiting", context.DeadlineExceeded, FailureTimedOut},
		{"unrecognised", errors.New("something else entirely"), FailureOther},
	} {
		if got := Classify(c.err).Kind; got != c.want {
			t.Errorf("%s: kind %q, want %q", c.name, got, c.want)
		}
	}
}

// The detail is repeated to the reader, so it must carry only what is worth
// repeating. Ptium naming the slide it could not lay out is the one case that
// tells an author what to change.
func TestClassifyRepeatsOnlyPtiumsUsefulWords(t *testing.T) {
	rejected := Classify(&StatusError{Status: 422, Detail: `slide 3: layout "two-column" is not in template basic`})
	if rejected.Detail != `slide 3: layout "two-column" is not in template basic` {
		t.Fatalf("the one explanation worth repeating was dropped: %q", rejected.Detail)
	}
	if rejected.Status != 422 {
		t.Fatalf("status %d", rejected.Status)
	}

	// A credential message is about umm's key, not about anything the reader
	// did, and repeating it sends them looking for a key they do not have.
	if detail := Classify(&StatusError{Status: 401, Detail: "api key is invalid or expired"}).Detail; detail != "" {
		t.Fatalf("the credential message was repeated to the reader: %q", detail)
	}
	// A Go type name is the whole of a shape error's message.
	if detail := Classify(&ShapeError{Err: errors.New("json: cannot unmarshal array into Go value of type presentation.deckEnvelope")}).Detail; detail != "" {
		t.Fatalf("a Go type name was repeated to the reader: %q", detail)
	}
	// A proxy answering instead of Ptium sends an HTML page, which is not an
	// explanation and is not Ptium.
	html := Classify(&StatusError{Status: 502, Detail: "<html><title>502 Bad Gateway</title><body>nginx/1.24.0</body></html>"})
	if html.Detail != "" {
		t.Fatalf("a proxy error page was passed off as Ptium's explanation: %q", html.Detail)
	}
	if html.Kind != FailureRemote {
		t.Fatalf("kind %q", html.Kind)
	}
}

// The address of an internal service is not for the person who wanted slides.
func TestClassifyKeepsTheInternalAddressOutOfWhatIsShown(t *testing.T) {
	err := fmt.Errorf("ptium is unreachable: %w",
		&url.Error{Op: "Post", URL: "http://ptium.internal:8080/api/v1/presentations", Err: errors.New("connect: connection refused")})
	failure := Classify(err)
	if failure.Detail != "" {
		t.Fatalf("detail carried the address: %q", failure.Detail)
	}
	// Kept in Technical, which only administrators are sent — losing it
	// entirely would leave whoever fixes it with nothing.
	if failure.Technical == "" {
		t.Fatal("the underlying error was dropped, leaving nothing for whoever has to fix it")
	}
}

type fakeTimeout struct{}

func (fakeTimeout) Error() string   { return "i/o timeout" }
func (fakeTimeout) Timeout() bool   { return true }
func (fakeTimeout) Temporary() bool { return true }

var _ net.Error = fakeTimeout{}

func TestClassifyRecognisesATransportTimeout(t *testing.T) {
	err := fmt.Errorf("ptium is unreachable: %w", &url.Error{Op: "Post", URL: "http://x/y", Err: fakeTimeout{}})
	if got := Classify(err).Kind; got != FailureTimedOut {
		t.Fatalf("kind %q, want %q — a timeout is worth retrying and being unreachable is a different sentence", got, FailureTimedOut)
	}
}

func TestClassifyHandlesNoError(t *testing.T) {
	if got := Classify(nil).Kind; got != FailureOther {
		t.Fatalf("kind %q", got)
	}
}

func TestHostDropsThePath(t *testing.T) {
	if got := Host("https://ptium.example.com/base/path"); got != "https://ptium.example.com" {
		t.Fatalf("Host() = %q", got)
	}
	if got := Host("not a url at all"); got != "" {
		t.Fatalf("Host() = %q", got)
	}
}
