package agent

// EventType enumerates stream event kinds emitted by an agent provider.
// The vocabulary is deliberately provider-neutral so a second backend can
// translate its wire protocol into this shape without the UI caring.
type EventType int

const (
	EventAgentDelta EventType = iota
	EventAgentCompleted
	EventToolCall
	EventCommandOutput
	EventTurnStarted
	EventTurnCompleted
	EventTokenUsage
	EventError
)

// Event carries streaming updates from a provider's backend.
type Event struct {
	Type     EventType
	ThreadID string
	TurnID   string
	ItemID   string
	Text     string
	Payload  map[string]any
}
