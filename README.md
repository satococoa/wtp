# wtp (Worktree Plus)

A powerful Git worktree management tool that extends git's worktree
functionality with automated setup, branch tracking, and project-specific hooks.

## Features - Why wtp Instead of git-worktree?

### 🚀 No More Path Gymnastics

**git-worktree pain:**
`git worktree add ../project-worktrees/feature/auth feature/auth` **wtp
solution:** `wtp add feature/auth`

wtp automatically generates sensible paths based on branch names. Your
`feature/auth` branch goes to `../worktrees/feature/auth` - no redundant typing,
no path errors.

### 🧹 Clean Branch Management

**git-worktree pain:** Remove worktree, then manually delete the branch. Forget
the second step? Orphaned branches accumulate. **wtp solution:**
`wtp remove --with-branch feature/done` - One command removes both

Keep your repository clean. When a feature is truly done, remove both the
worktree and its branch in one atomic operation. No more forgotten branches
cluttering your repo.

### 🛠️ Zero-Setup Development Environments

**git-worktree pain:** Create worktree → Copy .env → Install deps → Run
migrations → Finally start coding **wtp solution:** Configure once in
`.wtp.yml`, then every `wtp add` runs your setup automatically

```yaml
hooks:
  post_create:
    - type: copy
      from: ".env.example"
      to: ".env"
    - type: command
      command: "npm install && npm run db:setup"
```

Perfect for microservices, monorepos, or any project with complex setup
requirements.

### 📍 Instant Worktree Navigation

**git-worktree pain:** `cd ../../../worktrees/feature/auth` (if you remember the
path) **wtp solution:** `wtp cd feature/auth` with tab completion

Jump between worktrees instantly. Use `wtp cd @` to return to your main
worktree. No more terminal tab confusion.

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

# Create new branch tracking a different remote branch
# → Creates worktree at ../worktrees/feature/test with branch tracking origin/main
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

### Management Commands

```bash
# List all worktrees
wtp list

# Example output:
# PATH                      BRANCH           HEAD
# ----                      ------           ----
# @ (main worktree)*        main             c72c7800
# feature/auth              feature/auth     def45678
# ../project-hotfix         hotfix/urgent    abc12345

# Remove worktree only (by worktree name)
wtp remove feature/auth
wtp remove --force feature/auth  # Force removal even if dirty

# Remove worktree and its branch
wtp remove --with-branch feature/auth              # Only if branch is merged
wtp remove --with-branch --force-branch feature/auth  # Force branch deletion
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
# Change to a worktree by its name
wtp cd feature/auth

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


## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by git-worktree and the need for better multi-branch development
workflows.
