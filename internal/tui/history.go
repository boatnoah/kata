package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
)

type itemRenderCache struct {
	text   string
	title  string
	detail string
	kind   TranscriptKind
	tool   ToolState
	role   ToolRole
	final  bool
	width  int
	lines  []string
}

type HistoryScreen struct {
	items  []TranscriptItem
	active bool
	width  int
	height int
	theme  Theme

	// Viewport state. topLine is the first rendered-line index visible at the
	// top of the pane. followTail means topLine auto-tracks new content so
	// the newest line stays at the bottom of the visible area. Manual scroll
	// clears followTail; scrolling back to the bottom re-arms it.
	topLine    int
	followTail bool

	// CHAT-scope vim state. cursorLine is a rendered-line index; < 0 means
	// "not yet positioned". visual* track a line-wise selection anchored at
	// visualAnchor and extending to cursorLine. Yanking collects item-level
	// text for the items whose rendered lines overlap the selection.
	cursorLine   int
	visualActive bool
	visualAnchor int

	// Render cache — avoids re-running the block pipeline for unchanged items.
	renderCache []itemRenderCache

	// Assembled lines cache — avoids rebuilding the full line slice multiple
	// times per frame. Cleared whenever items or width change.
	linesCache      []renderedHistoryLine
	linesCacheValid bool
}

func NewHistoryScreen() *HistoryScreen {
	return &HistoryScreen{items: []TranscriptItem{}, followTail: true, theme: DefaultTheme(), cursorLine: -1}
}

// SetTheme swaps the active palette. Theme changes invalidate the line
// cache so re-rendered items pick up the new colors.
func (h *HistoryScreen) SetTheme(t Theme) {
	h.theme = t
	h.invalidateLines()
	// Per-item render caches bake theme-colored ANSI into their output, so
	// discard them too.
	for i := range h.renderCache {
		h.renderCache[i] = itemRenderCache{}
	}
}

func (h *HistoryScreen) OnWindowSize(width, height int) {
	if h.width != width {
		h.invalidateLines()
	}
	h.width = width
	h.height = height
}

func (h *HistoryScreen) SetActive(active bool) {
	h.active = active
}

func (h *HistoryScreen) AppendItem(item TranscriptItem, _ bool) int {
	item.Text = sanitizeHistoryMessage(item.Text)
	if item.Text == "" && item.Kind != TranscriptThinking && item.Kind != TranscriptTool && item.Kind != TranscriptDiff {
		return len(h.items) - 1
	}
	h.items = append(h.items, item)
	h.invalidateLines()
	return len(h.items) - 1
}

func (h *HistoryScreen) UpdateItemAt(index int, item TranscriptItem, _ bool) {
	if index < 0 || index >= len(h.items) {
		return
	}
	item.Text = sanitizeHistoryMessage(item.Text)

	old := h.items[index]
	contentChanged := old.Text != item.Text ||
		old.Kind != item.Kind ||
		old.Final != item.Final ||
		old.Title != item.Title ||
		old.Detail != item.Detail ||
		old.Tool != item.Tool

	h.items[index] = item

	if contentChanged {
		h.invalidateLines()
	} else if old.Status != item.Status {
		if (old.Status == "") != (item.Status == "") {
			h.invalidateLines()
		} else if item.Status != "" && h.linesCacheValid {
			h.updateCachedStatus(index, item)
		}
	}
}

// updateCachedStatus patches the status line for an item in-place in the flat
// lines cache, avoiding a full rebuild for spinner-dot changes.
func (h *HistoryScreen) updateCachedStatus(index int, item TranscriptItem) {
	styled := lipgloss.NewStyle().Foreground(h.theme.FgDim).Render(item.Status)
	lastLine := -1
	for i, l := range h.linesCache {
		if l.itemIndex == index {
			lastLine = i
		}
	}
	if lastLine >= 0 {
		h.linesCache[lastLine] = renderedHistoryLine{text: styled, itemIndex: index}
	}
}

