package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/boatnoah/kata/internal/agent"
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
	gPending           bool
	yPending           bool
	statusNotice       string
	width              int
	height             int
	theme              Theme
	sessionID          string
	branch             string
	provider           string
	model              string
	title              string
	ctxUsed            int
	ctxTotal           int
	ai                 *AIManager
	streams            map[string]*AIStream
	aiIndexes          map[string]int
	aiVerbIdx          map[string]int
	aiWaitFrames       map[string]int
	aiCompleted        map[string]bool
	aiTicking          map[string]bool
	aiTurnPlaceholders map[string]string
	// turnPrimed records turns where we already created the "thinking" row
	// (from turn/started or recovered on the first streamed event). Cleared on
	// turn/completed so a burst of Codex notifications cannot skip priming.
	turnPrimed   map[string]struct{}
	aiToolTitle  map[string]string
	aiToolDetail map[string]string
	aiToolState  map[string]ToolState

	// Pending server→client JSON-RPC approval (at most one). Filled from
	// agent.EventApprovalRequired; cleared after :approve / :deny or superseded.
	pendingRPCID json.RawMessage
	pendingKind  string

	// Parsed per-file changes from the most recent applyPatch approval. Kept
	// so :diff can re-render them unbounded after the initial truncated
	// preview scrolls out of view.
	lastPatchFiles []agent.FileChange
}

const aiTypeInterval = 35 * time.Millisecond
const aiRunesPerTick = 3

// localOutboundPendingID is the stream key for the placeholder row added
// synchronously when the user sends, before Codex streams anything. Some
// turns never emit turn/started or tool rows; without this the chat can jump
// straight from the user line to the final assistant text.
const localOutboundPendingID = "local:outbound"

type codexEventMsg struct{ ev agent.Event }
type codexErrorMsg struct{ err error }
type aiTickMsg struct{ itemID string }
type clearStatusMsg struct{}

const aiTypeResponse = TranscriptAssistant
const aiTypeWaiting = TranscriptThinking

var spinnerVerbs = []string{
	"Accomplishing",
	"Actioning",
	"Actualizing",
	"Architecting",
	"Baking",
	"Beaming",
	"Beboppin'",
	"Befuddling",
	"Billowing",
	"Blanching",
	"Bloviating",
	"Boogieing",
	"Boondoggling",
	"Booping",
	"Bootstrapping",
	"Brewing",
	"Bunning",
	"Burrowing",
	"Calculating",
	"Canoodling",
	"Caramelizing",
	"Cascading",
	"Catapulting",
	"Cerebrating",
	"Channeling",
	"Channelling",
	"Choreographing",
	"Churning",
	"Clauding",
	"Coalescing",
	"Cogitating",
	"Combobulating",
	"Composing",
	"Computing",
	"Concocting",
	"Considering",
	"Contemplating",
	"Cooking",
	"Crafting",
	"Creating",
	"Crunching",
	"Crystallizing",
	"Cultivating",
	"Deciphering",
	"Deliberating",
	"Determining",
	"Dilly-dallying",
	"Discombobulating",
	"Doing",
	"Doodling",
	"Drizzling",
	"Ebbing",
	"Effecting",
	"Elucidating",
	"Embellishing",
	"Enchanting",
	"Envisioning",
	"Evaporating",
	"Fermenting",
	"Fiddle-faddling",
	"Finagling",
	"Flambéing",
	"Flibbertigibbeting",
	"Flowing",
	"Flummoxing",
	"Fluttering",
	"Forging",
	"Forming",
	"Frolicking",
	"Frosting",
	"Gallivanting",
	"Galloping",
	"Garnishing",
	"Generating",
	"Gesticulating",
	"Germinating",
	"Gitifying",
	"Grooving",
	"Gusting",
	"Harmonizing",
	"Hashing",
	"Hatching",
	"Herding",
	"Honking",
	"Hullaballooing",
	"Hyperspacing",
	"Ideating",
	"Imagining",
	"Improvising",
	"Incubating",
	"Inferring",
	"Infusing",
	"Ionizing",
	"Jitterbugging",
	"Julienning",
	"Kneading",
	"Leavening",
	"Levitating",
	"Lollygagging",
	"Manifesting",
	"Marinating",
	"Meandering",
	"Metamorphosing",
	"Misting",
	"Moonwalking",
	"Moseying",
	"Mulling",
	"Mustering",
	"Musing",
	"Nebulizing",
	"Nesting",
	"Newspapering",
	"Noodling",
	"Nucleating",
	"Orbiting",
	"Orchestrating",
	"Osmosing",
	"Perambulating",
	"Percolating",
	"Perusing",
	"Philosophising",
	"Photosynthesizing",
	"Pollinating",
	"Pondering",
	"Pontificating",
	"Pouncing",
	"Precipitating",
	"Prestidigitating",
	"Processing",
	"Proofing",
	"Propagating",
	"Puttering",
	"Puzzling",
	"Quantumizing",
	"Razzle-dazzling",
	"Razzmatazzing",
	"Recombobulating",
	"Reticulating",
	"Roosting",
	"Ruminating",
	"Sautéing",
	"Scampering",
	"Schlepping",
	"Scurrying",
	"Seasoning",
	"Shenaniganing",
	"Shimmying",
	"Simmering",
	"Skedaddling",
	"Sketching",
	"Slithering",
	"Smooshing",
	"Sock-hopping",
	"Spelunking",
	"Spinning",
	"Sprouting",
	"Stewing",
	"Sublimating",
	"Swirling",
	"Swooping",
	"Symbioting",
	"Synthesizing",
	"Tempering",
	"Thinking",
	"Thundering",
	"Tinkering",
	"Tomfoolering",
	"Topsy-turvying",
	"Transfiguring",
	"Transmuting",
	"Twisting",
	"Undulating",
	"Unfurling",
	"Unravelling",
	"Vibing",
	"Waddling",
	"Wandering",
	"Warping",
	"Whatchamacalliting",
	"Whirlpooling",
	"Whirring",
	"Whisking",
	"Wibbling",
	"Working",
	"Wrangling",
	"Zesting",
	"Zigzagging",
}

