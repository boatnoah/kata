package codex

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/boatnoah/kata/internal/agent"
)

// Payload key under which applyPatch approval events expose their parsed
// per-file changes. The UI reads this to render a diff view.
const approvalPayloadParsedFileChanges = "parsedFileChanges"

// wireFileChange mirrors the Codex app-server FileChange shape — a map
// keyed by absolute path whose value is one of three single-key objects:
// {"add":{"content":...}}, {"update":{"unified_diff":...,"move_path":...}},
// or {"delete":{}}.
type wireFileChange struct {
	Add    *wireFileAdd    `json:"add,omitempty"`
	Update *wireFileUpdate `json:"update,omitempty"`
	Delete *wireFileDelete `json:"delete,omitempty"`
}

type wireFileAdd struct {
	Content string `json:"content"`
}

type wireFileUpdate struct {
	UnifiedDiff string `json:"unified_diff"`
	MovePath    string `json:"move_path,omitempty"`
}

type wireFileDelete struct{}

// ParseFileChanges decodes Codex's fileChanges map into a provider-neutral
// slice. The result is sorted by path so UI rendering is deterministic.
// An empty or null input returns (nil, nil).
func ParseFileChanges(raw json.RawMessage) ([]agent.FileChange, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var byPath map[string]wireFileChange
	if err := json.Unmarshal(raw, &byPath); err != nil {
		return nil, fmt.Errorf("codex: decode fileChanges: %w", err)
	}
	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	out := make([]agent.FileChange, 0, len(paths))
	for _, p := range paths {
		fc, err := convertFileChange(p, byPath[p])
		if err != nil {
			return nil, err
		}
		out = append(out, fc)
	}
	return out, nil
}

func convertFileChange(path string, w wireFileChange) (agent.FileChange, error) {
	switch {
	case w.Add != nil:
		hunk, added := synthesizeAddHunk(w.Add.Content)
		return agent.FileChange{
			Path:  path,
			Op:    agent.FileChangeAdd,
			Hunks: []agent.DiffHunk{hunk},
			Added: added,
		}, nil
	case w.Delete != nil:
		return agent.FileChange{
			Path: path,
			Op:   agent.FileChangeDelete,
			Hunks: []agent.DiffHunk{{
				Lines: []agent.DiffLine{{Kind: agent.DiffLineRemove, Text: "— file removed —"}},
			}},
		}, nil
	case w.Update != nil:
		hunks, added, removed := parseUnifiedDiff(w.Update.UnifiedDiff)
		return agent.FileChange{
			Path:     path,
			Op:       agent.FileChangeUpdate,
			MovePath: w.Update.MovePath,
			Hunks:    hunks,
			Added:    added,
			Removed:  removed,
		}, nil
	default:
		return agent.FileChange{}, fmt.Errorf("codex: fileChanges[%q] has no add/update/delete variant", path)
	}
}

func synthesizeAddHunk(content string) (agent.DiffHunk, int) {
	if content == "" {
		return agent.DiffHunk{}, 0
	}
	trimmed := strings.TrimRight(content, "\n")
	lines := strings.Split(trimmed, "\n")
	out := make([]agent.DiffLine, 0, len(lines))
	for _, line := range lines {
		out = append(out, agent.DiffLine{Kind: agent.DiffLineAdd, Text: line})
	}
	return agent.DiffHunk{Lines: out}, len(out)
}

// parseUnifiedDiff splits a unified diff body into hunks. The leading
// "---"/"+++" file headers some producers emit are skipped. A diff with
// no "@@" header is tolerated as a single synthetic hunk.
func parseUnifiedDiff(body string) ([]agent.DiffHunk, int, int) {
	if body == "" {
		return nil, 0, 0
	}
	var hunks []agent.DiffHunk
	var current *agent.DiffHunk
	var added, removed int

	flush := func() {
		if current != nil && (current.Header != "" || len(current.Lines) > 0) {
			hunks = append(hunks, *current)
		}
		current = nil
	}

	for _, raw := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
		switch {
		case strings.HasPrefix(raw, "@@"):
			flush()
			current = &agent.DiffHunk{Header: raw}
		case strings.HasPrefix(raw, "--- ") || strings.HasPrefix(raw, "+++ "):
			// File headers — irrelevant for per-line coloring.
			continue
		default:
			if current == nil {
				current = &agent.DiffHunk{}
			}
			kind, text := classifyDiffLine(raw)
			switch kind {
			case agent.DiffLineAdd:
				added++
			case agent.DiffLineRemove:
				removed++
			}
			current.Lines = append(current.Lines, agent.DiffLine{Kind: kind, Text: text})
		}
	}
	flush()
	return hunks, added, removed
}

func classifyDiffLine(raw string) (agent.DiffLineKind, string) {
	if raw == "" {
		return agent.DiffLineContext, ""
	}
	switch raw[0] {
	case '+':
		return agent.DiffLineAdd, raw[1:]
	case '-':
		return agent.DiffLineRemove, raw[1:]
	case ' ':
		return agent.DiffLineContext, raw[1:]
	default:
		// "\ No newline at end of file" markers and other oddities — treat as
		// context so they don't skew +/- counts.
		return agent.DiffLineContext, raw
	}
}
