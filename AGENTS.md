# Repository Guidelines

## Project Structure & Modules
- Root module: `github.com/satococoa/wtp` (Go 1.24).
- CLI entrypoint: `cmd/wtp`.
- Internal packages: `internal/{git,config,hooks,command,errors,io}`.
- Tests: unit tests alongside packages (`*_test.go`), end-to-end tests in `test/e2e`.
- Tooling/config: `.golangci.yml`, `.goreleaser.yml`, `Taskfile.yml`, `.wtp.yml` (project hooks), `docs/`.

## Build, Test, and Dev Commands
- Build: `go tool task build` (or `task build`) → outputs `./wtp`.
- Install: `go tool task install` → installs to `GOBIN`/`GOPATH/bin`.
- Test (unit + race + coverage): `go tool task test` → writes `coverage.out`.
- Lint: `go tool task lint` (golangci-lint).
- Format: `go tool task fmt` (gofmt + goimports).
- E2E tests: `go tool task test-e2e` (uses built binary; override with `WTP_E2E_BINARY=/abs/path/wtp`).
- Direct build (no Task): `go build -o wtp ./cmd/wtp`.
- Dev cycle: `go tool task dev` (runs fmt, lint, test).

## Coding Style & Naming
- Follow standard Go style (tabs, gofmt) and idioms; package names are short and lowercase.
- Imports: keep groups tidy; `goimports` organizes, local prefix follows module path.
- Linting: adhere to rules in `.golangci.yml` (vet, staticcheck, gosec, mnd, lll=120, etc.).
- Errors: wrap with context; avoid ignoring errors.
- Files/identifiers: `snake_case.go`, exported items documented when non-trivial.

## Testing Guidelines
- Frameworks: standard library `testing` with `testify` for assertions.
- Unit tests live next to code; E2E scenarios under `test/e2e` (see `framework/`).
- Naming: `TestXxx` functions; prefer table-driven tests.
- Run with coverage: `go tool task test`; view coverage: `go tool cover -func=coverage.out`.

## Commit & PR Guidelines
- Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `style:` (see `git log`).
- Branch naming: `feature/...`, `fix/...`, `hotfix/...` aligns with worktree paths.
- Before opening a PR: format, lint, tests green, update docs if needed.
- PR description: what/why/how, testing notes, related issues (e.g., `Closes #123`).
- Add CLI output or screenshots when UX changes.

## Security & Configuration Tips
- Project hooks are defined in `.wtp.yml`. Keep commands deterministic and safe; avoid destructive steps by default.
- Do not commit secrets; use example files (e.g., `.env.example`) and copy in hooks.

## Design & Workflow Reference
- See `CLAUDE.md` for design decisions (worktree naming, hooks), the TDD workflow, and quick dev tips (e.g., `go run ./cmd/wtp ...`).
