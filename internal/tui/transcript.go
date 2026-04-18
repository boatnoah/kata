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

type TranscriptItem struct {
	ID     string
	Kind   TranscriptKind
	Text   string
	Status string
	Final  bool
}
