# wtp (Worktree Plus)

A powerful Git worktree management tool that extends git's worktree
functionality with automated setup, branch tracking, and project-specific hooks.

## Features

### Core Commands

- [x] `wtp init` - Initialize configuration file
- [x] `wtp add` - Clean and unambiguous worktree creation
  - [x] **Automatic path generation**: `wtp add feature/auth` (no redundant
        typing)
  - [x] **Explicit path support**: `wtp add --path /custom/path feature/auth`
        (no ambiguity)
  - [x] **Transparent wrapper**: All git worktree options supported
  - [x] **Automatic remote tracking**: Tracks remote branches when local branch
        doesn't exist
  - [x] **Multiple remote handling**: Detects and reports when branch exists in
        multiple remotes
  - [x] Post-create hooks execution
- [x] `wtp remove` - Remove worktree
  - [x] Remove worktree only (git worktree compatible)
  - [x] Remove with branch (`--with-branch` option for convenience)
  - [x] Force removal (`--force` option)
- [x] `wtp list` - List all worktrees with relative paths and current worktree marker
- [x] `wtp cd` - Change directory to worktree (requires shell integration)

### Advanced Features

- [x] **Post-create hooks**
  - [x] Copy files from main worktree
  - [x] Execute commands
- [x] **Shell completion** (with custom completion for branches and worktrees)
  - [x] Bash completion with branch/worktree name completion
  - [x] Zsh completion with branch/worktree name completion
  - [x] Fish completion with branch/worktree name completion
- [x] **Cross-platform support**
  - [x] Linux (x86_64, ARM64)
  - [x] macOS (Apple Silicon only)
- [x] **Better error messages**
  - [x] Contextual error descriptions with clear causes
  - [x] Actionable suggestions for resolution
  - [x] Examples and helpful tips

## Requirements

- Git 2.17 or later (for worktree support)
- One of the following operating systems:
  - Linux (x86_64 or ARM64)
  - macOS (Apple Silicon M1/M2/M3)
- One of the following shells (for completion support):
  - Bash
  - Zsh
  - Fish

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install satococoa/tap/wtp
```

### Using Go

```bash
go install github.com/satococoa/wtp/cmd/wtp@latest
```

### Download Binary

Download the latest binary from
[GitHub Releases](https://github.com/satococoa/wtp/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/satococoa/wtp/releases/latest/download/wtp_Darwin_arm64.tar.gz | tar xz
sudo mv wtp /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/satococoa/wtp/releases/latest/download/wtp_Linux_x86_64.tar.gz | tar xz
sudo mv wtp /usr/local/bin/

# Linux (ARM64)
curl -L https://github.com/satococoa/wtp/releases/latest/download/wtp_Linux_arm64.tar.gz | tar xz
sudo mv wtp /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/satococoa/wtp.git
cd wtp
go build -o wtp ./cmd/wtp
sudo mv wtp /usr/local/bin/  # or add to PATH
```

## Quick Start

### Automatic Path Generation (Recommended)

```bash
# Create worktree from existing branch (local or remote)
# → Creates worktree at ../worktrees/feature/auth
# Automatically tracks remote branch if not found locally
wtp add feature/auth

# Create worktree with new branch
# → Creates worktree at ../worktrees/feature/new-feature
wtp add -b feature/new-feature

# Create new branch from specific commit
# → Creates worktree at ../worktrees/hotfix/urgent
wtp add -b hotfix/urgent abc1234

# Use all git worktree options
# → Creates worktree at ../worktrees/feature/test
wtp add -b feature/test --track origin/main

# Remote branch handling examples:

# Automatically tracks remote branch if not found locally
# → Creates worktree tracking origin/feature/remote-only
wtp add feature/remote-only

# If branch exists in multiple remotes, shows helpful error:
# Error: branch 'feature/shared' exists in multiple remotes: origin, upstream
# Please specify the remote explicitly (e.g., --track origin/feature/shared)
wtp add feature/shared

# Explicitly specify which remote to track
wtp add --track upstream/feature/shared feature/shared

# Control directory change behavior
wtp add --cd feature/auth        # Always change to new worktree
wtp add --no-cd feature/auth      # Never change directory
# Without flags, uses cd_after_create setting from .wtp.yml
```

### Explicit Path Specification (Full Flexibility)

```bash
# Create worktree at custom absolute path
wtp add --path /tmp/experiment feature/auth

# Create worktree at custom relative path
wtp add --path ./custom-location feature/auth

# Create detached HEAD worktree from commit
wtp add --path /tmp/debug --detach abc1234

# All git worktree options work with explicit paths
wtp add --path /tmp/test --force feature/auth

# No ambiguity: foobar/foo is always treated as branch name
wtp add --path /custom/location foobar/foo
```

