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
	if !strings.Contains(view, "⎿ Read go.mod") {
		t.Fatalf("expected compact tool call with prefix, got %q", view)
	}
}
