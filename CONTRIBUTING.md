# Contributing to Worktree Plus (wtp)

Thank you for your interest in contributing to wtp! This document provides guidelines and information for contributors.

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How to Contribute

### Reporting Issues

Before creating an issue, please:

1. Check if the issue already exists in the [issue tracker](https://github.com/satococoa/wtp/issues)
2. Use the search function to avoid duplicates
3. Provide clear reproduction steps and expected vs actual behavior
4. Include your environment details (OS, Go version, git version)

### Suggesting Features

We welcome feature suggestions! Please:

1. Check existing issues and discussions first
2. Clearly describe the use case and benefits
3. Consider implementation complexity and maintenance burden
4. Be open to discussion and alternative approaches

### Development Setup

```bash
# Clone the repository
git clone https://github.com/satococoa/wtp.git
cd wtp

# Install dependencies
go mod download

# Build the project
make build

# Run tests
make test

# Run linting
make lint

# Install locally for testing
make install
```

### Development Workflow

1. **Fork the repository** on GitHub
2. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes** following our coding standards
4. **Write tests** for new functionality
5. **Run the test suite** to ensure nothing is broken
6. **Commit your changes** with clear commit messages
7. **Push to your fork** and create a pull request

### Coding Standards

#### Go Code Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and reasonably sized
- Handle errors properly - don't ignore them

#### Commit Messages

Follow conventional commit format:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:
```
feat(add): implement branch resolution from remotes
fix(remove): handle worktrees with uncommitted changes
docs: update installation instructions
```

### Testing

#### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run specific test
go test ./internal/git -v
```

#### Writing Tests

- Write unit tests for all new functionality
- Use table-driven tests for multiple test cases
- Mock external dependencies (git commands, file system)
- Test both success and error scenarios
- Integration tests should use temporary git repositories

Example test structure:
```go
func TestBranchResolution(t *testing.T) {
    tests := []struct {
        name     string
        branch   string
        expected string
        wantErr  bool
    }{
        {
            name:     "local branch exists",
            branch:   "feature/test",
            expected: "feature/test",
            wantErr:  false,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Pull Request Guidelines

#### Before Submitting

- [ ] Tests pass locally
- [ ] Code is properly formatted (`make fmt`)
- [ ] Linting passes (`make lint`)
- [ ] Documentation is updated if needed
- [ ] CHANGELOG.md is updated for user-facing changes

#### PR Description

Include:

- **What**: Brief description of changes
- **Why**: Motivation and context
- **How**: Implementation approach if complex
- **Testing**: How you tested the changes
- **Breaking Changes**: If any, with migration guide

#### Review Process

1. Automated checks must pass (CI/CD)
2. At least one maintainer review required
3. Address feedback promptly
4. Keep discussions focused and constructive
5. Squash commits before merge if requested

### Project Structure

```
wtp/
├── cmd/                    # Command implementations
│   ├── add.go
│   ├── remove.go
│   ├── list.go
│   └── init.go
├── internal/               # Internal packages
│   ├── git/               # Git operations
│   ├── config/            # Configuration handling
│   └── hooks/             # Hook execution
├── pkg/                   # Public packages (if any)
├── testdata/              # Test fixtures
├── docs/                  # Documentation
├── scripts/               # Build and utility scripts
└── Makefile              # Build automation
```

### Architecture Principles

1. **Simplicity**: Prefer simple, readable solutions
2. **Reliability**: Handle errors gracefully, fail safely
3. **Performance**: Efficient git operations, minimal overhead
4. **Compatibility**: Support multiple git versions and platforms
5. **Extensibility**: Design for future enhancements

### Documentation

- Keep README.md up to date
- Document new features and configuration options
- Include examples for complex functionality
- Update CLAUDE.md for design decisions
- Add inline comments for complex logic

### Release Process

Releases are handled by maintainers:

1. Version bump in relevant files
2. Update CHANGELOG.md
3. Create git tag
4. GitHub Actions builds and publishes binaries
5. Update Homebrew formula if needed

### Getting Help

- Open an issue for bugs or feature requests
- Start a discussion for questions or ideas
- Check existing documentation and issues first
- Be patient and respectful in communications

## Development Environment

### Recommended Tools

- **Go**: Latest stable version
- **Git**: 2.25+ recommended
- **Make**: For build automation
- **golangci-lint**: For linting
- **VS Code** with Go extension (optional but recommended)

### Makefile Targets

```bash
make help          # Show available targets
make build         # Build binary
make test          # Run tests
make test-coverage # Run tests with coverage
make lint          # Run linter
make fmt           # Format code
make clean         # Clean build artifacts
make install       # Install locally
```

Thank you for contributing to wtp! Your efforts help make Git worktree management better for everyone.