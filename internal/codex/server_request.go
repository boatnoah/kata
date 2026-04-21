package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/boatnoah/kata/internal/agent"
)

// JSON-RPC method names for server→client requests (Codex app-server
// ServerRequest.ts). Keep in sync with upstream protocol.
const (
	methodItemCommandExecutionRequestApproval = "item/commandExecution/requestApproval"
	methodItemFileChangeRequestApproval       = "item/fileChange/requestApproval"
	methodItemPermissionsRequestApproval      = "item/permissions/requestApproval"
	methodItemToolRequestUserInput            = "item/tool/requestUserInput"
	methodItemToolCall                        = "item/tool/call"
	methodMcpServerElicitationRequest         = "mcpServer/elicitation/request"
	methodApplyPatchApproval                  = "applyPatchApproval"
	methodExecCommandApproval                 = "execCommandApproval"
	methodAccountChatgptAuthTokensRefresh     = "account/chatgptAuthTokens/refresh"
)

// Payload keys for agent.EventApprovalRequired (Codex-agnostic UI hints).
const (
	approvalPayloadKind   = "approvalKind"
	approvalPayloadMethod = "method"
)

// ApprovalKind values stored in Event.Payload[approvalPayloadKind].
const (
	ApprovalKindCommandExecution = "commandExecution"
	ApprovalKindFileChange       = "fileChange"
	ApprovalKindPermissions      = "permissions"
	ApprovalKindApplyPatch       = "applyPatch"
	ApprovalKindExecCommand      = "execCommand"
	ApprovalKindMcpElicitation   = "mcpElicitation"
	ApprovalKindToolUserInput    = "toolUserInput"
	ApprovalKindDynamicToolCall  = "dynamicToolCall"
)

// isServerInitiatedJSONRPCRequest reports JSON-RPC lines from the server that
// expect a client result (id + method + params), as opposed to responses to
// our own calls (id + result or id + error).
func isServerInitiatedJSONRPCRequest(msg rpcMessage) bool {
	return len(msg.ID) > 0 && msg.Method != "" && msg.Error == nil && len(msg.Result) == 0
}

func cloneJSONRPCID(id json.RawMessage) json.RawMessage {
	return json.RawMessage(append(json.RawMessage(nil), id...))
}

func (c *Client) handleServerRequest(msg rpcMessage) {
	id := cloneJSONRPCID(msg.ID)

	switch msg.Method {
	case methodItemCommandExecutionRequestApproval:
		c.handleCommandExecutionApproval(id, msg.Params)
	case methodItemFileChangeRequestApproval:
		c.handleFileChangeApproval(id, msg.Params)
	case methodItemPermissionsRequestApproval:
		c.handlePermissionsApproval(id, msg.Params)
	case methodItemToolRequestUserInput:
		c.handleToolRequestUserInput(id, msg.Params)
	case methodItemToolCall:
		c.handleDynamicToolCall(id, msg.Params)
	case methodMcpServerElicitationRequest:
		c.handleMcpElicitation(id, msg.Params)
	case methodApplyPatchApproval:
		c.handleApplyPatchApproval(id, msg.Params)
	case methodExecCommandApproval:
		c.handleExecCommandApproval(id, msg.Params)
	case methodAccountChatgptAuthTokensRefresh:
		c.emitError(fmt.Errorf("codex: %s requires interactive token refresh (not implemented in kata yet)", msg.Method))
		_ = c.sendJSONRPCError(id, -32003, "kata: ChatGPT token refresh is not implemented; restart with valid auth")
	default:
		c.emitError(fmt.Errorf("codex: unsupported JSON-RPC server request %q", msg.Method))
		_ = c.sendJSONRPCError(id, -32601, "kata: unsupported server request "+msg.Method)
	}
}

func (c *Client) handleCommandExecutionApproval(id json.RawMessage, params json.RawMessage) {
	ev, err := commandExecutionApprovalEvent(id, params)
	if err != nil {
		c.emitError(fmt.Errorf("codex %s: %w", methodItemCommandExecutionRequestApproval, err))
		_ = c.sendJSONRPCResult(id, map[string]any{"decision": "decline"})
		return
	}
	c.emit(ev)
}

