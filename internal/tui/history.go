package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HistoryScreen struct {
	messages     []string
	cursor       int
	cursorCol    int
	visualActive bool
	visualAnchor int
	yankBuf      []string
	active       bool
	width        int
	height       int
}

func NewHistoryScreen() *HistoryScreen {
	return &HistoryScreen{
		messages: []string{},
	}
}

func (h *HistoryScreen) OnWindowSize(width, height int) {
	h.width = width
	h.height = height
}

func (h *HistoryScreen) SetActive(active bool) {
	h.active = active
}

// AppendMessage adds a new entry to history and focuses the end.
// It returns the index of the appended message.
func (h *HistoryScreen) AppendMessage(msg string) int {
	return h.AppendMessageWithFocus(msg, true)
}

func (h *HistoryScreen) AppendMessageWithFocus(msg string, focus bool) int {
	msg = sanitizeHistoryMessage(msg)
	if msg == "" {
		return len(h.messages) - 1
	}
	h.messages = append(h.messages, msg)
	if focus || len(h.messages) == 1 {
		h.cursor = len(h.messages) - 1
		h.cursorCol = len([]rune(msg))
	}
	h.ExitVisual()
	h.clampCursorCol()
	return h.cursor
}

// UpdateMessageAt replaces the message at index and moves the cursor to it.
func (h *HistoryScreen) UpdateMessageAt(index int, msg string) {
	h.UpdateMessageAtWithFocus(index, msg, true)
}

func (h *HistoryScreen) UpdateMessageAtWithFocus(index int, msg string, focus bool) {
	if index < 0 || index >= len(h.messages) {
		return
	}
	msg = sanitizeHistoryMessage(msg)
	h.messages[index] = msg
	if focus {
		h.cursor = index
		h.cursorCol = len([]rune(msg))
	}
	h.ExitVisual()
	h.clampCursorCol()
}

// Move shifts the cursor by delta, clamped to available messages.
func (h *HistoryScreen) Move(delta int) {
	if len(h.messages) == 0 {
		return
	}
	next := max(h.cursor+delta, 0)

	if next >= len(h.messages) {
		next = len(h.messages) - 1
	}
	h.cursor = next
	h.clampCursorCol()
}

func (h *HistoryScreen) JumpStart() {
	if len(h.messages) == 0 {
		return
	}
	h.cursor = 0
	h.clampCursorCol()
}

func (h *HistoryScreen) JumpEnd() {
	if len(h.messages) == 0 {
		return
	}
	h.cursor = len(h.messages) - 1
	h.clampCursorCol()
}

func (h *HistoryScreen) MoveHalfPage(delta int) {
	if len(h.messages) == 0 || h.height == 0 {
		return
	}
	step := h.height / 2
	if step < 1 {
		step = 1
	}
	h.Move(delta * step)
}

func (h *HistoryScreen) MoveLeft() {
	if len(h.messages) == 0 {
		return
	}
	if h.cursorCol > 0 {
		h.cursorCol--
	}
}

func (h *HistoryScreen) MoveRight() {
	if len(h.messages) == 0 {
		return
	}
	line := h.messages[h.cursor]
	if h.cursorCol < len([]rune(line)) {
		h.cursorCol++
	}
}

func (h *HistoryScreen) LineStart() {
	h.cursorCol = 0
}

func (h *HistoryScreen) LineEnd() {
	if len(h.messages) == 0 {
		return
	}
	h.cursorCol = len([]rune(h.messages[h.cursor]))
}

func (h *HistoryScreen) EnterVisual() {
	h.visualActive = true
	h.visualAnchor = h.cursor
}

func (h *HistoryScreen) ExitVisual() {
	h.visualActive = false
	h.visualAnchor = 0
}

func (h *HistoryScreen) selectionRange() (int, int, bool) {
	if !h.visualActive {
		return 0, 0, false
	}
	lo := h.visualAnchor
	hi := h.cursor
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi, true
}

func (h *HistoryScreen) YankSelection() {
	lo, hi, ok := h.selectionRange()
	if !ok {
		h.ExitVisual()
		return
	}
	if hi < lo {
		lo, hi = hi, lo
	}
	hi = min(hi, len(h.messages)-1)
	lo = max(lo, 0)
	if hi < lo {
		h.ExitVisual()
		return
	}
	buf := make([]string, hi-lo+1)
	copy(buf, h.messages[lo:hi+1])
	h.yankBuf = buf
	copyToClipboard(strings.Join(buf, "\n"))
	h.ExitVisual()
}

func (h *HistoryScreen) clampCursorCol() {
	if len(h.messages) == 0 {
		h.cursorCol = 0
		return
	}
	line := []rune(h.messages[h.cursor])
	if h.cursorCol > len(line) {
		h.cursorCol = len(line)
	}
	if h.cursorCol < 0 {
		h.cursorCol = 0
	}
}

func (h *HistoryScreen) View() string {
	var b strings.Builder
	for i, line := range h.messages {
		marker := "  "
		if i == h.cursor {
			marker = "> "
		}

		entry := h.formatEntry(line)
		content := entry.content
		runes := []rune(entry.raw)
		if i == h.cursor && h.active && !entry.styled {
			var sb strings.Builder
			for idx, r := range runes {
				if idx == h.cursorCol {
					sb.WriteString(cursorBox().wrap(string(r)))
					continue
				}
				sb.WriteRune(r)
			}
			if h.cursorCol >= len(runes) {
				sb.WriteString(cursorBox().wrap(" "))
			}
			content = sb.String()
		}

		if lo, hi, ok := h.selectionRange(); ok && i >= lo && i <= hi {
			content = selectionBox().wrap(content)
		}

		renderMultilineEntry(&b, marker, content)
	}

	content := lipgloss.NewStyle().Bold(true).Padding(0, 1).Render(b.String())
	style := lipgloss.NewStyle()
	if h.width > 0 {
		style = style.Width(h.width)
	}
	if h.height > 0 {
		style = style.Height(h.height)
	}
	style = style.Align(lipgloss.Left).AlignVertical(lipgloss.Top)
	return style.Render(content)
}

type historyEntry struct {
	raw     string
	content string
	styled  bool
}

func (h *HistoryScreen) formatEntry(line string) historyEntry {
	const userPrefix = "User: "
	const aiPrefix = "AI: "
	const waitingPrefix = "AI_WAIT: "

	switch {
	case strings.HasPrefix(line, userPrefix):
		body := strings.TrimPrefix(line, userPrefix)
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
		if h.width > 8 {
			style = style.MaxWidth(h.width - 8)
		}
		return historyEntry{raw: body, content: style.Render(body), styled: true}
	case strings.HasPrefix(line, waitingPrefix):
		body := strings.TrimPrefix(line, waitingPrefix)
		content := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(body)
		return historyEntry{raw: body, content: content, styled: true}
	case strings.HasPrefix(line, aiPrefix):
		body := strings.TrimPrefix(line, aiPrefix)
		return historyEntry{raw: body, content: body}
	case strings.HasPrefix(line, "Tool: "), strings.HasPrefix(line, "Cmd: "), strings.HasPrefix(line, "Error: "), strings.HasPrefix(line, "System: "):
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			label := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(parts[0])
			return historyEntry{raw: parts[1], content: label + ": " + parts[1], styled: true}
		}
	}
	return historyEntry{raw: line, content: line}
}

func renderMultilineEntry(b *strings.Builder, marker, content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		prefix := "  "
		if i == 0 {
			prefix = marker
		}
		fmt.Fprintf(b, "%s%s\n", prefix, line)
	}
}
