# Git Worktree Plus (git-wtp)

A powerful Git worktree management tool that extends git's worktree
functionality with automated setup, branch tracking, and project-specific hooks.

## Features

### Core Commands

- [ ] `git wtp init` - Initialize configuration file
- [ ] `git wtp add` - Create worktree with automatic branch resolution
  - [ ] Create from existing local branch
  - [ ] Create from remote branch with automatic tracking
  - [ ] Create with new branch (`-b` option)
- [ ] `git wtp remove` - Remove worktree
  - [ ] Remove worktree only
  - [ ] Remove with branch (`--with-branch` option)
  - [ ] Force removal (`--force` option)
- [ ] `git wtp list` - List all worktrees with status
- [ ] `git wtp cd` - Change directory to worktree (requires shell integration)

### Advanced Features

- [ ] **Post-create hooks**
  - [ ] Copy files from main worktree
  - [ ] Execute commands
- [ ] **Shell completion**
  - [ ] Bash completion
  - [ ] Zsh completion
  - [ ] Fish completion
- [ ] **Cross-platform support**
  - [ ] Linux
  - [ ] macOS

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
# Initialize git-wtp in your repository
git wtp init

# Create worktree from existing branch
git wtp add feature/auth

# Create worktree from remote branch
git wtp add feat1  # Creates from origin/feat1 if exists

# Create worktree with new branch
git wtp add -b feature/new-feature
git wtp add -b feature/new-feature develop  # branch from develop

# List all worktrees
git wtp list

# Remove worktree
git wtp remove feature/auth
git wtp remove feature/auth --with-branch  # Also delete branch

# Change to worktree directory (requires shell integration)
git wtp cd feature/auth
```

## Configuration

Git-wtp uses `.git-worktree-plus.yml` for project-specific configuration:

```yaml
version: 1

defaults:
  # Base directory for worktrees (relative to project root)
  base_dir: "../worktrees"

hooks:
  # Commands to run after creating a worktree
  post_create:
    # Copy files from main worktree
    copy_files:
      - source: ".env.example"
        dest: ".env"
      - source: ".env.local"
        # dest defaults to source if omitted

    # Execute commands
    commands:
      - name: "Install dependencies"
        run: "npm install"

      - name: "Setup database"
        run: "make db:setup"
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

```
<project-root>/
├── .git/
├── .git-worktree-plus.yml
└── src/

../worktrees/
├── main/
├── feature/
│   ├── auth/          # feature/auth branch
│   └── payment/       # feature/payment branch
└── hotfix/
    └── bug-123/       # hotfix/bug-123 branch
```

## Error Handling

Git-wtp provides clear error messages:

```bash
# Worktree already exists
Error: Worktree 'feature/auth' already exists at ../worktrees/feature/auth

# Branch already exists (when using -b)
Error: Branch 'main' already exists. Use 'git wtp add main' instead

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

### v0.1.0 (MVP)

- [ ] Basic commands (add, remove, list)
- [ ] Local branch support
- [ ] Remote branch tracking

### v0.2.0

- [ ] Configuration file support
- [ ] Post-create hooks
- [ ] Shell completion

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
