package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// App is the root Bubble Tea model.
type App struct {
	mode          Mode
	activePane    Pane
	history       *HistoryScreen
	compose       *Compose
	bindings      []Binding
	leaderPending bool
	width         int
	height        int
}

// NewApp constructs the application with history + compose panes.
func NewApp() *App {
	return &App{
		mode:       ModeNormal,
		activePane: PaneHistory,
		history:    NewHistoryScreen(),
		compose:    NewCompose(),
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
		}
	case ActionEnterNormal:
		a.mode = ModeNormal
	case ActionEnterVisual:
		a.mode = ModeVisual
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

	historyHeight := max(a.height-composeLines-1, 1)

	a.history.width = a.width
	a.history.height = historyHeight
	historyView := a.history.View()

	return lipgloss.JoinVertical(lipgloss.Left, historyView, composeView, status)
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
