Status: needs-triage

# Queue `:w` submissions while a turn is in flight

## Problem

`:w` and `:wq` (`internal/tui/app.go:672-680`) fire `beginOutboundWaitSync` + `sendToAI` unconditionally, with no check for whether a turn is already streaming. If the user submits a second message mid-stream, both messages reach the Codex backend immediately. Ordering and interleaving are whatever Codex does on its end, and the UX shows two pending/streaming activity rows side-by-side with no indication that the second is conditional on the first.

## Desired behavior

Local FIFO queue. While a turn is in flight, `:w` enqueues the message instead of dispatching it. When the in-flight turn finalizes, the next queued message dispatches automatically. From the user's perspective: typing fast and hitting `:w` repeatedly should feel like sending a sequence of messages, not racing two against each other.

This is HITL — open the conversation on the questions below before writing code.

## Open questions

- **Visual cue.** When a message is queued, does it show in the transcript right away (e.g. user row + a "queued" status indicator) or stay invisible in the queue until it dispatches? Lean: append the user row immediately so the user sees their submission landed, with a faded / italic style until dispatch.
- **Drain trigger.** Drain on `EventTurnCompleted`, `EventAgentCompleted`, or the moment the in-flight `AIStream` finalizes? `EventTurnCompleted` is the conservative choice (waits for any tools in the same turn).
- **Error path.** If the in-flight turn errors out mid-stream, does the queue drain (next message kicks off a fresh turn) or pause for user input? Lean: pause — an error is unusual enough that the user should re-confirm.
- **Cancel.** Is there a way to drop a queued message before it dispatches? `:dq` ("drop queued")? Does `:q!` discard the queue? Does deleting the user row remove it?
- **Bound.** Cap queue at N? Probably overengineered for now; unbounded until we see a problem.
- **`:wq` semantics.** If queue is non-empty, does `:wq` flush before quitting or quit immediately? Lean: quit immediately — the user has explicitly said quit.

## Touched code

- `App` struct: add a queue field (`outboundQueue []string` or similar).
- `app.go:672-680` (`:w` handler) — branch on in-flight state: enqueue vs. dispatch.
- `app.go:906` (`sendToAI`) — likely extracted so the queue drain can call it.
- Codex event handler (`handleCodexEvent`) — drain trigger on the chosen completion event.

## Acceptance criteria

- [ ] `:w` while a turn is in flight does not call `client.SendText` immediately
- [ ] Queued message dispatches automatically when the in-flight turn finalizes
- [ ] User has a visible cue that a message is queued (specifics decided in HITL)
- [ ] Test through a fake provider: two `:w`s back-to-back result in only one `SendText` call until the first turn completes
- [ ] No regression in the no-overlap path (single message → immediate dispatch)

## Out of scope

- Interrupt-on-send (cancel current turn for new). Possible follow-up.
- Reject-on-send (refuse `:w` until idle). Possible follow-up.
- Codex-side parallel-message handling. Out of our control.

## Blocked by

None — can start once the open questions are settled.
