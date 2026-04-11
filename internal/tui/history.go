package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HistoryScreen struct {
	messages []string
	cursor   int
	width    int
	height   int
}

func NewHistoryScreen() *HistoryScreen {
	return &HistoryScreen{
		messages: []string{
			"Kata — vim-centric AI chat TUI",
			"History pane renders conversation threads.",
			"Compose pane will live alongside this soon.",
		},
	}
}

func (h *HistoryScreen) OnWindowSize(width, height int) {
	h.width = width
	h.height = height
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
}

func (h *HistoryScreen) JumpStart() {
	if len(h.messages) == 0 {
		return
	}
	h.cursor = 0
}

func (h *HistoryScreen) JumpEnd() {
	if len(h.messages) == 0 {
		return
	}
	h.cursor = len(h.messages) - 1
}

func (h *HistoryScreen) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "History (items: %v)\n", len(h.messages))
	for i, line := range h.messages {
		marker := "  "
		if i == h.cursor {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%s\n", marker, line)
	}

	if h.width > 0 {
		fmt.Fprintf(&b, "\n%vx%v\n", h.width, h.height)
	}

	content := b.String()
	if h.width == 0 || h.height == 0 {
		return content
	}

	return lipgloss.Place(h.width, h.height, lipgloss.Center, lipgloss.Center, content)
}