func (c *Client) handleFileChangeApproval(id json.RawMessage, params json.RawMessage) {
	ev, err := fileChangeApprovalEvent(id, params)
	if err != nil {
		c.emitError(fmt.Errorf("codex %s: %w", methodItemFileChangeRequestApproval, err))
		_ = c.sendJSONRPCResult(id, map[string]any{"decision": "decline"})
		return
	}
	c.emit(ev)
}

func (c *Client) handlePermissionsApproval(id json.RawMessage, params json.RawMessage) {
	ev, err := permissionsApprovalEvent(id, params)
	if err != nil {
		c.emitError(fmt.Errorf("codex %s: %w", methodItemPermissionsRequestApproval, err))
		_ = c.sendJSONRPCResult(id, map[string]any{"permissions": map[string]any{}, "scope": "turn"})
		return
	}
	c.emit(ev)
}

func (c *Client) handleToolRequestUserInput(id json.RawMessage, params json.RawMessage) {
	ev, err := toolRequestUserInputEvent(id, params)
	if err != nil {
		c.emitError(fmt.Errorf("codex %s: %w", methodItemToolRequestUserInput, err))
		_ = c.sendJSONRPCResult(id, map[string]any{"answers": map[string]any{}})
		return
	}
	c.emit(ev)
}

func (c *Client) handleDynamicToolCall(id json.RawMessage, params json.RawMessage) {
	ev, err := dynamicToolCallEvent(id, params)
	if err != nil {
		c.emitError(fmt.Errorf("codex %s: %w", methodItemToolCall, err))
		_ = c.sendJSONRPCResult(id, map[string]any{"success": false, "contentItems": []any{}})
		return
	}
	c.emit(ev)
}

func (c *Client) handleMcpElicitation(id json.RawMessage, params json.RawMessage) {
	ev, err := mcpElicitationEvent(id, params)
	if err != nil {
		c.emitError(fmt.Errorf("codex %s: %w", methodMcpServerElicitationRequest, err))
		_ = c.sendJSONRPCResult(id, map[string]any{"action": "decline"})
		return
	}
	c.emit(ev)
}

func (c *Client) handleApplyPatchApproval(id json.RawMessage, params json.RawMessage) {
	ev, err := applyPatchApprovalEvent(id, params)
	if err != nil {
		c.emitError(fmt.Errorf("codex %s: %w", methodApplyPatchApproval, err))
		_ = c.sendJSONRPCResult(id, map[string]any{"decision": "denied"})
		return
	}
	c.emit(ev)
}

func (c *Client) handleExecCommandApproval(id json.RawMessage, params json.RawMessage) {
	ev, err := execCommandApprovalEvent(id, params)
	if err != nil {
		c.emitError(fmt.Errorf("codex %s: %w", methodExecCommandApproval, err))
		_ = c.sendJSONRPCResult(id, map[string]any{"decision": "denied"})
		return
	}
	c.emit(ev)
}

// --- Params + event builders ---

type commandExecutionRequestApprovalParams struct {
	ThreadID                        string          `json:"threadId"`
	TurnID                          string          `json:"turnId"`
	ItemID                          string          `json:"itemId"`
	ApprovalID                      *string         `json:"approvalId"`
	Command                         *string         `json:"command"`
	Cwd                             *string         `json:"cwd"`
	Reason                          *string         `json:"reason"`
	NetworkApprovalContext          json.RawMessage `json:"networkApprovalContext"`
	ProposedExecpolicyAmendment     []string        `json:"proposedExecpolicyAmendment"`
	ProposedNetworkPolicyAmendments json.RawMessage `json:"proposedNetworkPolicyAmendments"`
	CommandActions                  json.RawMessage `json:"commandActions"`
}

func commandExecutionApprovalEvent(id json.RawMessage, paramsJSON []byte) (agent.Event, error) {
	var params commandExecutionRequestApprovalParams
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return agent.Event{}, err
	}
	pl := approvalPayloadCommandExecution(&params)
	return agent.Event{
		Type:     agent.EventApprovalRequired,
		ThreadID: params.ThreadID,
		TurnID:   params.TurnID,
		ItemID:   params.ItemID,
		Text:     summarizeCommandExecutionApproval(&params),
		RPCID:    cloneJSONRPCID(id),
		Payload:  pl,
	}, nil
}

