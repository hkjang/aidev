package presentation

import (
	"strings"
)

/*
The same talk, written down as a document.

A space compiles to an ordered run of slides, and that order is not umm's
guess: it is a follows chain, which is somebody stating a sequence outright,
and failing that the arrangement they made on the canvas, which is also
something they did. Those are the same two facts a person needs to start
writing rather than presenting — so the ordering work already done for decks
answers a question it was not built for, and answers it without a model.

This writes that run as Markdown headings and prose instead of deck source. It
is the same rule as everywhere else in this package: the sentences are the ones
the person wrote. Nothing here paraphrases, summarises or joins them up. What
it adds is structure — which of their sentences is a heading, which sits under
it — and structure is what they already expressed by drawing.

A document is not a deck, so two things differ from WriteSource. There is no
cover, because a document opens on its title. And a comparison keeps both
sides as prose under one heading rather than as a slide of two columns, since
a reader has as long as they want with it.
*/

// WriteOutline turns a compiled talk into a Markdown document.
//
// Headings nest one level below the title so the result can be pasted into a
// larger document without its headings outranking the ones around it.
func WriteOutline(story Storyline) string {
	var b strings.Builder

	if title := strings.TrimSpace(story.Title); title != "" {
		b.WriteString("# " + title + "\n\n")
	}

	for _, slide := range story.Slides {
		writeOutlineSection(&b, slide)
	}
	return b.String()
}

func writeOutlineSection(b *strings.Builder, slide Slide) {
	heading := strings.TrimSpace(slide.Title)
	if heading == "" {
		heading = strings.TrimSpace(firstLine(slide.leadOrTitle()))
	}
	if heading == "" {
		return // a slide with nothing to say is not a section
	}

	// A part heading opens a part; everything else sits inside one. Keeping the
	// two at different levels is the whole reason someone divided the talk.
	if slide.Role == RoleSection && slide.Sectioned {
		b.WriteString("## " + heading + "\n\n")
		return
	}
	b.WriteString("### " + heading + "\n\n")

	if lead := strings.TrimSpace(slide.Lead); lead != "" && lead != heading {
		b.WriteString(lead + "\n\n")
	}

	for _, point := range slide.Points {
		text := strings.TrimSpace(point.Text)
		if text == "" || text == heading {
			continue
		}
		// Depth is how far a thought sits from the claim it was attached to,
		// and it is kept because the person expressed it by connecting things.
		b.WriteString(strings.Repeat("  ", max(point.Depth, 0)) + "- " + text + "\n")
	}
	if len(slide.Points) > 0 {
		b.WriteString("\n")
	}
}
