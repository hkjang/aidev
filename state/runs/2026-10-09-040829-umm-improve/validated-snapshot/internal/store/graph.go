package store

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// A connection between two thoughts carries two separate facts: what the
// connection means, and who decided it existed. note_edges originally kept both
// in one free-text column supplied by the request body, which let any client
// write relation='dreamed' and produce an edge claiming umm's Dream layer had
// found the connection.
//
// Relation now means only meaning, checked against the vocabulary below. Origin
// means only provenance and is set by the code performing the write — it is
// never read from a request.

// Relation is what a connection asserts about two thoughts.
type Relation string

const (
	// RelationRelated is the unqualified connection: these belong together, and
	// the person drawing it did not say more than that. It is the default.
	RelationRelated Relation = "related"
	// RelationSupports: the source is evidence or argument for the target.
	RelationSupports Relation = "supports"
	// RelationContradicts: the two cannot both be right. Kept deliberately, so a
	// disagreement between two thoughts is recorded rather than resolved by
	// deleting one of them.
	RelationContradicts Relation = "contradicts"
	// RelationRefines: the target states the source more precisely.
	RelationRefines Relation = "refines"
	// RelationExpands: the target was developed out of the source. Written when
	// a Dream development is materialised into a new thought.
	RelationExpands Relation = "expands"
	// RelationFollows: the target comes after the source, in time or in sequence.
	RelationFollows Relation = "follows"
	// RelationAnswers: the source answers the target, which is a question. It is
	// what closes one — supports and refines sit near it, but neither says the
	// question has been settled.
	RelationAnswers Relation = "answers"
)

// Origin records who made the connection. It is not a request field.
type Origin string

const (
	// OriginManual: a person drew this line.
	OriginManual Origin = "manual"
	// OriginDream: written when a Dream was accepted into the space.
	OriginDream Origin = "dream"
	// OriginDevelopment: written when a Dream development became a new thought.
	OriginDevelopment Origin = "development"
	// OriginImport: created while importing an external document.
	OriginImport Origin = "import"
	// OriginAgent: asserted through MCP by an agent holding a scoped key. It is
	// a deliberate claim like a manual edge, not an inference, but the thing
	// making the claim was not a person — and once agents write to a memory, a
	// reader has to be able to see which parts of it they wrote.
	OriginAgent Origin = "agent"
	// OriginAuto: inferred by umm rather than asserted by anyone. These carry a
	// confidence, because a reader has to be able to weigh a guess.
	OriginAuto Origin = "auto"
)

// ErrUnknownRelation is returned for a relation outside the vocabulary. Callers
// map it to a 400 rather than a 500: the request was wrong, not the server.
var ErrUnknownRelation = errors.New("unknown relation")

var knownRelations = map[Relation]bool{
	RelationRelated:     true,
	RelationSupports:    true,
	RelationContradicts: true,
	RelationRefines:     true,
	RelationExpands:     true,
	RelationFollows:     true,
	RelationAnswers:     true,
}

// Relations lists the vocabulary in a stable order, for the API to advertise and
// for the interface to offer. Ordered by how often a person reaches for them,
// not alphabetically.
func Relations() []Relation {
	return []Relation{
		RelationRelated, RelationSupports, RelationContradicts,
		RelationRefines, RelationExpands, RelationFollows, RelationAnswers,
	}
}

// ParseRelation accepts what a client sent and returns the relation it names.
//
// An empty value means the client did not choose, which is the common case for a
// line dragged between two thoughts, and becomes the generic relation. Anything
// else must name a real one: silently rewriting an unrecognised relation to
// "related" would record a connection the user did not describe and hide the
// mistake from whoever sent it.
func ParseRelation(value string) (Relation, error) {
	trimmed := Relation(strings.ToLower(strings.TrimSpace(value)))
	if trimmed == "" {
		return RelationRelated, nil
	}
	if !knownRelations[trimmed] {
		return "", ErrUnknownRelation
	}
	return trimmed, nil
}

// ErrEdgeReasonTooLong is returned for a reason past MaxEdgeReason. Refused
// rather than truncated: cutting somebody's sentence in half and storing the
// half is worse than telling them it did not fit.
var ErrEdgeReasonTooLong = errors.New("edge reason too long")

// ParseEdgeReason accepts what a client sent as the why behind a connection.
//
// Empty is the normal answer and stays empty: most connections do not need a
// sentence, and an empty reason means the author did not feel the need rather
// than that one is missing. Whitespace-only collapses to empty for the same
// reason — a blank line is not a reason someone wrote.
//
// The bound is checked here as well as in SQL because a request that would
// violate the constraint should come back as a sentence about length, not as a
// database error that says nothing to whoever typed it.
func ParseEdgeReason(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	// Counted in runes: the limit is how much someone may write, and in Korean
	// a byte count would silently cut the allowance to a third.
	if utf8.RuneCountInString(trimmed) > MaxEdgeReason {
		return "", ErrEdgeReasonTooLong
	}
	return trimmed, nil
}

// Inferred reports whether umm decided this connection rather than a person.
// The interface uses it to mark an edge as a suggestion, and any reader that
// weighs evidence should treat these differently from a drawn line.
func (o Origin) Inferred() bool { return o == OriginAuto }
