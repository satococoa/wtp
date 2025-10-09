# wtp Architecture

This document describes the technical architecture and implementation details of wtp (Worktree Plus).

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
├── hooks/          # Hook execution
├── command/        # Command execution framework
└── errors/         # Error handling
```

## CommandExecutor Architecture

### Evolution of Command Design

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

### Key Benefits of New Architecture

1. **Type Safety**: Compile-time validation of command construction
2. **Testability**: Mock command execution without running git
3. **Composability**: Multiple commands in single execution
4. **Maintainability**: Centralized command building logic
5. **Extensibility**: Easy to add new git operations

### Command Building Pattern

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

## Configuration System

### YAML Configuration Format

We chose YAML for configuration because:
- Human-readable and writable
- Supports complex structures (arrays, maps)
- Well-supported in Go ecosystem
- Familiar to developers (CI/CD configs)

### Configuration Structure

```yaml
defaults:
  base_dir: "../worktrees"
  namespace_by_repo: true

hooks:
  post_create:
    - type: copy
      from: ".env.example"
      to: ".env"
    - type: command
      command: "npm install"
      work_dir: "."
```

### Worktree Namespacing

**Design Decision**: Namespace worktrees by repository name to prevent conflicts when
multiple projects share a parent directory.

**Architecture**:

```go
func (c *Config) ResolveWorktreePath(repoRoot, worktreeName string) string {
    baseDir := c.Defaults.BaseDir
    if !filepath.IsAbs(baseDir) {
        baseDir = filepath.Join(repoRoot, baseDir)
    }

    // Check if we should use namespacing
    if c.ShouldNamespaceByRepo() {
        repoName := filepath.Base(repoRoot)
        return filepath.Join(baseDir, repoName, worktreeName)
    }

    return filepath.Join(baseDir, worktreeName)
}
```

**Auto-detection Logic**:

When `namespace_by_repo` is not explicitly set in `.wtp.yml`, wtp auto-detects the
appropriate layout based on existing worktrees:

```go
func hasLegacyWorktrees(repoRoot, baseDir string) bool {
    // Check if base directory exists
    if _, err := os.Stat(baseDir); os.IsNotExist(err) {
        return false
    }

    // Look for worktrees directly under baseDir
    // (not under baseDir/<repo-name>/)
    entries, _ := os.ReadDir(baseDir)
    repoName := filepath.Base(repoRoot)

    for _, entry := range entries {
        if !entry.IsDir() || entry.Name() == repoName {
            continue
        }

        // Check if it looks like a worktree (has .git file)
        if hasGitFile(filepath.Join(baseDir, entry.Name())) {
            return true
        }
    }

    return false
}
```

**Behavior**:
- `namespace_by_repo: true` - Always use namespaced layout
- `namespace_by_repo: false` - Always use legacy layout
- `nil` (not set) - Auto-detect based on existing worktrees
  - If legacy worktrees found → use legacy layout (backwards compatible)
  - If no worktrees found → use namespaced layout (new projects)

**Migration**:

The `migrate-worktrees` command uses `git worktree move` to relocate worktrees:

```go
func migrateWorktree(repo *git.Repository, oldPath, newPath string) error {
    // Create parent directory
    os.MkdirAll(filepath.Dir(newPath), 0o755)

    // Use git worktree move (handles all internal path updates)
    repo.ExecuteGitCommand("worktree", "move", oldPath, newPath)

    return nil
}
```

## Hook System

### Design Philosophy

Post-create hooks support:
- File copying (for .env files, etc.)
- Command execution

This covers 90% of use cases without over-engineering.

### Hook Execution with Streaming Output

The hook system streams output in real-time to provide better user experience:

```go
// ExecutePostCreateHooks executes all post-create hooks and streams output to writer
func (e *Executor) ExecutePostCreateHooks(w io.Writer, worktreePath string) error {
    for i, hook := range e.config.Hooks.PostCreate {
        if err := e.executeHookWithWriter(w, &hook, worktreePath); err != nil {
            return fmt.Errorf("failed to execute hook %d: %w", i+1, err)
        }
    }
    return nil
}
```

## Branch Resolution Strategy

Following Git's own behavior:
- First check local branches
- If not found, search remote branches
- If multiple remotes have the same branch, error and ask for explicit remote
- This matches `git checkout` and `git switch` behavior

## Shell Integration

### CD Command Implementation

The `wtp cd` command uses a two-part architecture:

1. **Go Command**: `wtp cd <worktree>` finds the worktree path and outputs it
2. **Shell Function**: Wraps the Go command and performs the actual `cd`

### Shell Integration Flow

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

### Key Design Decisions

- **Environment Variable Check**: `WTP_SHELL_INTEGRATION=1` prevents accidental direct usage
- **Shell Function Wrapper**: Required because child processes can't change parent's directory
- **Unified Setup Command**: `wtp shell-init <shell>` generates both completion and cd functionality
- **Cross-Shell Support**: Bash, Zsh, and Fish implementations

## Go 1.24 Tool Directive

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

## Performance Optimizations

### Potential Future Improvements

- Cache git command outputs
- Parallel hook execution
- Lazy loading of worktree information
- Parallel command execution across worktrees

## Path Handling

Branch names with slashes become directory structure (e.g., `feature/auth` → `../worktrees/feature/auth/`)

## Error Handling

The project uses structured error handling with helpful user messages:

```go
func MultipleBranchesFound(branchName string, remotes []string) error {
    msg := fmt.Sprintf("branch '%s' exists in multiple remotes: %s", branchName, strings.Join(remotes, ", "))
    msg += fmt.Sprintf(`

Solution: Specify the remote explicitly:
  • wtp add --track %s/%s %s`, remotes[0], branchName, branchName)
    
    return errors.New(msg)
}
```

This provides clear guidance to users when errors occur.