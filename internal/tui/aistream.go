package tui

// AIStream owns the per-item state for one in-flight AI stream: the kind
// label, the raw delta buffer accumulated from the backend, and the
// progressively-revealed substring that the typing animation has caught up
// to. App holds a map[itemID]*AIStream and continues to own everything else
// (history index, tick lifecycle, completion, tool metadata) for now;
// subsequent slices migrate those onto AIStream.
type AIStream struct {
	label    TranscriptKind
	buffer   string
	rendered string
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

func (s *AIStream) SetRendered(text string) { s.rendered = text }
