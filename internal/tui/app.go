package tui

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/boatnoah/kata/internal/codex"
)

type App struct {
	mode               Mode
	prevMode           Mode
	activePane         Pane
	history            *HistoryScreen
	compose            *Compose
	command            *CommandLine
	bindings           []Binding
	leaderPending      bool
	deletePending      bool
	width              int
	height             int
	ai                 *AIManager
	aiStreams          map[string]string
	aiRendered         map[string]string
	aiIndexes          map[string]int
	aiIconIdx          map[string]int
	aiTypes            map[string]string
	aiCompleted        map[string]bool
	aiTicking          map[string]bool
	aiTurnPlaceholders map[string]string
}

const aiTypeInterval = 35 * time.Millisecond
const aiRunesPerTick = 3

type codexEventMsg struct{ ev codex.Event }
type codexErrorMsg struct{ err error }
type aiTickMsg struct{ itemID string }

const aiTypeResponse = "AI"
const aiTypeWaiting = "AI_WAIT"

var spinnerVerbs = []string{
	"Accomplishing", "Actioning", "Actualizing", "Architecting", "Baking", "Beaming", "Beboppin'", "Befuddling", "Billowing", "Blanching", "Bloviating", "Boogieing", "Boondoggling", "Booping", "Bootstrapping", "Brewing", "Bunning", "Burrowing", "Calculating", "Canoodling", "Caramelizing", "Cascading", "Catapulting", "Cerebrating", "Channeling", "Channelling", "Choreographing", "Churning", "Clauding", "Coalescing", "Cogitating", "Combobulating", "Composing", "Computing", "Concocting", "Considering", "Contemplating", "Cooking", "Crafting", "Creating", "Crunching", "Crystallizing", "Cultivating", "Deciphering", "Deliberating", "Determining", "Dilly-dallying", "Discombobulating", "Doing", "Doodling", "Drizzling", "Ebbing", "Effecting", "Elucidating", "Embellishing", "Enchanting", "Envisioning", "Evaporating", "Fermenting", "Fiddle-faddling", "Finagling", "Flambeing", "Flibbertigibbeting", "Flowing", "Flummoxing", "Fluttering", "Forging", "Forming", "Frolicking", "Frosting", "Gallivanting", "Galloping", "Garnishing", "Generating", "Gesticulating", "Germinating", "Gitifying", "Grooving", "Gusting", "Harmonizing", "Hashing", "Hatching", "Herding", "Honking", "Hullaballooing", "Hyperspacing", "Ideating", "Imagining", "Improvising", "Incubating", "Inferring", "Infusing", "Ionizing", "Jitterbugging", "Julienning", "Kneading", "Leavening", "Levitating", "Lollygagging", "Manifesting", "Marinating", "Meandering", "Metamorphosing", "Misting", "Moonwalking", "Moseying", "Mulling", "Mustering", "Musing", "Nebulizing", "Nesting", "Newspapering", "Noodling", "Nucleating", "Orbiting", "Orchestrating", "Osmosing", "Perambulating", "Percolating", "Perusing", "Philosophising", "Photosynthesizing", "Pollinating", "Pondering", "Pontificating", "Pouncing", "Precipitating", "Prestidigitating", "Processing", "Proofing", "Propagating", "Puttering", "Puzzling", "Quantumizing", "Razzle-dazzling", "Razzmatazzing", "Recombobulating", "Reticulating", "Roosting", "Ruminating", "Sauteing", "Scampering", "Schlepping", "Scurrying", "Seasoning", "Shenaniganing", "Shimmying", "Simmering", "Skedaddling", "Sketching", "Slithering", "Smooshing", "Sock-hopping", "Spelunking", "Spinning", "Sprouting", "Stewing", "Sublimating", "Swirling", "Swooping", "Symbioting", "Synthesizing", "Tempering", "Thinking", "Thundering", "Tinkering", "Tomfoolering", "Topsy-turvying", "Transfiguring", "Transmuting", "Twisting", "Undulating", "Unfurling", "Unravelling", "Vibing", "Waddling", "Wandering", "Warping", "Whatchamacalliting", "Whirlpooling", "Whirring", "Whisking", "Wibbling", "Working", "Wrangling", "Zesting", "Zigzagging",
}

