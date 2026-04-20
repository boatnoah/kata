package tui

import (
	"regexp"
	"strings"
	"testing"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func TestStripANSIRemovesOSCHyperlinks(t *testing.T) {
	// Emulate what styleInline emits for [label](https://example.com): an
	// OSC 8 wrapper around an SGR-styled label. The stripper must yield just
	// "label" so yank/selection don't leak URL fragments into the clipboard.
	in := "\x1b]8;;https://example.com\x1b\\\x1b[4;38;5;45mclick\x1b[0m\x1b]8;;\x1b\\"
	got := stripANSIForLayout(in)
	if got != "click" {
		t.Fatalf("expected bare label, got %q", got)
	}
}

func TestHistoryAppendItem(t *testing.T) {
	h := NewHistoryScreen()
	idx := h.AppendItem(TranscriptItem{Kind: TranscriptUser, Text: "hello"}, true)

	if idx != 0 || len(h.items) != 1 {
		t.Fatalf("expected one item appended, got idx=%d len=%d", idx, len(h.items))
	}
}

func TestHistoryUpdateItemAt(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 10
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "first"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "second"}, true)

	h.UpdateItemAt(1, TranscriptItem{Kind: TranscriptAssistant, Text: "second updated"}, false)

	if h.items[1].Text != "second updated" {
		t.Fatalf("expected item text updated, got %q", h.items[1].Text)
	}
}

func TestHistoryViewAutoScrollsToBottom(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 3
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "one"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "two"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "three"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "four"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "five"}, true)

	view := h.View()

	if strings.Contains(view, "one") || strings.Contains(view, "two") {
		t.Fatalf("expected top entries to be scrolled out, got %q", view)
	}
	if !strings.Contains(view, "five") {
		t.Fatalf("expected bottom entry to remain visible, got %q", view)
	}
}

func TestHistoryViewScrollsTallMultilineEntry(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 3
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "line1\nline2\nline3\nline4"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "tail"}, true)

	view := h.View()

	if strings.Contains(view, "line1") || strings.Contains(view, "line2") {
		t.Fatalf("expected earlier lines to scroll out of view, got %q", view)
	}
	if !strings.Contains(view, "tail") {
		t.Fatalf("expected tail to remain visible, got %q", view)
	}
}

func TestHistoryViewScrollsWrappedLongLine(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 24
	h.height = 2
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "this is a very long line that should wrap across multiple screen rows"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "tail"}, true)

	view := h.View()

	if !strings.Contains(view, "tail") {
		t.Fatalf("expected tail visible, got %q", view)
	}
	if strings.Contains(view, "this is a very long line") {
		t.Fatalf("expected earliest wrapped portion to scroll out of view, got %q", view)
	}
}

func TestHistoryRendersUserAndCodeBlockMarkdown(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 60
	h.AppendItem(TranscriptItem{Kind: TranscriptUser, Text: "show me this"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "# Title\n\n- item\n\n```go\nfmt.Println(\"hi\")\n```"}, true)

	view := stripANSI(h.View())
	if !strings.Contains(view, "show me this") {
		t.Fatalf("expected user text rendered, got %q", view)
	}
	if !strings.Contains(view, "Title") || !strings.Contains(view, "item") {
		t.Fatalf("expected markdown structure rendered, got %q", view)
	}
	if !strings.Contains(view, "fmt.Println") {
		t.Fatalf("expected code block rendered, got %q", view)
	}
}

func TestHistoryUserMessageHasPromptPrefix(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 60
	h.height = 10
	h.AppendItem(TranscriptItem{Kind: TranscriptUser, Text: "hello world"}, true)

	view := stripANSI(h.View())
	if !strings.Contains(view, "❯ hello world") {
		t.Fatalf("expected user message with prompt prefix, got %q", view)
	}
}

func TestHistoryToolCallRendersCompact(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 60
	h.height = 10
	h.AppendItem(TranscriptItem{Kind: TranscriptTool, Text: "Read go.mod"}, true)

	view := stripANSI(h.View())
	if !strings.Contains(view, "⏺ Read go.mod") {
		t.Fatalf("expected compact tool call with prefix, got %q", view)
	}
}

// With auto-follow on (default), appending new content keeps the newest
// content visible at the bottom of the pane.
func TestHistoryAutoFollowKeepsLatestVisible(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 10
	for i := 0; i < 6; i++ {
		h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "old content line"}, true)
	}
	h.AppendItem(TranscriptItem{Kind: TranscriptUser, Text: "current question"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "short reply"}, true)

	view := stripANSI(h.View())
	if !strings.Contains(view, "short reply") {
		t.Fatalf("expected latest reply visible, got %q", view)
	}
}

