package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/boatnoah/kata/internal/agent"
)

// Client manages a Codex app-server subprocess over JSON-RPC (JSONL over stdio).
// It performs the initialize/initialized handshake, starts or resumes a thread,
// and streams notifications (agent thinking, tool calls, command output, etc.)
// to callers via the Events channel.
type Client struct {
	cmdPath string
	cmdArgs []string
	model   string

	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	cmd    *exec.Cmd

	nextID   int64
	pending  map[int64]chan rpcMessage
	pendMu   sync.Mutex
	writeMu  sync.Mutex
	eventsCh chan agent.Event

	threadID string
	started  atomic.Bool

	closeFn func() error
}

// Option customizes the Codex client.
type Option func(*Client)

// WithCmd sets the codex binary path and args (default: "codex", "app-server").
func WithCmd(path string, args ...string) Option {
	return func(c *Client) {
		c.cmdPath = path
		c.cmdArgs = append([]string{}, args...)
	}
}

// WithModel sets the default model for thread/start (default: "gpt-5.4").
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// NewClient builds a Client; call Start to spawn and handshake.
func NewClient(opts ...Option) *Client {
	c := &Client{
		cmdPath:  "codex",
		cmdArgs:  []string{"app-server"},
		model:    "gpt-5.4",
		pending:  make(map[int64]chan rpcMessage),
		eventsCh: make(chan agent.Event, 128),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Start launches the app-server, performs handshake, and starts a thread.
func (c *Client) Start(ctx context.Context) error {
	if c.started.Load() {
		return nil
	}
	// The Codex subprocess is long-lived and must survive individual request
	// timeouts. Use per-RPC contexts for initialization/turn calls, but do not
	// tie the process lifetime itself to a request-scoped context.
	cmd := exec.Command(c.cmdPath, c.cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("codex stdout: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("codex stdin: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("codex stderr: %w", err)
	}
	c.cmd = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	c.closeFn = cmd.Process.Kill

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("codex start: %w", err)
	}

	go c.readLoop()
	go c.readStderr()

	if err := c.initialize(ctx); err != nil {
		return err
	}
	if err := c.startThread(ctx); err != nil {
		return err
	}
	c.started.Store(true)
	return nil
}

// Close terminates the process and closes streams.
func (c *Client) Close() error {
	var errs []string
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.stdout != nil {
		_ = c.stdout.Close()
	}
	if c.stderr != nil {
		_ = c.stderr.Close()
	}
	if c.cmd != nil {
		_ = c.cmd.Wait()
	}
	if c.closeFn != nil {
		if err := c.closeFn(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// Events returns a channel of streaming events (agent deltas, tool calls, etc.).
func (c *Client) Events() <-chan agent.Event { return c.eventsCh }

// Model reports the model the client was configured with. This is what
// thread/start was (or will be) called with — the UI uses it to label the
// active session.
func (c *Client) Model() string { return c.model }

// SendText starts a turn with user text and streams responses via Events.
func (c *Client) SendText(ctx context.Context, text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", nil
	}
	turn, err := c.startTurn(ctx, trimmed)
	if err != nil {
		return "", err
	}
	return turn, nil
}

// ---------------- internal plumbing ----------------

func (c *Client) initialize(ctx context.Context) error {
	params := initializeParams{
		ClientInfo: clientInfo{Name: "kata_tui", Title: "Kata TUI", Version: "0.1.0"},
	}
	var res map[string]any
	if err := c.call(ctx, "initialize", params, &res); err != nil {
		return err
	}
	// Acknowledge initialization.
	if err := c.notify("initialized", map[string]any{}); err != nil {
		return err
	}
	return nil
}

func (c *Client) startThread(ctx context.Context) error {
	params := threadStartParams{Model: c.model}
	var res threadStartResult
	if err := c.call(ctx, "thread/start", params, &res); err != nil {
		return err
	}
	c.threadID = res.Thread.ID
	return nil
}

func (c *Client) startTurn(ctx context.Context, text string) (string, error) {
	if c.threadID == "" {
		return "", fmt.Errorf("codex: thread not initialized")
	}
	params := turnStartParams{
		ThreadID: c.threadID,
		Input:    []turnInput{{Type: "text", Text: text}},
	}
	var res turnStartResult
	if err := c.call(ctx, "turn/start", params, &res); err != nil {
		return "", err
	}
	return res.Turn.ID, nil
}

func (c *Client) readLoop() {
	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		var msg rpcMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			c.emitError(fmt.Errorf("decode message: %w", err))
			continue
		}
		if len(msg.ID) > 0 {
			c.dispatchResponse(msg)
			continue
		}
		if msg.Method != "" {
			c.handleNotification(msg)
		}
	}
	if err := scanner.Err(); err != nil {
		c.emitError(err)
	}
}

func (c *Client) readStderr() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		c.emitError(fmt.Errorf("codex stderr: %s", line))
	}
	if err := scanner.Err(); err != nil {
		c.emitError(fmt.Errorf("stderr read: %w", err))
	}
}

