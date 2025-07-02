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

- **Path Handling**: Branch names with slashes become directory structure (e.g., `feature/auth` → `../worktrees/feature/auth/`)
- **Shell Integration**: The `cd` command requires shell functions since child processes can't change parent's directory

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

### Project Renaming (2024-12)

- **Command**: `git-wtp` → `wtp` (easier typing, follows patterns like gh/ghq/tig)
- **Config**: `.git-worktree-plus.yml` → `.wtp.yml` (consistency, shorter)


## Major Design Changes

### 2025-01: Shell Integration (cd command) Implementation

**Background**: v0.3.0 milestone included implementing the `wtp cd` command to quickly change directories to worktrees.

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
   - **Environment Variable Check**: `WTP_SHELL_INTEGRATION=1` prevents accidental direct usage
   - **Shell Function Wrapper**: Required because child processes can't change parent's directory
   - **Unified Setup Command**: `wtp shell-init --cd` generates both completion and cd functionality
   - **Cross-Shell Support**: Bash, Zsh, and Fish implementations

4. **User Experience**:
   ```bash
   # Enable shell integration
   eval "$(wtp shell-init --cd)"
   
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

**Testing**: All tests pass, shell integration tested manually across bash/zsh/fish

### 2025-01: Simplify add Command by Removing Rarely Used Options

**Background**: The add command was supporting all git worktree options, making it complex to maintain and understand. Following the 80/20 principle, we simplified it by keeping only the commonly used options.

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
- **Simpler Code**: Reduced complexity in `buildGitWorktreeArgs` and related functions
- **Easier Maintenance**: Less edge cases to handle
- **Clear Focus**: wtp focuses on the 80% common use cases
- **Fallback Available**: Users can still use `git worktree` directly for advanced options

**Files Modified**:
- `cmd/wtp/add.go`: Removed unused flags and simplified argument building logic
- `CLAUDE.md`: This documentation

**Testing**: All existing tests pass; no tests were using the removed options

### 2024-12: Explicit Path Flag Implementation

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

---

This document serves as a living record of the project's development. Update as
new decisions are made or challenges are encountered.
