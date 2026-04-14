package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	glamourstyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

type itemRenderCache struct {
	text  string
	kind  TranscriptKind
	final bool
	width int
	lines []string
}

type HistoryScreen struct {
	items    []TranscriptItem
	active   bool
	width    int
	height   int
	atBottom bool // tracks whether we were at the bottom before an update

	// Render cache — avoids re-running glamour for unchanged items.
	renderCache []itemRenderCache

	// Assembled lines cache — avoids rebuilding the full line slice multiple
	// times per frame. Cleared whenever items or width change.
	linesCache      []renderedHistoryLine
	linesCacheValid bool
}

func NewHistoryScreen() *HistoryScreen {
	return &HistoryScreen{items: []TranscriptItem{}, atBottom: true}
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
	h.atBottom = true
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
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(item.Status)
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
	var b strings.Builder
	lines := h.renderedLines()
	total := len(lines)

	start := 0
	end := total
	if h.height > 0 && total > h.height {
		// Auto-scroll: always show the bottom.
		start = total - h.height
		end = total
	}

	rendered := 0
	for i := start; i < end; i++ {
		line := lines[i]
		fmt.Fprintf(&b, "  %s \n", line.text)
		rendered++
	}

	// Pad to fill height.
	if h.height > 0 {
		for rendered < h.height {
			b.WriteRune('\n')
			rendered++
		}
	}

	return b.String()
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
		for _, line := range lines {
			out = append(out, renderedHistoryLine{text: line, itemIndex: i})
		}
		if i < len(h.items)-1 {
			out = append(out, renderedHistoryLine{text: "", itemIndex: i})
		}
	}
	h.linesCache = out
	h.linesCacheValid = true
	return out
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

// applyStatus appends a styled status line to a copy of the cached lines.
// For thinking items with no content, the status replaces the empty placeholder.
func (h *HistoryScreen) applyStatus(cached []string, status string, kind TranscriptKind) []string {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(status)
	if kind == TranscriptThinking && (len(cached) == 0 || (len(cached) == 1 && strings.TrimSpace(stripANSIForLayout(cached[0])) == "")) {
		return []string{styled}
	}
	out := make([]string, len(cached)+2)
	copy(out, cached)
	out[len(cached)] = ""
	out[len(cached)+1] = styled
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
			body = h.renderAssistant(item.Text, width)
		} else {
			body = h.renderAssistantPlain(item.Text, width)
		}
		if item.Status != "" {
			status := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(item.Status)
			if body != "" {
				body += "\n\n" + status
			} else {
				body = status
			}
		}
		return splitRenderedBlock(body)
	case TranscriptThinking:
		return splitRenderedBlock(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(item.Text))
	case TranscriptTool:
		return splitRenderedBlock(h.renderTool(item.Text, width))
	case TranscriptError:
		return splitRenderedBlock(lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(wrapToWidth(item.Text, width)))
	case TranscriptSystem:
		return splitRenderedBlock(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(wrapToWidth(item.Text, width)))
	default:
		return splitRenderedBlock(wrapToWidth(item.Text, width))
	}
}

func (h *HistoryScreen) renderUser(text string, width int) string {
	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true).Render("❯")
	wrapped := wrapToWidth(text, max(width-2, 8))
	lines := strings.Split(wrapped, "\n")
	lines[0] = prompt + " " + lines[0]
	for i := 1; i < len(lines); i++ {
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}

func (h *HistoryScreen) renderAssistant(text string, width int) string {
	return h.renderAssistantMarkdown(text, width, true)
}

func (h *HistoryScreen) renderAssistantPlain(text string, width int) string {
	return h.renderAssistantMarkdown(text, width, false)
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
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourstyles.DarkStyle),
		glamour.WithWordWrap(markdownWidth),
	)
	if err != nil {
		return wrapToWidth(text, markdownWidth)
	}
	out, err := r.Render(text)
	if err != nil {
		return wrapToWidth(text, markdownWidth)
	}
	return strings.TrimRight(out, "\n")
}

func (h *HistoryScreen) renderTool(text string, width int) string {
	parts := strings.SplitN(text, "\n", 2)
	summary := parts[0]
	if len(parts) > 1 {
		detail := strings.TrimSpace(parts[1])
		if detail != "" {
			summary = summary + " " + detail
		}
	}
	maxLen := width - 2
	if maxLen > 0 && len(summary) > maxLen {
		summary = summary[:max(maxLen-3, 0)] + "..."
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("⎿ " + summary)
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

func splitRenderedBlock(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return []string{""}
	}
	lines := strings.Split(s, "\n")
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	if start >= end {
		return []string{""}
	}
	return lines[start:end]
}
