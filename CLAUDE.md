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

## Implementation Challenges

### 1. Path Handling

- Cross-platform path separators
- Relative vs absolute paths
- Branch names with slashes preserved as directory structure (e.g., feature/auth
  → feature/auth/)

### 2. Shell Integration

The `cd` command requires shell functions since child processes can't change
parent's directory.

## Technical Architecture

```
cmd/
├── wtp/
│   └── main.go          # Entry point
├── add.go               # Add command
├── remove.go            # Remove command
├── list.go              # List command
├── init.go              # Init command
├── cd.go                # Change directory command
└── completion.go        # Shell completion & integration

internal/
├── git/
│   ├── repository.go    # Git operations
│   └── worktree.go      # Worktree management
├── config/
│   └── config.go        # YAML configuration
└── hooks/
    └── executor.go      # Hook execution
```

## Post-Implementation Checklist

When implementing new features, always remember to:

1. **Update Documentation**: After adding new features or flags, update both
   README.md and any relevant documentation
2. **Update Feature Checklists**: Mark completed features as done in README.md
   roadmap
3. **Add Usage Examples**: Include practical examples in the Quick Start section
4. **Update Help Text**: Ensure command help text reflects new options
5. **Format Code**: Run `make fmt` to ensure consistent code formatting
6. **Run Linter**: ALWAYS run `make lint` and fix ALL issues before committing
7. **Run Tests**: Always run `make test` to ensure nothing is broken
8. **Update CLAUDE.md**: Document any new design decisions or architectural
   changes

### Development Workflow

Before committing any changes, always run these commands in order:

```bash
# 1. Format code
make fmt

# 2. Check for lint issues
make lint

# 3. Run tests
make test

# Or run all checks at once:
make dev
```

**Important**: Never commit code that fails any of these checks. The `make dev` command runs all three checks and is the recommended way to verify your changes before committing.

## Testing Strategy

- Integration tests with real git repos
- Cross-platform CI/CD testing
- Manual testing of shell completions

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

## Lessons Learned

1. **Start simple** - MVP first, then iterate
2. **Follow existing conventions** - Git's behavior is well-understood
3. **Invest in completions early** - Greatly improves UX
4. **Design for extensibility** - Hook system allows user customization

## Original PRD Summary

The original Product Requirements Document specified:

- Simple worktree management commands
- Automatic environment setup
- Configuration via .wtp.yml
- Cross-platform support
- Shell integration

All core requirements are being met with room for future expansion.

## Code Snippets and Patterns

### Error Handling Pattern

```go
if err != nil {
    return fmt.Errorf("failed to create worktree: %w", err)
}
```

### Command Execution Pattern

```go
cmd := exec.Command("git", args...)
cmd.Dir = workDir
output, err := cmd.CombinedOutput()
```

### Configuration Loading Pattern

```go
func LoadConfig() (*Config, error) {
    // Check multiple locations
    // Validate version
    // Return with defaults
}
```

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

### Command Name Change (2024-12)

**From git-wtp to wtp**: The command was renamed from `git-wtp` to `wtp` for:

- Easier typing and better ergonomics
- Following patterns like `ghq`, `gh`, `tig` (successful Git tools)
- Simpler shell completion without git subcommand complexity
- Users can still use `git wtp` via git alias if preferred

### Configuration File Name Change (2024-12)

**From .git-worktree-plus.yml to .wtp.yml**: The configuration file was renamed
for:

- Consistency with the new command name
- Shorter and easier to type
- Following common patterns (.gitignore, .editorconfig, etc.)

### Required Commands

When implementing new features or fixing bugs:

1. **Linting First**: `make lint` - MUST pass with no errors before committing
2. **Testing**: `make test` - Ensure all tests pass
3. **Development**: `make dev` - Full check (runs lint, test, build)
4. **Formatting**: `make fmt` - Auto-format code with goimports

Common lint errors include:

- Unused variables/parameters (use `_` for intentionally unused)
- Magic numbers (define constants instead)
- Cyclomatic complexity (refactor complex functions)
- Parameter type combinations (e.g., `func(a, b string)` instead of
  `func(a string, b string)`)

## Major Design Changes

### 2024-12: Transparent Wrapper & Hybrid Approach Implementation

**Key Changes Implemented**:

1. **Path Resolution Logic**:
   - Added `isPath()` function to distinguish paths from branch names
   - Supports absolute paths (`/custom/path`), relative paths (`./path`,
     `../path`), and Windows paths (`C:\path`)
   - Everything else treated as branch name for automatic path generation

2. **Hybrid Command Syntax**:
   ```bash
   # Simple: automatic path generation
   wtp add feature/auth  # → ../worktrees/feature/auth

   # Flexible: explicit path specification
   wtp add /tmp/experiment feature/auth
   wtp add --detach /tmp/debug abc1234
   ```

3. **Transparent Wrapper**:
   - All git worktree flags pass through unchanged
   - Argument handling adapts based on path vs branch name detection
   - Error messages come directly from git worktree

4. **Benefits Achieved**:
   - **Learning cost reduction**: git worktree users can use immediately
   - **Redundancy elimination**: no more typing branch names twice
   - **Flexibility maintained**: all git worktree features available
   - **Team consistency**: shared path management via config

**Files Modified**:

- `cmd/wtp/main.go`: Core implementation
- `cmd/wtp/main_test.go`: Test coverage for new logic
- `README.md`: Updated Quick Start with hybrid examples
- `CLAUDE.md`: This documentation

**Testing**: All existing tests pass, new path resolution logic tested

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

### 2025-01: E2E Test Fixes and Lint Improvements

**Background**: Several e2e tests were failing and there were multiple lint issues that needed to be resolved without using suppression directives.

**Issues Fixed**:

1. **Shell-init Command Test Failures**:
   - Tests expected shell-init to output actual completion scripts
   - Fixed by changing shell-init to output the full completion script instead of just source commands
   - Added proper `compdef` for zsh to ensure completion functionality

2. **Invalid Branch Name Test**:
   - Test framework's `validateArg` was blocking `{` and `}` characters
   - These characters are valid in git branch names (e.g., `branch@{upstream}`)
   - Updated validation to allow these characters

3. **TrackWithDetach Test**:
   - Test expected `--track` with `--detach` to work
   - Git doesn't allow this combination: "fatal: --[no-]track can only be used if a new branch is created"
   - Updated test to expect an error, matching git's actual behavior

4. **Error Output Capture**:
   - Changed from `log.Fatal` to `fmt.Fprintf(os.Stderr, ...)` for proper error capture
   - Ensures test framework can capture error messages via CombinedOutput()

5. **Lint Issues Resolved**:
   - Fixed nil context warnings by using `context.TODO()`
   - Resolved gosec G204 warnings by refactoring exec.Command usage
   - Created `createSafeCommand` helper function to separate validation from execution
   - All fixes done without using `//nolint` or `#nosec` directives

**Key Learnings**:

- Always test with actual git behavior before making assumptions
- Error messages must go to stderr for proper test capture
- Security validations should allow legitimate use cases (like git ref syntax)
- Lint issues should be fixed properly, not suppressed

**Files Modified**:

- `cmd/wtp/main.go`: Error output handling
- `cmd/wtp/completion.go`: Shell-init output and context fixes
- `cmd/wtp/add.go`: Track with detach logic
- `test/e2e/framework/framework.go`: Validation and command execution
- `test/e2e/remote_test.go`: TrackWithDetach test expectations
- `test/e2e/error_test.go`: Debug output for troubleshooting

---

This document serves as a living record of the project's development. Update as
new decisions are made or challenges are encountered.
