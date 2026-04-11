package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Compose holds the state of the compose/input area.
type Compose struct {
	buf          []rune
	cursor       int
	width        int
	height       int
	col          int // preferred column for vertical moves
	visualActive bool
	visualAnchor int
	yankBuf      []rune
}

func NewCompose() *Compose {
	return &Compose{col: -1}
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
	c.col = c.cursorColumn()
	c.clearVisual()
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
	c.col = c.cursorColumn()
	c.clearVisual()
}

func (c *Compose) DeleteForward() {
	if c.cursor >= len(c.buf) {
		return
	}
	c.buf = append(c.buf[:c.cursor], c.buf[c.cursor+1:]...)
	c.col = c.cursorColumn()
	c.clearVisual()
}

func (c *Compose) MoveLeft() {
	if c.cursor > 0 {
		c.cursor--
	}
	c.col = c.cursorColumn()
}

func (c *Compose) MoveRight() {
	if c.cursor < len(c.buf) {
		c.cursor++
	}
	c.col = c.cursorColumn()
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
	c.col = c.cursorColumn()
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
	c.col = c.cursorColumn()
}

func (c *Compose) MoveLineStart() {
	c.cursor = c.lineStart()
	c.col = c.cursorColumn()
}

func (c *Compose) MoveLineStartNonSpace() {
	start := c.lineStart()
	i := start
	for i < len(c.buf) && c.buf[i] == ' ' {
		i++
	}
	if i >= len(c.buf) || c.buf[i] == '\n' {
		c.cursor = start
		c.col = c.cursorColumn()
		return
	}
	c.cursor = i
	c.col = c.cursorColumn()
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
	c.col = c.cursorColumn()
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
	c.col = c.cursorColumn()
	c.clearVisual()
}

func (c *Compose) DeleteCurrentLine() {
	if len(c.buf) == 0 {
		return
	}
	start := c.lineStartAt(c.cursor)
	end := c.lineEndAt(c.cursor)
	if end < len(c.buf) && c.buf[end] == '\n' {
		end++
	} else if start > 0 {
		start-- // remove preceding newline to avoid blank line
	}
	if start < 0 {
		start = 0
	}
	if end > len(c.buf) {
		end = len(c.buf)
	}
	if start >= end {
		return
	}
	c.yankBuf = append([]rune(nil), c.buf[start:end]...)
	c.buf = append(c.buf[:start], c.buf[end:]...)
	if start > len(c.buf) {
		start = len(c.buf)
	}
	c.cursor = start
	c.col = c.cursorColumn()
	c.clearVisual()
}

func (c *Compose) MoveUp() {
	if c.cursor == 0 || len(c.buf) == 0 {
		return
	}
	lineIdx, col := c.lineAndColumn()
	if lineIdx == 0 {
		return
	}
	targetLine := lineIdx - 1
	targetCursor := c.cursorAtLineColumn(targetLine, c.preferredColumn(col))
	c.cursor = targetCursor
	c.col = c.cursorColumn()
}

func (c *Compose) MoveDown() {
	if len(c.buf) == 0 {
		return
	}
	lineIdx, col := c.lineAndColumn()
	lastLine := c.totalLines() - 1
	if lineIdx >= lastLine {
		return
	}
	targetLine := lineIdx + 1
	targetCursor := c.cursorAtLineColumn(targetLine, c.preferredColumn(col))
	c.cursor = targetCursor
	c.col = c.cursorColumn()
}

func (c *Compose) EnterVisual() {
	c.visualActive = true
	c.visualAnchor = c.cursor
}

func (c *Compose) clearVisual() {
	c.visualActive = false
	c.visualAnchor = 0
}

func (c *Compose) exitVisualIfActive() {
	if c.visualActive {
		c.clearVisual()
	}
}

func (c *Compose) selectionRange() (int, int, bool) {
	if !c.visualActive {
		return 0, 0, false
	}
	lo := c.visualAnchor
	hi := c.cursor
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi, true
}

func (c *Compose) DeleteSelection() {
	lo, hi, ok := c.selectionRange()
	if !ok || hi == lo {
		c.clearVisual()
		return
	}
	if lo < 0 {
		lo = 0
	}
	if hi > len(c.buf) {
		hi = len(c.buf)
	}
	c.yankBuf = append([]rune(nil), c.buf[lo:hi]...)
	c.buf = append(c.buf[:lo], c.buf[hi:]...)
	if lo > len(c.buf) {
		lo = len(c.buf)
	}
	c.cursor = lo
	c.col = c.cursorColumn()
	c.clearVisual()
}

func (c *Compose) YankSelection() {
	lo, hi, ok := c.selectionRange()
	if !ok || hi == lo {
		c.clearVisual()
		return
	}
	if lo < 0 {
		lo = 0
	}
	if hi > len(c.buf) {
		hi = len(c.buf)
	}
	c.yankBuf = append([]rune(nil), c.buf[lo:hi]...)
	c.clearVisual()
}

