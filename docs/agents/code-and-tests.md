# Code and Tests

## Go Conventions
- Package names are short and lowercase.
- Keep imports tidy; rely on `goimports` with the module local prefix.
- Follow lint rules in `.golangci.yml`.
- Wrap errors with context; do not ignore errors.
- Use `snake_case.go` filenames.

## TDD Flow
1. RED: add a failing unit or E2E test for the behavior.
2. GREEN: implement the minimum change to pass.
3. REFACTOR: improve touched code while tests stay green.

## Testing Expectations
- Prefer unit tests for fast feedback (table-driven, mocked git interactions).
- Use E2E tests in `test/e2e` for real git workflow validation.
- Unit test naming: `TestFunction_Condition`.
- E2E naming: `TestUserAction_WhenCondition_ShouldOutcome`.
- Update README or CLI help when user-facing behavior changes.
