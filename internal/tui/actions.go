package tui

import tea "github.com/charmbracelet/bubbletea"

type ActionID int

const (
	ActionQuit ActionID = iota
	ActionSwitchPane
	ActionEnterInsert
	ActionEnterInsertLineStart
	ActionEnterAppend
	ActionEnterAppendLineEnd
	ActionEnterOpenBelow
	ActionEnterOpenAbove
	ActionEnterNormal
	ActionEnterVisual
	ActionEnterCommandLine
	ActionMoveLeft
	ActionMoveRight
	ActionMoveUpCompose
	ActionMoveDownCompose
	ActionMoveWordFwd
	ActionMoveWordBack
	ActionLineStart
	ActionLineStartNonSpace
	ActionLineEnd
	ActionDeleteLine
	ActionDeleteChar
	ActionDeleteToEOL
	ActionDeleteSelection
	ActionYankSelection
	ActionPasteAfter
	ActionPasteBefore
	ActionFocusCompose
	ActionFocusHistory
)

type KeyStroke struct {
	Type  tea.KeyType
	Runes string
}

type Binding struct {
	Pane   Pane
	Mode   Mode
	Key    KeyStroke
	Action ActionID
}

func keystrokeFromMsg(msg tea.KeyMsg) KeyStroke {
	ks := KeyStroke{Type: msg.Type}
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		ks.Runes = string(msg.Runes)
	}
	return ks
}

func keystrokeEqual(a, b KeyStroke) bool {
	if a.Type != b.Type {
		return false
	}
	if b.Runes == "" {
		return true
	}
	return a.Runes == b.Runes
}

func defaultBindings() []Binding {
	return []Binding{
		// Global
		{Pane: PaneAny, Mode: ModeAny, Key: KeyStroke{Type: tea.KeyCtrlC}, Action: ActionQuit},
		{Pane: PaneAny, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: ":"}, Action: ActionEnterCommandLine},
		{Pane: PaneAny, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: ":"}, Action: ActionEnterCommandLine},

		// Compose - Normal mode
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "i"}, Action: ActionEnterInsert},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "I"}, Action: ActionEnterInsertLineStart},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "a"}, Action: ActionEnterAppend},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "A"}, Action: ActionEnterAppendLineEnd},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "o"}, Action: ActionEnterOpenBelow},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "O"}, Action: ActionEnterOpenAbove},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "v"}, Action: ActionEnterVisual},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyEsc}, Action: ActionEnterNormal},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "h"}, Action: ActionMoveLeft},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "l"}, Action: ActionMoveRight},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "j"}, Action: ActionMoveDownCompose},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "k"}, Action: ActionMoveUpCompose},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "w"}, Action: ActionMoveWordFwd},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "b"}, Action: ActionMoveWordBack},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "0"}, Action: ActionLineStart},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "^"}, Action: ActionLineStartNonSpace},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "$"}, Action: ActionLineEnd},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "x"}, Action: ActionDeleteChar},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "D"}, Action: ActionDeleteLine},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "p"}, Action: ActionPasteAfter},
		{Pane: PaneCompose, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyRunes, Runes: "P"}, Action: ActionPasteBefore},

		// Compose - Insert mode
		{Pane: PaneCompose, Mode: ModeInsert, Key: KeyStroke{Type: tea.KeyEsc}, Action: ActionEnterNormal},

		// Compose - Visual mode
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "d"}, Action: ActionDeleteSelection},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "y"}, Action: ActionYankSelection},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "p"}, Action: ActionPasteAfter},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "P"}, Action: ActionPasteBefore},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyEsc}, Action: ActionEnterNormal},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "h"}, Action: ActionMoveLeft},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "l"}, Action: ActionMoveRight},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "j"}, Action: ActionMoveDownCompose},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "k"}, Action: ActionMoveUpCompose},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "0"}, Action: ActionLineStart},
		{Pane: PaneCompose, Mode: ModeVisual, Key: KeyStroke{Type: tea.KeyRunes, Runes: "$"}, Action: ActionLineEnd},

		// History - Normal mode (view-only, no vim navigation)
		{Pane: PaneHistory, Mode: ModeNormal, Key: KeyStroke{Type: tea.KeyEsc}, Action: ActionEnterNormal},
	}
}

func findBinding(bindings []Binding, pane Pane, mode Mode, ks KeyStroke) (ActionID, bool) {
	for _, b := range bindings {
		if b.Pane != PaneAny && b.Pane != pane {
			continue
		}
		if b.Mode != ModeAny && b.Mode != mode {
			continue
		}
		if keystrokeEqual(b.Key, ks) {
			return b.Action, true
		}
	}
	return 0, false
}
func leaderBindings() []Binding {
	return []Binding{
		{Pane: PaneAny, Mode: ModeAny, Key: KeyStroke{Type: tea.KeyRunes, Runes: "j"}, Action: ActionFocusCompose},
		{Pane: PaneAny, Mode: ModeAny, Key: KeyStroke{Type: tea.KeyRunes, Runes: "k"}, Action: ActionFocusHistory},
	}
}
