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
		messages: []string{
			"User: How do I set up project X?",
			"AI: You can start by running `kata init` in the repo.",
			"User: Thanks! How do I add tests?",
			"AI: Run `go test ./...` after adding files under internal/...",
		},
	}
}

func (h *HistoryScreen) OnWindowSize(width, height int) {
	h.width = width
	h.height = height
}

func (h *HistoryScreen) SetActive(active bool) {
	h.active = active
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
	fmt.Fprintf(&b, "History (items: %v)\n", len(h.messages))
	for i, line := range h.messages {
		marker := "  "
		if i == h.cursor {
			marker = "> "
		}

		content := line
		runes := []rune(line)
		if i == h.cursor && h.active {
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

		fmt.Fprintf(&b, "%s%s\n", marker, content)
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
