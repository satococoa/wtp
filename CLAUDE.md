# wtp (Worktree Plus) - Development Notes

This document contains the key design decisions and development guidance for wtp (formerly git-wtp).

## Recent Changes (2025-01-23)

### Major Simplification: wtp add Command Interface

The `wtp add` command has been significantly simplified to provide a cleaner, more intuitive interface:

**Removed Features:**
- ❌ `--detach` flag - Detached HEAD functionality completely removed
- ❌ `--track` flag - Remote branch tracking now automatic
- ❌ `--force` flag - Dangerous operations no longer simplified
- ❌ `--cd`/`--no-cd` flags - Directory change options removed
- ❌ `wtp add <worktree-name> <commit-ish>` pattern - Ambiguous syntax removed

**New Simplified Interface:**
- ✅ `wtp add <existing-branch>` - Create worktree from existing branch
- ✅ `wtp add -b <new-branch>` - Create new branch and worktree
- ✅ `wtp add -b <new-branch> [<commit>]` - Create new branch from specific commit

**Benefits:**
- **Reduced Complexity:** Eliminated 100+ lines of complex auto-detection logic
- **Improved UX:** Clear, git-checkout-like interface with -b flag pattern
- **Better Maintainability:** Fewer test cases, simpler code paths
- **Consistent Behavior:** Predictable outcomes for all use cases

## Project Background

This project was born from a conversation about improving Git's worktree functionality. The main pain points identified were:

1. Manual setup required after creating worktrees
2. No automatic branch tracking from remotes
3. Lack of project-specific initialization hooks
4. Cumbersome command syntax for common operations

## Worktree Naming Convention

**wtp** uses a consistent naming convention for worktrees across all commands and interfaces:

### Naming Rules

1. **Main worktree**: Always displayed as `@`
2. **Non-main worktrees**: Displayed as relative path from `base_dir`
   - Example: If `base_dir` is `.worktrees` and worktree is at `/repo/.worktrees/feat/hogehoge`, the worktree name is `feat/hogehoge`

### Usage Examples

```bash
# Completion and error messages show consistent names
wtp remove feat/hogehoge    # Not "hogehoge" 
wtp cd feat/hogehoge        # Not "hogehoge"

# List command shows the same names
wtp list
# PATH                     BRANCH        HEAD
# ----                     ------        ----
# @ (main worktree)        main          043130cc
# feat/hogehoge*           feat/hogehoge 043130cc
# fix/bug-123              fix/bug-123   def456bb
```

### Implementation

The `getWorktreeNameFromPath()` function in `cmd/wtp/completion.go` implements this logic:

- Takes worktree path, config, main repo path, and isMain flag
- Returns `@` for main worktree
- Returns relative path from `base_dir` for non-main worktrees
- Falls back to directory name if path calculation fails

This function is used consistently across:
- Shell completion (`wtp remove`, `wtp cd`)
- Error messages (worktree not found)
- Command parsing and resolution

## Recent Changes

### Git Worktree Compatible Syntax

The `wtp add` command now follows `git worktree add` syntax exactly for maximum compatibility:

- **Two-argument syntax**: `wtp add <worktree-name> <commit-ish>` automatically creates detached HEAD
- **Git compatibility**: Matches `git worktree add <path> <commit-ish>` behavior exactly
- **Error on single commit-ish**: `wtp add HEAD~1` shows helpful error requiring worktree name
- **Explicit control**: `--detach` flag remains for explicit detached HEAD

**Examples:**
```bash
wtp add experiment HEAD~1         # Auto-detached HEAD (git compatible)
wtp add test abc1234              # Auto-detached HEAD with commit hash
wtp add --detach lab HEAD~2       # Explicit detach flag
wtp add HEAD~1                    # ERROR: requires worktree name
```

This change ensures perfect compatibility with `git worktree` syntax while maintaining all existing wtp features.

## Core Design Decisions

### Why Go Instead of Shell Script?

We chose Go for several reasons:

1. **Cross-platform compatibility** - Native Windows support without WSL
2. **Better error handling** - Type safety and structured error messages
3. **Unified shell completion** - Single source for all shells
4. **Easier testing** - Unit tests vs shell script testing
5. **Single binary distribution** - Easy installation via brew, scoop, etc.

