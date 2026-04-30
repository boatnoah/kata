Status: needs-triage

# Migrate completion + tool metadata onto `AIStream`

## What to build

Move the remaining per-stream state — `aiCompleted`, `aiTurnPlaceholders`, `aiToolTitle`, `aiToolDetail`, `aiToolState` — onto `AIStream`. Tool-call streams may warrant a sub-type or a discriminated field (decide as part of this slice).

## Acceptance criteria

- [ ] Completion and tool-metadata state lives on `AIStream`
- [ ] Tool-call rendering unchanged (titles, details, state transitions)
- [ ] No remaining `map[itemID]X` fields on `App` for AI stream concerns

## Blocked by

- Issue 01 (`AIStream` type introduced)