func (h *HistoryScreen) View() string {
	lines := h.renderedLines()
	total := len(lines)

	// Clamp cursor against rebuilt line set (items or width may have changed).
	if h.cursorLine >= total {
		h.cursorLine = total - 1
	}
	if h.cursorLine < -1 {
		h.cursorLine = -1
	}
	if h.visualActive {
		if h.visualAnchor < 0 {
			h.visualAnchor = 0
		}
		if h.visualAnchor >= total {
			h.visualAnchor = total - 1
		}
	}

	if h.height <= 0 {
		return h.paintLines(lines, 0, total)
	}

	// When following the tail, snap topLine so the newest content is at the
	// bottom of the visible area. When manually scrolled, leave topLine alone
	// but still clamp it against the current content so resizes don't orphan
	// the view past the end.
	maxTop := max(total-h.height, 0)
	if h.followTail {
		h.topLine = maxTop
	} else if h.topLine > maxTop {
		h.topLine = maxTop
	}
	if h.topLine < 0 {
		h.topLine = 0
	}

	// Keep cursor visible when it's set: scroll the viewport to include it
	// without disturbing followTail semantics.
	if h.cursorLine >= 0 {
		if h.cursorLine < h.topLine {
			h.topLine = h.cursorLine
		} else if h.cursorLine >= h.topLine+h.height {
			h.topLine = h.cursorLine - h.height + 1
		}
	}

	end := h.topLine + h.height
	if end > total {
		end = total
	}
	return h.paintLines(lines[h.topLine:end], h.topLine, end)
}

// paintLines renders each line with history's indent and pads to h.height.
// startIdx is the absolute rendered-line index of lines[0] so the cursor row
// and visual selection can be decorated when this pane is active. The result
// has EXACTLY h.height rows joined by h.height-1 newlines — no trailing \n —
// so lipgloss.JoinVertical in app.View doesn't treat the block as one row
// taller than the pane.
func (h *HistoryScreen) paintLines(lines []renderedHistoryLine, startIdx, _ int) string {
	lo, hi, hasSel := h.selectionRange()
	reverse := lipgloss.NewStyle().Reverse(true)

	total := h.height
	if total <= 0 {
		total = len(lines)
	}
	rows := make([]string, 0, total)
	for i, line := range lines {
		absIdx := startIdx + i
		row := "  " + line.text
		if h.active {
			selected := hasSel && absIdx >= lo && absIdx <= hi
			isCursor := absIdx == h.cursorLine
			if selected || isCursor {
				// Strip ANSI first so the reverse style applies cleanly
				// without fighting nested resets. Then pad to the full pane
				// width before reversing so the highlight fills the row —
				// otherwise the invert box bounces between content widths
				// as the cursor moves and reads as flicker.
				plain := stripANSIForLayout(row)
				if h.width > 0 {
					pw := lipgloss.Width(plain)
					if pw < h.width {
						plain += strings.Repeat(" ", h.width-pw)
					}
				}
				row = reverse.Render(plain)
			}
		}
		rows = append(rows, row)
	}
	for len(rows) < total {
		rows = append(rows, "")
	}
	return strings.Join(rows, "\n")
}

// selectionRange returns the [lo, hi] inclusive rendered-line range that is
// currently selected. When not in visual mode this returns (0,0,false).
func (h *HistoryScreen) selectionRange() (int, int, bool) {
	if !h.visualActive {
		return 0, 0, false
	}
	lo, hi := h.visualAnchor, h.cursorLine
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi, true
}

// ---- scroll API ----

// maxTopLine is the largest valid topLine for the current content and pane
// height — i.e. the offset at which the bottom of the content sits on the
// bottom row of the pane.
func (h *HistoryScreen) maxTopLine() int {
	if h.height <= 0 {
		return 0
	}
	return max(len(h.renderedLines())-h.height, 0)
}

// ScrollUp scrolls the viewport up by n lines (toward older content).
func (h *HistoryScreen) ScrollUp(n int) {
	if n <= 0 {
		return
	}
	h.topLine -= n
	if h.topLine < 0 {
		h.topLine = 0
	}
	h.followTail = false
}

// ScrollDown scrolls the viewport down by n lines (toward newer content).
// If the scroll reaches the bottom, auto-follow is re-armed.
func (h *HistoryScreen) ScrollDown(n int) {
	if n <= 0 {
		return
	}
	m := h.maxTopLine()
	h.topLine += n
	if h.topLine >= m {
		h.topLine = m
		h.followTail = true
	} else {
		h.followTail = false
	}
}

// ScrollHalfPageUp scrolls up by half the visible pane height.
func (h *HistoryScreen) ScrollHalfPageUp() { h.ScrollUp(max(h.height/2, 1)) }

// ScrollHalfPageDown scrolls down by half the visible pane height.
func (h *HistoryScreen) ScrollHalfPageDown() { h.ScrollDown(max(h.height/2, 1)) }