### Configuration Format

We chose YAML for configuration because:
- Human-readable and writable
- Supports complex structures (arrays, maps)
- Well-supported in Go ecosystem
- Familiar to developers (CI/CD configs)

### Hook System Design

Post-create hooks support:
- File copying (for .env files, etc.)
- Command execution

This covers 90% of use cases without over-engineering.

## Development Workflow

Before committing any changes, always run:

```bash
go tool task dev  # Runs fmt, lint, and test
```

**Checklist for new features:**

1. Update README.md documentation and examples
2. Update command help text
3. Run `go tool task dev` and fix all issues
4. Update documentation for architectural changes

**Important**: Never commit code that fails lint or tests.

### Quick Testing During Development

For rapid testing during development, you can use `go run` instead of building binaries:

```bash
# Instead of building and running:
# go build -o wtp ./cmd/wtp && ./wtp list

# You can directly use:
go run ./cmd/wtp list

# Test from within a worktree or repository:
cd test-repo
go run ../cmd/wtp list
go run ../cmd/wtp add feature/new-feature

# Test shell integration commands:
WTP_SHELL_INTEGRATION=1 go run ../cmd/wtp cd @
```

This approach is faster for iterative development and testing.

## TDD Approach for Bug Fixes

When fixing bugs, always follow the Test-Driven Development (TDD) cycle:

### 1. 🔴 RED - Write a Failing Test First

Before fixing any bug, write a test that reproduces the problem:

```go
// Example: Hook output streaming bug
func TestExecutePostCreateHooks_StreamingOutput(t *testing.T) {
    // Create a writer that tracks when writes occur
    sw := &streamingWriter{}
    
    // Execute hooks that produce output over time
    err := executor.ExecutePostCreateHooks(sw, worktreeDir)
    
    // Verify output was streamed (multiple writes), not buffered
    if len(sw.writes) < expectedWrites {
        t.Error("Output should be streamed in real-time")
    }
}
```

### 2. 🟢 GREEN - Write Minimal Code to Pass

Implement just enough to make the test pass:

```go
// Fix: Direct output to writer instead of buffering
func (e *Executor) ExecutePostCreateHooks(w io.Writer, path string) error {
    // Stream output directly to writer
    cmd.Stdout = w
    cmd.Stderr = w
    return cmd.Run()
}
```

### 3. ♻️ REFACTOR - Improve the Code

Once tests pass, refactor for clarity and maintainability:
- Remove duplicate code
- Simplify APIs
- Improve naming
- Add documentation

### Benefits of TDD for Bug Fixes

1. **Regression Prevention**: Test ensures bug doesn't reappear
2. **Clear Understanding**: Test documents expected behavior
3. **Focused Fix**: Only write code needed to pass test
4. **Safe Refactoring**: Tests protect against breaking changes
5. **Better Design**: TDD often leads to cleaner APIs

### Guidelines

- **Never skip the RED phase**: Ensure test fails before fixing
- **Keep tests focused**: One test per bug/behavior
- **Test behavior, not implementation**: Focus on what, not how
- **Use meaningful assertions**: Clearly express expectations
- **Consider edge cases**: Test boundary conditions

## Testing Strategy

### Test Levels

1. **Unit Tests (70%)**: Fast, mocked, business logic testing
2. **E2E Tests (30%)**: Real git operations, user workflows

For detailed testing guidelines, see [docs/testing-guidelines.md](docs/testing-guidelines.md).

### Guidelines for New Commands

1. **Start with Tests**: Write command executor tests first (TDD)
2. **Use Command Builders**: Leverage existing or create new builders
3. **Mock in Unit Tests**: Never execute real git in unit tests
4. **Document in E2E**: Add realistic user scenarios to E2E tests

## Architecture

For technical architecture details, see [docs/architecture.md](docs/architecture.md).

## Development History

For detailed development history and major changes, see [docs/development-history.md](docs/development-history.md).

## Future Considerations

### Potential Features

- Template system for different project types
- Integration with project generators
- Worktree status dashboard
- Parallel command execution across worktrees

### Performance Optimizations

- Cache git command outputs
- Parallel hook execution
- Lazy loading of worktree information

---

This document serves as a living record of the project's key decisions and development practices. Update as new patterns emerge.