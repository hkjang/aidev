package presentation

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

/*
Writing a talk out in Ptium's deck source language.

Ptium takes a deck as text and compiles it against a template: `PUT
/api/v1/presentations/{id}/source` replaces the slides with exactly what the
text says, and reports every adjustment it had to make in `warnings`. That is
the endpoint this file targets, and choosing it decides something important.

The other way in is `prompt`, a single string capped at 12000 characters that a
model turns into a deck. A space of any size overruns that cap, the connections
have nowhere to go, and — the part that matters — the sentences on the slides
come back written by the model rather than by the person. Compiling to source
instead means a thought reaches its slide unchanged, the same space always
produces the same deck, and `dryRun` gives a preview that is the real thing
rather than an impression of it.

The language, in full, is in ptium's docs/deck-source.md. What is used here:

	# title      a new slide and its title
	@kind        cover | section | content | comparison | closing
	> lead       the line under the title
	- bullet     a point, two spaces per level of nesting
	// comment   ignored by the compiler

Escaping is deliberately narrow: a line is only ambiguous when it starts with a
marker, so that is the only thing escaped. Doing more would alter the person's
text, which is the one thing this package must never do.
*/

// SourceOptions controls what the emitted text carries besides the talk.
type SourceOptions struct {
	// Trace writes each slide's originating note ids as a comment. Ptium ignores
	// comments, and the round trip through its editor keeps them, which is what
	// lets a slide say which thoughts it came from.
	Trace bool
}

// WriteSource renders a talk as Ptium deck source.
func WriteSource(story Storyline, opts SourceOptions) string {
	var b strings.Builder

	if title := strings.TrimSpace(story.Title); title != "" {
		b.WriteString("# " + escapeLine(title) + "\n")
		b.WriteString("@cover\n")
		if len(story.Slides) > 0 {
			// The first thought is what the talk opens on, so it is also the
			// most honest subtitle available without writing one.
			if lead := firstLine(story.Slides[0].leadOrTitle()); lead != "" {
				b.WriteString("> " + escapeLine(lead) + "\n")
			}
		}
		b.WriteString("\n")
	}

	for _, slide := range story.Slides {
		writeSlide(&b, slide, opts)
	}
	return b.String()
}

func writeSlide(b *strings.Builder, slide Slide, opts SourceOptions) {
	title := strings.TrimSpace(slide.Title)
	if title == "" {
		title = strings.TrimSpace(firstLine(slide.leadOrTitle()))
	}
	if title == "" {
		return // a slide with nothing to say is not a slide
	}
	b.WriteString("# " + escapeLine(title) + "\n")

	switch slide.Role {
	case RoleSection:
		b.WriteString("@section\n")
	case RoleComparison:
		b.WriteString("@comparison\n")
	default:
		b.WriteString("@content\n")
	}

	if opts.Trace && len(slide.From) > 0 {
		b.WriteString("// " + TraceComment(slide.From) + "\n")
	}

	if lead := strings.TrimSpace(slide.Lead); lead != "" && lead != title {
		for _, line := range splitLines(lead) {
			b.WriteString("> " + escapeLine(line) + "\n")
		}
	}

	for _, point := range slide.Points {
		text := strings.TrimSpace(point.Text)
		if text == "" {
			continue
		}
		// The title is already this thought, word for word — on a comparison
		// slide, whose title is one of the two sides, printing it again put the
		// same sentence on the slide twice.
		if text == title {
			continue
		}
		indent := strings.Repeat("  ", point.Depth)
		lines := splitLines(text)
		for i, line := range lines {
			if i == 0 {
				b.WriteString(indent + "- " + escapeLine(line) + "\n")
				continue
			}
			// A thought written over several lines stays several lines, one
			// level in, rather than being joined into a paragraph it was not.
			b.WriteString(indent + "  - " + escapeLine(line) + "\n")
		}
	}
	b.WriteString("\n")
}

// TracePrefix marks the comments this package writes, so a reader — and the
// code that maps a slide back to its thoughts — can tell them from a comment
// someone typed.
const TracePrefix = "umm:notes="

// TraceComment renders the ids a slide came from.
func TraceComment(ids []uuid.UUID) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, id.String())
	}
	return TracePrefix + strings.Join(parts, ",")
}

// ParseTrace reads back the ids from a comment written by TraceComment. It
// returns nil for any other comment, including one a person wrote themselves.
func ParseTrace(line string) []uuid.UUID {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "//"))
	rest, ok := strings.CutPrefix(line, TracePrefix)
	if !ok {
		return nil
	}
	var ids []uuid.UUID
	for _, part := range strings.Split(rest, ",") {
		id, err := uuid.Parse(strings.TrimSpace(part))
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// escapeLine protects a line that would otherwise be read as markup.
//
// Only a leading marker is ambiguous. A `#` or a `-` in the middle of a
// sentence is just punctuation, and escaping it would put a backslash into
// someone's words.
func escapeLine(line string) string {
	line = strings.TrimRight(line, " \t")
	for _, marker := range []string{"#", "@", ">", "-", "!", "::", "//", `\`} {
		if strings.HasPrefix(line, marker) {
			return `\` + line
		}
	}
	return line
}

func splitLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstLine(text string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(first)
}

func (s Slide) leadOrTitle() string {
	if lead := strings.TrimSpace(s.Lead); lead != "" {
		return lead
	}
	return s.Title
}

// Summary is a one-line description of what compiling produced, for a log line
// or a status message.
func (s Storyline) Summary() string {
	return fmt.Sprintf("%d slides from %d thoughts, %d left out", len(s.Slides), s.thoughtCount(), len(s.Excluded))
}

func (s Storyline) thoughtCount() int {
	seen := map[uuid.UUID]bool{}
	for _, slide := range s.Slides {
		for _, id := range slide.From {
			seen[id] = true
		}
	}
	return len(seen)
}
