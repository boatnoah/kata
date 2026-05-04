Status: needs-triage

# UI jitter and rendering tears when scrolling with `j`/`k` mid-stream

## Symptom

While the AI is streaming a response (especially one that includes a tool-call row plus a long markdown body, e.g. *"Read go.mod"*), pressing `j` / `k` to scroll the history viewport produces:

- Visible jitter — the rendered region flickers as content updates and the viewport moves at the same time.
- A horizontal blank/white bar appears mid-screen, splitting the rendered content visually (see attached screenshot).
- The TUI layout looks broken until the stream completes or the user stops scrolling.

Confirmed pre-existing on `main` (commit `bcda63f`, post-slice-02). Not introduced by the AIStream extraction work — the streaming path's typing-tick math was identical before and after slices 01–03, and `history.go` (where rendering / cache / scroll plumbing lives) was not touched.

## Likely cause

`internal/tui/history.go` keeps a per-item `renderCache` and an `invalidateLines()` global cache buster. Every streamed delta calls `UpdateItemAt` which (in some paths) invalidates the cache, forcing a full re-layout. If the user is also scrolling, the scroll handler updates `topLine` / viewport state in the same Update tick, and the next `View()` paints a partially-recomputed frame.

Hypotheses to verify before settling on a fix:

- The cache is invalidated globally on every delta, even when only the streaming row changed (over-invalidation).
- Scroll handlers don't coordinate with mid-stream layout — they assume cached layout is current.
- `block.go` height/wrap recomputation during stream may produce inconsistent top/bottom anchors when the viewport is also moving.

## Repro

1. `go run ./cmd/kata`
2. Send a prompt that triggers a tool-call plus a longish body, e.g. *"Read go.mod and explain each section."*
3. While the response is streaming, hold `j` (or alternate `j`/`k`) to scroll the history viewport.
4. Observe jitter and the split rendering region.

Easier-to-trigger variants: longer responses, smaller terminal heights, faster scroll cadence.

## Open questions for HITL

- **Scope of fix.** Is this best fixed by (a) tightening cache invalidation so only the streaming row's cache entry is busted, (b) deferring scroll input while a delta is being applied within the same Update, or (c) reworking the layout pass so the frame is always self-consistent? Option (a) is the smallest change.
- **Acceptable visual.** Is *"slight stutter while both happen at once"* acceptable, or does it have to be perfectly smooth? Sets the bar for what counts as fixed.
- **Frame-rate gating.** Worth coalescing rapid deltas + scroll events into one frame (e.g. tick-driven repaint at 60 fps) instead of repainting on every Bubble Tea Update?
- **Reproduction in a test.** Is this reproducible with a fake provider in a headless test, or does it need a real terminal?

## Touched code (likely)

- `internal/tui/history.go` — `renderCache`, `invalidateLines`, `UpdateItemAt`, scroll API (`ScrollUp`, `ScrollDown`, `ScrollHalfPage*`)
- `internal/tui/block.go` — wrap/height recomputation
- `internal/tui/app.go` — `renderAIStream` (the per-delta caller of `UpdateItemAt`)

## Out of scope

- Changing the streaming cadence (35 ms / 3 runes per tick) — this is about layout coordination, not stream speed.
- Bubble Tea framework changes.

## Acceptance criteria

- [ ] Scrolling with `j` / `k` while a stream is in flight does not produce visible tears or split-region rendering
- [ ] No regression in the no-scroll path (streaming alone still reveals smoothly)
- [ ] If feasible, a regression test that drives a fake provider streaming + simulated scroll inputs and asserts the rendered frame remains consistent

## Blocked by

None.
