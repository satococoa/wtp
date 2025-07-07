# wtp (Worktree Plus) - Development Notes

This document contains the design decisions, implementation details, and
discussions that led to the creation of wtp (formerly git-wtp).

## Background

This project was born from a conversation about improving Git's worktree
functionality. The main pain points identified were:

1. Manual setup required after creating worktrees
2. No automatic branch tracking from remotes
3. Lack of project-specific initialization hooks
4. Cumbersome command syntax for common operations

## Key Design Decisions

### Why Go Instead of Shell Script?

We initially considered shell scripting but chose Go for several reasons:

1. **Cross-platform compatibility** - Native Windows support without WSL
2. **Better error handling** - Type safety and structured error messages
3. **Unified shell completion** - Single source for all shells (bash, zsh, fish,
   PowerShell)
4. **Easier testing** - Unit tests vs shell script testing
5. **Single binary distribution** - Easy installation via brew, scoop, etc.

### Branch Resolution Strategy

Following Git's own behavior:

- First check local branches
- If not found, search remote branches
- If multiple remotes have the same branch, error and ask for explicit remote
- This matches `git checkout` and `git switch` behavior

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

## Implementation Notes

- **Path Handling**: Branch names with slashes become directory structure (e.g.,
  `feature/auth` → `../worktrees/feature/auth/`)
- **Shell Integration**: The `cd` command requires shell functions since child
  processes can't change parent's directory

## Technical Architecture

```
cmd/wtp/
├── main.go         # Entry point
├── add.go          # Add command
├── remove.go       # Remove command
├── list.go         # List command
├── init.go         # Init command
├── cd.go           # Change directory command
└── completion.go   # Shell completion & integration

internal/
├── git/            # Git operations
├── config/         # YAML configuration
└── hooks/          # Hook execution
```

## Development Workflow

Before committing any changes, always run:

```bash
make dev  # Runs fmt, lint, and test
```

**Checklist for new features:**

1. Update README.md documentation and examples
2. Update command help text
3. Run `make dev` and fix all issues
4. Update CLAUDE.md for architectural changes

**Important**: Never commit code that fails lint or tests.

## Testing Strategy

- Unit tests for core functionality
- E2E tests with real git repositories
- Manual testing of shell completions across bash/zsh/fish

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

## Development Tools

### Go 1.24 Tool Directive

This project uses Go 1.24's new tool directive for development tools:

```
tool (
    github.com/golangci/golangci-lint/cmd/golangci-lint
    golang.org/x/tools/cmd/goimports
)
```

**Important**: Always use `make` commands instead of calling tools directly:

- ✅ `make lint` (calls `go tool golangci-lint run`)
- ✅ `make fmt` (calls `go tool goimports -w .`)
- ❌ `golangci-lint run` (may use different version)
- ❌ `goimports -w .` (may use different version)

This ensures all team members use the same tool versions defined in go.mod.

### Project Renaming

- **Command**: `git-wtp` → `wtp` (easier typing, follows patterns like
  gh/ghq/tig)
- **Config**: `.git-worktree-plus.yml` → `.wtp.yml` (consistency, shorter)

## Major Design Changes

### Shell Integration (cd command) Implementation

**Background**: v0.3.0 milestone included implementing the `wtp cd` command to
quickly change directories to worktrees.

**Implementation Details**:

1. **Two-Part Architecture**:
   - **Go Command**: `wtp cd <worktree>` finds the worktree path and outputs it
   - **Shell Function**: Wraps the Go command and performs the actual `cd`

2. **Shell Integration Flow**:
   ```bash
   # User types:
   wtp cd feature/auth

   # Shell function intercepts, runs:
   WTP_SHELL_INTEGRATION=1 wtp cd feature/auth

   # Go command returns path:
   /path/to/worktrees/feature/auth

   # Shell function performs:
   cd /path/to/worktrees/feature/auth
   ```

3. **Key Design Decisions**:
   - **Environment Variable Check**: `WTP_SHELL_INTEGRATION=1` prevents
     accidental direct usage
   - **Shell Function Wrapper**: Required because child processes can't change
     parent's directory
   - **Unified Setup Command**: `wtp completion <shell>` generates both
     completion and cd functionality
   - **Cross-Shell Support**: Bash, Zsh, and Fish implementations

