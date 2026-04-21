package tui

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/boatnoah/kata/internal/agent"
	"github.com/boatnoah/kata/internal/codex"
)

// AIManager owns an agent provider's lifecycle and start-once behavior.
// It is the single place the UI asks "what AI am I talking to?" — today
// that's always Codex, but Provider()/Model() give us a seam to grow into
// additional backends without touching the statusline.
type AIManager struct {
	provider  string
	client    agent.Provider
	rpc       agent.RPCResponder
	startOnce sync.Once
	startErr  error
}

func newAIManager() *AIManager {
	c := codex.NewClient()
	m := &AIManager{provider: "codex", client: c}
	if rr, ok := any(c).(agent.RPCResponder); ok {
		m.rpc = rr
	}
	return m
}

func newAIManagerWithClient(c agent.Provider) *AIManager {
	m := &AIManager{provider: "codex", client: c}
	if rr, ok := any(c).(agent.RPCResponder); ok {
		m.rpc = rr
	}
	return m
}

// Provider reports the backend label shown in the statusline (e.g. "codex").
func (m *AIManager) Provider() string { return m.provider }

// Model reports the underlying client's configured model, or "" if unset.
func (m *AIManager) Model() string {
	if m.client == nil {
		return ""
	}
	return m.client.Model()
}

func (m *AIManager) Start(ctx context.Context) error {
	m.startOnce.Do(func() {
		if m.client != nil {
			m.startErr = m.client.Start(ctx)
		}
	})
	return m.startErr
}

func (m *AIManager) SendText(ctx context.Context, text string) error {
	if err := m.Start(ctx); err != nil {
		return err
	}
	if m.client == nil {
		return nil
	}
	_, err := m.client.SendText(ctx, text)
	return err
}

func (m *AIManager) Events() <-chan agent.Event {
	if m.client == nil {
		return nil
	}
	return m.client.Events()
}

func (m *AIManager) Close() error {
	if m.client == nil {
		return nil
	}
	return m.client.Close()
}

// RespondServerRPC forwards a JSON-RPC result for a pending server request
// (see agent.Event.RPCID). Returns agent.ErrRPCResponderUnsupported when the
// backend cannot reply on the wire.
func (m *AIManager) RespondServerRPC(ctx context.Context, id json.RawMessage, result any) error {
	if m.rpc == nil {
		return agent.ErrRPCResponderUnsupported
	}
	return m.rpc.RespondServerRPC(ctx, id, result)
}
