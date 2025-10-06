# Repository Guidelines

## Document Purpose
- Single source of truth for AI assistants contributing to `wtp`.
- Consolidates design decisions, workflow expectations, and coding standards previously split across `AGENTS.md` and `CLAUDE.md`.

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

## Commit & PR Guidelines
- Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `style:` (see `git log`).
- Branch naming: `feature/...`, `fix/...`, `hotfix/...` aligns with worktree paths.
- Before opening a PR: format, lint, tests green, update docs if needed.
- PR description: what/why/how, testing notes, related issues (e.g., `Closes #123`).
- Add CLI output or screenshots when UX changes.

## Security & Configuration Tips
- Project hooks are defined in `.wtp.yml`. Keep commands deterministic and safe; avoid destructive steps by default.
- Do not commit secrets; use example files (e.g., `.env.example`) and copy in hooks.

## TDD Workflow
### Development Cycle Expectations
- Run `go tool task dev` before committing; it formats code, runs lint, and executes unit tests.
- Keep the repository in a buildable state; never commit failing tests or lint errors.
- Update README/CLI help when user-facing behavior changes.

### RED → GREEN → REFACTOR Example
1. **RED**: write an end-to-end or unit test that captures the desired behavior. Example: add a failing E2E test for `wtp prune` that asserts a detached worktree is removed.
2. **GREEN**: implement the minimum code to satisfy that test. Keep the change small and focused on the behavior under test.
3. **REFACTOR**: clean up duplication, rename helpers, and harden edge cases while keeping tests green.

### Quick Testing Tips
- Use `go run ./cmd/wtp <args>` for rapid feedback instead of building binaries.
- Run commands from inside a worktree to mimic real usage (e.g., `go run ../cmd/wtp add feature/new-feature`).
- Toggle shell integration paths with `WTP_SHELL_INTEGRATION=1` when testing cd behavior.

### Testing Strategy
- Unit tests target 70% of coverage: fast feedback, mocked git interactions, table-driven cases.
- E2E tests cover 30%: real git workflows in `test/e2e` exercising command plumbing.
- See `docs/testing-guidelines.md` for extended advice on fixtures, naming, and reliability.

### Guidelines for New Commands
- Start with tests: design executor behavior through failing tests.
- Prefer existing command builders; add new ones when patterns diverge.
- Mock git operations in unit tests; rely on E2E suites for real git interactions.
- Document new workflows with realistic scenarios in `test/e2e`.

## Core Design Decisions
### Why Go Instead of Shell Script?
1. **Cross-platform compatibility**: native support for Windows without WSL.
2. **Better error handling**: types and structured errors instead of brittle shell parsing.
3. **Unified shell completion**: one implementation for all supported shells.
4. **Easier testing**: standard Go testing beats shell-script harnesses.
5. **Single binary distribution**: simplifies packaging for brew, scoop, etc.

### Configuration Format (Why YAML?)
- Human-readable and writable.
- Expressive enough for arrays, maps, and nested structures used by hooks.
- Well-supported in the Go ecosystem.
- Familiar syntax for developers accustomed to CI/CD tooling.

### Hook System Design
- Post-create hooks handle both file copying (e.g., `.env`) and command execution.
- Captures the common 90% of project bootstrap needs without over-engineering.
- Hooks are declared in `.wtp.yml`, keeping repository-specific automation explicit.

### Worktree Naming Convention
1. **Main worktree**: always rendered as `@`.
2. **Non-main worktrees**: display the path relative to `base_dir` (e.g., `.worktrees/feat/hoge` → `feat/hoge`).
3. **Fallback**: directory name is used if path resolution fails so that UX remains consistent.

**Implementation Notes**
- `getWorktreeNameFromPath()` in `cmd/wtp/completion.go` resolves display names.
- Shared across shell completion, user-facing errors, and command parsing.
- Ensures commands such as `wtp remove feat/foo` and `wtp cd feat/foo` share the same identifier.

## Shell Integration Architecture (v1.2.0)
### "Less is More" Approach
- Completion and shell integration are separated for clarity and maintainability.
- Homebrew and direct installs can compose the pieces they need.

### Command Structure
- **`wtp cd <worktree>`**: outputs the absolute worktree path with no side effects.
- **`wtp completion <shell>`**: generates pure completion scripts via `urfave/cli`.
- **`wtp hook <shell>`**: emits shell functions that intercept `wtp cd` for seamless navigation (bash, zsh, fish).
- **`wtp shell-init <shell>`**: convenience command that combines completion and hook output.

### User Experience Matrix
| Installation Method | Tab Completion | cd Functionality |
|---------------------|----------------|------------------|
| Homebrew            | Automatic      | `eval "$(wtp hook <shell>)"` |
| `go install`        | `eval "$(wtp completion <shell>)"` | `eval "$(wtp hook <shell>)"` |

### Migration Notes
- `wtp cd` no longer depends on `WTP_SHELL_INTEGRATION=1`; the command itself only prints paths.
- Existing completion scripts from v1.1.x still work but omit cd helpers unless users add hooks.
- Future Homebrew enhancements may lazy-load integration by wrapping `wtp shell-init` on first use.

## Additional References
- Architecture overview: `docs/architecture.md`.
- Testing deep dive: `docs/testing-guidelines.md`.