4. **User Experience**:
   ```bash
   # Enable shell integration
   eval "$(wtp completion zsh)"  # or bash/fish

   # Use cd command with tab completion
   wtp cd <TAB>           # Shows available worktrees
   wtp cd feature/auth    # Changes to worktree directory
   ```

5. **Benefits**:
   - **Fast Navigation**: No need to remember worktree paths
   - **Tab Completion**: Discover available worktrees easily
   - **Consistent Interface**: Same command across all shells
   - **Backward Compatible**: Doesn't break existing functionality

**Files Added/Modified**:

- `cmd/wtp/cd.go`: Core cd command implementation
- `cmd/wtp/cd_test.go`: Tests for cd functionality
- `cmd/wtp/completion.go`: Extended with shell function generation
- `cmd/wtp/main.go`: Added cd command registration
- `README.md`: Updated documentation and feature checklist

**Testing**: All tests pass, shell integration tested manually across
bash/zsh/fish

### Simplify add Command by Removing Rarely Used Options

**Background**: The add command was supporting all git worktree options, making
it complex to maintain and understand. Following the 80/20 principle, we
simplified it by keeping only the commonly used options.

**Options Removed**:

- `--checkout`: Always enabled by default in git worktree, so redundant
- `--lock` and `--reason`: Worktree locking is a very specialized use case
- `--orphan`: Creating orphan branches is extremely rare

**Options Kept**:

- `-b/--branch`: Creating new branches (frequent use case)
- `--track`: Remote branch tracking (core wtp functionality)
- `--path`: Explicit path specification (wtp convenience feature)
- `-f/--force`: Force checkout when needed
- `--detach`: Detached HEAD for investigating specific commits

**Benefits**:

- **Simpler Code**: Reduced complexity in `buildGitWorktreeArgs` and related
  functions
- **Easier Maintenance**: Less edge cases to handle
- **Clear Focus**: wtp focuses on the 80% common use cases
- **Fallback Available**: Users can still use `git worktree` directly for
  advanced options

**Files Modified**:

- `cmd/wtp/add.go`: Removed unused flags and simplified argument building logic
- `CLAUDE.md`: This documentation

**Testing**: All existing tests pass; no tests were using the removed options

### Explicit Path Flag Implementation

**Background**: User feedback identified ambiguity issue with automatic path
detection - `foobar/foo` could be interpreted as either a path or branch name,
causing confusion.

**Solution**: Replaced automatic path detection with explicit `--path` flag for
unambiguous behavior.

**Changes Made**:

1. **Added --path Flag**:
   ```bash
   # Before (ambiguous)
   wtp add foobar/foo          # Is this a path or branch?

   # After (explicit)
   wtp add --path foobar/foo feature/auth  # Clear: foobar/foo is path
   wtp add foobar/foo                       # Clear: foobar/foo is branch
   ```

2. **Removed isPath() Function**:
   - Eliminated automatic path detection logic
   - No more heuristics based on path patterns

3. **Updated resolveWorktreePath**:
   - Simple flag-based logic: `--path` present = explicit path, otherwise
     auto-generate
   - More predictable and testable

4. **Benefits**:
   - **Clarity**: No ambiguity between paths and branch names
   - **Safety**: Users always get expected behavior
   - **Consistency**: Follows git worktree flag pattern
   - **Maintainability**: Simpler logic without heuristics

**Files Modified**:

- `cmd/wtp/main.go`: Added --path flag, removed isPath(), updated logic
- `cmd/wtp/main_test.go`: Updated tests for new behavior
- `README.md`: Updated examples to use --path flag
- `CLAUDE.md`: This documentation

**Testing**: All tests pass, explicit path logic verified

### TDD-Driven Command Architecture & Test Strategy Revolution

**Background**: Following Issue #3 (Test Quality Improvement) and user feedback
about testable command design, we implemented a revolutionary new command
architecture using TDD principles.

#### Evolution of Command Design

**Generation 1: Direct Git Execution**

