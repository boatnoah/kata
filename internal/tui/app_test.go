package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func drainAIStream(app *App, itemID string) {
	for app.aiTicking[itemID] || app.aiRendered[itemID] != app.aiStreams[itemID] {
		_ = app.handleAITick(itemID)
	}
}

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

func TestWriteCommandSendsComposeToHistory(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeNormal
	app.compose.buf = []rune("hello world")
	app.compose.cursor = len(app.compose.buf)

	// Enter command mode and issue :w
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if app.mode != ModeNormal {
		t.Fatalf("expected to return to normal mode, got %v", app.mode)
	}
	if got := app.compose.Content(); got != "" {
		t.Fatalf("expected compose to clear after write, got %q", got)
	}
	if len(app.history.messages) == 0 || app.history.messages[len(app.history.messages)-1] != "User: hello world" {
		t.Fatalf("expected history to include written message, got last %q", app.history.messages[len(app.history.messages)-1])
	}
}

func TestUpsertAIStreamPreservesDeltaSpacing(t *testing.T) {
	app := NewApp()

	app.upsertAIStream("item-1", "AI", "Hello", false)
	app.upsertAIStream("item-1", "AI", " there", false)
	app.upsertAIStream("item-1", "AI", " friend", false)
	app.upsertAIStream("item-1", "AI", "", true)
	drainAIStream(app, "item-1")

	if got := app.history.messages[len(app.history.messages)-1]; got != "AI: Hello there friend" {
		t.Fatalf("expected spaced final AI message, got %q", got)
	}
}

func TestUpsertAIStreamCompletedUsesAuthoritativeFinalText(t *testing.T) {
	app := NewApp()

	app.upsertAIStream("item-1", "AI", "Hi! ", false)
	app.upsertAIStream("item-1", "AI", "I'm here.", false)
	app.upsertAIStream("item-1", "AI", "Hi! I'm here.", true)
	drainAIStream(app, "item-1")

	if got := app.history.messages[len(app.history.messages)-1]; got != "AI: Hi! I'm here." {
		t.Fatalf("expected final AI message without duplication, got %q", got)
	}
}

func TestUpsertAIStreamDoesNotForceHistoryFollow(t *testing.T) {
	app := NewApp()
	app.history.AppendMessage("older")
	app.history.AppendMessage("newer")
	app.activePane = PaneHistory
	app.history.cursor = 0

	app.upsertAIStream("item-1", "AI", "hello", false)
	drainAIStream(app, "item-1")

	if app.history.cursor != 0 {
		t.Fatalf("expected history cursor to stay on older entry, got %d", app.history.cursor)
	}
}

func TestUpsertAIStreamRevealsTextProgressively(t *testing.T) {
	app := NewApp()

	app.upsertAIStream("item-1", "AI", "abcdef", false)

	if got := app.aiRendered["item-1"]; got == app.aiStreams["item-1"] {
		t.Fatalf("expected rendered text to lag behind full stream, got %q", got)
	}
	drainAIStream(app, "item-1")
	if got := app.aiRendered["item-1"]; got != "abcdef" {
		t.Fatalf("expected full rendered text after ticks, got %q", got)
	}
}
