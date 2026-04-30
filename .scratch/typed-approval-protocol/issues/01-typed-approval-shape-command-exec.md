Status: needs-triage

# Define typed approval shape + migrate `CommandExecApproval`

## What to build

Define typed approval values in `internal/agent` and migrate `item/commandExecution/requestApproval` end-to-end through codex → agent → TUI. This is the tracer slice that fixes the typed-approval shape; later slices follow the pattern for the remaining approval kinds.

Concretely:

- Add a typed value in `internal/agent` for command exec approvals, plus a `Decision` type (accept / decline / cancel / acceptForSession or whatever shape we settle on).
- Add a typed `Approve` API on `agent.Provider` (one method per kind, or a single typed-value method — decide as part of this slice). The existing `RPCResponder` may stay as a fallback for not-yet-migrated kinds.
- In `internal/codex/server_request.go`, translate the JSON-RPC `item/commandExecution/requestApproval` into the typed value when emitting; translate the typed decision back into a JSON-RPC result on respond.
- In `internal/tui/app.go` and `internal/tui/approval_reply.go`, switch on the typed value for command exec approvals; leave the existing string-dispatch path in place for the other kinds (those migrate in 02–04).

This is HITL — the API shape is the decision. Open the conversation with one or two alternatives (interface method per kind vs. single typed-value method) before writing.

## Acceptance criteria

- [ ] `internal/agent` exposes a typed value for command exec approvals and a typed decision API
- [ ] Codex translates command exec approval requests to the typed value and decisions back to JSON-RPC results
- [ ] TUI handles command exec approvals through the typed surface; no `Payload["approvalKind"]` read for this kind
- [ ] Other approval kinds keep working unchanged through the existing string-dispatch path
- [ ] A test exercises the typed path end-to-end (fake provider emits typed approval, TUI dispatches, provider receives typed decision)

## Blocked by

None - can start immediately
