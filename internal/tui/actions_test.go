package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCtrlBindingsIgnoreRunes(t *testing.T) {
	bindings := defaultBindings()
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want ActionID
	}{
		{"ctrl-j focuses compose", tea.KeyMsg{Type: tea.KeyCtrlJ, Runes: []rune{'j'}}, ActionFocusCompose},
		{"ctrl-k focuses history", tea.KeyMsg{Type: tea.KeyCtrlK, Runes: []rune{'k'}}, ActionFocusHistory},
		{"ctrl-j as lf rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\n'}}, ActionFocusCompose},
		{"ctrl-j as enter key", tea.KeyMsg{Type: tea.KeyEnter}, ActionFocusCompose},
		{"ctrl-j as cr rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\r'}}, ActionFocusCompose},
		{"ctrl-k as vt rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\v'}}, ActionFocusHistory},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ks := keystrokeFromMsg(tc.msg)
			action, ok := findBinding(bindings, PaneHistory, ModeNormal, ks)
			if !ok {
				t.Fatalf("binding not found: %+v", ks)
			}
			if action != tc.want {
				t.Fatalf("got %v, want %v", action, tc.want)
			}
		})
	}
}

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
