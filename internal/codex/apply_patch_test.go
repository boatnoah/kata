package codex

import (
	"encoding/json"
	"testing"

	"github.com/boatnoah/kata/internal/agent"
)

func TestParseFileChangesEmpty(t *testing.T) {
	t.Parallel()
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`{}`)} {
		got, err := ParseFileChanges(raw)
		if err != nil {
			t.Fatalf("ParseFileChanges(%q) err: %v", raw, err)
		}
		if len(got) != 0 {
			t.Fatalf("ParseFileChanges(%q) = %+v, want empty", raw, got)
		}
	}
}

func TestParseFileChangesAdd(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"/a/new.go":{"add":{"content":"package a\n\nfunc X() {}\n"}}}`)
	got, err := ParseFileChanges(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	fc := got[0]
	if fc.Path != "/a/new.go" || fc.Op != agent.FileChangeAdd {
		t.Fatalf("path/op: %+v", fc)
	}
	if fc.Added != 3 || fc.Removed != 0 {
		t.Fatalf("stat: +%d -%d", fc.Added, fc.Removed)
	}
	if len(fc.Hunks) != 1 || len(fc.Hunks[0].Lines) != 3 {
		t.Fatalf("hunks: %+v", fc.Hunks)
	}
	for _, l := range fc.Hunks[0].Lines {
		if l.Kind != agent.DiffLineAdd {
			t.Fatalf("kind %q", l.Kind)
		}
	}
}

func TestParseFileChangesDelete(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"/a/gone.go":{"delete":{}}}`)
	got, err := ParseFileChanges(raw)
	if err != nil {
		t.Fatal(err)
	}
	fc := got[0]
	if fc.Op != agent.FileChangeDelete {
		t.Fatalf("op %q", fc.Op)
	}
	if len(fc.Hunks) != 1 || fc.Hunks[0].Lines[0].Kind != agent.DiffLineRemove {
		t.Fatalf("hunk shape: %+v", fc.Hunks)
	}
}

func TestParseFileChangesUpdateMultiHunk(t *testing.T) {
	t.Parallel()
	diff := "@@ -1,3 +1,4 @@\n ctx\n-old\n+new1\n+new2\n@@ -10,2 +11,2 @@\n ctx2\n-bye\n+hi\n"
	body, err := json.Marshal(map[string]any{
		"/a/file.go": map[string]any{
			"update": map[string]any{
				"unified_diff": diff,
				"move_path":    "/a/renamed.go",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseFileChanges(body)
	if err != nil {
		t.Fatal(err)
	}
	fc := got[0]
	if fc.Op != agent.FileChangeUpdate || fc.MovePath != "/a/renamed.go" {
		t.Fatalf("op/move: %+v", fc)
	}
	if fc.Added != 3 || fc.Removed != 2 {
		t.Fatalf("stat: +%d -%d", fc.Added, fc.Removed)
	}
	if len(fc.Hunks) != 2 {
		t.Fatalf("hunks: %d", len(fc.Hunks))
	}
	if fc.Hunks[0].Header != "@@ -1,3 +1,4 @@" {
		t.Fatalf("hunk0 header: %q", fc.Hunks[0].Header)
	}
}

func TestParseFileChangesUpdateNoHunkHeader(t *testing.T) {
	t.Parallel()
	// Tolerate malformed diffs with no @@ header — treat the body as a
	// single synthetic hunk so we still show something useful.
	body, err := json.Marshal(map[string]any{
		"/a.go": map[string]any{"update": map[string]any{"unified_diff": "-old\n+new\n"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseFileChanges(body)
	if err != nil {
		t.Fatal(err)
	}
	fc := got[0]
	if fc.Added != 1 || fc.Removed != 1 {
		t.Fatalf("stat: +%d -%d", fc.Added, fc.Removed)
	}
	if len(fc.Hunks) != 1 || fc.Hunks[0].Header != "" {
		t.Fatalf("hunks: %+v", fc.Hunks)
	}
}

func TestParseFileChangesSorted(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{
		"/b.go": {"delete": {}},
		"/a.go": {"delete": {}},
		"/c.go": {"delete": {}}
	}`)
	got, err := ParseFileChanges(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Path != "/a.go" || got[1].Path != "/b.go" || got[2].Path != "/c.go" {
		t.Fatalf("order: %+v", got)
	}
}

func TestParseFileChangesRejectsUnknownVariant(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"/x.go":{}}`)
	if _, err := ParseFileChanges(raw); err == nil {
		t.Fatal("expected error for empty variant")
	}
}

func TestApplyPatchApprovalEventIncludesParsedFileChanges(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"callId":"c1","conversationId":"th","fileChanges":{"/a.go":{"add":{"content":"x\n"}}},"reason":"r"}`)
	ev, err := applyPatchApprovalEvent(json.RawMessage(`1`), raw)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := ev.Payload[approvalPayloadParsedFileChanges]
	if !ok {
		t.Fatal("missing parsedFileChanges")
	}
	parsed, ok := v.([]agent.FileChange)
	if !ok {
		t.Fatalf("type: %T", v)
	}
	if len(parsed) != 1 || parsed[0].Path != "/a.go" || parsed[0].Op != agent.FileChangeAdd {
		t.Fatalf("parsed: %+v", parsed)
	}
}
