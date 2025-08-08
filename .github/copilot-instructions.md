# GitHub Copilot – Repository Instructions

Use this guidance when answering questions or generating code for this repo.

## Context
- Project: wtp (Worktree Plus) — Git worktree management CLI in Go.
- Entry: `cmd/wtp`, internals in `internal/{git,config,hooks,command,errors,io}`.
- Docs: see `AGENTS.md` (contributor quick guide) and `CLAUDE.md` (design + TDD workflow).

## Build, Test, Lint
- Build: `go tool task build` (or `go build -o wtp ./cmd/wtp`).
- Dev cycle: `go tool task dev` (fmt + lint + test).
- Tests: `go tool task test` (race, coverage to `coverage.out`). E2E: `go tool task test-e2e`.
- Lint: `go tool task lint` (rules in `.golangci.yml`). Format with `go tool task fmt`.

## Coding Style
- Go idioms; keep functions focused and small. Use `gofmt`/`goimports`.
- Follow lint rules: vet, staticcheck, gosec, `lll=120`, avoid ignored errors.
- Packaging: short, lowercase package names; test files `*_test.go` with table-driven tests.

## Domain Rules (Do/Don’t)
- Do preserve worktree naming rules: main is `@`; others are paths relative to `base_dir` (see `CLAUDE.md`).
- Do keep hooks safe and deterministic; stream output; filter env like `WTP_SHELL_INTEGRATION` where applicable.
- Do keep CLI UX consistent with README examples and autocompletion behavior.
- Don’t introduce breaking changes to path/name resolution without updating docs and tests.

## Answering Guidelines
- Prefer TDD: write/describe failing tests first, then minimal code, then refactor (see `CLAUDE.md`).
- When proposing changes, include exact commands to build/test and mention affected files/paths.
- Reference `AGENTS.md` for contributor workflow; deep design questions should cite `CLAUDE.md` sections.

## Security & Config
- Never commit secrets; prefer examples (e.g., `.env.example`) and copying in hooks.
- `.wtp.yml` defines project hooks executed on worktree creation; keep defaults safe.

