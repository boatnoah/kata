package tui

import (
	"strings"
	"testing"

	"github.com/boatnoah/kata/internal/agent"
)

func TestDiffBlockHeaderStats(t *testing.T) {
	t.Parallel()
	fc := agent.FileChange{
		Path:    "/repo/file.go",
		Op:      agent.FileChangeUpdate,
		Added:   3,
		Removed: 1,
	}
	lines := DiffBlock{File: fc}.Render(80, DefaultTheme())
	if len(lines) == 0 {
		t.Fatal("no lines")
	}
	header := stripANSIForLayout(lines[0])
	if !strings.Contains(header, "/repo/file.go") {
		t.Fatalf("missing path: %q", header)
	}
	if !strings.Contains(header, "+3") || !strings.Contains(header, "-1") {
		t.Fatalf("missing stat: %q", header)
	}
}

func TestDiffBlockHeaderRename(t *testing.T) {
	t.Parallel()
	fc := agent.FileChange{
		Path:     "/repo/old.go",
		Op:       agent.FileChangeUpdate,
		MovePath: "/repo/new.go",
	}
	lines := DiffBlock{File: fc}.Render(80, DefaultTheme())
	header := stripANSIForLayout(lines[0])
	if !strings.Contains(header, "→") || !strings.Contains(header, "/repo/new.go") {
		t.Fatalf("rename arrow: %q", header)
	}
}

func TestDiffBlockHeaderAddTag(t *testing.T) {
	t.Parallel()
	fc := agent.FileChange{Path: "/a.go", Op: agent.FileChangeAdd, Added: 2}
	lines := DiffBlock{File: fc}.Render(80, DefaultTheme())
	header := stripANSIForLayout(lines[0])
	if !strings.Contains(header, "new") {
		t.Fatalf("missing 'new' tag: %q", header)
	}
}

func TestDiffBlockBodySignColumn(t *testing.T) {
	t.Parallel()
	fc := agent.FileChange{
		Path: "/a.go",
		Op:   agent.FileChangeUpdate,
		Hunks: []agent.DiffHunk{{
			Header: "@@ -1,2 +1,3 @@",
			Lines: []agent.DiffLine{
				{Kind: agent.DiffLineContext, Text: "keep"},
				{Kind: agent.DiffLineRemove, Text: "gone"},
				{Kind: agent.DiffLineAdd, Text: "new"},
			},
		}},
	}
	lines := DiffBlock{File: fc}.Render(80, DefaultTheme())
	// header + hunk-header + 3 lines
	if len(lines) != 5 {
		t.Fatalf("len=%d lines=%v", len(lines), lines)
	}
	stripped := make([]string, len(lines))
	for i, l := range lines {
		stripped[i] = stripANSIForLayout(l)
	}
	if !strings.HasPrefix(stripped[1], "@@") {
		t.Fatalf("hunk header missing: %q", stripped[1])
	}
	if !strings.HasPrefix(stripped[2], "  ") {
		t.Fatalf("context sign: %q", stripped[2])
	}
	if !strings.HasPrefix(stripped[3], "- ") {
		t.Fatalf("remove sign: %q", stripped[3])
	}
	if !strings.HasPrefix(stripped[4], "+ ") {
		t.Fatalf("add sign: %q", stripped[4])
	}
}

func TestDiffBlockTruncation(t *testing.T) {
	t.Parallel()
	var body []agent.DiffLine
	for i := 0; i < 30; i++ {
		body = append(body, agent.DiffLine{Kind: agent.DiffLineAdd, Text: "x"})
	}
	fc := agent.FileChange{
		Path:  "/a.go",
		Op:    agent.FileChangeUpdate,
		Hunks: []agent.DiffHunk{{Lines: body}},
	}
	lines := DiffBlock{File: fc, MaxLines: 5}.Render(80, DefaultTheme())
	last := stripANSIForLayout(lines[len(lines)-1])
	if !strings.Contains(last, "truncated") || !strings.Contains(last, ":diff to review") {
		t.Fatalf("truncation footer: %q", last)
	}
	// Header + 4 preview lines + truncation footer = 6
	if len(lines) != 6 {
		t.Fatalf("len=%d", len(lines))
	}
}

func TestDiffBlockUnlimitedShowsAll(t *testing.T) {
	t.Parallel()
	var body []agent.DiffLine
	for i := 0; i < 50; i++ {
		body = append(body, agent.DiffLine{Kind: agent.DiffLineAdd, Text: "x"})
	}
	fc := agent.FileChange{
		Path:  "/a.go",
		Op:    agent.FileChangeUpdate,
		Hunks: []agent.DiffHunk{{Lines: body}},
	}
	lines := DiffBlock{File: fc, MaxLines: 0}.Render(80, DefaultTheme())
	// Header + 50 lines (no hunk header on this one).
	if len(lines) != 51 {
		t.Fatalf("len=%d", len(lines))
	}
}
