# wtp (Worktree Plus) - Development Notes

This document contains the key design decisions and development guidance for wtp (formerly git-wtp).

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

## TDD Approach for Development

For all development work (bug fixes and new features), always follow the Test-Driven Development (TDD) cycle:

### 1. 🔴 RED - Write a Failing Test First

Before implementing any code change, write a test that describes the expected behavior:

```go
// Example: Bug fix - Hook output streaming
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

// Example: New feature - Template support
func TestAddCommand_WithTemplate(t *testing.T) {
    // Arrange: Mock template resolver
    mockResolver := &mockTemplateResolver{
        template: &Template{Name: "go-service", Files: []string{".env", "Dockerfile"}},
    }
    
    // Act: Add worktree with template
    err := addCmd.Execute([]string{"--template", "go-service", "feature/auth"})
    
    // Assert: Template files are created
    assert.NoError(t, err)
    assert.FileExists(t, filepath.Join(worktreeDir, ".env"))
    assert.FileExists(t, filepath.Join(worktreeDir, "Dockerfile"))
}
```

### 2. 🟢 GREEN - Write Minimal Code to Pass

Implement just enough to make the test pass:

```go
// Bug fix: Direct output to writer instead of buffering
func (e *Executor) ExecutePostCreateHooks(w io.Writer, path string) error {
    // Stream output directly to writer
    cmd.Stdout = w
    cmd.Stderr = w
    return cmd.Run()
}

// New feature: Add template support to add command
func (cmd *AddCommand) Execute(args []string) error {
    if cmd.templateName != "" {
        template, err := cmd.templateResolver.Resolve(cmd.templateName)
        if err != nil {
            return err
        }
        return cmd.applyTemplate(template, cmd.worktreePath)
    }
    // ... existing logic
}
```

### 3. ♻️ REFACTOR - Improve the Code

Once tests pass, refactor for clarity and maintainability:
- Remove duplicate code
- Simplify APIs
- Improve naming
- Add documentation

### Benefits of TDD for All Development

1. **Regression Prevention**: Tests ensure bugs don't reappear and features work as expected
2. **Clear Understanding**: Tests document expected behavior before implementation
3. **Focused Development**: Only write code needed to pass tests (no over-engineering)
4. **Safe Refactoring**: Tests protect against breaking changes during improvements
5. **Better Design**: TDD often leads to cleaner, more testable APIs
6. **Faster Feedback**: Quick validation that implementation meets requirements

### Guidelines

- **Never skip the RED phase**: Ensure test fails before implementing
- **Keep tests focused**: One test per feature/behavior/bug
- **Test behavior, not implementation**: Focus on what, not how
- **Use meaningful assertions**: Clearly express expectations
- **Consider edge cases**: Test boundary conditions and error scenarios

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