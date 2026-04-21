package agent

import (
	"context"
	"encoding/json"
	"errors"
)

// ErrRPCResponderUnsupported is returned when the active provider cannot send
// JSON-RPC result payloads for server-initiated requests (approvals, etc.).
var ErrRPCResponderUnsupported = errors.New("agent: server RPC responses not supported by this provider")

// RPCResponder is implemented by backends that can answer server→client
// JSON-RPC requests (Codex approvals, MCP elicitation, dynamic tool calls).
// The id is the raw JSON "id" from agent.Event.RPCID.
type RPCResponder interface {
	RespondServerRPC(ctx context.Context, id json.RawMessage, result any) error
}
