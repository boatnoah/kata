package tui

type TranscriptKind string

const (
	TranscriptUser      TranscriptKind = "user"
	TranscriptAssistant TranscriptKind = "assistant"
	TranscriptTool      TranscriptKind = "tool"
	TranscriptThinking  TranscriptKind = "thinking"
	TranscriptSystem    TranscriptKind = "system"
	TranscriptError     TranscriptKind = "error"
)

// ToolState controls the color of the tool-call glyph in ToolBlock rendering.
type ToolState int

const (
	ToolPending ToolState = iota
	ToolOK
	ToolErr
)

type TranscriptItem struct {
	ID     string
	Kind   TranscriptKind
	Text   string
	Status string
	Final  bool

	// Structured fields used by tool-call rendering. When Title is empty the
	// renderer falls back to Text as a single-line summary (for items that
	// were constructed before this path existed).
	Title  string
	Detail string
	Tool   ToolState
}
