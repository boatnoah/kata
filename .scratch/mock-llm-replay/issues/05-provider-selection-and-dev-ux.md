Status: needs-triage
Category: enhancement

# Wire provider selection (real/mock/record/replay) + dev UX

## What to build

Add a simple, low-friction way to choose how `kata` gets AI events:

- real backend (current default)
- mock provider
- replay provider (fixture path/name)
- record mode (wrap real backend and write fixture)

Prefer minimal surface area (env vars or flags) and good error messages.

## Acceptance criteria

- [ ] Selecting mock/replay/record does not require code changes
- [ ] Replay accepts a fixture path (and optionally a standard fixtures directory lookup)
- [ ] Record accepts an output path and refuses to overwrite by default (or uses timestamps)
- [ ] Help text / README section documents the switches

## Blocked by

- Might be easier after provider refactors the maintainer plans.

