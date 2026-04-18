package agent

import "context"

// Provider is the minimal contract the TUI needs from an AI backend.
// Codex is the current implementation; future backends satisfy this
// interface and plug in without touching the UI layer.
type Provider interface {
	Start(ctx context.Context) error
	SendText(ctx context.Context, text string) (string, error)
	Events() <-chan Event
	Model() string
	Close() error
}
