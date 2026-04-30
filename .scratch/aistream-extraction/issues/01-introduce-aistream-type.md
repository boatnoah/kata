Status: needs-triage

# Introduce `AIStream` type; migrate rendered text + delta buffer

## What to build

Introduce an `AIStream` type in `internal/tui` that owns the state for one in-flight AI stream. Migrate the first slice of state — rendered text and the raw delta buffer (`aiStreams`, `aiRendered`) — onto it. `App` keeps `map[itemID]*AIStream`. Other parallel maps (animation, type-tick, completion, tool metadata) stay on `App` for now and migrate in later slices.

This is HITL — the surface of `AIStream` is the architectural decision. Open the conversation with what `AIStream` owns vs. what stays on `App` (e.g., does the map ownership stay with `App`? does `AIStream` know about rendering width?).

## Acceptance criteria

- [ ] `internal/tui` exposes an `AIStream` type with methods for the migrated state
- [ ] `App` holds `map[itemID]*AIStream` instead of `aiStreams` and `aiRendered`
- [ ] Stream behavior is unchanged from a user perspective
- [ ] At least one `AIStream`-level unit test exercises the migrated state without spinning up `App`

## Blocked by

None - can start immediately