func (c *Client) dispatchResponse(msg rpcMessage) {
	id, err := parseID(msg.ID)
	if err != nil {
		c.emitError(fmt.Errorf("parse id: %w", err))
		return
	}
	c.pendMu.Lock()
	ch := c.pending[id]
	delete(c.pending, id)
	c.pendMu.Unlock()
	if ch != nil {
		ch <- msg
	}
}

func (c *Client) handleNotification(msg rpcMessage) {
	switch {
	case msg.Method == "turn/started":
		var params turnStartedParams
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			c.emit(agent.Event{Type: agent.EventTurnStarted, ThreadID: params.ThreadID, TurnID: params.Turn.ID, ItemID: params.Turn.ID})
		}
	case msg.Method == "turn/completed":
		var params turnCompletedParams
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			payload := map[string]any{"status": params.Turn.Status}
			if params.Turn.Error != nil {
				payload["error"] = params.Turn.Error
			}
			c.emit(agent.Event{Type: agent.EventTurnCompleted, ThreadID: params.ThreadID, TurnID: params.Turn.ID, ItemID: params.Turn.ID, Payload: payload})
		}
	case msg.Method == "item/agentMessage/delta":
		var params agentMessageDeltaParams
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			c.emit(agent.Event{Type: agent.EventAgentDelta, ThreadID: params.ThreadID, TurnID: params.TurnID, ItemID: params.ItemID, Text: params.Delta})
		}
	case strings.HasPrefix(msg.Method, "item/agentMessage/"):
		var params agentMessageParams
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			text := params.Item.Text()
			etype := agent.EventAgentDelta
			if strings.HasSuffix(msg.Method, "completed") {
				etype = agent.EventAgentCompleted
			}
			c.emit(agent.Event{Type: etype, ThreadID: params.ThreadID, TurnID: params.TurnID, ItemID: params.Item.ID, Text: text})
		}
	case strings.HasPrefix(msg.Method, "item/toolCall/"):
		var params toolCallParams
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			c.emit(agent.Event{Type: agent.EventToolCall, ThreadID: params.ThreadID, TurnID: params.TurnID, ItemID: params.Item.ID, Text: params.Item.Text(), Payload: map[string]any{"name": params.Item.Name}})
		}
	case strings.HasPrefix(msg.Method, "item/commandExecution/"):
		if msg.Method == "item/commandExecution/outputDelta" {
			var params commandExecOutputDeltaParams
			if err := json.Unmarshal(msg.Params, &params); err == nil {
				c.emit(agent.Event{Type: agent.EventCommandOutput, ThreadID: params.ThreadID, TurnID: params.TurnID, ItemID: params.ItemID, Text: params.Delta, Payload: map[string]any{"phase": "output"}})
				return
			}
		}
		var params commandExecParams
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			phase := strings.TrimPrefix(msg.Method, "item/commandExecution/")
			payload := map[string]any{
				"phase":   phase,
				"stream":  params.Item.Stream,
				"command": params.Item.CommandLine(),
			}
			if len(params.Item.CommandArr) > 0 {
				payload["commandArgs"] = append([]string(nil), params.Item.CommandArr...)
			}
			if params.Item.ExitCode != nil {
				payload["exitCode"] = *params.Item.ExitCode
			}
			if params.Item.Status != "" {
				payload["status"] = params.Item.Status
			}
			c.emit(agent.Event{Type: agent.EventCommandOutput, ThreadID: params.ThreadID, TurnID: params.TurnID, ItemID: params.Item.ID, Text: params.Item.OutputDelta, Payload: payload})
		}
	case msg.Method == "thread/tokenUsage/updated":
		var params tokenUsageUpdatedParams
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			payload := map[string]any{
				"used":               int(params.TokenUsage.Last.InputTokens),
				"lastTotalTokens":    int(params.TokenUsage.Last.TotalTokens),
				"sessionTotalTokens": int(params.TokenUsage.Total.TotalTokens),
			}
			if params.TokenUsage.ModelContextWindow != nil {
				payload["contextWindow"] = int(*params.TokenUsage.ModelContextWindow)
			}
			c.emit(agent.Event{Type: agent.EventTokenUsage, ThreadID: params.ThreadID, TurnID: params.TurnID, Payload: payload})
		}
	case msg.Method == "item/completed":
		var params itemCompletedParams
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			switch params.Item.Type {
			case "agentMessage":
				c.emit(agent.Event{Type: agent.EventAgentCompleted, ThreadID: params.ThreadID, TurnID: params.TurnID, ItemID: params.Item.ID, Text: params.Item.Text})
			case "commandExecution":
				payload := map[string]any{
					"phase":   "completed",
					"stream":  params.Item.Stream,
					"command": params.Item.CommandLine(),
				}
				if len(params.Item.CommandArr) > 0 {
					payload["commandArgs"] = append([]string(nil), params.Item.CommandArr...)
				}
				if params.Item.ExitCode != nil {
					payload["exitCode"] = *params.Item.ExitCode
				}
				if params.Item.Status != "" {
					payload["status"] = params.Item.Status
				}
				c.emit(agent.Event{Type: agent.EventCommandOutput, ThreadID: params.ThreadID, TurnID: params.TurnID, ItemID: params.Item.ID, Text: params.Item.Output, Payload: payload})
			}
		}
	}
}

