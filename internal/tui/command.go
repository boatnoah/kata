package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CommandLine holds transient state for ":" command entry.
type CommandLine struct {
	buf    []rune
	cursor int
}

func NewCommandLine() *CommandLine {
	return &CommandLine{}
}

func (c *CommandLine) Reset() {
	c.buf = c.buf[:0]
	c.cursor = 0
}

// HandleKey mutates the buffer and reports whether to execute or cancel.
func (c *CommandLine) HandleKey(msg tea.KeyMsg) (execute bool, cancel bool, input string) {
	switch msg.Type {
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			c.insertRune(r)
		}
	case tea.KeySpace:
		c.insertRune(' ')
	case tea.KeyBackspace:
		c.backspace()
	case tea.KeyDelete:
		c.deleteForward()
	case tea.KeyLeft:
		c.moveLeft()
	case tea.KeyRight:
		c.moveRight()
	case tea.KeyHome:
		c.cursor = 0
	case tea.KeyEnd:
		c.cursor = len(c.buf)
	case tea.KeyEsc, tea.KeyCtrlC:
		cancel = true
	case tea.KeyEnter:
		execute = true
		input = string(c.buf)
	}
	return execute, cancel, input
}

func (c *CommandLine) insertRune(r rune) {
	if r == 0 {
		return
	}
	c.buf = append(c.buf[:c.cursor], append([]rune{r}, c.buf[c.cursor:]...)...)
	c.cursor++
}

func (c *CommandLine) backspace() {
	if c.cursor == 0 || len(c.buf) == 0 {
		return
	}
	c.buf = append(c.buf[:c.cursor-1], c.buf[c.cursor:]...)
	c.cursor--
}

func (c *CommandLine) deleteForward() {
	if c.cursor >= len(c.buf) {
		return
	}
	c.buf = append(c.buf[:c.cursor], c.buf[c.cursor+1:]...)
}

func (c *CommandLine) moveLeft() {
	if c.cursor > 0 {
		c.cursor--
	}
}

func (c *CommandLine) moveRight() {
	if c.cursor < len(c.buf) {
		c.cursor++
	}
}

// View renders the command prompt and buffer with a visible cursor.
func (c *CommandLine) View() string {
	var b strings.Builder
	box := cursorBox()
	for i, r := range c.buf {
		if i == c.cursor {
			b.WriteString(box.wrap(string(r)))
			continue
		}
		b.WriteRune(r)
	}
	if c.cursor == len(c.buf) {
		b.WriteString(box.wrap(" "))
	}
	return ":" + b.String()
}
