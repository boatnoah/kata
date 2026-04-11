package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newComposeWithBuffer(buf string, cursor int) *Compose {
	c := NewCompose()
	c.buf = []rune(buf)
	c.cursor = cursor
	c.col = c.cursorColumn()
	return c
}

func TestComposeMoveDownAndUpPreferredColumn(t *testing.T) {
	c := newComposeWithBuffer("foo\nbarbaz\nqux", 6) // column 2 of second line
	c.MoveDown()
	if c.cursor != 13 {
		t.Fatalf("move down expected cursor 13, got %v", c.cursor)
	}
	c.MoveUp()
	if c.cursor != 6 {
		t.Fatalf("move up expected cursor 6, got %v", c.cursor)
	}
}

func TestComposeMoveDownClampsShortLine(t *testing.T) {
	c := newComposeWithBuffer("foo\nlongerline\nhi", 12) // deep column on second line
	c.MoveDown()
	if c.cursor != len([]rune("foo\nlongerline\nhi")) {
		t.Fatalf("expected cursor at end after clamp, got %v", c.cursor)
	}
}

func TestComposeMoveUpAtTopNoop(t *testing.T) {
	c := newComposeWithBuffer("foo", 0)
	c.MoveUp()
	if c.cursor != 0 {
		t.Fatalf("expected cursor stay at 0, got %v", c.cursor)
	}
}

func TestComposeMoveDownEmptyBufferNoop(t *testing.T) {
	c := NewCompose()
	c.MoveDown()
	if c.cursor != 0 {
		t.Fatalf("expected cursor stay at 0, got %v", c.cursor)
	}
}

func TestBindingsUseComposeVerticalMoves(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeNormal
	app.compose.buf = []rune("a\nb")
	app.compose.cursor = 0
	app.compose.MoveRight() // set preferred column to 1

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.compose.cursor != 3 {
		t.Fatalf("expected move down to cursor 3, got %v", app.compose.cursor)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if app.compose.cursor != 1 {
		t.Fatalf("expected move up to cursor 1, got %v", app.compose.cursor)
	}
}

func TestCursorVisibleOnEmptyLine(t *testing.T) {
	c := newComposeWithBuffer("one\n\nthree", 4) // cursor on empty line (at newline rune)
	c.active = true
	view := c.renderWithCursor()
	if !strings.Contains(view, "\x1b[7m ") && !strings.Contains(view, "[ ]") {
		t.Fatalf("expected block cursor placeholder on empty line, got %q", view)
	}
}

func TestVisualSelectionDeleteYank(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeVisual
	app.compose.buf = []rune("hello\nworld")
	app.compose.visualActive = true
	app.compose.visualAnchor = 0
	app.compose.cursor = 5 // select "hello"

	app.compose.YankSelection()
	if string(app.compose.yankBuf) != "hello" {
		t.Fatalf("yank expected 'hello', got %q", string(app.compose.yankBuf))
	}
	app.mode = ModeVisual
	app.compose.visualActive = true
	app.compose.visualAnchor = 0
	app.compose.cursor = 5
	app.compose.DeleteSelection()
	if string(app.compose.buf) != "\nworld" {
		t.Fatalf("delete expected remaining newline+world, got %q", string(app.compose.buf))
	}
}

func TestPasteAfterBelowCurrentLine(t *testing.T) {
	c := newComposeWithBuffer("foo\nbar", 1) // cursor on first line
	copyToClipboard("baz")
	c.PasteAfter()
	if string(c.buf) != "foo\nbaz\nbar" {
		t.Fatalf("paste after expected below current line, got %q", string(c.buf))
	}
}

func TestPasteBeforeAboveCurrentLine(t *testing.T) {
	c := newComposeWithBuffer("foo\nbar", 5) // cursor on second line
	copyToClipboard("baz")
	c.PasteBefore()
	if string(c.buf) != "foo\nbaz\nbar" {
		t.Fatalf("paste before expected above current line, got %q", string(c.buf))
	}
}

func TestDeleteLineWithDAndDD(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeNormal
	app.compose.buf = []rune("line1\nline2\nline3")
	app.compose.cursor = 7 // in line2

	// Capital D deletes current line
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if string(app.compose.buf) != "line1\nline3" {
		t.Fatalf("D should delete current line, got %q", string(app.compose.buf))
	}

	// Reset buffer and test dd
	app.compose.buf = []rune("line1\nline2\nline3")
	app.compose.cursor = 7
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if string(app.compose.buf) != "line1\nline3" {
		t.Fatalf("dd should delete current line, got %q", string(app.compose.buf))
	}
}

func TestVisualEscapeAndMovement(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeVisual
	app.compose.buf = []rune("abc\nb")
	app.compose.visualActive = true
	app.compose.visualAnchor = 0
	app.compose.cursor = 0

	// Move down in visual mode
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if app.compose.cursor != 4 {
		t.Fatalf("expected cursor move down to 4, got %v", app.compose.cursor)
	}
	// Move up in visual mode
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if app.compose.cursor != 0 {
		t.Fatalf("expected cursor move up to 0, got %v", app.compose.cursor)
	}
	// Move to line end with $
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	if app.compose.cursor != 3 {
		t.Fatalf("expected cursor at line end (3), got %v", app.compose.cursor)
	}
	// Move to line start with 0
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	if app.compose.cursor != 0 {
		t.Fatalf("expected cursor at line start (0) after 0, got %v", app.compose.cursor)
	}
	// Escape exits visual
	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if app.mode != ModeNormal {
		t.Fatalf("expected normal mode after esc, got %v", app.mode)
	}
	if app.compose.visualActive {
		t.Fatalf("visual state should be cleared after esc")
	}
}

func TestInsertLineStartWithCapitalI(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeNormal
	app.compose.buf = []rune("  abc")
	app.compose.cursor = len(app.compose.buf)

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'I'}})
	if app.mode != ModeInsert {
		t.Fatalf("expected insert mode after I, got %v", app.mode)
	}
	if app.compose.cursor != 2 {
		t.Fatalf("expected cursor at first non-space (2), got %v", app.compose.cursor)
	}
}

