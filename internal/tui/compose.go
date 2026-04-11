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
	active       bool
}

func NewCompose() *Compose {
	return &Compose{col: -1}
}

// Content returns the current compose buffer as a string.
func (c *Compose) Content() string {
	return string(c.buf)
}

// Reset clears the compose buffer and cursor state.
func (c *Compose) Reset() {
	c.buf = c.buf[:0]
	c.cursor = 0
	c.col = -1
	c.clearVisual()
}

func (c *Compose) SetActive(active bool) {
	c.active = active
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
	copyToClipboard(string(c.yankBuf))
	c.buf = append(c.buf[:start], c.buf[end:]...)
	if start > len(c.buf) {
		start = len(c.buf)
	}
	c.cursor = start
	c.col = c.cursorColumn()
	c.clearVisual()
}

func (c *Compose) pasteContent() []rune {
	if s := pasteFromClipboard(); s != "" {
		return []rune(s)
	}
	if len(c.yankBuf) > 0 {
		return append([]rune(nil), c.yankBuf...)
	}
	return nil
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
	copyToClipboard(string(c.yankBuf))
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
	copyToClipboard(string(c.yankBuf))
	c.clearVisual()
}

func (c *Compose) PasteAfter() {
	content := c.pasteContent()
	if len(content) == 0 {
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
	paste := append([]rune(nil), content...)
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
	content := c.pasteContent()
	if len(content) == 0 {
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
	paste := append([]rune(nil), content...)
	paste = ensureTrailingNewline(paste)
	if insertAt > 0 && c.buf[insertAt-1] != '\n' {
		paste = append([]rune{'\n'}, paste...)
	}
	c.buf = append(c.buf[:insertAt], append(paste, c.buf[insertAt:]...)...)
	c.cursor = insertAt + len(paste)
	c.col = c.cursorColumn()
}

// Append enters insert after the current cursor position (vim 'a').
func (c *Compose) Append() {
	if c.cursor < len(c.buf) {
		c.cursor++
	}
	c.col = c.cursorColumn()
}

// OpenBelow inserts a new blank line below the current line and moves into it.
func (c *Compose) OpenBelow() {
	end := c.lineEndAt(c.cursor)
	insertAt := end
	if insertAt < len(c.buf) && c.buf[insertAt] == '\n' {
		insertAt++
	}
	newline := []rune{'\n'}
	c.buf = append(c.buf[:insertAt], append(newline, c.buf[insertAt:]...)...)
	c.cursor = insertAt
	c.col = 0
	c.clearVisual()
}

// OpenAbove inserts a new blank line above the current line and moves into it.
func (c *Compose) OpenAbove() {
	insertAt := c.lineStartAt(c.cursor)
	newline := []rune{'\n'}
	c.buf = append(c.buf[:insertAt], append(newline, c.buf[insertAt:]...)...)
	c.cursor = insertAt
	c.col = 0
	c.clearVisual()
}

// View renders the compose buffer with a visible cursor.
// targetLines limits the total rendered height (including the border). Pass 0 for no limit.
// It returns the rendered string and its line count.
func (c *Compose) View(width int, targetLines int) (string, int) {
	content := c.renderWithCursor()

	// Cap the number of rendered lines (including the header) before adding the border.
	// Account for the border's top and bottom lines so the final height stays within targetLines.
	if targetLines > 0 {
		available := targetLines - 2 // border adds two lines
		if available < 1 {
			available = 1
		}
		lines := strings.Split(content, "\n")
		if len(lines) > available {
			// Keep the header (line 0) and scroll the body to keep the cursor visible.
			cursorLine := c.cursorRenderLine()
			body := lines[1:]
			bodyCursorLine := cursorLine - 1
			if bodyCursorLine < 0 {
				bodyCursorLine = 0
			}

			window := available - 1 // leave room for header
			if window < 1 {
				window = 1
			}

			start := bodyCursorLine - (window - 1)
			if start < 0 {
				start = 0
			}
			end := start + window
			if end > len(body) {
				end = len(body)
				start = end - window
				if start < 0 {
					start = 0
				}
			}
			body = body[start:end]
			lines = append([]string{lines[0]}, body...)
		}
		content = strings.Join(lines, "\n")
	}

	borderColor := lipgloss.Color("#666666")
	if c.active {
		borderColor = lipgloss.Color("#4ea4ff")
	}

	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(borderColor)
	if width > 0 {
		inner := width - 2 // account for left/right border runes
		if inner < 1 {
			inner = 1
		}
		style = style.Width(inner)
	}
	if targetLines > 0 {
		contentHeight := targetLines - 2 // exclude border lines
		if contentHeight < 1 {
			contentHeight = 1
		}
		style = style.Height(contentHeight)
	}

	box := style.Render(content)
	lines := strings.Count(box, "\n") + 1
	return box, lines
}

// cursorRenderLine returns the zero-based line index (within the unbordered render)
// where the cursor is placed, accounting for the header line.
func (c *Compose) cursorRenderLine() int {
	lineIdx, _ := c.lineAndColumn()
	// Add one for the header line in renderWithCursor.
	return lineIdx + 1
}

func (c *Compose) renderWithCursor() string {
	var b strings.Builder
	box := cursorBox()
	lo, hi, hasSel := c.selectionRange()
	for i, r := range c.buf {
		inSel := hasSel && i >= lo && i < hi
		if i == c.cursor && c.active {
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
		cursor := ""
		if c.active {
			cursor = box.wrap(" ")
		}
		if hasSel && c.cursor >= lo && c.cursor < hi {
			cursor = selectionBox().wrap(cursor)
		}
		if cursor != "" {
			b.WriteString(cursor)
		}
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
