# Git Worktree Plus (git-wtp)

A powerful Git worktree management tool that extends git's worktree
functionality with automated setup, branch tracking, and project-specific hooks.

## Features

### Core Commands

- [ ] `git wtp init` - Initialize configuration file
- [x] `git wtp add` - Create worktree with automatic branch resolution
  - [x] Create from existing local branch
  - [x] Create from remote branch with automatic tracking
  - [ ] Create with new branch (`-b` option)
- [x] `git wtp remove` - Remove worktree
  - [x] Remove worktree only
  - [ ] Remove with branch (`--with-branch` option)
  - [x] Force removal (`--force` option)
- [x] `git wtp list` - List all worktrees with status
- [ ] `git wtp cd` - Change directory to worktree (requires shell integration)

### Advanced Features

- [x] **Post-create hooks**
  - [x] Copy files from main worktree
  - [x] Execute commands
- [ ] **Shell completion**
  - [ ] Bash completion
  - [ ] Zsh completion
  - [ ] Fish completion
- [x] **Cross-platform support**
  - [x] Linux
  - [x] macOS

## Installation

### Using Go

```bash
go install github.com/satococoa/git-wtp@latest
```

### Using Homebrew (macOS/Linux)

```bash
# Coming soon
brew install satococoa/tap/git-wtp
```

### From source

```bash
git clone https://github.com/satococoa/git-wtp.git
cd git-wtp
make build
sudo make install
```

## Quick Start

```bash
# Create worktree from existing local branch
git-wtp add feature/auth

# Create worktree from remote branch (automatically tracks)
git-wtp add feat1  # Creates from origin/feat1 if exists locally

# Create worktree with specific branch name
git-wtp add my-worktree feature/auth

# List all worktrees
git-wtp list

# Remove worktree
git-wtp remove feature/auth
git-wtp remove --force feature/auth  # Force removal even if dirty
```

## Configuration

Git-wtp uses `.git-worktree-plus.yml` for project-specific configuration:

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

### Bash

```bash
# Add to ~/.bashrc
source <(git-wtp completion bash)
eval "$(git-wtp shell-init bash)"
```

### Zsh

```zsh
# Add to ~/.zshrc
source <(git-wtp completion zsh)
eval "$(git-wtp shell-init zsh)"
```

### Fish

```fish
# Add to ~/.config/fish/config.fish
git-wtp completion fish | source
git-wtp shell-init fish | source
```

## Worktree Structure

With the default configuration (`base_dir: "../worktrees"`):

```
<project-root>/
├── .git/
├── .git-worktree-plus.yml
└── src/

../worktrees/
├── main/
├── feature/
│   ├── auth/          # git-wtp add feature/auth
│   └── payment/       # git-wtp add feature/payment
└── hotfix/
    └── bug-123/       # git-wtp add hotfix/bug-123
```

Branch names with slashes are preserved as directory structure, automatically organizing worktrees by type/category.

## Error Handling

Git-wtp provides clear error messages:

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
git clone https://github.com/satococoa/git-wtp.git
cd git-wtp

# Install dependencies
go mod download

# Run tests
make test

# Build
make build

# Run locally
./git-wtp --help
```

## Roadmap

### v0.1.0 (MVP) ✅ COMPLETED

- [x] Basic commands (add, remove, list)
- [x] Local branch support
- [x] Remote branch tracking
- [x] Configuration file support
- [x] Post-create hooks

### v0.2.0

- [ ] Shell completion
- [ ] Init command for configuration
- [ ] Branch creation (`-b` flag)

### v0.3.0

- [ ] Shell integration (cd command)
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