var spinnerIcons = []string{"|", "/", "-", "\\"}

func NewApp() *App {
	return &App{
		mode:               ModeNormal,
		prevMode:           ModeNormal,
		activePane:         PaneCompose,
		history:            NewHistoryScreen(),
		compose:            NewCompose(),
		command:            NewCommandLine(),
		bindings:           defaultBindings(),
		ai:                 newAIManager(),
		aiStreams:          make(map[string]string),
		aiRendered:         make(map[string]string),
		aiIndexes:          make(map[string]int),
		aiIconIdx:          make(map[string]int),
		aiTypes:            make(map[string]string),
		aiCompleted:        make(map[string]bool),
		aiTicking:          make(map[string]bool),
		aiTurnPlaceholders: make(map[string]string),
	}
}

var _ tea.Model = (*App)(nil)

func (a *App) Init() tea.Cmd {
	return a.subscribeAI()
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
	case codexEventMsg:
		return a, tea.Batch(a.subscribeAI(), a.handleCodexEvent(m.ev))
	case codexErrorMsg:
		a.history.AppendMessage("System: " + m.err.Error())
		return a, a.subscribeAI()
	case aiTickMsg:
		return a, a.handleAITick(m.itemID)
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
	}

	if a.activePane == PaneCompose && a.mode == ModeNormal {
		if msg.Type == tea.KeyRunes && string(msg.Runes) == "d" {
			a.deletePending = true
			return nil
		}
	}

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

	if a.leaderPending {
		a.leaderPending = false
		ks := keystrokeFromMsg(msg)
		if action, ok := findBinding(leaderBindings(), PaneAny, ModeAny, ks); ok {
			return a.applyAction(action)
		}
	}

	if a.mode != ModeInsert && isLeaderMsg(msg) {
		a.leaderPending = true
		return nil
	}

	ks := keystrokeFromMsg(msg)
	if action, ok := findBinding(a.bindings, a.activePane, a.mode, ks); ok {
		return a.applyAction(action)
	}

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
	case ActionEnterInsertLineStart:
		if a.activePane == PaneCompose {
			a.compose.MoveLineStartNonSpace()
			a.mode = ModeInsert
			a.compose.exitVisualIfActive()
		}
	case ActionEnterAppend:
		if a.activePane == PaneCompose {
			a.compose.Append()
			a.mode = ModeInsert
			a.compose.exitVisualIfActive()
		}
	case ActionEnterAppendLineEnd:
		if a.activePane == PaneCompose {
			a.compose.MoveLineEnd()
			a.compose.Append()
			a.mode = ModeInsert
			a.compose.exitVisualIfActive()
		}
	case ActionEnterOpenBelow:
		if a.activePane == PaneCompose {
			a.compose.OpenBelow()
			a.mode = ModeInsert
			a.compose.exitVisualIfActive()
		}
	case ActionEnterOpenAbove:
		if a.activePane == PaneCompose {
			a.compose.OpenAbove()
			a.mode = ModeInsert
			a.compose.exitVisualIfActive()
		}
	case ActionEnterNormal:
		a.mode = ModeNormal
		a.compose.exitVisualIfActive()
		a.history.ExitVisual()
	case ActionEnterVisual:
		switch a.activePane {
		case PaneCompose:
			a.mode = ModeVisual
			a.compose.EnterVisual()
		case PaneHistory:
			a.mode = ModeVisual
			a.history.EnterVisual()
		default:
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
	case ActionHistoryLeft:
		if a.activePane == PaneHistory {
			a.history.MoveLeft()
		}
	case ActionHistoryRight:
		if a.activePane == PaneHistory {
			a.history.MoveRight()
		}
	case ActionHistoryHalfPageDown:
		if a.activePane == PaneHistory {
			a.history.MoveHalfPage(1)
		}
	case ActionHistoryHalfPageUp:
		if a.activePane == PaneHistory {
			a.history.MoveHalfPage(-1)
		}
	case ActionHistoryLineStart:
		if a.activePane == PaneHistory {
			a.history.LineStart()
		}
	case ActionHistoryLineEnd:
		if a.activePane == PaneHistory {
			a.history.LineEnd()
		}
	case ActionHistoryYank:
		if a.activePane == PaneHistory {
			a.history.YankSelection()
			a.mode = ModeNormal
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
	case "q":
		a.exitCommandLine()
		return tea.Quit
	case "w":
		a.exitCommandLine()
		message := a.compose.Content()
		a.compose.Reset()
		if strings.TrimSpace(message) == "" {
			return nil
		}
		a.history.AppendMessage("User: " + message)
		return a.sendToAI(message)
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

// subscribeAI waits for the next Codex event and returns it as a tea.Msg.
func (a *App) subscribeAI() tea.Cmd {
	if a == nil || a.ai == nil {
		return nil
	}
	ch := a.ai.Events()
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return codexEventMsg{ev: ev}
	}
}

// sendToAI kicks off a Codex turn and keeps streaming via messages.
func (a *App) sendToAI(text string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := a.ai.SendText(ctx, text); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("codex request timed out (is the backend running?)")
			}
			return codexErrorMsg{err: err}
		}
		return nil
	}
}

