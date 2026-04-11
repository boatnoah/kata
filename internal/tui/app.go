package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// App is the root Bubble Tea model.
type App struct {
	mode          Mode
	prevMode      Mode
	activePane    Pane
	history       *HistoryScreen
	compose       *Compose
	command       *CommandLine
	bindings      []Binding
	leaderPending bool
	deletePending bool
	width         int
	height        int
}

// NewApp constructs the application with history + compose panes.
func NewApp() *App {
	return &App{
		mode:       ModeNormal,
		prevMode:   ModeNormal,
		activePane: PaneCompose,
		history:    NewHistoryScreen(),
		compose:    NewCompose(),
		command:    NewCommandLine(),
		bindings:   defaultBindings(),
	}
}

var _ tea.Model = (*App)(nil)

func (a *App) Init() tea.Cmd {
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height
		a.history.OnWindowSize(m.Width, m.Height)
		a.compose.OnWindowSize(m.Width, m.Height)
		return a, nil
	case tea.KeyMsg:
		logKey(m)
		return a, a.handleKey(m)
	default:
		return a, nil
	}
}

func (a *App) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Handle pending delete (for dd in compose normal mode).
	if a.activePane == PaneCompose && a.mode == ModeNormal && a.deletePending {
		a.deletePending = false
		if msg.Type == tea.KeyRunes && string(msg.Runes) == "d" {
			a.compose.DeleteCurrentLine()
			return nil
		}
		// fallthrough to normal handling
	}

	if a.activePane == PaneCompose && a.mode == ModeNormal {
		if msg.Type == tea.KeyRunes && string(msg.Runes) == "d" {
			a.deletePending = true
			return nil
		}
	}
	// Command-line mode bypasses other bindings and leader logic.
	if a.mode == ModeCommandLine {
		execute, cancel, input := a.command.HandleKey(msg)
		if cancel {
			a.exitCommandLine()
			return nil
		}
		if execute {
			return a.runCommand(input)
		}
		return nil
	}

	// If a leader key was pressed previously, interpret this key using leader bindings.
	if a.leaderPending {
		a.leaderPending = false
		ks := keystrokeFromMsg(msg)
		if action, ok := findBinding(leaderBindings(), PaneAny, ModeAny, ks); ok {
			return a.applyAction(action)
		}
		// fallthrough to normal handling if leader combo not recognized
	}

	// Start leader sequence when not in insert mode.
	if a.mode != ModeInsert && isLeaderMsg(msg) {
		a.leaderPending = true
		return nil
	}

	ks := keystrokeFromMsg(msg)
	if action, ok := findBinding(a.bindings, a.activePane, a.mode, ks); ok {
		return a.applyAction(action)
	}

	// Compose insert-mode typing fallback.
	if a.activePane == PaneCompose && a.mode == ModeInsert {
		if a.handleComposeInsertKey(msg) {
			return nil
		}
	}

	return nil
}

// isLeaderMsg reports whether the key should start a leader sequence.
// Accept both a literal space rune and Bubble Tea's space key type.
func isLeaderMsg(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeySpace {
		return true
	}
	if msg.Type == tea.KeyRunes && string(msg.Runes) == " " {
		return true
	}
	return false
}

