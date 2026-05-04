package tui

import "math/rand/v2"

// AIStream owns the per-item state for one in-flight AI stream: the kind
// label, the raw delta buffer accumulated from the backend, the
// progressively-revealed substring that the typing animation has caught up
// to, and the animation state (rotating verb, dot frame counter, and the
// "tick scheduled" flag). App holds a map[itemID]*AIStream and continues to
// own history index, completion, and tool metadata for now; subsequent
// slices migrate those onto AIStream.
type AIStream struct {
	label    TranscriptKind
	buffer   string
	rendered string

	verbIdx    int
	verbSet    bool
	waitFrames int
	ticking    bool
}

func newAIStream(label TranscriptKind) *AIStream {
	return &AIStream{label: label}
}

func (s *AIStream) Label() TranscriptKind { return s.label }

func (s *AIStream) SetLabel(l TranscriptKind) { s.label = l }

func (s *AIStream) Buffer() string { return s.buffer }

func (s *AIStream) AppendDelta(clean string) { s.buffer += clean }

func (s *AIStream) ReplaceBuffer(text string) { s.buffer = text }

func (s *AIStream) Rendered() string { return s.rendered }

// SetRendered overwrites the rendered substring. Used by setToolCallSummary
// to clear prior text on a tool transition; slated for removal in slice 04
// when that path gets restructured.
func (s *AIStream) SetRendered(text string) { s.rendered = text }

// Advance reveals up to runesPerTick more runes from the buffer into the
// rendered substring. Idempotent once rendered has caught up to the buffer.
func (s *AIStream) Advance(runesPerTick int) {
	target := []rune(s.buffer)
	current := []rune(s.rendered)
	if len(current) >= len(target) {
		s.rendered = string(target)
		return
	}
	next := len(current) + runesPerTick
	if next > len(target) {
		next = len(target)
	}
	s.rendered = string(target[:next])
}

func (s *AIStream) IsTicking() bool { return s.ticking }

func (s *AIStream) SetTicking(t bool) { s.ticking = t }

// EnsureWaitingVerb picks a starting verb index if one has not been chosen
// yet. Idempotent — once a stream has a verb, it stays stable until
// ResetWaiting is called.
func (s *AIStream) EnsureWaitingVerb() {
	if s.verbSet || len(spinnerVerbs) == 0 {
		return
	}
	s.verbIdx = rand.IntN(len(spinnerVerbs))
	s.verbSet = true
}

func (s *AIStream) CurrentVerb() string {
	if len(spinnerVerbs) == 0 {
		return "Thinking"
	}
	return spinnerVerbs[s.verbIdx%len(spinnerVerbs)]
}

func (s *AIStream) AdvanceWaitingFrame() { s.waitFrames++ }

func (s *AIStream) CurrentDots() string {
	if len(spinnerDots) == 0 {
		return "..."
	}
	return spinnerDots[(s.waitFrames/6)%len(spinnerDots)]
}

func (s *AIStream) WaitingStatus() string {
	return s.CurrentVerb() + " " + s.CurrentDots()
}

// ResetWaiting drops the verb pick and frame counter so the next
// EnsureWaitingVerb / tick reseeds the animation. Used by
// beginOutboundWaitSync re-entry.
func (s *AIStream) ResetWaiting() {
	s.verbIdx = 0
	s.verbSet = false
	s.waitFrames = 0
}
