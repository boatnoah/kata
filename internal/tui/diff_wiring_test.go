package tui

import (
	"encoding/json"
	"testing"

	"github.com/boatnoah/kata/internal/agent"
	"github.com/boatnoah/kata/internal/codex"
)

func applyPatchEvent(files []agent.FileChange) agent.Event {
	return agent.Event{
		Type:     agent.EventApprovalRequired,
		ThreadID: "t",
		ItemID:   "c1",
		Text:     "apply patch",
		RPCID:    json.RawMessage(`1`),
		Payload: map[string]any{
			"approvalKind":      codex.ApprovalKindApplyPatch,
			"parsedFileChanges": files,
		},
	}
}

func TestApplyPatchApprovalAppendsDiffItems(t *testing.T) {
	t.Parallel()
	app := NewApp()
	stub := &stubAgentProvider{}
	app.ai = newAIManagerWithClient(stub)

	files := []agent.FileChange{
		{Path: "/a.go", Op: agent.FileChangeAdd, Added: 1, Hunks: []agent.DiffHunk{{Lines: []agent.DiffLine{{Kind: agent.DiffLineAdd, Text: "x"}}}}},
		{Path: "/b.go", Op: agent.FileChangeDelete, Hunks: []agent.DiffHunk{{Lines: []agent.DiffLine{{Kind: agent.DiffLineRemove, Text: "—"}}}}},
	}
	_ = app.handleCodexEvent(applyPatchEvent(files))

	var diffCount int
	for _, it := range app.history.items {
		if it.Kind == TranscriptDiff {
			diffCount++
			if it.DiffMaxLines != applyPatchDiffPreviewLines {
				t.Fatalf("preview cap: %d", it.DiffMaxLines)
			}
		}
	}
	if diffCount != 2 {
		t.Fatalf("diff items: %d", diffCount)
	}
	if len(app.lastPatchFiles) != 2 {
		t.Fatalf("lastPatchFiles: %d", len(app.lastPatchFiles))
	}
}

func TestDiffCommandFlashesWhenNoPatch(t *testing.T) {
	t.Parallel()
	app := NewApp()
	stub := &stubAgentProvider{}
	app.ai = newAIManagerWithClient(stub)

	_ = app.runCommand("diff")
	if app.statusNotice != "no pending patch" {
		t.Fatalf("status: %q", app.statusNotice)
	}
}

func TestDiffCommandAppendsFullDiffs(t *testing.T) {
	t.Parallel()
	app := NewApp()
	stub := &stubAgentProvider{}
	app.ai = newAIManagerWithClient(stub)
	app.lastPatchFiles = []agent.FileChange{
		{Path: "/a.go", Op: agent.FileChangeAdd},
	}
	before := len(app.history.items)
	_ = app.runCommand("diff")
	if got := len(app.history.items) - before; got != 1 {
		t.Fatalf("appended %d items", got)
	}
	tail := app.history.items[len(app.history.items)-1]
	if tail.Kind != TranscriptDiff {
		t.Fatalf("kind %q", tail.Kind)
	}
	if tail.DiffMaxLines != 0 {
		t.Fatalf(":diff should be unlimited, got %d", tail.DiffMaxLines)
	}
}

func TestNonApplyPatchApprovalSkipsDiffs(t *testing.T) {
	t.Parallel()
	app := NewApp()
	stub := &stubAgentProvider{}
	app.ai = newAIManagerWithClient(stub)

	ev := agent.Event{
		Type:    agent.EventApprovalRequired,
		RPCID:   json.RawMessage(`1`),
		Payload: map[string]any{"approvalKind": codex.ApprovalKindCommandExecution},
	}
	_ = app.handleCodexEvent(ev)
	for _, it := range app.history.items {
		if it.Kind == TranscriptDiff {
			t.Fatal("unexpected diff item for non-applyPatch approval")
		}
	}
}
