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

func TestDefaultPaneIsCompose(t *testing.T) {
	app := NewApp()
	if app.activePane != PaneCompose {
		t.Fatalf("expected default active pane compose, got %v", app.activePane)
	}
	if app.mode != ModeNormal {
		t.Fatalf("expected default mode normal, got %v", app.mode)
	}
}

func TestCommandModeEntryFromNormal(t *testing.T) {
	app := NewApp()
	app.mode = ModeNormal
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	if app.mode != ModeCommandLine {
		t.Fatalf("expected command mode, got %v", app.mode)
	}
	if app.prevMode != ModeNormal {
		t.Fatalf("prev mode should be normal, got %v", app.prevMode)
	}
}

func TestCommandModeEntryFromVisual(t *testing.T) {
	app := NewApp()
	app.mode = ModeVisual
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	if app.mode != ModeCommandLine {
		t.Fatalf("expected command mode, got %v", app.mode)
	}
	if app.prevMode != ModeVisual {
		t.Fatalf("prev mode should be visual, got %v", app.prevMode)
	}
}

func TestCommandModeBlockedInInsert(t *testing.T) {
	app := NewApp()
	app.mode = ModeInsert
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	if app.mode != ModeInsert {
		t.Fatalf("insert mode should remain active, got %v", app.mode)
	}
}

func TestCommandModeCancelRestoresPrevious(t *testing.T) {
	app := NewApp()
	app.mode = ModeVisual
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if app.mode != ModeVisual {
		t.Fatalf("expected to restore visual mode, got %v", app.mode)
	}
}

func TestQuitCommand(t *testing.T) {
	app := NewApp()
	app.mode = ModeNormal
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	cmd := app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected quit msg, got %T", msg)
	}
	if app.mode != ModeNormal {
		t.Fatalf("mode should restore to previous, got %v", app.mode)
	}
}