func (a *App) handleCodexEvent(ev codex.Event) tea.Cmd {
	switch ev.Type {
	case codex.EventTurnStarted:
		return a.startAIThinking(ev.TurnID)
	case codex.EventAgentDelta:
		a.adoptThinkingPlaceholder(ev.TurnID, ev.ItemID)
		return a.upsertAIStream(ev.ItemID, aiTypeResponse, ev.Text, false)
	case codex.EventAgentCompleted:
		a.adoptThinkingPlaceholder(ev.TurnID, ev.ItemID)
		return a.upsertAIStream(ev.ItemID, aiTypeResponse, ev.Text, true)
	case codex.EventToolCall:
		return a.upsertAIStream(ev.ItemID, "Tool", ev.Text, false)
	case codex.EventCommandOutput:
		return a.upsertAIStream(ev.ItemID, "Cmd", ev.Text, false)
	case codex.EventTurnCompleted:
		if ev.Payload != nil {
			if errVal, ok := ev.Payload["error"]; ok {
				a.history.AppendMessage("Error: " + sanitizeText(fmt.Sprint(errVal)))
			}
		}
		// Finalize any active streams.
		var cmds []tea.Cmd
		for id, label := range a.aiTypes {
			cmds = append(cmds, a.upsertAIStream(id, label, "", true))
		}
		return tea.Batch(cmds...)
	case codex.EventError:
		if ev.Payload != nil {
			if msg, ok := ev.Payload["error"].(string); ok {
				a.history.AppendMessage("Error: " + sanitizeText(msg))
			}
		}
	}
	return nil
}

func (a *App) upsertAIStream(itemID, label, delta string, completed bool) tea.Cmd {
	a.aiTypes[itemID] = label
	if completed {
		finalText := sanitizeHistoryMessage(delta)
		if finalText != "" {
			a.aiStreams[itemID] = finalText
		} else if _, ok := a.aiStreams[itemID]; !ok {
			a.aiStreams[itemID] = ""
		}
	} else if strings.TrimSpace(delta) == "" {
		if _, ok := a.aiStreams[itemID]; !ok {
			a.aiStreams[itemID] = ""
		}
	} else {
		cleanDelta := sanitizeStreamDelta(delta)
		if cleanDelta != "" {
			a.aiStreams[itemID] = a.aiStreams[itemID] + cleanDelta
		}
	}
	if completed {
		a.aiCompleted[itemID] = true
	}
	if _, ok := a.aiRendered[itemID]; !ok {
		a.aiRendered[itemID] = ""
	}
	if !a.aiTicking[itemID] && a.aiRendered[itemID] == "" && a.aiStreams[itemID] != "" {
		a.advanceAIStream(itemID)
	}
	a.renderAIStream(itemID)
	if a.aiRendered[itemID] == a.aiStreams[itemID] {
		if a.aiCompleted[itemID] {
			a.finalizeAIStream(itemID)
		}
		return nil
	}
	if a.aiTicking[itemID] {
		return nil
	}
	a.aiTicking[itemID] = true
	return a.scheduleAITick(itemID)
}