// ScrollUp past line 0 clamps at 0 and disables auto-follow.
func TestHistoryScrollUpClampsAtZero(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 4
	for i := 0; i < 20; i++ {
		h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "line"}, true)
	}
	// Prime the cache so maxTopLine() is stable.
	_ = h.View()

	h.ScrollUp(9999)
	if h.topLine != 0 {
		t.Fatalf("expected topLine clamped to 0, got %d", h.topLine)
	}
	if h.followTail {
		t.Fatalf("expected followTail cleared after manual scroll")
	}
}

// Scrolling all the way down re-arms auto-follow so new content keeps
// tracking the tail.
func TestHistoryScrollDownAtMaxRearmsFollow(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 4
	for i := 0; i < 20; i++ {
		h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "line"}, true)
	}
	_ = h.View()

	h.ScrollUp(10) // park the view
	if h.followTail {
		t.Fatalf("setup: followTail should be false after ScrollUp")
	}
	h.ScrollDown(9999)
	if !h.followTail {
		t.Fatalf("expected followTail re-armed after scrolling to bottom")
	}
}

// When the user has scrolled up, appending new items should NOT snap the
// view back to the bottom — their scroll position is sticky.
func TestHistoryManualScrollIsSticky(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 4
	for i := 0; i < 20; i++ {
		h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "oldline"}, true)
	}
	_ = h.View()
	h.ScrollToTop()
	_ = h.View()
	topBefore := h.topLine

	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "freshline"}, true)
	view := stripANSI(h.View())

	if h.topLine != topBefore {
		t.Fatalf("expected topLine to stay at %d after append, got %d", topBefore, h.topLine)
	}
	if strings.Contains(view, "freshline") {
		t.Fatalf("expected new item not to snap into view while scrolled up, got:\n%s", view)
	}
}

// CHAT scope: cursor movement clamps to content bounds.
func TestHistoryCursorMoveClamped(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 10
	for i := 0; i < 3; i++ {
		h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "line"}, true)
	}
	_ = h.View()

	h.CursorTop()
	if h.cursorLine != 0 {
		t.Fatalf("expected cursorLine=0 after CursorTop, got %d", h.cursorLine)
	}
	h.CursorUp(5)
	if h.cursorLine != 0 {
		t.Fatalf("expected cursorLine clamped at 0, got %d", h.cursorLine)
	}
	h.CursorBottom()
	total := len(h.renderedLines())
	if h.cursorLine != total-1 {
		t.Fatalf("expected cursorLine=%d after CursorBottom, got %d", total-1, h.cursorLine)
	}
	h.CursorDown(999)
	if h.cursorLine != total-1 {
		t.Fatalf("expected cursorLine clamped at %d, got %d", total-1, h.cursorLine)
	}
}

// CHAT scope: entering visual mode anchors selection at cursor; expanding
// the cursor extends the selection range.
func TestHistoryVisualSelectionExpands(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 20
	h.AppendItem(TranscriptItem{Kind: TranscriptUser, Text: "first"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "second"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptUser, Text: "third"}, true)
	_ = h.View()

	h.CursorTop()
	h.EnterVisual()
	if !h.VisualActive() {
		t.Fatalf("expected visual active after EnterVisual")
	}
	h.CursorDown(2)
	lo, hi, ok := h.selectionRange()
	if !ok {
		t.Fatalf("expected active selection")
	}
	if lo != 0 || hi < 2 {
		t.Fatalf("expected selection [0..>=2], got [%d..%d]", lo, hi)
	}
}

// CHAT scope: ExitVisual clears selection without disturbing cursor.
func TestHistoryExitVisualClearsSelectionOnly(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 10
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "a"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "b"}, true)
	_ = h.View()

	h.CursorTop()
	h.EnterVisual()
	h.CursorDown(1)
	prev := h.cursorLine
	h.ExitVisual()
	if h.VisualActive() {
		t.Fatalf("expected visual cleared after ExitVisual")
	}
	if h.cursorLine != prev {
		t.Fatalf("expected cursorLine preserved (%d), got %d", prev, h.cursorLine)
	}
}

// Repeated CursorUp from the bottom drives the viewport up until the cursor
// hits rendered-line 0, then stays stable. No flicker / no stuck topLine.
func TestHistoryCursorUpScrollsViewportToTop(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 60
	h.height = 5
	h.SetActive(true)
	for i := 0; i < 40; i++ {
		h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "line"}, true)
	}
	// Prime cache + followTail-snapped topLine.
	_ = h.View()

	// Position cursor at bottom (simulates focus-on-history).
	h.EnsureCursor()
	_ = h.View()

	total := len(h.renderedLines())
	// Keep pressing k; after each press, the cursor must stay visible and
	// topLine must never be negative / past cursorLine.
	for step := 0; step < total+5; step++ {
		prevCursor := h.cursorLine
		h.CursorUp(1)
		view := h.View()
		_ = view

		if h.cursorLine < 0 {
			t.Fatalf("step=%d cursorLine went negative: %d", step, h.cursorLine)
		}
		if h.topLine < 0 {
			t.Fatalf("step=%d topLine went negative: %d", step, h.topLine)
		}
		if h.cursorLine < h.topLine {
			t.Fatalf("step=%d cursor=%d above topLine=%d (viewport not following)", step, h.cursorLine, h.topLine)
		}
		if h.cursorLine > prevCursor {
			t.Fatalf("step=%d cursor moved DOWN from %d to %d on CursorUp", step, prevCursor, h.cursorLine)
		}
	}
	if h.cursorLine != 0 {
		t.Fatalf("expected cursorLine=0 at top, got %d", h.cursorLine)
	}
	if h.topLine != 0 {
		t.Fatalf("expected topLine=0 at top, got %d", h.topLine)
	}
}

