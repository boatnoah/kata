Status: needs-triage
Category: enhancement

# Add recorder wrapper (real provider → fixture)

## What to build

Create a wrapper around the “real” provider that:

- forwards all calls/events unchanged to the TUI
- records the full event stream into a fixture on disk
- optionally supports basic redaction hooks (opt-in) to avoid checking secrets into fixtures

Goal: make it easy to generate new fixtures from real usage, without spending tokens repeatedly.

## Acceptance criteria

- [ ] Wrapper does not change UI-visible behavior (no event drops/reordering beyond existing guarantees)
- [ ] Output fixture can be replayed by the replay provider
- [ ] Recorder can be enabled/disabled via a simple switch (env var/flag/config)
- [ ] Optional redaction mode is documented (even if minimal initially)

## Blocked by

- Depends on fixture format/codec (issue 01).
- Provider selection wiring may be refactored first.

