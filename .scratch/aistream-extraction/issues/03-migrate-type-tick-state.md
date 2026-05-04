Status: ready-for-human

# Migrate the typing-reveal tick onto `AIStream`

## What to build

Move the typing animation tick — the rune-by-rune progressive reveal currently in `App.advanceAIStream` — onto `AIStream` as a method. Drop the now-unneeded `AIStream.SetRendered` and `App.advanceAIStream`.

### Reframe

The original issue text said "type-tick state — `aiTypes`, `aiIndexes`" but that's stale: `aiTypes` already moved in slice 01, and `aiIndexes` stays on `App` (history is App's concern, decided in slice 01). The genuine "type tick" is the typing animation tick — the rune-slicing reveal that runs every `aiTypeInterval`. That's what this slice migrates.

### Decisions

- **`Advance` takes `runesPerTick` as a parameter.** Hard-coding `aiRunesPerTick` inside the method couples tests to the package constant; passing it in keeps tests explicit about what they're exercising.
- **Drop `App.advanceAIStream`** entirely. The two callers (`handleAITick`, `upsertAIStream`) already hold a non-nil `*AIStream`; they call `s.Advance(aiRunesPerTick)` directly. No thin wrapper.
- **`Advance` does not nil-check.** Callers already gate on a non-nil stream.
- **Keep `SetRendered` alive in slice 03.** Slice 01 predicted it would die here, but `setToolCallSummary` still uses `SetRendered("")` to reset rendered text on a tool transition. Slice 04 (completion + tool metadata migration) restructures `setToolCallSummary` anyway; that's the natural home for the cleanup. Tighter slice 03 wins.

### API change

```go
// New on AIStream
func (s *AIStream) Advance(runesPerTick int)   // rune-slicing reveal math; same semantics as old advanceAIStream
```

### Call sites to touch

`App.advanceAIStream` (delete) · `handleAITick` (replace `a.advanceAIStream(itemID)` with `s.Advance(aiRunesPerTick)`) · `upsertAIStream` (same).

`aistream_test.go`: add new `Advance` tests; existing `TestAIStream_SetRenderedWriteback` stays for now (removed in slice 04).

## Acceptance criteria

- [ ] `AIStream.Advance(runesPerTick int)` exists with byte-identical reveal semantics to old `advanceAIStream`
- [ ] `App.advanceAIStream` removed
- [ ] `aistream_test.go` covers `Advance`: empty buffer no-op, partial reveal, lock-in at completion, idempotent past-completion
- [ ] `app_test.go` event-dispatch tests still pass
- [ ] No user-visible behavior change (typing reveal cadence and final output identical)

## Blocked by

Issue 01 (`AIStream` type introduced) — already shipped.