// After parking cursor at the top, successive CursorUp presses must be pure
// no-ops: View() output must be byte-identical so Bubble Tea can skip redraw.
func TestHistoryCursorUpAtTopIsStable(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 60
	h.height = 5
	h.SetActive(true)
	for i := 0; i < 20; i++ {
		h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "line"}, true)
	}
	_ = h.View()
	h.CursorTop()
	base := h.View()
	for i := 0; i < 5; i++ {
		h.CursorUp(1)
		got := h.View()
		if got != base {
			t.Fatalf("iteration %d: View differs at top\nbase=%q\ngot=%q", i, base, got)
		}
	}
}

// After parking cursor at the top, pressing j must scroll cursor AND viewport
// down in lockstep — the screen position should advance without "correcting"
// from a broken state.
func TestHistoryCursorDownAtTopAdvancesViewport(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 60
	h.height = 5
	h.SetActive(true)
	for i := 0; i < 20; i++ {
		h.AppendItem(TranscriptItem{Kind: TranscriptAssistant, Text: "line"}, true)
	}
	_ = h.View()
	h.CursorTop()
	_ = h.View()
	if h.cursorLine != 0 || h.topLine != 0 {
		t.Fatalf("setup: expected top state, got cursor=%d top=%d", h.cursorLine, h.topLine)
	}

	// Advance past pane height; once cursor crosses bottom row, topLine follows.
	for i := 0; i < h.height+3; i++ {
		prevCursor := h.cursorLine
		h.CursorDown(1)
		_ = h.View()
		if h.cursorLine <= prevCursor {
			t.Fatalf("iter %d: cursor did not advance (%d -> %d)", i, prevCursor, h.cursorLine)
		}
		if h.cursorLine >= h.topLine+h.height {
			t.Fatalf("iter %d: cursor=%d left viewport top=%d height=%d", i, h.cursorLine, h.topLine, h.height)
		}
	}
}

// Consecutive same-verb tool calls collapse into a single `⏺ Verb` header
// followed by `⎿ <arg>` continuation lines, mirroring Claude Code's batched
// tool display instead of repeating the glyph per call.
func TestHistoryAdjacentToolsGroupUnderOneHeader(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 20
	h.AppendItem(TranscriptItem{Kind: TranscriptTool, Text: "Read one.go"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptTool, Text: "Read two.go"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptTool, Text: "Read three.go"}, true)

	view := stripANSI(h.View())
	// A single header for the whole run.
	if strings.Count(view, "⏺ Read") != 1 {
		t.Fatalf("expected exactly one ⏺ Read header, got %q", view)
	}
	// Every file surfaces as a ⎿ continuation line.
	for _, name := range []string{"one.go", "two.go", "three.go"} {
		if !strings.Contains(view, "⎿ "+name) {
			t.Fatalf("expected `⎿ %s` continuation, got %q", name, view)
		}
	}
}

// A tool call with no neighbor of the same verb keeps the compact
// single-line form — grouping only kicks in when there's something to group.
func TestHistorySingletonToolStaysInline(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 10
	h.AppendItem(TranscriptItem{Kind: TranscriptTool, Text: "Read go.mod"}, true)

	view := stripANSI(h.View())
	if !strings.Contains(view, "⏺ Read go.mod") {
		t.Fatalf("expected inline single-line render, got %q", view)
	}
	if strings.Contains(view, "⎿") {
		t.Fatalf("singleton should not use ⎿ continuation, got %q", view)
	}
}

// A run of reads separated by a different verb breaks the group: the second
// Read starts its own header rather than continuing under the first.
func TestHistoryToolGroupBreaksOnDifferentVerb(t *testing.T) {
	h := NewHistoryScreen()
	h.width = 80
	h.height = 20
	h.AppendItem(TranscriptItem{Kind: TranscriptTool, Text: "Read one.go"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptTool, Text: "Searched TODO"}, true)
	h.AppendItem(TranscriptItem{Kind: TranscriptTool, Text: "Read two.go"}, true)

	view := stripANSI(h.View())
	if strings.Count(view, "⏺ Read") != 2 {
		t.Fatalf("expected two ⏺ Read headers split by Searched, got %q", view)
	}
	if !strings.Contains(view, "⏺ Searched TODO") {
		t.Fatalf("expected Searched singleton between reads, got %q", view)
	}
}