func (a *App) applyAction(action ActionID) tea.Cmd {
	switch action {
	case ActionQuit:
		return tea.Quit
	case ActionSwitchPane:
		a.switchPane()
	case ActionFocusCompose:
		a.activePane = PaneCompose
		a.ensureModeSupported()
	case ActionFocusHistory:
		a.activePane = PaneHistory
		a.ensureModeSupported()
	case ActionEnterInsert:
		if a.activePane == PaneCompose {
			a.mode = ModeInsert
			a.compose.exitVisualIfActive()
		}
	case ActionEnterNormal:
		a.mode = ModeNormal
		a.compose.exitVisualIfActive()
	case ActionEnterVisual:
		if a.activePane == PaneCompose {
			a.mode = ModeVisual
			a.compose.EnterVisual()
		} else {
			a.mode = ModeVisual
		}
	case ActionEnterCommandLine:
		a.enterCommandLine()
	case ActionMoveUpCompose:
		if a.activePane == PaneCompose {
			a.compose.MoveUp()
		}
	case ActionMoveDownCompose:
		if a.activePane == PaneCompose {
			a.compose.MoveDown()
		}
	case ActionDeleteSelection:
		if a.activePane == PaneCompose {
			a.compose.DeleteSelection()
			a.mode = ModeNormal
		}
	case ActionYankSelection:
		if a.activePane == PaneCompose {
			a.compose.YankSelection()
			a.mode = ModeNormal
		}
	case ActionPasteAfter:
		if a.activePane == PaneCompose {
			a.compose.PasteAfter()
			a.mode = ModeNormal
		}
	case ActionPasteBefore:
		if a.activePane == PaneCompose {
			a.compose.PasteBefore()
			a.mode = ModeNormal
		}
	case ActionMoveLeft:
		if a.activePane == PaneCompose {
			a.compose.MoveLeft()
		}
	case ActionMoveRight:
		if a.activePane == PaneCompose {
			a.compose.MoveRight()
		}
	case ActionMoveWordFwd:
		if a.activePane == PaneCompose {
			a.compose.MoveWordFwd()
		}
	case ActionMoveWordBack:
		if a.activePane == PaneCompose {
			a.compose.MoveWordBack()
		}
	case ActionLineStart:
		if a.activePane == PaneCompose {
			a.compose.MoveLineStart()
		}
	case ActionLineStartNonSpace:
		if a.activePane == PaneCompose {
			a.compose.MoveLineStartNonSpace()
		}
	case ActionLineEnd:
		if a.activePane == PaneCompose {
			a.compose.MoveLineEnd()
		}
	case ActionDeleteChar:
		if a.activePane == PaneCompose {
			a.compose.DeleteForward()
		}
	case ActionDeleteToEOL:
		if a.activePane == PaneCompose {
			a.compose.DeleteToEOL()
		}
	case ActionDeleteLine:
		if a.activePane == PaneCompose {
			a.compose.DeleteCurrentLine()
		}
	case ActionMoveUp:
		if a.activePane == PaneHistory {
			a.history.Move(-1)
		}
	case ActionMoveDown:
		if a.activePane == PaneHistory {
			a.history.Move(1)
		}
	case ActionJumpStart:
		if a.activePane == PaneHistory {
			a.history.JumpStart()
		}
	case ActionJumpEnd:
		if a.activePane == PaneHistory {
			a.history.JumpEnd()
		}
	}

	// Enforce mode compatibility if we changed panes or modes.
	a.ensureModeSupported()
	return nil
}

func (a *App) ensureModeSupported() {
	if a.activePane == PaneHistory && a.mode == ModeInsert {
		a.mode = ModeNormal
	}
	if a.activePane == PaneHistory && a.mode == ModeVisual {
		a.mode = ModeNormal
		a.compose.exitVisualIfActive()
	}
}

func (a *App) enterCommandLine() {
	a.prevMode = a.mode
	a.mode = ModeCommandLine
	a.leaderPending = false
	a.command.Reset()
	a.compose.exitVisualIfActive()
}

func (a *App) exitCommandLine() {
	a.mode = a.prevMode
	a.command.Reset()
}

func (a *App) runCommand(input string) tea.Cmd {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		a.exitCommandLine()
		return nil
	}

	fields := strings.Fields(trimmed)
	name := strings.ToLower(fields[0])
	switch name {
	case "q", "quit":
		a.exitCommandLine()
		return tea.Quit
	default:
		// Unknown command: exit command mode quietly.
		a.exitCommandLine()
		_ = fields
		return nil
	}
}

func (a *App) switchPane() {
	if a.activePane == PaneHistory {
		a.activePane = PaneCompose
	} else {
		a.activePane = PaneHistory
	}
	a.ensureModeSupported()
}

func (a *App) handleComposeInsertKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			a.compose.InsertRune(r)
		}
		return true
	case tea.KeySpace:
		a.compose.InsertRune(' ')
		return true
	case tea.KeyBackspace:
		a.compose.Backspace()
		return true
	case tea.KeyDelete:
		a.compose.DeleteForward()
		return true
	case tea.KeyEnter:
		a.compose.InsertNewline()
		return true
	case tea.KeyLeft:
		a.compose.MoveLeft()
		return true
	case tea.KeyRight:
		a.compose.MoveRight()
		return true
	default:
		return false
	}
}

func (a *App) View() string {
	status := a.statusLine()
	composeView, composeLines := a.compose.View(a.width)

	commandView := ""
	commandLines := 0
	if a.mode == ModeCommandLine {
		commandView = a.command.View()
		commandLines = strings.Count(commandView, "\n") + 1
	}

	historyHeight := max(a.height-composeLines-1-commandLines, 1)

	a.history.width = a.width
	a.history.height = historyHeight
	historyView := a.history.View()

	parts := []string{historyView, composeView, status}
	if commandView != "" {
		parts = append(parts, commandView)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (a *App) statusLine() string {
	active := "History"
	if a.activePane == PaneCompose {
		active = "Compose"
	}
	mode := "NORMAL"
	switch a.mode {
	case ModeInsert:
		mode = "INSERT"
	case ModeVisual:
		mode = "VISUAL"
	case ModeCommandLine:
		mode = "COMMAND"
	}

	parts := []string{active, mode}
	return lipgloss.NewStyle().Reverse(true).Render(strings.Join(parts, " | "))
}
