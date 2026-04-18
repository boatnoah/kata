package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
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
	theme        Theme
}

func NewCompose() *Compose {
	return &Compose{col: -1, theme: DefaultTheme()}
}

// SetTheme swaps the palette used to paint the prompt chrome.
func (c *Compose) SetTheme(t Theme) {
	c.theme = t
}

// currentMode derives the effective mode for painting the prompt frame.
// Visual and Insert dominate when active; otherwise Normal.
func (c *Compose) currentMode(fallback Mode) Mode {
	if c.visualActive {
		return ModeVisual
	}
	return fallback
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

// BodyLineCount returns the number of wrapped display lines the compose body
// currently occupies at the given pane width. Used by the app layout to size
// the compose pane dynamically (border + body). Returns at least 1.
func (c *Compose) BodyLineCount(width int) int {
	innerWidth := c.innerWidth(width)
	lines := composeWrappedLines(c.renderWithCursor(), innerWidth)
	if len(lines) == 0 {
		return 1
	}
	return len(lines)
}

// innerWidth computes the text column budget inside the bordered prompt box.
// Budget = width − (2 border cols + 1 pad col on each side + 2 cols for
// "❯ " prefix) = width − 6. Falls back to 1 when clamped very narrow.
func (c *Compose) innerWidth(width int) int {
	if width <= 0 {
		return 0
	}
	iw := width - promptFrameOverhead
	if iw < 1 {
		iw = 1
	}
	return iw
}

// promptFrameOverhead is the number of display cells consumed by the
// bordered prompt row that are NOT available for the user's text:
//   • 2 columns for the left + right border (│ … │)
//   • 2 columns of inner padding (one each side)
//   • 2 columns for the "❯ " prefix on the first line / "  " on wraps
const promptFrameOverhead = 6

// View renders the compose buffer inside a bordered prompt row with a
// mode-colored border and a `❯` marker. When command mode is active the
// caller uses a different view; this function ignores the modeLabel arg
// (kept for signature stability) and paints the border based on the app's
// mode.
//
// targetLines bounds the total rendered height (including borders). Pass 0
// for no limit. Returns the rendered string and its line count.
func (c *Compose) View(width int, targetLines int, modeLabel string) (string, int) {
	content := c.renderWithCursor()
	innerWidth := c.innerWidth(width)
	bodyLines := composeWrappedLines(content, innerWidth)

	// Cap visible body lines to keep cursor visible. Borders take 2 rows.
	if targetLines > 0 {
		available := targetLines - 2
		if available < 1 {
			available = 1
		}
		if len(bodyLines) > available {
			cursorLine := c.cursorRenderLineWrapped(innerWidth)
			start := cursorLine - (available - 1)
			if start < 0 {
				start = 0
			}
			end := start + available
			if end > len(bodyLines) {
				end = len(bodyLines)
				start = end - available
				if start < 0 {
					start = 0
				}
			}
			bodyLines = bodyLines[start:end]
		}
	}

	mode := c.resolveMode(modeLabel)
	borderColor := c.theme.Border
	if c.active {
		borderColor = c.theme.ModeColor(mode)
	}
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	markerColor := c.theme.Accent
	if c.active {
		markerColor = c.theme.ModeColor(mode)
	}
	markerStyle := lipgloss.NewStyle().Foreground(markerColor).Bold(true)

	// Build a rounded box: ╭── … ──╮ / │ ❯ … │ / ╰── … ──╯
	innerCols := 0
	if width > 2 {
		innerCols = width - 2 // cols between the two vertical borders
	}
	if innerCols < 1 {
		innerCols = 1
	}
	top := "╭" + strings.Repeat("─", innerCols) + "╮"
	bot := "╰" + strings.Repeat("─", innerCols) + "╯"

	var b strings.Builder
	b.WriteString(borderStyle.Render(top))
	b.WriteRune('\n')

	if len(bodyLines) == 0 {
		bodyLines = []string{""}
	}
	for i, line := range bodyLines {
		b.WriteString(borderStyle.Render("│"))
		b.WriteRune(' ')
		if i == 0 {
			b.WriteString(markerStyle.Render("❯"))
			b.WriteRune(' ')
		} else {
			b.WriteString("  ")
		}
		// Pad the interior to innerCols − 1 (minus leading space) so the
		// right border lines up on every row, even for empty/short lines.
		rendered := line
		renderedWidth := lipgloss.Width(rendered)
		want := innerCols - 1 - 2 // minus leading space + "❯ " / "  " prefix
		if want < 0 {
			want = 0
		}
		if renderedWidth < want {
			rendered += strings.Repeat(" ", want-renderedWidth)
		}
		b.WriteString(rendered)
		b.WriteRune(' ')
		b.WriteString(borderStyle.Render("│"))
		if i < len(bodyLines)-1 {
			b.WriteRune('\n')
		}
	}
	b.WriteRune('\n')
	b.WriteString(borderStyle.Render(bot))

	result := b.String()
	lineCount := strings.Count(result, "\n") + 1
	return result, lineCount
}

// resolveMode maps an optional text modeLabel back to a Mode so the border
// can color-match. It accepts the legacy labels the app passes in ("INSERT",
// "VISUAL", "COMMAND") plus empty = NORMAL.
func (c *Compose) resolveMode(label string) Mode {
	switch label {
	case "INSERT":
		return ModeInsert
	case "VISUAL":
		return ModeVisual
	case "COMMAND":
		return ModeCommandLine
	default:
		return ModeNormal
	}
}

func (c *Compose) cursorRenderLineWrapped(width int) int {
	if width <= 0 {
		return c.cursorRenderLine()
	}
	lineIdx, col := c.lineAndColumn()
	wrappedLine := 0
	for i := 0; i < lineIdx; i++ {
		start, end := c.boundsForLine(i)
		wrappedLine += max(len(strings.Split(wrap.String(string(c.buf[start:end]), width), "\n")), 1)
	}
	start, _ := c.boundsForLine(lineIdx)
	prefix := string(c.buf[start:min(start+col, len(c.buf))])
	wrappedPrefix := strings.Split(wrap.String(prefix, width), "\n")
	wrappedLine += max(len(wrappedPrefix), 1) - 1
	return wrappedLine
}

func composeWrappedLines(content string, width int) []string {
	if width <= 0 {
		return strings.Split(content, "\n")
	}
	rawLines := strings.Split(content, "\n")
	wrapped := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		segments := strings.Split(wrap.String(line, width), "\n")
		if len(segments) == 0 {
			wrapped = append(wrapped, "")
			continue
		}
		wrapped = append(wrapped, segments...)
	}
	return wrapped
}