var spinnerDots = []string{".", "..", "..."}

func NewApp() *App {
	a := &App{
		mode:               ModeNormal,
		prevMode:           ModeNormal,
		activePane:         PaneCompose,
		history:            NewHistoryScreen(),
		compose:            NewCompose(),
		command:            NewCommandLine(),
		bindings:           defaultBindings(),
		theme:              DefaultTheme(),
		sessionID:          newSessionID(),
		branch:             detectBranch(),
		ctxTotal:           200_000,
		ai:                 newAIManager(),
		streams:            make(map[string]*AIStream),
		aiIndexes:          make(map[string]int),
		aiVerbIdx:          make(map[string]int),
		aiWaitFrames:       make(map[string]int),
		aiCompleted:        make(map[string]bool),
		aiTicking:          make(map[string]bool),
		aiTurnPlaceholders: make(map[string]string),
		turnPrimed:         make(map[string]struct{}),
		aiToolTitle:        make(map[string]string),
		aiToolDetail:       make(map[string]string),
		aiToolState:        make(map[string]ToolState),
	}
	a.provider = a.ai.Provider()
	a.model = a.ai.Model()
	a.compose.SetTheme(a.theme)
	a.history.SetTheme(a.theme)
	return a
}

// newSessionID returns a short, terminal-friendly id like "4f2a". Reusing
// the existing rand/v2 dependency keeps the process lightweight.
func newSessionID() string {
	const alphabet = "0123456789abcdef"
	b := make([]byte, 4)
	for i := range b {
		b[i] = alphabet[rand.IntN(len(alphabet))]
	}
	return string(b)
}

