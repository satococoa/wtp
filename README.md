# wtp (Worktree Plus)

A powerful Git worktree management tool that extends git's worktree
functionality with automated setup, branch tracking, and project-specific hooks.

## Features - Why wtp Instead of git-worktree?

### 🚀 No More Path Gymnastics

**🧩 Problem:**
`git worktree add ../project-worktrees/feature/auth feature/auth`<br>
**✨ wtp:** `wtp add feature/auth`

wtp automatically generates sensible paths based on branch names. Your
`feature/auth` branch goes to `../worktrees/feature/auth` - no redundant typing,
no path errors.

### 🧹 Clean Branch Management

**🧩 Problem:** Remove worktree, then manually delete the branch. Forget
the second step? Orphaned branches accumulate.<br>
**✨ wtp:**
`wtp remove --with-branch feature/done` - One command removes both

Keep your repository clean. When a feature is truly done, remove both the
worktree and its branch in one atomic operation. No more forgotten branches
cluttering your repo.

### 🛠️ Zero-Setup Development Environments

**🧩 Problem:** Create worktree → Copy .env → Install deps → Run
migrations → Finally start coding<br>
**✨ wtp:** Configure once in
`.wtp.yml`, then every `wtp add` runs your setup automatically

```yaml
hooks:
  post_create:
    # Copy real files from the MAIN worktree into the NEW worktree
    - type: copy
      from: ".env" # Allowed even if gitignored. 'from' is always relative to the MAIN worktree
      to: ".env" # Destination is relative to the NEW worktree

    # Share directories between the MAIN and NEW worktree
    - type: symlink
      from: ".bin"
      to: ".bin"

    # Prefer explicit, single-step setup commands
    - type: command
      command: "npm ci" # Example for Node.js (replace with your build/deps tool)
    - type: command
      command: "npm run db:setup"
    # Alternative: using make or a task runner
    # - type: command
    #   command: "make bootstrap"
```

Perfect for microservices, monorepos, or any project with complex setup
requirements.

### 📍 Instant Worktree Navigation

**🧩 Problem:** `cd ../../../worktrees/feature/auth` (if you remember the
path)<br>
**✨ wtp:** `wtp cd feature/auth` with tab completion

Jump between worktrees instantly. Use `wtp cd @` to return to your main
worktree (or just `wtp cd`). No more terminal tab confusion.

## Requirements

- Git 2.17 or later (for worktree support)
- One of the following operating systems:
  - Linux (x86_64 or ARM64)
  - macOS (Apple Silicon M1/M2/M3)
- One of the following shells (for completion support):
  - Bash (4+/5.x) with bash-completion v2
  - Zsh
  - Fish

## Releases