func summarizeCommandExecutionApproval(p *commandExecutionRequestApprovalParams) string {
	if p.Command != nil {
		if s := strings.TrimSpace(*p.Command); s != "" {
			return s
		}
	}
	if p.Reason != nil {
		if s := strings.TrimSpace(*p.Reason); s != "" {
			return s
		}
	}
	return "command execution approval"
}

func approvalPayloadCommandExecution(p *commandExecutionRequestApprovalParams) map[string]any {
	out := map[string]any{
		approvalPayloadMethod: methodItemCommandExecutionRequestApproval,
		approvalPayloadKind:   ApprovalKindCommandExecution,
	}
	if p.ApprovalID != nil {
		out["approvalId"] = *p.ApprovalID
	}
	if p.Command != nil {
		out["command"] = *p.Command
	}
	if p.Cwd != nil {
		out["cwd"] = *p.Cwd
	}
	if p.Reason != nil {
		out["reason"] = *p.Reason
	}
	if len(p.ProposedExecpolicyAmendment) > 0 {
		out["proposedExecpolicyAmendment"] = append([]string(nil), p.ProposedExecpolicyAmendment...)
	}
	if len(p.NetworkApprovalContext) > 0 {
		out["networkApprovalContext"] = json.RawMessage(append(json.RawMessage(nil), p.NetworkApprovalContext...))
	}
	if len(p.ProposedNetworkPolicyAmendments) > 0 {
		out["proposedNetworkPolicyAmendments"] = json.RawMessage(append(json.RawMessage(nil), p.ProposedNetworkPolicyAmendments...))
	}
	if len(p.CommandActions) > 0 {
		out["commandActions"] = json.RawMessage(append(json.RawMessage(nil), p.CommandActions...))
	}
	return out
}

type fileChangeRequestApprovalParams struct {
	ThreadID  string  `json:"threadId"`
	TurnID    string  `json:"turnId"`
	ItemID    string  `json:"itemId"`
	Reason    *string `json:"reason"`
	GrantRoot *string `json:"grantRoot"`
}

func fileChangeApprovalEvent(id json.RawMessage, paramsJSON []byte) (agent.Event, error) {
	var p fileChangeRequestApprovalParams
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return agent.Event{}, err
	}
	pl := map[string]any{
		approvalPayloadMethod: methodItemFileChangeRequestApproval,
		approvalPayloadKind:   ApprovalKindFileChange,
	}
	if p.Reason != nil {
		pl["reason"] = *p.Reason
	}
	if p.GrantRoot != nil {
		pl["grantRoot"] = *p.GrantRoot
	}
	text := "file change approval"
	if p.Reason != nil && strings.TrimSpace(*p.Reason) != "" {
		text = strings.TrimSpace(*p.Reason)
	}
	return agent.Event{
		Type:     agent.EventApprovalRequired,
		ThreadID: p.ThreadID,
		TurnID:   p.TurnID,
		ItemID:   p.ItemID,
		Text:     text,
		RPCID:    cloneJSONRPCID(id),
		Payload:  pl,
	}, nil
}

type permissionsRequestApprovalParams struct {
	ThreadID    string          `json:"threadId"`
	TurnID      string          `json:"turnId"`
	ItemID      string          `json:"itemId"`
	Cwd         string          `json:"cwd"`
	Reason      *string         `json:"reason"`
	Permissions json.RawMessage `json:"permissions"`
}

func permissionsApprovalEvent(id json.RawMessage, paramsJSON []byte) (agent.Event, error) {
	var p permissionsRequestApprovalParams
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return agent.Event{}, err
	}
	pl := map[string]any{
		approvalPayloadMethod: methodItemPermissionsRequestApproval,
		approvalPayloadKind:   ApprovalKindPermissions,
		"cwd":                 p.Cwd,
	}
	if p.Reason != nil {
		pl["reason"] = *p.Reason
	}
	if len(p.Permissions) > 0 {
		pl["permissions"] = json.RawMessage(append(json.RawMessage(nil), p.Permissions...))
	}
	text := "permissions approval"
	if p.Reason != nil && strings.TrimSpace(*p.Reason) != "" {
		text = strings.TrimSpace(*p.Reason)
	}
	return agent.Event{
		Type:     agent.EventApprovalRequired,
		ThreadID: p.ThreadID,
		TurnID:   p.TurnID,
		ItemID:   p.ItemID,
		Text:     text,
		RPCID:    cloneJSONRPCID(id),
		Payload:  pl,
	}, nil
}