// ScrollToTop jumps to the first line and clears auto-follow.
func (h *HistoryScreen) ScrollToTop() {
	h.topLine = 0
	h.followTail = false
}

// ScrollToBottom jumps to the newest content and arms auto-follow.
func (h *HistoryScreen) ScrollToBottom() {
	h.followTail = true
}

// ---- CHAT scope cursor API ----

// EnsureCursor positions the cursor at the bottom-most rendered line if it
// hasn't been set yet. Called by the App when CHAT scope is first focused
// so the user sees a cursor without having to move first.
func (h *HistoryScreen) EnsureCursor() {
	if h.cursorLine >= 0 {
		return
	}
	total := len(h.renderedLines())
	if total == 0 {
		h.cursorLine = 0
		return
	}
	h.cursorLine = total - 1
}

// CursorUp moves the cursor up n rendered lines.
func (h *HistoryScreen) CursorUp(n int) {
	if n <= 0 {
		return
	}
	h.EnsureCursor()
	h.cursorLine -= n
	if h.cursorLine < 0 {
		h.cursorLine = 0
	}
	h.followTail = false
}

// CursorDown moves the cursor down n rendered lines, clamping at the bottom.
// When the cursor lands at the last line, followTail re-arms so new content
// keeps the cursor pinned at the tail.
func (h *HistoryScreen) CursorDown(n int) {
	if n <= 0 {
		return
	}
	h.EnsureCursor()
	total := len(h.renderedLines())
	if total == 0 {
		return
	}
	h.cursorLine += n
	if h.cursorLine >= total-1 {
		h.cursorLine = total - 1
		h.followTail = true
	} else {
		h.followTail = false
	}
}

// CursorTop jumps to the first rendered line and clears auto-follow.
func (h *HistoryScreen) CursorTop() {
	h.cursorLine = 0
	h.followTail = false
}

// CursorBottom jumps to the last rendered line and arms auto-follow.
func (h *HistoryScreen) CursorBottom() {
	total := len(h.renderedLines())
	if total == 0 {
		h.cursorLine = 0
	} else {
		h.cursorLine = total - 1
	}
	h.followTail = true
}

// EnterVisual starts a visual selection anchored at the current cursor line.
func (h *HistoryScreen) EnterVisual() {
	h.EnsureCursor()
	h.visualActive = true
	h.visualAnchor = h.cursorLine
}

// ExitVisual clears the visual selection but leaves the cursor in place.
func (h *HistoryScreen) ExitVisual() {
	h.visualActive = false
	h.visualAnchor = 0
}

// VisualActive reports whether a visual selection is currently in progress.
func (h *HistoryScreen) VisualActive() bool { return h.visualActive }

// YankCurrentLine copies the single rendered line under the cursor, stripped
// of ANSI styling and leading indent. This is the vim `yy` motion in CHAT
// scope — grab exactly what the cursor is sitting on.
func (h *HistoryScreen) YankCurrentLine() {
	lines := h.renderedLines()
	if h.cursorLine < 0 || h.cursorLine >= len(lines) {
		return
	}
	plain := strings.TrimLeft(stripANSIForLayout(lines[h.cursorLine].text), " ")
	if plain == "" {
		return
	}
	copyToClipboard(plain)
}

// YankSelection copies the rendered lines covered by the current selection,
// stripped of ANSI styling and the 2-col indent. Line-wise: whatever is
// highlighted is what gets yanked — no item-level expansion.
func (h *HistoryScreen) YankSelection() {
	lo, hi, ok := h.selectionRange()
	if !ok {
		return
	}
	lines := h.renderedLines()
	if lo < 0 {
		lo = 0
	}
	if hi >= len(lines) {
		hi = len(lines) - 1
	}
	if lo > hi {
		h.ExitVisual()
		return
	}
	out := make([]string, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		out = append(out, strings.TrimLeft(stripANSIForLayout(lines[i].text), " "))
	}
	// Trim trailing blank lines so the clipboard doesn't carry padding.
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	if len(out) > 0 {
		copyToClipboard(strings.Join(out, "\n"))
	}
	h.ExitVisual()
}

type renderedHistoryLine struct {
	text      string
	itemIndex int
}

