# Git Worktree Plus - Development Notes

This document contains the design decisions, implementation details, and
discussions that led to the creation of git-wtp.

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

### 2. Git Command Integration

- Parsing git output reliably
- Handling different git versions
- Error message consistency

### 3. Shell Integration

The `cd` command requires shell functions since child processes can't change
parent's directory.

## Technical Architecture

```
cmd/
├── git-wtp/
│   └── main.go          # Entry point
├── add.go               # Add command
├── remove.go            # Remove command
├── list.go              # List command
├── init.go              # Init command
└── completion.go        # Shell completion

internal/
├── git/
│   ├── repository.go    # Git operations
│   └── worktree.go      # Worktree management
├── config/
│   └── config.go        # YAML configuration
└── hooks/
    └── executor.go      # Hook execution
```

## Development Timeline

1. **Day 1-2**: Core structure, add/list commands
2. **Day 3-4**: Branch resolution, remote tracking
3. **Day 5-6**: Configuration and hooks
4. **Day 7**: Shell completion and integration
5. **Day 8+**: Testing, documentation, release

## Post-Implementation Checklist

When implementing new features, always remember to:

1. **Update Documentation**: After adding new features or flags, update both README.md and any relevant documentation
2. **Update Feature Checklists**: Mark completed features as done in README.md roadmap
3. **Add Usage Examples**: Include practical examples in the Quick Start section
4. **Update Help Text**: Ensure command help text reflects new options
5. **Run Tests**: Always run `make lint` and `make test` before committing
6. **Update CLAUDE.md**: Document any new design decisions or architectural changes

## Testing Strategy

- Unit tests for branch resolution logic
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
- Configuration via .git-worktree-plus.yml
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

### Required Commands

When implementing new features or fixing bugs:

1. **Development**: `make dev` (runs lint, test, build)
2. **Testing**: `make test`
3. **Linting**: `make lint` 
4. **Formatting**: `make fmt`

**Critical**: ALWAYS run `make lint` and fix all issues before committing.

---

This document serves as a living record of the project's development. Update as
new decisions are made or challenges are encountered.
