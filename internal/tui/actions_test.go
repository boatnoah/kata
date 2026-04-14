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

func TestHistoryPaneHasNoVimBindings(t *testing.T) {
	bindings := defaultBindings()
	j := keystrokeFromMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if _, ok := findBinding(bindings, PaneHistory, ModeNormal, j); ok {
		t.Fatalf("history pane should not have j binding")
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