func (c *Client) emit(ev agent.Event) {
	select {
	case c.eventsCh <- ev:
	default:
		// drop if slow consumer to avoid blocking the transport loop
	}
}

func (c *Client) emitError(err error) {
	if err == nil {
		return
	}
	c.emit(agent.Event{Type: agent.EventError, Payload: map[string]any{"error": err.Error()}})
}

func (c *Client) call(ctx context.Context, method string, params any, out any) error {
	id := atomic.AddInt64(&c.nextID, 1)
	req := rpcRequest{ID: id, Method: method, Params: params}
	if err := c.send(req); err != nil {
		return err
	}
	ch := make(chan rpcMessage, 1)
	c.pendMu.Lock()
	c.pending[id] = ch
	c.pendMu.Unlock()
	select {
	case <-ctx.Done():
		c.removePending(id)
		return ctx.Err()
	case msg := <-ch:
		if msg.Error != nil {
			return fmt.Errorf("rpc error %d: %s", msg.Error.Code, msg.Error.Message)
		}
		if out != nil && len(msg.Result) > 0 {
			if err := json.Unmarshal(msg.Result, out); err != nil {
				return err
			}
		}
		return nil
	}
}

func (c *Client) removePending(id int64) {
	c.pendMu.Lock()
	delete(c.pending, id)
	c.pendMu.Unlock()
}

func (c *Client) notify(method string, params any) error {
	n := rpcNotify{Method: method, Params: params}
	return c.send(n)
}

func (c *Client) send(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	c.writeMu.Lock()
	_, err = c.stdin.Write(data)
	c.writeMu.Unlock()
	return err
}

func parseID(raw json.RawMessage) (int64, error) {
	var num json.Number
	if err := json.Unmarshal(raw, &num); err != nil {
		return 0, err
	}
	return num.Int64()
}

// jitteredBackoff returns a duration with random jitter.