func (h *HistoryScreen) renderedLines() []renderedHistoryLine {
	if h.linesCacheValid {
		return h.linesCache
	}
	contentWidth := h.renderContentWidth()
	for len(h.renderCache) < len(h.items) {
		h.renderCache = append(h.renderCache, itemRenderCache{})
	}
	h.renderCache = h.renderCache[:len(h.items)]

	var out []renderedHistoryLine
	for i, item := range h.items {
		role := h.toolRoleAt(i)
		lines := h.cachedRenderItem(i, item, contentWidth, role)
		if len(lines) == 0 {
			continue
		}
		for _, line := range lines {
			out = append(out, renderedHistoryLine{text: line, itemIndex: i})
		}
		if i < len(h.items)-1 && h.separatorBetween(item, h.items[i+1]) {
			out = append(out, renderedHistoryLine{text: "", itemIndex: i})
		}
	}
	h.linesCache = out
	h.linesCacheValid = true
	return out
}

// separatorBetween decides whether to insert a blank line between two
// adjacent items. Consecutive tool calls stack tightly; consecutive
// thinking+tool also stack. Everything else gets a breathing-room gap.
func (h *HistoryScreen) separatorBetween(a, b TranscriptItem) bool {
	tight := func(k TranscriptKind) bool {
		return k == TranscriptTool || k == TranscriptThinking
	}
	if tight(a.Kind) && tight(b.Kind) {
		return false
	}
	return true
}

func (h *HistoryScreen) invalidateLines() {
	h.linesCacheValid = false
}

func (h *HistoryScreen) cachedRenderItem(index int, item TranscriptItem, width int, role ToolRole) []string {
	c := &h.renderCache[index]
	if c.text == item.Text &&
		c.title == item.Title &&
		c.detail == item.Detail &&
		c.kind == item.Kind &&
		c.tool == item.Tool &&
		c.role == role &&
		c.final == item.Final &&
		c.width == width &&
		c.lines != nil {
		if item.Status != "" {
			return h.applyStatus(c.lines, item.Status, item.Kind)
		}
		return c.lines
	}
	saved := item.Status
	item.Status = ""
	lines := h.renderItemWithRole(item, width, role)
	item.Status = saved
	c.text = item.Text
	c.title = item.Title
	c.detail = item.Detail
	c.kind = item.Kind
	c.tool = item.Tool
	c.role = role
	c.final = item.Final
	c.width = width
	c.lines = lines
	if item.Status != "" {
		return h.applyStatus(lines, item.Status, item.Kind)
	}
	return lines
}

// applyStatus appends a styled status line to the cached lines. For items
// whose body is empty, the status replaces the empty placeholder entirely.
// Otherwise the status line is appended directly beneath the body with no
// extra blank — item-level spacing is handled in renderedLines.
func (h *HistoryScreen) applyStatus(cached []string, status string, _ TranscriptKind) []string {
	styled := lipgloss.NewStyle().Foreground(h.theme.FgDim).Render(status)
	if len(cached) == 0 || (len(cached) == 1 && strings.TrimSpace(stripANSIForLayout(cached[0])) == "") {
		return []string{styled}
	}
	out := make([]string, 0, len(cached)+1)
	out = append(out, cached...)
	out = append(out, styled)
	return out
}

func stripANSIForLayout(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch != 0x1b {
			b.WriteByte(ch)
			continue
		}
		// ESC starts one of several sequences. We care about three shapes:
		//   CSI — ESC [ … final (0x40-0x7E)
		//   OSC — ESC ] … ST (ESC \) or BEL (0x07)
		//   two-byte — ESC + single final byte
		// OSC 8 hyperlinks live inside OSC; skipping past the ST terminator is
		// what lets yank/selection see the label text without the URL bleeding
		// through as if it were ordinary prose.
		if i+1 >= len(s) {
			return b.String()
		}
		kind := s[i+1]
		i++
		switch kind {
		case '[':
			for i+1 < len(s) {
				i++
				c := s[i]
				if (c >= 0x40 && c <= 0x7E) {
					break
				}
			}
		case ']':
			for i+1 < len(s) {
				i++
				c := s[i]
				if c == 0x07 {
					break
				}
				if c == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
					i++
					break
				}
			}
		default:
			// two-byte sequence like ESC = or ESC \; nothing more to consume.
		}
	}
	return b.String()
}

// renderItem produces the rendered lines for a single transcript item. It
// dispatches the item to a set of Blocks, renders each, and inserts a
// single blank line between adjacent non-empty block outputs.
func (h *HistoryScreen) renderItem(item TranscriptItem, width int) []string {
	return h.renderItemWithRole(item, width, ToolRoleSingle)
}

