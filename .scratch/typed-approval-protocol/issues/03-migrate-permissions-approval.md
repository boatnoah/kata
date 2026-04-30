Status: needs-triage

# Migrate `PermissionsApproval` to typed

## What to build

Apply the typed approval pattern to `item/permissions/requestApproval`. The decision shape includes a `permissions` map and a `scope` ("turn" or "session") — design typed analogues rather than reusing `map[string]any` and a free-form string.

## Acceptance criteria

- [ ] `internal/agent` exposes a typed value for permissions approvals with a typed scope and permissions payload
- [ ] Codex translates the request and decision in both directions
- [ ] TUI handles permissions approvals via typed dispatch
- [ ] Tests cover the typed path

## Blocked by

- Issue 01 (typed approval shape locked in)
