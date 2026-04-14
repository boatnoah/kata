package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/boatnoah/kata/internal/codex"
)

func drainAIStream(app *App, itemID string) {
	for app.aiTicking[itemID] || app.aiRendered[itemID] != app.aiStreams[itemID] {
		_ = app.handleAITick(itemID)
	}
}

func lastHistoryItem(t *testing.T, app *App) TranscriptItem {
	t.Helper()
	if len(app.history.items) == 0 {
		t.Fatalf("expected history item")
	}
	return app.history.items[len(app.history.items)-1]
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
	app.activePane = PaneCompose
	app.mode = ModeVisual
	app.compose.EnterVisual()
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	// enterCommandLine clears compose visual, so exiting command mode
	// falls back to normal instead of restoring orphaned visual mode.
	if app.mode != ModeNormal {
		t.Fatalf("expected to restore normal mode (visual was cleared), got %v", app.mode)
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
	item := lastHistoryItem(t, app)
	if item.Kind != TranscriptUser || item.Text != "hello world" {
		t.Fatalf("expected user transcript item, got %+v", item)
	}
}

func TestUpsertAIStreamPreservesDeltaSpacing(t *testing.T) {
	app := NewApp()

	app.upsertAIStream("item-1", TranscriptAssistant, "Hello", false)
	app.upsertAIStream("item-1", TranscriptAssistant, " there", false)
	app.upsertAIStream("item-1", TranscriptAssistant, " friend", false)
	app.upsertAIStream("item-1", TranscriptAssistant, "", true)
	drainAIStream(app, "item-1")

	item := lastHistoryItem(t, app)
	if item.Kind != TranscriptAssistant || item.Text != "Hello there friend" {
		t.Fatalf("expected spaced final AI message, got %+v", item)
	}
}

func TestUpsertAIStreamCompletedUsesAuthoritativeFinalText(t *testing.T) {
	app := NewApp()

	app.upsertAIStream("item-1", TranscriptAssistant, "Hi! ", false)
	app.upsertAIStream("item-1", TranscriptAssistant, "I'm here.", false)
	app.upsertAIStream("item-1", TranscriptAssistant, "Hi! I'm here.", true)
	drainAIStream(app, "item-1")

	item := lastHistoryItem(t, app)
	if item.Kind != TranscriptAssistant || item.Text != "Hi! I'm here." {
		t.Fatalf("expected final AI message without duplication, got %+v", item)
	}
}


func TestUpsertAIStreamRevealsTextProgressively(t *testing.T) {
	app := NewApp()

	app.upsertAIStream("item-1", TranscriptAssistant, "abcdef", false)

	if got := app.aiRendered["item-1"]; got == app.aiStreams["item-1"] {
		t.Fatalf("expected rendered text to lag behind full stream, got %q", got)
	}
	drainAIStream(app, "item-1")
	if got := app.aiRendered["item-1"]; got != "abcdef" {
		t.Fatalf("expected full rendered text after ticks, got %q", got)
	}
}

func TestRenderAIStreamShowsWaitingStatusWhenCaughtUp(t *testing.T) {
	app := NewApp()
	app.aiTypes["item-1"] = TranscriptAssistant
	app.aiStreams["item-1"] = "hello"
	app.aiRendered["item-1"] = "hello"

	app.renderAIStream("item-1")

	item := lastHistoryItem(t, app)
	if item.Kind != TranscriptAssistant || item.Text != "hello" || item.Status == "" {
		t.Fatalf("expected assistant item with waiting status, got %+v", item)
	}
}

func TestSummarizeToolCallCollapsesReadActivity(t *testing.T) {
	ev := codex.Event{
		Payload: map[string]any{"name": "read"},
		Text:    "go.mod\nmodule github.com/boatnoah/kata\n",
	}

	got := summarizeToolCall(ev)
	if got != "Read go.mod" {
		t.Fatalf("expected compact read summary, got %q", got)
	}
}

func TestHandleCodexEventSkipsCommandOutput(t *testing.T) {
	app := NewApp()
	before := len(app.history.items)

	cmd := app.handleCodexEvent(codex.Event{Type: codex.EventCommandOutput, ItemID: "cmd-1", Text: "package main\nfunc main() {}"})

	if cmd != nil {
		t.Fatalf("expected no command output render cmd, got %v", cmd)
	}
	if len(app.history.items) != before {
		t.Fatalf("expected command output to be suppressed")
	}
}

