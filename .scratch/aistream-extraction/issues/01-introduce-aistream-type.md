Status: ready-for-human

# Introduce `AIStream` type; migrate label + buffer + rendered

## What to build

Introduce an `AIStream` type in a new file `internal/tui/aistream.go`. In slice 1, migrate the **content trio** — `aiStreams` (raw delta buffer), `aiRendered` (revealed substring), and `aiTypes` (the kind label) — onto it. `App` keeps `streams map[string]*AIStream`. All other parallel maps (`aiIndexes`, `aiVerbIdx`, `aiWaitFrames`, `aiCompleted`, `aiTicking`, `aiTurnPlaceholders`, `aiToolTitle/Detail/State`) stay on `App` for now and migrate in 02–04.

### Decisions

- **Map stays on `App`.** `AIStream` is a value, not self-managing.
- **`aiIndexes` stays on `App`.** History is App's concern; AIStream is content, not placement.
- **AIStream is width-unaware.** Rendering math stays in the renderer.
- **`aiTypes` moves with the buffer.** It is the "is this stream live" check (`renderAIStream` early return at app.go:1262, `isWaitingForAI` at app.go:1430). Splitting it from the buffer would put the existence flag and the data in different structs. Replacing three parallel maps with one struct in slice 1 is also the clearest payoff.
- **`aiCompleted` stays on `App` for slice 1.** It is tightly bound to the tick lifecycle and migrates in slice 02.
- **Sanitization stays in `App`.** `sanitizeStreamDelta` / `sanitizeHistoryMessage` are tangential; `AIStream` takes pre-cleaned strings.
- **Existence check becomes `a.streams[id] != nil`.** No `Live()` helper — call sites that did `_, ok := a.aiTypes[id]` switch to a nil-check.

### API

```go
type AIStream struct {
    label    TranscriptKind
    buffer   string  // accumulated raw deltas
    rendered string  // progressively revealed substring of buffer
}

func newAIStream(label TranscriptKind) *AIStream
func (s *AIStream) Label() TranscriptKind
func (s *AIStream) SetLabel(l TranscriptKind)   // upsert flips waiting→response on turn:* placeholders
func (s *AIStream) Buffer() string
func (s *AIStream) AppendDelta(clean string)    // buffer += clean
func (s *AIStream) ReplaceBuffer(text string)   // completed-branch in upsertAIStream
func (s *AIStream) Rendered() string
func (s *AIStream) SetRendered(text string)     // advanceAIStream still on App, writes back
```

App helpers:

```go
func (a *App) stream(id string) *AIStream                                 // nil if absent
func (a *App) ensureStream(id string, label TranscriptKind) *AIStream
```

`SetLabel`, `SetRendered`, and `ReplaceBuffer` are scaffolding — they exist because the lifecycle bodies still live on `App` in slice 1. They migrate or get repackaged as 02–04 pull those bodies onto `AIStream`.

### Call sites to touch

`New` (app.go:297) · `teardownAIStreamKeys` (app.go:1001) · `migrateAIStreamKeys` (app.go:1046) · `beginOutboundWaitSync` (app.go:1096) · `setToolCallSummary` (app.go:1150) · `upsertAIStream` (app.go:1173) · `handleAITick` (app.go:1215) · `advanceAIStream` (app.go:1247) · `renderAIStream` (app.go:1261) · `finalizeAIStream` (app.go:1289) · `startAIThinking` (app.go:1354) · `adoptThinkingPlaceholder` (app.go:1396) · `isWaitingForAI` (app.go:1430) · plus the `aiTypes` iteration in the network reset path (app.go:959).

## Acceptance criteria

- [x] `internal/tui/aistream.go` exists; `AIStream` owns `label`, `buffer`, `rendered`
- [x] `App.streams map[string]*AIStream` replaces `aiStreams`, `aiRendered`, `aiTypes` (3 fields → 1)
- [x] `migrateAIStreamKeys` and `teardownAIStreamKeys` shrink: three move/delete blocks each become one
- [x] All other parallel maps remain unchanged
- [x] `app_test.go` event-dispatch tests pass — see comment below for the small caveat
- [x] `aistream_test.go` exercises `AppendDelta` accumulation, `ReplaceBuffer` overwrite, and `SetRendered` writeback — no `App`, no Bubble Tea
- [x] No user-visible behavior change (waiting spinner, typing reveal, tool-call rows render identically)

## Blocked by

None — can start immediately.

## Comments

- **2026-05-01:** Implemented on `refactor/aistream-content-trio`, shipped as PR #6 (https://github.com/boatnoah/kata/pull/6). Smoke-tested locally against the real Codex backend: streaming reveal, `:w` outbound placeholder → turn adoption, and tool-call rendering all unchanged.
- Caveat on the "tests pass without modification" criterion: three assertions in `app_test.go` reached into the now-private `aiStreams` / `aiRendered` / `aiTypes` maps directly (`drainAIStream`, `TestUpsertAIStreamRevealsTextProgressively`, `TestRenderAIStreamShowsWaitingStatusWhenCaughtUp`, plus the post-finalize check in the cmd-3 test). Those were updated to use `app.stream(id)` / `ensureStream(id, label)`. Test *behavior* is unchanged; only assertions on internal field names had to track the refactor.