type toolRequestUserInputParams struct {
	ThreadID  string          `json:"threadId"`
	TurnID    string          `json:"turnId"`
	ItemID    string          `json:"itemId"`
	Questions json.RawMessage `json:"questions"`
}

func toolRequestUserInputEvent(id json.RawMessage, paramsJSON []byte) (agent.Event, error) {
	var p toolRequestUserInputParams
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return agent.Event{}, err
	}
	pl := map[string]any{
		approvalPayloadMethod: methodItemToolRequestUserInput,
		approvalPayloadKind:   ApprovalKindToolUserInput,
	}
	if len(p.Questions) > 0 {
		pl["questions"] = json.RawMessage(append(json.RawMessage(nil), p.Questions...))
	}
	text := summarizeToolUserInputText(p.Questions)
	return agent.Event{
		Type:     agent.EventApprovalRequired,
		ThreadID: p.ThreadID,
		TurnID:   p.TurnID,
		ItemID:   p.ItemID,
		Text:     text,
		RPCID:    cloneJSONRPCID(id),
		Payload:  pl,
	}, nil
}

func summarizeToolUserInputText(questions json.RawMessage) string {
	var qs []struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal(questions, &qs); err != nil || len(qs) == 0 {
		return "tool user input"
	}
	if t := strings.TrimSpace(qs[0].Question); t != "" {
		return t
	}
	return "tool user input"
}

type dynamicToolCallParams struct {
	ThreadID  string          `json:"threadId"`
	TurnID    string          `json:"turnId"`
	CallID    string          `json:"callId"`
	Tool      string          `json:"tool"`
	Namespace *string         `json:"namespace"`
	Arguments json.RawMessage `json:"arguments"`
}

func dynamicToolCallEvent(id json.RawMessage, paramsJSON []byte) (agent.Event, error) {
	var p dynamicToolCallParams
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return agent.Event{}, err
	}
	pl := map[string]any{
		approvalPayloadMethod: methodItemToolCall,
		approvalPayloadKind:   ApprovalKindDynamicToolCall,
		"tool":                p.Tool,
	}
	if p.Namespace != nil {
		pl["namespace"] = *p.Namespace
	}
	if len(p.Arguments) > 0 {
		pl["arguments"] = json.RawMessage(append(json.RawMessage(nil), p.Arguments...))
	}
	text := strings.TrimSpace(p.Tool)
	if text == "" {
		text = "dynamic tool call"
	}
	return agent.Event{
		Type:     agent.EventApprovalRequired,
		ThreadID: p.ThreadID,
		TurnID:   p.TurnID,
		ItemID:   p.CallID,
		Text:     text,
		RPCID:    cloneJSONRPCID(id),
		Payload:  pl,
	}, nil
}

func mcpElicitationEvent(id json.RawMessage, paramsJSON []byte) (agent.Event, error) {
	var wire struct {
		ThreadID   string          `json:"threadId"`
		TurnID     json.RawMessage `json:"turnId"`
		ServerName string          `json:"serverName"`
		Mode       string          `json:"mode"`
		Message    string          `json:"message"`
		URL        string          `json:"url"`
	}
	if err := json.Unmarshal(paramsJSON, &wire); err != nil {
		return agent.Event{}, err
	}
	turnID := ""
	if len(wire.TurnID) > 0 && string(wire.TurnID) != "null" {
		_ = json.Unmarshal(wire.TurnID, &turnID)
	}
	pl := map[string]any{
		approvalPayloadMethod: methodMcpServerElicitationRequest,
		approvalPayloadKind:   ApprovalKindMcpElicitation,
		"serverName":          wire.ServerName,
		"mode":                wire.Mode,
	}
	if wire.Message != "" {
		pl["message"] = wire.Message
	}
	if wire.URL != "" {
		pl["url"] = wire.URL
	}
	// Stash full params for UI / future structured replies (form schema, etc.).
	pl["rawParams"] = json.RawMessage(append(json.RawMessage(nil), paramsJSON...))

	text := strings.TrimSpace(wire.Message)
	if text == "" {
		text = fmt.Sprintf("MCP elicitation (%s)", wire.ServerName)
	}
	return agent.Event{
		Type:     agent.EventApprovalRequired,
		ThreadID: wire.ThreadID,
		TurnID:   turnID,
		ItemID:   "",
		Text:     text,
		RPCID:    cloneJSONRPCID(id),
		Payload:  pl,
	}, nil
}

