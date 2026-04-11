package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCtrlINotBoundToSwitch(t *testing.T) {
	bindings := defaultBindings()
	ks := keystrokeFromMsg(tea.KeyMsg{Type: tea.KeyTab, Runes: []rune{'\t'}})
	if _, ok := findBinding(bindings, PaneHistory, ModeNormal, ks); ok {
		t.Fatalf("ctrl+i/tab should not be bound, got %+v", ks)
	}
}

func TestPlainJKStillNavigate(t *testing.T) {
	bindings := defaultBindings()
	j := keystrokeFromMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if action, ok := findBinding(bindings, PaneHistory, ModeNormal, j); !ok || action != ActionMoveDown {
		t.Fatalf("plain j should move down, got %v, ok=%v", action, ok)
	}
	k := keystrokeFromMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if action, ok := findBinding(bindings, PaneHistory, ModeNormal, k); !ok || action != ActionMoveUp {
		t.Fatalf("plain k should move up, got %v, ok=%v", action, ok)
	}
}

func TestRuneBindingsPreserved(t *testing.T) {
	bindings := defaultBindings()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	ks := keystrokeFromMsg(msg)
	action, ok := findBinding(bindings, PaneCompose, ModeNormal, ks)
	if !ok {
		t.Fatalf("binding not found: %+v", ks)
	}
	if action != ActionEnterInsert {
		t.Fatalf("got %v, want %v", action, ActionEnterInsert)
	}
}
