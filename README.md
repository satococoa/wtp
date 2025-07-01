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
  - [x] Post-create hooks execution
- [x] `wtp remove` - Remove worktree
  - [x] Remove worktree only (git worktree compatible)
  - [x] Remove with branch (`--with-branch` option for convenience)
  - [x] Force removal (`--force` option)
- [x] `wtp list` - List all worktrees with status
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
  - [x] Linux
  - [x] macOS

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

# macOS (Intel)
curl -L https://github.com/satococoa/wtp/releases/latest/download/wtp_Darwin_x86_64.tar.gz | tar xz
sudo mv wtp /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/satococoa/wtp/releases/latest/download/wtp_Linux_x86_64.tar.gz | tar xz
sudo mv wtp /usr/local/bin/

# Windows (download .zip from releases page)
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

# Remove worktree only
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
      command: "npm"
      args: ["install"]
      env:
        NODE_ENV: "development"

    - type: command
      command: "make"
      args: ["db:setup"]
      work_dir: "."
```

## Shell Integration

### Quick Setup (Recommended)

#### If installed via Homebrew or Package Manager

Shell completions are automatically installed and should work immediately! No manual setup required.

#### Manual Setup

If you installed wtp manually:

##### For shell completion only:

```bash
# Bash: Add to ~/.bashrc
source <(wtp completion bash)

# Zsh: Add to ~/.zshrc
source <(wtp completion zsh)

# Fish: Add to ~/.config/fish/config.fish
wtp completion fish | source
```

##### For full integration (completion + cd command):

```bash
# Bash: Add to ~/.bashrc
eval "$(wtp shell-init --cd)"

# Zsh: Add to ~/.zshrc
eval "$(wtp shell-init --cd)"

# Fish: Add to ~/.config/fish/config.fish
wtp shell-init --cd | source
```

#### One-time Setup Helper

For convenience, wtp can show the exact commands for your current shell:

```bash
# Show completion setup
wtp shell-init

# Show full integration setup (with cd command)
wtp shell-init --cd
```

### Using the cd Command

Once shell integration is enabled, you can quickly change to any worktree:

```bash
# Change to a worktree by branch name
wtp cd feature/auth

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
make test

# Build
make build

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
- [ ] Multiple remote handling
- [ ] Better error messages

### v1.0.0

- [ ] Stable release
- [ ] Full test coverage
- [ ] Package manager support

### Future Ideas

- [ ] `git wtp status` - Show status of all worktrees
- [ ] `git wtp foreach` - Run command in all worktrees
- [ ] `git wtp clean` - Remove merged worktrees
- [ ] VSCode extension
- [ ] GitHub Actions integration

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Inspired by git-worktree and the need for better multi-branch development
workflows.