func (a *App) handleAITick(itemID string) tea.Cmd {
	a.aiTicking[itemID] = false
	if a.aiRendered[itemID] != a.aiStreams[itemID] {
		a.advanceAIStream(itemID)
		a.renderAIStream(itemID)
	}
	if a.aiRendered[itemID] == a.aiStreams[itemID] {
		if a.isWaitingForAI(itemID) {
			a.renderAIStream(itemID)
			a.aiTicking[itemID] = true
			return a.scheduleAITick(itemID)
		}
		if a.aiCompleted[itemID] {
			a.finalizeAIStream(itemID)
		}
		return nil
	}
	a.aiTicking[itemID] = true
	return a.scheduleAITick(itemID)
}

func (a *App) scheduleAITick(itemID string) tea.Cmd {
	return tea.Tick(aiTypeInterval, func(time.Time) tea.Msg {
		return aiTickMsg{itemID: itemID}
	})
}

func (a *App) advanceAIStream(itemID string) {
	target := []rune(a.aiStreams[itemID])
	current := []rune(a.aiRendered[itemID])
	if len(current) >= len(target) {
		a.aiRendered[itemID] = string(target)
		return
	}
	next := len(current) + aiRunesPerTick
	if next > len(target) {
		next = len(target)
	}
	a.aiRendered[itemID] = string(target[:next])
}

func (a *App) renderAIStream(itemID string) {
	label, ok := a.aiTypes[itemID]
	if !ok {
		return
	}
	buf := a.aiRendered[itemID]
	completed := a.aiCompleted[itemID] && a.aiRendered[itemID] == a.aiStreams[itemID]
	line := label + ": " + buf
	if label == aiTypeWaiting && !completed {
		line = aiTypeWaiting + ": " + a.nextIcon(itemID) + " Thinking"
	}
	focus := a.shouldFollowHistory(itemID)
	if idx, ok := a.aiIndexes[itemID]; ok {
		a.history.UpdateMessageAtWithFocus(idx, line, focus)
	} else {
		idx := a.history.AppendMessageWithFocus(line, focus)
		a.aiIndexes[itemID] = idx
	}
}

func (a *App) finalizeAIStream(itemID string) {
	label, ok := a.aiTypes[itemID]
	if !ok {
		return
	}
	final := label + ": " + a.aiRendered[itemID]
	focus := a.shouldFollowHistory(itemID)
	if idx, ok := a.aiIndexes[itemID]; ok {
		a.history.UpdateMessageAtWithFocus(idx, final, focus)
	}
	delete(a.aiIconIdx, itemID)
	delete(a.aiTypes, itemID)
	delete(a.aiCompleted, itemID)
	delete(a.aiTicking, itemID)
}

func (a *App) shouldFollowHistory(itemID string) bool {
	idx, ok := a.aiIndexes[itemID]
	if !ok {
		return a.activePane != PaneHistory || a.history.cursor >= len(a.history.messages)-1
	}
	return a.activePane != PaneHistory || a.history.cursor == idx || a.history.cursor >= len(a.history.messages)-1
}

func (a *App) nextIcon(itemID string) string {
	if len(spinnerIcons) == 0 {
		return ">"
	}
	idx := a.aiIconIdx[itemID]
	icon := spinnerIcons[idx%len(spinnerIcons)]
	a.aiIconIdx[itemID] = idx + 1
	return icon
}

func (a *App) startAIThinking(turnID string) tea.Cmd {
	if turnID == "" {
		return nil
	}
	itemID := "turn:" + turnID
	a.aiTurnPlaceholders[turnID] = itemID
	a.aiTypes[itemID] = aiTypeWaiting
	if _, ok := a.aiRendered[itemID]; !ok {
		a.aiRendered[itemID] = ""
	}
	if _, ok := a.aiStreams[itemID]; !ok {
		a.aiStreams[itemID] = ""
	}
	a.renderAIStream(itemID)
	if a.aiTicking[itemID] {
		return nil
	}
	a.aiTicking[itemID] = true
	return a.scheduleAITick(itemID)
}

