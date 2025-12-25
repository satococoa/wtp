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
  cd: true

hooks:
  post_create:
    - type: copy
      from: ".env.example"
      to: ".env"
    - type: command
      command: "npm install"
      work_dir: "."
    - type: symlink
      from: ".bin"
      to: ".bin"
```

## Hook System

### Design Philosophy

Post-create hooks support:
- File copying (for .env files, etc.)
- Command execution
- Symlink creation (for shared binaries, caches, etc.)

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

1. **Go Command**: `wtp cd [worktree]` finds the worktree path and outputs it (defaults to the main worktree when omitted)
2. **Shell Function**: Wraps the Go command and performs the actual `cd`

### Shell Integration Flow

```bash
# User types:
wtp cd feature/auth

# Shell function intercepts, runs:
command wtp cd feature/auth

# Go command returns path:
/path/to/worktrees/feature/auth

# Shell function performs:
cd /path/to/worktrees/feature/auth
```

### Key Design Decisions

- **Pure Path Output**: `wtp cd` only prints a path (no side effects), so hooks can safely consume it
- **Shell Function Wrapper**: Required because child processes can't change the parent shell's directory
- **Unified Setup Command**: `wtp shell-init <shell>` generates both completion and cd functionality
- **Cross-Shell Support**: Bash, Zsh, and Fish implementations

## Go 1.24 Tool Directive

This project uses Go 1.24's new tool directive for development tools:

```
tool (
    github.com/go-task/task/v3/cmd/task
    github.com/golangci/golangci-lint/v2/cmd/golangci-lint
)
```

**Important**: Always use `make` commands instead of calling tools directly:

- ✅ `make lint` (calls `go tool golangci-lint run`)
- ✅ `make fmt` (calls `go tool golangci-lint fmt ./...`)
- ❌ `golangci-lint run` (may use different version)
- ❌ `golangci-lint fmt` (may use different version)

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
