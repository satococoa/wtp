# Testing Guidelines for wtp

This document establishes testing standards and best practices for the wtp
project, based on lessons learned from implementing a specification-centered
testing approach.

> **Note**: For TDD bug fixing examples and architectural decisions, see
> [CLAUDE.md](../CLAUDE.md).

## Core Testing Philosophy

### 1. Specification-First Testing

- Tests express **user expectations**, not implementation details
- Every test must have clear business value
- Tests serve as **living documentation** of system behavior
- Focus on "What" the system should do, not "How" it does it

### 2. Test Pyramid with Proper Abstraction Levels

```
    E2E Tests (30%)
  Real User Workflows
 Living Specifications
-------------------------
   Unit Tests (70%)
  Simple, Fast, Mocked
 What Testing, Minimal Docs
```

## Test Categories and When to Use Each

### Real-World Edge Cases (Table Tests)

**Purpose**: Test scenarios users actually encounter with international
characters, special inputs, and edge cases

**Characteristics**:

- Focus on user-facing edge cases, not artificial ones
- Use table tests for multiple similar scenarios
- Include reason/description for each test case
- Cover international character support
- Test filesystem and OS limitations

**When to Add Edge Cases**:

- Unicode/international character support
- Special characters in file paths and branch names
- Long names and path depth limits
- Common flag combinations
- Real user error scenarios (based on support requests)

**Example Categories**:

```go
// International character support
TestCommand_InternationalCharacters()

// Special characters and edge cases
TestCommand_SpecialCharacters()

// Filesystem limitations
TestCommand_PathLengthLimits()

// Flag combinations
TestCommand_FlagCombinations()
```

### Unit Tests (70% of test suite)

**Purpose**: Test business logic and command flow in isolation

**Characteristics**:

- Fast execution (< 100ms per test)
- All external dependencies mocked
- Simple naming: `TestFunction_Condition`
- Minimal documentation
- Environment-independent

**Example**:

```go
func TestAddCommand_ExistingBranch(t *testing.T) {
    mockExec := &mockCommandExecutor{shouldFail: false}
    var buf bytes.Buffer
    cmd := createTestCLICommand(map[string]any{}, []string{"feature/auth"})
    cfg := &config.Config{Defaults: config.Defaults{BaseDir: "/test/worktrees"}}

    err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

    assert.NoError(t, err)
    assert.Contains(t, buf.String(), "Created worktree 'feature/auth'")
}
```

### E2E Tests (30% of test suite)

**Purpose**: Test complete user workflows with real git operations

**Characteristics**:

- Real git repositories and commands
- Living Specifications with user stories
- Given-When-Then structure
- Business value documentation
- Specification-style naming: `TestUserAction_WhenCondition_ShouldOutcome`

**Example**:

```go
// TestUserCreatesWorktree_WithExistingLocalBranch_ShouldCreateWorktreeAtDefaultPath tests
// the most common user workflow: creating a worktree for an existing local branch.
//
// User Story: As a developer working on a feature branch, I want to create a worktree
// for an existing branch so I can quickly switch to working on that feature in isolation.
//
// Business Value: This eliminates the need to stash changes or commit incomplete work
// when switching between features, improving developer productivity.
func TestUserCreatesWorktree_WithExistingLocalBranch_ShouldCreateWorktreeAtDefaultPath(t *testing.T) {
    // Given: User has an existing local branch named "feature/auth"
    // When: User runs "wtp add feature/auth"
    // Then: Worktree should be created successfully
}
```

## Test Naming Conventions

### Unit Tests

**Pattern**: `TestFunction_Condition`

**Examples**:

- ✅ `TestAddCommand_ExistingBranch`
- ✅ `TestRemoveCommand_WithBranch`
- ✅ `TestListCommand_NoWorktrees`
- ❌ `TestAddCommandWithCommandExecutor_Success` (too verbose)
- ❌ `TestWorktreeCreation` (too vague)

### E2E Tests

**Pattern**: `TestUserAction_WhenCondition_ShouldOutcome`

**Examples**:

- ✅
  `TestUserCreatesWorktree_WithExistingLocalBranch_ShouldCreateWorktreeAtDefaultPath`
- ✅ `TestUserRemovesWorktree_WhenBranchHasChanges_ShouldWarnBeforeDeleting`
- ❌ `TestAddCommand_Success` (implementation-focused)

## Testing Anti-Patterns to Avoid

### ❌ Coverage-Driven Tests

Tests that exist solely to increase coverage without business value.

```go
// BAD: Testing implementation detail for coverage
func TestParseWorktreeListIntegration_InvalidFormat(t *testing.T)

// GOOD: Testing user-facing behavior
func TestListCommand_WhenGitFails_ShowsHelpfulError(t *testing.T)
```

### ❌ Implementation-Focused Names

Test names that describe code structure rather than user behavior.

```go
// BAD: Describes what code does
func TestAddCommandWithCommandExecutor_Success(t *testing.T)

// GOOD: Describes what user experiences
func TestAddCommand_ExistingBranch(t *testing.T)
```

### ❌ Over-Documentation in Unit Tests

Excessive Given-When-Then documentation in unit tests reduces maintainability.