func (c *Compose) PasteAfter() {
	if len(c.yankBuf) == 0 {
		c.clearVisual()
		return
	}
	insertAt := c.cursor
	anchorPos := insertAt
	if c.visualActive {
		_, hi, _ := c.selectionRange()
		anchorPos = hi
		c.clearVisual()
	}
	end := c.lineEndAt(anchorPos)
	insertAt = end
	paste := append([]rune(nil), c.yankBuf...)
	paste = ensureTrailingNewline(paste)
	if end < len(c.buf) && c.buf[end] == '\n' {
		insertAt = end + 1
	} else {
		paste = append([]rune{'\n'}, paste...)
	}
	c.buf = append(c.buf[:insertAt], append(paste, c.buf[insertAt:]...)...)
	c.cursor = insertAt + len(paste)
	c.col = c.cursorColumn()
}

func (c *Compose) PasteBefore() {
	if len(c.yankBuf) == 0 {
		c.clearVisual()
		return
	}
	insertAt := c.cursor
	anchorPos := insertAt
	if c.visualActive {
		lo, _, _ := c.selectionRange()
		anchorPos = lo
		c.clearVisual()
	}
	insertAt = c.lineStartAt(anchorPos)
	paste := append([]rune(nil), c.yankBuf...)
	paste = ensureTrailingNewline(paste)
	if insertAt > 0 && c.buf[insertAt-1] != '\n' {
		paste = append([]rune{'\n'}, paste...)
	}
	c.buf = append(c.buf[:insertAt], append(paste, c.buf[insertAt:]...)...)
	c.cursor = insertAt + len(paste)
	c.col = c.cursorColumn()
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
	box := cursorBox()
	lo, hi, hasSel := c.selectionRange()
	for i, r := range c.buf {
		inSel := hasSel && i >= lo && i < hi
		if i == c.cursor {
			if r == '\n' {
				cursor := box.wrap(" ")
				if inSel {
					cursor = selectionBox().wrap(cursor)
				}
				b.WriteString(cursor)
				b.WriteRune('\n')
				continue
			}
			cursor := box.wrap(string(r))
			if inSel {
				cursor = selectionBox().wrap(cursor)
			}
			b.WriteString(cursor)
			continue
		}
		if inSel {
			b.WriteString(selectionBox().wrap(string(r)))
			continue
		}
		b.WriteRune(r)
	}
	if c.cursor == len(c.buf) {
		cursor := box.wrap(" ")
		if hasSel && c.cursor >= lo && c.cursor < hi {
			cursor = selectionBox().wrap(cursor)
		}
		b.WriteString(cursor)
	}
	return "Compose\n" + b.String()
}

func (c *Compose) cursorColumn() int {
	return c.cursor - c.lineStart()
}

func (c *Compose) lineAndColumn() (int, int) {
	line := 0
	col := 0
	for i := 0; i < c.cursor && i < len(c.buf); i++ {
		if c.buf[i] == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	return line, col
}

func (c *Compose) totalLines() int {
	if len(c.buf) == 0 {
		return 1
	}
	lines := 1
	for _, r := range c.buf {
		if r == '\n' {
			lines++
		}
	}
	return lines
}

func (c *Compose) cursorAtLineColumn(targetLine, targetCol int) int {
	line := 0
	idx := 0
	col := 0
	for idx < len(c.buf) {
		if line == targetLine {
			break
		}
		if c.buf[idx] == '\n' {
			line++
		}
		idx++
	}
	for idx < len(c.buf) && col < targetCol {
		if c.buf[idx] == '\n' {
			break
		}
		idx++
		col++
	}
	return idx
}

func (c *Compose) preferredColumn(current int) int {
	if c.col < 0 {
		return current
	}
	return c.col
}

func (c *Compose) lineStartAt(pos int) int {
	if len(c.buf) == 0 || pos == 0 {
		return 0
	}
	if pos > len(c.buf) {
		pos = len(c.buf)
	}
	i := pos
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

func (c *Compose) lineEndAt(pos int) int {
	if len(c.buf) == 0 {
		return 0
	}
	start := c.lineStartAt(pos)
	end := len(c.buf)
	for i := start; i < len(c.buf); i++ {
		if c.buf[i] == '\n' {
			end = i
			break
		}
	}
	return end
}

func ensureTrailingNewline(runes []rune) []rune {
	if len(runes) == 0 || runes[len(runes)-1] != '\n' {
		runes = append(runes, '\n')
	}
	return runes
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

// cursorBox renders a block cursor by reversing foreground/background.
type cursorBoxStyle struct{}

func cursorBox() cursorBoxStyle {
	return cursorBoxStyle{}
}

func (cursorBoxStyle) wrap(s string) string {
	styled := lipgloss.NewStyle().Reverse(true).Render(s)
	if styled == s {
		return "[" + s + "]"
	}
	return styled
}

type selectionBoxStyle struct{}

func selectionBox() selectionBoxStyle {
	return selectionBoxStyle{}
}

func (selectionBoxStyle) wrap(s string) string {
	styled := lipgloss.NewStyle().Reverse(true).Render(s)
	if styled == s {
		return "{" + s + "}"
	}
	return styled
}
