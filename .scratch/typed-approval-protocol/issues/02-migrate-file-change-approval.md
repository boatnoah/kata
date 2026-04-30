Status: needs-triage

# Migrate `FileChangeApproval` and `applyPatchApproval` to typed

## What to build

Apply the typed approval pattern from issue 01 to `item/fileChange/requestApproval` and `applyPatchApproval`. The latter carries `parsedFileChanges` (a slice of `agent.FileChange`) which already has a typed shape — fold that into the typed approval value rather than carrying it through `Payload`.

## Acceptance criteria

- [ ] `internal/agent` exposes typed values for file change and apply patch approvals
- [ ] Codex translates both kinds to typed; existing parsed-file-change behavior preserved
- [ ] TUI handles both kinds via typed dispatch; diff rendering still works
- [ ] `Payload["approvalKind"]` no longer read for these kinds in `app.go`
- [ ] Tests for file change and apply patch approvals exercise the typed path

## Blocked by

- Issue 01 (typed approval shape locked in)
