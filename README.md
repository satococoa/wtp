# wtp (Worktree Plus)

A powerful Git worktree management tool that extends git's worktree
functionality with automated setup, branch tracking, and project-specific hooks.

## Features - Why wtp Instead of git-worktree?

### 🚀 No More Path Gymnastics

**git-worktree pain:**
`git worktree add ../project-worktrees/feature/auth feature/auth` **wtp
solution:** `wtp add feature/auth`

wtp automatically generates sensible paths based on branch names. Your
`feature/auth` branch goes to `../worktrees/<repo-name>/feature/auth` - no redundant typing,
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
    # Copy real files from the MAIN worktree into the NEW worktree
    - type: copy
      from: ".env" # Allowed even if gitignored. 'from' is always relative to the MAIN worktree
      to: ".env" # Destination is relative to the NEW worktree

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
# → Creates worktree at ../worktrees/<repo-name>/feature/auth
# Automatically tracks remote branch if not found locally
wtp add feature/auth

# Create worktree with new branch
# → Creates worktree at ../worktrees/<repo-name>/feature/new-feature
wtp add -b feature/new-feature

# Create new branch from specific commit
# → Creates worktree at ../worktrees/<repo-name>/hotfix/urgent
wtp add -b hotfix/urgent abc1234

# Create new branch tracking a different remote branch
# → Creates worktree at ../worktrees/<repo-name>/feature/test with branch tracking origin/main
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
```

## Configuration

wtp uses `.wtp.yml` for project-specific configuration:

```yaml
version: "1.0"
defaults:
  # Base directory for worktrees (relative to project root)
  # ${WTP_REPO_BASENAME} expands to the repository directory name
  base_dir: "../worktrees/${WTP_REPO_BASENAME}"

hooks:
  post_create:
    # Copy gitignored files from main worktree to new worktree
    # Note: 'from' is relative to main worktree, 'to' is relative to new worktree
    - type: copy
      from: ".env" # Copy actual .env file (gitignored)
      to: ".env"

    - type: copy
      from: ".claude" # Copy AI context file (gitignored)
      to: ".claude"

    # Execute commands in the new worktree
    - type: command
      command: "npm install"
      env:
        NODE_ENV: "development"

    - type: command
      command: "make db:setup"
      work_dir: "."
```

The `${WTP_REPO_BASENAME}` placeholder expands to the repository's directory
name when resolving paths, ensuring zero-config isolation between different
repositories. You can combine it with additional path segments as needed.

> **Breaking change (vNEXT):** If you relied on the previous implicit default
> of `../worktrees` without a `.wtp.yml`, existing worktrees will now appear
> unmanaged because the new default expects
> `../worktrees/${WTP_REPO_BASENAME}`. Add a `.wtp.yml` with
> `base_dir: "../worktrees"` (or reorganize your worktrees) before upgrading
> to keep the legacy layout working.

### Copy Hooks: Main Worktree Reference

Copy hooks are designed to help you bootstrap new worktrees using files from
your main worktree (even if they are gitignored):

- `from`: path is always resolved relative to the main worktree.
- `to`: path is resolved relative to the newly created worktree.
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
      to: ".claude"

    # Directory copy also works
    - type: copy
      from: ".cursor/"
      to: ".cursor/"
```

This behavior applies regardless of where you run `wtp add` from (main worktree
or any other worktree).

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

With the default configuration (`base_dir: "../worktrees/${WTP_REPO_BASENAME}"`):

```
<project-root>/
├── .git/
├── .wtp.yml
└── src/

../worktrees/
└── <repo-name>/
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
