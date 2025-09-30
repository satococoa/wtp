# Development History

This document records the major changes and decisions made during the development of wtp.

## Project Renaming

- **Command**: `git-wtp` → `wtp` (easier typing, follows patterns like gh/ghq/tig)
- **Config**: `.git-worktree-plus.yml` → `.wtp.yml` (consistency, shorter)

## Major Design Changes

### Shell Integration (cd command) Implementation

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
   - **Unified Setup Command**: `wtp completion <shell>` generates both completion and cd functionality
   - **Cross-Shell Support**: Bash, Zsh, and Fish implementations

**Files Added/Modified**:
- `cmd/wtp/cd.go`: Core cd command implementation
- `cmd/wtp/cd_test.go`: Tests for cd functionality
- `cmd/wtp/completion.go`: Extended with shell function generation
- `cmd/wtp/main.go`: Added cd command registration
- `README.md`: Updated documentation and feature checklist

### Simplify add Command by Removing Rarely Used Options

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

### Explicit Path Flag Implementation

**Background**: User feedback identified ambiguity issue with automatic path detection - `foobar/foo` could be interpreted as either a path or branch name, causing confusion.

**Solution**: Replaced automatic path detection with explicit `--path` flag for unambiguous behavior.

**Changes Made**:

1. **Added --path Flag**:
   ```bash
   # Before (ambiguous)
   wtp add foobar/foo          # Is this a path or branch?

   # After (explicit)
   wtp add --path foobar/foo feature/auth  # Clear: foobar/foo is path
   wtp add foobar/foo                       # Clear: foobar/foo is branch
   ```

2. **Removed isPath() Function**: Eliminated automatic path detection logic
3. **Updated resolveWorktreePath**: Simple flag-based logic
4. **Benefits**: Clarity, Safety, Consistency, Maintainability

### TDD-Driven Command Architecture & Test Strategy Revolution

**Background**: Following Issue #3 (Test Quality Improvement) and user feedback about testable command design, we implemented a revolutionary new command architecture using TDD principles.

**Evolution Summary**:
- **Generation 1**: Direct Git Execution (slow, environment dependent)
- **Generation 2**: GitExecutor Interface (mockable, but string-based)
- **Generation 3**: CommandExecutor with Type Safety (TDD-designed, structured)

**Benefits Achieved**:
- **Fast TDD Cycle**: Unit tests run in milliseconds
- **Error Testing**: Easy to test failure scenarios with mocks
- **Environment Independence**: Tests don't require git installation
- **Clear Separation**: Unit tests verify wtp logic, E2E tests verify git integration

### Hook Output Streaming Fix (TDD Example)

**Problem**: Hook output was buffered and shown only after completion instead of streaming in real-time.

**TDD Solution**:

1. **RED**: Created `TestExecutePostCreateHooks_StreamingOutput` that verified output appears in real-time by tracking write timestamps.

2. **GREEN**: Modified `ExecutePostCreateHooks` to accept `io.Writer` and stream output directly instead of buffering.

3. **REFACTOR**: Removed duplicate methods, simplified API to single writer-based method.

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

## Evolution to Environment-Independent Testing

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

## Migration Status

**Completed**:
- ✅ CommandExecutor architecture for all commands
- ✅ Comprehensive unit test mocks
- ✅ Dependency injection patterns
- ✅ Build tag separation
- ✅ TDD-driven bug fixes (hook streaming)

**In Progress**:
- 🔄 Full integration test coverage
- 🔄 Docker-based test isolation

**Future**:
- 📋 Performance benchmarks
- 📋 Mutation testing
- 📋 Property-based testing for edge cases

## Current Test Coverage

- **Unit Tests**: 83%+ coverage for core functionality
- **Integration Tests**: Critical git operations covered
- **E2E Tests**: Major user workflows verified
- **Manual Testing**: Shell completions across bash/zsh/fish