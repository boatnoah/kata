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

func TestHistoryPaneNormalBindings(t *testing.T) {
	bindings := defaultBindings()
	cases := []struct {
		key    tea.KeyMsg
		action ActionID
	}{
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, ActionHistoryCursorDown},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, ActionHistoryCursorUp},
		{tea.KeyMsg{Type: tea.KeyCtrlU, Runes: nil}, ActionScrollHalfPageUp},
		{tea.KeyMsg{Type: tea.KeyCtrlD, Runes: nil}, ActionScrollHalfPageDown},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}, ActionHistoryCursorBottom},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}, ActionHistoryEnterVisual},
	}
	for _, c := range cases {
		ks := keystrokeFromMsg(c.key)
		action, ok := findBinding(bindings, PaneHistory, ModeNormal, ks)
		if !ok {
			t.Fatalf("missing history binding for %+v", ks)
		}
		if action != c.action {
			t.Fatalf("for %+v got %v, want %v", ks, action, c.action)
		}
	}
}

func TestHistoryPaneVisualBindings(t *testing.T) {
	bindings := defaultBindings()
	cases := []struct {
		key    tea.KeyMsg
		action ActionID
	}{
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, ActionHistoryCursorDown},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, ActionHistoryCursorUp},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, ActionHistoryYank},
		{tea.KeyMsg{Type: tea.KeyEsc, Runes: nil}, ActionEnterNormal},
	}
	for _, c := range cases {
		ks := keystrokeFromMsg(c.key)
		action, ok := findBinding(bindings, PaneHistory, ModeVisual, ks)
		if !ok {
			t.Fatalf("missing history visual binding for %+v", ks)
		}
		if action != c.action {
			t.Fatalf("for %+v got %v, want %v", ks, action, c.action)
		}
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