### Management Commands

```bash
# List all worktrees
wtp list

# Example output:
# PATH                      BRANCH           HEAD
# ----                      ------           ----
# @ (main worktree)*        main             c72c7800
# feature-auth              feature/auth     def45678
# ../project-hotfix         hotfix/urgent    abc12345

# Remove worktree only (by worktree directory name)
wtp remove auth
wtp remove --force auth  # Force removal even if dirty

# Remove worktree and its branch
wtp remove --with-branch auth              # Only if branch is merged
wtp remove --with-branch --force-branch auth  # Force branch deletion
```

## Configuration

wtp uses `.wtp.yml` for project-specific configuration:

```yaml
version: "1.0"
defaults:
  # Base directory for worktrees (relative to project root)
  base_dir: "../worktrees"

  # Automatically change to the new worktree directory after creation
  cd_after_create: true

hooks:
  post_create:
    # Copy files from repository root to new worktree
    - type: copy
      from: ".env.example"
      to: ".env"

    - type: copy
      from: "config/database.yml.example"
      to: "config/database.yml"

    # Execute commands in the new worktree
    - type: command
      command: "npm install"
      env:
        NODE_ENV: "development"

    - type: command
      command: "make db:setup"
      work_dir: "."
```

## Shell Integration

### Quick Setup (Recommended)

#### If installed via Homebrew or Package Manager

Shell completions are automatically installed and should work immediately! No
manual setup required.

#### Manual Setup

If you installed wtp manually, add the following to your shell configuration
file:

```bash
# Bash: Add to ~/.bashrc
eval "$(wtp completion bash)"

# Zsh: Add to ~/.zshrc
eval "$(wtp completion zsh)"

# Fish: Add to ~/.config/fish/config.fish
wtp completion fish | source
```

This enables:

- Tab completion for all wtp commands, flags, and options
- Branch name completion for `wtp add`
- Worktree name completion for `wtp remove` and `wtp cd`
- The `wtp cd` command for quick navigation to worktrees

### Using the cd Command

Once shell integration is enabled, you can quickly change to any worktree:

```bash
# Change to a worktree by its directory name
wtp cd auth

# Change to the root worktree using the '@' shorthand
wtp cd @

# Tab completion works!
wtp cd <TAB>
```

## Worktree Structure

With the default configuration (`base_dir: "../worktrees"`):

```
<project-root>/
├── .git/
├── .wtp.yml
└── src/

../worktrees/
├── main/
├── feature/
│   ├── auth/          # wtp add feature/auth
│   └── payment/       # wtp add feature/payment
└── hotfix/
    └── bug-123/       # wtp add hotfix/bug-123
```

Branch names with slashes are preserved as directory structure, automatically
organizing worktrees by type/category.

## Error Handling

wtp provides clear error messages:

```bash
# Branch not found
Error: branch 'nonexistent' not found in local or remote branches

# Multiple remotes have same branch
Error: branch 'feature' exists in multiple remotes: origin, upstream. Please specify remote explicitly

# Worktree already exists
Error: failed to create worktree: exit status 128

# Uncommitted changes
Error: Cannot remove worktree with uncommitted changes. Use --force to override
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md)
for details.

### Development Setup

```bash
# Clone repository
git clone https://github.com/satococoa/wtp.git
cd wtp

# Install dependencies
go mod download

# Run tests
go tool task test

# Build
go tool task build

# Run locally
./wtp --help
```

## Roadmap

### v0.1.0 (MVP) ✅ COMPLETED

- [x] Basic commands (add, remove, list)
- [x] Local branch support
- [x] Remote branch tracking
- [x] Configuration file support
- [x] Post-create hooks

### v0.2.0

- [x] Shell completion (with custom branch/worktree completion)
- [x] Init command for configuration
- [x] Branch creation (`-b` flag)
- [x] Hybrid approach (automatic + explicit path support)

### v0.3.0

- [x] Remove with branch (`--with-branch` option)
- [x] Shell integration (cd command)
- [x] Multiple remote handling
- [x] Better error messages

### v1.0.0

- [x] Stable release
- [x] Full test coverage (current: 77.4%, target: 80%+)
- [x] Package manager support (Homebrew, apt/yum/apk)

### Future Ideas

- [ ] `git wtp status` - Show status of all worktrees
- [ ] `git wtp foreach` - Run command in all worktrees
- [ ] `git wtp clean` - Remove merged worktrees

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by git-worktree and the need for better multi-branch development
workflows.