type applyPatchApprovalParams struct {
	CallID         string          `json:"callId"`
	ConversationID string          `json:"conversationId"`
	FileChanges    json.RawMessage `json:"fileChanges"`
	GrantRoot      *string         `json:"grantRoot"`
	Reason         *string         `json:"reason"`
}

func applyPatchApprovalEvent(id json.RawMessage, paramsJSON []byte) (agent.Event, error) {
	var p applyPatchApprovalParams
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return agent.Event{}, err
	}
	pl := map[string]any{
		approvalPayloadMethod: methodApplyPatchApproval,
		approvalPayloadKind:   ApprovalKindApplyPatch,
		"callId":              p.CallID,
	}
	if p.GrantRoot != nil {
		pl["grantRoot"] = *p.GrantRoot
	}
	if p.Reason != nil {
		pl["reason"] = *p.Reason
	}
	if len(p.FileChanges) > 0 {
		pl["fileChanges"] = json.RawMessage(append(json.RawMessage(nil), p.FileChanges...))
		if parsed, err := ParseFileChanges(p.FileChanges); err == nil && len(parsed) > 0 {
			pl[approvalPayloadParsedFileChanges] = parsed
		}
	}
	text := "apply patch approval"
	if p.Reason != nil && strings.TrimSpace(*p.Reason) != "" {
		text = strings.TrimSpace(*p.Reason)
	}
	return agent.Event{
		Type:     agent.EventApprovalRequired,
		ThreadID: p.ConversationID,
		TurnID:   "",
		ItemID:   p.CallID,
		Text:     text,
		RPCID:    cloneJSONRPCID(id),
		Payload:  pl,
	}, nil
}

type execCommandApprovalParams struct {
	ApprovalID     *string         `json:"approvalId"`
	CallID         string          `json:"callId"`
	Command        []string        `json:"command"`
	ConversationID string          `json:"conversationId"`
	Cwd            string          `json:"cwd"`
	Reason         *string         `json:"reason"`
	ParsedCmd      json.RawMessage `json:"parsedCmd"`
}

func execCommandApprovalEvent(id json.RawMessage, paramsJSON []byte) (agent.Event, error) {
	var p execCommandApprovalParams
	if err := json.Unmarshal(paramsJSON, &p); err != nil {
		return agent.Event{}, err
	}
	pl := map[string]any{
		approvalPayloadMethod: methodExecCommandApproval,
		approvalPayloadKind:   ApprovalKindExecCommand,
		"callId":              p.CallID,
		"cwd":                 p.Cwd,
	}
	if p.ApprovalID != nil {
		pl["approvalId"] = *p.ApprovalID
	}
	if len(p.Command) > 0 {
		pl["command"] = append([]string(nil), p.Command...)
	}
	if p.Reason != nil {
		pl["reason"] = *p.Reason
	}
	if len(p.ParsedCmd) > 0 {
		pl["parsedCmd"] = json.RawMessage(append(json.RawMessage(nil), p.ParsedCmd...))
	}
	text := strings.TrimSpace(strings.Join(p.Command, " "))
	if text == "" && p.Reason != nil {
		text = strings.TrimSpace(*p.Reason)
	}
	if text == "" {
		text = "exec command approval"
	}
	return agent.Event{
		Type:     agent.EventApprovalRequired,
		ThreadID: p.ConversationID,
		TurnID:   "",
		ItemID:   p.CallID,
		Text:     text,
		RPCID:    cloneJSONRPCID(id),
		Payload:  pl,
	}, nil
}

// --- JSON-RPC reply helpers ---

// RespondJSONRPCResult sends a raw JSON-RPC result for a server-initiated
// request id. Prefer the typed Respond* methods when possible.
func (c *Client) RespondJSONRPCResult(id json.RawMessage, result any) error {
	if len(id) == 0 {
		return fmt.Errorf("codex: approval response missing rpc id")
	}
	return c.sendJSONRPCResult(id, result)
}

