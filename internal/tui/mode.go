package tui

// Mode represents the current interaction mode of the TUI.
type Mode int

const (
	ModeAny    Mode = -1
	ModeNormal Mode = iota
	ModeInsert
	ModeCommandLine
	ModeVisual
)

// Pane represents the currently focused pane.
type Pane int

const (
	PaneAny     Pane = -1
	PaneHistory Pane = iota
	PaneCompose
)
