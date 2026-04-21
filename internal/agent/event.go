package agent

import "encoding/json"

// EventType enumerates stream event kinds emitted from an agent provider.
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
	// EventApprovalRequired signals a server-initiated JSON-RPC request that
	// needs a user decision before the backend can continue (e.g. Codex
	// command execution approval). RPCID holds the raw JSON "id" value to echo
	// in the JSON-RPC result response.
	//
	// Payload conventions (Codex app-server today): keys include "method"
	// (full JSON-RPC method string), "approvalKind" (short discriminator such
	// as "commandExecution", "fileChange", "permissions"), plus method-specific
	// fields (command, cwd, reason, rawParams, etc.).
	EventApprovalRequired
)

// Event carries streaming updates from a provider's backend.
type Event struct {
	Type     EventType
	ThreadID string
	TurnID   string
	ItemID   string
	Text     string
	Payload  map[string]any

	// RPCID is set for server-initiated JSON-RPC requests that expect a client
	// result (e.g. EventApprovalRequired). It is the raw JSON encoding of the
	// request "id" field and must be sent back verbatim when responding.
	RPCID json.RawMessage `json:"-"`
}
