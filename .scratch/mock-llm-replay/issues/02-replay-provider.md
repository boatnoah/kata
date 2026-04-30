Status: needs-triage
Category: enhancement

# Implement replay provider (fixture → Events stream)

## What to build

Implement a provider implementation that reads a fixture and emits the recorded event stream through `Events()`. It should support:

- fast mode (no sleeps; emits as quickly as the consumer can read)
- optional paced mode (replays approximate recorded timing)
- correct start/close lifecycle behavior expected by the TUI

## Acceptance criteria

- [ ] Replay provider can run the TUI without contacting a real backend
- [ ] Fast mode is the default and suitable for tests
- [ ] Paced mode is optional and can be enabled for manual UI validation
- [ ] Clear errors when fixture is missing/invalid, including location info (line/event index)

## Blocked by

- Depends on fixture format/codec being defined (issue 01).

