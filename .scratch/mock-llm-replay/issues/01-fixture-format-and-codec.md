Status: needs-triage
Category: enhancement

# Define fixture format + codec for provider event streams

## What to build

Specify and implement a stable, versioned fixture format that can represent a stream of AI provider events (assistant deltas, tool calls, command output, diffs, errors, completion) plus minimal metadata (provider label, model label, optional timing).

The format should be easy to diff/review (line-oriented) and forward-compatible (unknown fields tolerated).

## Acceptance criteria

- [ ] A written fixture schema (header + event records) with a version field
- [ ] Encode/decode API that round-trips fixtures without loss
- [ ] Parser tolerates unknown fields for forward compatibility
- [ ] Fixtures are line-oriented and human-reviewable

## Blocked by

- Provider/event shapes may change during upcoming refactors; schema should be designed to evolve.

