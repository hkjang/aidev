package store

/*
What an AI call was for.

ai_calls was written for an operator — model, status, tokens, cost, latency —
which answers "what is this costing" and nothing else. The person whose
thoughts were in the prompt has a different question: when did my writing go to
an outside service, and what for. Twelve rows saying "gpt-4o success" do not
tell a Dream from an AI Assist from a heading proposed for a deck.

The vocabulary lives here because the CHECK constraint that enforces it is on a
table this package owns, and because both the package that makes the calls and
the package that compiles decks need to name one without depending on each
other.
*/

// Purpose is what an AI call was for.
//
// Its own type rather than a string so it cannot be swapped with the model name
// it sits next to — the two are both text, and the compiler is the only thing
// that would notice.
type Purpose string

// The vocabulary the ai_calls CHECK constraint accepts. Each one is a thing a
// person did, named the way they would describe it rather than the way the code
// is organised.
const (
	// PurposeDream is the nightly generation, and regenerating a candidate.
	PurposeDream Purpose = "dream"
	// PurposeAssist is AI Assist on notes the person selected.
	PurposeAssist Purpose = "assist"
	// PurposeAsk is a question put to the memory.
	PurposeAsk Purpose = "ask"
	// PurposeAgent is the assistant that looks things up before answering.
	PurposeAgent Purpose = "agent"
	// PurposeDevelop is developing an accepted Dream further.
	PurposeDevelop Purpose = "develop"
	// PurposeDeckHeadings is naming the groups on a deck's slides.
	PurposeDeckHeadings Purpose = "deck-headings"
	// PurposeDeckSections is proposing where a long talk divides into parts.
	PurposeDeckSections Purpose = "deck-sections"
)

// Purposes lists the vocabulary in the order a person reads it: what umm did on
// its own first, then what they asked for, then what a deck needed.
func Purposes() []Purpose {
	return []Purpose{
		PurposeDream, PurposeAssist, PurposeAsk, PurposeAgent, PurposeDevelop,
		PurposeDeckHeadings, PurposeDeckSections,
	}
}
