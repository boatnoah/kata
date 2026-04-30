Status: needs-triage
Category: enhancement

# Add tests + sample fixtures for offline CI coverage

## What to build

Use mock/replay to add tests that validate external TUI behavior without paid calls:

- replay-driven tests that assert transcript/diff blocks render and approvals apply patches
- codec round-trip tests for fixture parsing
- a small set of checked-in sample fixtures (sanitized) that exercise the key paths

## Acceptance criteria

- [ ] CI can run at least one “approval flow” test without network/subscriptions
- [ ] Fixture codec has round-trip coverage
- [ ] Sample fixtures are small, reviewed, and sanitized

## Blocked by

- Depends on fixture codec + replay provider (issues 01–02).

