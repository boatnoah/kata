package tui

import (
	"context"
	"sync"

	"github.com/boatnoah/kata/internal/codex"
)

// codexClient defines the subset of codex.Client we rely on; helps with tests.
type codexClient interface {
	Start(ctx context.Context) error
	SendText(ctx context.Context, text string) (string, error)
	Events() <-chan codex.Event
	Close() error
}

// AIManager owns the Codex client lifecycle and start-once behavior.
type AIManager struct {
	client    codexClient
	startOnce sync.Once
	startErr  error
}

func newAIManager() *AIManager {
	return &AIManager{client: codex.NewClient()}
}

func newAIManagerWithClient(c codexClient) *AIManager {
	return &AIManager{client: c}
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

func (m *AIManager) Events() <-chan codex.Event {
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
