Status: ready-for-human

# Migrate completion + tool metadata onto `AIStream`

## What to build

Move the remaining per-stream itemID-keyed state — `aiCompleted`, `aiToolTitle`, `aiToolDetail`, `aiToolState` — onto `AIStream`. After this slice, the only itemID-keyed `App` maps are `streams` and `aiIndexes` (history index). `aiTurnPlaceholders` stays where it is.

### Reframe

The original issue text included `aiTurnPlaceholders` in the migration list, but that map is keyed by **turnID → itemID** — it's a turn-level index ("which placeholder represents this turn"), not per-stream state. It belongs on `App` for the same reason `aiIndexes` does.

### Decisions

- **Plain fields on `AIStream`.** Tool-call streams don't get a discriminated sub-type. The fields are read only when `label == TranscriptTool`; flat structure beats union ceremony for a one-bool guard.
- **`Complete(finalText string)` collapses the two-step pattern.** Today `upsertAIStream` does `s.ReplaceBuffer(finalText)` (conditional on non-empty) followed by `a.aiCompleted[id] = true`. One method captures the semantic: "mark done; if final text was provided, lock it in".
- **`UpdateTool` preserves prior on empty title/detail.** Same semantic as today's `setToolCallSummary` so a "started → completed" sequence with only state changes doesn't blank the title.
- **`Reset()` clears buffer + rendered.** Used by `setToolCallSummary` on tool transitions where prior streamed text should be wiped.
- **Drop `SetRendered` and `ReplaceBuffer`.** Predicted in slice 01. Their callers move to `Reset` and `Complete`.
- **`teardownAIStreamKeys` becomes vestigial.** Down to a single `delete(a.streams, id)` after this slice; inline at the one caller (`abandonOutboundPending`) and delete the helper.
- **Drop `aiCompleted[id] = false` in `beginOutboundWaitSync`.** Fresh streams default to `completed = false`; the explicit zero was redundant once completion lives on the struct.

### API additions on AIStream

```go
// Completion lifecycle
func (s *AIStream) IsCompleted() bool
func (s *AIStream) SetCompleted(c bool)
// Complete marks the stream as completed; if finalText is non-empty,
// replaces the buffer with it. Empty finalText preserves accumulated text
// (some completion events arrive without a final-text snapshot).
func (s *AIStream) Complete(finalText string)

// Tool metadata
func (s *AIStream) Tool() (title, detail string, state ToolState)
// UpdateTool merges incoming summary into the stream's tool fields.
// Empty Title or Detail in the summary preserve the prior values.
func (s *AIStream) UpdateTool(summary toolSummary)

// Reset clears buffer + rendered (for tool transitions that wipe prior text).
func (s *AIStream) Reset()
```

### API removed

- `SetRendered`
- `ReplaceBuffer`

### Call sites to touch

`App` struct + `New(...)` (drop `aiCompleted`, `aiToolTitle`, `aiToolDetail`, `aiToolState`) · `teardownAIStreamKeys` (delete; inline at caller) · `abandonOutboundPending` (inline `delete(a.streams, itemID)`) · `migrateAIStreamKeys` (drop 4 move/delete blocks) · `adoptThinkingPlaceholder` (drop `delete(a.aiCompleted, placeholderID)`) · `beginOutboundWaitSync` (drop `aiCompleted[id] = false`) · `setToolCallSummary` (rewrite using `Reset` + `SetCompleted` + `UpdateTool`) · `upsertAIStream` (use `Complete(finalText)` for the completed branch; replace `aiCompleted[id]` reads with `s.IsCompleted()`) · `handleAITick` (`s.IsCompleted()`) · `renderAIStream` (`s.IsCompleted()`, `s.Tool()` for the TranscriptTool branch) · `finalizeAIStream` (`s.Tool()`; collapse to `delete(a.streams, itemID)`) · `isWaitingForAI` (`!s.IsCompleted()`).

## Acceptance criteria

- [ ] `App` struct loses `aiCompleted`, `aiToolTitle`, `aiToolDetail`, `aiToolState` (4 fields → 0)
- [ ] `AIStream` owns `completed`, `toolTitle`, `toolDetail`, `toolState`
- [ ] `migrateAIStreamKeys` shrinks: 4 fewer move/delete blocks
- [ ] `teardownAIStreamKeys` deleted; inlined at `abandonOutboundPending`
- [ ] `setToolCallSummary` reduced to ~6 lines using `Reset` + `SetCompleted` + `UpdateTool`
- [ ] `SetRendered` and `ReplaceBuffer` removed from `AIStream`
- [ ] `aistream_test.go` adds tests for `Complete` (with and without finalText), `UpdateTool` (empty preserves prior), `Reset`, `IsCompleted`/`SetCompleted` toggle. Remove tests for the removed methods.
- [ ] `app_test.go` event-dispatch tests pass (assertion updates allowed where they reach into the now-removed maps)
- [ ] No user-visible behavior change: tool-call titles/details/states render identically, "started → completed" with empty title in completion preserves prior title, completion finalizes correctly

## Blocked by

Issue 01 (`AIStream` type introduced) — already shipped.
