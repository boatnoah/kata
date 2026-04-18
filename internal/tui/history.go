package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	glamourstyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

// Style palette for history items.
var (
	stylePrompt   = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)
	styleUserText = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Bold(true)
	styleMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleError    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

type itemRenderCache struct {
	text  string
	kind  TranscriptKind
	final bool
	width int
	lines []string
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

	// Render cache — avoids re-running glamour for unchanged items.
	renderCache []itemRenderCache

	// Assembled lines cache — avoids rebuilding the full line slice multiple
	// times per frame. Cleared whenever items or width change.
	linesCache      []renderedHistoryLine
	linesCacheValid bool

	// Glamour renderer cache, keyed by render width.
	glamourWidth    int
	glamourRenderer *glamour.TermRenderer
}

func NewHistoryScreen() *HistoryScreen {
	return &HistoryScreen{items: []TranscriptItem{}, followTail: true, theme: DefaultTheme()}
}

// SetTheme swaps the active palette. Theme changes invalidate the line
// cache so re-rendered items pick up the new colors.
func (h *HistoryScreen) SetTheme(t Theme) {
	h.theme = t
	h.invalidateLines()
	h.glamourRenderer = nil
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
	if item.Text == "" && item.Kind != TranscriptThinking {
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
	contentChanged := old.Text != item.Text || old.Kind != item.Kind || old.Final != item.Final

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
	styled := styleMuted.Render(item.Status)
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

	if h.height <= 0 {
		return h.paintLines(lines)
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

	end := h.topLine + h.height
	if end > total {
		end = total
	}
	return h.paintLines(lines[h.topLine:end])
}

// paintLines renders each line with history's indent and pads to h.height.
func (h *HistoryScreen) paintLines(lines []renderedHistoryLine) string {
	var b strings.Builder
	rendered := 0
	for _, line := range lines {
		fmt.Fprintf(&b, "  %s\n", line.text)
		rendered++
	}
	if h.height > 0 {
		for rendered < h.height {
			b.WriteRune('\n')
			rendered++
		}
	}
	return b.String()
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
		lines := h.cachedRenderItem(i, item, contentWidth)
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

func (h *HistoryScreen) cachedRenderItem(index int, item TranscriptItem, width int) []string {
	c := &h.renderCache[index]
	if c.text == item.Text && c.kind == item.Kind && c.final == item.Final && c.width == width && c.lines != nil {
		if item.Status != "" {
			return h.applyStatus(c.lines, item.Status, item.Kind)
		}
		return c.lines
	}
	saved := item.Status
	item.Status = ""
	lines := h.renderItem(item, width)
	item.Status = saved
	c.text = item.Text
	c.kind = item.Kind
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
	styled := styleMuted.Render(status)
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
	inEscape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inEscape {
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b {
			inEscape = true
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func (h *HistoryScreen) renderItem(item TranscriptItem, width int) []string {
	switch item.Kind {
	case TranscriptUser:
		return splitRenderedBlock(h.renderUser(item.Text, width))
	case TranscriptAssistant:
		var body string
		if item.Final {
			body = h.renderAssistantMarkdown(item.Text, width, true)
		} else {
			body = h.renderAssistantMarkdown(item.Text, width, false)
		}
		return splitRenderedBlock(body)
	case TranscriptThinking:
		return splitRenderedBlock(styleMuted.Italic(true).Render(item.Text))
	case TranscriptTool:
		return splitRenderedBlock(h.renderTool(item.Text, width))
	case TranscriptError:
		return splitRenderedBlock(styleError.Render(wrapToWidth(item.Text, width)))
	case TranscriptSystem:
		return splitRenderedBlock(styleMuted.Render(wrapToWidth(item.Text, width)))
	default:
		return splitRenderedBlock(wrapToWidth(item.Text, width))
	}
}

func (h *HistoryScreen) renderUser(text string, width int) string {
	prompt := stylePrompt.Render("❯")
	wrapped := wrapToWidth(text, max(width-2, 8))
	lines := strings.Split(wrapped, "\n")
	for i, line := range lines {
		lines[i] = styleUserText.Render(line)
	}
	lines[0] = prompt + " " + lines[0]
	for i := 1; i < len(lines); i++ {
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}

func (h *HistoryScreen) renderAssistantMarkdown(text string, width int, useGlamour bool) string {
	if width <= 0 {
		return text
	}
	markdownWidth := width - 4
	if markdownWidth < 20 {
		markdownWidth = width
	}
	if !useGlamour {
		return wrapToWidth(text, markdownWidth)
	}
	r := h.glamourFor(markdownWidth)
	if r == nil {
		return wrapToWidth(text, markdownWidth)
	}
	out, err := r.Render(text)
	if err != nil {
		return wrapToWidth(text, markdownWidth)
	}
	return strings.TrimRight(out, "\n")
}

// glamourFor returns a cached glamour renderer for the given width, rebuilding
// only when the width changes.
func (h *HistoryScreen) glamourFor(width int) *glamour.TermRenderer {
	if h.glamourRenderer != nil && h.glamourWidth == width {
		return h.glamourRenderer
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourstyles.DarkStyle),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	h.glamourRenderer = r
	h.glamourWidth = width
	return r
}

// renderTool produces a compact single-line tool summary like "⎿ Read go.mod".
// The summary arrives pre-assembled from summarizeToolCall; we strip everything
// after the first newline so the pane stays clean and truncate on display
// width (rune-safe) instead of bytes.
func (h *HistoryScreen) renderTool(text string, width int) string {
	summary := strings.TrimSpace(firstLine(text))
	full := "⎿ " + summary
	if width > 0 {
		full = truncateToWidth(full, width)
	}
	return styleMuted.Render(full)
}

// truncateToWidth shortens s to fit within maxWidth display cells, ending in
// an ellipsis if truncated. Operates on runes, ignoring ANSI (none expected
// here since renderTool truncates before styling).
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
		wrapped = append(wrapped, strings.Split(wrap.String(line, width), "\n")...)
	}
	return strings.Join(wrapped, "\n")
}

// splitRenderedBlock trims surrounding blank lines and returns the remaining
// lines. Returns nil for an empty body so the caller can skip the item
// entirely rather than reserving a blank row.
func splitRenderedBlock(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	start := 0
	for start < len(lines) && strings.TrimSpace(stripANSIForLayout(lines[start])) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(stripANSIForLayout(lines[end-1])) == "" {
		end--
	}
	if start >= end {
		return nil
	}
	return lines[start:end]
}
