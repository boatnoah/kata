package tui

import (
	"fmt"
	"strings"
	"testing"
)

func TestComposeViewCapsHeight(t *testing.T) {
	c := NewCompose()
	c.SetActive(true)

	for i := 0; i < 20; i++ {
		c.buf = append(c.buf, []rune(fmt.Sprintf("line%02d\n", i))...)
	}
	c.cursor = len(c.buf) // place cursor at end so it must remain visible

	view, lines := c.View(80, 5, "")

	if lines > 5 {
		t.Fatalf("expected capped height of at most 5 lines, got %d", lines)
	}

	if got := strings.Count(view, "\n") + 1; got != lines {
		t.Fatalf("line count mismatch: reported %d, actual %d", lines, got)
	}

	if !strings.Contains(view, "line19") {
		t.Fatalf("expected view to include bottom lines near the cursor")
	}
}

func TestComposeViewCapsWrappedLongLineNearCursor(t *testing.T) {
	c := NewCompose()
	c.SetActive(true)
	c.buf = []rune("Compose a very long line that should wrap several times in the viewport while keeping the cursor visible near the end")
	c.cursor = len(c.buf)

	view, lines := c.View(30, 5, "")

	if lines > 5 {
		t.Fatalf("expected capped height of at most 5 lines, got %d", lines)
	}
	if !strings.Contains(view, "near the end") {
		t.Fatalf("expected wrapped view to keep tail near cursor visible, got %q", view)
	}
}
