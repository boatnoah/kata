# Session persistence commands

## What to build

Implement durable session persistence in `internal/session` and expose it through five command-mode commands:

- `:save [name]` — write the current session (transcript history at minimum, plus session ID, model, and any state needed to resume) to disk under a name. If `name` is omitted, save under the current session ID.
- `:load <name>` — read a saved session and replace the current state with it.
- `:sessions` (alias `:ls`) — list saved sessions, sorted by most recently modified, with name, ID, and last-modified time.
- `:fork` — duplicate the current session to a new name (e.g. `<current>-fork-N`) and switch to it. The original is left intact.
- `:rename <name>` — rename the current session's saved file in place.

This is the first real consumer of `internal/session/store.go` and `internal/session/session.go`, which currently only have package declarations. The architectural decisions sit in this slice:

- What goes into a saved session: transcript items, session ID, model, possibly approval grants, possibly thread/turn IDs for resuming the upstream provider. Pick the minimum that lets `:load` produce an equivalent UI to the live session it was saved from.
- File format and location (e.g., JSON under `$XDG_DATA_HOME/kata/sessions/<name>.json`).
- Whether `:fork` and `:rename` operate on the on-disk file or also on a separate in-memory metadata layer.
- How `:load` interacts with an in-flight turn (probably refuse, or cancel first).

## Acceptance criteria

- [ ] `internal/session/store.go` exposes a Save / Load / List / Fork / Rename surface
- [ ] `:save [name]` persists the current session and reports the path or name
- [ ] `:load <name>` restores transcript and session state
- [ ] `:fork` produces an independent copy and switches to it; original left intact
- [ ] `:rename <name>` renames in place
- [ ] `:sessions` / `:ls` lists sessions sorted by recency
- [ ] Round-trip test: save a session, mutate state, load it back, assert equivalence
- [ ] `:load` refuses (or cancels) gracefully when a turn is in flight
- [ ] The existing `:sess` stub is either repurposed as `:sessions` or removed

## Blocked by

None - can start immediately