```go
// Problems: Slow tests, environment dependent, hard to test error cases
repo, err := git.NewRepository(cwd)
err = repo.ExecuteGitCommand("worktree", "add", "--force", path, branch)
```

**Generation 2: GitExecutor Interface**

```go
// Improvement: Mockable, but still string-based command building
type GitExecutor interface {
    ExecuteGitCommand(args ...string) error
    ResolveBranch(branch string) (string, bool, error)
}

args := buildGitWorktreeArgs(cmd, path, branch)  // []string construction
err := gitExec.ExecuteGitCommand(args...)
```

**Generation 3: CommandExecutor with Type Safety**

```go
// TDD-designed: Type-safe builders, structured commands, testable
type Command struct {
    Name    string   // "git"
    Args    []string // ["worktree", "add", "--force", ...]
    WorkDir string   // optional
}

// Type-safe command builders
opts := command.GitWorktreeAddOptions{Force: true, Branch: "feature"}
cmd := command.GitWorktreeAdd(path, commitish, opts)
result, err := executor.Execute([]command.Command{cmd})
```

#### Key Benefits of New Architecture

1. **Type Safety**: Compile-time validation of command construction
2. **Testability**: Mock command execution without running git
3. **Composability**: Multiple commands in single execution
4. **Maintainability**: Centralized command building logic
5. **Extensibility**: Easy to add new git operations

#### Command Building Pattern

```go
// Structured options instead of boolean flags
type GitWorktreeAddOptions struct {
    Force  bool
    Detach bool
    Branch string
    Track  string
}

// Builder functions generate commands
func GitWorktreeAdd(path, commitish string, opts GitWorktreeAddOptions) Command {
    args := []string{"worktree", "add"}
    if opts.Force { args = append(args, "--force") }
    // ... other options
    return Command{Name: "git", Args: args}
}
```

#### Test Strategy: Test Pyramid

Following test pyramid principles, we eliminated the problematic "integration
test" middle layer:

**Before (Problematic)**:

```
    E2E Tests
 Integration Tests  ← Middle layer, often redundant
Unit Tests (Large)
```

**After (Clean)**:

```
  E2E Tests (Few, Real Git)
Unit Tests (Many, Fast Mocks)
```

**Test Structure**:

- `cmd/wtp/add_test.go`: Pure unit tests with mocks (no git execution)
- `test/e2e/worktree_test.go`: E2E tests with real git operations

**Unit Test Example**:

```go
func TestAddCommand_WithCommandExecutor(t *testing.T) {
    mockExec := &mockCommandExecutor{}

    err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

    // Verify correct command was built (no git execution)
    assert.Equal(t, []command.Command{{
        Name: "git",
        Args: []string{"worktree", "add", "--force", "/path", "branch"},
    }}, mockExec.executedCommands)
}
```

**Benefits Achieved**:

- **Fast TDD Cycle**: Unit tests run in milliseconds
- **Error Testing**: Easy to test failure scenarios with mocks
- **Environment Independence**: Tests don't require git installation
- **Clear Separation**: Unit tests verify wtp logic, E2E tests verify git
  integration

#### Implementation Status

- ✅ **command package**: Complete with builders and executor
- ✅ **add command**: Dual implementation (legacy + new architecture)
- ⏳ **Other commands**: Can be migrated to new architecture incrementally

#### Guidelines for New Commands

1. **Start with Tests**: Write command executor tests first (TDD)
2. **Use Command Builders**: Leverage existing or create new builders
3. **Mock in Unit Tests**: Never execute real git in unit tests
4. **Document in E2E**: Add realistic user scenarios to E2E tests

**Files Added/Modified**:

- `internal/command/`: New package with executor, builders, and shell
  abstraction
- `cmd/wtp/add.go`: Added new CommandExecutor-based implementation
- `cmd/wtp/add_test.go`: Consolidated, pure unit tests only
- `test/e2e/worktree_test.go`: Existing E2E tests (unchanged)

**Testing**: All tests pass, coverage maintained at 83%+

This architecture establishes a foundation for consistent, testable command
handling across all wtp commands.

---

This document serves as a living record of the project's development. Update as
new decisions are made or challenges are encountered.
