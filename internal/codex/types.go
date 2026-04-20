package codex

import (
	"encoding/json"
	"strings"
)

type rpcMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type rpcRequest struct {
	ID     int64  `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type rpcNotify struct {
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type initializeParams struct {
	ClientInfo   clientInfo    `json:"clientInfo"`
	Capabilities *capabilities `json:"capabilities,omitempty"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

type capabilities struct {
	ExperimentalAPI         bool     `json:"experimentalApi,omitempty"`
	OptOutNotificationNames []string `json:"optOutNotificationMethods,omitempty"`
}

type threadStartParams struct {
	Model string `json:"model,omitempty"`
}

type thread struct {
	ID string `json:"id"`
}

type threadStartResult struct {
	Thread thread `json:"thread"`
}

type turnStartParams struct {
	ThreadID string      `json:"threadId"`
	Input    []turnInput `json:"input"`
}

type turnInput struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type turn struct {
	ID     string     `json:"id"`
	Status string     `json:"status,omitempty"`
	Error  *turnError `json:"error,omitempty"`
}

type turnError struct {
	Code        string `json:"code,omitempty"`
	Message     string `json:"message,omitempty"`
	HTTPStatus  int    `json:"httpStatusCode,omitempty"`
	Description string `json:"description,omitempty"`
}

type turnStartResult struct {
	Turn turn `json:"turn"`
}

type turnStartedParams struct {
	ThreadID string `json:"threadId"`
	Turn     turn   `json:"turn"`
}

type turnCompletedParams struct {
	ThreadID string `json:"threadId"`
	Turn     turn   `json:"turn"`
}

type textBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type agentMessageParams struct {
	ThreadID string           `json:"threadId,omitempty"`
	TurnID   string           `json:"turnId,omitempty"`
	Item     agentMessageItem `json:"item"`
}

type agentMessageDeltaParams struct {
	ThreadID string `json:"threadId,omitempty"`
	TurnID   string `json:"turnId,omitempty"`
	ItemID   string `json:"itemId,omitempty"`
	Delta    string `json:"delta,omitempty"`
}

type agentMessageItem struct {
	ID           string      `json:"id"`
	Content      []textBlock `json:"content,omitempty"`
	ContentDelta []textBlock `json:"contentDelta,omitempty"`
}

func (i agentMessageItem) Text() string {
	var b strings.Builder
	for _, c := range i.ContentDelta {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	if b.Len() > 0 {
		return b.String()
	}
	for _, c := range i.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

type toolCallParams struct {
	ThreadID string       `json:"threadId,omitempty"`
	TurnID   string       `json:"turnId,omitempty"`
	Item     toolCallItem `json:"item"`
}

type toolCallItem struct {
	ID           string      `json:"id"`
	Name         string      `json:"name,omitempty"`
	CallID       string      `json:"callId,omitempty"`
	ContentDelta []textBlock `json:"contentDelta,omitempty"`
	OutputDelta  string      `json:"outputDelta,omitempty"`
}

func (i toolCallItem) Text() string {
	if i.OutputDelta != "" {
		return i.OutputDelta
	}
	var b strings.Builder
	for _, c := range i.ContentDelta {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

type commandExecParams struct {
	ThreadID string          `json:"threadId,omitempty"`
	TurnID   string          `json:"turnId,omitempty"`
	Item     commandExecItem `json:"item"`
}

type commandExecOutputDeltaParams struct {
	ThreadID string `json:"threadId,omitempty"`
	TurnID   string `json:"turnId,omitempty"`
	ItemID   string `json:"itemId,omitempty"`
	Delta    string `json:"delta,omitempty"`
}

type commandExecItem struct {
	ID          string   `json:"id"`
	Command     string   `json:"command,omitempty"`
	CommandArr  []string `json:"commandArray,omitempty"`
	Stream      string   `json:"stream,omitempty"`
	OutputDelta string   `json:"outputDelta,omitempty"`
	Output      string   `json:"aggregatedOutput,omitempty"`
	ExitCode    *int     `json:"exitCode,omitempty"`
	Status      string   `json:"status,omitempty"`
	Type        string   `json:"type,omitempty"`
}

func (i commandExecItem) CommandLine() string {
	if i.Command != "" {
		return i.Command
	}
	if len(i.CommandArr) > 0 {
		return strings.Join(i.CommandArr, " ")
	}
	return ""
}

type itemCompletedParams struct {
	ThreadID string              `json:"threadId,omitempty"`
	TurnID   string              `json:"turnId,omitempty"`
	Item     completedThreadItem `json:"item"`
}

type tokenUsageUpdatedParams struct {
	ThreadID   string           `json:"threadId"`
	TurnID     string           `json:"turnId,omitempty"`
	TokenUsage threadTokenUsage `json:"tokenUsage"`
}

type threadTokenUsage struct {
	Last               tokenUsageBreakdown `json:"last"`
	Total              tokenUsageBreakdown `json:"total"`
	ModelContextWindow *int64              `json:"modelContextWindow,omitempty"`
}

type tokenUsageBreakdown struct {
	CachedInputTokens     int64 `json:"cachedInputTokens"`
	InputTokens           int64 `json:"inputTokens"`
	OutputTokens          int64 `json:"outputTokens"`
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens"`
	TotalTokens           int64 `json:"totalTokens"`
}

type completedThreadItem struct {
	ID         string   `json:"id"`
	Type       string   `json:"type,omitempty"`
	Text       string   `json:"text,omitempty"`
	Command    string   `json:"command,omitempty"`
	CommandArr []string `json:"commandArray,omitempty"`
	Output     string   `json:"aggregatedOutput,omitempty"`
	Stream     string   `json:"stream,omitempty"`
	ExitCode   *int     `json:"exitCode,omitempty"`
	Status     string   `json:"status,omitempty"`
}

func (i completedThreadItem) CommandLine() string {
	if i.Command != "" {
		return i.Command
	}
	if len(i.CommandArr) > 0 {
		return strings.Join(i.CommandArr, " ")
	}
	return ""
}
