Status: needs-triage

# Remove untyped approval dispatch in TUI

## What to build

Final cleanup once all approval kinds flow through the typed surface:

- Delete `internal/tui/approval_reply.go` and its tests, or reduce it to whatever residual helpers remain.
- Remove `Payload["approvalKind"]` reads in `internal/tui/app.go`.
- Remove the `approval_payload*` constants from `internal/codex/server_request.go` if they no longer have callers.
- Decide whether `agent.RPCResponder` can be removed entirely, or whether some non-approval RPC still uses it.

## Acceptance criteria

- [ ] `Payload["approvalKind"]` is not referenced anywhere outside the codex package's internal translation
- [ ] No string-based approval dispatch remains in the TUI
- [ ] `agent.RPCResponder` is either removed or documented as the fallback for a remaining non-approval RPC kind
- [ ] Tests still pass; coverage for approvals exists at the typed-surface level

## Blocked by

- Issue 02 (file change / apply patch migrated)
- Issue 03 (permissions migrated)
- Issue 04 (remaining kinds migrated)