func (c *Compose) boundsForLine(targetLine int) (int, int) {
	line := 0
	start := 0
	for i, r := range c.buf {
		if line == targetLine && r == '\n' {
			return start, i
		}
		if r == '\n' {
			line++
			start = i + 1
		}
	}
	if line == targetLine {
		return start, len(c.buf)
	}
	return len(c.buf), len(c.buf)
}

// cursorRenderLine returns the zero-based line index (within the unbordered render)
// where the cursor is placed, accounting for the header line.
func (c *Compose) cursorRenderLine() int {
	lineIdx, _ := c.lineAndColumn()
	return lineIdx
}

func (c *Compose) renderWithCursor() string {
	var b strings.Builder
	box := cursorBox()
	sel := selectionBox()
	lo, hi, hasSel := c.selectionRange()

	// Flush a batch of selected characters as a single styled run.
	var selBatch strings.Builder
	flushSel := func() {
		if selBatch.Len() > 0 {
			b.WriteString(sel.wrap(selBatch.String()))
			selBatch.Reset()
		}
	}

	for i, r := range c.buf {
		inSel := hasSel && i >= lo && i < hi
		isCursor := i == c.cursor && c.active

		if isCursor {
			flushSel()
			if r == '\n' {
				b.WriteString(box.wrap(" "))
				b.WriteRune('\n')
			} else {
				b.WriteString(box.wrap(string(r)))
			}
			continue
		}
		if inSel {
			if r == '\n' {
				flushSel()
				b.WriteRune('\n')
			} else {
				selBatch.WriteRune(r)
			}
			continue
		}
		flushSel()
		b.WriteRune(r)
	}
	flushSel()

	if c.cursor == len(c.buf) && c.active {
		b.WriteString(box.wrap(" "))
	}
	return b.String()
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
