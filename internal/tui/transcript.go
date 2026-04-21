package tui

import "github.com/boatnoah/kata/internal/agent"

type TranscriptKind string

const (
	TranscriptUser      TranscriptKind = "user"
	TranscriptAssistant TranscriptKind = "assistant"
	TranscriptTool      TranscriptKind = "tool"
	TranscriptThinking  TranscriptKind = "thinking"
	TranscriptSystem    TranscriptKind = "system"
	TranscriptError     TranscriptKind = "error"
	TranscriptDiff      TranscriptKind = "diff"
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

	// Diff carries the single-file patch payload for TranscriptDiff items.
	// DiffMaxLines caps the inline preview; 0 means unlimited.
	Diff         agent.FileChange
	DiffMaxLines int
}