```go
// BAD: Over-documented unit test
func TestAddCommand_ExistingBranch(t *testing.T) {
    // User Story: As a developer working on a feature branch...
    // Business Value: This eliminates the need to stash changes...
    // Given: User has an existing local branch named "feature/auth"
    // When: User runs "wtp add feature/auth"
    // Then: Worktree should be created successfully

    // ... test implementation
}

// GOOD: Simple unit test
func TestAddCommand_ExistingBranch(t *testing.T) {
    mockExec := &mockCommandExecutor{shouldFail: false}
    // ... test implementation
    assert.NoError(t, err)
    assert.Contains(t, buf.String(), "Created worktree 'feature/auth'")
}
```

## Test Implementation Patterns

### Unit Test Template

```go
func TestCommand_Scenario(t *testing.T) {
    // Setup mocks and test data
    mockExec := &mockCommandExecutor{shouldFail: false}
    var buf bytes.Buffer
    cmd := createTestCLICommand(flags, args)

    // Execute the command
    err := commandWithExecutor(cmd, &buf, mockExec, config, repoPath)

    // Verify results
    assert.NoError(t, err)
    assert.Contains(t, buf.String(), "expected output")
    assert.Equal(t, expectedCommands, mockExec.executedCommands)
}
```

### Table Test Template

```go
func TestCommand_MultipleScenarios(t *testing.T) {
    tests := []struct {
        name           string
        flags          map[string]any
        args           []string
        expectError    bool
        expectedOutput string
    }{
        {"basic scenario", map[string]any{}, []string{"branch"}, false, "expected"},
        {"error scenario", map[string]any{}, []string{}, true, "error message"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Real-World Edge Case Template

```go
func TestCommand_InternationalCharacters(t *testing.T) {
    tests := []struct {
        name         string
        branchName   string
        shouldWork   bool
        reason       string
    }{
        {
            name:       "Japanese characters",
            branchName: "機能/ログイン",
            shouldWork: true,
            reason:     "Git supports Unicode in branch names",
        },
        {
            name:       "Emoji characters",
            branchName: "feature/🚀-rocket",
            shouldWork: true,
            reason:     "Modern Git handles emoji characters",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test with real-world scenario
            // Focus on user-facing behavior, not implementation
        })
    }
}
```

### Mock Pattern

```go
type mockCommandExecutor struct {
    executedCommands []command.Command
    shouldFail       bool
}

func (m *mockCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
    m.executedCommands = commands

    if m.shouldFail {
        return &command.ExecutionResult{
            Results: []command.Result{{
                Command: commands[0],
                Error:   errors.GitCommandFailed("git", "mock error"),
            }},
        }, nil
    }

    // Success case implementation
}
```

## Code Review Checklist

### For All Tests

- [ ] Test name clearly describes the scenario being tested
- [ ] Test has obvious business value (not just coverage)
- [ ] Test failure provides actionable error message
- [ ] Test setup is minimal and focused
- [ ] Assertions match user expectations

### For Unit Tests

- [ ] All external dependencies are mocked
- [ ] Test executes in < 100ms
- [ ] Follows simple naming convention: `TestFunction_Condition`
- [ ] Minimal documentation (no excessive Given-When-Then)
- [ ] Environment-independent (no git commands, file system dependencies)

### For E2E Tests

- [ ] Tests complete user workflow
- [ ] Includes user story and business value documentation
- [ ] Follows specification naming: `TestUserAction_WhenCondition_ShouldOutcome`
- [ ] Uses real git operations and file system
- [ ] Given-When-Then structure clearly documented

## Quality Gates

### Automated Checks

- All tests must pass before merge
- Coverage should maintain 80%+ (as byproduct, not goal)
- Unit tests must complete in < 5 seconds total
- E2E tests must complete in < 60 seconds total

### Manual Review

- Every new test must be reviewed for business value
- Test names must follow established conventions
- Mock usage should be appropriate for test level
- Error messages should guide users to solutions

## Common Pitfalls and Solutions

### Problem: Slow Unit Tests

**Cause**: Using real git commands or file system operations in unit tests
**Solution**: Mock all external dependencies using dependency injection

### Problem: Brittle E2E Tests

**Cause**: Testing implementation details instead of user behavior **Solution**:
Focus on user workflows and outcomes, not internal state

### Problem: Unclear Test Intent

**Cause**: Generic test names and missing context **Solution**: Use descriptive
names and add user story context for E2E tests

### Problem: Coverage-Driven Development

**Cause**: Writing tests to increase coverage metrics **Solution**: Always start
with user requirement or bug report, then write test

## Migration Guide

### Existing Tests

When updating existing tests:

1. Identify the user behavior being tested
2. Rename test to reflect user scenario
3. Simplify unit tests by removing excessive documentation
4. Move complex workflows to E2E level
5. Ensure proper mocking for unit tests

### New Features

When adding new features:

1. Start with E2E test expressing user workflow
2. Add unit tests for edge cases and error conditions
3. Use table tests for multiple similar scenarios
4. Mock external dependencies in unit tests
5. Document business value for complex scenarios

---

This document reflects the testing strategy established through iterative
improvement and should be updated as new patterns emerge.
