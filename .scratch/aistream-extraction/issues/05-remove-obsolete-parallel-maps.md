Status: needs-triage

# Remove obsolete parallel maps; consolidate lifecycle on `AIStream`

## What to build

Final cleanup once all state is on `AIStream`:

- Remove the now-empty parallel maps from `App`.
- Consolidate stream lifecycle (creation, append, complete, tick, cleanup) into `AIStream` methods rather than `App` helpers.
- Add coverage gaps as needed; `AIStream` should be testable in isolation.

## Acceptance criteria

- [ ] No per-stream maps remain on `App` other than `map[itemID]*AIStream`
- [ ] Stream lifecycle methods live on `AIStream`, not as `App` helpers
- [ ] Unit tests for `AIStream` cover the streaming lifecycle without `App`

## Blocked by

- Issue 02 (animation frame migrated)
- Issue 03 (type-tick migrated)
- Issue 04 (completion + tool migrated)
