package tui

// approvalKind* values match agent.Event Payload["approvalKind"] produced by
// the Codex client (internal/codex/server_request.go ApprovalKind*).
const (
	approvalKindCommandExecution = "commandExecution"
	approvalKindFileChange       = "fileChange"
	approvalKindPermissions      = "permissions"
	approvalKindApplyPatch       = "applyPatch"
	approvalKindExecCommand      = "execCommand"
	approvalKindMcpElicitation   = "mcpElicitation"
	approvalKindToolUserInput    = "toolUserInput"
	approvalKindDynamicToolCall  = "dynamicToolCall"
)

// approvalAcceptResult builds a JSON-RPC result body for a simple accept.
// The second return is false when the UI cannot construct a valid accept
// without extra user data (e.g. permissions profile, tool answers).
func approvalAcceptResult(kind string) (any, bool) {
	switch kind {
	case approvalKindCommandExecution, approvalKindFileChange:
		return map[string]any{"decision": "accept"}, true
	case approvalKindApplyPatch, approvalKindExecCommand:
		return map[string]any{"decision": "approved"}, true
	case approvalKindMcpElicitation:
		return map[string]any{"action": "accept"}, true
	case approvalKindDynamicToolCall:
		// Minimal success; callers with richer tool output should use a
		// dedicated path once the UI collects structured results.
		return map[string]any{"success": true, "contentItems": []any{}}, true
	case approvalKindPermissions, approvalKindToolUserInput:
		return nil, false
	default:
		return nil, false
	}
}

// approvalAcceptSessionResult is like approvalAcceptResult but session-scoped
// where the protocol supports it.
func approvalAcceptSessionResult(kind string) (any, bool) {
	switch kind {
	case approvalKindCommandExecution, approvalKindFileChange:
		return map[string]any{"decision": "acceptForSession"}, true
	case approvalKindApplyPatch, approvalKindExecCommand:
		return map[string]any{"decision": "approved_for_session"}, true
	default:
		return nil, false
	}
}

// approvalDeclineResult builds a JSON-RPC result body for deny / safe cancel.
func approvalDeclineResult(kind string) (any, bool) {
	switch kind {
	case approvalKindCommandExecution, approvalKindFileChange:
		return map[string]any{"decision": "decline"}, true
	case approvalKindApplyPatch, approvalKindExecCommand:
		return map[string]any{"decision": "denied"}, true
	case approvalKindPermissions:
		return map[string]any{"permissions": map[string]any{}, "scope": "turn"}, true
	case approvalKindMcpElicitation:
		return map[string]any{"action": "decline"}, true
	case approvalKindToolUserInput:
		return map[string]any{"answers": map[string]any{}}, true
	case approvalKindDynamicToolCall:
		return map[string]any{"success": false, "contentItems": []any{}}, true
	default:
		return nil, false
	}
}
