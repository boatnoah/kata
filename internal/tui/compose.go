package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Compose holds the state of the compose/input area.
type Compose struct {
	buf    []rune
	cursor int
	width  int
	height int
}

func NewCompose() *Compose {
	return &Compose{}
}

func (c *Compose) OnWindowSize(width, height int) {
	c.width = width
	c.height = height
}

// InsertRune inserts r at the current cursor position.
func (c *Compose) InsertRune(r rune) {
	if r == 0 {
		return
	}
	c.buf = append(c.buf[:c.cursor], append([]rune{r}, c.buf[c.cursor:]...)...)
	c.cursor++
}

func (c *Compose) InsertNewline() {
	c.InsertRune('\n')
}

func (c *Compose) Backspace() {
	if c.cursor == 0 || len(c.buf) == 0 {
		return
	}
	c.buf = append(c.buf[:c.cursor-1], c.buf[c.cursor:]...)
	c.cursor--
}

func (c *Compose) DeleteForward() {
	if c.cursor >= len(c.buf) {
		return
	}
	c.buf = append(c.buf[:c.cursor], c.buf[c.cursor+1:]...)
}

func (c *Compose) MoveLeft() {
	if c.cursor > 0 {
		c.cursor--
	}
}

func (c *Compose) MoveRight() {
	if c.cursor < len(c.buf) {
		c.cursor++
	}
}

func (c *Compose) MoveWordFwd() {
	n := len(c.buf)
	i := c.cursor
	for i < n && isSpace(c.buf[i]) {
		i++
	}
	for i < n && !isSpace(c.buf[i]) {
		i++
	}
	// Skip trailing spaces to land on start of next word.
	for i < n && isSpace(c.buf[i]) {
		i++
	}
	c.cursor = i
}

func (c *Compose) MoveWordBack() {
	i := c.cursor
	if i > 0 {
		i--
	}
	for i > 0 && isSpace(c.buf[i]) {
		i--
	}
	for i > 0 && !isSpace(c.buf[i-1]) {
		i--
	}
	c.cursor = i
}

func (c *Compose) MoveLineStart() {
	c.cursor = c.lineStart()
}

func (c *Compose) MoveLineStartNonSpace() {
	start := c.lineStart()
	i := start
	for i < len(c.buf) && c.buf[i] == ' ' {
		i++
	}
	if i >= len(c.buf) || c.buf[i] == '\n' {
		c.cursor = start
		return
	}
	c.cursor = i
}

func (c *Compose) MoveLineEnd() {
	start := c.lineStart()
	end := len(c.buf)
	for i := start; i < len(c.buf); i++ {
		if c.buf[i] == '\n' {
			end = i
			break
		}
	}
	// Place cursor at end (before newline).
	c.cursor = end
}

func (c *Compose) DeleteToEOL() {
	start := c.cursor
	end := len(c.buf)
	for i := c.cursor; i < len(c.buf); i++ {
		if c.buf[i] == '\n' {
			end = i
			break
		}
	}
	if start >= end {
		return
	}
	c.buf = append(c.buf[:start], c.buf[end:]...)
}

// View renders the compose buffer with a visible cursor.
// It returns the rendered string and its line count.
func (c *Compose) View(width int) (string, int) {
	content := c.renderWithCursor()
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Width(width).Render(content)
	lines := strings.Count(box, "\n") + 1
	return box, lines
}

func (c *Compose) renderWithCursor() string {
	var b strings.Builder
	for i, r := range c.buf {
		if i == c.cursor {
			b.WriteString(cursorGlyph())
		}
		b.WriteRune(r)
	}
	if c.cursor == len(c.buf) {
		b.WriteString(cursorGlyph())
	}
	return "Compose\n" + b.String()
}

func (c *Compose) lineStart() int {
	if len(c.buf) == 0 || c.cursor == 0 {
		return 0
	}
	i := c.cursor
	if i > 0 && i == len(c.buf) {
		i--
	}
	for i > 0 {
		if c.buf[i-1] == '\n' {
			break
		}
		i--
	}
	return i
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

func cursorGlyph() string {
	return "|"
}
