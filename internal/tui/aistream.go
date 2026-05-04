package tui

import "math/rand/v2"

// AIStream owns the per-item state for one in-flight AI stream: kind label,
// raw delta buffer, progressively-revealed substring, animation state
// (rotating verb, dot frame counter, "tick scheduled" flag), completion
// flag, and tool metadata. App holds a map[itemID]*AIStream; the only
// per-itemID maps still on App are streams and aiIndexes (history index).
type AIStream struct {
	label    TranscriptKind
	buffer   string
	rendered string

	verbIdx    int
	verbSet    bool
	waitFrames int
	ticking    bool

	completed bool

	toolTitle  string
	toolDetail string
	toolState  ToolState
}

func newAIStream(label TranscriptKind) *AIStream {
	return &AIStream{label: label}
}

func (s *AIStream) Label() TranscriptKind { return s.label }

func (s *AIStream) SetLabel(l TranscriptKind) { s.label = l }

func (s *AIStream) Buffer() string { return s.buffer }

func (s *AIStream) AppendDelta(clean string) { s.buffer += clean }

func (s *AIStream) Rendered() string { return s.rendered }

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

// Reset clears buffer and rendered. Used on tool transitions where any
// prior streamed text should be wiped before tool metadata fills in.
func (s *AIStream) Reset() {
	s.buffer = ""
	s.rendered = ""
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

func (s *AIStream) IsCompleted() bool { return s.completed }

func (s *AIStream) SetCompleted(c bool) { s.completed = c }

// Complete marks the stream as completed. If finalText is non-empty, it
// replaces the buffer with it; empty preserves accumulated text (some
// completion events arrive without a final-text snapshot).
func (s *AIStream) Complete(finalText string) {
	if finalText != "" {
		s.buffer = finalText
	}
	s.completed = true
}

// Tool returns the current tool-call metadata. Meaningful only when
// Label() == TranscriptTool.
func (s *AIStream) Tool() (title, detail string, state ToolState) {
	return s.toolTitle, s.toolDetail, s.toolState
}

// UpdateTool merges the incoming summary into the stream's tool metadata.
// Empty Title or Detail in the summary preserve the prior values so that
// "started → completed" sequences with only state changes don't blank the
// command line from the prior render.
func (s *AIStream) UpdateTool(summary toolSummary) {
	if summary.Title != "" {
		s.toolTitle = summary.Title
	}
	if summary.Detail != "" {
		s.toolDetail = summary.Detail
	}
	s.toolState = summary.State
}
