package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLeaderFocusCompose(t *testing.T) {
	app := NewApp()
	// Begin leader
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	// Leader + j
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.activePane != PaneCompose {
		t.Fatalf("expected compose pane after leader+j, got %v", app.activePane)
	}
}

func TestLeaderIgnoresInInsertMode(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeInsert
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if app.leaderPending {
		t.Fatalf("leader should not arm in insert mode")
	}
}