// detectBranch returns the current git branch if discoverable, else "".
// Cheap best-effort via `git rev-parse --abbrev-ref HEAD`.
func detectBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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
		a.abandonOutboundPending()
		a.history.AppendItem(TranscriptItem{Kind: TranscriptSystem, Text: m.err.Error()}, true)
		return a, a.subscribeAI()
	case aiTickMsg:
		return a, a.handleAITick(m.itemID)
	case clearStatusMsg:
		a.statusNotice = ""
		return a, nil
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

	// History chords: gg → top, yy → yank current item (CHAT scope).
	if a.activePane == PaneHistory && a.mode == ModeNormal && a.gPending {
		a.gPending = false
		if msg.Type == tea.KeyRunes && string(msg.Runes) == "g" {
			a.history.CursorTop()
			return nil
		}
	}
	if a.activePane == PaneHistory && a.mode == ModeNormal && a.yPending {
		a.yPending = false
		if msg.Type == tea.KeyRunes && string(msg.Runes) == "y" {
			a.history.YankCurrentLine()
			return nil
		}
	}
	if a.activePane == PaneHistory && a.mode == ModeNormal && msg.Type == tea.KeyRunes {
		switch string(msg.Runes) {
		case "g":
			a.gPending = true
			return nil
		case "y":
			a.yPending = true
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
		a.supersedePendingDecline(context.Background())
		return tea.Quit
	case ActionSwitchPane:
		a.switchPane()
	case ActionFocusCompose:
		a.activePane = PaneCompose
		a.ensureModeSupported()
	case ActionFocusHistory:
		a.activePane = PaneHistory
		a.history.EnsureCursor()
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
	case ActionEnterVisual:
		if a.activePane == PaneCompose {
			a.mode = ModeVisual
			a.compose.EnterVisual()
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
	case ActionScrollUp:
		a.history.ScrollUp(1)
	case ActionScrollDown:
		a.history.ScrollDown(1)
	case ActionScrollHalfPageUp:
		a.history.ScrollHalfPageUp()
	case ActionScrollHalfPageDown:
		a.history.ScrollHalfPageDown()
	case ActionScrollTop:
		a.history.ScrollToTop()
	case ActionScrollBottom:
		a.history.ScrollToBottom()
	case ActionHistoryCursorUp:
		a.history.CursorUp(1)
	case ActionHistoryCursorDown:
		a.history.CursorDown(1)
	case ActionHistoryCursorTop:
		a.history.CursorTop()
	case ActionHistoryCursorBottom:
		a.history.CursorBottom()
	case ActionHistoryEnterVisual:
		a.history.EnterVisual()
		a.mode = ModeVisual
	case ActionHistoryYank:
		a.history.YankSelection()
		a.mode = ModeNormal
	}

	// Enforce mode compatibility if we changed panes or modes.
	a.ensureModeSupported()
	return nil
}

func (a *App) ensureModeSupported() {
	// Forbid INSERT in history; NORMAL/VISUAL/COMMAND are all allowed so the
	// CHAT scope can host line-wise visual selection.
	if a.activePane == PaneHistory && a.mode == ModeInsert {
		a.mode = ModeNormal
	}
	// Always keep compose visual state in sync with app mode.
	if a.mode != ModeVisual {
		a.compose.exitVisualIfActive()
	}
	// Same for history: clearing ModeVisual at the app level must tear down
	// the selection so the next `v` starts fresh.
	if a.mode != ModeVisual {
		a.history.ExitVisual()
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
	// If we were in visual mode but visual was cleared (e.g. by enterCommandLine),
	// don't restore to visual — fall back to normal.
	if a.mode == ModeVisual && !a.compose.visualActive {
		a.mode = ModeNormal
	}
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
	args := fields[1:]
	switch name {
	case "q", "q!":
		a.exitCommandLine()
		a.supersedePendingDecline(context.Background())
		return tea.Quit
	case "w":
		a.exitCommandLine()
		message := a.compose.Content()
		a.compose.Reset()
		if strings.TrimSpace(message) == "" {
			return nil
		}
		a.history.AppendItem(TranscriptItem{Kind: TranscriptUser, Text: message}, true)
		return mergeTeaCmd(a.beginOutboundWaitSync(), a.sendToAI(message))
	case "wq":
		a.exitCommandLine()
		a.supersedePendingDecline(context.Background())
		message := a.compose.Content()
		a.compose.Reset()
		var cmds []tea.Cmd
		if strings.TrimSpace(message) != "" {
			a.history.AppendItem(TranscriptItem{Kind: TranscriptUser, Text: message}, true)
			cmds = append(cmds, mergeTeaCmd(a.beginOutboundWaitSync(), a.sendToAI(message)))
		}
		cmds = append(cmds, tea.Quit)
		return tea.Batch(cmds...)
	case "clear":
		a.exitCommandLine()
		a.history.items = nil
		a.history.invalidateLines()
		a.flashStatus("cleared")
		return a.clearStatusAfter(2 * time.Second)
	case "new":
		a.exitCommandLine()
		a.supersedePendingDecline(context.Background())
		a.history.items = nil
		a.history.invalidateLines()
		a.compose.Reset()
		a.sessionID = newSessionID()
		a.flashStatus("new session · " + a.sessionID)
		return a.clearStatusAfter(2 * time.Second)
	case "colorscheme", "colo":
		a.exitCommandLine()
		if len(args) == 0 {
			a.flashStatus("usage: :colorscheme " + strings.Join(ThemeNames(), "|"))
			return a.clearStatusAfter(3 * time.Second)
		}
		if theme, ok := ThemeByName(args[0]); ok {
			a.theme = theme
			a.compose.SetTheme(theme)
			a.history.SetTheme(theme)
			a.flashStatus(":colorscheme " + args[0])
		} else {
			a.flashStatus("E185: cannot find color scheme '" + args[0] + "'")
		}
		return a.clearStatusAfter(2 * time.Second)
	case "model":
		a.exitCommandLine()
		if len(args) == 0 {
			a.flashStatus("current model: " + a.provider + " · " + a.model)
		} else {
			a.flashStatus("model swap not yet supported (only " + a.provider + " is wired)")
		}
		return a.clearStatusAfter(2 * time.Second)
	case "help":
		a.exitCommandLine()
		a.flashStatus("keys: i insert · : cmd · j/k scroll · gg/G top/bot · esc normal · :w send · :q quit · :approve / :deny (pending approval) · :diff (expand patch)")
		return a.clearStatusAfter(4 * time.Second)
	case "sess":
		a.exitCommandLine()
		a.flashStatus("session picker not yet implemented")
		return a.clearStatusAfter(2 * time.Second)
	case "diff":
		a.exitCommandLine()
		if len(a.lastPatchFiles) == 0 {
			a.flashStatus("no pending patch")
			return a.clearStatusAfter(2 * time.Second)
		}
		for _, f := range a.lastPatchFiles {
			a.history.AppendItem(TranscriptItem{
				Kind: TranscriptDiff,
				Diff: f,
			}, true)
		}
		return nil
	case "approve":
		a.exitCommandLine()
		return a.submitPendingApproval(args, true)
	case "deny", "decline":
		a.exitCommandLine()
		return a.submitPendingApproval(nil, false)
	default:
		a.exitCommandLine()
		a.flashStatus("E492: not a kata command: " + name)
		return a.clearStatusAfter(3 * time.Second)
	}
}

func (a *App) clearPendingApproval() {
	a.pendingRPCID = nil
	a.pendingKind = ""
}

func (a *App) supersedePendingDecline(ctx context.Context) {
	if len(a.pendingRPCID) == 0 {
		return
	}
	if res, ok := approvalDeclineResult(a.pendingKind); ok {
		id := append(json.RawMessage(nil), a.pendingRPCID...)
		_ = a.ai.RespondServerRPC(ctx, id, res)
	}
	a.clearPendingApproval()
}

// applyPatchDiffPreviewLines caps how many lines of each file's diff show
// inline on the initial approval event. Users can expand with :diff.
const applyPatchDiffPreviewLines = 12

// appendApplyPatchDiffs inserts one TranscriptDiff item per file carried by
// an applyPatch approval event. No-op when the event is not an applyPatch
// or carries no parsed file changes. Stashes the parsed slice on the App so
// :diff can re-render it unbounded later.
func (a *App) appendApplyPatchDiffs(ev agent.Event) {
	if ev.Payload == nil {
		return
	}
	if kind, _ := ev.Payload["approvalKind"].(string); kind != codex.ApprovalKindApplyPatch {
		return
	}
	files, _ := ev.Payload["parsedFileChanges"].([]agent.FileChange)
	if len(files) == 0 {
		return
	}
	a.lastPatchFiles = files
	for _, f := range files {
		a.history.AppendItem(TranscriptItem{
			Kind:         TranscriptDiff,
			Diff:         f,
			DiffMaxLines: applyPatchDiffPreviewLines,
		}, true)
	}
}

func (a *App) armPendingApproval(ev agent.Event) {
	if len(ev.RPCID) == 0 {
		return
	}
	ctx := context.Background()
	if len(a.pendingRPCID) > 0 {
		a.supersedePendingDecline(ctx)
	}
	var kind string
	if ev.Payload != nil {
		kind, _ = ev.Payload["approvalKind"].(string)
	}
	a.pendingRPCID = append(json.RawMessage(nil), ev.RPCID...)
	a.pendingKind = kind
}

func (a *App) submitPendingApproval(args []string, accept bool) tea.Cmd {
	ctx := context.Background()
	if len(a.pendingRPCID) == 0 {
		a.flashStatus("E518: no pending approval")
		return a.clearStatusAfter(2 * time.Second)
	}
	kind := a.pendingKind
	id := append(json.RawMessage(nil), a.pendingRPCID...)
	var result any
	var ok bool
	if accept {
		session := len(args) > 0 && strings.EqualFold(args[0], "session")
		if session {
			result, ok = approvalAcceptSessionResult(kind)
		} else {
			result, ok = approvalAcceptResult(kind)
		}
	} else {
		result, ok = approvalDeclineResult(kind)
	}
	if !ok {
		a.flashStatus("E497: :approve not supported for " + kind + " (needs structured reply)")
		return a.clearStatusAfter(3 * time.Second)
	}
	if err := a.ai.RespondServerRPC(ctx, id, result); err != nil {
		a.clearPendingApproval()
		if errors.Is(err, agent.ErrRPCResponderUnsupported) {
			a.flashStatus("backend cannot send approval replies")
		} else {
			a.history.AppendItem(TranscriptItem{Kind: TranscriptError, Text: sanitizeText(err.Error())}, true)
		}
		return a.clearStatusAfter(3 * time.Second)
	}
	a.clearPendingApproval()
	if accept {
		a.flashStatus("approved")
	} else {
		a.flashStatus("declined")
	}
	return a.clearStatusAfter(1500 * time.Millisecond)
}

// flashStatus sets a transient status-line notice.
func (a *App) flashStatus(msg string) {
	a.statusNotice = msg
}

// clearStatusAfter returns a command that clears statusNotice after d.
func (a *App) clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

func (a *App) switchPane() {
	if a.activePane == PaneHistory {
		a.activePane = PaneCompose
	} else {
		a.activePane = PaneHistory
		a.history.EnsureCursor()
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

func (a *App) handleCodexEvent(ev agent.Event) tea.Cmd {
	switch ev.Type {
	case agent.EventTurnStarted:
		return a.startAIThinking(ev.TurnID)
	case agent.EventAgentDelta:
		c1 := a.startAIThinking(ev.TurnID)
		a.adoptThinkingPlaceholder(ev.TurnID, ev.ItemID)
		c2 := a.upsertAIStream(ev.ItemID, aiTypeResponse, ev.Text, false)
		return mergeTeaCmd(c1, c2)
	case agent.EventAgentCompleted:
		c1 := a.startAIThinking(ev.TurnID)
		a.adoptThinkingPlaceholder(ev.TurnID, ev.ItemID)
		c2 := a.upsertAIStream(ev.ItemID, aiTypeResponse, ev.Text, true)
		return mergeTeaCmd(c1, c2)
	case agent.EventToolCall:
		c1 := a.startAIThinking(ev.TurnID)
		c2 := a.setToolCallSummary(ev.ItemID, summarizeToolCall(ev), true)
		return mergeTeaCmd(c1, c2)
	case agent.EventCommandOutput:
		return a.handleCommandExecution(ev)
	case agent.EventTokenUsage:
		a.applyTokenUsage(ev.Payload)
		return nil
	case agent.EventTurnCompleted:
		if ev.TurnID != "" {
			delete(a.turnPrimed, ev.TurnID)
		}
		if ev.Payload != nil {
			if errVal, ok := ev.Payload["error"]; ok {
				a.history.AppendItem(TranscriptItem{Kind: TranscriptError, Text: sanitizeText(fmt.Sprint(errVal))}, true)
			}
		}
		// Finalize any active streams.
		var cmds []tea.Cmd
		for id, s := range a.streams {
			cmds = append(cmds, a.upsertAIStream(id, s.Label(), "", true))
		}
		return tea.Batch(cmds...)
	case agent.EventError:
		a.abandonOutboundPending()
		if ev.Payload != nil {
			if msg, ok := ev.Payload["error"].(string); ok {
				a.history.AppendItem(TranscriptItem{Kind: TranscriptError, Text: sanitizeText(msg)}, true)
			}
		}
	case agent.EventApprovalRequired:
		a.armPendingApproval(ev)
		// Surface a system line; resolve with :approve / :deny (see :help).
		msg := "Approval required"
		if ev.Payload != nil {
			// Stable key emitted by codex server_request (approvalKind).
			if k, _ := ev.Payload["approvalKind"].(string); k != "" {
				msg = "Approval required (" + k + ")"
			}
		}
		if t := strings.TrimSpace(ev.Text); t != "" {
			msg = msg + ": " + t
		}
		a.history.AppendItem(TranscriptItem{Kind: TranscriptSystem, Text: sanitizeText(msg)}, true)
		a.appendApplyPatchDiffs(ev)
	}
	return nil
}

// mergeTeaCmd combines two Bubble Tea commands, dropping nils.
func mergeTeaCmd(a, b tea.Cmd) tea.Cmd {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		return tea.Batch(a, b)
	}
}

func (a *App) teardownAIStreamKeys(itemID string) {
	delete(a.streams, itemID)
	delete(a.aiVerbIdx, itemID)
	delete(a.aiWaitFrames, itemID)
	delete(a.aiCompleted, itemID)
	delete(a.aiTicking, itemID)
	delete(a.aiToolTitle, itemID)
	delete(a.aiToolDetail, itemID)
	delete(a.aiToolState, itemID)
}

// stream returns the AIStream for itemID, or nil if absent.
func (a *App) stream(itemID string) *AIStream { return a.streams[itemID] }

// ensureStream returns the AIStream for itemID, creating one with the given
// label if absent. The label is set on the returned stream regardless.
func (a *App) ensureStream(itemID string, label TranscriptKind) *AIStream {
	s, ok := a.streams[itemID]
	if !ok {
		s = newAIStream(label)
		a.streams[itemID] = s
		return s
	}
	s.SetLabel(label)
	return s
}

func (a *App) spliceHistoryRemove(idx int) {
	if idx < 0 || idx >= len(a.history.items) {
		return
	}
	a.history.items = append(a.history.items[:idx], a.history.items[idx+1:]...)
	if idx < len(a.history.renderCache) {
		a.history.renderCache = append(a.history.renderCache[:idx], a.history.renderCache[idx+1:]...)
	} else if len(a.history.renderCache) > len(a.history.items) {
		a.history.renderCache = a.history.renderCache[:len(a.history.items)]
	}
	for id, i := range a.aiIndexes {
		if i > idx {
			a.aiIndexes[id] = i - 1
		}
	}
	a.history.invalidateLines()
}

// abandonOutboundPending drops the pre-Codex "waiting" row if it is still
// present (e.g. RPC error before any streamed events).
func (a *App) abandonOutboundPending() {
	idx, ok := a.aiIndexes[localOutboundPendingID]
	if !ok {
		return
	}
	delete(a.aiIndexes, localOutboundPendingID)
	a.teardownAIStreamKeys(localOutboundPendingID)
	a.spliceHistoryRemove(idx)
}

// migrateAIStreamKeys renames per-item stream state from one id to another
// (same history row index).
func (a *App) migrateAIStreamKeys(from, to string) {
	if from == to {
		return
	}
	if idx, ok := a.aiIndexes[from]; ok {
		delete(a.aiIndexes, from)
		a.aiIndexes[to] = idx
	}
	if s, ok := a.streams[from]; ok {
		a.streams[to] = s
		delete(a.streams, from)
	}
	if v, ok := a.aiVerbIdx[from]; ok {
		a.aiVerbIdx[to] = v
		delete(a.aiVerbIdx, from)
	}
	if v, ok := a.aiWaitFrames[from]; ok {
		a.aiWaitFrames[to] = v
		delete(a.aiWaitFrames, from)
	}
	if v, ok := a.aiCompleted[from]; ok {
		a.aiCompleted[to] = v
		delete(a.aiCompleted, from)
	}
	delete(a.aiTicking, from)
	a.aiTicking[to] = false
	if v, ok := a.aiToolTitle[from]; ok {
		a.aiToolTitle[to] = v
		delete(a.aiToolTitle, from)
	}
	if v, ok := a.aiToolDetail[from]; ok {
		a.aiToolDetail[to] = v
		delete(a.aiToolDetail, from)
	}
	if v, ok := a.aiToolState[from]; ok {
		a.aiToolState[to] = v
		delete(a.aiToolState, from)
	}
}

// beginOutboundWaitSync runs in the same Update as :w — before sendToAI's
// command — so the user always sees an immediate waiting row.
func (a *App) beginOutboundWaitSync() tea.Cmd {
	id := localOutboundPendingID
	if _, exists := a.aiIndexes[id]; exists {
		delete(a.aiVerbIdx, id)
		delete(a.aiWaitFrames, id)
		a.ensureWaitingVerb(id)
		a.renderAIStream(id)
		if a.aiTicking[id] {
			return nil
		}
		a.aiTicking[id] = true
		return a.scheduleAITick(id)
	}

	a.ensureStream(id, aiTypeWaiting)
	a.aiCompleted[id] = false
	a.ensureWaitingVerb(id)
	a.renderAIStream(id)
	if a.aiTicking[id] {
		return nil
	}
	a.aiTicking[id] = true
	return a.scheduleAITick(id)
}

// applyTokenUsage folds a Codex token-usage snapshot into the statusline
// counters. `used` is the last turn's prompt size (what currently fills the
// context window); `contextWindow`, when present, replaces the default limit
// so the displayed total matches the model the backend actually picked.
func (a *App) applyTokenUsage(payload map[string]any) {
	if payload == nil {
		return
	}
	if v, ok := payload["used"].(int); ok && v >= 0 {
		a.ctxUsed = v
	}
	if v, ok := payload["contextWindow"].(int); ok && v > 0 {
		a.ctxTotal = v
	}
}

// setToolCallSummary writes or updates a tool-call summary for itemID.
// When final is true the item is finalized and the per-item maps are torn
// down; when false the item stays live so subsequent events (e.g. command
// execution completion) can update its title, detail, or state in place.
// Empty Title/Detail on the incoming summary preserve the prior value so a
// "started" event followed by a "completed" event with only an exit code
// doesn't blank the command line from the previous render.
func (a *App) setToolCallSummary(itemID string, s toolSummary, final bool) tea.Cmd {
	stream := a.ensureStream(itemID, TranscriptTool)
	stream.ReplaceBuffer("")
	stream.SetRendered("")
	a.aiCompleted[itemID] = final
	if s.Title != "" {
		a.aiToolTitle[itemID] = s.Title
	} else if _, ok := a.aiToolTitle[itemID]; !ok {
		a.aiToolTitle[itemID] = ""
	}
	if s.Detail != "" {
		a.aiToolDetail[itemID] = s.Detail
	} else if _, ok := a.aiToolDetail[itemID]; !ok {
		a.aiToolDetail[itemID] = ""
	}
	a.aiToolState[itemID] = s.State
	a.renderAIStream(itemID)
	if final {
		a.finalizeAIStream(itemID)
	}
	return nil
}

func (a *App) upsertAIStream(itemID string, label TranscriptKind, delta string, completed bool) tea.Cmd {
	s := a.ensureStream(itemID, label)
	if completed {
		finalText := sanitizeHistoryMessage(delta)
		if finalText != "" {
			s.ReplaceBuffer(finalText)
		}
	} else if strings.TrimSpace(delta) != "" {
		cleanDelta := sanitizeStreamDelta(delta)
		if cleanDelta != "" {
			s.AppendDelta(cleanDelta)
		}
	}
	if completed {
		a.aiCompleted[itemID] = true
	}
	if !a.aiTicking[itemID] && s.Rendered() == "" && s.Buffer() != "" {
		a.advanceAIStream(itemID)
	}
	a.renderAIStream(itemID)
	if s.Rendered() == s.Buffer() {
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
	s := a.stream(itemID)
	if s == nil {
		return nil
	}
	a.aiTicking[itemID] = false
	if s.Rendered() != s.Buffer() {
		a.advanceAIStream(itemID)
		a.renderAIStream(itemID)
	}
	if s.Rendered() == s.Buffer() {
		if a.isWaitingForAI(itemID) {
			a.ensureWaitingVerb(itemID)
			a.advanceWaitingFrame(itemID)
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
	s := a.stream(itemID)
	if s == nil {
		return
	}
	target := []rune(s.Buffer())
	current := []rune(s.Rendered())
	if len(current) >= len(target) {
		s.SetRendered(string(target))
		return
	}
	next := len(current) + aiRunesPerTick
	if next > len(target) {
		next = len(target)
	}
	s.SetRendered(string(target[:next]))
}

func (a *App) renderAIStream(itemID string) {
	s := a.stream(itemID)
	if s == nil {
		return
	}
	label := s.Label()
	caughtUp := s.Rendered() == s.Buffer()
	completed := a.aiCompleted[itemID] && caughtUp
	item := TranscriptItem{ID: itemID, Kind: label, Text: s.Rendered(), Final: completed}
	if label == TranscriptTool {
		item.Title = a.aiToolTitle[itemID]
		item.Detail = a.aiToolDetail[itemID]
		item.Tool = a.aiToolState[itemID]
	}
	if label == aiTypeWaiting && !completed {
		item.Status = a.waitingStatus(itemID)
	}
	if label == aiTypeResponse && !completed && caughtUp {
		item.Status = a.waitingStatus(itemID)
	}
	focus := a.shouldFollowHistory(itemID)
	if idx, ok := a.aiIndexes[itemID]; ok {
		a.history.UpdateItemAt(idx, item, focus)
	} else {
		idx := a.history.AppendItem(item, focus)
		a.aiIndexes[itemID] = idx
	}
}

func (a *App) finalizeAIStream(itemID string) {
	s := a.stream(itemID)
	if s == nil {
		return
	}
	label := s.Label()
	final := TranscriptItem{ID: itemID, Kind: label, Text: s.Rendered(), Final: true}
	if label == TranscriptTool {
		final.Title = a.aiToolTitle[itemID]
		final.Detail = a.aiToolDetail[itemID]
		final.Tool = a.aiToolState[itemID]
	}
	focus := a.shouldFollowHistory(itemID)
	if idx, ok := a.aiIndexes[itemID]; ok {
		a.history.UpdateItemAt(idx, final, focus)
	}
	delete(a.streams, itemID)
	delete(a.aiVerbIdx, itemID)
	delete(a.aiWaitFrames, itemID)
	delete(a.aiCompleted, itemID)
	delete(a.aiTicking, itemID)
	delete(a.aiToolTitle, itemID)
	delete(a.aiToolDetail, itemID)
	delete(a.aiToolState, itemID)
}

func (a *App) shouldFollowHistory(_ string) bool {
	return true
}

func (a *App) waitingStatus(itemID string) string {
	return a.currentVerb(itemID) + " " + a.currentDots(itemID)
}

func (a *App) currentVerb(itemID string) string {
	if len(spinnerVerbs) == 0 {
		return "Thinking"
	}
	idx := a.aiVerbIdx[itemID] % len(spinnerVerbs)
	return spinnerVerbs[idx]
}

func (a *App) currentDots(itemID string) string {
	if len(spinnerDots) == 0 {
		return "..."
	}
	frame := a.aiWaitFrames[itemID]
	idx := (frame / 6) % len(spinnerDots)
	return spinnerDots[idx]
}

func (a *App) advanceWaitingFrame(itemID string) {
	frame := a.aiWaitFrames[itemID] + 1
	a.aiWaitFrames[itemID] = frame
}

func (a *App) ensureWaitingVerb(itemID string) {
	if len(spinnerVerbs) == 0 {
		return
	}
	if _, ok := a.aiVerbIdx[itemID]; ok {
		return
	}
	a.aiVerbIdx[itemID] = rand.IntN(len(spinnerVerbs))
}

func (a *App) startAIThinking(turnID string) tea.Cmd {
	if turnID == "" {
		return nil
	}
	itemID := "turn:" + turnID

	if idx, ok := a.aiIndexes[localOutboundPendingID]; ok {
		a.migrateAIStreamKeys(localOutboundPendingID, itemID)
		a.aiTurnPlaceholders[turnID] = itemID
		it := a.history.items[idx]
		it.ID = itemID
		a.history.UpdateItemAt(idx, it, true)
		a.turnPrimed[turnID] = struct{}{}
		a.renderAIStream(itemID)
		if a.aiTicking[itemID] {
			return nil
		}
		a.aiTicking[itemID] = true
		return a.scheduleAITick(itemID)
	}

	if _, ok := a.turnPrimed[turnID]; ok {
		return nil
	}
	a.turnPrimed[turnID] = struct{}{}
	a.aiTurnPlaceholders[turnID] = itemID
	a.ensureStream(itemID, aiTypeWaiting)
	a.ensureWaitingVerb(itemID)
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
	if s, ok := a.streams[placeholderID]; ok {
		a.streams[itemID] = s
		delete(a.streams, placeholderID)
	}
	if _, ok := a.aiTicking[placeholderID]; ok {
		a.aiTicking[itemID] = false
		delete(a.aiTicking, placeholderID)
	}
	if verbIdx, ok := a.aiVerbIdx[placeholderID]; ok {
		a.aiVerbIdx[itemID] = verbIdx
		delete(a.aiVerbIdx, placeholderID)
	}
	if waitFrames, ok := a.aiWaitFrames[placeholderID]; ok {
		a.aiWaitFrames[itemID] = waitFrames
		delete(a.aiWaitFrames, placeholderID)
	}
	delete(a.aiCompleted, placeholderID)
	delete(a.aiTurnPlaceholders, turnID)
}

func (a *App) isWaitingForAI(itemID string) bool {
	s := a.stream(itemID)
	if s == nil {
		return false
	}
	return s.Label() == aiTypeWaiting && s.Buffer() == "" && !a.aiCompleted[itemID]
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

// toolSummary is the structured form of a tool-call event — a title (e.g.
// "Read") and an optional detail (first line of output). setToolCallSummary
// writes these into the TranscriptItem's Title/Detail so ToolBlock can
// render them across two lines.
type toolSummary struct {
	Title  string
	Detail string
	State  ToolState
}

// handleCommandExecution routes shell command events into the tool-call
// rendering path. gpt-5 emits these as item/commandExecution/* instead of
// item/toolCall/*; we translate the phase (started, completed, output) into
// a toolSummary so the ToolBlock renderer can show "⏺ Ran" plus the command
// line, with the glyph color tracking pending → ok/err.
func (a *App) handleCommandExecution(ev agent.Event) tea.Cmd {
	phase, _ := ev.Payload["phase"].(string)
	if phase == "output" || phase == "outputDelta" {
		return nil
	}

	state := ToolPending
	final := false
	switch phase {
	case "completed":
		final = true
		state = ToolOK
		if code, ok := ev.Payload["exitCode"].(int); ok && code != 0 {
			state = ToolErr
		}
		if status, ok := ev.Payload["status"].(string); ok && strings.EqualFold(status, "failed") {
			state = ToolErr
		}
	}

	var summary toolSummary
	if args, ok := ev.Payload["commandArgs"].([]string); ok && len(args) > 0 {
		summary = classifyArgs(args)
	} else {
		cmdLine, _ := ev.Payload["command"].(string)
		summary = classifyCommand(cmdLine)
	}
	summary.State = state
	c1 := a.startAIThinking(ev.TurnID)
	c2 := a.setToolCallSummary(ev.ItemID, summary, final)
	return mergeTeaCmd(c1, c2)
}

// classifyCommand turns a raw shell command line into a single-line tool
// summary — verb + argument. Inspired by Codex: disguised reads (`cat`,
// `sed -n 'Np'`, `head`, `tail`) surface as `Read path` instead of the full
// `/bin/zsh -lc "..."` invocation, which used to pile up one two-line Ran
// entry per file read. Search and list flavors get the same treatment.
// Anything we can't classify falls through as `Ran <inner>`.
func classifyCommand(cmd string) toolSummary {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return toolSummary{Title: "Ran"}
	}
	args := shellTokens(cmd)
	if len(args) == 0 {
		return toolSummary{Title: "Ran " + truncateInline(cmd)}
	}
	return classifyArgs(args)
}

// classifyArgs runs the read/search/list detection on a pre-tokenized
// command. Codex provides commandArray for shell tool calls, which avoids
// the ambiguity of reparsing a string with embedded quotes.
func classifyArgs(args []string) toolSummary {
	args = unwrapShellArgs(args)
	if len(args) == 0 {
		return toolSummary{Title: "Ran"}
	}
	bin := baseBinary(args[0])
	rest := args[1:]

	switch bin {
	case "cat", "head", "tail", "less", "more", "bat":
		if target, ok := lastFileArg(rest); ok {
			return toolSummary{Title: "Read " + target}
		}
	case "sed":
		if isSedPrintOnly(rest) {
			if target, ok := lastFileArg(rest); ok {
				return toolSummary{Title: "Read " + target}
			}
		}
	case "awk":
		if target, ok := lastFileArg(rest); ok {
			return toolSummary{Title: "Read " + target}
		}
	case "rg", "grep", "ag", "ack":
		if pat := firstNonFlag(rest); pat != "" {
			return toolSummary{Title: "Searched " + truncateInline(pat)}
		}
	case "ls", "find", "fd", "tree":
		if path := firstNonFlag(rest); path != "" {
			return toolSummary{Title: "Listed " + path}
		}
		return toolSummary{Title: "Listed ."}
	case "git":
		if len(rest) > 0 {
			return toolSummary{Title: "Ran git " + truncateInline(strings.Join(rest, " "))}
		}
	}

	joined := strings.Join(args, " ")
	return toolSummary{Title: "Ran " + truncateInline(joined)}
}

// unwrapShellArgs strips a "/bin/sh -c SCRIPT" / "bash -lc SCRIPT" wrapper,
// re-tokenizing the inner script so classification runs against the command
// the model meant to run, not the shim the backend wraps around it.
func unwrapShellArgs(args []string) []string {
	if len(args) < 3 {
		return args
	}
	bin := baseBinary(args[0])
	switch bin {
	case "sh", "bash", "zsh", "dash", "ash":
	default:
		return args
	}
	for i := 1; i < len(args); i++ {
		tok := args[i]
		if strings.HasPrefix(tok, "-") {
			continue
		}
		inner := shellTokens(tok)
		if len(inner) > 0 {
			return inner
		}
		return args
	}
	return args
}

// shellTokens splits a shell command string into argv, honoring single and
// double quotes. It's intentionally minimal — no parameter expansion, no
// backslash escapes beyond pass-through — which is enough for classifying
// the command shapes we care about here.
func shellTokens(s string) []string {
	var out []string
	var buf strings.Builder
	inSingle, inDouble := false, false
	flush := func() {
		if buf.Len() > 0 {
			out = append(out, buf.String())
			buf.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case ch == ' ' && !inSingle && !inDouble:
			flush()
		default:
			buf.WriteByte(ch)
		}
	}
	flush()
	return out
}

// baseBinary returns the basename of a command like "/bin/zsh" → "zsh".
func baseBinary(s string) string {
	if idx := strings.LastIndexByte(s, '/'); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

// lastFileArg returns the final non-flag token, treating it as the file the
// read-like command targets. Used by cat/head/tail/sed/awk classifiers.
func lastFileArg(args []string) (string, bool) {
	for i := len(args) - 1; i >= 0; i-- {
		tok := args[i]
		if tok == "" || strings.HasPrefix(tok, "-") {
			continue
		}
		return strings.Trim(tok, `"'`), true
	}
	return "", false
}

func firstNonFlag(args []string) string {
	for _, tok := range args {
		if tok == "" || strings.HasPrefix(tok, "-") {
			continue
		}
		return strings.Trim(tok, `"'`)
	}
	return ""
}

// isSedPrintOnly reports whether a sed invocation is a pure `-n '...p'`
// excerpt, which we treat as a Read rather than a generic Ran.
func isSedPrintOnly(args []string) bool {
	hasN := false
	hasPrint := false
	for _, tok := range args {
		if tok == "-n" {
			hasN = true
			continue
		}
		stripped := strings.Trim(tok, `"'`)
		if strings.HasSuffix(stripped, "p") && (strings.ContainsAny(stripped, "0123456789") || strings.HasPrefix(stripped, "/")) {
			hasPrint = true
		}
	}
	return hasN && hasPrint
}

func truncateInline(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 90 {
		return s
	}
	return s[:87] + "..."
}

func summarizeToolCall(ev agent.Event) toolSummary {
	name, _ := ev.Payload["name"].(string)
	text := strings.TrimSpace(sanitizeHistoryMessage(ev.Text))
	lower := strings.ToLower(name)

	state := ToolOK
	if ev.Payload != nil {
		if _, bad := ev.Payload["error"]; bad {
			state = ToolErr
		}
	}

	mk := func(title, detail string) toolSummary {
		return toolSummary{Title: strings.TrimSpace(title), Detail: strings.TrimSpace(detail), State: state}
	}

	switch lower {
	case "read":
		return mk("Read", summarizeToolDetail(text))
	case "web_search", "websearch":
		return mk("Web search", summarizeToolDetail(text))
	case "glob", "list", "ls":
		return mk("Explored", summarizeToolDetail(text))
	case "grep", "search":
		return mk("Searched", summarizeToolDetail(text))
	case "bash", "command", "shell":
		return mk("Ran", firstLine(text))
	case "write", "edit", "apply_patch", "applypatch":
		return mk("Edited", summarizeToolDetail(text))
	case "question":
		return mk("Asked", summarizeToolDetail(text))
	case "task":
		return mk("Delegated", summarizeToolDetail(text))
	default:
		if name == "" {
			return mk("Working", summarizeToolDetail(text))
		}
		return mk(toTitleLabel(name), summarizeToolDetail(text))
	}
}

func summarizeToolDetail(s string) string {
	if s == "" {
		return ""
	}
	line := firstLine(s)
	if len(line) > 90 {
		line = line[:87] + "..."
	}
	return line
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

func toTitleLabel(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "_", " "))
	if s == "" {
		return "Working"
	}
	return strings.ToUpper(s[:1]) + s[1:]
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
	a.compose.SetActive(a.activePane == PaneCompose)

	// Compose height = borders (2) + body lines, clamped.
	composeHeight := a.compose.BodyLineCount(a.width) + 2
	if composeHeight < 3 {
		composeHeight = 3
	}
	if composeHeight > 10 {
		composeHeight = 10
	}
	if a.height > 0 {
		maxAllowed := max(a.height-3, 3) // leave room for topbar + statusline + some history
		if composeHeight > maxAllowed {
			composeHeight = maxAllowed
		}
	}

	var inputView string
	var inputLines int

	if a.mode == ModeCommandLine {
		inputView = a.renderCommandInput()
		inputLines = strings.Count(inputView, "\n") + 1
	} else {
		inputView, inputLines = a.compose.View(a.width, composeHeight, a.modeLabel())
	}

	topbar := renderTopbar(a.chromeSnapshot())
	statusline := renderStatusline(a.chromeSnapshot())

	// History fills everything between topbar and input + statusline.
	// Reserve 1 row each for topbar and statusline.
	historyHeight := max(a.height-inputLines-2, 1)

	a.history.width = a.width
	a.history.height = historyHeight
	a.history.SetActive(a.activePane == PaneHistory)
	historyView := a.history.View()

	historyView = padToWidth(historyView, a.width)
	inputView = padToWidth(inputView, a.width)
	topbar = padToWidth(topbar, a.width)
	statusline = padToWidth(statusline, a.width)

	parts := []string{topbar, historyView, inputView, statusline}
	frame := lipgloss.JoinVertical(lipgloss.Left, parts...)
	lines := strings.Count(frame, "\n") + 1
	if a.height > 0 && lines < a.height {
		padding := strings.Repeat("\n", a.height-lines)
		frame += padding
	}
	return frame
}

// renderCommandInput paints the ":…" prompt row shown while command mode
// is active. It mimics the compose frame height (3 rows) so the statusline
// doesn't jitter between modes.
func (a *App) renderCommandInput() string {
	theme := a.theme
	borderStyle := lipgloss.NewStyle().Foreground(theme.ModeColor(ModeCommandLine))
	innerCols := max(a.width-2, 1)
	top := borderStyle.Render("╭" + strings.Repeat("─", innerCols) + "╮")
	bot := borderStyle.Render("╰" + strings.Repeat("─", innerCols) + "╯")

	cmdView := a.command.View() // starts with ":"
	cmdView = lipgloss.NewStyle().Foreground(theme.FgBright).Render(cmdView)
	// The command view itself contains an ANSI cursor; pad to innerCols−2
	// so the right border sits flush.
	cmdWidth := lipgloss.Width(cmdView)
	want := innerCols - 2
	if want < 0 {
		want = 0
	}
	pad := ""
	if cmdWidth < want {
		pad = strings.Repeat(" ", want-cmdWidth)
	}
	row := borderStyle.Render("│") + " " + cmdView + pad + " " + borderStyle.Render("│")
	return top + "\n" + row + "\n" + bot
}

// chromeSnapshot collects the fields renderTopbar / renderStatusline need.
func (a *App) chromeSnapshot() chromeSnapshot {
	cwd, _ := os.Getwd()
	return chromeSnapshot{
		theme:           a.theme,
		width:           a.width,
		path:            shortPath(cwd),
		title:           a.title,
		sessionID:       a.sessionID,
		mode:            a.mode,
		scope:           a.scopeLabel(),
		branch:          a.branch,
		provider:        a.provider,
		model:           a.model,
		ctxUsed:         a.ctxUsed,
		ctxTotal:        a.ctxTotal,
		msgCount:        len(a.history.items),
		notice:          a.statusNotice,
		lineCol:         a.lineColLabel(),
		pendingApproval: len(a.pendingRPCID) > 0,
	}
}

// scopeLabel returns the statusline scope label. History pane uses CHAT,
// compose pane uses PROMPT — the design's two-scope indicator.
func (a *App) scopeLabel() string {
	if a.activePane == PaneHistory {
		return "CHAT"
	}
	return "PROMPT"
}

// lineColLabel reports a vim-style line:col indicator for the compose
// buffer when that's the active pane.
func (a *App) lineColLabel() string {
	if a.activePane != PaneCompose {
		return ""
	}
	line, col := a.compose.lineAndColumn()
	return fmt.Sprintf("%d:%d", line+1, col)
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

func (a *App) modeLabel() string {
	switch a.mode {
	case ModeInsert:
		return "INSERT"
	case ModeVisual:
		return "VISUAL"
	case ModeCommandLine:
		return "COMMAND"
	default:
		return ""
	}
}
