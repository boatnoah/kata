Status: needs-triage

# Migrate type-tick state onto `AIStream`

## What to build

Move the type-tick state — `aiTypes`, `aiIndexes` — from `App` onto `AIStream`. The progressive-reveal rendering math becomes a method on `AIStream`.

## Acceptance criteria

- [ ] Type-tick state lives on `AIStream`
- [ ] Progressive-reveal output is byte-identical to current behavior
- [ ] Unit tests cover the reveal math on `AIStream` directly

## Blocked by

- Issue 01 (`AIStream` type introduced)
