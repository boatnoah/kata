Status: needs-triage

# Migrate remaining approval kinds to typed

## What to build

Apply the typed approval pattern to the remaining server-initiated request kinds:

- `mcpServer/elicitation/request`
- `item/tool/requestUserInput`
- `item/tool/call` (dynamic tool call)
- `execCommandApproval`

These share enough structure with the earlier slices that they migrate together as a single PR.

## Acceptance criteria

- [ ] All four kinds expose typed approval values in `internal/agent`
- [ ] Codex translates each kind in both directions
- [ ] TUI handles each via typed dispatch
- [ ] Tests cover each typed path

## Blocked by

- Issue 01 (typed approval shape locked in)
