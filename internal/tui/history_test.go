package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHistoryVisualEnterExitAndNavigation(t *testing.T) {
	h := NewHistoryScreen()
	h.height = 4

	// enter visual
	h.EnterVisual()
	if !h.visualActive {
		t.Fatalf("expected visual active")
	}

	// move down and half-page down
	h.Move(1)
	h.MoveHalfPage(1)
	if h.cursor == 0 {
		t.Fatalf("expected cursor to move")
	}

	// jump to start/end
	h.JumpStart()
	if h.cursor != 0 {
		t.Fatalf("expected jump start to 0, got %v", h.cursor)
	}
	h.JumpEnd()
	if h.cursor != len(h.messages)-1 {
		t.Fatalf("expected jump end to last, got %v", h.cursor)
	}

	// horizontal clamps
	h.MoveLeft()
	if h.cursorCol != 0 {
		t.Fatalf("expected cursorCol clamp at 0, got %v", h.cursorCol)
	}
	h.MoveRight()
	if h.cursorCol == 0 {
		t.Fatalf("expected cursorCol to move right")
	}
	h.LineEnd()
	endCol := h.cursorCol
	h.MoveRight()
	if h.cursorCol != endCol {
		t.Fatalf("expected clamp at line end")
	}

	// selection and yank
	h.EnterVisual()
	h.Move(-1)
	h.YankSelection()
	if len(h.yankBuf) == 0 {
		t.Fatalf("expected yank to copy selection")
	}
	if h.visualActive {
		t.Fatalf("expected visual to exit after yank")
	}
}

func TestHistoryBindingsVisualAndNormal(t *testing.T) {
	app := NewApp()
	app.activePane = PaneHistory
	app.mode = ModeNormal
	app.history.ExitVisual()

	// Normal navigation
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.history.cursor != 1 {
		t.Fatalf("expected move down to 1, got %v", app.history.cursor)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if app.history.cursor != 0 {
		t.Fatalf("expected move up to 0, got %v", app.history.cursor)
	}

	// Enter visual and yank
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if app.mode != ModeVisual {
		t.Fatalf("expected visual mode")
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if app.mode != ModeNormal {
		t.Fatalf("expected normal mode after yank")
	}
	if len(app.history.yankBuf) == 0 {
		t.Fatalf("expected yank buffer to be filled")
	}
}