View all releases and changelogs:
[GitHub Releases](https://github.com/satococoa/wtp/releases)

Latest stable version:
[See releases](https://github.com/satococoa/wtp/releases/latest)

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install satococoa/tap/wtp
```

### Using Go

```bash
go install github.com/satococoa/wtp/v2/cmd/wtp@latest
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

# Create worktree and run a command inside it after hooks
# → Useful for bootstrap steps (supports interactive commands when TTY is available)
wtp add -b feature/new-feature --exec "npm test"

# Create new branch tracking a different remote branch
# → Creates worktree at ../worktrees/feature/test with branch tracking origin/main
wtp add -b feature/test origin/main

# Remote branch handling examples:

# Automatically tracks remote branch if not found locally
# → Creates worktree tracking origin/feature/remote-only
wtp add feature/remote-only

# If branch exists in multiple remotes, shows helpful error:
# Error: branch 'feature/shared' exists in multiple remotes: origin, upstream
# Please specify the remote explicitly (e.g., --track origin/feature/shared)
wtp add feature/shared

# Explicitly specify which remote to track
wtp add -b feature/shared upstream/feature/shared
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

# Execute a command in an existing worktree (uses same target resolution as `wtp cd`)
wtp exec feature/auth -- go test ./...
wtp exec @ -- pwd
```

## Configuration

wtp uses `.wtp.yml` for project-specific configuration:

```yaml
version: "1.0"
defaults:
  # Base directory for worktrees (relative to project root)
  base_dir: "../worktrees"

hooks:
  post_create:
    # Copy gitignored files from main worktree to new worktree
    # Note: 'from' is relative to main worktree, 'to' is relative to new worktree
    # If 'to' is omitted, it defaults to the same value as 'from' (relative paths only)
    - type: copy
      from: ".env" # Copy actual .env file (gitignored)
      to: ".env"

    - type: copy
      from: ".claude" # Copy AI context file (gitignored)

    # Share directories between the main and new worktree
    - type: symlink
      from: ".bin"
      to: ".bin"

    # Execute commands in the new worktree
    - type: command
      command: "npm install"
      env:
        NODE_ENV: "development"

    - type: command
      command: "make db:setup"
      work_dir: "."
```

### Copy Hooks: Main Worktree Reference

Copy hooks are designed to help you bootstrap new worktrees using files from
your main worktree (even if they are gitignored):

- `from`: path is always resolved relative to the main worktree.
- `to`: path is resolved relative to the newly created worktree (defaults to `from` if omitted; absolute `from` requires explicit `to`).
- Supports files and directories, including entries ignored by Git (e.g.,
  `.env`, `.claude`, `.cursor/`).

Examples:

```yaml
hooks:
  post_create:
    # Copy local env and AI context from MAIN worktree into the new worktree
    - type: copy
      from: ".env"
      to: ".env"

    - type: copy
      from: ".claude"

    # Directory copy also works
    - type: copy
      from: ".cursor/"
      to: ".cursor/"
```

This behavior applies regardless of where you run `wtp add` from (main worktree
or any other worktree).

### Symlink Hooks: Shared Assets

Symlink hooks are useful for sharing large or mutable directories from the main
worktree (e.g. `.bin`, `.cache`, `node_modules`).

- `from`: path is resolved relative to the main worktree (or absolute).
- `to`: path is resolved relative to the newly created worktree (or absolute).

Example:

```yaml
hooks:
  post_create:
    - type: symlink
      from: ".bin"
      to: ".bin"
```

## Shell Integration

### Tab Completion Setup

#### If installed via Homebrew

No manual setup required. Homebrew installs a tiny bootstrapper that runs
`wtp shell-init <shell>` the first time you press `TAB` after typing `wtp`. That
lazy call gives you both tab completion and the `wtp cd` integration for the
rest of the session—no rc edits needed.

Need to refresh inside an existing shell? Just run `wtp shell-init <shell>`
yourself.

#### If installed via go install

Add a single line to your shell configuration file to enable both completion and
shell integration:

```bash
# Bash: Add to ~/.bashrc or ~/.bash_profile
eval "$(wtp shell-init bash)"

# Zsh: Add to ~/.zshrc
eval "$(wtp shell-init zsh)"

# Fish: Add to ~/.config/fish/config.fish
wtp shell-init fish | source
```

> **Note:** Bash completion requires bash-completion v2. On macOS, install
> Homebrew’s Bash 5.x and `bash-completion@2`, then
> `source /opt/homebrew/etc/profile.d/bash_completion.sh` (or the path shown
> after installation) before enabling the one-liner above.

After reloading your shell you get the same experience as Homebrew users.

### Navigation with wtp cd

The `wtp cd` command outputs the absolute path to a worktree. You can use it in
two ways:

#### Direct Usage

```bash
# Change to a worktree using command substitution
cd "$(wtp cd feature/auth)"

# Change to the main worktree
cd "$(wtp cd)"

# Or explicitly:
cd "$(wtp cd @)"
```

#### With Shell Hook (Recommended)

For a more seamless experience, enable the shell hook. `wtp shell-init <shell>`
already bundles it, so Homebrew users get the hook automatically and go install
users get it from the one-liner above. If you only want the hook without
completions, you can still run `wtp hook <shell>` manually.

Then use the simplified syntax:

```bash
# Change to a worktree by its name
wtp cd feature/auth

# Go to the main worktree (same as @)
wtp cd

# Change to the root worktree using the '@' shorthand
wtp cd @

# Tab completion works!
wtp cd <TAB>
```

#### Complete Setup (Lazy Loading for Homebrew Users)

Homebrew ships a lightweight bootstrapper. Press `TAB` after typing `wtp` and it
evaluates `wtp shell-init <shell>` once for your session—tab completion and
`wtp cd` just work.

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

### Formatting

Run `go tool task fmt` before sending changes. The formatter uses
`golangci-lint fmt` (gofmt + goimports) and automatically derives the
`goimports` `-local` prefix from `go list -m`, so forks and renamed modules
stay grouped correctly. `go tool golangci-lint fmt ./...` still works for
one-off runs, but the task is the authoritative workflow.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by git-worktree and the need for better multi-branch development
workflows.
