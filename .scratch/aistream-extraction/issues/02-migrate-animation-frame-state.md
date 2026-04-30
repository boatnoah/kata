Status: needs-triage

# Migrate animation frame state onto `AIStream`

## What to build

Move the animation frame state — `aiVerbIdx`, `aiWaitFrames`, `aiTicking` — from `App` onto `AIStream`. `handleAITick` and `scheduleAITick` should largely become methods on `AIStream` (or a small lifecycle struct alongside it).

## Acceptance criteria

- [ ] Animation frame state lives on `AIStream`, not as parallel maps on `App`
- [ ] Spinner / verb cycling behavior unchanged
- [ ] Unit tests exercise tick advancement on `AIStream` directly
- [ ] No regressions in `app_test.go` event-dispatch tests

## Blocked by

- Issue 01 (`AIStream` type introduced)
