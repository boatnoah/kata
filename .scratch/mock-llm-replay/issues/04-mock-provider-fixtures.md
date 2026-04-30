Status: needs-triage
Category: enhancement

# Implement mock provider (canned scenarios without recording)

## What to build

Add a “mock” provider that does not call any backend. It should serve a small set of deterministic scenarios useful for development and CI, for example:

- a simple assistant reply (no tools)
- a reply that includes a diff/patch needing approval
- bursts of tool calls + command output
- an error path

This is distinct from replay (recorded sessions): mock is hand-authored, minimal, deterministic.

## Acceptance criteria

- [ ] Mock provider selectable at runtime
- [ ] At least 3–5 canned scenarios available
- [ ] Scenarios exercise diff/approval UI paths
- [ ] Deterministic across runs

## Blocked by

- Provider selection wiring may be refactored first.

