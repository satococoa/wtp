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

### Evolution to Environment-Independent Testing

**Previous Approach** (Before CommandExecutor):

- Mixed unit and integration tests
- Environment-dependent behavior
- Difficult to mock git operations

**Current Approach** (TDD-driven CommandExecutor):

- Clear separation of test levels
- Environment-independent unit tests
- Comprehensive mocking capabilities

### Test Levels

#### 1. Unit Tests (`*_test.go`)

- **Purpose**: Test business logic and command flow in isolation
- **Dependencies**: All external dependencies mocked via dependency injection
- **Execution Time**: < 100ms per test
- **Environment**: Any (no git required)
- **Coverage**: 80%+ for core functionality

**Example Pattern**:

```go
// Dependency injection for testability
type ListDependencies struct {
    CommandExecutor command.Executor
    GetWorkingDir   func() (string, error)
    NewRepository   func(string) (GitRepository, error)
}

// Test with mocked dependencies
func TestListCommand_WithMocks(t *testing.T) {
    mockExecutor := testutil.NewMockCommandExecutor(t)
    mockExecutor.SetWorktreeListResult(testFixtures)
    // ... test logic
}
```

#### 2. Integration Tests (`*_integration_test.go`, build tag: `integration`)

- **Purpose**: Test integration with real git commands in isolated environments
- **Dependencies**: Git binary, temporary repositories
- **Execution Time**: < 1s per test
- **Environment**: Isolated temporary git repositories

#### 3. E2E Tests (`test/e2e/*.go`, build tag: `e2e`)

- **Purpose**: Test complete user workflows and scenarios
- **Dependencies**: Full environment (git, shell, filesystem)
- **Execution Time**: < 10s per test
- **Environment**: Realistic user scenarios

### Key Improvements

#### Dependency Injection Pattern

All commands now support dependency injection for external dependencies:

```go
// Production dependencies
var DefaultListDependencies = ListDependencies{
    CommandExecutor: command.NewRealExecutor(),
    GetWorkingDir:   os.Getwd,
    NewRepository:   git.NewRepository,
}

// Test helper for dependency injection
func SetListDependenciesForTest(deps ListDependencies) func() {
    // Returns cleanup function
}
```

#### Comprehensive Mocking

- `MockCommandExecutor`: Simulates git command execution
- `MockGitRepository`: Provides test fixtures for repository state
- Test fixtures for common scenarios (empty repos, multiple worktrees, etc.)

#### Build Tags for Test Separation

```bash
# Unit tests only (fast, no dependencies)
make test-unit

# Integration tests (requires git)
make test-integration

# All tests
make test-all
```

### Test Execution Strategy

#### Development Workflow

```bash
# Fast feedback loop (< 5 seconds)
make dev-fast    # fmt + lint + unit tests

# Complete verification (< 30 seconds)
make dev         # includes integration tests

# Full CI pipeline (< 60 seconds)
make ci          # all tests + coverage
```

#### Continuous Integration

- **Pull Requests**: Unit tests + lint (fast feedback)
- **Main Branch**: Full test suite including integration and E2E
- **Release**: Additional manual testing on multiple platforms

### Testing Best Practices Adopted

1. **Test Pyramid with Appropriate Abstraction Levels**:
   - **Unit Tests (70%)**: Simple, fast, focused on What not How
   - **Integration Tests**: Removed - eliminated problematic middle layer
   - **E2E Tests (30%)**: Real user workflows with Living Specifications

2. **Simplicity Over Complexity**:
   - Unit tests kept simple and maintainable
   - No over-abstraction in test code
   - Clear, concise test names expressing intent

3. **Living Specifications at the Right Level**:
   - **E2E Tests**: User Stories, Business Value, Given-When-Then
   - **Unit Tests**: Simple What testing, minimal documentation
   - **Separation**: Avoid mixing specification concerns with unit testing

4. **Test Naming Strategy**:
   - **Unit Tests**: `TestFunction_Condition` (simple, direct)
   - **E2E Tests**: `TestUserAction_WhenCondition_ShouldOutcome` (specification
     style)
   - Avoid over-long names that obscure intent

5. **Environment Independence**:
   - No implicit dependencies on git repository state
   - All external dependencies explicitly injected
   - Deterministic test results

6. **Fast Feedback**:
   - Unit tests run in milliseconds
   - Fail fast on lint/format issues
   - Clear error messages without excessive verbosity

### Current Test Coverage

- **Unit Tests**: 83%+ coverage for core functionality
- **Integration Tests**: Critical git operations covered
- **E2E Tests**: Major user workflows verified
- **Manual Testing**: Shell completions across bash/zsh/fish

### Migration Status

**Completed**:

- ✅ CommandExecutor architecture for all commands
- ✅ Comprehensive unit test mocks
- ✅ Dependency injection patterns
- ✅ Build tag separation

**In Progress**:

- 🔄 Full integration test coverage
- 🔄 Docker-based test isolation

**Future**:

- 📋 Performance benchmarks
- 📋 Mutation testing
- 📋 Property-based testing for edge cases

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

For detailed testing guidelines, see [docs/testing-guidelines.md](docs/testing-guidelines.md).

**Files Added/Modified**:

- `internal/command/`: New package with executor, builders, and shell
  abstraction
- `cmd/wtp/add.go`: Added new CommandExecutor-based implementation
- `cmd/wtp/add_test.go`: Consolidated, pure unit tests only
- `test/e2e/worktree_test.go`: Existing E2E tests (unchanged)

**Testing**: All tests pass, coverage maintained at 83%+

This architecture establishes a foundation for consistent, testable command
handling across all wtp commands.

### TDD Approach for Bug Fixes

When fixing bugs, always follow the Test-Driven Development (TDD) cycle:

#### 1. 🔴 RED - Write a Failing Test First

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

#### 2. 🟢 GREEN - Write Minimal Code to Pass

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

#### 3. ♻️ REFACTOR - Improve the Code

Once tests pass, refactor for clarity and maintainability:
- Remove duplicate code
- Simplify APIs
- Improve naming
- Add documentation

#### Real Example: Hook Output Streaming Fix

**Problem**: Hook output was buffered and shown only after completion.

**TDD Solution**:

1. **RED**: Created `TestExecutePostCreateHooks_StreamingOutput` that verified
   output appears in real-time by tracking write timestamps.

2. **GREEN**: Modified `ExecutePostCreateHooks` to accept `io.Writer` and
   stream output directly instead of buffering.

3. **REFACTOR**: Removed duplicate methods, simplified API to single
   writer-based method.

**Key Testing Techniques**:

- **Custom Writers**: Create writers that track when writes occur
- **Time Verification**: Check timestamps to ensure real-time behavior
- **Mock Commands**: Use scripts with controlled output timing

```go
type streamingWriter struct {
    writes []writeRecord
    mu     sync.Mutex
}

type writeRecord struct {
    data string
    time time.Time
}
```

#### Benefits of TDD for Bug Fixes

1. **Regression Prevention**: Test ensures bug doesn't reappear
2. **Clear Understanding**: Test documents expected behavior
3. **Focused Fix**: Only write code needed to pass test
4. **Safe Refactoring**: Tests protect against breaking changes
5. **Better Design**: TDD often leads to cleaner APIs

#### Guidelines

- **Never skip the RED phase**: Ensure test fails before fixing
- **Keep tests focused**: One test per bug/behavior
- **Test behavior, not implementation**: Focus on what, not how
- **Use meaningful assertions**: Clearly express expectations
- **Consider edge cases**: Test boundary conditions

---

This document serves as a living record of the project's development. Update as
new decisions are made or challenges are encountered.