// RespondServerRPC implements agent.RPCResponder.
func (c *Client) RespondServerRPC(ctx context.Context, id json.RawMessage, result any) error {
	_ = ctx
	return c.RespondJSONRPCResult(id, result)
}

// RespondCommandExecutionApproval sends the JSON-RPC result for
// item/commandExecution/requestApproval. decision is typically "accept",
// "decline", "cancel", or "acceptForSession", or a structured
// CommandExecutionApprovalDecision variant.
func (c *Client) RespondCommandExecutionApproval(id json.RawMessage, decision any) error {
	return c.RespondJSONRPCResult(id, map[string]any{"decision": decision})
}

// RespondFileChangeRequestApproval sends the result for item/fileChange/requestApproval.
func (c *Client) RespondFileChangeRequestApproval(id json.RawMessage, decision string) error {
	return c.RespondJSONRPCResult(id, map[string]any{"decision": decision})
}

// RespondApplyPatchApproval sends the result for applyPatchApproval (ReviewDecision).
func (c *Client) RespondApplyPatchApproval(id json.RawMessage, decision any) error {
	return c.RespondJSONRPCResult(id, map[string]any{"decision": decision})
}

// RespondExecCommandApproval sends the result for execCommandApproval (ReviewDecision).
func (c *Client) RespondExecCommandApproval(id json.RawMessage, decision any) error {
	return c.RespondJSONRPCResult(id, map[string]any{"decision": decision})
}

// RespondPermissionsRequestApproval sends the result for item/permissions/requestApproval.
func (c *Client) RespondPermissionsRequestApproval(id json.RawMessage, permissions map[string]any, scope string) error {
	if scope == "" {
		scope = "turn"
	}
	return c.RespondJSONRPCResult(id, map[string]any{
		"permissions": permissions,
		"scope":       scope,
	})
}

// RespondMcpServerElicitation sends the result for mcpServer/elicitation/request.
// action is "accept", "decline", or "cancel". content is optional (e.g. form answers).
func (c *Client) RespondMcpServerElicitation(id json.RawMessage, action string, content any) error {
	out := map[string]any{"action": action}
	if content != nil {
		out["content"] = content
	}
	return c.RespondJSONRPCResult(id, out)
}

// RespondToolRequestUserInput sends the result for item/tool/requestUserInput.
// answers maps question id → { "answers": [...] } per ToolRequestUserInputResponse.
func (c *Client) RespondToolRequestUserInput(id json.RawMessage, answers map[string]map[string]any) error {
	return c.RespondJSONRPCResult(id, map[string]any{"answers": answers})
}

// RespondDynamicToolCall sends the result for item/tool/call.
func (c *Client) RespondDynamicToolCall(id json.RawMessage, success bool, contentItems []any) error {
	if contentItems == nil {
		contentItems = []any{}
	}
	return c.RespondJSONRPCResult(id, map[string]any{
		"success":      success,
		"contentItems": contentItems,
	})
}

// RespondAccountChatgptAuthTokensRefresh sends the result for account/chatgptAuthTokens/refresh.
func (c *Client) RespondAccountChatgptAuthTokensRefresh(id json.RawMessage, accessToken, chatgptAccountID string, chatgptPlanType *string) error {
	res := map[string]any{
		"accessToken":      accessToken,
		"chatgptAccountId": chatgptAccountID,
	}
	if chatgptPlanType != nil {
		res["chatgptPlanType"] = *chatgptPlanType
	}
	return c.RespondJSONRPCResult(id, res)
}

func (c *Client) sendJSONRPCResult(id json.RawMessage, result any) error {
	v := struct {
		ID     json.RawMessage `json:"id"`
		Result any             `json:"result"`
	}{ID: id, Result: result}
	return c.send(v)
}

func (c *Client) sendJSONRPCError(id json.RawMessage, code int, message string) error {
	v := struct {
		ID    json.RawMessage `json:"id"`
		Error *rpcError       `json:"error"`
	}{ID: id, Error: &rpcError{Code: code, Message: message}}
	return c.send(v)
}

var _ agent.RPCResponder = (*Client)(nil)
