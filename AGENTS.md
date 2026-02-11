# AGENTS

`wtp` is a Go CLI that simplifies `git worktree` workflows for creating, listing, moving, and removing worktrees.

## Essentials
- Toolchain: Go 1.24
- Package manager: Go modules (`go`)
- Project task runner: `go tool task`
- Non-standard project commands:
  - `go tool task build`
  - `go tool task test`
  - `go tool task lint`
  - `go tool task fmt`
  - `go tool task test-e2e`
  - `go tool task dev`

## Docs
- [Dev commands](docs/agents/dev-commands.md)
- [Code and tests](docs/agents/code-and-tests.md)
