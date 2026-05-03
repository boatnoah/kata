Status: ready-for-human

# Migrate animation frame state onto `AIStream`

## What to build

Move the per-stream animation state — `aiVerbIdx`, `aiWaitFrames`, `aiTicking` — from `App` onto `AIStream`. The spinner helpers (`currentVerb`, `currentDots`, `advanceWaitingFrame`, `ensureWaitingVerb`, `waitingStatus`) become methods on `AIStream`. `handleAITick` and `scheduleAITick` stay on `App` and read/write tick state through `AIStream`.

### Decisions

- **`aiTicking` moves with verb/frame.** Per-stream lifecycle state of the same shape — collapse three more maps in one slice.
- **`handleAITick` stays on `App`.** It orchestrates `tea.Cmd` plumbing and history rendering. Original issue text suggested moving it onto `AIStream`; overriding because that pulls Bubble Tea into the type. State moves; orchestration stays.
- **`scheduleAITick` stays on `App`.** It produces a `tea.Cmd` — Bubble Tea concern, not stream concern.
- **`advanceAIStream` stays on `App`.** That's slice 03's territory.
- **`EnsureWaitingVerb` uses an internal `verbSet` bool.** The `int` zero value isn't a sentinel for "unpicked" the way map-presence was; an explicit flag preserves the no-overwrite semantic.

### API additions on AIStream

```go
// Tick scheduling state
func (s *AIStream) IsTicking() bool
func (s *AIStream) SetTicking(t bool)

// Spinner verb (rotating "Bootstrapping…", "Caramelizing…", etc.)
func (s *AIStream) EnsureWaitingVerb()       // idempotent; picks once
func (s *AIStream) CurrentVerb() string

// Wait frames (cycling dot pattern)
func (s *AIStream) AdvanceWaitingFrame()
func (s *AIStream) CurrentDots() string

// Composed for renderAIStream
func (s *AIStream) WaitingStatus() string

// Re-entry reset for beginOutboundWaitSync (drops verb + frames)
func (s *AIStream) ResetWaiting()
```

`SetTicking`, `EnsureWaitingVerb`, `AdvanceWaitingFrame`, and `ResetWaiting` are scaffolding for slice 02. Likely fates: `SetTicking` survives (real state transition); `ResetWaiting` survives in some form; `AdvanceWaitingFrame` may collapse into a single `Tick()` method in slice 05 if the lifecycle migrates.

### Call sites to touch

`App` struct + `New(...)` (drop `aiVerbIdx`, `aiWaitFrames`, `aiTicking`) · `teardownAIStreamKeys` (drop three lines) · `migrateAIStreamKeys` (drop three move/delete blocks) · `adoptThinkingPlaceholder` (drop three blocks) · `beginOutboundWaitSync` (`ResetWaiting`, `EnsureWaitingVerb`, `IsTicking`/`SetTicking`) · `upsertAIStream` (`IsTicking`/`SetTicking`) · `handleAITick` (`SetTicking`, `EnsureWaitingVerb`, `AdvanceWaitingFrame`) · `renderAIStream` (`WaitingStatus()`) · `finalizeAIStream` (drop three deletes) · `startAIThinking` (`EnsureWaitingVerb`, `IsTicking`/`SetTicking`).

The package-level `currentVerb` / `currentDots` / `waitingStatus` / `advanceWaitingFrame` / `ensureWaitingVerb` helpers on `App` go away.

## Acceptance criteria

- [ ] `App` struct loses `aiVerbIdx`, `aiWaitFrames`, `aiTicking` (3 fields → 0)
- [ ] `AIStream` owns verb idx (with set-flag), wait frames, and ticking
- [ ] `migrateAIStreamKeys`, `teardownAIStreamKeys`, `adoptThinkingPlaceholder` shrink: 3 fewer move/delete blocks each
- [ ] `aistream_test.go` adds tests for `EnsureWaitingVerb` idempotence, `AdvanceWaitingFrame` increment, `ResetWaiting` semantics, and `IsTicking`/`SetTicking` toggle — no `App`, no Bubble Tea
- [ ] `app_test.go` event-dispatch tests still pass (assertion updates allowed where they reach into the now-removed maps)
- [ ] No user-visible behavior change: spinner cycles, verb stays stable per stream until reset, dots animate at the same cadence

## Blocked by

Issue 01 (`AIStream` type introduced) — already shipped.