func (a *App) adoptThinkingPlaceholder(turnID, itemID string) {
	placeholderID, ok := a.aiTurnPlaceholders[turnID]
	if !ok || placeholderID == itemID {
		return
	}
	if idx, ok := a.aiIndexes[placeholderID]; ok {
		a.aiIndexes[itemID] = idx
		delete(a.aiIndexes, placeholderID)
	}
	if rendered, ok := a.aiRendered[placeholderID]; ok {
		a.aiRendered[itemID] = rendered
		delete(a.aiRendered, placeholderID)
	}
	if stream, ok := a.aiStreams[placeholderID]; ok {
		a.aiStreams[itemID] = stream
		delete(a.aiStreams, placeholderID)
	}
	if _, ok := a.aiTicking[placeholderID]; ok {
		a.aiTicking[itemID] = false
		delete(a.aiTicking, placeholderID)
	}
	if iconIdx, ok := a.aiIconIdx[placeholderID]; ok {
		a.aiIconIdx[itemID] = iconIdx
		delete(a.aiIconIdx, placeholderID)
	}
	delete(a.aiTypes, placeholderID)
	delete(a.aiCompleted, placeholderID)
	delete(a.aiTurnPlaceholders, turnID)
}

func (a *App) isWaitingForAI(itemID string) bool {
	return a.aiTypes[itemID] == aiTypeWaiting && a.aiStreams[itemID] == "" && !a.aiCompleted[itemID]
}

func sanitizeText(s string) string {
	s = stripSystemReminders(s)
	s = strings.TrimSpace(s)
	return s
}

var systemReminderPattern = regexp.MustCompile(`(?s)<system-reminder>.*?</system-reminder>`)

func stripSystemReminders(s string) string {
	return systemReminderPattern.ReplaceAllString(s, "")
}

func sanitizeStreamDelta(s string) string {
	s = stripSystemReminders(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func sanitizeHistoryMessage(s string) string {
	for {
		start := strings.Index(s, "<system-reminder>")
		if start == -1 {
			break
		}
		end := strings.Index(s, "</system-reminder>")
		if end == -1 {
			s = s[:start]
			break
		}
		s = s[:start] + s[end+len("</system-reminder>"):]
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.TrimRight(s, "\n")
	return s
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
	a.compose.SetActive(a.activePane == PaneCompose)
	// Keep compose compact (Claude/Opencode-like): target 5 lines including border.
	composeHeight := 5
	if a.height > 0 {
		// Leave room for status line and at least one history line.
		maxAllowed := max(a.height-2, 3)
		if composeHeight > maxAllowed {
			composeHeight = maxAllowed
		}
	}

	composeView, composeLines := a.compose.View(a.width, composeHeight)

	commandView := ""
	commandLines := 0
	if a.mode == ModeCommandLine {
		commandView = a.command.View()
		commandLines = strings.Count(commandView, "\n") + 1
	}

	historyHeight := max(a.height-composeLines-1-commandLines, 1)

	a.history.width = a.width
	a.history.height = historyHeight
	a.history.SetActive(a.activePane == PaneHistory)
	historyView := a.history.View()

	historyView = padToWidth(historyView, a.width)
	composeView = padToWidth(composeView, a.width)
	status = padToWidth(status, a.width)

	if commandView != "" {
		commandView = padToWidth(commandView, a.width)
	}

	parts := []string{historyView, composeView, status}
	if commandView != "" {
		parts = append(parts, commandView)
	}
	frame := lipgloss.JoinVertical(lipgloss.Left, parts...)
	lines := strings.Count(frame, "\n") + 1
	if a.height > 0 && lines < a.height {
		padding := strings.Repeat("\n", a.height-lines)
		frame += padding
	}
	return frame
}

// padToWidth ensures every line is space-padded to the given width so that
// shorter lines fully overwrite previous frames in the terminal.
func padToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		w := lipgloss.Width(ln)
		if w < width {
			lines[i] = ln + strings.Repeat(" ", width-w)
		}
	}
	return strings.Join(lines, "\n")
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