func TestAppendWithA(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeNormal
	app.compose.buf = []rune("abc")
	app.compose.cursor = 1 // on 'b'

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if app.mode != ModeInsert {
		t.Fatalf("expected insert mode after a, got %v", app.mode)
	}
	if app.compose.cursor != 2 {
		t.Fatalf("expected cursor move one right (2), got %v", app.compose.cursor)
	}
}

func TestAppendLineEndWithCapitalA(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeNormal
	app.compose.buf = []rune("abc")
	app.compose.cursor = 1

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if app.mode != ModeInsert {
		t.Fatalf("expected insert mode after A, got %v", app.mode)
	}
	if app.compose.cursor != len([]rune("abc")) {
		t.Fatalf("expected cursor at line end, got %v", app.compose.cursor)
	}
}

func TestOpenBelowWithLowercaseO(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeNormal
	app.compose.buf = []rune("foo\nbar")
	app.compose.cursor = 1 // on first line

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if app.mode != ModeInsert {
		t.Fatalf("expected insert mode after o, got %v", app.mode)
	}
	if got := string(app.compose.buf); got != "foo\n\nbar" {
		t.Fatalf("expected blank line inserted below, got %q", got)
	}
	if app.compose.cursor != 4 {
		t.Fatalf("expected cursor at start of new line (4), got %v", app.compose.cursor)
	}
}

func TestOpenAboveWithCapitalO(t *testing.T) {
	app := NewApp()
	app.activePane = PaneCompose
	app.mode = ModeNormal
	app.compose.buf = []rune("foo\nbar")
	app.compose.cursor = 5 // on second line

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'O'}})

	if app.mode != ModeInsert {
		t.Fatalf("expected insert mode after O, got %v", app.mode)
	}
	if got := string(app.compose.buf); got != "foo\n\nbar" {
		t.Fatalf("expected blank line inserted above current line, got %q", got)
	}
	if app.compose.cursor != 4 {
		t.Fatalf("expected cursor at start of new line (4), got %v", app.compose.cursor)
	}
}