func (h *HistoryScreen) renderItemWithRole(item TranscriptItem, width int, role ToolRole) []string {
	blocks := itemBlocksWithRole(item, role)
	var out []string
	for i, block := range blocks {
		lines := block.Render(width, h.theme)
		if len(lines) == 0 {
			continue
		}
		if i > 0 && len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, lines...)
	}
	for len(out) > 0 && strings.TrimSpace(stripANSIForLayout(out[len(out)-1])) == "" {
		out = out[:len(out)-1]
	}
	start := 0
	for start < len(out) && strings.TrimSpace(stripANSIForLayout(out[start])) == "" {
		start++
	}
	if start > 0 {
		out = out[start:]
	}
	return out
}

// itemBlocks maps a TranscriptItem onto the blocks that represent it.
// Assistant items produce multiple blocks when the response contains
// fenced code; every other kind currently maps to a single block.
func itemBlocks(item TranscriptItem) []Block {
	return itemBlocksWithRole(item, ToolRoleSingle)
}

func itemBlocksWithRole(item TranscriptItem, role ToolRole) []Block {
	switch item.Kind {
	case TranscriptUser:
		return []Block{UserBlock{Text: item.Text}}
	case TranscriptAssistant:
		if item.Final {
			return ParseAssistantText(item.Text)
		}
		if strings.TrimSpace(item.Text) == "" {
			return nil
		}
		return []Block{ProseBlock{Text: item.Text}}
	case TranscriptThinking:
		return []Block{ThinkingBlock{Text: item.Text}}
	case TranscriptTool:
		title, detail := item.Title, item.Detail
		if title == "" {
			title = firstLine(item.Text)
		}
		return []Block{ToolBlock{Title: title, Detail: detail, State: item.Tool, Role: role}}
	case TranscriptDiff:
		return []Block{DiffBlock{File: item.Diff, MaxLines: item.DiffMaxLines}}
	case TranscriptError:
		return []Block{ErrorBlock{Text: item.Text}}
	case TranscriptSystem:
		return []Block{SystemBlock{Text: item.Text}}
	default:
		return []Block{SystemBlock{Text: item.Text}}
	}
}

// toolRoleAt determines whether item i is the start of, inside of, or
// unrelated to a run of adjacent same-verb tool items. Used by the render
// loop to collapse "Read one.go / Read two.go / Read three.go" into a
// single `⏺ Read` header followed by `⎿ <file>` continuation lines.
func (h *HistoryScreen) toolRoleAt(i int) ToolRole {
	curr := h.items[i]
	if curr.Kind != TranscriptTool {
		return ToolRoleSingle
	}
	verb, arg := toolVerbArg(curr)
	if verb == "" || arg == "" {
		return ToolRoleSingle
	}

	prevSame := i > 0 && toolMatchesVerb(h.items[i-1], verb)
	nextSame := i+1 < len(h.items) && toolMatchesVerb(h.items[i+1], verb)

	switch {
	case prevSame:
		return ToolRoleContinuation
	case nextSame:
		return ToolRoleGroupStart
	default:
		return ToolRoleSingle
	}
}

// toolVerbArg extracts the verb and argument from a tool item, using the
// same Title→Text fallback as itemBlocks so behavior stays consistent
// regardless of how the item was constructed.
func toolVerbArg(item TranscriptItem) (string, string) {
	if item.Kind != TranscriptTool {
		return "", ""
	}
	title := item.Title
	if title == "" {
		title = firstLine(item.Text)
	}
	return splitTitle(title)
}

func toolMatchesVerb(item TranscriptItem, verb string) bool {
	if item.Kind != TranscriptTool {
		return false
	}
	v, arg := toolVerbArg(item)
	return arg != "" && v == verb
}

// truncateToWidth shortens s to fit within maxWidth display cells, ending in
// an ellipsis if truncated. Operates on runes; callers pass pre-styled strings
// only when they know the ANSI won't be sliced (short tool titles, etc.).
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	const ellipsis = "…"
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + ellipsis
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}
	return ellipsis
}

func (h *HistoryScreen) renderContentWidth() int {
	if h.width <= 0 {
		return 0
	}
	width := h.width - 4
	if width < 12 {
		return 12
	}
	return width
}

func wrapToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		// Word-wrap on spaces first so we don't split words. Any line that
		// still exceeds width (e.g. a URL longer than the pane) gets
		// hard-wrapped via reflow/wrap as a backstop.
		wrappedLine := wrap.String(wordwrap.String(line, width), width)
		wrapped = append(wrapped, strings.Split(wrappedLine, "\n")...)
	}
	return strings.Join(wrapped, "\n")
}
